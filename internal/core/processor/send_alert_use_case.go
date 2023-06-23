package processor

import "errors"

type AlertRepo interface {
	Notify(alt Alert) error
}

type SendAlertUseCase struct {
	alertRepo AlertRepo
}

func NewSendAlertUseCase(alertRepo AlertRepo) (*SendAlertUseCase, error) {
	if alertRepo == nil {
		return nil, errors.New("alert repo argument must be non-nil")
	}
	return &SendAlertUseCase{alertRepo: alertRepo}, nil
}

func (s *SendAlertUseCase) Run(alt Alert) error {
	err := s.alertRepo.Notify(alt)
	if err != nil {
		return err
	}
	return nil
}
