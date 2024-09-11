package ui

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/tejiriaustin/savannah-assessment/config"
	"github.com/tejiriaustin/savannah-assessment/db"
)

var (
	mu sync.Mutex
)

func Start(cfg *config.Config, dbClient db.Repository) {
	a := app.New()
	w := a.NewWindow("File Modification Tracker")

	logs := widget.NewMultiLineEntry()
	logs.SetText("Log entries will appear here...")
	logs.Disable()

	execPath := filepath.Join(filepath.Dir(cfg.ConfigPath), "filemodtracker")

	status := widget.NewLabel("Service Status: Unknown")
	startButton := widget.NewButton("Start Service", func() {
		go func() {
			startService(status, execPath)
			periodicLogRefresh(logs, dbClient)
			periodicStatusCheck(status, execPath)
		}()
	})

	stopButton := widget.NewButton("Stop Service", func() {
		go stopService(status, execPath)
	})

	monitorDirLabel := widget.NewLabel(fmt.Sprintf("Monitoring Directory: %s", cfg.MonitorDir))
	checkFreqLabel := widget.NewLabel(fmt.Sprintf("Check Frequency: %s", cfg.CheckFrequency))

	refreshLogsButton := widget.NewButton("Refresh Logs", func() {
		refreshLogs(logs, dbClient)
	})

	buttons := container.NewHBox(startButton, stopButton, refreshLogsButton)
	info := container.NewVBox(status, monitorDirLabel, checkFreqLabel)
	content := container.NewVBox(info, buttons, logs)

	w.SetContent(content)
	w.Resize(fyne.NewSize(600, 400))

	w.ShowAndRun()
}

func startService(status *widget.Label, execPath string) {
	mu.Lock()
	defer mu.Unlock()

	cmd := exec.Command(execPath, "daemon", "start")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		status.SetText(fmt.Sprintf("Error starting service: %v\nOutput: %s", err, out.String()))
		return
	}

	status.SetText("Service Status: Running")
}

func stopService(status *widget.Label, execPath string) {
	mu.Lock()
	defer mu.Unlock()

	cmd := exec.Command(execPath, "daemon", "stop")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		status.SetText(fmt.Sprintf("Error stopping service: %v\nOutput: %s", err, out.String()))
		return
	}

	status.SetText("Service Status: Stopped")
}

func checkServiceStatus(execPath string) string {
	cmd := exec.Command(execPath, "daemon", "status")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return fmt.Sprintf("Error checking status: %v\nOutput: %s", err, out.String())
	}
	return out.String()
}

func periodicStatusCheck(status *widget.Label, execPath string) {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		currentStatus := checkServiceStatus(execPath)
		status.SetText("Service Status: " + currentStatus)
	}
}

func refreshLogs(logs *widget.Entry, dbClient db.Repository) {
	events, err := dbClient.GetFileEvents()
	if err != nil {
		logs.SetText(fmt.Sprintf("Error fetching logs: %v", err))
		return
	}

	var logText string
	for _, event := range events {
		logText += fmt.Sprintf("%s: %s %s\n", event.Timestamp.Format("2006-01-02 15:04:05"), event.Operation, event.Path)
	}

	if logText == "" {
		logText = "No logs available."
	}

	logs.SetText(logText)
}

func periodicLogRefresh(logs *widget.Entry, dbClient db.Repository) {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		refreshLogs(logs, dbClient)
	}
}
