package gmailalert

import (
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/gregdel/pushover"
)

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

// PushoverClient represents a type providing behavior for
// sending Pushover notifications.
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
func (p PushoverClient) Notify(alt Alert) error {
	req, err := prepareNotifyReq(alt)
	if err != nil {
		return fmt.Errorf("got error preparing request to send pushover notification: %v", err)
	}

	tgt := pushover.NewRecipient(req.recipient)
	p.logger.Printf("sending pushover message %+q to recipient %s", req.msg, req.recipient)
	resp, err := p.app.SendMessage(&req.msg, tgt)
	return p.handle(resp, err)
}

// handle accepts a Pushover response and error returned after making a call to
// Pushover. If the error is not nil, it is returned. If the error is nil, then
// the Pushover response is logged.
func (p PushoverClient) handle(resp *pushover.Response, err error) error {
	if err != nil {
		return fmt.Errorf("got error sending pushover notification: %v", err)
	}

	p.logger.Printf("pushover message sent, got response: %s", resp.String())

	return nil
}

// notifyReq provides data that is expected to create a Pushover notification
// to a specific recipient and the message that the notification should contain.
type notifyReq struct {
	recipient string
	msg       pushover.Message
}

// prepareNotifyReq accepts an Alert, uses it to create a notifyReq struct
// containing the details needed for sending a Pushover notification, and
// returns it. An error is returned if the given Alert argument is invalid.
func prepareNotifyReq(alt Alert) (notifyReq, error) {
	if err := alt.OK(); err != nil {
		return notifyReq{}, err
	}

	n := notifyReq{
		recipient: alt.PushoverTarget,
		msg: pushover.Message{
			Message: alt.PushoverMsg,
			Title:   alt.PushoverTitle,
			Sound:   alt.PushoverSound,
		},
	}
	return n, nil
}
