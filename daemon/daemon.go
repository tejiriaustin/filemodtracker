package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tejiriaustin/savannah-assessment/config"
	"github.com/tejiriaustin/savannah-assessment/db"
	"github.com/tejiriaustin/savannah-assessment/models"
)

type Daemon struct {
	cfg         *config.Config
	dbClient    *db.Client
	fileTracker *FileTracker
	cmdQueue    chan string
	pidFile     string
	wg          sync.WaitGroup
	stopChan    chan struct{}
}

func New(cfg *config.Config, dbClient *db.Client, cmd chan string) (*Daemon, error) {
	ft, err := NewFileTracker(cfg, dbClient)
	if err != nil {
		return nil, fmt.Errorf("error creating file tracker: %w", err)
	}

	return &Daemon{
		cfg:         cfg,
		dbClient:    dbClient,
		fileTracker: ft,
		cmdQueue:    make(chan string, 100), // Buffer for 100 commands
		pidFile:     filepath.Join(os.TempDir(), "filemodtracker.pid"),
		stopChan:    make(chan struct{}),
	}, nil
}

func (d *Daemon) Start() error {
	if d.isRunning() {
		return fmt.Errorf("daemon is already running")
	}

	// Write PID file
	pid := os.Getpid()
	if err := os.WriteFile(d.pidFile, []byte(fmt.Sprintf("%d", pid)), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Start file tracker
	if err := d.fileTracker.Start(); err != nil {
		return fmt.Errorf("failed to start file tracker: %w", err)
	}

	// Start command executor
	d.wg.Add(1)
	go d.commandExecutor()

	return nil
}

func (d *Daemon) Stop() error {
	if !d.isRunning() {
		return fmt.Errorf("daemon is not running")
	}

	// Stop file tracker
	d.fileTracker.Stop()

	// Stop command executor
	close(d.stopChan)
	d.wg.Wait()

	// Remove PID file
	if err := os.Remove(d.pidFile); err != nil {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}

	return nil
}

func (d *Daemon) Status() (string, error) {
	if d.isRunning() {
		return "Running", nil
	}
	return "Stopped", nil
}

func (d *Daemon) isRunning() bool {
	_, err := os.Stat(d.pidFile)
	return err == nil
}

func (d *Daemon) EnqueueCommand(cmd string) {
	d.cmdQueue <- cmd
}

func (d *Daemon) commandExecutor() {
	defer d.wg.Done()

	ticker := time.NewTicker(time.Minute) // Run every minute
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.executeCommands()
		case <-d.stopChan:
			return
		}
	}
}

func (d *Daemon) executeCommands() {
	for {
		select {
		case cmd := <-d.cmdQueue:
			// Execute the command
			// This is where you'd implement the actual command execution logic
			fmt.Printf("Executing command: %s\n", cmd)

			// After execution, you might want to log it or update a status
			event := models.FileEvent{
				Path:      "Command Execution",
				Operation: cmd,
				Timestamp: time.Now(),
			}
			if err := d.dbClient.InsertFileEvent(event); err != nil {
				fmt.Printf("Error logging command execution: %v\n", err)
			}
		default:
			// No more commands in the queue
			return
		}
	}
}
