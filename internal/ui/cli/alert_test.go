package cli_test

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/aculclasure/gmailalert/internal/ui/cli"
	"github.com/google/go-cmp/cmp"
)

func TestDecodeAlerts(t *testing.T) {
	t.Parallel()
	const (
		singleAlert = `
		{
		  "pushoverapp": "test",
		  "alerts": [
		    {   
		      "gmailquery": "test",     
		      "pushovertarget": "test",
		      "pushovertitle": "test",
		      "pushoversound": "test"
		    }
		  ]
		}
		`
		multipleAlerts = `
		{
		  "pushoverapp": "test",
		  "alerts": [
		    {
		      "gmailquery": "test1",     
	              "pushovertarget": "test1",
		      "pushovertitle": "test1",
		      "pushoversound": "test1"	
		    },
		    {
			"gmailquery": "test2",     
			"pushovertarget": "test2",
			"pushovertitle": "test2",
			"pushoversound": "test2"
		    }
		  ]
		}
		`
	)
	testCases := map[string]struct {
		input       io.Reader
		want        cli.AlertConfig
		errExpected bool
	}{
		"Nil io.Reader argument returns an error": {
			input:       nil,
			want:        cli.AlertConfig{},
			errExpected: true,
		},
		"Problem reading alert data returns an error": {
			input:       iotest.ErrReader(errors.New("read error")),
			want:        cli.AlertConfig{},
			errExpected: true,
		},
		"Decoding non-JSON data returns an error": {
			input:       strings.NewReader("this-is-not-json"),
			want:        cli.AlertConfig{},
			errExpected: true,
		},
		"Decoding a single valid alert returns an expected AlertConfig": {
			input: strings.NewReader(singleAlert),
			want: cli.AlertConfig{
				PushoverApp: "test",
				Alerts: []cli.Alert{
					{
						GmailQuery:     "test",
						PushoverTarget: "test",
						PushoverTitle:  "test",
						PushoverSound:  "test",
					},
				},
			},
			errExpected: false,
		},
		"Decoding multiple valid alerts returns an expected AlertConfig": {
			input: strings.NewReader(multipleAlerts),
			want: cli.AlertConfig{
				PushoverApp: "test",
				Alerts: []cli.Alert{
					{
						GmailQuery:     "test1",
						PushoverTarget: "test1",
						PushoverTitle:  "test1",
						PushoverSound:  "test1",
					},
					{
						GmailQuery:     "test2",
						PushoverTarget: "test2",
						PushoverTitle:  "test2",
						PushoverSound:  "test2",
					},
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := cli.DecodeAlerts(tc.input)
			errReceived := err != nil

			if tc.errExpected != errReceived {
				t.Errorf("%s: DecodeAlerts(%q) returned unexpected error status: %v",
					name, tc.input, errReceived)
			}
			if !tc.errExpected && !cmp.Equal(tc.want, got) {
				t.Errorf("%s: DecodeAlerts(%+v)\nwant != got\ndiff=%s",
					name, tc.input, cmp.Diff(tc.want, got))
			}
		})
	}

}

func TestAlert_OKErrorCases(t *testing.T) {
	t.Parallel()
	testCases := map[string]cli.Alert{
		"Missing gmail query returns error": {
			GmailQuery:     "",
			PushoverTarget: "abcd",
			PushoverTitle:  "title",
			PushoverSound:  "siren",
			PushoverMsg:    "got an alert",
		},
		"Missing pushover target returns error": {
			GmailQuery:     "is:unread",
			PushoverTarget: "",
			PushoverTitle:  "title",
			PushoverSound:  "siren",
			PushoverMsg:    "got an alert",
		},
		"Missing pushover title returns error": {
			GmailQuery:     "is:unread",
			PushoverTarget: "abce",
			PushoverTitle:  "",
			PushoverSound:  "siren",
			PushoverMsg:    "got an alert",
		},
		"Missing pushover sound returns error": {
			GmailQuery:     "is:unread",
			PushoverTarget: "abce",
			PushoverTitle:  "title",
			PushoverSound:  "",
			PushoverMsg:    "got an alert",
		},
		"Missing pushover message returns error": {
			GmailQuery:     "is:unread",
			PushoverTarget: "abce",
			PushoverTitle:  "title",
			PushoverSound:  "siren",
			PushoverMsg:    "",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := tc.OK()
			if err == nil {
				t.Fatal("expected an error but did not get one")
			}
		})
	}
}

func TestAlert_OKWithValidAlertDoesNotReturnError(t *testing.T) {
	t.Parallel()
	validAlert := cli.Alert{
		GmailQuery:     "is:unread",
		PushoverTarget: "abce",
		PushoverTitle:  "title",
		PushoverSound:  "siren",
		PushoverMsg:    "message",
	}
	err := validAlert.OK()
	if err != nil {
		t.Fatalf("got unexpected error: %v", err)
	}
}
