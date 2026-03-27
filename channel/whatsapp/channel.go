// Package whatsapp provides a WhatsApp channel using the whatsmeow library.
// Device credentials are persisted in a plain JSON file — no SQL database or
// foreign-key constraints required.
package whatsapp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waAdv"
	"go.mau.fi/whatsmeow/proto/waE2E"
	waStore "go.mau.fi/whatsmeow/store"
	waTypes "go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"go.mau.fi/whatsmeow/util/keys"

	"github.com/ai-gateway/pi-go/gateway"
	piTypes "github.com/ai-gateway/pi-go/types"
)

const channelID = "whatsapp"

// Status values returned by GetStatus.
const (
	StatusDisconnected = "disconnected"
	StatusQRPending    = "qr_pending"
	StatusConnected    = "connected"
	StatusLoggedOut    = "logged_out"
)

// Channel is a long-running WhatsApp channel.
type Channel struct {
	registry     *gateway.AgentRegistry
	defaultAgent string

	mu          sync.RWMutex
	waClient    *whatsmeow.Client
	status      string
	currentQR   string          // base64 PNG data URL, non-empty only while StatusQRPending
	sessionSubs map[string]bool // chatJID.String() → subscription registered
	storePath   string          // path to the JSON credentials file
}

// New creates a Channel. Call Start to connect.
func New(registry *gateway.AgentRegistry, defaultAgent string) *Channel {
	return &Channel{
		registry:     registry,
		defaultAgent: defaultAgent,
		status:       StatusDisconnected,
		sessionSubs:  make(map[string]bool),
	}
}

// GetStatus returns the current connection status string.
func (c *Channel) GetStatus() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

// GetQR returns the current QR code as a data URL, or "" if not in QR-pending state.
func (c *Channel) GetQR() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentQR
}

// Logout logs the WhatsApp session out and resets state.
func (c *Channel) Logout(ctx context.Context) error {
	c.mu.RLock()
	client := c.waClient
	storePath := c.storePath
	c.mu.RUnlock()
	if client == nil {
		return nil
	}
	err := client.Logout(ctx)
	// Remove the credentials file so next Start() triggers fresh QR pairing.
	_ = os.Remove(storePath)
	c.mu.Lock()
	c.currentQR = ""
	c.status = StatusDisconnected
	c.sessionSubs = make(map[string]bool)
	c.mu.Unlock()
	return err
}

// Start connects to WhatsApp. It blocks until ctx is cancelled.
func (c *Channel) Start(ctx context.Context) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".pi-go")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	storePath := filepath.Join(dir, "whatsapp-device.json")

	c.mu.Lock()
	c.storePath = storePath
	c.mu.Unlock()

	// Build the device store — either load existing credentials or create new.
	fc := &fileContainer{path: storePath}
	device, err := fc.loadOrNew()
	if err != nil {
		return fmt.Errorf("whatsapp: device store: %w", err)
	}

	client := whatsmeow.NewClient(device, waLog.Noop)
	client.AddEventHandler(c.handleEvent)

	c.mu.Lock()
	c.waClient = client
	c.mu.Unlock()

	if device.ID == nil {
		// No existing session — show QR code for pairing.
		qrChan, err := client.GetQRChannel(ctx)
		if err != nil {
			return fmt.Errorf("whatsapp: GetQRChannel: %w", err)
		}
		if err := client.Connect(); err != nil {
			return fmt.Errorf("whatsapp: Connect: %w", err)
		}
		c.setStatus(StatusQRPending)
		go c.drainQRChannel(qrChan)
	} else {
		// Existing session — connect directly; Connected event will set status.
		if err := client.Connect(); err != nil {
			return fmt.Errorf("whatsapp: Connect: %w", err)
		}
	}

	<-ctx.Done()
	client.Disconnect()
	return nil
}

// drainQRChannel consumes QR code items and updates currentQR / status.
func (c *Channel) drainQRChannel(ch <-chan whatsmeow.QRChannelItem) {
	for item := range ch {
		switch item.Event {
		case whatsmeow.QRChannelEventCode:
			dataURL, err := qrCodeToDataURL(item.Code, 300)
			if err != nil {
				log.Printf("whatsapp: qr encode: %v", err)
				continue
			}
			c.mu.Lock()
			c.currentQR = dataURL
			c.mu.Unlock()
		case "success":
			c.mu.Lock()
			c.currentQR = ""
			c.mu.Unlock()
			c.setStatus(StatusConnected)
		case "timeout":
			c.setStatus(StatusDisconnected)
		}
	}
}

// handleEvent is the whatsmeow event handler.
func (c *Channel) handleEvent(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		c.onMessage(v)
	case *events.Connected:
		c.setStatus(StatusConnected)
	case *events.Disconnected:
		c.setStatus(StatusDisconnected)
	case *events.LoggedOut:
		c.mu.Lock()
		c.currentQR = ""
		c.sessionSubs = make(map[string]bool)
		_ = os.Remove(c.storePath)
		c.mu.Unlock()
		c.setStatus(StatusLoggedOut)
	}
}

// onMessage handles an incoming WhatsApp message.
func (c *Channel) onMessage(v *events.Message) {
	// Skip groups, self-sent messages, and broadcasts.
	if v.Info.IsGroup || v.Info.IsFromMe || v.Info.Chat.IsBroadcastList() {
		return
	}

	// Extract text content.
	text := v.Message.GetConversation()
	if text == "" {
		if ext := v.Message.GetExtendedTextMessage(); ext != nil {
			text = ext.GetText()
		}
	}
	if text == "" {
		return
	}

	senderPhone := v.Info.Sender.User // e.g. "8613900001234"
	chatJID := v.Info.Chat

	mgr, ok := c.registry.Get(c.defaultAgent)
	if !ok {
		log.Printf("whatsapp: default agent %q not found", c.defaultAgent)
		return
	}

	sess, err := mgr.GetOrCreate(channelID, senderPhone)
	if err != nil {
		log.Printf("whatsapp: GetOrCreate session: %v", err)
		return
	}

	c.ensureReplySubscription(sess, chatJID)

	if !sess.Send(gateway.IncomingMessage{
		ChannelID: channelID,
		UserID:    senderPhone,
		Content:   text,
	}) {
		log.Printf("whatsapp: session queue full for %s", senderPhone)
	}
}

// ensureReplySubscription registers a reply subscription for the chat if not already done.
func (c *Channel) ensureReplySubscription(sess *gateway.Session, chatJID waTypes.JID) {
	key := chatJID.String()
	c.mu.Lock()
	if c.sessionSubs[key] {
		c.mu.Unlock()
		return
	}
	c.sessionSubs[key] = true
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
		client := c.waClient
		c.mu.RUnlock()
		if client == nil {
			return
		}

		ctx := context.Background()
		if _, err := client.SendMessage(ctx, chatJID, &waE2E.Message{
			Conversation: proto.String(reply),
		}); err != nil {
			log.Printf("whatsapp: SendMessage to %s: %v", chatJID, err)
		}
	})
}

func (c *Channel) setStatus(s string) {
	c.mu.Lock()
	c.status = s
	c.mu.Unlock()
}

// ─── JSON device store (no SQL, no foreign keys) ──────────────────────────────

// deviceJSON is the on-disk format for persisting WhatsApp device credentials.
type deviceJSON struct {
	NoiseKeyPriv    []byte `json:"noise_key_priv"`
	IdentityKeyPriv []byte `json:"identity_key_priv"`
	PreKeyPriv      []byte `json:"pre_key_priv"`
	PreKeyID        uint32 `json:"pre_key_id"`
	PreKeySig       []byte `json:"pre_key_sig"`
	AdvSecretKey    []byte `json:"adv_secret_key"`
	RegistrationID  uint32 `json:"registration_id"`

	// Set after successful pairing.
	JID              string `json:"jid,omitempty"`
	AdvDetails       []byte `json:"adv_details,omitempty"`
	AdvAccountSig    []byte `json:"adv_account_sig,omitempty"`
	AdvAccountSigKey []byte `json:"adv_account_sig_key,omitempty"`
	AdvDeviceSig     []byte `json:"adv_device_sig,omitempty"`
	Platform         string `json:"platform,omitempty"`
	PushName         string `json:"push_name,omitempty"`
}

// fileContainer implements store.DeviceContainer backed by a JSON file.
type fileContainer struct {
	path string
	mu   sync.Mutex
}

func (fc *fileContainer) loadOrNew() (*waStore.Device, error) {
	data, err := os.ReadFile(fc.path)
	if os.IsNotExist(err) {
		return fc.newDevice(), nil
	}
	if err != nil {
		return nil, err
	}
	var dj deviceJSON
	if err := json.Unmarshal(data, &dj); err != nil {
		return nil, err
	}
	return fc.fromJSON(&dj)
}

func (fc *fileContainer) newDevice() *waStore.Device {
	d := &waStore.Device{
		Log:            waLog.Noop,
		Container:      fc,
		NoiseKey:       keys.NewKeyPair(),
		IdentityKey:    keys.NewKeyPair(),
		RegistrationID: rand.Uint32(),
		AdvSecretKey:   randBytes(32),
	}
	d.SignedPreKey = d.IdentityKey.CreateSignedPreKey(1)
	noop := &waStore.NoopStore{}
	d.SetAllStores(noop)
	d.LIDs = noop
	return d
}

func (fc *fileContainer) fromJSON(dj *deviceJSON) (*waStore.Device, error) {
	if len(dj.NoiseKeyPriv) != 32 || len(dj.IdentityKeyPriv) != 32 ||
		len(dj.PreKeyPriv) != 32 || len(dj.PreKeySig) != 64 {
		return nil, fmt.Errorf("whatsapp: corrupt credentials file")
	}
	d := &waStore.Device{
		Log:            waLog.Noop,
		Container:      fc,
		NoiseKey:       keys.NewKeyPairFromPrivateKey(*(*[32]byte)(dj.NoiseKeyPriv)),
		IdentityKey:    keys.NewKeyPairFromPrivateKey(*(*[32]byte)(dj.IdentityKeyPriv)),
		RegistrationID: dj.RegistrationID,
		AdvSecretKey:   dj.AdvSecretKey,
		Platform:       dj.Platform,
		PushName:       dj.PushName,
		Initialized:    true,
	}
	d.SignedPreKey = &keys.PreKey{
		KeyPair:   *keys.NewKeyPairFromPrivateKey(*(*[32]byte)(dj.PreKeyPriv)),
		KeyID:     dj.PreKeyID,
		Signature: (*[64]byte)(dj.PreKeySig),
	}
	if dj.JID != "" {
		jid, err := waTypes.ParseJID(dj.JID)
		if err != nil {
			return nil, fmt.Errorf("whatsapp: parse jid: %w", err)
		}
		d.ID = &jid
		d.Account = &waAdv.ADVSignedDeviceIdentity{
			Details:             dj.AdvDetails,
			AccountSignature:    dj.AdvAccountSig,
			AccountSignatureKey: dj.AdvAccountSigKey,
			DeviceSignature:     dj.AdvDeviceSig,
		}
	}
	noop := &waStore.NoopStore{}
	d.SetAllStores(noop)
	d.LIDs = noop
	return d, nil
}

// PutDevice implements store.DeviceContainer — saves credentials to JSON.
func (fc *fileContainer) PutDevice(ctx context.Context, device *waStore.Device) error {
	if device.ID == nil {
		return fmt.Errorf("whatsapp: device JID not set")
	}
	dj := &deviceJSON{
		NoiseKeyPriv:    device.NoiseKey.Priv[:],
		IdentityKeyPriv: device.IdentityKey.Priv[:],
		PreKeyPriv:      device.SignedPreKey.Priv[:],
		PreKeyID:        device.SignedPreKey.KeyID,
		PreKeySig:       device.SignedPreKey.Signature[:],
		AdvSecretKey:    device.AdvSecretKey,
		RegistrationID:  device.RegistrationID,
		JID:             device.ID.String(),
		Platform:        device.Platform,
		PushName:        device.PushName,
	}
	if device.Account != nil {
		dj.AdvDetails = device.Account.Details
		dj.AdvAccountSig = device.Account.AccountSignature
		dj.AdvAccountSigKey = device.Account.AccountSignatureKey
		dj.AdvDeviceSig = device.Account.DeviceSignature
	}
	data, err := json.MarshalIndent(dj, "", "  ")
	if err != nil {
		return err
	}
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return os.WriteFile(fc.path, data, 0o600)
}

// DeleteDevice implements store.DeviceContainer.
func (fc *fileContainer) DeleteDevice(ctx context.Context, device *waStore.Device) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	err := os.Remove(fc.path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// qrCodeToDataURL renders a QR code as a base64-encoded PNG data URL.
func qrCodeToDataURL(code string, size int) (string, error) {
	png, err := qrcode.Encode(code, qrcode.Medium, size)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rand.Uint32())
	}
	return b
}

// Ensure uuid is used (imported for potential future use in FacebookUUID handling).
var _ = uuid.Nil
