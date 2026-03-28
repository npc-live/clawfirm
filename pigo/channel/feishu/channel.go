// Package feishu provides a Feishu (Lark) channel via WebSocket long-connection.
// No public webhook URL is required — the SDK opens an outbound WebSocket to
// Feishu's servers and receives events over it.
package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"github.com/ai-gateway/pi-go/gateway"
	piTypes "github.com/ai-gateway/pi-go/types"
)

const channelID = "feishu"

// Channel receives Feishu messages via WebSocket and routes them to AI agents.
type Channel struct {
	appID        string
	appSecret    string
	registry     *gateway.AgentRegistry
	defaultAgent string

	mu          sync.RWMutex
	apiClient   *lark.Client
	sessionSubs map[string]bool // chatID → subscription registered
}

// New creates a Channel. Call Start to connect.
func New(appID, appSecret string, registry *gateway.AgentRegistry, defaultAgent string) *Channel {
	return &Channel{
		appID:        appID,
		appSecret:    appSecret,
		registry:     registry,
		defaultAgent: defaultAgent,
		sessionSubs:  make(map[string]bool),
	}
}

// Start connects to Feishu via WebSocket and blocks until ctx is cancelled.
func (c *Channel) Start(ctx context.Context) error {
	if c.appID == "" || c.appSecret == "" {
		return fmt.Errorf("feishu: appID and appSecret are required")
	}

	// REST client for sending replies.
	apiClient := lark.NewClient(c.appID, c.appSecret,
		lark.WithLogLevel(larkcore.LogLevelError),
	)
	c.mu.Lock()
	c.apiClient = apiClient
	c.mu.Unlock()

	// Event dispatcher — VerificationToken and EncryptKey are empty for WebSocket mode.
	eventHandler := dispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			c.onMessage(ctx, event)
			return nil
		})

	// WebSocket client.
	wsClient := larkws.NewClient(c.appID, c.appSecret,
		larkws.WithEventHandler(eventHandler),
		larkws.WithLogLevel(larkcore.LogLevelError),
	)

	log.Printf("feishu: starting WebSocket long-connection (appID=%s)", c.appID)

	// wsClient.Start blocks, auto-reconnects, and only returns on fatal error or ctx cancel.
	if err := wsClient.Start(ctx); err != nil && ctx.Err() == nil {
		return fmt.Errorf("feishu: WebSocket: %w", err)
	}
	return nil
}

// onMessage handles an incoming Feishu message event.
func (c *Channel) onMessage(ctx context.Context, event *larkim.P2MessageReceiveV1) {
	if event.Event == nil || event.Event.Message == nil || event.Event.Sender == nil {
		return
	}
	msg := event.Event.Message
	sender := event.Event.Sender

	// Only handle text messages for now.
	if msg.MessageType == nil || *msg.MessageType != "text" {
		return
	}
	// Skip bot messages (sender_type != "user").
	if sender.SenderType != nil && *sender.SenderType != "user" {
		return
	}

	text := extractTextContent(msg.Content)
	if text == "" {
		return
	}

	// Use open_id as the stable user identifier.
	userID := ""
	if sender.SenderId != nil && sender.SenderId.OpenId != nil {
		userID = *sender.SenderId.OpenId
	}
	if userID == "" {
		return
	}

	// chat_id is used both as the session key and the reply target.
	chatID := ""
	if msg.ChatId != nil {
		chatID = *msg.ChatId
	}
	if chatID == "" {
		// Fallback for p2p: reply to the sender's open_id.
		chatID = userID
	}

	mgr, ok := c.registry.Get(c.defaultAgent)
	if !ok {
		log.Printf("feishu: default agent %q not found", c.defaultAgent)
		return
	}

	sess, err := mgr.GetOrCreate(channelID, userID)
	if err != nil {
		log.Printf("feishu: GetOrCreate session: %v", err)
		return
	}

	c.ensureReplySubscription(ctx, sess, chatID)

	if !sess.Send(gateway.IncomingMessage{
		ChannelID: channelID,
		UserID:    userID,
		Content:   text,
	}) {
		log.Printf("feishu: session queue full for %s", userID)
	}
}

// ensureReplySubscription registers a reply subscription for the chat if not already done.
func (c *Channel) ensureReplySubscription(ctx context.Context, sess *gateway.Session, chatID string) {
	c.mu.Lock()
	if c.sessionSubs[chatID] {
		c.mu.Unlock()
		return
	}
	c.sessionSubs[chatID] = true
	c.mu.Unlock()

	sess.Subscribe(func(ev piTypes.AgentEvent) {
		if ev.Type != piTypes.EventAgentEnd {
			return
		}

		// Collect all assistant text from the final message list.
		var sb strings.Builder
		for _, msg := range ev.Messages {
			am, ok := msg.(*piTypes.AssistantMessage)
			if !ok {
				continue
			}
			for _, block := range am.Content {
				if tc, ok := block.(*piTypes.TextContent); ok {
					if sb.Len() > 0 {
						sb.WriteByte('\n')
					}
					sb.WriteString(tc.Text)
				}
			}
		}
		reply := sb.String()
		if reply == "" {
			return
		}

		c.mu.RLock()
		apiClient := c.apiClient
		c.mu.RUnlock()
		if apiClient == nil {
			return
		}

		if err := sendText(ctx, apiClient, chatID, reply); err != nil {
			log.Printf("feishu: send to %s: %v", chatID, err)
		}
	})
}

// sendText sends a plain-text message to a Feishu chat or user (by chat_id).
func sendText(ctx context.Context, client *lark.Client, chatID, text string) error {
	content, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return err
	}
	msgType := larkim.MsgTypeText
	receiveID := chatID
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(larkim.ReceiveIdTypeChatId).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			MsgType(msgType).
			ReceiveId(receiveID).
			Content(string(content)).
			Build()).
		Build()

	resp, err := client.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return err
	}
	if !resp.Success() {
		return fmt.Errorf("code=%d msg=%s", resp.Code, resp.Msg)
	}
	return nil
}

// extractTextContent parses the Feishu text message JSON and returns the plain text.
// Feishu text content looks like: {"text":"hello @user"} — we strip @mention tags.
func extractTextContent(raw *string) string {
	if raw == nil {
		return ""
	}
	var v struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(*raw), &v); err != nil {
		return ""
	}
	// Remove @mention tokens (format: @_user_xxx or @xxx).
	text := v.Text
	parts := strings.Fields(text)
	var kept []string
	for _, p := range parts {
		if strings.HasPrefix(p, "@") {
			continue
		}
		kept = append(kept, p)
	}
	return strings.TrimSpace(strings.Join(kept, " "))
}
