package daemon

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/tejiriaustin/savannah-assessment/config"
	"github.com/tejiriaustin/savannah-assessment/monitoring"
)

type (
	Daemon struct {
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

func New(cfg *config.Config, fileTracker monitoring.Monitor, cmdChan <-chan Command) (*Daemon, error) {
	d := newDaemon()
	d.cfg = cfg
	d.fileTracker = fileTracker
	d.cmdChan = cmdChan

	return d, nil
}

func (daemon *Daemon) StartDaemon() error {
	//execPath := filepath.Join(filepath.Dir(daemon.cfg.ConfigPath))

	for {
		time.Sleep(10 * time.Second)

		log.Println("Checking for new commands...")

		select {
		case cmd := <-daemon.cmdChan:
			err := daemon.executeCommand(cmd)
			if err != nil {
				log.Printf("Error executing command: %v\n", err)
			}
		default:
		}
	}
}

func (daemon *Daemon) executeCommand(cmd Command) error {
	command := exec.Command(cmd.Command, cmd.Args...)
	var out bytes.Buffer
	command.Stdout = &out
	command.Stderr = &out
	err := command.Run()
	if err != nil {
		return fmt.Errorf("Command execution failed: %v, output: %s", err, out.String())
	}
	log.Printf("Command executed successfully. Output: %s", out.String())
	return nil
}
