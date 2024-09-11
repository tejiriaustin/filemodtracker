package models

import "time"

type FileEvent struct {
	ID        int64
	Path      string    `json:"path"`
	Operation string    `json:"operation"`
	Timestamp time.Time `json:"timestamp"`
}
