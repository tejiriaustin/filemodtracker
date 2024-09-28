package daemon

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/tejiriaustin/savannah-assessment/config"
	"github.com/tejiriaustin/savannah-assessment/logger"
	"github.com/tejiriaustin/savannah-assessment/monitoring"
)

type (
	Daemon struct {
		logger      *logger.Logger
		ticker      *time.Ticker
		cfg         *config.Config
		fileTracker monitoring.Monitor
		cmdChan     <-chan Command
		pidFile     string
	}
	Command struct {
		Command string
		Args    []string
	}
)

func newDaemon() *Daemon {
	return &Daemon{}
}

func New(cfg *config.Config, logger *logger.Logger, fileTracker monitoring.Monitor, cmdChan <-chan Command) (*Daemon, error) {
	d := newDaemon()
	d.cfg = cfg
	d.fileTracker = fileTracker
	d.cmdChan = cmdChan
	d.logger = logger

	return d, nil
}

func (d *Daemon) StartDaemon(ctx context.Context) error {
	d.logger.Info("Starting daemon...")

	d.ticker = time.NewTicker(10 * time.Second)
	defer d.ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("Daemon stopping due to context cancellation")
			return ctx.Err()
		case <-d.ticker.C:
			d.logger.Debug("Performing periodic check")
			if err := d.performPeriodicTasks(); err != nil {
				d.logger.Error("Error during periodic tasks", "error", err)
			}
		case cmd := <-d.cmdChan:
			d.logger.Info("Received command", "command", cmd)
			if err := d.executeCommand(cmd); err != nil {
				d.logger.Error("Error executing command", "error", err)
			}
		}
	}
}

func (d *Daemon) performPeriodicTasks() error {
	d.logger.Debug("Performing periodic tasks")
	return nil
}

func (d *Daemon) executeCommand(cmd Command) error {
	command := exec.Command(cmd.Command, cmd.Args...)
	var out bytes.Buffer
	command.Stdout = &out
	command.Stderr = &out
	err := command.Run()
	if err != nil {
		return fmt.Errorf("command execution failed: %v, output: %s", err, out.String())
	}
	d.logger.Infof("Command executed successfully. Output: %s", out.String())
	return nil
}
