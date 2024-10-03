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

	err := d.fileTracker.Start(ctx)
	if err != nil {
		d.logger.Info("Started file tracking...")
		return err
	}
	for {
		select {
		case <-ctx.Done():
			d.logger.Info("daemon stopping due to context cancellation")
			return nil
		case <-d.ticker.C:
			d.logger.Debug("Performing periodic check")
			cmd := <-d.cmdChan
			d.logger.Info("Received command", "command", cmd)
			if err := d.executeCommand(cmd); err != nil {
				return fmt.Errorf("error executing command: %v", err)
			}
		}
	}
}

func (d *Daemon) executeCommand(cmd Command) error {
	command := exec.Command(cmd.Command, cmd.Args...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()

	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	if err != nil {
		d.logger.Error("Command execution failed",
			"error", err,
			"stdout", stdoutStr,
			"stderr", stderrStr,
			"command", cmd.Command,
			"args", cmd.Args,
		)
		return fmt.Errorf("command execution failed: %v, stdout: %s, stderr: %s", err, stdoutStr, stderrStr)
	}

	if stdoutStr != "" {
		d.logger.Info("Command executed successfully", "stdout", stdoutStr)
	} else {
		d.logger.Info("Command executed successfully (no output)")
	}

	if stderrStr != "" {
		d.logger.Warn("Command produced stderr output", "stderr", stderrStr)
	}

	return nil
}
