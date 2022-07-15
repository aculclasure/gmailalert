package gmailalert_test

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/aculclasure/gmailalert"
)

func TestReceiveAuthCodeHandlerWithInvalidHttpMethodReturnsErrorResponse(t *testing.T) {
	t.Parallel()

	svrAddr := "127.0.0.1:9001"
	svr := gmailalert.NewRedirectServer(gmailalert.WithRedirectSvrAddr(svrAddr))
	go func() {
		svr.ListenAndServe()
	}()
	defer svr.Shutdown()
	waitForServer(t, svrAddr)

	resp, err := http.Head("http://" + svrAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	got := resp.StatusCode
	want := http.StatusMethodNotAllowed

	if got != want {
		t.Fatalf("want resp status code %d, got %d", want, got)
	}
}

func TestReceiveAuthCodeHandlerWithInvalidRequests(t *testing.T) {
	t.Parallel()

	svrAddr := "127.0.0.1:9002"
	svr := gmailalert.NewRedirectServer(gmailalert.WithRedirectSvrAddr(svrAddr))
	go func() {
		svr.ListenAndServe()
	}()
	defer svr.Shutdown()
	waitForServer(t, svrAddr)

	testCases := []struct {
		description    string
		input          string
		wantStatusCode int
		errExpected    bool
	}{
		{
			description:    "No query parameters returns HTTP bad request response",
			input:          "http://" + svrAddr,
			wantStatusCode: http.StatusBadRequest,
			errExpected:    false,
		},
		{
			description:    `Missing query parameter "state" returns HTTP bad request response`,
			input:          "http://" + svrAddr + "/?code=asdfadsf_afsa4234l",
			wantStatusCode: http.StatusBadRequest,
			errExpected:    false,
		},
		{
			description:    `Empty "code" query parameter returns HTTP bad request response`,
			input:          "http://" + svrAddr + "/?state=state-token&code=",
			wantStatusCode: http.StatusBadRequest,
			errExpected:    false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			resp, err := http.Get(tc.input)
			if err != nil {
				if !tc.errExpected {
					t.Errorf("got unexpected error: %v", err)
				}
				return
			}
			defer resp.Body.Close()

			got := resp.StatusCode
			if tc.wantStatusCode != got {
				t.Errorf("want status code %d, got %d", tc.wantStatusCode, got)
			}

		})
	}
}

func TestReceiveAuthCodeHandlerWithValidRequestWritesAuthCode(t *testing.T) {
	t.Parallel()

	svrAddr := "127.0.0.1:9003"
	svr := gmailalert.NewRedirectServer(gmailalert.WithRedirectSvrAddr(svrAddr))
	go func() {
		svr.ListenAndServe()
	}()
	defer svr.Shutdown()
	waitForServer(t, svrAddr)

	wantAuthCode := "abcdef__999asfb_zzrkrlyadfa88312"
	url := "http://" + svrAddr + "/?state=state-token&code=" + wantAuthCode
	resp, err := http.Get(url)
	if err != nil {
		t.Error(err)
	}
	defer resp.Body.Close()

	gotBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
	}
	wantBody := []byte("Authorization Code: " + wantAuthCode)

	if !bytes.Equal(wantBody, gotBody) {
		t.Errorf("want body %s, got body %s", string(wantBody), string(gotBody))
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
