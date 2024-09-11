package service

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/tejiriaustin/savannah-assessment/clients"
	"github.com/tejiriaustin/savannah-assessment/config"
	"github.com/tejiriaustin/savannah-assessment/db"
	"github.com/tejiriaustin/savannah-assessment/models"
)

type FileTracker struct {
	watcher  *fsnotify.Watcher
	config   *config.Config
	logger   *log.Logger
	done     chan bool
	dbClient *db.Client
}

func NewFileTracker(cfg *config.Config, dbClient *db.Client) (*FileTracker, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("error creating watcher: %w", err)
	}

	logger := log.New(os.Stdout, "FileTracker: ", log.Ldate|log.Ltime|log.Lshortfile)

	return &FileTracker{
		watcher:  watcher,
		config:   cfg,
		logger:   logger,
		done:     make(chan bool),
		dbClient: dbClient,
	}, nil
}

func (ft *FileTracker) Start() error {
	ft.logger.Printf("Starting file tracking in directory: %s", ft.config.MonitorDir)

	err := filepath.Walk(ft.config.MonitorDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return ft.watcher.Add(path)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking directory: %w", err)
	}

	go ft.watch()

	return nil
}

func (ft *FileTracker) Stop() {
	ft.logger.Println("Stopping file tracking")
	ft.watcher.Close()
	ft.done <- true
}

func (ft *FileTracker) watch() {
	for {
		select {
		case event, ok := <-ft.watcher.Events:
			if !ok {
				return
			}
			ft.handleEvent(event)
		case err, ok := <-ft.watcher.Errors:
			if !ok {
				return
			}
			ft.logger.Printf("error: %v", err)
		case <-ft.done:
			return
		}
	}
}
func (ft *FileTracker) handleEvent(event fsnotify.Event) {
	var operation string
	switch {
	case event.Op&fsnotify.Write == fsnotify.Write:
		operation = "modified"
	case event.Op&fsnotify.Create == fsnotify.Create:
		operation = "created"
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			ft.watcher.Add(event.Name)
		}
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		operation = "removed"
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		operation = "renamed"
	default:
		operation = "unknown"
	}

	ft.logger.Printf("%s file: %s", operation, event.Name)

	fileEvent := models.FileEvent{
		Path:      event.Name,
		Operation: operation,
		Timestamp: time.Now(),
	}

	// Add event to osquery extension
	clients.AddFileEvent(models.FileEvent{
		Path:      fileEvent.Path,
		Operation: fileEvent.Operation,
		Timestamp: fileEvent.Timestamp,
	})

	// Insert event into database
	err := ft.dbClient.InsertFileEvent(fileEvent)
	if err != nil {
		ft.logger.Printf("Error inserting event into database: %v", err)
	}
}
