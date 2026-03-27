// Command wschat is a minimal WebSocket chat client for testing the gateway.
//
// Usage:
//
//	go run ./cmd/wschat ws://localhost:9988/ws/zenmux/alice "你好"
//	go run ./cmd/wschat ws://localhost:9988/ws/minimax/alice "你好"
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	neturl "net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: wschat <ws-url> <message>")
		os.Exit(1)
	}
	url, msg := os.Args[1], os.Args[2]

	// Use nil proxy to bypass http_proxy env vars (which strip Connection: Upgrade).
	dialer := websocket.Dialer{
		Proxy:            func(*http.Request) (*neturl.URL, error) { return nil, nil },
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		log.Fatalf("dial %s: %v", url, err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(map[string]string{"type": "message", "content": msg}); err != nil {
		log.Fatalf("write: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Fatalf("read: %v", err)
		}
		var m map[string]any
		json.Unmarshal(data, &m)
		switch m["type"] {
		case "delta":
			fmt.Print(m["content"])
		case "error":
			fmt.Fprintf(os.Stderr, "\nERROR: %v\n", m["content"])
		case "done":
			fmt.Println()
			return
		}
	}
}
