package pushover

import (
	"errors"
	"io"
	"log"

	"github.com/aculclasure/gmailalert/internal/core/processor"

	"github.com/gregdel/pushover"
)

type Logger interface {
	Printf(string, ...interface{})
}

// PushoverClientOpt represents a functional option that can be wired to a
// PushoverClient.
type PushoverClientOpt func(p *PushoverClient)

// WithPushoverClientLogger accepts a Logger and returns a function that
// wires the Logger to a PushoverClient.
func WithPushoverClientLogger(l Logger) PushoverClientOpt {
	return func(p *PushoverClient) {
		p.logger = l
	}
}

// PushoverClient provides a client type for sending Pushover notifications.
type PushoverClient struct {
	app    *pushover.Pushover
	logger Logger
}

// NewPushoverClient accepts a Pushover app token and returns a new
// PushoverClient. An error is returned if the Pushover app token is invalid.
func NewPushoverClient(token string, opts ...PushoverClientOpt) (PushoverClient, error) {
	if token == "" {
		return PushoverClient{}, errors.New("token argument must be non-empty")
	}

	client := PushoverClient{
		app:    pushover.New(token),
		logger: log.New(io.Discard, "", log.LstdFlags),
	}

	for _, opt := range opts {
		opt(&client)
	}

	return client, nil
}

// Notify accepts an Alert struct, constructs a Pushover notification
// from the data in the Alert and emits the Pushover notification.
// An error is returned if the message send fails.
func (p PushoverClient) Notify(alt processor.Alert) error {
	err := alt.OK()
	if err != nil {
		return err
	}
	tgt := pushover.NewRecipient(alt.Recipient)
	msg := pushover.Message{
		Message: alt.Message,
		Title:   alt.Title,
		Sound:   alt.Sound,
	}
	p.logger.Printf("sending pushover message %+q to recipient %s", msg, tgt)
	resp, err := p.app.SendMessage(&msg, tgt)
	if err != nil {
		return err
	}
	p.logger.Printf("pushover message sent, got response: %s", resp.String())
	return nil
}
