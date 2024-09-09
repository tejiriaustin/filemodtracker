package ui

import (
	_ "github.com/lxn/walk"
	_ "github.com/lxn/walk/declarative"
)

type UI struct {
	window   *walk.MainWindow
	logView  *walk.TextEdit
	startBtn *walk.PushButton
	stopBtn  *walk.PushButton
}

func NewUI() (*UI, error) {
	ui := &UI{}

	if err := (MainWindow{
		Title:   "File Modification Tracker",
		MinSize: Size{Width: 500, Height: 300},
		Layout:  VBox{},
		Children: []Widget{
			PushButton{
				AssignTo: &ui.startBtn,
				Text:     "Start Service",
				OnClicked: func() {
					// Implement start service logic
				},
			},
			PushButton{
				AssignTo: &ui.stopBtn,
				Text:     "Stop Service",
				OnClicked: func() {
					// Implement stop service logic
				},
			},
			TextEdit{
				AssignTo: &ui.logView,
				ReadOnly: true,
				VScroll:  true,
			},
		},
	}.Create()); err != nil {
		return nil, err
	}

	return ui, nil
}

func (ui *UI) Run() {
	ui.window.Run()
}

func (ui *UI) AppendLog(text string) {
	ui.logView.AppendText(text + "\r\n")
}
