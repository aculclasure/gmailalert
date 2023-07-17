package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// AlertConfig represents a configuration containing a Pushover application to
// send alerts to and the alerts to notify on.
type AlertConfig struct {
	PushoverApp string  `json:"pushoverapp"`
	Alerts      []Alert `json:"alerts"`
}

// Alert represents a Gmail filtering query to find matches against and the
// corresponding configuration to use in the Pushover notification.
type Alert struct {
	// The Gmail query expression to match emails against.
	// See https://support.google.com/mail/answer/7190?hl=en
	GmailQuery string `json:"gmailquery"`
	// The pushover notification recipient.
	PushoverTarget string `json:"pushovertarget"`
	// The title of the pushover notification.
	PushoverTitle string `json:"pushovertitle"`
	// The pushover sound to use for the notification.
	PushoverSound string `json:"pushoversound"`
	// The message to put in the pushover notification.
	PushoverMsg string
}

// DecodeAlerts accepts an io.Reader containing JSON-formatted alert configuration,
// decodes the JSON object into an AlertConfig value and returns the AlertConfig. An
// error is returned if the io.Reader argument is nil or if there is a problem
// JSON-decoding the io.Reader.
func DecodeAlerts(rdr io.Reader) (AlertConfig, error) {
	if rdr == nil {
		return AlertConfig{}, errors.New("io.Reader argument must be non-nil")
	}

	var a AlertConfig
	if err := json.NewDecoder(rdr).Decode(&a); err != nil {
		return AlertConfig{}, fmt.Errorf("got an error decoding JSON: %v", err)
	}

	return a, nil
}

// OK validates a given Alert and returns an error if any of its fields are empty.
func (a Alert) OK() error {
	// if a.GmailQuery == "" || a.PushoverMsg == "" || a.PushoverSound == "" || a.PushoverTarget == "" || a.PushoverTitle == "" {
	// 	return fmt.Errorf("error validating alert %+q: all fields in the alert must be non-empty", a)
	// }
	switch {
	case a.GmailQuery == "":
		return errors.New("error: alert must have a non-empty gmail query field")
	case a.PushoverTitle == "":
		return errors.New("error: alert must have a non-empty pushover title field")
	case a.PushoverSound == "":
		return errors.New("error: alert must have a non-empty pushover sound field")
	case a.PushoverTarget == "":
		return errors.New("error: alert must have a non-empty pushover target field")
	default:
		return nil
	}
}
