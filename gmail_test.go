package gmailalert_test

import (
	"os"
	"testing"

	"github.com/aculclasure/gmailalert"
)

func TestNewGmailClient(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input       gmailalert.GmailClientConfig
		errExpected bool
	}{
		"Empty credentials file returns an error": {
			input: gmailalert.GmailClientConfig{
				CredentialsFile: "",
				TokenFile:       "token.json",
				UserInput:       os.Stdin,
				RedirectSvrPort: 9999,
			},
			errExpected: true,
		},
		"Invalid user input source returns an error": {
			input: gmailalert.GmailClientConfig{
				CredentialsFile: "credentials.json",
				UserInput:       nil,
				RedirectSvrPort: 9999,
			},
			errExpected: true,
		},
		"Invalid redirect server port returns an error": {
			input: gmailalert.GmailClientConfig{
				CredentialsFile: "credentials.json",
				UserInput:       os.Stdin,
				RedirectSvrPort: -9999,
			},
			errExpected: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			_, err := gmailalert.NewGmailClient(tc.input)
			errReceived := err != nil

			if tc.errExpected != errReceived {
				t.Errorf("got unexpected error status %t", errReceived)
			}
		})
	}
}
