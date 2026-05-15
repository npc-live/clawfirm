package provider

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

// NewStreamingHTTPClient returns an http.Client tuned for long-lived SSE streaming.
//
// Set CLAWFIRM_NO_PROXY=1 to bypass the system proxy entirely (for debugging).
func NewStreamingHTTPClient() *http.Client {
	proxyFn := http.ProxyFromEnvironment
	noProxy := os.Getenv("CLAWFIRM_NO_PROXY") == "1"
	if noProxy {
		proxyFn = nil
		log.Printf("[provider] CLAWFIRM_NO_PROXY=1: proxy disabled")
	}

	transport := &http.Transport{
		Proxy: proxyFn,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			NextProtos: []string{"http/1.1"},
		},
		TLSNextProto:       make(map[string]func(string, *tls.Conn) http.RoundTripper),
		DisableCompression: true,
		MaxIdleConns:       10,
		IdleConnTimeout:    90 * time.Second,
	}
	proxyLabel := "env"
	if noProxy {
		proxyLabel = "none"
	}
	log.Printf("[provider] created streaming HTTP client: HTTP/1.1 forced, proxy=%s, DisableCompression=true", proxyLabel)
	return &http.Client{
		Timeout:   10 * time.Minute,
		Transport: transport,
	}
}
