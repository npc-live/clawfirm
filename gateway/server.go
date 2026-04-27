package gateway

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"sync"
)

// Server is the HTTP gateway server.
type Server struct {
	addr     string
	registry *AgentRegistry
	mux      *http.ServeMux
	httpSrv  *http.Server
	mu       sync.Mutex

	// dedup tracks recently seen message IDs to prevent double-delivery.
	dedup *dedupCache
}

// ServerConfig configures a Server.
type ServerConfig struct {
	Addr     string // default ":9988"
	Frontend fs.FS  // if set, serves static files with SPA fallback
}

// NewServer creates a Server backed by an AgentRegistry.
// Register channel handlers via Handle before calling Start.
func NewServer(registry *AgentRegistry, cfg ServerConfig) *Server {
	if cfg.Addr == "" {
		cfg.Addr = ":9988"
	}
	s := &Server{
		addr:     cfg.Addr,
		registry: registry,
		mux:      http.NewServeMux(),
		dedup:    newDedupCache(4096),
	}
	s.httpSrv = &http.Server{Addr: cfg.Addr, Handler: s.mux}

	// Health: {"status":"ok","agents":{"zenmux":{"sessions":2},"minimax":{"sessions":0}}}
	s.mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		counts := registry.Counts()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","agents":{`)
		first := true
		for _, name := range registry.Names() {
			if !first {
				fmt.Fprint(w, ",")
			}
			fmt.Fprintf(w, `%q:{"sessions":%d}`, name, counts[name])
			first = false
		}
		fmt.Fprint(w, `}}`)
	})

	// Serve embedded frontend (SPA with index.html fallback).
	if cfg.Frontend != nil {
		fileServer := http.FileServerFS(cfg.Frontend)
		s.mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			p := r.URL.Path[1:] // strip leading /
			// Serve exact file if it exists and is not a directory.
			if p != "" {
				if f, err := cfg.Frontend.Open(p); err == nil {
					stat, _ := f.Stat()
					f.Close()
					if stat != nil && !stat.IsDir() {
						fileServer.ServeHTTP(w, r)
						return
					}
				}
			}
			// SPA fallback: serve index.html for root and unknown paths.
			f, err := cfg.Frontend.Open("index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer f.Close()
			stat, _ := f.Stat()
			http.ServeContent(w, r, "index.html", stat.ModTime(), f.(io.ReadSeeker))
		})
	}

	return s
}

// Handle registers an HTTP handler pattern (e.g. "GET /ws/{sessionID}").
func (s *Server) Handle(pattern string, h http.Handler) {
	s.mux.Handle(pattern, h)
}

// HandleFunc registers an HTTP handler function.
func (s *Server) HandleFunc(pattern string, h func(http.ResponseWriter, *http.Request)) {
	s.mux.HandleFunc(pattern, h)
}

// Start begins listening. Blocks until ctx is cancelled or an error occurs.
func (s *Server) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("gateway: listen %s: %w", s.addr, err)
	}
	log.Printf("gateway: listening on %s", ln.Addr())

	errCh := make(chan error, 1)
	go func() {
		if err := s.httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		_ = s.httpSrv.Shutdown(context.Background())
		return nil
	case err := <-errCh:
		return err
	}
}

// Listen binds the server to its address and returns the actual address.
// Call Serve after storing the address to avoid TOCTOU races.
func (s *Server) Listen() (net.Listener, error) {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return nil, fmt.Errorf("gateway: listen %s: %w", s.addr, err)
	}
	s.addr = ln.Addr().String()
	log.Printf("gateway: listening on %s", s.addr)
	return ln, nil
}

// Serve accepts connections on ln. Blocks until ctx is cancelled or an error occurs.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	errCh := make(chan error, 1)
	go func() {
		if err := s.httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		_ = s.httpSrv.Shutdown(context.Background())
		return nil
	case err := <-errCh:
		return err
	}
}

// Addr returns the configured listen address.
func (s *Server) Addr() string { return s.addr }

// IsDuplicate returns true if msgID was seen recently.
func (s *Server) IsDuplicate(msgID string) bool {
	return s.dedup.SeenOrAdd(msgID)
}

// dedupCache is a fixed-size LRU-ish set for message dedup.
type dedupCache struct {
	mu   sync.Mutex
	seen map[string]struct{}
	keys []string
	cap  int
}

func newDedupCache(cap int) *dedupCache {
	return &dedupCache{seen: make(map[string]struct{}, cap), cap: cap}
}

// SeenOrAdd returns true if id was already present; otherwise adds it and returns false.
func (c *dedupCache) SeenOrAdd(id string) bool {
	if id == "" {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.seen[id]; ok {
		return true
	}
	if len(c.keys) >= c.cap {
		oldest := c.keys[0]
		c.keys = c.keys[1:]
		delete(c.seen, oldest)
	}
	c.seen[id] = struct{}{}
	c.keys = append(c.keys, id)
	return false
}
