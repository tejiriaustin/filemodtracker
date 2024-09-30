package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tejiriaustin/savannah-assessment/config"
	"github.com/tejiriaustin/savannah-assessment/logger"
)

var (
	startButton *widget.Button
	stopButton  *widget.Button
)

func Start(cfg *config.Config, logger *logger.Logger) {
	a := app.New()
	a.Settings().SetTheme(&darkTheme{})
	w := a.NewWindow("File Modification Tracker")

	headers := []string{"action", "category", "target_path", "time", "size", "md5", "sha1", "sha256"}

	table := widget.NewTable(
		func() (int, int) { return 1, len(headers) },
		func() fyne.CanvasObject {
			return widget.NewLabel("wide content")
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			label := cell.(*widget.Label)
			label.SetText("Loading...")
		},
	)

	// Supposed to make columns adjustable
	for i := range headers {
		table.SetColumnWidth(i, 150)
	}

	status := widget.NewLabel(fmt.Sprintf("Service Status: , %s", checkServiceStatus()))
	monitorDirLabel := widget.NewLabel(fmt.Sprintf("Monitoring Directory: %s", cfg.MonitoredDirectory))
	checkFreqLabel := widget.NewLabel(fmt.Sprintf("Check Frequency: %s", cfg.CheckFrequency))

	startButton = widget.NewButtonWithIcon("Start Monitoring", theme.MediaPlayIcon(), func() {
		go func() {
			startService(w, status)
			periodicLogRefresh(table, cfg.Port)
			periodicStatusCheck(status)
		}()
	})
	startButton.Importance = widget.HighImportance

	stopButton = widget.NewButtonWithIcon("Stop Service", theme.MediaStopIcon(), func() {
		go func() {
			stopService(w, status)
		}()
	})
	stopButton.Importance = widget.DangerImportance

	updateButtonStates(checkServiceStatus())

	refreshLogsButton := widget.NewButtonWithIcon("Refresh Logs", theme.ViewRefreshIcon(), func() {
		refreshLogs(table, cfg.Port)
	})

	infoBox := container.NewVBox(status, monitorDirLabel, checkFreqLabel)
	buttonsBox := container.NewHBox(startButton, stopButton, refreshLogsButton)
	topContent := container.NewVBox(infoBox, buttonsBox)

	content := container.NewBorder(topContent, nil, nil, nil, table)

	w.SetContent(content)
	w.Resize(fyne.NewSize(1024, 768))

	// Load initial data
	go refreshLogs(table, cfg.Port)

	w.ShowAndRun()
}

func startService(w fyne.Window, status *widget.Label) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		script := `do shell script "filemodtracker daemon" with administrator privileges`
		cmd = exec.Command("osascript", "-e", script)
	case "windows":
		cmd = exec.Command("powershell", "-Command", "Start-Process", "filemodtracker", "daemon", "-Verb", "runAs")
	case "linux":
		cmd = exec.Command("pkexec", "filemodtracker", "daemon")
	default:
		dialog.ShowError(fmt.Errorf("unsupported operating system: %s", runtime.GOOS), w)
		return
	}

	err := cmd.Start()
	if err != nil {
		errMsg := fmt.Sprintf("Failed to start service: %v", err)
		status.SetText("Service Status: " + errMsg)
		dialog.ShowError(fmt.Errorf(errMsg), w)
	} else {
		status.SetText("Service Status: Starting...")
		updateButtonStates("Starting")
		go func() {
			err = cmd.Wait()
			if err != nil {
				errMsg := fmt.Sprintf("Service exited with error: %v", err)
				status.SetText("Service Status: " + errMsg)
				dialog.ShowError(fmt.Errorf(errMsg), w)
			}
			currentStatus := checkServiceStatus()
			status.SetText("Service Status: " + currentStatus)
			updateButtonStates(currentStatus)
		}()
	}
}

func stopService(w fyne.Window, status *widget.Label) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		script := `do shell script "filemodtracker stop" with administrator privileges`
		cmd = exec.Command("osascript", "-e", script)
	case "windows":
		cmd = exec.Command("powershell", "-Command", "Start-Process", "filemodtracker", "stop", "-Verb", "runAs")
	case "linux":
		cmd = exec.Command("pkexec", "filemodtracker", "stop")
	default:
		dialog.ShowError(fmt.Errorf("unsupported operating system: %s", runtime.GOOS), w)
		return
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := fmt.Sprintf("Failed to stop service: %v output: %s", err, string(output))
		status.SetText("Service Status: " + errMsg)
		dialog.ShowError(fmt.Errorf(errMsg), w)
	} else {
		status.SetText("Service Status: Stopping...")
		go func() {
			err = cmd.Wait()
			if err != nil {
				errMsg := fmt.Sprintf("Service exited with error: %v", err)
				status.SetText("Service Status: " + errMsg)
				dialog.ShowError(fmt.Errorf(errMsg), w)
			} else {
				status.SetText("Service Status: Stopping...")
				updateButtonStates("Stopping")
				currentStatus := checkServiceStatus()
				status.SetText("Service Status: " + currentStatus)
				updateButtonStates(currentStatus)
			}
		}()
	}
}

func checkServiceStatus() string {
	cmd := exec.Command("filemodtracker", "status")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return fmt.Sprintf("Error checking status: %v\nOutput: %s", err, out.String())
	}
	return out.String()
}

func periodicStatusCheck(status *widget.Label) {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		currentStatus := checkServiceStatus()
		status.SetText("Service Status: " + currentStatus)
		updateButtonStates(currentStatus)
	}
}

func refreshLogs(table *widget.Table, port string) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	url := fmt.Sprintf("http://localhost%s/events", port)
	resp, err := client.Get(url)
	if err != nil {
		updateTableWithError(table, fmt.Sprintf("Error fetching logs: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		updateTableWithError(table, fmt.Sprintf("Error fetching logs: HTTP status %d", resp.StatusCode))
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		updateTableWithError(table, fmt.Sprintf("Error reading response: %v", err))
		return
	}

	var events []map[string]interface{}
	if err := json.Unmarshal(body, &events); err != nil {
		updateTableWithError(table, fmt.Sprintf("Error parsing JSON: %v", err))
		return
	}

	updateTableWithEvents(table, events)
}

func updateTableWithError(table *widget.Table, errorMsg string) {
	table.Length = func() (int, int) { return 2, 1 }
	table.UpdateCell = func(id widget.TableCellID, cell fyne.CanvasObject) {
		label := cell.(*widget.Label)
		if id.Row == 0 {
			label.SetText("Error")
			label.TextStyle = fyne.TextStyle{Bold: true}
		} else {
			label.SetText(errorMsg)
		}
	}
	table.Refresh()
}

func updateTableWithEvents(table *widget.Table, events []map[string]interface{}) {
	headers := []string{"action", "category", "target_path", "time", "size", "md5", "sha1", "sha256"}

	table.Length = func() (int, int) { return len(events) + 1, len(headers) }
	table.UpdateCell = func(id widget.TableCellID, cell fyne.CanvasObject) {
		label := cell.(*widget.Label)
		label.TextStyle = fyne.TextStyle{Monospace: true}
		label.Alignment = fyne.TextAlignLeading
		label.Wrapping = fyne.TextTruncate

		if id.Row == 0 {
			label.SetText(headers[id.Col])
			label.TextStyle.Bold = true
		} else {
			event := events[id.Row-1]
			value := fmt.Sprintf("%v", event[headers[id.Col]])
			if headers[id.Col] == "time" {
				timestamp, err := strconv.ParseInt(value, 10, 64)
				if err == nil {
					value = time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
				}
			}
			label.SetText(value)
		}
	}
	table.Refresh()
}

func periodicLogRefresh(table *widget.Table, port string) {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		refreshLogs(table, port)
	}
}

func updateButtonStates(status string) {
	status = strings.ToLower(status)
	switch {
	case strings.Contains(status, "running"):
		stopButton.Enable()
	case strings.Contains(status, "starting"):
		stopButton.Disable()
	case strings.Contains(status, "stopped"):
		stopButton.Disable()
	case strings.Contains(status, "stopping"):
		stopButton.Disable()
	default:
		// If status is unknown, enable both buttons
		stopButton.Enable()
	}
}
