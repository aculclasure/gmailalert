package gmailalert

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

const defaultTokenFile = "token.json"

// GmailClientConfig represents the configuration needed to create a GmailClient.
type GmailClientConfig struct {
	// The file containing the user's Google Developers Console credentials.
	CredentialsFile string
	// The file containing the user's Gmail OAuth2 token.
	TokenFile string
	// The input source for entering the Gmail OAuth2 authentication code.
	UserInput io.Reader
	// The port that the local HTTP server should listen on for handling
	// redirect requests from the Gmail OAuth2 resource provider.
	RedirectSvrPort int
	// The Logger to use for debugging.
	Logger Logger
}

// OK returns an error if the given GmailClientConfig contains invalid values
// for the Gmail OAuth2 credentials file, the user input source, or the port
// that the local HTTP server should listen on for redirect requests coming from
// the Gmail OAuth2 resource provider.
func (g GmailClientConfig) OK() error {
	if g.CredentialsFile == "" {
		return errors.New("credentials file name must not be empty")
	}

	if g.UserInput == nil {
		return errors.New("user input reader must not be nil")
	}

	if g.RedirectSvrPort < 1 {
		return errors.New("redirect server port must not be negative")
	}

	return nil
}

// GmailClient represents a client for communicating with the Gmail API.
type GmailClient struct {
	svc *gmail.Service
}

// NewGmailClient accepts a GmailClientConfig and returns a new GmailClient.
// An error is returned if the GmailClientConfig is invalid, if the gmail oauth2
// configuration cannot be generated, or if there is a problem creating the
// gmail service.
func NewGmailClient(cfg GmailClientConfig) (*GmailClient, error) {
	if err := cfg.OK(); err != nil {
		return nil, fmt.Errorf("got error validating gmail client config: %s", err)
	}

	if cfg.Logger == nil {
		cfg.Logger = log.New(io.Discard, "", log.LstdFlags)
	}

	oauth := &gmailOAuth2{GmailClientConfig: cfg}
	if err := oauth.initializeConfig(); err != nil {
		return nil, fmt.Errorf("got error initializing gmail oauth: %s", err)
	}
	cfg.Logger.Printf("successfully initialized google oauth2 configuration: %s", oauth.oauthCfg)

	httpClient, err := oauth.client()
	if err != nil {
		return nil, fmt.Errorf("got error creating oauth2-enabled http client: %s", err)
	}

	svc, err := gmail.NewService(context.Background(), option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("got error creating new gmail service: %s", err)
	}

	return &GmailClient{svc: svc}, nil
}

// Match queries Gmail for any emails matching the given query, which can be any
// valid Gmail query expression, like "is:unread", "from:gopher@gmail.com", etc.
// It returns a slice of raw email messages matching the query
// where raw means the email message is RFC 2822 formatted and base64 encoded.
// An error is returned if the query to the Gmail API fails.
func (g GmailClient) Match(query string) ([]string, error) {
	resp, err := g.svc.Users.Messages.List("me").Q(query).Do()
	if err != nil {
		return nil, fmt.Errorf("got error executing gmail query %s: %v", query, err)
	}

	return prepareMatchResp(resp.Messages), nil
}

// gmailOAuth2 provides behavior for handling the OAuth2 requests to the Gmail
// API.
type gmailOAuth2 struct {
	GmailClientConfig
	oauthCfg *oauth2.Config
}

// initializeConfig generates an oauth2.Config from a Google Developers Console
// credentials file and returns it. An error is returned if there is a problem
// opening the credentials file or if the credentials data is invalid.
func (g *gmailOAuth2) initializeConfig() error {
	g.Logger.Printf("building gmail oauth2 configuration from google credentials file %s", g.CredentialsFile)
	f, err := os.Open(g.CredentialsFile)
	if err != nil {
		return err
	}
	defer f.Close()

	req, err := prepareConfigRequest(f)
	if err != nil {
		return err
	}

	cfg, err := google.ConfigFromJSON(req.credentials, req.scope)
	if err != nil {
		return err
	}

	g.oauthCfg = cfg

	return nil
}

// token() attempts to retrive the Gmail OAuth2 token from a local file. If that
// fails, it attempts to fetch the token from the Gmail OAuth2 resource
// provider. An error is returned if no OAuth2 token can be determined.
func (g gmailOAuth2) token() (*oauth2.Token, error) {
	tok, err := g.localToken()
	if err == nil {
		g.Logger.Printf("successfully read gmail oauth2 token from file %s: %+q", g.TokenFile, tok)
		return tok, nil
	}

	g.Logger.Printf("unable to read gmail oauth2 token from local file %s, attempting to fetch token from remote resource provider", g.TokenFile)
	tok, err = g.remoteToken()
	if err != nil {
		return nil, fmt.Errorf("got error when remotely fetching gmail oauth2 token: %s", err)
	}
	g.Logger.Printf("successfully fetched gmail oauth2 token from remote resource provider: %+q", tok)

	if g.TokenFile == "" {
		g.TokenFile = defaultTokenFile
	}

	err = saveToken(g.TokenFile, tok)
	if err != nil {
		g.Logger.Printf("got error saving token to file: %s", err)
	}
	g.Logger.Printf("successfully wrote gmail oauth2 token to file %s", g.TokenFile)

	return tok, nil
}

// localToken attemps to create a Gmail OAuth2 token from a local file. If
// successful, then the token is returned. Otherwise, an error is returned.
func (g gmailOAuth2) localToken() (*oauth2.Token, error) {
	f, err := os.Open(g.TokenFile)
	if err != nil {
		return nil, fmt.Errorf("got error opening gmail oauth2 token file %s: %v", g.TokenFile, err)
	}
	defer f.Close()

	var tok oauth2.Token
	err = json.NewDecoder(f).Decode(&tok)
	if err != nil {
		return nil, fmt.Errorf("got error json-decoding gmail oauth2 token: %v", err)
	}

	return &tok, nil
}

// remoteToken attempts to create a Gmail OAuth2 token by first capturing an
// authorization code from user input and then exchanging that authorization
// code for a token. The token is returned if it is successfully exchanged for
// the auth code. Otherwise, an error is returned.
func (g gmailOAuth2) remoteToken() (*oauth2.Token, error) {
	authURL := g.oauthCfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	g.Logger.Printf("generated gmail oauth2 exchange url for getting the authentication code: %s", authURL)
	authCode, err := getAuthCode(authURL, g.UserInput, g.RedirectSvrPort)
	if err != nil {
		return nil, fmt.Errorf("got error retrieving oauth2 auth code: %v", err)
	}
	g.Logger.Printf("got authentication code from user input: %s", authCode)

	return g.oauthCfg.Exchange(context.Background(), authCode)
}

// client returns an HTTP client that is configured for sending requests to the
// Gmail API using an OAuth2 access token. An error is returned if there is
// problem reading the Google Developers Console credentials or generating the
// Gmail OAuth2 access token.
func (g *gmailOAuth2) client() (*http.Client, error) {
	tok, err := g.token()
	if err != nil {
		return nil, fmt.Errorf("got error fetching gmail oauth2 token: %s", err)
	}

	return g.oauthCfg.Client(context.Background(), tok), nil
}

// saveToken accepts a file name and and OAuth2 token and saves the token into
// the file. An error is returned if there is a problem opening the file or
// writing the token into the file.
func saveToken(file string, token *oauth2.Token) error {
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("got error opening file %s to save gmail oauth2 token into: %s", file, err)
	}
	defer f.Close()

	err = json.NewEncoder(f).Encode(token)
	if err != nil {
		return fmt.Errorf("got error writing gmail oauth2 token into file %s: %s", file, err)
	}

	return nil
}

// getAuthCode accepts the URL of a Gmail OAuth2 resource provider, an io.Reader
// for reading user input, and a port number for the local local HTTP server to
// listen on for redirects from the Gmail OAuth2 resource provider. After the
// user navigates their web browser to the authURL, the Gmail OAuth2 resource
// provider redirects back to a local HTTP server with the authorization code.
// The user is prompted to enter the authorization code shown by the local HTTP
// server. The value entered by the user is returned as a string. An error is
// returned if any of the function's arguments are invalid or if there is
// problem reading the user's input.
func getAuthCode(authURL string, userInput io.Reader, redirectSvrPort int) (string, error) {
	_, err := url.ParseRequestURI(authURL)
	if err != nil {
		return "", fmt.Errorf("got error parsing url %s: %s", authURL, err)
	}
	if userInput == nil {
		return "", errors.New("user input must be non-nil")
	}
	if redirectSvrPort < 1 {
		return "", errors.New("redirect server port must be a positive number")
	}

	redirectSvr := NewRedirectServer(WithRedirectSvrAddr(fmt.Sprintf("127.0.0.1:%d", redirectSvrPort)))
	go func() {
		redirectSvr.ListenAndServe()
	}()
	defer redirectSvr.Shutdown()

	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)
	var authCode string
	if _, err := fmt.Fscan(userInput, &authCode); err != nil {
		return "", fmt.Errorf("got error reading auth code from user input: %v", err)
	}

	return authCode, nil
}

// configRequest represents a type containing the arguments that are expected in
// the google.ConfigFromJSON function.
type configRequest struct {
	credentials []byte
	scope       string
}

// prepareConfigRequest accepts an io.Reader containing a user's Google
// Developers Console credentials, ensures the credentials data is valid and
// and returns a configRequest struct. An error is returned if the credentials
// data is invalid.
func prepareConfigRequest(cfgData io.Reader) (configRequest, error) {
	var req configRequest

	c, err := io.ReadAll(cfgData)
	if err != nil {
		return req, fmt.Errorf("got error reading credentials data: %s", err)
	}

	if len(c) == 0 {
		return req, errors.New("credentials data must not be empty")
	}

	req.credentials, req.scope = c, gmail.GmailReadonlyScope

	return req, nil
}

// prepareMatchResp accepts a slice of gmail.Message, iterates through them,
// and returns a slice of raw (RFC 2822-formatted, base64-encoded) email
// messages.
func prepareMatchResp(msgs []*gmail.Message) []string {
	rawMsgs := make([]string, 0, len(msgs))
	for _, m := range msgs {
		rawMsgs = append(rawMsgs, m.Raw)
	}

	return rawMsgs
}
