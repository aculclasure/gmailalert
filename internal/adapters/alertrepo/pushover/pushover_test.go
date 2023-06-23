package pushover_test

import (
	"testing"

	"github.com/aculclasure/gmailalert/internal/adapters/alertrepo/pushover"
	"github.com/aculclasure/gmailalert/internal/core/processor"
)

func TestNewPushoverClientWithEmptyTokenReturnsError(t *testing.T) {
	tok := ""
	_, err := pushover.NewPushoverClient(tok)
	if err == nil {
		t.Errorf("got unexpected error: %s", err)
	}
}

func TestNotifyWithInvalidAlertReturnsError(t *testing.T) {
	clt := &pushover.PushoverClient{}
	invalidAlert := processor.Alert{}
	err := clt.Notify(invalidAlert)
	if err == nil {
		t.Error("expected an error but did not get one")
	}
}
