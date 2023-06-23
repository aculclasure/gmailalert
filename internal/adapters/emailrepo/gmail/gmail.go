package gmail

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Client represents a client for communicating with the Gmail API.
type Client struct {
	svc *gmail.Service
}

// NewClient accepts an HTTP Client that is OAuth2-enabled for sending requests
// to the Gmail API and an optional slice of ClientOpt and returns a Client
// struct that can communicate with the Gmail API. An error is returned if there
// is a problem creating thewrapped gmail service.
func NewClient(hc *http.Client) (*Client, error) {
	if hc == nil {
		return nil, errors.New("http client must be non-nil")
	}
	svc, err := gmail.NewService(context.Background(), option.WithHTTPClient(hc))
	if err != nil {
		return nil, fmt.Errorf("got error creating new gmail service: %s", err)
	}
	client := &Client{
		svc: svc,
	}
	return client, nil
}

// Find queries Gmail for any emails matching the given query, which can be any
// valid Gmail query expression, like "is:unread", "from:gopher@gmail.com", etc.
// It returns a slice of raw email messages matching the query
// where raw means the email message is RFC 2822 formatted and base64 encoded.
// An error is returned if the query to the Gmail API fails.
func (c Client) Find(query string) ([]string, error) {
	resp, err := c.svc.Users.Messages.List("me").Q(query).Do()
	if err != nil {
		return nil, fmt.Errorf("got error executing gmail query %s: %v", query, err)
	}
	rawMsgs := make([]string, 0, len(resp.Messages))
	for _, m := range resp.Messages {
		rawMsgs = append(rawMsgs, m.Raw)
	}
	return rawMsgs, nil
}
