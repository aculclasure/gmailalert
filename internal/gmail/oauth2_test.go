package gmail_test

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aculclasure/gmailalert/internal/gmail"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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

func TestNewOAuth2RedirectServerWithInvalidListenerPortReturnsError(t *testing.T) {
	t.Parallel()
	testCases := map[string]int{
		"listener port smaller than 1024": -1025,
		"listener port bigger than 65535": 70000,
	}
	for name, port := range testCases {
		t.Run(name, func(t *testing.T) {
			_, err := gmail.NewOAuth2RedirectServer(port)
			if err == nil {
				t.Error("expected an error but did not get one")
			}
		})
	}
}

func TestNewOAuth2RedirectServerWithValidListenerPortReturnsValidOAuth2RedirectServer(t *testing.T) {
	t.Parallel()
	validListenerPort := 4000
	want := &gmail.OAuth2RedirectServer{Port: validListenerPort}
	got, err := gmail.NewOAuth2RedirectServer(validListenerPort)

	if err != nil {
		t.Errorf("gmail.NewOAuth2RedirectServer(%d) returned unexpected error: %s", validListenerPort, err)
	}

	ignoreOpt := cmpopts.IgnoreUnexported(gmail.OAuth2RedirectServer{})
	if !cmp.Equal(want, got, ignoreOpt) {
		cmp.Diff(want, got, ignoreOpt)
	}
}

func TestOAuth2RedirectServer_HandlerErrorCases(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		requestMethod string
		requestURI    string
		wantRespCode  int
	}{
		"RequestWithInvalidHttpMethod": {
			requestMethod: http.MethodHead,
			wantRespCode:  http.StatusMethodNotAllowed,
		},
		"RequestURLMissingStateQueryParam": {
			requestMethod: http.MethodGet,
			requestURI:    "/?code=asdfadsf_afsa4234l",
			wantRespCode:  http.StatusBadRequest,
		},
		"RequestURLMissingCodeQueryParam": {
			requestMethod: http.MethodGet,
			requestURI:    "/?state=state-token",
			wantRespCode:  http.StatusBadRequest,
		},
		"RequestURLWithEmptyCodeQueryParam": {
			requestMethod: http.MethodGet,
			requestURI:    "/?state=state-token&code=",
			wantRespCode:  http.StatusBadRequest,
		},
	}
	httpClient := &http.Client{Timeout: 5 * time.Second}
	svrPort := 9001
	svr, err := gmail.NewOAuth2RedirectServer(svrPort)
	if err != nil {
		t.Errorf("NewOAuth2RedirectServer(%d) returned unexpected error: %s", svrPort, err)
	}

	go func() {
		svr.ListenAndServe()
	}()
	defer svr.Shutdown()
	svrAddr := fmt.Sprintf("localhost:%d", svrPort)
	waitForServer(t, svrAddr)
	notificationTimeout := 100 * time.Millisecond

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			req, err := http.NewRequest(tc.requestMethod, "http://"+svrAddr+tc.requestURI, nil)
			if err != nil {
				t.Error(err)
			}
			resp, err := httpClient.Do(req)
			if err != nil {
				t.Error(err)
			}
			defer resp.Body.Close()
			if tc.wantRespCode != resp.StatusCode {
				t.Errorf("want response code %d, got %d", tc.wantRespCode, resp.StatusCode)
			}

			select {
			case <-svr.NotifyError():
				return
			case <-time.After(notificationTimeout):
				t.Errorf("expected an error notification but did not receive one within %s", notificationTimeout)
			}
		})
	}
}

// waitForServer attempts to establish a TCP connection to addr in a given
// amount of time. It returns upon successfully connecting. Otherwise it crashes
// the calling test with an error.
// Credit belongs to https://stackoverflow.com/a/56865986
func waitForServer(t *testing.T, addr string) {
	t.Helper()

	backoff := 50 * time.Millisecond

	for i := 0; i < 10; i++ {
		conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
		if err != nil {
			time.Sleep(backoff)
			continue
		}
		err = conn.Close()
		if err != nil {
			t.Fatal(err)
		}
		return
	}

	t.Fatalf("server on addr %s not up after 10 attempts", addr)
}
