package monitoring

type Monitor interface {
	GetFileStats(path string) (map[string]interface{}, error)
	Close()
}
