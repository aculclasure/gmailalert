package gmailalert

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gregdel/pushover"
)

func TestNewPushoverClient(t *testing.T) {
	t.Parallel()

	t.Run(`empty appToken argument returns an error`, func(t *testing.T) {
		token := ""
		_, err := NewPushoverClient(token)

		if err == nil {
			t.Fatalf("wanted an error but did not get one")
		}
	})

	t.Run("valid token argument returns no errors", func(t *testing.T) {
		token := "da123321safdad"
		_, err := NewPushoverClient(token)

		if err != nil {
			t.Fatalf("got an unexpected error: %v", err)
		}
	})
}

func TestPrepareNotifyReq(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		input       Alert
		want        notifyReq
		errExpected bool
	}{
		"Giving an Alert with empty fields returns an error": {
			input:       Alert{},
			want:        notifyReq{},
			errExpected: true,
		},
		"Valid notification request": {
			input: Alert{
				GmailQuery:     "test",
				PushoverTarget: "test",
				PushoverTitle:  "test",
				PushoverSound:  "test",
				PushoverMsg:    "test",
			},
			want: notifyReq{
				recipient: "test",
				msg: pushover.Message{
					Message: "test",
					Title:   "test",
					Sound:   "test",
				},
			},
			errExpected: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := prepareNotifyReq(tc.input)
			errReceived := err != nil

			if tc.errExpected != errReceived {
				t.Fatalf("%s: got unexpected error status %v", name, errReceived)
			}

			if !tc.errExpected && !cmp.Equal(tc.want, got, cmp.AllowUnexported(notifyReq{}, pushover.Message{})) {
				t.Fatalf("%s: want != got\ndiff=%s",
					name,
					cmp.Diff(tc.want, got, cmp.AllowUnexported(notifyReq{}, pushover.Message{})))
			}
		})
	}
}

func TestHandle(t *testing.T) {
	t.Parallel()
	t.Run("Error returned from pushover call returns an error", func(t *testing.T) {
		client := PushoverClient{}
		err := client.handle(nil, errors.New("error from pushover call"))

		if err == nil {
			t.Fatalf("expected an error but did not get one")
		}
	})
}
