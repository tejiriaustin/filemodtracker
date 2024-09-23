package ui

import (
	"fmt"
	"os"
)

type CustomLogger struct{}

func (l *CustomLogger) Trace(s string) {}
func (l *CustomLogger) Debug(s string) {}
func (l *CustomLogger) Info(s string)  {}
func (l *CustomLogger) Warn(s string)  {}
func (l *CustomLogger) Error(err error) {
	if err != nil && err.Error() == "Error parsing user locale" {
		fmt.Println("Caught locale parsing error. Using default locale.")
		err := os.Setenv("LANG", "en_US.UTF-8")
		if err != nil {
			fmt.Printf("Warning: Failed to set default locale: %v\n", err)
		}
		return
	}
	fmt.Printf("Error: %v\n", err)
}
