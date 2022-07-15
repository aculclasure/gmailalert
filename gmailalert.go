// Package gmailalert provides types and functions for searching
// for Gmail messages matching specified criteria and emitting
// Pushover notifications when matches are found.
package gmailalert

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
)

// Matcher is the interface that wraps the Match method
// used by any types implementing email searching behavior.
type Matcher interface {
	Match(query string) ([]string, error)
}

// Notifier is the interface that wraps the Notify method
// used by any types implementing notification behavior.
type Notifier interface {
	Notify(a Alert) error
}

// Logger represents logger behavior that can be used by
// the Alerter.
type Logger interface {
	Printf(string, ...interface{})
}

// Alerter is a type that provides behavior for matching emails
// and sending a notification for any positive match results.
type Alerter struct {
	Matcher  Matcher
	Notifier Notifier
	Logger   Logger
}

// AlerterOption represents a functional option that can be passed to
// an Alerter.
type AlerterOption func(a *Alerter)

// WithAlerterLogger accepts a Logger and returns a functional option for
// wiring the Logger to an Alerter.
func WithAlerterLogger(l Logger) AlerterOption {
	return func(a *Alerter) {
		a.Logger = l
	}
}

// NewAlerter accepts a Matcher, a Notifier, and a slice of AlerterOptions
// creates a new Alerter struct from them, and returns the Alerter. An
// error is returned if the Matcher or Notifier arguments are nil.
func NewAlerter(m Matcher, n Notifier, opts ...AlerterOption) (Alerter, error) {
	if m == nil || n == nil {
		return Alerter{}, errors.New("matcher and notifier arguments must not be nil")
	}

	alerter := Alerter{
		Matcher:  m,
		Notifier: n,
		Logger:   log.New(os.Stdout, "INFO: ", log.LstdFlags),
	}
	for _, opt := range opts {
		opt(&alerter)
	}

	return alerter, nil
}

// Process accepts a slice of Alert structs, processes them concurrently
// to determine if any emails satisfying the alert criteria are found, and
// sends a notification if any matches are found. An error is returned if
// the the Alerter receiver has any nil fields.
func (a Alerter) Process(alerts []Alert) error {
	if a.Matcher == nil || a.Notifier == nil || a.Logger == nil {
		return fmt.Errorf("alerter must have non-nil matcher, notifier, and logger fields, got: %+q", a)
	}

	wg := sync.WaitGroup{}
	wg.Add(len(alerts))

	for _, alert := range alerts {
		go func(alt Alert) {
			defer wg.Done()
			matches, err := a.Matcher.Match(alt.GmailQuery)
			if err != nil {
				a.Logger.Printf("got error searching for email matches: %v", err)
				return
			}

			alt.PushoverMsg = fmt.Sprintf(`Found %d emails matching query "%s"`,
				len(matches), alt.GmailQuery)
			a.Logger.Printf("%s", alt.PushoverMsg)

			if len(matches) == 0 {
				return
			}

			err = a.Notifier.Notify(alt)
			if err != nil {
				a.Logger.Printf("got error sending notification: %v", err)
				return
			}
			a.Logger.Printf(`notification titled "%s" successfully sent via %T`,
				alt.PushoverTitle, a.Notifier)
		}(alert)
	}
	wg.Wait()
	return nil
}
