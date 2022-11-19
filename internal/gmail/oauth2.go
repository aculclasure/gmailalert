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

type OAuth2 struct {
	GoogleCfg []byte
	cfg       *oauth2.Config
	tok       *oauth2.Token
}

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

func (o *OAuth2) LoadConfig() error {
	cfg, err := google.ConfigFromJSON(o.GoogleCfg, gmail.GmailReadonlyScope)
	if err != nil {
		return err
	}

	o.cfg = cfg
	return nil
}

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

func NewOAuth2RedirectServer(port int) (*OAuth2RedirectServer, error) {
	if port < 1024 || port > 65535 {
		return nil, fmt.Errorf("port must be in the range 1024-65535 (got %d)", port)
	}

	// authCodes := make(chan string, 1)
	// authCodeErrors := make(chan error, 1)

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
	//rcv := &receiver{authCodes, authCodeErrors}
	redirectSvr.svr.Handler = http.HandlerFunc(redirectSvr.Handler)

	return redirectSvr, nil
}

func (o *OAuth2RedirectServer) NotifyAuthCode() <-chan string {
	return o.authCodes
}

func (o *OAuth2RedirectServer) NotifyError() <-chan error {
	return o.authCodeErrors
}

func (o *OAuth2RedirectServer) ListenAndServe() error {
	err := o.svr.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// Shutdown gracefully shuts down the HTTP server wrapped by o. If the server
// is not shutdown within 5 seconds, then it is force-stopped.
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

// type receiver struct {
// 	authCodes      chan string
// 	authCodeErrors chan error
// }

// func (r *receiver) receiveAuthCodeHandler(w http.ResponseWriter, req *http.Request) {
// 	if req.Method != http.MethodGet {
// 		errMsg := fmt.Sprintf("request method must be an http get (got %s)", req.Method)
// 		http.Error(w, errMsg, http.StatusMethodNotAllowed)
// 		r.authCodeErrors <- errors.New(errMsg)
// 		return
// 	}

// 	queryString := req.URL.Query()
// 	paramVal := queryString.Get("state")
// 	if paramVal != "state-token" {
// 		errMsg := `request must contain a query parameter "state=state-token"`
// 		http.Error(w, errMsg, http.StatusBadRequest)
// 		r.authCodeErrors <- errors.New(errMsg)
// 		return
// 	}

// 	paramVal = queryString.Get("code")
// 	if paramVal == "" {
// 		errMsg := `request must contain a non-empty query parameter "code"`
// 		http.Error(w, errMsg, http.StatusBadRequest)
// 		r.authCodeErrors <- errors.New(errMsg)
// 		return
// 	}

// 	w.Write([]byte("Successfully read authorization code sent by OAuth2 resource provider!"))
// 	r.authCodes <- paramVal
// }
