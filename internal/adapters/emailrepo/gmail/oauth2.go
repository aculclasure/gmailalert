package gmail

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

type Logger interface {
	Printf(string, ...interface{})
}

// OAuth2Opt represents a functional option that can be applied to an OAuth2.
type OAuth2Opt func(*OAuth2)

// WithTokenFile accepts a filename for a file containing an OAuth2 token and
// returns an OAuth2Opt that wires the token file into an OAuth2 struct.
func WithTokenFile(tokFile string) OAuth2Opt {
	return func(o *OAuth2) {
		o.TokenFile = tokFile
	}
}

// WithRedirectServerport accepts a port number and returns an OAuth2Opt that
// wires the port number into an OAuth2 struct.
func WithRedirectServerPort(port int) OAuth2Opt {
	return func(o *OAuth2) {
		o.RedirectServerPort = port
	}
}

// WithLogger accepts a Logger and returns an OAuth2Opt that wires the logger
// into an OAuth2 struct.
func WithLogger(logger Logger) OAuth2Opt {
	return func(o *OAuth2) {
		o.logger = logger
	}
}

// The OAuth2 type contains fields needed for communicating with the Google
// OAuth2 provider.
type OAuth2 struct {
	// The user's Google console client credentials in JSON format.
	GoogleCfg []byte
	// The name of the file containing the JSON-formatted OAuth2 token.
	TokenFile string
	// The port that the OAuth2 redirect server should listen on for requests
	// from the Google OAuth2 resource provider. This is necessary when the
	// OAuth2 token must be remotely fetched.
	RedirectServerPort int
	cfg                *oauth2.Config
	tok                *oauth2.Token
	logger             Logger
}

// NewOAuth2 accepts a JSON Google configuration (typically read in from the
// user's config.json file) and returns an OAuth2 struct. An error is returned
// if the Google configuration is nil, cannot be read, or is empty.
func NewOAuth2(googleCfg io.Reader, opts ...OAuth2Opt) (*OAuth2, error) {
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
	o := &OAuth2{
		GoogleCfg:          cfgBytes,
		TokenFile:          "token.json",
		RedirectServerPort: 9999,
		logger:             log.New(io.Discard, "", log.LstdFlags)}
	for _, opt := range opts {
		opt(o)
	}

	return o, nil
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
	o.logger.Printf("successfully loaded google oauth2 configuration: %+v", cfg)
	o.cfg = cfg
	return nil
}

// LoadToken attempts to load an OAuth2 token from the TokenFile pointed to by
// o. If the attempt to read the token from a local file fails, then a remote
// token fetch attempt is made. An error is returned if there is a problem
// fetching the token remotely or loading the fetched token.
func (o *OAuth2) LoadToken() error {
	err := o.loadLocalToken()
	if err != nil {
		o.logger.Printf("got error when attempting to load an oauth2 token from local file: %s: %s", o.TokenFile, err)
		err = o.loadRemoteToken()
		if err != nil {
			return err
		}
		o.logger.Printf("successfully loaded an oauth2 token via a remote call: %+v", o.tok)
		return nil
	}
	o.logger.Printf("successfully loaded an oauth2 token from local file %s: %+v", o.TokenFile, o.tok)
	return nil
}

func (o *OAuth2) loadLocalToken() error {
	f, err := os.Open(o.TokenFile)
	if err != nil {
		return err
	}
	defer f.Close()

	var tok oauth2.Token
	err = json.NewDecoder(f).Decode(&tok)
	if err != nil {
		return err
	}

	o.tok = &tok
	return nil
}

func (o *OAuth2) loadRemoteToken() error {
	svr, err := NewOAuth2RedirectServer(o.RedirectServerPort)
	if err != nil {
		return err
	}
	defer svr.Shutdown()
	go func() {
		svr.ListenAndServe()
	}()

	authURL := o.cfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("To continue, please open a web browser and go to the following URL: %s\n", authURL)
	var code string
	select {
	case code = <-svr.NotifyAuthCode():
	case err = <-svr.NotifyError():
		return err
	}

	tok, err := o.cfg.Exchange(context.Background(), code)
	if err != nil {
		return err
	}
	o.tok = tok
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

// SaveToken writes the privately stored OAuth2 token into the given file. An
// error is returned if there is a problem writing the token into the file.
func (o *OAuth2) SaveToken(file string) error {
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	err = json.NewEncoder(f).Encode(o.tok)
	if err != nil {
		return err
	}
	return nil
}

// Client returns an HTTP client that is OAuth2-enabled for communicating with
// the Gmail API. An error is returned if the privately stored OAuth2
// configuration or token fields are nil.
func (o *OAuth2) Client() (*http.Client, error) {
	if o.cfg == nil {
		return nil, errors.New("oauth2 configuration must be non-nil")
	}
	if o.tok == nil {
		return nil, errors.New("oauth2 token must be non-nil")
	}
	return o.cfg.Client(context.Background(), o.tok), nil
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
