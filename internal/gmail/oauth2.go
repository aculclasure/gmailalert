package gmail

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

// The OAuth2 type contains fields needed for communicating with the Google
// OAuth2 provider.
type OAuth2 struct {
	// The user's Google console client credentials in JSON format.
	GoogleCfg []byte
	cfg       *oauth2.Config
	tok       *oauth2.Token
}

// NewOAuth2 accepts a JSON Google configuration (typically read in from the
// user's config.json file) and returns an OAuth2 struct. An error is returned
// if the Google configuration is nil, cannot be read, or is empty.
func NewOAuth2(googleCfg io.Reader) (*OAuth2, error) {
	if googleCfg == nil {
		return nil, errors.New("google configuration must not be nil")
	}
	cfgBytes, err := io.ReadAll(googleCfg)
	if err != nil {
		return nil, fmt.Errorf("got unexpected error reading google configuration: %s", err)
	}

	if len(cfgBytes) == 0 {
		return nil, errors.New("google configuration must not be empty")
	}

	return &OAuth2{GoogleCfg: cfgBytes}, nil
}

// LoadConfig initializes the private OAuth2 configuration of the *OAuth2 receiver using
// the data stored in the receiver's GoogleCfg field. An error is returned if
// there is an issue creating the *oauth2.Config that is privately stored in the
// *OAuth2 receiver.
func (o *OAuth2) LoadConfig() error {
	cfg, err := google.ConfigFromJSON(o.GoogleCfg, gmail.GmailReadonlyScope)
	if err != nil {
		return err
	}

	o.cfg = cfg
	return nil
}

// LoadToken accepts an OAuth2 token reader and uses it to initialize the
// private *oauth2.Token field in the *OAuth2 receiver. An error is returned if
// the token reader is nil or cannot be decoded into a valid oauth2.Token struct.
func (o *OAuth2) LoadToken(token io.Reader) error {
	if token == nil {
		return errors.New("token must not be nil")
	}

	var tok oauth2.Token
	err := json.NewDecoder(token).Decode(&tok)
	if err != nil {
		return err
	}

	o.tok = &tok
	return nil
}

// GetToken returns the privately stored OAuth2 token in the *OAuth2 receiver as
// a slice of bytes. An error is returned if the underlying OAuth2 token is nil
// or if there is problem encoding the underlying OAuth2 token into a byte slice.
func (o *OAuth2) GetToken() ([]byte, error) {
	if o.tok == nil {
		return nil, errors.New("underlying oauth2 token in oauth2 struct must not be nil")
	}

	bfr := new(bytes.Buffer)
	err := json.NewEncoder(bfr).Encode(o.tok)
	if err != nil {
		return nil, err
	}

	return bfr.Bytes(), nil
}

// OAuth2RedirectServer represents an HTTP server that handles oauth2 redirect
// requests and displays the state token returned by the oauth2 resource
// provider.
type OAuth2RedirectServer struct {
	Port           int
	authCodes      chan string
	authCodeErrors chan error
	svr            *http.Server
}

// NewOAuth2RedirectServer accepts a listener port and returns an OAuth2RedirectServer
// struct. An error is returned if the port is invalid (e.g. not in the ephemeral
// port range 1024-65525).
func NewOAuth2RedirectServer(port int) (*OAuth2RedirectServer, error) {
	if port < 1024 || port > 65535 {
		return nil, fmt.Errorf("port must be in the range 1024-65535 (got %d)", port)
	}

	redirectSvr := &OAuth2RedirectServer{
		Port:           port,
		authCodes:      make(chan string, 1),
		authCodeErrors: make(chan error, 1),
		svr: &http.Server{
			Addr:         fmt.Sprintf("localhost:%d", port),
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
	}
	redirectSvr.svr.Handler = http.HandlerFunc(redirectSvr.Handler)

	return redirectSvr, nil
}

// NotifyAuthCode returns a receive-only channel which receives OAuth2 auth codes
// from the OAuth2RedirectServer's Handle method when it handles a successful request
// from the Google OAuth2 provider.
func (o *OAuth2RedirectServer) NotifyAuthCode() <-chan string {
	return o.authCodes
}

// NotifyError returns a receive-only channel which receives any errors encountered
// by the OAuth2RedirectServer's Handle method.
func (o *OAuth2RedirectServer) NotifyError() <-chan error {
	return o.authCodeErrors
}

// ListenAndServe starts the OAuth2RedirectServer. An error is returned if the
// underlying HTTP server encounters any error other than the standard server
// closed error.
func (o *OAuth2RedirectServer) ListenAndServe() error {
	err := o.svr.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// Shutdown gracefully shuts down the HTTP server wrapped by the OAuth2RedirectServer.
// If the server is not shutdown within 5 seconds, then it is force-stopped.
func (o *OAuth2RedirectServer) Shutdown() error {
	close(o.authCodeErrors)
	close(o.authCodes)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := o.svr.Shutdown(ctx)
	if err != nil {
		return err
	}

	return nil
}

// Handler receives OAuth2 redirect requests from the Google OAuth2 provider,
// extracts the auth code field from the request's URL and returns an HTTP response
// indicating the auth code was successfully extracted. The extracted auth code is
// also sent to the OAuth2RedirectServer's underyling auth code channel (which
// can be accessed via the NotifyAuthCode() method.) An error HTTP response is sent
// if the incoming request is not an HTTP GET request, or if it has an invalid URL
// (e.g. missing the "state" and "code" query parameters). Any errors also are sent to
// the OAuth2RedirectServer's underlying error channel (which can be accessed via
// the NotifyError() method).
func (o *OAuth2RedirectServer) Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		errMsg := fmt.Sprintf("request method must be an http get (got %s)", r.Method)
		http.Error(w, errMsg, http.StatusMethodNotAllowed)
		o.authCodeErrors <- errors.New(errMsg)
		return
	}

	queryString := r.URL.Query()
	paramVal := queryString.Get("state")
	if paramVal != "state-token" {
		errMsg := `request must contain a query parameter "state=state-token"`
		http.Error(w, errMsg, http.StatusBadRequest)
		o.authCodeErrors <- errors.New(errMsg)
		return
	}

	paramVal = queryString.Get("code")
	if paramVal == "" {
		errMsg := `request must contain a non-empty query parameter "code"`
		http.Error(w, errMsg, http.StatusBadRequest)
		o.authCodeErrors <- errors.New(errMsg)
		return
	}

	w.Write([]byte("Successfully read authorization code sent by OAuth2 resource provider!"))
	o.authCodes <- paramVal
}
