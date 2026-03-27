package oauth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
)

// CallbackServer is a temporary HTTP server that waits for an OAuth redirect.
type CallbackServer struct {
	listener net.Listener
	server   *http.Server
	codeCh   chan string
	errCh    chan error
	once     sync.Once
}

// NewCallbackServer creates a CallbackServer ready to start.
func NewCallbackServer() *CallbackServer {
	return &CallbackServer{
		codeCh: make(chan string, 1),
		errCh:  make(chan error, 1),
	}
}

// Start binds a random port and begins listening.
// Returns the full callback URL, e.g. "http://localhost:54321/callback".
func (s *CallbackServer) Start() (callbackURL string, err error) {
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", fmt.Errorf("callback server: listen: %w", err)
	}
	s.listener = ln

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.handleCallback)
	s.server = &http.Server{Handler: mux}

	go func() {
		if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			s.once.Do(func() { s.errCh <- err })
		}
	}()

	port := ln.Addr().(*net.TCPAddr).Port
	return fmt.Sprintf("http://localhost:%d/callback", port), nil
}

// handleCallback handles the OAuth redirect, extracts the code, and signals WaitForCode.
func (s *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	errParam := r.URL.Query().Get("error")

	if errParam != "" {
		desc := r.URL.Query().Get("error_description")
		if desc == "" {
			desc = errParam
		}
		s.once.Do(func() { s.errCh <- fmt.Errorf("oauth error: %s", desc) })
		http.Error(w, "Authentication failed: "+desc, http.StatusBadRequest)
		return
	}

	if code == "" {
		s.once.Do(func() { s.errCh <- fmt.Errorf("no code in callback") })
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	s.once.Do(func() { s.codeCh <- code })
	fmt.Fprintln(w, "Authentication successful! You can close this window.")
}

// WaitForCode blocks until the authorization code arrives or ctx is cancelled.
func (s *CallbackServer) WaitForCode(ctx context.Context) (code string, err error) {
	select {
	case code := <-s.codeCh:
		return code, nil
	case err := <-s.errCh:
		return "", err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// Close shuts down the callback server.
func (s *CallbackServer) Close() {
	if s.server != nil {
		_ = s.server.Close()
	}
}
