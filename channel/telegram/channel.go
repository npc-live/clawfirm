// Package telegram provides a Telegram Bot channel via long polling.
// No public webhook URL is required — the bot polls Telegram's servers for updates.
package telegram

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/ai-gateway/clawfirm/gateway"
	piTypes "github.com/ai-gateway/clawfirm/types"
)

const channelID = "telegram"

// maxFileBytes is the per-file download limit (20 MB — Telegram Bot API limit).
const maxFileBytes = 20 * 1024 * 1024

// maxTextLen is the Telegram message length limit.
const maxTextLen = 4096

// Channel receives Telegram messages via long polling and routes them to AI agents.
type Channel struct {
	botToken     string
	registry     *gateway.AgentRegistry
	defaultAgent string

	mu          sync.RWMutex
	bot         *tgbotapi.BotAPI
	sessionSubs map[int64]bool // chatID → subscription registered
}

// New creates a Channel. Call Start to connect.
func New(botToken string, registry *gateway.AgentRegistry, defaultAgent string) *Channel {
	return &Channel{
		botToken:     botToken,
		registry:     registry,
		defaultAgent: defaultAgent,
		sessionSubs:  make(map[int64]bool),
	}
}

// Start connects to Telegram via long polling and blocks until ctx is cancelled.
func (c *Channel) Start(ctx context.Context) error {
	if c.botToken == "" {
		return fmt.Errorf("telegram: bot_token is required")
	}

	bot, err := tgbotapi.NewBotAPI(c.botToken)
	if err != nil {
		return fmt.Errorf("telegram: init bot: %w", err)
	}

	c.mu.Lock()
	c.bot = bot
	c.mu.Unlock()

	log.Printf("telegram: bot @%s started (long polling)", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			bot.StopReceivingUpdates()
			return nil
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			if update.Message != nil {
				c.onMessage(ctx, bot, update.Message)
			}
		}
	}
}

// onMessage handles an incoming Telegram message.
func (c *Channel) onMessage(ctx context.Context, bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	userID := fmt.Sprintf("%d", msg.From.ID)

	// Extract text content.
	text := msg.Text
	if text == "" {
		text = msg.Caption
	}

	// Download media files.
	var files []gateway.FileData
	if f := c.downloadMedia(ctx, bot, msg); f != nil {
		files = append(files, *f)
	}

	if text == "" && len(files) == 0 {
		return
	}

	mgr, ok := c.registry.Get(c.defaultAgent)
	if !ok {
		log.Printf("telegram: default agent %q not found", c.defaultAgent)
		return
	}

	sess, err := mgr.GetOrCreate(channelID, userID)
	if err != nil {
		log.Printf("telegram: GetOrCreate session: %v", err)
		return
	}

	c.ensureReplySubscription(ctx, sess, chatID)

	if !sess.Send(gateway.IncomingMessage{
		ChannelID: channelID,
		UserID:    userID,
		Content:   text,
		Files:     files,
	}) {
		log.Printf("telegram: session queue full for %s", userID)
	}
}

// ensureReplySubscription registers a reply subscription for the chat if not already done.
func (c *Channel) ensureReplySubscription(ctx context.Context, sess *gateway.Session, chatID int64) {
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
				bot := c.bot
				c.mu.RUnlock()
				if bot != nil {
					_ = sendText(bot, chatID, "[Error] "+ev.ErrorText)
				}
			}
			return
		}
		if ev.Type != piTypes.EventAgentEnd {
			return
		}

		// Find the last AssistantMessage that contains text.
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
				break
			}
		}
		reply := sb.String()
		if reply == "" {
			return
		}

		c.mu.RLock()
		bot := c.bot
		c.mu.RUnlock()
		if bot == nil {
			return
		}

		if err := sendText(bot, chatID, reply); err != nil {
			log.Printf("telegram: send to %d: %v", chatID, err)
		}
	})
}

// sendText sends a text message, splitting into chunks if it exceeds the Telegram limit.
func sendText(bot *tgbotapi.BotAPI, chatID int64, text string) error {
	for len(text) > 0 {
		chunk := text
		if len(chunk) > maxTextLen {
			// Try to split at the last newline before the limit.
			cut := strings.LastIndex(chunk[:maxTextLen], "\n")
			if cut <= 0 {
				cut = maxTextLen
			}
			chunk = text[:cut]
		}
		msg := tgbotapi.NewMessage(chatID, chunk)
		msg.ParseMode = "Markdown"
		if _, err := bot.Send(msg); err != nil {
			// Retry without parse mode in case Markdown is invalid.
			msg.ParseMode = ""
			if _, err2 := bot.Send(msg); err2 != nil {
				return err2
			}
		}
		text = text[len(chunk):]
	}
	return nil
}

// downloadMedia attempts to download a media file from the message.
// Returns nil if the message has no downloadable media.
func (c *Channel) downloadMedia(ctx context.Context, bot *tgbotapi.BotAPI, msg *tgbotapi.Message) *gateway.FileData {
	var fileID, fileName, mimeType, placeholder string

	switch {
	case msg.Photo != nil && len(msg.Photo) > 0:
		// Pick the largest photo.
		photo := msg.Photo[len(msg.Photo)-1]
		fileID = photo.FileID
		mimeType = "image/jpeg"
		placeholder = "<media:image>"
		fileName = "photo.jpg"

	case msg.Document != nil:
		fileID = msg.Document.FileID
		fileName = msg.Document.FileName
		mimeType = msg.Document.MimeType
		placeholder = "<media:document>"

	case msg.Audio != nil:
		fileID = msg.Audio.FileID
		fileName = msg.Audio.FileName
		mimeType = msg.Audio.MimeType
		placeholder = "<media:audio>"

	case msg.Voice != nil:
		fileID = msg.Voice.FileID
		mimeType = msg.Voice.MimeType
		if mimeType == "" {
			mimeType = "audio/ogg"
		}
		placeholder = "<media:audio>"
		fileName = "voice.ogg"

	case msg.Video != nil:
		fileID = msg.Video.FileID
		fileName = msg.Video.FileName
		mimeType = msg.Video.MimeType
		placeholder = "<media:video>"

	case msg.Sticker != nil:
		fileID = msg.Sticker.FileID
		mimeType = "image/webp"
		placeholder = "<media:sticker>"
		fileName = "sticker.webp"

	default:
		return nil
	}

	if fileID == "" {
		return nil
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	data, err := downloadFile(ctx, bot, fileID)
	if err != nil {
		log.Printf("telegram: download %s: %v", fileID, err)
		return nil
	}

	return &gateway.FileData{
		Data:        data,
		MimeType:    mimeType,
		FileName:    fileName,
		Placeholder: placeholder,
	}
}

// downloadFile fetches file bytes from Telegram by file ID.
func downloadFile(ctx context.Context, bot *tgbotapi.BotAPI, fileID string) ([]byte, error) {
	fileConfig := tgbotapi.FileConfig{FileID: fileID}
	tf, err := bot.GetFile(fileConfig)
	if err != nil {
		return nil, fmt.Errorf("getFile: %w", err)
	}

	fileURL := tf.Link(bot.Token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	lr := io.LimitReader(resp.Body, int64(maxFileBytes)+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if len(data) > maxFileBytes {
		return nil, fmt.Errorf("file exceeds %d bytes limit", maxFileBytes)
	}
	return data, nil
}
