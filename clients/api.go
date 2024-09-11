package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/tejiriaustin/savannah-assessment/config"
	"github.com/tejiriaustin/savannah-assessment/models"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		baseURL: cfg.APIAddress,
		httpClient: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

func (c *Client) SendFileEvent(event models.FileEvent) error {
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error marshaling event: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/events", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error sending event to API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned non-OK status: %s", resp.Status)
	}

	return nil
}

func (c *Client) GetFileEvents() ([]models.FileEvent, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/events")
	if err != nil {
		return nil, fmt.Errorf("error getting events from API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned non-OK status: %s", resp.Status)
	}

	var events []models.FileEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("error decoding events: %w", err)
	}

	return events, nil
}
