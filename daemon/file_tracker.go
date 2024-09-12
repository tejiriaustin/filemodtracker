package daemon

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/tejiriaustin/savannah-assessment/clients"
	"github.com/tejiriaustin/savannah-assessment/config"
	"github.com/tejiriaustin/savannah-assessment/db"
	"github.com/tejiriaustin/savannah-assessment/models"
)

type FileTracker struct {
	config    *config.Config
	logger    *log.Logger
	done      chan bool
	osqClient *clients.OsQueryClient
	dbClient  *db.Client
}

func NewFileTracker(cfg *config.Config, dbClient *db.Client, osqueryClient *clients.OsQueryClient) (*FileTracker, error) {
	logger := log.New(os.Stdout, "FileTracker: ", log.Ldate|log.Ltime|log.Lshortfile)

	return &FileTracker{
		config:    cfg,
		logger:    logger,
		done:      make(chan bool),
		osqClient: osqueryClient,
		dbClient:  dbClient,
	}, nil
}

func (ft *FileTracker) Start() error {
	ft.logger.Printf("Starting file tracking in directory: %s", ft.config.MonitorDir)

	if err := ft.osqClient.EnsureFileEventMonitoring(ft.config.OsquerySocket, ft.config.ConfigPath); err != nil {
		return fmt.Errorf("file event monitoring is not properly configured: %w", err)
	}

	go ft.monitorFiles()

	return nil
}

func (ft *FileTracker) Stop() {
	ft.logger.Println("Stopping file tracking")
	ft.osqClient.Close()
	ft.done <- true
}

func (ft *FileTracker) monitorFiles() {
	ticker := time.NewTicker(10 * time.Second) // Poll every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ft.checkFileEvents()
		case <-ft.done:
			return
		}
	}
}

func (ft *FileTracker) checkFileEvents() {
	events, err := ft.osqClient.GetFileEvents()
	if err != nil {
		ft.logger.Printf("Error getting file events: %v", err)
		return
	}

	for _, event := range events {
		fileEvent := models.FileEvent{
			Path:      event.Path,
			Operation: event.Action,
			Timestamp: event.Timestamp,
		}

		err := ft.dbClient.InsertFileEvent(fileEvent)
		if err != nil {
			ft.logger.Printf("Error inserting event into database: %v", err)
		} else {
			ft.logger.Printf("%s file: %s", fileEvent.Operation, fileEvent.Path)
		}
	}
}
