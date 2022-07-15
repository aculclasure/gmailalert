package gmailalert

import (
	"errors"
	"io"
	"log"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
)

func TestPrepareMatchResp(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input []*gmail.Message
		want  []string
	}{
		"Nil raw input returns an empty slice": {
			input: nil,
			want:  []string{},
		},
		"Valid non-empty raw input returns a valid string slice": {
			input: []*gmail.Message{
				{Raw: "email0"},
				{Raw: "email1"},
			},
			want: []string{"email0", "email1"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := prepareMatchResp(tc.input)

			if !cmp.Equal(tc.want, got) {
				t.Fatalf("%s: prepareMatchResp(%+v) want != got\nwant=%+v\ngot=%+v",
					name, tc.input, tc.want, got)
			}
		})
	}
}

func TestPrepareConfigRequest(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input       io.Reader
		want        configRequest
		errExpected bool
	}{
		"Error reading credentials data returns an error": {
			input:       iotest.ErrReader(errors.New("read error")),
			errExpected: true,
		},
		"Empty credentials data returns an error": {
			input:       strings.NewReader(""),
			errExpected: true,
		},
		"Valid credentials data returns a configRequest struct": {
			input: strings.NewReader(`{"installed": {"client_id": 792312312}}`),
			want: configRequest{
				credentials: []byte(`{"installed": {"client_id": 792312312}}`),
				scope:       gmail.GmailReadonlyScope,
			},
			errExpected: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := prepareConfigRequest(tc.input)
			errReceived := err != nil

			if errReceived != tc.errExpected {
				t.Errorf("got unexpected error status: %t", errReceived)
			}

			if !errReceived && !cmp.Equal(tc.want, got, cmp.AllowUnexported(configRequest{})) {
				t.Errorf("want != got\ndiff=%s", cmp.Diff(tc.want, got, cmp.AllowUnexported(configRequest{})))
			}
		})
	}
}

func TestTokenReturnsTokenFromFileWhenFileExists(t *testing.T) {
	t.Parallel()

	wantTime, err := time.Parse("2006-01-02T15:04:05.000000000-07:00", "2022-08-16T12:00:42.516357003-04:00")
	if err != nil {
		t.Fatal(err)
	}
	want := &oauth2.Token{
		AccessToken:  "ab12.gophercd4567",
		TokenType:    "Bearer",
		RefreshToken: "1//gopher9876",
		Expiry:       wantTime,
	}

	myOAuth := gmailOAuth2{
		GmailClientConfig: GmailClientConfig{
			TokenFile: "testdata/test-oauth2-token.json",
			Logger:    log.New(io.Discard, "", log.LstdFlags),
		},
	}

	got, err := myOAuth.token()
	if err != nil {
		t.Errorf("got unexpected error: %s", err)
	}

	if !cmp.Equal(want, got, cmpopts.IgnoreUnexported(oauth2.Token{})) {
		t.Errorf("want != got\ndiff=%s", cmp.Diff(want, got, cmpopts.IgnoreUnexported(oauth2.Token{})))
	}
}

func TestGetAuthCode(t *testing.T) {
	t.Parallel()

	type input struct {
		authURL         string
		userInput       io.Reader
		redirectSvrPort int
	}

	testCases := map[string]struct {
		input       input
		want        string
		errExpected bool
	}{
		"Invalid URL argument returns an error": {
			input:       input{"://localhost:9999", nil, 9999},
			want:        "",
			errExpected: true,
		},
		"Nil user input source returns an error": {
			input:       input{"http://localhost:9999", nil, 9999},
			want:        "",
			errExpected: true,
		},
		"Invalid redirect server port returns an error": {
			input:       input{"http://localhost:9999", strings.NewReader(""), -9999},
			want:        "",
			errExpected: true,
		},
		"Error when reading user input returns an error": {
			input:       input{"http://localhost:9999", iotest.ErrReader(errors.New("read error")), 9999},
			want:        "",
			errExpected: true,
		},
		"Captured user input is returned as string": {
			input:       input{"http://localhost:9999", strings.NewReader("abc123"), 9999},
			want:        "abc123",
			errExpected: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := getAuthCode(tc.input.authURL, tc.input.userInput, tc.input.redirectSvrPort)
			errReceived := err != nil

			if errReceived != tc.errExpected {
				t.Errorf("got unexpected error status: %v", errReceived)
			}

			if !errReceived && tc.want != got {
				t.Errorf("want %s, got %s", tc.want, got)
			}
		})
	}
}
