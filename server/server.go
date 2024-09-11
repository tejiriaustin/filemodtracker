package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tejiriaustin/savannah-assessment/config"
	"github.com/tejiriaustin/savannah-assessment/db"
	"github.com/tejiriaustin/savannah-assessment/models"
)

type Server struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Server {
	return &Server{
		cfg: cfg,
	}
}

func (s *Server) Start(dbClient *db.Client, cmdChan chan<- string) {
	router := s.setupRouter(dbClient, cmdChan)

	srv := &http.Server{
		Addr:    s.cfg.APIAddress,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server exiting")
}

func (s *Server) setupRouter(dbClient *db.Client, cmdChan chan<- string) *gin.Engine {
	r := gin.Default()

	r.GET("/health", s.healthCheck())
	r.GET("/events", s.retrieveEvents(dbClient))
	r.POST("/events", s.receiveFileEvent(dbClient))
	r.POST("/command", s.receiveCommand(cmdChan))

	return r
}

func (s *Server) healthCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "alive and well",
		})
	}
}

func (s *Server) retrieveEvents(dbClient db.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		events, err := dbClient.GetFileEvents()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, events)
	}
}

func (s *Server) receiveFileEvent(dbClient db.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var event models.FileEvent
		if err := c.ShouldBindJSON(&event); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := dbClient.InsertFileEvent(event)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "event received"})
	}
}

func (s *Server) receiveCommand(cmdChan chan<- string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cmd struct {
			Command string `json:"command" binding:"required"`
		}
		if err := c.ShouldBindJSON(&cmd); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		cmdChan <- cmd.Command

		c.JSON(http.StatusOK, gin.H{"status": "command received"})
	}
}
