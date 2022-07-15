package gmailalert_test

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/aculclasure/gmailalert"
)

var (
	errFetchingMail        = errors.New("error fetching mail")
	errSendingNotification = errors.New("error sending notification")
)

func TestNewAlerter(t *testing.T) {
	t.Parallel()

	t.Run("Nil matcher or notifier argument returns error", func(t *testing.T) {
		_, err := gmailalert.NewAlerter(nil, nil)

		if err == nil {
			t.Errorf("wanted an error but did not get one")
		}
	})

	t.Run("Wire in a custom Logger", func(t *testing.T) {
		fakeLogDest := &bytes.Buffer{}
		fakeLogger := log.New(fakeLogDest, "", log.LstdFlags)
		alerter, err := gmailalert.NewAlerter(fakeMatcher{}, fakeNotifier{}, gmailalert.WithAlerterLogger(fakeLogger))

		if err != nil {
			t.Errorf("did not want an error but got one")
		}

		alerter.Logger.Printf("hello")

		gotLogs := fakeLogDest.String()
		if !strings.Contains(gotLogs, "hello") {
			t.Errorf(`Want logs to contain "hello", got: %s`, gotLogs)
		}
	})
}

func TestProcess(t *testing.T) {
	t.Parallel()

	t.Run("nil field in Alerter struct returns an error", func(t *testing.T) {
		a := gmailalert.Alerter{Matcher: nil, Notifier: nil, Logger: nil}
		alerts := []gmailalert.Alert{{}}
		err := a.Process(alerts)

		if err == nil {
			t.Fatalf("wanted an error but did not get one")
		}
	})

	t.Run("error when fetching emails is logged", func(t *testing.T) {
		spyLog := &spyLogger{}
		alt := gmailalert.Alerter{
			Matcher:  fakeMatcher{err: errFetchingMail},
			Notifier: fakeNotifier{},
			Logger:   spyLog,
		}
		alerts := []gmailalert.Alert{{}}

		err := alt.Process(alerts)
		if err != nil {
			t.Fatalf("got unexpected error: %v", err)
		}

		if spyLog.numErrCalls != 1 {
			t.Fatalf("wanted 1 error to be logged, got %d", spyLog.numErrCalls)
		}

	})

	t.Run("no email matches found is logged", func(t *testing.T) {
		spyLog := &spyLogger{}
		alt := gmailalert.Alerter{
			Matcher:  fakeMatcher{},
			Notifier: fakeNotifier{},
			Logger:   spyLog,
		}
		alerts := []gmailalert.Alert{{GmailQuery: "find:me"}}

		err := alt.Process(alerts)
		if err != nil {
			t.Fatalf("got unexpected error: %v", err)
		}

		if spyLog.numOKCalls != 1 {
			t.Fatalf("wanted 1 ok call to be logged, got %d", spyLog.numOKCalls)
		}

	})

	t.Run("error during notification sending is logged", func(t *testing.T) {
		spyLog := &spyLogger{}
		alt := gmailalert.Alerter{
			Matcher:  fakeMatcher{matches: []string{"matching-email"}},
			Notifier: fakeNotifier{err: errSendingNotification},
			Logger:   spyLog,
		}
		alerts := []gmailalert.Alert{{GmailQuery: "is:unread"}}

		err := alt.Process(alerts)
		if err != nil {
			t.Fatalf("got unexpected error: %v", err)
		}

		if spyLog.numErrCalls != 1 {
			t.Fatalf("wanted 1 error to be logged, got %d", spyLog.numErrCalls)
		}
	})

	t.Run("successful single notification", func(t *testing.T) {
		spyLog := &spyLogger{}
		spyNotif := &spyNotifier{}
		alt := gmailalert.Alerter{
			Matcher:  fakeMatcher{matches: []string{"matching-email"}},
			Notifier: spyNotif,
			Logger:   spyLog,
		}
		alerts := []gmailalert.Alert{{GmailQuery: "is:unread", PushoverTitle: "GotAHit!"}}

		err := alt.Process(alerts)
		if err != nil {
			t.Fatalf("got unexpected error: %v", err)
		}

		if spyNotif.numCalls != 1 {
			t.Fatalf("wanted notifier %T.Notify() to be called but it was not", spyNotif)
		}

		if spyLog.numOKCalls != 2 || spyLog.numErrCalls != 0 {
			t.Fatalf("wanted 2 ok logs and 0 error logs, got ok: %d, err: %d",
				spyLog.numOKCalls, spyLog.numErrCalls)
		}
	})

	t.Run("multiple successful notifications", func(t *testing.T) {
		spyLog := &spyLogger{}
		spyNotif := &spyNotifier{}
		alt := gmailalert.Alerter{
			Matcher:  fakeMatcher{matches: []string{"matching-email1", "matching-email2"}},
			Notifier: spyNotif,
			Logger:   spyLog,
		}
		alerts := []gmailalert.Alert{
			{GmailQuery: "is:unread", PushoverTitle: "FoundUnreadEmail!"},
			{GmailQuery: "from:someone", PushoverTitle: "FoundEmailFromSomeone!"},
			{GmailQuery: "to:someone", PushoverTitle: "FoundEmailToSomeone!"},
		}

		err := alt.Process(alerts)
		if err != nil {
			t.Fatalf("got unexpected error: %v", err)
		}

		if spyNotif.numCalls != 3 {
			t.Fatalf("wanted 3 notifications to be sent, got %d", spyNotif.numCalls)
		}

		if spyLog.numOKCalls != 6 {
			t.Fatalf("wanted 6 ok logs, got %d", spyLog.numOKCalls)
		}
	})

	t.Run("mixed successful and failed notifications", func(t *testing.T) {
		spyLog := &spyLogger{}
		mockNotif := &mockNotifier{
			errResponses: []error{errSendingNotification, nil, errSendingNotification, nil},
		}
		alt := gmailalert.Alerter{
			Matcher:  fakeMatcher{matches: []string{"matching-email1"}},
			Notifier: mockNotif,
			Logger:   spyLog,
		}
		alerts := []gmailalert.Alert{
			{GmailQuery: "is:unread", PushoverTitle: "FoundUnreadEmail!"},
			{GmailQuery: "from:someone", PushoverTitle: "FoundEmailFromSomeone!"},
			{GmailQuery: "to:someone", PushoverTitle: "FoundEmailToSomeone!"},
			{GmailQuery: "has:attachment", PushoverTitle: "FoundEmailWithAttachment!"},
		}

		err := alt.Process(alerts)
		if err != nil {
			t.Fatalf("got unexpected error: %v", err)
		}

		if spyLog.numErrCalls != 2 {
			t.Fatalf("wanted 2 error logs, got %d", spyLog.numErrCalls)
		}

		if spyLog.numOKCalls != 6 {
			t.Fatalf("wanted 6 ok logs, got %d", spyLog.numOKCalls)
		}
	})
}

// fakeMatcher represents a test double type that implements the
// Matcher interface. It's match method simply returns the matches
// and err values that the fakeMatcher struct was created with.
type fakeMatcher struct {
	matches []string
	err     error
}

// Match returns the matches and err fields of the receiver f.
func (f fakeMatcher) Match(_ string) ([]string, error) {
	return f.matches, f.err
}

// fakeNotifier represents a test double type that implements the
// Notifier interface. It's Notify method simply returns the err
// value that the fakeNotifier struct was created with.
type fakeNotifier struct {
	err error
}

// Notify returns the err field of the receiver f.
func (f fakeNotifier) Notify(_ gmailalert.Alert) error {
	return f.err
}

// spyNotifier represents a test double type that implements the
// Notifier interface and keeps a count of how many times its
// Notify method is called. It is safe to be used concurrently
// by multiple goroutines.
type spyNotifier struct {
	numCalls int64
}

// Notify increments the numCalls field of the receiver s and
// always returns a nil error.
func (s *spyNotifier) Notify(_ gmailalert.Alert) error {
	atomic.AddInt64(&s.numCalls, 1)
	return nil
}

// mockNotifier represents a test double type that implements the
// Notifier interface and is initialized with a set of error values
// to provide when it's Notify method is called. It is safe to be
// used concurrently by multiple goroutines.
type mockNotifier struct {
	errResponses []error
	next         int
	mtx          sync.Mutex
}

// Notify returns the next error value from the errResponses
// field of the receiver m. It is safe for concurrent use by
// multiple goroutines.
func (m *mockNotifier) Notify(_ gmailalert.Alert) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	resp := m.errResponses[m.next]
	m.next++
	return resp
}

// spyLogger represents a test double type that implements the Logger
// interface and keeps a count of how many error and non-error log calls
// are made to its Printf method.
type spyLogger struct {
	numOKCalls  int64
	numErrCalls int64
}

// Printf accepts a format string and slice of formatting directives,
// constructs the log statement from them and increments the numErrCalls
// field of the receiver s if the log statement contains "error" and
// increments the numOKCalls field of the receiver s otherwise. It is
// safe for concurrent use by multiple goroutines.
func (s *spyLogger) Printf(format string, args ...interface{}) {
	bufWr := &bytes.Buffer{}
	fmt.Fprintf(bufWr, format, args...)

	if bytes.Contains(bufWr.Bytes(), []byte("error")) {
		atomic.AddInt64(&s.numErrCalls, 1)
	} else {
		atomic.AddInt64(&s.numOKCalls, 1)
	}
}
