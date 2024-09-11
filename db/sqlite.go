package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/tejiriaustin/savannah-assessment/models"
)

type Client struct {
	db *sql.DB
}

func NewClient(dbPath string) (*Client, error) {
	database, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	if err := database.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	return &Client{db: database}, nil
}

func (c *Client) Close() error {
	return c.db.Close()
}

func (c *Client) CreateFileEventsTable() error {
	query := `CREATE TABLE IF NOT EXISTS file_events (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        path TEXT,
        operation TEXT,
        timestamp DATETIME
    )`
	_, err := c.db.Exec(query)
	return err
}

func (c *Client) InsertFileEvent(event models.FileEvent) error {
	query := `INSERT INTO file_events (path, operation, timestamp) VALUES (?, ?, ?)`
	_, err := c.db.Exec(query, event.Path, event.Operation, event.Timestamp)
	return err
}

func (c *Client) GetFileEvents() ([]models.FileEvent, error) {
	rows, err := c.db.Query(`SELECT id, path, operation, timestamp FROM file_events ORDER BY timestamp DESC`)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}(rows)

	events := make([]models.FileEvent, 0)

	for rows.Next() {
		var (
			event     models.FileEvent
			timestamp string
		)

		err := rows.Scan(&event.ID, &event.Path, &event.Operation, &timestamp)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		event.Timestamp, err = time.Parse(time.RFC3339, timestamp)
		if err != nil {
			return nil, fmt.Errorf("error parsing timestamp: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}
