package service

type Service struct {
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) GetLogs() error {
	return nil
}
