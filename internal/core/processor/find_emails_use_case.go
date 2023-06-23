package processor

import (
	"errors"
)

type EmailRepo interface {
	Find(searchExpression string) ([]string, error)
}

type FindEmailsUseCase struct {
	emailRepo EmailRepo
}

func NewFindEmailsUseCase(emailRepo EmailRepo) (*FindEmailsUseCase, error) {
	if emailRepo == nil {
		return nil, errors.New("email repo argument must be non-nil")
	}
	return &FindEmailsUseCase{emailRepo: emailRepo}, nil
}

func (f *FindEmailsUseCase) Run(query EmailQuery) (EmailQueryResult, error) {
	emails, err := f.emailRepo.Find(query.SearchExpression)
	if err != nil {
		return EmailQueryResult{}, err
	}
	return EmailQueryResult{Query: query, MatchingEmails: emails}, nil
}
