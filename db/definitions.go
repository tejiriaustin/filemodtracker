package db

import "github.com/tejiriaustin/savannah-assessment/models"

type Repository interface {
	Close() error
	CreateFileEventsTable() error
	InsertFileEvent(event models.FileEvent) error
	GetFileEvents() ([]models.FileEvent, error)
}
