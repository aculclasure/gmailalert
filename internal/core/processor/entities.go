package processor

import "errors"

type Alert struct {
	Message   string
	Title     string
	Recipient string
	Sound     string
}

func (a Alert) OK() error {
	if a.Message == "" {
		return errors.New("alert must contain a non-empty message")
	}
	if a.Title == "" {
		return errors.New("alert must contain a non-empty title")
	}
	if a.Recipient == "" {
		return errors.New("alert must contain a non-empty recipient")
	}
	return nil
}

type EmailQuery struct {
	SearchExpression string
}

type EmailQueryResult struct {
	Query          EmailQuery
	MatchingEmails []string
}
