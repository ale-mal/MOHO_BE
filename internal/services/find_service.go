package services

type FindService struct {
}

func (s *FindService) ticker() {
}

func NewFindService() *FindService {
	s := &FindService{}
	go s.ticker()
	return s
}
