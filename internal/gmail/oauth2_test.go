package gmail_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/aculclasure/gmailalert/internal/gmail"
	"github.com/google/go-cmp/cmp"
)

func TestNewOAuth2ErrorCases(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		input io.Reader
	}{
		"NewOAuth2WithNilGoogleConfigReturnsError": {
			input: nil,
		},
		"NewOAuth2WithEmptyGoogleConfigReturnsError": {
			input: strings.NewReader(""),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			_, err := gmail.NewOAuth2(tc.input)
			if err == nil {
				t.Error("expected an error but did not get one")
			}
		})
	}
}

func TestNewOAuth2WithValidGoogleConfigReturnsConfiguredOAuth2Struct(t *testing.T) {
	t.Parallel()
	validCfg := `{"installed":{"client_id":"ID","project_id":"PROJECTID","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","auth_provider_x509_cert_url":"https://www.googleapis.com/oauth2/v1/certs","client_secret":"SECRET","redirect_uris":["http://localhost:9999"]}}`
	googleCfg := strings.NewReader(validCfg)
	want := []byte(validCfg)

	oauth, err := gmail.NewOAuth2(googleCfg)
	if err != nil {
		t.Errorf("got unexpected error: %s", err)
	}

	got := oauth.GoogleCfg
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}

func TestLoadTokenErrorCases(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		input io.Reader
	}{
		"LoadTokenWithNilTokenReaderReturnsError": {
			input: nil,
		},
		"LoadTokenWithEmptyTokenReturnsError": {
			input: strings.NewReader(""),
		},
		"LoadTokenWithInvalidTokenJSONReturnsError": {
			input: strings.NewReader(`{"access_token":`),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			oauth := &gmail.OAuth2{}
			err := oauth.LoadToken(tc.input)
			if err == nil {
				t.Error("expected an error but did not get one")
			}
		})
	}
}

func TestLoadTokenWithValidTokenJSONLoadsTokenInOAuth2Struct(t *testing.T) {
	t.Parallel()
	want := []byte(`{"access_token":"ACCESSTOKEN","token_type":"Bearer","refresh_token":"REFRESHTOKEN","expiry":"2022-08-16T12:00:42.516357003-04:00"}
`)
	oauth := &gmail.OAuth2{}

	err := oauth.LoadToken(bytes.NewReader(want))
	if err != nil {
		t.Fatalf("got unexpected error: %s", err)
	}
	got, err := oauth.GetToken()
	if err != nil {
		t.Fatalf("got unexpected error: %s", err)
	}
	if !cmp.Equal(want, got) {
		t.Error(cmp.Diff(want, got))
	}
}
