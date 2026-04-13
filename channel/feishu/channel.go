// Package feishu provides a Feishu (Lark) channel via WebSocket long-connection.
// No public webhook URL is required — the SDK opens an outbound WebSocket to
// Feishu's servers and receives events over it.
package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"unicode"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"github.com/ai-gateway/clawfirm/gateway"
	piTypes "github.com/ai-gateway/clawfirm/types"
)

const channelID = "feishu"

// maxFileBytes is the per-file download limit (30 MB).
const maxFileBytes = 30 * 1024 * 1024

// supportedMsgTypes is the set of message types we attempt to process.
var supportedMsgTypes = map[string]bool{
	"text":    true,
	"image":   true,
	"file":    true,
	"audio":   true,
	"video":   true,
	"media":   true,
	"sticker": true,
	"post":    true,
}

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

	// Skip bot messages (sender_type != "user").
	if sender.SenderType != nil && *sender.SenderType != "user" {
		return
	}

	msgType := ""
	if msg.MessageType != nil {
		msgType = *msg.MessageType
	}
	if !supportedMsgTypes[msgType] {
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
		chatID = userID
	}

	// Extract message ID for media downloads.
	messageID := ""
	if msg.MessageId != nil {
		messageID = *msg.MessageId
	}

	// Extract text content (text and post types).
	text := ""
	if msgType == "text" || msgType == "post" {
		text = extractTextContent(msg.Content)
	}

	// Download media files.
	c.mu.RLock()
	apiClient := c.apiClient
	c.mu.RUnlock()

	var files []gateway.FileData
	if apiClient != nil && messageID != "" {
		files = resolveFeishuMediaList(ctx, apiClient, messageID, msgType, msg.Content)
	}

	if text == "" && len(files) == 0 {
		return
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
		Files:     files,
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
		if ev.Type == piTypes.EventAgentError {
			if ev.ErrorText != "" {
				c.mu.RLock()
				apiClient := c.apiClient
				c.mu.RUnlock()
				if apiClient != nil {
					_ = sendText(ctx, apiClient, chatID, "[Error] "+ev.ErrorText)
				}
			}
			return
		}
		if ev.Type != piTypes.EventAgentEnd {
			return
		}

		// Find the last AssistantMessage that contains text — that is the new reply.
		// (Tool-use turns also produce AssistantMessages with no text blocks.)
		var sb strings.Builder
		for i := len(ev.Messages) - 1; i >= 0; i-- {
			am, ok := ev.Messages[i].(*piTypes.AssistantMessage)
			if !ok {
				continue
			}
			for _, block := range am.Content {
				if tc, ok := block.(*piTypes.TextContent); ok && tc.Text != "" {
					if sb.Len() > 0 {
						sb.WriteByte('\n')
					}
					sb.WriteString(tc.Text)
				}
			}
			if sb.Len() > 0 {
				break // found the latest text reply
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
// For post messages, it extracts text tags from the rich-text content array.
func extractTextContent(raw *string) string {
	if raw == nil {
		return ""
	}
	var v struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(*raw), &v); err == nil && v.Text != "" {
		return stripMentions(v.Text)
	}
	// post type: {"title":"...","content":[[{"tag":"text","text":"..."}]]}
	var post struct {
		Title   string              `json:"title"`
		Content [][]map[string]any  `json:"content"`
	}
	if err := json.Unmarshal([]byte(*raw), &post); err != nil {
		return ""
	}
	var sb strings.Builder
	if post.Title != "" {
		sb.WriteString(post.Title)
		sb.WriteByte('\n')
	}
	for _, line := range post.Content {
		for _, elem := range line {
			if tag, _ := elem["tag"].(string); tag == "text" {
				if t, _ := elem["text"].(string); t != "" {
					sb.WriteString(t)
				}
			}
		}
		sb.WriteByte('\n')
	}
	return strings.TrimSpace(sb.String())
}

// stripMentions removes @mention tokens from text.
func stripMentions(text string) string {
	parts := strings.Fields(text)
	kept := parts[:0]
	for _, p := range parts {
		if strings.HasPrefix(p, "@") {
			continue
		}
		kept = append(kept, p)
	}
	return strings.TrimSpace(strings.Join(kept, " "))
}

// normalizeKey removes control characters and path-traversal sequences from a key.
func normalizeKey(key string) string {
	key = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '/' || r == '\\' {
			return -1
		}
		return r
	}, key)
	// Reject anything that still looks like a path traversal.
	if strings.Contains(key, "..") {
		return ""
	}
	return key
}

// inferPlaceholder returns the placeholder string for a given Feishu message type.
func inferPlaceholder(msgType string) string {
	switch msgType {
	case "image":
		return "<media:image>"
	case "audio":
		return "<media:audio>"
	case "video", "media":
		return "<media:video>"
	case "file":
		return "<media:document>"
	case "sticker":
		return "<media:sticker>"
	default:
		return "<media:document>"
	}
}

// mediaKeyInfo holds the extracted key and download API type for a single media item.
type mediaKeyInfo struct {
	key         string
	apiType     string // "image" or "file"
	placeholder string
	fileName    string
}

// parseMediaKeys extracts the primary media key from a Feishu message content JSON.
// For post messages use parsePostMediaKeys instead.
func parseMediaKeys(content *string, msgType string) (info mediaKeyInfo) {
	if content == nil {
		return
	}
	var v map[string]any
	if err := json.Unmarshal([]byte(*content), &v); err != nil {
		return
	}
	switch msgType {
	case "image":
		info.key, _ = v["image_key"].(string)
		info.apiType = "image"
		info.placeholder = "<media:image>"
	case "audio", "file", "sticker":
		info.key, _ = v["file_key"].(string)
		info.fileName, _ = v["file_name"].(string)
		info.apiType = "file"
		info.placeholder = inferPlaceholder(msgType)
	case "video", "media":
		// video/media may carry file_key or video_file_key
		if k, ok := v["file_key"].(string); ok && k != "" {
			info.key = k
		} else {
			info.key, _ = v["video_file_key"].(string)
		}
		info.fileName, _ = v["file_name"].(string)
		info.apiType = "file"
		info.placeholder = "<media:video>"
	}
	info.key = normalizeKey(info.key)
	return
}

// parsePostMediaKeys extracts all media keys embedded inside a post (rich-text) message.
func parsePostMediaKeys(content *string) []mediaKeyInfo {
	if content == nil {
		return nil
	}
	var post struct {
		Content [][]map[string]any `json:"content"`
	}
	if err := json.Unmarshal([]byte(*content), &post); err != nil {
		return nil
	}
	var result []mediaKeyInfo
	for _, line := range post.Content {
		for _, elem := range line {
			tag, _ := elem["tag"].(string)
			switch tag {
			case "img":
				key, _ := elem["image_key"].(string)
				key = normalizeKey(key)
				if key != "" {
					result = append(result, mediaKeyInfo{
						key:         key,
						apiType:     "image",
						placeholder: "<media:image>",
					})
				}
			case "media":
				key, _ := elem["file_key"].(string)
				key = normalizeKey(key)
				if key != "" {
					result = append(result, mediaKeyInfo{
						key:         key,
						apiType:     "file",
						placeholder: "<media:video>",
					})
				}
			}
		}
	}
	return result
}

// downloadMessageResource downloads a message resource from Feishu.
// apiType must be "image" or "file".
func downloadMessageResource(ctx context.Context, client *lark.Client, messageID, fileKey, apiType string) ([]byte, string, error) {
	req := larkim.NewGetMessageResourceReqBuilder().
		MessageId(messageID).
		FileKey(fileKey).
		Type(apiType).
		Build()
	resp, err := client.Im.V1.MessageResource.Get(ctx, req)
	if err != nil {
		return nil, "", err
	}
	if !resp.Success() {
		return nil, "", fmt.Errorf("code=%d msg=%s", resp.Code, resp.Msg)
	}
	lr := io.LimitReader(resp.File, maxFileBytes+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, "", err
	}
	if len(data) > maxFileBytes {
		return nil, "", fmt.Errorf("file exceeds %d bytes limit", maxFileBytes)
	}
	return data, resp.FileName, nil
}

// resolveFeishuMediaList is the unified entry-point: parses keys then downloads them.
func resolveFeishuMediaList(ctx context.Context, client *lark.Client, messageID, msgType string, content *string) []gateway.FileData {
	var keys []mediaKeyInfo
	if msgType == "post" {
		keys = parsePostMediaKeys(content)
	} else {
		info := parseMediaKeys(content, msgType)
		if info.key != "" {
			keys = append(keys, info)
		}
	}
	if len(keys) == 0 {
		return nil
	}

	var result []gateway.FileData
	for _, k := range keys {
		data, fname, err := downloadMessageResource(ctx, client, messageID, k.key, k.apiType)
		if err != nil {
			log.Printf("feishu: download %s (type=%s): %v", k.key, k.apiType, err)
			continue
		}
		if k.fileName != "" {
			fname = k.fileName
		}
		result = append(result, gateway.FileData{
			Data:        data,
			MimeType:    detectMimeType(k.apiType, fname),
			FileName:    fname,
			Placeholder: k.placeholder,
		})
	}
	return result
}

// detectMimeType returns a best-effort MIME type based on the download API type and file name.
func detectMimeType(apiType, fileName string) string {
	if apiType == "image" {
		ext := strings.ToLower(fileExt(fileName))
		switch ext {
		case ".png":
			return "image/png"
		case ".gif":
			return "image/gif"
		case ".webp":
			return "image/webp"
		default:
			return "image/jpeg"
		}
	}
	// file type
	ext := strings.ToLower(fileExt(fileName))
	switch ext {
	case ".mp3":
		return "audio/mpeg"
	case ".ogg", ".opus":
		return "audio/ogg"
	case ".mp4":
		return "video/mp4"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

// fileExt returns the file extension from a filename (e.g. ".jpg").
func fileExt(name string) string {
	i := strings.LastIndexByte(name, '.')
	if i < 0 {
		return ""
	}
	return name[i:]
}
