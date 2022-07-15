package gmailalert_test

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/aculclasure/gmailalert"
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
		want        gmailalert.AlertConfig
		errExpected bool
	}{
		"Nil io.Reader argument returns an error": {
			input:       nil,
			want:        gmailalert.AlertConfig{},
			errExpected: true,
		},
		"Problem reading alert data returns an error": {
			input:       iotest.ErrReader(errors.New("read error")),
			want:        gmailalert.AlertConfig{},
			errExpected: true,
		},
		"Decoding non-JSON data returns an error": {
			input:       strings.NewReader("this-is-not-json"),
			want:        gmailalert.AlertConfig{},
			errExpected: true,
		},
		"Decoding a single valid alert returns an AlertConfig": {
			input: strings.NewReader(singleAlert),
			want: gmailalert.AlertConfig{
				PushoverApp: "test",
				Alerts: []gmailalert.Alert{
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
		"Decoding multiple valid alerts returns an AlertConfig": {
			input: strings.NewReader(multipleAlerts),
			want: gmailalert.AlertConfig{
				PushoverApp: "test",
				Alerts: []gmailalert.Alert{
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
			got, err := gmailalert.DecodeAlerts(tc.input)
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
