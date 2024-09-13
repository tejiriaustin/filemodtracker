package monitoring

import "context"

type (
	Monitor interface {
		Querier
		StartMonitoring(tableName string, columnDefinitions map[string]string, opts ...Options) error
		StartConfigServer(configName string, GenerateConfigs func(ctx context.Context) (map[string]string, error)) error
	}
	Querier interface {
		Query(query string) ([]map[string]string, error)
	}
)
