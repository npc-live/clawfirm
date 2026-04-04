package remote

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	ngrok "golang.ngrok.com/ngrok/v2"
)

// tunnel wraps an ngrok listener.
type tunnel struct {
	listener net.Listener
	url      string
	cancel   context.CancelFunc
	agent    ngrok.Agent
}

// close shuts down the ngrok tunnel.
func (t *tunnel) close() {
	if t.listener != nil {
		t.listener.Close()
	}
	if t.agent != nil {
		t.agent.Disconnect()
	}
	if t.cancel != nil {
		t.cancel()
	}
}

// StartTunnel starts an ngrok tunnel that serves the same handler as the LAN server.
// The public URL is available via Status().NgrokURL.
func (s *Server) StartTunnel(ctx context.Context, authToken string) error {
	s.mu.Lock()
	if s.tunnel != nil {
		s.mu.Unlock()
		return fmt.Errorf("remote: ngrok tunnel already running")
	}
	s.mu.Unlock()

	tunCtx, cancel := context.WithCancel(ctx)

	agent, err := ngrok.NewAgent(ngrok.WithAuthtoken(authToken))
	if err != nil {
		cancel()
		return fmt.Errorf("remote: ngrok agent: %w", err)
	}
	if err := agent.Connect(tunCtx); err != nil {
		cancel()
		return fmt.Errorf("remote: ngrok connect: %w", err)
	}

	listener, err := agent.Listen(tunCtx)
	if err != nil {
		agent.Disconnect()
		cancel()
		return fmt.Errorf("remote: ngrok listen: %w", err)
	}

	tun := &tunnel{
		listener: listener,
		url:      listener.URL().String(),
		cancel:   cancel,
		agent:    agent,
	}

	s.mu.Lock()
	s.tunnel = tun
	s.mu.Unlock()

	mux := s.buildMux()
	go func() {
		if err := http.Serve(listener, mux); err != nil {
			log.Printf("remote: ngrok serve: %v", err)
		}
	}()

	log.Printf("remote: ngrok tunnel started: %s", tun.url)
	return nil
}

// StopTunnel shuts down the ngrok tunnel without affecting the LAN server.
func (s *Server) StopTunnel() {
	s.mu.Lock()
	tun := s.tunnel
	s.tunnel = nil
	s.mu.Unlock()

	if tun != nil {
		tun.close()
		log.Printf("remote: ngrok tunnel stopped")
	}
}
