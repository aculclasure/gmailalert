package processor_test

import (
	"testing"

	"github.com/aculclasure/gmailalert/internal/core/processor"
)

func TestAlertOKErrorCases(t *testing.T) {
	testCases := map[string]processor.Alert{
		"Empty message field returns error": {
			Message:   "",
			Title:     "Alert",
			Recipient: "Bob",
			Sound:     "cashregister",
		},
		"Empty title field returns error": {
			Message:   "Got an alert",
			Title:     "",
			Recipient: "Bob",
			Sound:     "cashregister",
		},
		"Empty recipient field returns error": {
			Message:   "Got an alert",
			Title:     "Alert",
			Recipient: "",
			Sound:     "cashregister",
		},
	}
	for name, alt := range testCases {
		t.Run(name, func(t *testing.T) {
			err := alt.OK()
			if err == nil {
				t.Error("expected an error but did not get one")
			}
		})
	}
}

func TestAlertOKWithValidAlertDoesNotReturnError(t *testing.T) {
	validAlert := processor.Alert{
		Message:   "Got an alert",
		Title:     "Alert",
		Recipient: "Bob",
		Sound:     "cashregister",
	}
	err := validAlert.OK()
	if err != nil {
		t.Errorf("got an unexpected error: %s", err)
	}
}
