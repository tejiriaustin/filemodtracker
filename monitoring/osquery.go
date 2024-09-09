package monitoring

import (
	"fmt"
	"time"

	"github.com/osquery/osquery-go"
)

type OsqueryClient struct {
	client *osquery.ExtensionManagerClient
}

var _ Monitor = (*OsqueryClient)(nil)

func NewOsqueryClient(socketPath string) (*OsqueryClient, error) {
	client, err := osquery.NewClient(socketPath, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("error creating osquery client: %w", err)
	}
	return &OsqueryClient{client: client}, nil
}

func (o *OsqueryClient) GetFileStats(path string) (map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM file WHERE path = '%s'", path)
	resp, err := o.client.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying osquery: %w", err)
	}

	if len(resp.Response) == 0 {
		return nil, fmt.Errorf("no results found for file: %s", path)
	}

	result := make(map[string]interface{})
	for key, value := range resp.Response[0] {
		result[key] = value
	}
	return result, nil
}

func (o *OsqueryClient) Close() {
	o.client.Close()
}
