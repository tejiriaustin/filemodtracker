package monitoring

import (
	"context"
	"time"
)

type (
	Monitor interface {
		Start(ctx context.Context) error
		Close() error
		Wait() error
		GetFileEvents() ([]map[string]interface{}, error)
		GetFileEventsByPath(path string, since time.Time) ([]map[string]interface{}, error)
		GetFileChangesSummary(since time.Time) ([]map[string]interface{}, error)
	}
)
