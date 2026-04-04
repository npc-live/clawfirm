// Package remote provides a mobile-accessible HTTP/WebSocket server for
// remote control of clawfirm agents. It listens on 0.0.0.0 (all interfaces)
// with token-based authentication, allowing phones on the same LAN — or via
// an optional ngrok tunnel — to interact with agents.
package remote

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ai-gateway/clawfirm/config"
	"github.com/ai-gateway/clawfirm/gateway"
	"github.com/ai-gateway/clawfirm/store"
)

// ChannelStatus describes the status of a channel (WhatsApp, Feishu, etc.).
type ChannelStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "connected", "disconnected", etc.
}

// RemoteStatus is returned to the frontend with connection info.
type RemoteStatus struct {
	Enabled  bool   `json:"enabled"`
	LanURL   string `json:"lanUrl"`
	NgrokURL string `json:"ngrokUrl"`
	QRCode   string `json:"qrCode"` // data:image/png;base64,...
	Token    string `json:"token"`
	Port     int    `json:"port"`
	LanIP    string `json:"lanIP"`
	NgrokOn  bool   `json:"ngrokOn"`
	Clients  int    `json:"clients"`
}

// Server is the remote-control HTTP server.
type Server struct {
	mu       sync.RWMutex
	httpSrv  *http.Server
	listener net.Listener
	token    string
	port     int
	lanIP    string

	registry        *gateway.AgentRegistry
	db              *store.DB
	cfg             *config.Config
	canvasDir       string
	channelStatusFn func() []ChannelStatus

	clients atomic.Int32

	// ngrok tunnel (optional)
	tunnel *tunnel
}

// Config configures the remote Server.
type Config struct {
	Registry        *gateway.AgentRegistry
	DB              *store.DB
	Cfg             *config.Config
	CanvasDir       string
	ChannelStatusFn func() []ChannelStatus
}

// NewServer creates a remote Server (not yet started).
func NewServer(c Config) *Server {
	return &Server{
		registry:        c.Registry,
		db:              c.DB,
		cfg:             c.Cfg,
		canvasDir:       c.CanvasDir,
		channelStatusFn: c.ChannelStatusFn,
	}
}

// Start begins listening on 0.0.0.0 with a random port.
func (s *Server) Start(ctx context.Context) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("remote: panic in Start: %v", r)
		}
	}()

	token, err := generateToken()
	if err != nil {
		return fmt.Errorf("remote: generate token: %w", err)
	}
	s.token = token

	ln, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return fmt.Errorf("remote: listen: %w", err)
	}
	s.listener = ln
	s.port = ln.Addr().(*net.TCPAddr).Port
	s.lanIP = discoverLANIP()

	mux := s.buildMux()
	s.httpSrv = &http.Server{Handler: mux}

	go func() {
		if err := s.httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("remote: serve: %v", err)
		}
	}()

	log.Printf("remote: listening on 0.0.0.0:%d  LAN IP: %s  token: %s", s.port, s.lanIP, s.token)
	return nil
}

// Stop shuts down the HTTP server and any ngrok tunnel.
func (s *Server) Stop() error {
	s.mu.Lock()
	tun := s.tunnel
	s.tunnel = nil
	s.mu.Unlock()

	if tun != nil {
		tun.close()
	}

	if s.httpSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return s.httpSrv.Shutdown(ctx)
	}
	return nil
}

// Status returns the current RemoteStatus.
func (s *Server) Status() RemoteStatus {
	s.mu.RLock()
	tun := s.tunnel
	s.mu.RUnlock()

	st := RemoteStatus{
		Enabled: true,
		Token:   s.token,
		Port:    s.port,
		LanIP:   s.lanIP,
		Clients: int(s.clients.Load()),
	}

	if s.lanIP != "" {
		st.LanURL = fmt.Sprintf("http://%s:%d/remote/?token=%s", s.lanIP, s.port, s.token)
	}

	if tun != nil {
		st.NgrokOn = true
		st.NgrokURL = tun.url + "/remote/?token=" + s.token
	}

	// QR code: prefer ngrok URL, fallback to LAN.
	qrURL := st.LanURL
	if st.NgrokURL != "" {
		qrURL = st.NgrokURL
	}
	if qrURL != "" {
		st.QRCode, _ = qrCodeToDataURL(qrURL, 300)
	}

	return st
}

// Token returns the auth token.
func (s *Server) Token() string { return s.token }

// ClientCount returns the number of connected WebSocket clients.
func (s *Server) ClientCount() int { return int(s.clients.Load()) }

// ─── Helpers ──────────────────────────────────────────────────────────────────

// generateToken returns a cryptographically random 32-character hex token.
func generateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// discoverLANIP returns the first private IPv4 address found on the machine.
func discoverLANIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() || ipNet.IP.To4() == nil {
			continue
		}
		ip := ipNet.IP.To4()
		if isPrivateIP(ip) {
			return ip.String()
		}
	}
	return ""
}

// isPrivateIP checks if an IPv4 address is in a private range.
func isPrivateIP(ip net.IP) bool {
	private := []net.IPNet{
		{IP: net.IPv4(192, 168, 0, 0), Mask: net.CIDRMask(16, 32)},
		{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
		{IP: net.IPv4(172, 16, 0, 0), Mask: net.CIDRMask(12, 32)},
	}
	for _, pn := range private {
		if pn.Contains(ip) {
			return true
		}
	}
	return false
}

// qrCodeToDataURL renders a QR code as a base64-encoded PNG data URL.
func qrCodeToDataURL(content string, size int) (string, error) {
	// Inline import to keep the dependency light.
	// The project already depends on github.com/skip2/go-qrcode.
	return qrEncode(content, size)
}
