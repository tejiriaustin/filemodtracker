package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"fyne.io/fyne/v2/dialog"
	"image/color"
	"io/ioutil"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/tejiriaustin/savannah-assessment/config"
)

var (
	mu sync.Mutex
)

type darkTheme struct{}

func (d darkTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, theme.VariantDark)
}

func (d darkTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (d darkTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (d darkTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

func Start(cfg *config.Config) {
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

	status := widget.NewLabel("Service Status: Running")
	monitorDirLabel := widget.NewLabel(fmt.Sprintf("Monitoring Directory: %s", cfg.MonitorDir))
	checkFreqLabel := widget.NewLabel(fmt.Sprintf("Check Frequency: %s", cfg.CheckFrequency))

	execPath := filepath.Join(filepath.Dir(cfg.ConfigPath))

	startButton := widget.NewButtonWithIcon("Start Monitoring", theme.MediaPlayIcon(), func() {
		go func() {
			if requestRootAccess(w) {
				startService(status, execPath)
				periodicLogRefresh(table, cfg.Port)
				periodicStatusCheck(status, execPath)
			} else {
				status.SetText("Service Status: Root access denied")
			}
		}()
	})
	startButton.Importance = widget.HighImportance

	stopButton := widget.NewButtonWithIcon("Stop Service", theme.MediaStopIcon(), func() {
		go func() {
			if requestRootAccess(w) {
				stopService(status, execPath)
			} else {
				status.SetText("Service Status: Root access denied")
			}
		}()
	})
	stopButton.Importance = widget.DangerImportance

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

func requestRootAccess(w fyne.Window) bool {
	// Check if we're running on macOS
	if runtime.GOOS != "darwin" {
		dialog.ShowError(fmt.Errorf("root access request is only supported on macOS"), w)
		return false
	}

	cmd := exec.Command("osascript", "-e", `do shell script "echo Root access granted" with administrator privileges`)

	err := cmd.Run()
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to obtain root access: %v", err), w)
		return false
	}

	dialog.ShowInformation("Root Access", "Root access granted successfully", w)
	return true
}

func startService(status *widget.Label, execPath string) {
	cmd := exec.Command("sudo", "./filemodtracker", "start", "-d")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		status.SetText(fmt.Sprintf("Service Status: Failed to start - %v", err))
	} else {
		status.SetText(fmt.Sprintf("Service Status: %s", out.String()))
	}
}

func stopService(status *widget.Label, execPath string) {
	cmd := exec.Command("sudo", "filemodtracker", "stop")
	err := cmd.Run()
	if err != nil {
		status.SetText(fmt.Sprintf("Service Status: Failed to stop - %v", err))
	} else {
		status.SetText("Service Status: Stopped")
	}
}

func checkServiceStatus(execPath string) string {
	cmd := exec.Command("filemodtracker", "daemon", "status")
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
