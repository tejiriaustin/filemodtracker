package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tejiriaustin/savannah-assessment/config"
	"github.com/tejiriaustin/savannah-assessment/daemon"
	"github.com/tejiriaustin/savannah-assessment/logger"
	"github.com/tejiriaustin/savannah-assessment/monitoring"
)

type Server struct {
	cfg    *config.Config
	server *http.Server
	logger *logger.Logger
}

func New(cfg *config.Config, logger *logger.Logger) *Server {
	return &Server{
		cfg:    cfg,
		server: &http.Server{Addr: cfg.Port},
		logger: logger,
	}
}

func (s *Server) Start(monitor monitoring.Monitor, cmdChan chan<- daemon.Command) error {
	router := s.setupRouter(monitor, cmdChan)

	srv := &http.Server{
		Addr:    s.cfg.Port,
		Handler: router,
	}

	errChan := make(chan error, 1)
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("Server error: %s", err)
			errChan <- err
		}
	}()

	select {
	case <-quit:
		s.logger.Info("Shutdown signal received")
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		s.logger.Error("Server forced to shutdown: %s", err)
		return err
	}

	s.logger.Info("Server gracefully stopped")
	return nil
}

func (s *Server) setupRouter(monitor monitoring.Monitor, cmdChan chan<- daemon.Command) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(s.loggerMiddleware())
	r.Use(gin.Recovery())

	r.GET("/health", s.healthCheck())
	r.GET("/events", s.retrieveEvents(monitor))
	r.POST("/command", s.receiveCommand(cmdChan))
	r.POST("/execute", s.executeCommand())

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"status": "not found",
		})
	})

	return r
}

func (s *Server) healthCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "alive and well",
		})
	}
}

func (s *Server) retrieveEvents(monitor monitoring.Monitor) gin.HandlerFunc {
	return func(c *gin.Context) {
		query, err := monitor.GetFileEvents()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, query)
	}
}

func (s *Server) receiveCommand(cmdChan chan<- daemon.Command) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cmd struct {
			Command string `json:"command" binding:"required"`
		}
		if err := c.ShouldBindJSON(&cmd); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate the command
		sanitizedCmd, err := validateAndSanitizeCommand(cmd.Command)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmdChan <- daemon.Command{
			Command: sanitizedCmd[0],
			Args:    sanitizedCmd[1:],
		}

		c.JSON(http.StatusOK, gin.H{"status": "command received"})
	}
}

func (s *Server) executeCommand() gin.HandlerFunc {
	return func(c *gin.Context) {
		var cmd struct {
			Command string `json:"command" binding:"required"`
		}
		if err := c.ShouldBindJSON(&cmd); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		sanitizedCmd, err := validateAndSanitizeCommand(cmd.Command)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		command := exec.Command(sanitizedCmd[0], sanitizedCmd...)
		var out bytes.Buffer
		command.Stdout = &out
		command.Stderr = &out
		err = command.Run()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("command execution failed: %v, output: %s", err, out.String())})
			return
		}

		payload := gin.H{
			"status": "command received",
			"output": out.String(),
		}
		c.JSON(http.StatusOK, payload)
	}
}
