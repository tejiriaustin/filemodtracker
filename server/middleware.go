package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Handler) loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		duration := time.Since(start)
		statusCode := c.Writer.Status()

		_, file, line, _ := runtime.Caller(1)

		logEntry := struct {
			Level     string    `json:"level"`
			Timestamp time.Time `json:"timestamp"`
			Caller    string    `json:"caller"`
			Msg       string    `json:"msg"`
			Pid       int       `json:"pid"`
			Method    string    `json:"method"`
			Path      string    `json:"path"`
			Query     string    `json:"query"`
			Status    int       `json:"status"`
			Duration  string    `json:"duration"`
			ClientIP  string    `json:"clientIP"`
		}{
			Level:     "info",
			Timestamp: time.Now(),
			Caller:    filepath.Base(file) + ":" + strconv.Itoa(line),
			Msg:       "Request processed",
			Pid:       os.Getpid(),
			Method:    c.Request.Method,
			Path:      path,
			Query:     query,
			Status:    statusCode,
			Duration:  duration.String(),
			ClientIP:  c.ClientIP(),
		}

		jsonLog, err := json.Marshal(logEntry)
		if err != nil {
			s.logger.Error("Failed to marshal log entry: %v", err)
			return
		}

		s.logger.Info(string(jsonLog))

		if len(c.Errors) > 0 {
			s.logger.Error("Request errors: %v", c.Errors)
		}
	}
}
