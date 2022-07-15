package gmailalert

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// RedirectServer represents an HTTP server that handles oauth2 redirect
// requests and displays the state token returned by the oauth2 resource
// provider.
type RedirectServer struct {
	svr *http.Server
}

// NewRedirectServer accepts an optional slice of RedirectServerOpt functional
// options and returns a RedirectServer that is configured to handle redirects
// from an oauth2 resource provider to the local host. If the server address is
// not overridden with a RedirectServerOpt argument, then the server will listen
// on TCP port 9999.
func NewRedirectServer(opts ...RedirectServerOpt) *RedirectServer {
	rs := &RedirectServer{
		svr: &http.Server{
			Addr:         "localhost:9999",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 20 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(rs)
	}

	rs.svr.Handler = http.HandlerFunc(receiveAuthCodeHandler)

	return rs
}

// ListenAndServe listens on the TCP address configured in the HTTP server
// wrapped by r and sends all requests to the handler configured in the HTTP
// server wrapped by r.
func (r *RedirectServer) ListenAndServe() error {
	err := r.svr.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// Shutdown gracefully shuts down the HTTP server wrapped by r. If the server
// is not shutdown within 5 seconds, then it is force-stopped.
func (r *RedirectServer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := r.svr.Shutdown(ctx)
	if err != nil {
		return err
	}

	return nil
}

// receiveAuthCodeHandler receives redirect requests from an oauth2 resource
// provider, validates the requests, and writes the state token contained in the
// request's query parameters to w. An HTTP error response is returned if the
// request is not an HTTP GET or if the request does not contain the expected
// query parameters.
func receiveAuthCodeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "request method must be an HTTP GET, got "+r.Method, http.StatusMethodNotAllowed)
		return
	}

	queryString := r.URL.Query()
	got := queryString.Get("state")
	if got != "state-token" {
		http.Error(w, `request must contain a query parameter "state=state-token", got `+got, http.StatusBadRequest)
		return
	}
	got = queryString.Get("code")
	if got == "" {
		http.Error(w, `request must contain a non-empty query parameter "code"`, http.StatusBadRequest)
		return
	}

	w.Write([]byte("Authorization Code: " + got))
}

// RedirectServerOpt represents a functional option that can be used when
// creating a new RedirectServer.
type RedirectServerOpt func(*RedirectServer)

// WithAddr accepts a TCP address in the form "host:port" and returns a
// RedirectServerOpt that applies this address to a RedirectServer.
func WithRedirectSvrAddr(addr string) RedirectServerOpt {
	return func(rs *RedirectServer) {
		rs.svr.Addr = addr
	}
}
