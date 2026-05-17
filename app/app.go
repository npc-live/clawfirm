// Package app contains the Wails App struct and all bindings exposed to the frontend.
package app

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ai-gateway/clawfirm/agent"
	"github.com/ai-gateway/clawfirm/auth"
	"github.com/ai-gateway/clawfirm/browser"
	"github.com/ai-gateway/clawfirm/channel/feishu"
	"github.com/ai-gateway/clawfirm/channel/remote"
	"github.com/ai-gateway/clawfirm/channel/telegram"
	"github.com/ai-gateway/clawfirm/channel/webchat"
	"github.com/ai-gateway/clawfirm/channel/whatsapp"
	"github.com/ai-gateway/clawfirm/config"
	picron "github.com/ai-gateway/clawfirm/cron"
	"github.com/ai-gateway/clawfirm/gateway"
	"github.com/ai-gateway/clawfirm/internal/agentbuilder"
	"github.com/ai-gateway/clawfirm/tool/builtin"
	"github.com/ai-gateway/clawfirm/memory"
	"github.com/ai-gateway/clawfirm/provider"
	"github.com/ai-gateway/clawfirm/skill"
	"github.com/ai-gateway/clawfirm/skillctl"
	"github.com/ai-gateway/clawfirm/store"
	"github.com/ai-gateway/clawfirm/tool"
	"github.com/ai-gateway/clawfirm/types"
	"github.com/ai-gateway/clawfirm/vault"
	"github.com/ai-gateway/clawfirm/vault/keychain"
	"github.com/google/uuid"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Version is set at build time via ldflags.
var Version = "dev"

// ProviderInfo is returned to the frontend.
type ProviderInfo struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	HasKey bool   `json:"hasKey"`
}

// ChannelInfo represents a configured agent for the dashboard.
type ChannelInfo struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Sessions int    `json:"sessions"`
}

// SessionInfo is a minimal session descriptor.
type SessionInfo struct {
	AgentName string `json:"agentName"`
	ChannelID string `json:"channelId"`
	UserID    string `json:"userId"`
}

// ToolExecRecord tracks a single tool execution's lifecycle in the KV store.
type ToolExecRecord struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"` // "running" | "done" | "error" | "interrupted"
	StartedAt int64  `json:"startedAt"`
	EndedAt   int64  `json:"endedAt,omitempty"`
}

// App is the Wails application struct. All exported methods become RPC calls available to the frontend.
type App struct {
	ctx      context.Context
	cancelFn context.CancelFunc

	mu             sync.RWMutex
	cfg            *config.Config
	db             *store.DB
	authStor       *auth.AuthStorage
	registry       *gateway.AgentRegistry
	srvAddr        string // "127.0.0.1:PORT" once gateway is running
	externalGW     string // external gateway URL for WebSocket connections (e.g. "http://localhost:9988")
	whatsappCh     *whatsapp.Channel
	whatsappCancel context.CancelFunc
	feishuCh       *feishu.Channel
	feishuCancel   context.CancelFunc
	telegramCh     *telegram.Channel
	telegramCancel context.CancelFunc
	cronScheduler  *picron.Scheduler
	vault          *vault.Vault
	remoteSrv      *remote.Server
	remoteCancel   context.CancelFunc
	memoryMgr      *memory.Manager
	startupErrors  []string

	// EvalJS state
	evalSeq      int
	evalPending  map[int]chan string
}

// New creates the App. Call wails.Run with a.OnStartup / a.OnShutdown.
func New() *App {
	return &App{}
}

// applySystemProxy reads macOS system proxy settings via scutil --proxy and
// sets HTTPS_PROXY / HTTP_PROXY env vars so net/http.ProxyFromEnvironment picks
// them up even when the app is launched from Finder/Dock (no shell env).
func applySystemProxy() {
	if runtime.GOOS != "darwin" {
		return
	}
	// Already set — nothing to do.
	if os.Getenv("HTTPS_PROXY") != "" || os.Getenv("HTTP_PROXY") != "" {
		return
	}
	out, err := exec.Command("scutil", "--proxy").Output()
	if err != nil {
		return
	}
	vals := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.SplitN(line, " : ", 2)
		if len(parts) == 2 {
			vals[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	if vals["HTTPSEnable"] == "1" && vals["HTTPSProxy"] != "" {
		proxy := "http://" + vals["HTTPSProxy"] + ":" + vals["HTTPSPort"]
		_ = os.Setenv("HTTPS_PROXY", proxy)
		_ = os.Setenv("https_proxy", proxy)
		log.Printf("app: system HTTPS proxy applied: %s", proxy)
	}
	if vals["HTTPEnable"] == "1" && vals["HTTPProxy"] != "" {
		proxy := "http://" + vals["HTTPProxy"] + ":" + vals["HTTPPort"]
		_ = os.Setenv("HTTP_PROXY", proxy)
		_ = os.Setenv("http_proxy", proxy)
		log.Printf("app: system HTTP proxy applied: %s", proxy)
	}
	if vals["SOCKSEnable"] == "1" && vals["SOCKSProxy"] != "" {
		proxy := "socks5://" + vals["SOCKSProxy"] + ":" + vals["SOCKSPort"]
		_ = os.Setenv("ALL_PROXY", proxy)
		_ = os.Setenv("all_proxy", proxy)
		log.Printf("app: system SOCKS proxy applied: %s", proxy)
	}
}

// migrateFromPiGo renames ~/.pi-go to ~/.clawfirm if the old directory exists
// and the new one does not. A symlink is left behind for backward compatibility.
// registerClaudePlugin ensures pluginID is enabled in ~/.clawfirm/settings.json.
func registerClaudePlugin(pluginID string) {
	settingsPath := filepath.Join(clawfirmDataDir(), "settings.json")
	data, err := os.ReadFile(settingsPath)
	var settings map[string]any
	if err != nil {
		settings = make(map[string]any)
	} else if err := json.Unmarshal(data, &settings); err != nil {
		settings = make(map[string]any)
	}
	enabledPlugins, _ := settings["enabledPlugins"].(map[string]any)
	if enabledPlugins == nil {
		enabledPlugins = make(map[string]any)
	}
	if _, already := enabledPlugins[pluginID]; already {
		return
	}
	enabledPlugins[pluginID] = true
	settings["enabledPlugins"] = enabledPlugins
	updated, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return
	}
	if err := os.WriteFile(settingsPath, updated, 0o644); err != nil {
		log.Printf("app: register plugin %s: %v", pluginID, err)
	} else {
		log.Printf("app: registered claw plugin %s", pluginID)
	}
}

func migrateFromPiGo() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	oldDir := filepath.Join(home, ".pi-go")
	newDir := filepath.Join(home, ".clawfirm")

	oldInfo, oldErr := os.Lstat(oldDir)
	_, newErr := os.Lstat(newDir)

	if oldErr != nil || newErr == nil {
		return // old doesn't exist or new already exists
	}
	// Only migrate if old is a real directory (not already a symlink).
	if oldInfo.Mode()&os.ModeSymlink != 0 {
		return
	}

	if err := os.Rename(oldDir, newDir); err != nil {
		log.Printf("app: migrate ~/.pi-go → ~/.clawfirm: %v", err)
		return
	}
	// Leave a symlink for backward compatibility.
	if err := os.Symlink(newDir, oldDir); err != nil {
		log.Printf("app: create symlink ~/.pi-go → ~/.clawfirm: %v", err)
	} else {
		log.Printf("app: migrated ~/.pi-go → ~/.clawfirm (symlink created)")
	}
}

// migrateVault copies entries from the old unencrypted SQLite vault table
// into the new AES-256-GCM encrypted BBolt vault. Runs once — the old table
// is dropped after a successful migration.
func migrateVault(db *store.DB) {
	sqlDB := db.SQL()

	// Check if old vault table exists.
	var name string
	err := sqlDB.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='vault'`).Scan(&name)
	if err != nil {
		return // table doesn't exist, nothing to migrate
	}

	// Read old entries.
	rows, err := sqlDB.Query(`SELECT key, value FROM vault ORDER BY key`)
	if err != nil {
		log.Printf("app: vault migrate: read old entries: %v", err)
		return
	}
	defer rows.Close()

	type kv struct{ k, v string }
	var entries []kv
	for rows.Next() {
		var e kv
		if err := rows.Scan(&e.k, &e.v); err != nil {
			log.Printf("app: vault migrate: scan: %v", err)
			return
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		log.Printf("app: vault migrate: rows: %v", err)
		return
	}
	if len(entries) == 0 {
		// Table exists but is empty — just drop it.
		sqlDB.Exec(`DROP TABLE vault`)
		return
	}

	// Ensure new encrypted vault is initialized.
	dbPath := vault.DefaultDBPath()
	kc := keychain.New()
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		if err := vault.Init(dbPath, kc); err != nil {
			log.Printf("app: vault migrate: init: %v", err)
			return
		}
		log.Printf("app: vault migrate: initialized new encrypted vault")
	}

	// Open, copy, close.
	v, err := vault.Open(dbPath, kc)
	if err != nil {
		log.Printf("app: vault migrate: open: %v", err)
		return
	}
	for _, e := range entries {
		if err := v.Set(e.k, []byte(e.v)); err != nil {
			log.Printf("app: vault migrate: set %q: %v", e.k, err)
			v.Close()
			return
		}
	}
	v.Close()

	// Drop old table.
	if _, err := sqlDB.Exec(`DROP TABLE vault`); err != nil {
		log.Printf("app: vault migrate: drop old table: %v", err)
		return
	}
	log.Printf("app: vault migrate: migrated %d entries to encrypted vault", len(entries))
}

// initAppLog redirects the default logger to <dataDir>/app.log (append mode).
// Output is also mirrored to stderr so `wails dev` / terminal runs still show logs.
// CLAWFIRM_LOG_PATH overrides the default path.
func initAppLog() {
	logPath := os.Getenv("CLAWFIRM_LOG_PATH")
	if logPath == "" {
		logPath = filepath.Join(clawfirmDataDir(), "app.log")
	} else if strings.HasPrefix(logPath, "~/") {
		home, _ := os.UserHomeDir()
		logPath = filepath.Join(home, logPath[2:])
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	w := io.MultiWriter(f, os.Stderr)
	log.SetOutput(w)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.Printf("app: log started → %s", logPath)
}

// initUserDirs creates the data directory and its subdirectories on first run,
// and writes a minimal default config.yml if one does not exist yet.
// It also extracts bundled binaries (e.g. func) to <dataDir>/bin/.
func initUserDirs() {
	migrateFromPiGo()
	base := clawfirmDataDir()
	bundled := filepath.Join(base, "bundled")
	dirs := []string{
		base,
		filepath.Join(base, "skills"),
		filepath.Join(base, "memory"),
		filepath.Join(base, "workflows"),
		filepath.Join(base, "canvas"),
		filepath.Join(base, "bin"),
		filepath.Join(base, "shortcuts"),
		bundled,
		filepath.Join(bundled, "skills"),
		filepath.Join(bundled, "workflows"),
		filepath.Join(bundled, "shortcuts"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			log.Printf("app: mkdir %s: %v", d, err)
		}
	}

	// Extract bundled binaries — only if they're real (> 4 KB, not placeholders).
	if len(embeddedFunc) > 4096 {
		funcPath := filepath.Join(base, "bin", "func")
		if err := os.WriteFile(funcPath, embeddedFunc, 0o755); err != nil {
			log.Printf("app: write func binary: %v", err)
		} else {
			log.Printf("app: extracted func binary to %s", funcPath)
		}
	}
	if len(embeddedClaw) > 4096 {
		clawPath := filepath.Join(base, "bin", "claw")
		if err := os.WriteFile(clawPath, embeddedClaw, 0o755); err != nil {
			log.Printf("app: write claw binary: %v", err)
		} else {
			log.Printf("app: extracted claw binary to %s", clawPath)
		}
	}
	if len(embeddedWhip) > 4096 {
		whipPath := filepath.Join(base, "bin", "whip")
		if err := os.WriteFile(whipPath, embeddedWhip, 0o755); err != nil {
			log.Printf("app: write whip binary: %v", err)
		} else {
			log.Printf("app: extracted whip binary to %s", whipPath)
		}
	}

	// Extract browser-shortcut binary and register as claw-code plugin.
	if len(embeddedBrowserShortcut) > 4096 {
		bsBin := filepath.Join(base, "bin", "browser-shortcut")
		if err := os.WriteFile(bsBin, embeddedBrowserShortcut, 0o755); err != nil {
			log.Printf("app: write browser-shortcut binary: %v", err)
		} else {
			log.Printf("app: extracted browser-shortcut binary to %s", bsBin)
			// Register as a claw-code plugin so claw agents can discover it.
			pluginDir := filepath.Join(base, "plugins", "installed", "browser-shortcut", ".claude-plugin")
			if err := os.MkdirAll(pluginDir, 0o755); err == nil {
				pluginJSON := fmt.Sprintf(`{
  "name": "browser-shortcut",
  "version": "0.1.0",
  "description": "Execute YAML browser automation shortcuts via Chrome DevTools Protocol (CDP). Shortcuts in ~/.clawfirm/shortcuts/.",
  "defaultEnabled": true,
  "tools": [{
    "name": "browser_shortcut",
    "description": "Execute a browser automation shortcut (YAML adapter) via CDP. Shortcuts are YAML files in ~/.clawfirm/shortcuts/ (e.g. douyin.yaml, xhs.yaml). Each shortcut defines commands like search, like, comment, follow, post. Requires Chrome with --remote-debugging-port=9222.",
    "inputSchema": {
      "type": "object",
      "properties": {
        "file": {"type":"string","description":"YAML shortcut filename (e.g. douyin.yaml)"},
        "command": {"type":"string","description":"Command to execute (e.g. search, like, comment)"},
        "args": {"type":"array","items":{"type":"string"},"description":"Positional arguments"}
      },
      "required": ["file","command"]
    },
    "command": %q,
    "requiredPermission": "danger-full-access"
  }]
}`, bsBin)
				if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(pluginJSON), 0o644); err != nil {
					log.Printf("app: write browser-shortcut plugin.json: %v", err)
				} else {
					// Also register in ~/.clawfirm/settings.json enabledPlugins so claw discovers it.
					registerClaudePlugin("browser-shortcut@external")
				}
			}
		}
	}

	migrateBundledAssets(base)
	extractBuiltinAssets(embeddedSkills, "assets/skills", filepath.Join(bundled, "skills"))
	extractBuiltinAssets(embeddedWorkflows, "assets/workflows", filepath.Join(bundled, "workflows"))
	extractBuiltinAssets(embeddedShortcuts, "assets/shortcuts", filepath.Join(bundled, "shortcuts"))

	// Extract media-understand binary and register as claw-code plugin.
	if len(embeddedMediaUnderstand) > 4096 {
		muBin := filepath.Join(base, "bin", "media-understand")
		if err := os.WriteFile(muBin, embeddedMediaUnderstand, 0o755); err != nil {
			log.Printf("app: write media-understand binary: %v", err)
		} else {
			pluginDir := filepath.Join(base, "plugins", "installed", "media-understand", ".claude-plugin")
			if err := os.MkdirAll(pluginDir, 0o755); err == nil {
				pluginJSON := fmt.Sprintf(`{
  "name": "media-understand",
  "version": "1.0.0",
  "description": "用视觉模型分析图片或视频帧内容，返回文字描述。",
  "defaultEnabled": true,
  "tools": [{
    "name": "media_understand",
    "description": "Analyse an image or video frame with a vision LLM and return a text description. Supports JPEG, PNG, GIF, WEBP.",
    "inputSchema": {
      "type": "object",
      "properties": {
        "file_path": {"type": "string", "description": "Local path to the image or video frame file"},
        "prompt":    {"type": "string", "description": "Optional analysis prompt"}
      },
      "required": ["file_path"]
    },
    "command": %q,
    "requiredPermission": "danger-full-access"
  }]
}`, muBin)
				if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(pluginJSON), 0o644); err != nil {
					log.Printf("app: write media-understand plugin.json: %v", err)
				} else {
					registerClaudePlugin("media-understand@external")
					log.Printf("app: registered claw plugin media-understand")
				}
			}
		}
	}

	// Extract media-gen binary and register as claw-code plugin.
	if len(embeddedMediaGen) > 4096 {
		mgBin := filepath.Join(base, "bin", "media-gen")
		if err := os.WriteFile(mgBin, embeddedMediaGen, 0o755); err != nil {
			log.Printf("app: write media-gen binary: %v", err)
		} else {
			pluginDir := filepath.Join(base, "plugins", "installed", "media-gen", ".claude-plugin")
			if err := os.MkdirAll(pluginDir, 0o755); err == nil {
				pluginJSON := fmt.Sprintf(`{
  "name": "media-gen",
  "version": "1.0.0",
  "description": "根据文字描述用 Gemini 生成图片，保存为 PNG 文件。",
  "defaultEnabled": true,
  "tools": [{
    "name": "media_gen",
    "description": "Generate an image from a text prompt using Gemini image generation and save it to a PNG file.",
    "inputSchema": {
      "type": "object",
      "properties": {
        "prompt":      {"type": "string", "description": "Image generation prompt (English works best)"},
        "output_path": {"type": "string", "description": "Output file path (default /tmp/media_gen_output.png)"}
      },
      "required": ["prompt"]
    },
    "command": %q,
    "requiredPermission": "danger-full-access"
  }]
}`, mgBin)
				if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(pluginJSON), 0o644); err != nil {
					log.Printf("app: write media-gen plugin.json: %v", err)
				} else {
					registerClaudePlugin("media-gen@external")
					log.Printf("app: registered claw plugin media-gen")
				}
			}
		}
	}

	cfgPath := filepath.Join(base, "config.yml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		defaultCfg := `# clawfirm configuration
# Docs: https://github.com/ai-gateway/clawfirm
providers:
    # anthropic:
    #     type: anthropic
    #     api_key: ${ANTHROPIC_API_KEY}
    # openai:
    #     type: openai
    #     api_key: ${OPENAI_API_KEY}

agents:
    - name: default
      provider: anthropic
      model: claude-sonnet-4-6
      system_prompt: ""
      tools:
        - read
        - write
        - edit
        - bash
        - memory_search
        - memory_get
        - whipflow_run
      skill_paths:
        - ~/.clawfirm/skills/

default_agent: ""

feishu:
    app_id: ""
    app_secret: ""

whatsapp:
    enabled: false

whipflow:
    default_provider: ""

cron_jobs: []
`
		if err := os.WriteFile(cfgPath, []byte(defaultCfg), 0o644); err != nil {
			log.Printf("app: write default config: %v", err)
		} else {
			log.Printf("app: created default config at %s", cfgPath)
		}
	}
}

// extractBuiltinAssets walks an embedded FS and extracts files to destDir.
// Files are always overwritten — destDir is app-managed (bundled/).
func extractBuiltinAssets(src embed.FS, prefix, destDir string) {
	fs.WalkDir(src, prefix, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(prefix, path)
		target := filepath.Join(destDir, rel)

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			log.Printf("app: mkdir %s: %v", filepath.Dir(target), err)
			return nil
		}

		data, err := src.ReadFile(path)
		if err != nil {
			log.Printf("app: read embedded %s: %v", path, err)
			return nil
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			log.Printf("app: write %s: %v", target, err)
		}
		return nil
	})
}

// migrateBundledAssets removes unmodified bundled files from user directories
// (one-time migration). Compares each embedded file with the user-dir copy;
// if identical, deletes the user copy so only the bundled/ copy remains.
func migrateBundledAssets(base string) {
	sentinel := filepath.Join(base, "bundled", ".migrated-v1")
	if _, err := os.Stat(sentinel); err == nil {
		return
	}

	type spec struct {
		src     embed.FS
		prefix  string
		userDir string
	}
	specs := []spec{
		{embeddedSkills, "assets/skills", filepath.Join(base, "skills")},
		{embeddedWorkflows, "assets/workflows", filepath.Join(base, "workflows")},
		{embeddedShortcuts, "assets/shortcuts", filepath.Join(base, "shortcuts")},
	}
	for _, s := range specs {
		fs.WalkDir(s.src, s.prefix, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(s.prefix, path)
			userFile := filepath.Join(s.userDir, rel)

			embeddedData, _ := s.src.ReadFile(path)
			diskData, err := os.ReadFile(userFile)
			if err != nil {
				return nil // not on disk
			}
			if bytes.Equal(embeddedData, diskData) {
				os.Remove(userFile)
				removeEmptyParents(filepath.Dir(userFile), s.userDir)
			}
			return nil
		})
	}
	_ = os.WriteFile(sentinel, []byte("migrated"), 0o644)
}

// removeEmptyParents removes empty directories from dir up to (but not including) stopAt.
func removeEmptyParents(dir, stopAt string) {
	for dir != stopAt && dir != filepath.Dir(dir) {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}
		os.Remove(dir)
		dir = filepath.Dir(dir)
	}
}

// clawfirmDataDir returns the data directory for this instance.
// Reads CLAWFIRM_DATA_DIR env var; defaults to ~/.clawfirm.
func clawfirmDataDir() string {
	if dir := os.Getenv("CLAWFIRM_DATA_DIR"); dir != "" {
		if strings.HasPrefix(dir, "~/") {
			home, _ := os.UserHomeDir()
			return filepath.Join(home, dir[2:])
		}
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".clawfirm")
}

// bundledDir returns <dataDir>/bundled/<sub>.
func bundledDir(sub string) string {
	return filepath.Join(clawfirmDataDir(), "bundled", sub)
}

// appendBundledSkillsDir returns a copy of paths with the bundled skills dir appended.
func appendBundledSkillsDir(paths []string) []string {
	out := make([]string, len(paths), len(paths)+1)
	copy(out, paths)
	return append(out, bundledDir("skills"))
}


// OnStartup is called by Wails once the frontend webview is ready.
func (a *App) OnStartup(ctx context.Context) {
	initUserDirs()
	_ = os.Chdir(clawfirmDataDir())
	initAppLog()
	applySystemProxy()
	a.ctx, a.cancelFn = context.WithCancel(ctx)

	// Load config (tolerates missing file).
	// PI_GO_CONFIG env var overrides the default path (used in tests).
	cfg, err := config.Load(os.Getenv("PI_GO_CONFIG"))
	if err != nil {
		log.Printf("app: config load: %v", err)
		cfg = &config.Config{Providers: make(map[string]config.ProviderConfig)}
	}
	a.cfg = cfg

	// Open auth storage.
	a.authStor = auth.NewAuthStorage("")
	if err := a.authStor.Load(); err != nil {
		log.Printf("app: auth load: %v", err)
	}

	// Open SQLite store.
	db, err := store.Open(filepath.Join(clawfirmDataDir(), "data.db"))
	if err != nil {
		log.Printf("app: store open: %v", err)
		a.startupErrors = append(a.startupErrors, fmt.Sprintf("数据库打开失败: %v", err))
	} else {
		a.db = db
	}

	// Migrate old unencrypted vault → new encrypted vault, then open.
	if a.db != nil {
		migrateVault(a.db)
	}
	if _, err := os.Stat(vault.DefaultDBPath()); err == nil {
		kc := keychain.New()
		if v, err := vault.Open(vault.DefaultDBPath(), kc); err != nil {
			log.Printf("app: vault open: %v", err)
		} else {
			a.vault = v
			if env, err := v.Env(); err != nil {
				log.Printf("app: vault env inject: %v", err)
			} else {
				for k, val := range env {
					os.Setenv(k, val)
				}
				writeVaultEnvFile(env)
				installVaultEnvHook()
				log.Printf("app: injected %d vault secrets into process env", len(env))
			}
		}
	}

	// Start gateway only if agents are configured.
	if len(cfg.Agents) > 0 {
		if err := a.startGateway(); err != nil {
			log.Printf("app: gateway start: %v", err)
			a.startupErrors = append(a.startupErrors, fmt.Sprintf("Gateway 启动失败: %v", err))
		}
	}

	// Write PID file so the watchdog daemon can track us.
	writeAppPID()

	// Start cron scheduler if DB is available.
	if a.db != nil {
		// Clean up any "running" history rows left over from a previous crash/restart.
		if err := a.db.CronJobs().MarkStaleRunningHistory(); err != nil {
			log.Printf("app: cron stale cleanup: %v", err)
		}
		a.syncCronConfigToDB()
		sched := picron.New(a.db.CronJobs(), a.buildAgentForCron)
		if err := sched.Start(a.ctx); err != nil {
			log.Printf("app: cron scheduler start: %v", err)
		}
		a.cronScheduler = sched
	}
}

// OnDomReady is called after the DOM is ready.
func (a *App) OnDomReady(_ context.Context) {
	for _, msg := range a.startupErrors {
		wailsruntime.EventsEmit(a.ctx, "startup:error", msg)
	}
}

// OnShutdown is called when the application is shutting down.
func (a *App) OnShutdown(_ context.Context) {
	// Signal the watchdog daemon that this was a clean exit.
	writeCleanExitMarker()
	removeAppPID()

	a.mu.Lock()
	if a.remoteSrv != nil {
		_ = a.remoteSrv.Stop()
		a.remoteSrv = nil
	}
	if a.remoteCancel != nil {
		a.remoteCancel()
		a.remoteCancel = nil
	}
	if a.whatsappCancel != nil {
		a.whatsappCancel()
		a.whatsappCancel = nil
	}
	if a.feishuCancel != nil {
		a.feishuCancel()
		a.feishuCancel = nil
	}
	if a.telegramCancel != nil {
		a.telegramCancel()
		a.telegramCancel = nil
	}
	a.mu.Unlock()
	if a.cancelFn != nil {
		a.cancelFn()
	}
	a.mu.Lock()
	if a.cronScheduler != nil {
		a.cronScheduler.Stop()
	}
	if a.registry != nil {
		a.registry.Stop()
	}
	if a.vault != nil {
		_ = a.vault.Close()
	}
	if a.db != nil {
		_ = a.db.Close()
	}
	a.mu.Unlock()
}

// startGateway builds providers and agents, then starts the HTTP server.
// Must be called while holding no locks (it acquires a.mu.Lock briefly).
func (a *App) startGateway() error {
	a.mu.Lock()
	cfg := a.cfg
	db := a.db
	memMgr := a.memoryMgr
	a.mu.Unlock()

	providerMap, err := buildProviders(cfg)
	if err != nil {
		return fmt.Errorf("startGateway: providers: %w", err)
	}

	// Resolve dedicated media provider if configured.
	var mediaProvider provider.LLMProvider
	if cfg.Media.Provider != "" {
		mediaProvider = providerMap[cfg.Media.Provider]
	}

	registry := gateway.NewAgentRegistry()
	var msgStore *store.MessageStore
	if db != nil {
		msgStore = db.Messages()
	}

	// Mark any "running" tool executions from a previous crash as "interrupted".
	a.recoverInterruptedToolExecs()

	for _, ac := range cfg.Agents {
		prov, ok := providerMap[ac.Provider]
		if !ok {
			log.Printf("app: skipping agent %q: provider %q not available", ac.Name, ac.Provider)
			continue
		}
		maxTokens := ac.MaxTokens
		if maxTokens == 0 {
			maxTokens = 16384
		}
		model := types.Model{ID: ac.Model, Provider: ac.Provider, MaxTokens: maxTokens}
		agentName := ac.Name
		tools := buildTools(ac.Tools, memMgr, cfg, a.vault, mediaProvider, agentbuilder.AgentRef{Provider: prov, Model: ac.Model})

		// Inject MessageSaver into WhipflowRun so each session's conversation
		// is persisted to DB as channelID="whipflow/<toolExecID>" userID="<idx>".
		if msgStore != nil {
			for _, t := range tools {
				if wr, ok := t.(*builtin.WhipflowRun); ok {
					ms := msgStore // capture
					wr.MessageSaver = func(toolExecID string, sessionIndex int, _ string, messages []any) {
						chID := "whipflow/" + toolExecID
						uID := strconv.Itoa(sessionIndex)
						for _, m := range messages {
							if msg, ok := m.(types.Message); ok {
								if err := ms.SaveMessage(chID, uID, msg); err != nil {
									log.Printf("app: whipflow save message: %v", err)
								}
							}
						}
					}
					break
				}
			}
		}

		// Load skills and build the skills prompt with size limits.
		// Bundled skills dir appended as fallback (user paths take priority).
		allSkillPaths := appendBundledSkillsDir(ac.SkillPaths)
		skillResult := skill.Load(skill.LoadOptions{SkillPaths: allSkillPaths})
		for _, d := range skillResult.Diagnostics {
			log.Printf("app: agent %s: skill warning: %s: %s", agentName, d.Path, d.Message)
		}
		if len(skillResult.Skills) > 0 {
			names := make([]string, len(skillResult.Skills))
			for i, s := range skillResult.Skills {
				names[i] = s.Name
			}
			log.Printf("app: agent %s: loaded %d skill(s): %s", agentName, len(skillResult.Skills), strings.Join(names, ", "))
		}
		compacted := skill.CompactSkillPaths(skillResult.Skills)
		skillsPrompt, truncated, compact := skill.ApplySkillsPromptLimits(compacted)
		if truncated {
			log.Printf("app: agent %s: skills prompt truncated (compact=%v)", agentName, compact)
		} else if compact {
			log.Printf("app: agent %s: skills prompt using compact format", agentName)
		}

		// Load bootstrap context files (AGENTS.md / CLAUDE.md / SOUL.md).
		bootstrap := agent.LoadBootstrapContext(ac.WorkspaceDir)

		// Build full system prompt.
		systemPrompt := agent.BuildSystemPrompt(agent.SystemPromptParams{
			WorkspaceDir:   ac.WorkspaceDir,
			SkillsPrompt:   skillsPrompt,
			ContextFiles:   bootstrap.ContextFiles,
			WorkspaceNotes: bootstrap.WorkspaceNotes,
			PromptMode:     agent.PromptModeFull,
			RuntimeInfo:    buildRuntimeInfo(ac),
			ExtraPrompt:    ac.SystemPrompt,
		})

		temporal := agent.NewTemporalInjector(0)
		loopCfg := agent.AgentLoopConfig{
			TransformContext: temporal.TransformContext,
		}

		factory := gateway.AgentFactory(func(channelID, userID string) gateway.AgentRunner {
			opts := []agent.AgentOption{
				agent.WithModel(model),
				agent.WithSystemPrompt(systemPrompt),
				agent.WithLoopConfig(loopCfg),
			}
			if len(tools) > 0 {
				opts = append(opts, agent.WithTools(tools))
			}
			ag := agent.NewAgent(prov, opts...)
			// Restore message history from store.
			storeChannelID := "webchat/" + agentName
			if msgStore != nil {
				if history, err := msgStore.ListMessages(store.QueryParams{
					ChannelID: storeChannelID,
					UserID:    userID,
				}); err == nil && len(history) > 0 {
					ag.ReplaceMessages(history)
					log.Printf("[%s] session %s/%s: restored %d messages", agentName, storeChannelID, userID, len(history))
				}
			}
			// Track tool executions and whipflow chains.
			ag.Subscribe(func(ev types.AgentEvent) {
				switch ev.Type {
				case types.EventToolExecutionStart:
					a.updateToolExec(storeChannelID, userID, ev.ToolCallID, func(r *ToolExecRecord) {
						r.Name = ev.ToolName
						r.Status = "running"
						r.StartedAt = time.Now().UnixMilli()
					})
					// Record whipflow chain entry on start.
					if ev.ToolName == "whipflow_run" && db != nil {
						parentID := ""
						sessionIdx := -1
						if args, ok := ev.ToolArgs.(map[string]any); ok {
							if p, ok := args["parent_call_id"].(string); ok {
								parentID = p
							}
							if raw, ok := args["stop_after_session"]; ok {
								switch v := raw.(type) {
								case float64:
									sessionIdx = int(v)
								case int:
									sessionIdx = v
								}
							}
						}
						if err := db.WhipflowChain().Insert(store.WhipflowChainEntry{
						ChannelID:  storeChannelID,
						UserID:     userID,
						CallID:     ev.ToolCallID,
						ParentID:   parentID,
						SessionIdx: sessionIdx,
						Status:     "running",
					}); err != nil {
						log.Printf("[%s] whipflow chain insert: %v", agentName, err)
					}
					}
				case types.EventToolExecutionEnd:
					a.updateToolExec(storeChannelID, userID, ev.ToolCallID, func(r *ToolExecRecord) {
						r.Status = "done"
						if ev.ToolIsError {
							r.Status = "error"
						}
						r.EndedAt = time.Now().UnixMilli()
					})
					// Update whipflow chain status on end.
					if ev.ToolName == "whipflow_run" && db != nil {
						status := "done"
						if ev.ToolIsError {
							status = "error"
						}
						if err := db.WhipflowChain().UpdateStatus(ev.ToolCallID, status); err != nil {
						log.Printf("[%s] whipflow chain update status: %v", agentName, err)
					}
					}
				}
			})
			return ag
		})

		var sessStore *store.SessionStore
		if db != nil {
			sessStore = db.Sessions()
		}

		// Build conversation summarizer if memory manager is available.
		var summarizer gateway.ConversationSummarizer
		if memMgr != nil {
			sumModel := types.Model{ID: ac.Model, Provider: ac.Provider}
			sumProv := prov
			sumFn := memory.SummarizeFunc(func(ctx context.Context, msgs []types.Message) (string, error) {
				return streamSummarize(ctx, sumProv, sumModel, msgs)
			})
			summarizer = memory.NewSummarizer(memMgr, sumFn, memory.SummarizerConfig{
				FilenamePrefix: "conv-" + ac.Name,
			})
		}

		// Parse reset policy from agent config.
		resetMode := store.ResetMode(ac.ResetMode)
		if resetMode == "" {
			resetMode = store.ResetModeNever
		}

		mgr := gateway.NewSessionManager(factory, gateway.ManagerConfig{
			AgentName:          ac.Name,
			SessionStore:       sessStore,
			Summarizer:         summarizer,
			DefaultResetMode:   resetMode,
			DefaultResetHour:   ac.ResetAtHour,
			DefaultIdleMinutes: ac.IdleMinutes,
			OnUserMessage: func(_, uid string, msg types.Message) error {
				if msgStore == nil {
					return nil
				}
				storeChannelID := "webchat/" + agentName
				if err := msgStore.SaveMessage(storeChannelID, uid, msg); err != nil {
					log.Printf("[%s] save user message: %v", agentName, err)
					return fmt.Errorf("保存消息失败，请重试")
				}
				return nil
			},
			OnAgentEvent: func(_, uid string, ev types.AgentEvent) error {
				if msgStore == nil {
					return nil
				}
				storeChannelID := "webchat/" + agentName
				switch ev.Type {
				case types.EventMessageEnd:
					if ev.AssistantMsg != nil {
						if err := msgStore.SaveMessage(storeChannelID, uid, ev.AssistantMsg); err != nil {
							log.Printf("[%s] save assistant message: %v", agentName, err)
							return fmt.Errorf("保存回复失败")
						}
					}
				case types.EventTurnEnd:
					if len(ev.ToolResults) > 0 {
						msgs := make([]types.Message, len(ev.ToolResults))
						for i := range ev.ToolResults {
							msgs[i] = &ev.ToolResults[i]
						}
						if err := msgStore.SaveMessages(storeChannelID, uid, msgs); err != nil {
							log.Printf("[%s] save tool results: %v", agentName, err)
							return fmt.Errorf("保存工具结果失败")
						}
					}
				}
				return nil
			},
		})
		registry.Register(ac.Name, mgr)
		log.Printf("app: agent %s  provider: %s  model: %s", ac.Name, ac.Provider, ac.Model)
	}

	srv := gateway.NewServer(registry, gateway.ServerConfig{Addr: "127.0.0.1:0"})

	defaultAgent := cfg.DefaultAgent
	if defaultAgent == "" && len(cfg.Agents) > 0 {
		defaultAgent = cfg.Agents[0].Name
	}
	handler := webchat.NewHandler(registry, defaultAgent)
	srv.Handle("GET /ws/{agentName}/{sessionID}", handler)
	srv.Handle("GET /ws/{sessionID}", handler)

	// Listen first, then store the address so GetWebhookBaseURL returns a
	// reachable endpoint only after the port is actually bound.
	ln, err := srv.Listen()
	if err != nil {
		return fmt.Errorf("startGateway: %w", err)
	}

	a.mu.Lock()
	a.registry = registry
	a.srvAddr = srv.Addr()
	a.mu.Unlock()

	go func() {
		if err := srv.Serve(a.ctx, ln); err != nil {
			log.Printf("app: gateway stopped: %v", err)
		}
	}()

	// Start WhatsApp channel only when explicitly enabled in config.
	if defaultAgent != "" && cfg.WhatsApp.Enabled {
		waCh := whatsapp.New(registry, defaultAgent)
		waCtx, waCancel := context.WithCancel(a.ctx)
		a.mu.Lock()
		a.whatsappCh = waCh
		a.whatsappCancel = waCancel
		a.mu.Unlock()
		go func() {
			if err := waCh.Start(waCtx); err != nil {
				log.Printf("app: whatsapp: %v", err)
			}
		}()
	}

	// Start Feishu channel if credentials are configured.
	if defaultAgent != "" && cfg.Feishu.AppID != "" && cfg.Feishu.AppSecret != "" {
		fsCh := feishu.New(cfg.Feishu.AppID, cfg.Feishu.AppSecret, registry, defaultAgent)
		fsCtx, fsCancel := context.WithCancel(a.ctx)
		a.mu.Lock()
		a.feishuCh = fsCh
		a.feishuCancel = fsCancel
		a.mu.Unlock()
		go func() {
			if err := fsCh.Start(fsCtx); err != nil {
				log.Printf("app: feishu: %v", err)
			}
		}()
	}

	// Start Telegram channel if bot token is configured.
	if defaultAgent != "" && cfg.Telegram.BotToken != "" {
		tgCh := telegram.New(cfg.Telegram.BotToken, registry, defaultAgent)
		tgCtx, tgCancel := context.WithCancel(a.ctx)
		a.mu.Lock()
		a.telegramCh = tgCh
		a.telegramCancel = tgCancel
		a.mu.Unlock()
		go func() {
			if err := tgCh.Start(tgCtx); err != nil {
				log.Printf("app: telegram: %v", err)
			}
		}()
	}

	log.Printf("app: gateway started on %s", srv.Addr())

	// Start lightweight HTTP server for JS eval (enables browser-shortcut control of App UI).
	a.startEvalServer()

	return nil
}

// stopGateway stops the running gateway, registry, and all channels (no-op if not running).
func (a *App) stopGateway() {
	a.mu.Lock()
	reg := a.registry
	a.registry = nil
	a.srvAddr = ""
	waCancel := a.whatsappCancel
	a.whatsappCh = nil
	a.whatsappCancel = nil
	fsCancel := a.feishuCancel
	a.feishuCh = nil
	a.feishuCancel = nil
	tgCancel := a.telegramCancel
	a.telegramCh = nil
	a.telegramCancel = nil
	remoteSrv := a.remoteSrv
	a.remoteSrv = nil
	remoteCancel := a.remoteCancel
	a.remoteCancel = nil
	a.mu.Unlock()
	if remoteSrv != nil {
		_ = remoteSrv.Stop()
	}
	if remoteCancel != nil {
		remoteCancel()
	}
	if waCancel != nil {
		waCancel()
	}
	if fsCancel != nil {
		fsCancel()
	}
	if tgCancel != nil {
		tgCancel()
	}
	if reg != nil {
		reg.Stop()
	}
}

// restartGateway stops any running gateway and starts a fresh one.
func (a *App) restartGateway() error {
	a.stopGateway()
	if len(a.cfg.Agents) == 0 {
		return nil
	}
	err := a.startGateway()
	// Reload cron scheduler after gateway restart (agent config may have changed).
	if a.cronScheduler != nil {
		if rerr := a.cronScheduler.Reload(); rerr != nil {
			log.Printf("app: cron reload: %v", rerr)
		}
	}
	return err
}

// ─── Frontend API ─────────────────────────────────────────────────────────────

// IsFirstRun returns true when no providers or agents are configured.
// Returns an error if the app is still initialising (frontend should retry).
func (a *App) IsFirstRun() (bool, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.cfg == nil {
		return false, fmt.Errorf("initialising")
	}
	return len(a.cfg.Providers) == 0 && len(a.cfg.Agents) == 0, nil
}

// GetConfig returns the current configuration.
func (a *App) GetConfig() *config.Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg
}

// installVaultEnvHook adds a source line to ~/.zshrc (or ~/.bashrc) so that
// vault secrets are automatically injected into every new shell session.
// It is idempotent — the line is only added once.
func installVaultEnvHook() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("app: vault hook: home dir: %v", err)
		return
	}

	const hookLine = `[ -f ~/.clawfirm/env ] && source ~/.clawfirm/env`

	// Pick rc file: prefer zsh, fall back to bash.
	rcPath := filepath.Join(home, ".zshrc")
	if _, err := os.Stat(rcPath); os.IsNotExist(err) {
		rcPath = filepath.Join(home, ".bashrc")
	}

	data, _ := os.ReadFile(rcPath)
	if strings.Contains(string(data), hookLine) {
		return // already installed
	}

	f, err := os.OpenFile(rcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("app: vault hook: open %s: %v", rcPath, err)
		return
	}
	defer f.Close()

	if _, err := fmt.Fprintf(f, "\n%s\n", hookLine); err != nil {
		log.Printf("app: vault hook: write: %v", err)
		return
	}
	log.Printf("app: vault hook installed in %s", rcPath)
}

// writeAppPID writes the current process PID to <dataDir>/app.pid.
func writeAppPID() {
	_ = os.WriteFile(filepath.Join(clawfirmDataDir(), "app.pid"), []byte(strconv.Itoa(os.Getpid())), 0644)
}

// removeAppPID removes the <dataDir>/app.pid file.
func removeAppPID() {
	os.Remove(filepath.Join(clawfirmDataDir(), "app.pid"))
}

// writeCleanExitMarker writes a <dataDir>/clean_exit file to signal the
// watchdog daemon that the app shut down intentionally.
func writeCleanExitMarker() {
	_ = os.WriteFile(filepath.Join(clawfirmDataDir(), "clean_exit"), []byte("clean"), 0644)
}

// writeVaultEnvFile writes vault secrets to ~/.clawfirm/env as export statements
// so that shells sourcing that file inherit them automatically.
// File is written with mode 0600 (owner read/write only).
func writeVaultEnvFile(env map[string]string) {
	dir := vault.DefaultDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		log.Printf("app: vault env file: mkdir: %v", err)
		return
	}
	path := filepath.Join(dir, "env")
	var b strings.Builder
	b.WriteString("# Generated by clawfirm — do not edit manually\n")
	for k, val := range env {
		b.WriteString("export ")
		b.WriteString(k)
		b.WriteString("='")
		b.WriteString(strings.ReplaceAll(val, "'", `'\''`))
		b.WriteString("'\n")
	}
	if err := os.WriteFile(path, []byte(b.String()), 0600); err != nil {
		log.Printf("app: vault env file: write: %v", err)
	}
}

// ─── Vault ────────────────────────────────────────────────────────────────────

// VaultEntry is returned by GetVault for the frontend.
type VaultEntry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// GetVault returns all vault entries (keys + decrypted values).
func (a *App) GetVault() ([]VaultEntry, error) {
	a.mu.RLock()
	v := a.vault
	a.mu.RUnlock()
	if v == nil {
		return nil, nil
	}
	keys, err := v.List()
	if err != nil {
		return nil, err
	}
	out := make([]VaultEntry, 0, len(keys))
	for _, k := range keys {
		val, err := v.Get(k)
		if err != nil {
			return nil, fmt.Errorf("vault get %q: %w", k, err)
		}
		out = append(out, VaultEntry{Key: k, Value: string(val)})
	}
	return out, nil
}

// SetVaultEntry encrypts and stores a vault key-value pair.
func (a *App) SetVaultEntry(key, value string) error {
	a.mu.RLock()
	v := a.vault
	a.mu.RUnlock()
	if v == nil {
		return fmt.Errorf("vault: not initialized — run 'clawfirm vault init' first")
	}
	return v.Set(key, []byte(value))
}

// DeleteVaultEntry removes a vault entry by key.
func (a *App) DeleteVaultEntry(key string) error {
	a.mu.RLock()
	v := a.vault
	a.mu.RUnlock()
	if v == nil {
		return fmt.Errorf("vault: not initialized — run 'clawfirm vault init' first")
	}
	return v.Delete(key)
}

// ReadCanvasFile reads <dataDir>/canvas/{name}.html and returns its content.
// Returns empty string if the file does not exist yet.
func (a *App) ReadCanvasFile(name string) (string, error) {
	if name == "" || strings.ContainsAny(name, "/\\..") {
		return "", fmt.Errorf("invalid canvas name: %q", name)
	}
	path := filepath.Join(clawfirmDataDir(), "canvas", name+".html")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ListCanvasFiles returns all .html file names (without extension) in <dataDir>/canvas/.
func (a *App) ListCanvasFiles() ([]string, error) {
	dir := filepath.Join(clawfirmDataDir(), "canvas")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".html" {
			names = append(names, strings.TrimSuffix(e.Name(), ".html"))
		}
	}
	return names, nil
}

// WriteCanvasFile writes content to <dataDir>/canvas/{name}.html.
func (a *App) WriteCanvasFile(name, content string) error {
	if name == "" || strings.ContainsAny(name, "/\\..") {
		return fmt.Errorf("invalid canvas name: %q", name)
	}
	dir := filepath.Join(clawfirmDataDir(), "canvas")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name+".html"), []byte(content), 0o644)
}

// SaveConfig writes the config to ~/.clawfirm/config.yml and restarts the gateway.
func (a *App) SaveConfig(cfg *config.Config) error {
	if err := saveConfig(cfg); err != nil {
		return err
	}
	a.mu.Lock()
	a.cfg = cfg
	a.mu.Unlock()
	return a.restartGateway()
}

// GetProviders returns all configured providers with a hasKey flag.
func (a *App) GetProviders() []ProviderInfo {
	a.mu.RLock()
	cfg := a.cfg
	authStor := a.authStor
	a.mu.RUnlock()

	out := make([]ProviderInfo, 0, len(cfg.Providers))
	for id, pc := range cfg.Providers {
		hasKey := pc.APIKey != ""
		if !hasKey && authStor != nil {
			_, hasKey = authStor.GetAPIKey(id)
		}
		out = append(out, ProviderInfo{ID: id, Type: pc.Type, HasKey: hasKey})
	}
	return out
}

// SaveAPIKey stores an API key for the given provider and updates config.
func (a *App) SaveAPIKey(providerID, key string) error {
	a.mu.RLock()
	authStor := a.authStor
	cfg := a.cfg
	a.mu.RUnlock()

	if authStor != nil {
		if err := authStor.SetAPIKey(providerID, key); err != nil {
			return err
		}
	}

	// Also patch config so the provider exists with the right type if absent.
	if _, ok := cfg.Providers[providerID]; !ok {
		a.mu.Lock()
		if cfg.Providers == nil {
			cfg.Providers = make(map[string]config.ProviderConfig)
		}
		cfg.Providers[providerID] = config.ProviderConfig{Type: providerID, APIKey: key}
		a.mu.Unlock()
		if err := saveConfig(cfg); err != nil {
			return err
		}
	}
	return nil
}

// StartOAuthLogin opens the system browser for OAuth flow and returns immediately.
// The frontend should listen for the "oauth:callback" Wails event.
func (a *App) StartOAuthLogin(providerID string) error {
	// Build the OAuth URL based on provider.
	// Currently a placeholder — each provider would need its own redirect URL.
	oauthURL := fmt.Sprintf("https://auth.%s.example.com/oauth/authorize", providerID)
	return openBrowser(oauthURL)
}

// GetModels returns a static list of known models for a provider type.
// For Ollama it returns an empty list (user should type model name).
func (a *App) GetModels(providerID string) []string {
	a.mu.RLock()
	cfg := a.cfg
	a.mu.RUnlock()

	pc, ok := cfg.Providers[providerID]
	if !ok {
		return nil
	}
	t := pc.Type
	if t == "" {
		t = providerID
	}
	switch t {
	case "anthropic":
		return []string{
			"claude-opus-4-5-20251101",
			"claude-sonnet-4-5-20251001",
			"claude-haiku-4-5-20251001",
			"claude-3-5-sonnet-20241022",
			"claude-3-5-haiku-20241022",
		}
	case "openai":
		return []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "o1-mini", "o3-mini"}
	case "gemini":
		return []string{"gemini-2.0-flash", "gemini-1.5-pro", "gemini-1.5-flash", "gemini-2.0-flash-lite"}
	case "zenmux":
		return []string{
			"anthropic/claude-sonnet-4-5", "anthropic/claude-haiku-4-5",
			"openai/gpt-4o", "openai/gpt-4o-mini",
			"google/gemini-2.0-flash",
		}
	case "ollama", "sglang", "vllm", "litellm":
		return []string{} // dynamic; user types model name
	case "deepseek":
		return []string{"deepseek-chat", "deepseek-reasoner"}
	case "minimax":
		return []string{"MiniMax-M2.7", "MiniMax-M2.7-highspeed", "MiniMax-M2.5", "MiniMax-M2.5-highspeed", "MiniMax-M2.1", "MiniMax-M2"}
	case "moonshot":
		return []string{"moonshot-v1-8k", "moonshot-v1-32k", "moonshot-v1-128k"}
	case "volcengine":
		return []string{"doubao-pro-32k", "doubao-pro-4k", "doubao-lite-32k", "doubao-lite-4k"}
	case "modelstudio":
		return []string{"qwen-max", "qwen-plus", "qwen-turbo", "qwen-long", "qwen2.5-72b-instruct"}
	case "glm", "zai":
		return []string{"glm-4-flash", "glm-4-air", "glm-4", "glm-4-plus", "glm-z1-flash"}
	case "groq":
		return []string{"llama-3.3-70b-versatile", "llama-3.1-8b-instant", "mixtral-8x7b-32768", "gemma2-9b-it"}
	case "openrouter":
		return []string{
			"anthropic/claude-3.5-sonnet", "openai/gpt-4o",
			"google/gemini-flash-1.5", "meta-llama/llama-3.1-70b-instruct",
		}
	case "together":
		return []string{
			"meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo",
			"mistralai/Mixtral-8x7B-Instruct-v0.1",
			"Qwen/Qwen2.5-72B-Instruct-Turbo",
		}
	case "mistral":
		return []string{"mistral-large-latest", "mistral-small-latest", "codestral-latest", "open-mistral-nemo"}
	case "xai":
		return []string{"grok-3", "grok-3-mini", "grok-2", "grok-beta"}
	case "nvidia":
		return []string{"nvidia/llama-3.1-nemotron-70b-instruct", "meta/llama-3.1-405b-instruct", "mistralai/mistral-large-2-instruct"}
	case "xiaomi":
		return []string{"mimo-v2-flash", "mimo-v2-pro", "mimo-v2-omni"}
	case "venice":
		return []string{"llama-3.3-70b", "mistral-31-24b", "qwen-2.5-qwq-32b"}
	case "huggingface":
		return []string{"meta-llama/Llama-3.3-70B-Instruct", "Qwen/Qwen2.5-72B-Instruct", "mistralai/Mistral-7B-Instruct-v0.3"}
	case "perplexity":
		return []string{"sonar-pro", "sonar", "sonar-reasoning-pro", "sonar-reasoning"}
	default:
		return nil
	}
}

// TestProviderConnection is a stub — provider credential testing is now done
// via the claw subprocess. Always returns true to avoid breaking the frontend.
func (a *App) TestProviderConnection(providerID string) bool {
	return true
}

// GetChannels returns the list of configured agents with live session counts.
func (a *App) GetChannels() []ChannelInfo {
	a.mu.RLock()
	cfg := a.cfg
	registry := a.registry
	a.mu.RUnlock()

	var counts map[string]int
	if registry != nil {
		counts = registry.Counts()
	}

	out := make([]ChannelInfo, 0, len(cfg.Agents))
	for _, ac := range cfg.Agents {
		ci := ChannelInfo{
			Name:     ac.Name,
			Provider: ac.Provider,
			Model:    ac.Model,
			Sessions: counts[ac.Name],
		}
		out = append(out, ci)
	}
	return out
}

// SaveChannelConfig upserts an agent config and restarts the gateway.
func (a *App) SaveChannelConfig(ac config.AgentConfig) error {
	a.mu.Lock()
	updated := false
	for i, existing := range a.cfg.Agents {
		if existing.Name == ac.Name {
			a.cfg.Agents[i] = ac
			updated = true
			break
		}
	}
	if !updated {
		a.cfg.Agents = append(a.cfg.Agents, ac)
	}
	cfg := a.cfg
	a.mu.Unlock()

	if err := saveConfig(cfg); err != nil {
		return err
	}
	return a.restartGateway()
}

// DeleteChannelConfig removes an agent by name and restarts the gateway.
func (a *App) DeleteChannelConfig(name string) error {
	a.mu.Lock()
	agents := make([]config.AgentConfig, 0, len(a.cfg.Agents))
	for _, ac := range a.cfg.Agents {
		if ac.Name != name {
			agents = append(agents, ac)
		}
	}
	a.cfg.Agents = agents
	cfg := a.cfg
	a.mu.Unlock()

	if err := saveConfig(cfg); err != nil {
		return err
	}
	return a.restartGateway()
}

// TestChannelConnection checks if the named agent's session manager is registered.
func (a *App) TestChannelConnection(name string) bool {
	a.mu.RLock()
	registry := a.registry
	a.mu.RUnlock()
	if registry == nil {
		return false
	}
	_, ok := registry.Get(name)
	return ok
}

// SendMessage enqueues a user message to the named agent/session.
func (a *App) SendMessage(agentName, sessionID, content string) error {
	a.mu.RLock()
	registry := a.registry
	a.mu.RUnlock()
	if registry == nil {
		return fmt.Errorf("gateway not running")
	}
	mgr, ok := registry.Get(agentName)
	if !ok {
		return fmt.Errorf("agent %q not found", agentName)
	}
	sess, err := mgr.GetOrCreate("desktop", sessionID)
	if err != nil {
		return err
	}
	if !sess.Send(gateway.IncomingMessage{ChannelID: "desktop", UserID: sessionID, Content: content}) {
		return fmt.Errorf("session queue full")
	}
	return nil
}

// AbortCurrentTurn cancels the in-progress agent turn for the given session.
func (a *App) AbortCurrentTurn(agentName, sessionID string) error {
	a.mu.RLock()
	registry := a.registry
	a.mu.RUnlock()
	if registry == nil {
		return fmt.Errorf("gateway not running")
	}
	mgr, ok := registry.Get(agentName)
	if !ok {
		return fmt.Errorf("agent %q not found", agentName)
	}
	sess, err := mgr.GetOrCreate("webchat", sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}
	sess.Abort()
	return nil
}

// ToolExecutionInfo is a summary of one tool call + result for the frontend.
type ToolExecutionInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Args      any    `json:"args,omitempty"`
	Result    string `json:"result,omitempty"`
	IsError   bool   `json:"isError"`
	Timestamp int64  `json:"timestamp"`
	Status    string `json:"status,omitempty"`    // "running" | "done" | "error" | "interrupted"
	StartedAt int64  `json:"startedAt,omitempty"`
	EndedAt   int64  `json:"endedAt,omitempty"`
}

// ---------------------------------------------------------------------------
// Tool execution ledger (KV-backed)
// ---------------------------------------------------------------------------

// toolExecKey returns the KV key for a channel+user's tool execution ledger.
func toolExecKey(channelID, userID string) string {
	return "tool_execs:" + channelID + ":" + userID
}

// loadToolExecs reads the tool execution ledger from KV. Returns empty slice on not-found.
func (a *App) loadToolExecs(channelID, userID string) []ToolExecRecord {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		return nil
	}
	var records []ToolExecRecord
	if err := db.KV().Get(toolExecKey(channelID, userID), &records); err != nil {
		return nil
	}
	return records
}

// saveToolExecs writes the tool execution ledger to KV.
func (a *App) saveToolExecs(channelID, userID string, records []ToolExecRecord) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		return
	}
	if err := db.KV().Set(toolExecKey(channelID, userID), records); err != nil {
		log.Printf("app: save tool execs: %v", err)
	}
}

// updateToolExec applies fn to the record with the given id (creating it if needed), then saves.
func (a *App) updateToolExec(channelID, userID, id string, fn func(*ToolExecRecord)) {
	records := a.loadToolExecs(channelID, userID)
	found := false
	for i := range records {
		if records[i].ID == id {
			fn(&records[i])
			found = true
			break
		}
	}
	if !found {
		r := ToolExecRecord{ID: id}
		fn(&r)
		records = append(records, r)
	}
	a.saveToolExecs(channelID, userID, records)
}

// recoverInterruptedToolExecs marks any "running" tool executions as "interrupted".
// Called once at startup before creating session managers.
func (a *App) recoverInterruptedToolExecs() {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		return
	}
	entries, err := db.KV().ScanPrefix("tool_execs:")
	if err != nil {
		log.Printf("app: scan tool_execs: %v", err)
		return
	}
	for key, raw := range entries {
		var records []ToolExecRecord
		if err := json.Unmarshal([]byte(raw), &records); err != nil {
			continue
		}
		changed := false
		for i := range records {
			if records[i].Status == "running" {
				records[i].Status = "interrupted"
				records[i].EndedAt = time.Now().UnixMilli()
				changed = true
			}
		}
		if changed {
			if err := db.KV().Set(key, records); err != nil {
				log.Printf("app: recover tool execs %s: %v", key, err)
			}
		}
	}
}

// GetToolExecutionState returns tool execution info with correct lifecycle status from the KV ledger.
// For done/error records, it reads the ToolResultMessage from the message store for result text.
func (a *App) GetToolExecutionState(channelID, userID string) ([]ToolExecutionInfo, error) {
	records := a.loadToolExecs(channelID, userID)
	if len(records) == 0 {
		return nil, nil
	}

	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()

	// Build a map of tool call ID → result text from messages (for done/error records).
	resultByID := make(map[string]string)
	errorByID := make(map[string]bool)
	if db != nil {
		msgs, err := db.Messages().ListMessages(store.QueryParams{
			ChannelID: channelID,
			UserID:    userID,
			Limit:     200,
		})
		if err == nil {
			for _, m := range msgs {
				if tr, ok := m.(*types.ToolResultMessage); ok {
					for _, b := range tr.Content {
						if t, ok := b.(*types.TextContent); ok {
							resultByID[tr.ToolCallID] = t.Text
							break
						}
					}
					errorByID[tr.ToolCallID] = tr.IsError
				}
			}
		}
	}

	out := make([]ToolExecutionInfo, 0, len(records))
	for _, r := range records {
		info := ToolExecutionInfo{
			ID:        r.ID,
			Name:      r.Name,
			IsError:   r.Status == "error",
			Timestamp: r.StartedAt,
			Status:    r.Status,
			StartedAt: r.StartedAt,
			EndedAt:   r.EndedAt,
		}
		// Fill result text from message store.
		if text, ok := resultByID[r.ID]; ok {
			info.Result = text
		}
		if isErr, ok := errorByID[r.ID]; ok {
			info.IsError = isErr
		}
		out = append(out, info)
	}
	return out, nil
}


// GetWhipflowChain returns the whipflow execution chain for a channel+user session.
func (a *App) GetWhipflowChain(channelID, userID string) ([]store.WhipflowChainEntry, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}
	return db.WhipflowChain().ListBySession(channelID, userID)
}

// GetHistory returns the last N messages for a channel+user.
// If no messages are found under channelID, it falls back to the legacy "webchat" channel.
func (a *App) GetHistory(channelID, userID string) ([]map[string]string, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}
	msgs, err := db.Messages().ListMessages(store.QueryParams{
		ChannelID: channelID,
		UserID:    userID,
		Limit:     50,
	})
	if err != nil {
		return nil, err
	}
	// Fall back to legacy "webchat" channel if no messages found under the new key.
	if len(msgs) == 0 && channelID != "webchat" {
		msgs, err = db.Messages().ListMessages(store.QueryParams{
			ChannelID: "webchat",
			UserID:    userID,
			Limit:     50,
		})
		if err != nil {
			return nil, err
		}
	}
	out := make([]map[string]string, 0, len(msgs))
	for _, m := range msgs {
		role := "user"
		if _, ok := m.(*types.AssistantMessage); ok {
			role = "assistant"
		}
		// Flatten to role+text for simple frontend rendering.
		// Skip tool_use / tool_result messages (empty text).
		text := extractText(m)
		if text == "" {
			continue
		}
		out = append(out, map[string]string{"role": role, "content": text})
	}
	return out, nil
}


// GetSessions returns a list of active session keys from all registered agents.
func (a *App) GetSessions() []SessionInfo {
	a.mu.RLock()
	registry := a.registry
	cfg := a.cfg
	a.mu.RUnlock()
	if registry == nil {
		return nil
	}

	var out []SessionInfo
	for _, agentName := range registry.Names() {
		mgr, ok := registry.Get(agentName)
		if !ok {
			continue
		}
		_ = mgr
		_ = cfg
		out = append(out, SessionInfo{AgentName: agentName})
	}
	return out
}

// GetChatSessions returns distinct session IDs (user_ids) that have messages
// stored for a given agent, ordered by most recent activity.
// It merges results from both "webchat/<agentName>" (new) and "webchat" (legacy).
func (a *App) GetChatSessions(agentName string) ([]string, error) {
	a.mu.RLock()
	db := a.db
	a.mu.RUnlock()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}
	ids1, err := db.Messages().ListUserIDs("webchat/" + agentName)
	if err != nil {
		return nil, err
	}
	ids2, err := db.Messages().ListUserIDs("webchat")
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(ids1))
	out := make([]string, 0, len(ids1)+len(ids2))
	for _, id := range ids1 {
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, id := range ids2 {
		if _, ok := seen[id]; !ok {
			out = append(out, id)
		}
	}
	log.Printf("GetChatSessions(%s): webchat/%s=%d webchat=%d total=%d", agentName, agentName, len(ids1), len(ids2), len(out))
	return out, nil
}

// SessionDetail is a rich session descriptor returned to the frontend.
type SessionDetail struct {
	SessionKey       string  `json:"sessionKey"`
	SessionID        string  `json:"sessionId"`
	AgentName        string  `json:"agentName"`
	ChannelID        string  `json:"channelId"`
	UserID           string  `json:"userId"`
	Subject          string  `json:"subject"`
	ChatType         string  `json:"chatType"`
	InputTokens      int     `json:"inputTokens"`
	OutputTokens     int     `json:"outputTokens"`
	TotalTokens      int     `json:"totalTokens"`
	EstimatedCostUSD float64 `json:"estimatedCostUsd"`
	Model            string  `json:"model"`
	ModelProvider    string  `json:"modelProvider"`
	CreatedAt        int64   `json:"createdAt"` // epoch ms
	UpdatedAt        int64   `json:"updatedAt"` // epoch ms
	IsActive         bool    `json:"isActive"`
}

// ListSessions returns all persisted sessions for a given agent name.
func (a *App) ListSessions(agentName string) ([]SessionDetail, error) {
	a.mu.RLock()
	db := a.db
	registry := a.registry
	a.mu.RUnlock()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}
	entries, err := db.Sessions().ListByAgent(agentName)
	if err != nil {
		return nil, err
	}

	// Build set of active session keys for IsActive flag.
	activeKeys := make(map[string]struct{})
	if registry != nil {
		if mgr, ok := registry.Get(agentName); ok {
			for _, s := range mgr.ActiveSessions() {
				activeKeys[s.Key()] = struct{}{}
			}
		}
	}

	out := make([]SessionDetail, 0, len(entries))
	for _, e := range entries {
		_, active := activeKeys[e.SessionKey]
		out = append(out, SessionDetail{
			SessionKey:       e.SessionKey,
			SessionID:        e.SessionID,
			AgentName:        agentName,
			ChannelID:        e.ChannelID,
			UserID:           e.UserID,
			Subject:          e.Subject,
			ChatType:         e.ChatType,
			InputTokens:      e.InputTokens,
			OutputTokens:     e.OutputTokens,
			TotalTokens:      e.TotalTokens,
			EstimatedCostUSD: e.EstimatedCostUSD,
			Model:            e.Model,
			ModelProvider:    e.ModelProvider,
			CreatedAt:        e.CreatedAt.UnixMilli(),
			UpdatedAt:        e.UpdatedAt.UnixMilli(),
			IsActive:         active,
		})
	}
	return out, nil
}

// ResetSession clears the message history for the given session and marks it reset.
func (a *App) ResetSession(agentName, sessionKey string) error {
	a.mu.RLock()
	db := a.db
	registry := a.registry
	a.mu.RUnlock()

	// Evict the active session so the next message starts fresh.
	if registry != nil {
		if mgr, ok := registry.Get(agentName); ok {
			mgr.RemoveByKey(sessionKey)
		}
	}

	if db == nil {
		return nil
	}

	// Look up channel/user for message deletion.
	entry, err := db.Sessions().Get(sessionKey)
	if err != nil {
		return err
	}
	if entry == nil {
		return nil
	}

	storeChannelID := "webchat/" + agentName
	if err := db.Messages().DeleteByChannelUser(storeChannelID, entry.UserID); err != nil {
		return err
	}
	return db.Sessions().MarkReset(sessionKey)
}

// GetConfigRaw returns the raw YAML content of config.yml and its file path.
func (a *App) GetConfigRaw() (map[string]string, error) {
	p := filepath.Join(clawfirmDataDir(), "config.yml")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return map[string]string{"path": p, "content": string(data)}, nil
}

// SaveConfigRaw writes raw YAML to config.yml and reloads the gateway.
func (a *App) SaveConfigRaw(content string) error {
	cfg, err := config.ParseYAML([]byte(content))
	if err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}
	if err := saveConfig(cfg); err != nil {
		return err
	}
	a.mu.Lock()
	a.cfg = cfg
	a.mu.Unlock()
	return a.restartGateway()
}

// GetVersion returns the application version string.
func (a *App) GetVersion() string { return Version }

// EvalJS evaluates arbitrary JavaScript in the WKWebView and returns the result.
// It is the bridge that enables browser-shortcut tools to control the App UI via JS eval.
func (a *App) EvalJS(script string) string {
	id, ch := a.addPendingEval()
	wailsruntime.EventsEmit(a.ctx, "app:eval", map[string]any{
		"id":     id,
		"script": script,
	})
	select {
	case result := <-ch:
		return result
	case <-time.After(30 * time.Second):
		return fmt.Sprintf("eval timeout for id %d", id)
	}
}

// EvalResult is called by the frontend with the result of a JS eval.
func (a *App) EvalResult(id int, result string) {
	a.mu.Lock()
	ch, ok := a.evalPending[id]
	delete(a.evalPending, id)
	a.mu.Unlock()
	if ok && !closed(ch) {
		ch <- result
	}
}

func (a *App) addPendingEval() (int, chan string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.evalSeq++
	if a.evalPending == nil {
		a.evalPending = make(map[int]chan string)
	}
	ch := make(chan string, 1)
	a.evalPending[a.evalSeq] = ch
	return a.evalSeq, ch
}

func closed(ch chan string) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

// OpenLogsFolder opens the data directory in the system file manager.
func (a *App) OpenLogsFolder() error {
	return openBrowser(clawfirmDataDir())
}

// GetWebhookBaseURL returns the HTTP base URL of the embedded gateway.
func (a *App) GetWebhookBaseURL() string {
	a.mu.RLock()
	addr := a.srvAddr
	a.mu.RUnlock()
	if addr == "" {
		return ""
	}
	return "http://" + addr
}

// GetWhatsAppStatus returns the WhatsApp channel connection status.
// Returns "disabled" when not enabled in config.
func (a *App) GetWhatsAppStatus() string {
	a.mu.RLock()
	ch := a.whatsappCh
	enabled := a.cfg != nil && a.cfg.WhatsApp.Enabled
	a.mu.RUnlock()
	if !enabled {
		return "disabled"
	}
	if ch == nil {
		return whatsapp.StatusDisconnected
	}
	return ch.GetStatus()
}

// GetWhatsAppQR returns the current QR code data URL, or "" if not in qr_pending state.
func (a *App) GetWhatsAppQR() string {
	a.mu.RLock()
	ch := a.whatsappCh
	a.mu.RUnlock()
	if ch == nil {
		return ""
	}
	return ch.GetQR()
}

// GetFeishuConfig returns the current Feishu credentials (AppSecret is masked).
func (a *App) GetFeishuConfig() map[string]string {
	a.mu.RLock()
	cfg := a.cfg
	a.mu.RUnlock()
	appID := cfg.Feishu.AppID
	hasSecret := cfg.Feishu.AppSecret != ""
	masked := ""
	if hasSecret {
		masked = "••••••••"
	}
	return map[string]string{"appId": appID, "appSecretMasked": masked}
}

// SaveFeishuConfig persists Feishu credentials and restarts the gateway.
func (a *App) SaveFeishuConfig(appID, appSecret string) error {
	a.mu.Lock()
	a.cfg.Feishu.AppID = appID
	a.cfg.Feishu.AppSecret = appSecret
	cfg := a.cfg
	a.mu.Unlock()
	if err := saveConfig(cfg); err != nil {
		return err
	}
	return a.restartGateway()
}

// LogoutWhatsApp logs the WhatsApp session out.
func (a *App) LogoutWhatsApp() error {
	a.mu.RLock()
	ch := a.whatsappCh
	a.mu.RUnlock()
	if ch == nil {
		return nil
	}
	return ch.Logout(context.Background())
}

// ─── Remote Control ───────────────────────────────────────────────────────────

// remoteLog writes a line to <dataDir>/remote.log for debugging.
func remoteLog(format string, args ...any) {
	f, err := os.OpenFile(filepath.Join(clawfirmDataDir(), "remote.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, time.Now().Format("15:04:05")+" "+format+"\n", args...)
}

// EnableRemote starts the remote-control server (LAN mode).
func (a *App) EnableRemote() (remote.RemoteStatus, error) {
	remoteLog("EnableRemote called")

	a.mu.Lock()
	if a.remoteSrv != nil {
		st := a.remoteSrv.Status()
		a.mu.Unlock()
		remoteLog("already running, returning status")
		return st, nil
	}
	registry := a.registry
	db := a.db
	cfg := a.cfg
	a.mu.Unlock()

	if registry == nil {
		remoteLog("ERROR: registry is nil")
		return remote.RemoteStatus{}, fmt.Errorf("gateway not running — configure agents first")
	}

	canvasDir := filepath.Join(clawfirmDataDir(), "canvas")

	remoteLog("creating server...")
	srv := remote.NewServer(remote.Config{
		Registry:        registry,
		DB:              db,
		Cfg:             cfg,
		CanvasDir:       canvasDir,
		ChannelStatusFn: a.channelStatusFunc(),
	})

	ctx, cancel := context.WithCancel(a.ctx)
	remoteLog("starting server...")
	if err := srv.Start(ctx); err != nil {
		cancel()
		remoteLog("ERROR start: %v", err)
		return remote.RemoteStatus{}, err
	}
	remoteLog("Start() returned OK")

	a.mu.Lock()
	a.remoteSrv = srv
	a.remoteCancel = cancel
	a.mu.Unlock()
	remoteLog("saved to app struct")

	st := srv.Status()
	remoteLog("Status() returned: enabled=%v LAN=%s port=%d QR=%d bytes", st.Enabled, st.LanURL, st.Port, len(st.QRCode))

	// Serialize to JSON to verify it can be marshaled.
	if data, err := json.Marshal(st); err != nil {
		remoteLog("ERROR json.Marshal: %v", err)
	} else {
		remoteLog("JSON OK: %d bytes", len(data))
	}

	remoteLog("returning status to Wails")
	return st, nil
}

// DisableRemote stops the remote-control server (including any ngrok tunnel).
func (a *App) DisableRemote() error {
	a.mu.Lock()
	srv := a.remoteSrv
	a.remoteSrv = nil
	cancel := a.remoteCancel
	a.remoteCancel = nil
	a.mu.Unlock()

	if srv != nil {
		_ = srv.Stop()
	}
	if cancel != nil {
		cancel()
	}
	return nil
}

// EnableNgrok starts an ngrok tunnel for cross-network access.
// If the remote server is not running, it starts it first.
func (a *App) EnableNgrok(authToken string) (remote.RemoteStatus, error) {
	remoteLog("EnableNgrok called with token len=%d", len(authToken))

	// Ensure the remote server is running first.
	a.mu.RLock()
	srv := a.remoteSrv
	a.mu.RUnlock()

	if srv == nil {
		remoteLog("remote server not running, starting it first...")
		st, err := a.EnableRemote()
		if err != nil {
			return st, err
		}
		a.mu.RLock()
		srv = a.remoteSrv
		a.mu.RUnlock()
	}

	remoteLog("starting ngrok tunnel...")
	if err := srv.StartTunnel(a.ctx, authToken); err != nil {
		remoteLog("ERROR ngrok: %v", err)
		return remote.RemoteStatus{}, err
	}
	st := srv.Status()
	remoteLog("ngrok OK: %s", st.NgrokURL)
	return st, nil
}

// DisableNgrok stops the ngrok tunnel without affecting the LAN server.
func (a *App) DisableNgrok() error {
	a.mu.RLock()
	srv := a.remoteSrv
	a.mu.RUnlock()

	if srv != nil {
		srv.StopTunnel()
	}
	return nil
}

// GetRemoteStatus returns the current remote server status.
func (a *App) GetRemoteStatus() remote.RemoteStatus {
	a.mu.RLock()
	srv := a.remoteSrv
	a.mu.RUnlock()

	if srv == nil {
		return remote.RemoteStatus{}
	}
	return srv.Status()
}

// channelStatusFunc returns a callback that reports channel statuses
// without creating a circular dependency on the channel packages.
func (a *App) channelStatusFunc() func() []remote.ChannelStatus {
	return func() []remote.ChannelStatus {
		a.mu.RLock()
		waCh := a.whatsappCh
		fsCh := a.feishuCh
		tgCh := a.telegramCh
		a.mu.RUnlock()

		var out []remote.ChannelStatus
		if waCh != nil {
			out = append(out, remote.ChannelStatus{Name: "whatsapp", Status: waCh.GetStatus()})
		}
		if fsCh != nil {
			out = append(out, remote.ChannelStatus{Name: "feishu", Status: "connected"})
		}
		if tgCh != nil {
			out = append(out, remote.ChannelStatus{Name: "telegram", Status: "connected"})
		}
		return out
	}
}

// ─── Skills RPC ───────────────────────────────────────────────────────────────

// SkillInfo is returned to the frontend to describe a loaded skill.
type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	FilePath    string `json:"filePath"`
	Source      string `json:"source"` // the skill_path entry it came from
}

// GetSkillContent returns the raw markdown content of a skill file.
func (a *App) GetSkillContent(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read skill: %w", err)
	}
	return string(data), nil
}

// GetAllSkills returns every skill across all agents plus the default ~/.clawfirm/skills/ directory.
func (a *App) GetAllSkills() []SkillInfo {
	a.mu.RLock()
	cfg := a.cfg
	a.mu.RUnlock()

	seen := make(map[string]bool)
	var out []SkillInfo

	addFromPaths := func(paths []string) {
		result := skill.Load(skill.LoadOptions{SkillPaths: paths})
		for _, s := range result.Skills {
			if seen[s.FilePath] {
				continue
			}
			seen[s.FilePath] = true
			out = append(out, SkillInfo{
				Name:        s.Name,
				Description: s.Description,
				FilePath:    s.FilePath,
			})
		}
	}

	// Always scan the default skill directory.
	addFromPaths([]string{filepath.Join(clawfirmDataDir(), "skills")})

	// Then scan each agent's configured skill_paths.
	for _, ac := range cfg.Agents {
		addFromPaths(ac.SkillPaths)
	}

	// Bundled skills as fallback (lowest priority).
	addFromPaths([]string{bundledDir("skills")})

	return out
}

// GetAgentSkills returns all currently loaded skills for the given agent.
func (a *App) GetAgentSkills(agentName string) []SkillInfo {
	a.mu.RLock()
	cfg := a.cfg
	a.mu.RUnlock()

	ac, ok := cfg.Agent(agentName)
	if !ok {
		return nil
	}
	allPaths := appendBundledSkillsDir(ac.SkillPaths)
	result := skill.Load(skill.LoadOptions{SkillPaths: allPaths})
	out := make([]SkillInfo, 0, len(result.Skills))
	for _, s := range result.Skills {
		// Map each skill back to the skill_path entry it came from
		src := ""
		for _, p := range allPaths {
			resolved := skill.ResolvePath(p, "")
			if strings.HasPrefix(s.FilePath, resolved) || s.FilePath == resolved {
				src = p
				break
			}
		}
		out = append(out, SkillInfo{
			Name:        s.Name,
			Description: s.Description,
			FilePath:    s.FilePath,
			Source:      src,
		})
	}
	return out
}

// AddSkillPath appends a skill path to the agent's config and restarts the gateway.
func (a *App) AddSkillPath(agentName, path string) error {
	a.mu.Lock()
	cfg := a.cfg
	for i, ac := range cfg.Agents {
		if ac.Name == agentName {
			// Avoid duplicates
			for _, p := range ac.SkillPaths {
				if p == path {
					a.mu.Unlock()
					return nil
				}
			}
			cfg.Agents[i].SkillPaths = append(cfg.Agents[i].SkillPaths, path)
			break
		}
	}
	a.mu.Unlock()
	if err := saveConfig(cfg); err != nil {
		return err
	}
	return a.restartGateway()
}

// RemoveSkillPath removes a skill path from the agent's config and restarts.
func (a *App) RemoveSkillPath(agentName, path string) error {
	a.mu.Lock()
	cfg := a.cfg
	for i, ac := range cfg.Agents {
		if ac.Name == agentName {
			paths := make([]string, 0, len(ac.SkillPaths))
			for _, p := range ac.SkillPaths {
				if p != path {
					paths = append(paths, p)
				}
			}
			cfg.Agents[i].SkillPaths = paths
			break
		}
	}
	a.mu.Unlock()
	if err := saveConfig(cfg); err != nil {
		return err
	}
	return a.restartGateway()
}

// RemoteSkillInfo is the frontend representation of a remote skill from the registry.
type RemoteSkillInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Downloads   int    `json:"downloads"`
}

// SearchRemoteSkills searches the skillctl remote registry.
func (a *App) SearchRemoteSkills(query string) ([]RemoteSkillInfo, error) {
	client := skillctl.NewClient()
	result, err := client.Search(a.ctx, query)
	if err != nil {
		return nil, err
	}
	out := make([]RemoteSkillInfo, len(result.Skills))
	for i, s := range result.Skills {
		out[i] = RemoteSkillInfo{
			Name:        s.Name,
			Version:     s.Version,
			Description: s.Description,
			Author:      s.Author,
			Downloads:   s.Downloads,
		}
	}
	return out, nil
}

// InstallRemoteSkill installs a skill from the remote registry and optionally syncs.
func (a *App) InstallRemoteSkill(name string, sync bool) (string, error) {
	client := skillctl.NewClient()
	result, err := client.Install(a.ctx, skillctl.InstallOptions{
		Name:  name,
		Force: true,
		Sync:  sync,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Installed %s@%s → %s", result.Name, result.Version, result.InstallDir), nil
}

// EmitChannelStatus sends a channel:status event to the frontend.
func (a *App) EmitChannelStatus(agentName, status string) {
	wailsruntime.EventsEmit(a.ctx, "channel:status", map[string]string{
		"agent":  agentName,
		"status": status,
	})
}

// ─── Memory ───────────────────────────────────────────────────────────────────

// MemoryFile is the frontend representation of an indexed Markdown file.
type MemoryFile struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	Hash      string `json:"hash"`
	ModifiedAt int64  `json:"modifiedAt"`
	IndexedAt  int64  `json:"indexedAt"`
	ChunkCount int    `json:"chunkCount"`
}

// MemorySearchResult is a single search hit returned to the frontend.
type MemorySearchResult struct {
	FilePath  string  `json:"filePath"`
	StartLine int     `json:"startLine"`
	EndLine   int     `json:"endLine"`
	Content   string  `json:"content"`
	Score     float32 `json:"score"`
}

// ListMemoryFiles returns all indexed memory files with chunk counts.
func (a *App) ListMemoryFiles() ([]MemoryFile, error) {
	if a.db == nil {
		return nil, nil
	}
	rows, err := a.db.SQL().QueryContext(a.ctx, `
		SELECT mf.path, mf.hash, mf.modified_at, mf.indexed_at,
		       COUNT(mc.id) AS chunk_count
		FROM memory_files mf
		LEFT JOIN memory_chunks mc ON mc.file_id = mf.id
		GROUP BY mf.id
		ORDER BY mf.indexed_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var files []MemoryFile
	for rows.Next() {
		var f MemoryFile
		if err := rows.Scan(&f.Path, &f.Hash, &f.ModifiedAt, &f.IndexedAt, &f.ChunkCount); err != nil {
			continue
		}
		f.Name = filepath.Base(f.Path)
		files = append(files, f)
	}
	return files, rows.Err()
}

// GetMemoryFileContent returns the raw Markdown content of a memory file.
func (a *App) GetMemoryFileContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveMemoryFileContent writes new content to a memory file.
func (a *App) SaveMemoryFileContent(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// DeleteMemoryFile removes a memory file from disk.
func (a *App) DeleteMemoryFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// CreateMemoryFile creates a new Markdown file in the memory directory and indexes it.
func (a *App) CreateMemoryFile(name string) (string, error) {
	dir := filepath.Join(clawfirmDataDir(), "memory")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	if !strings.HasSuffix(name, ".md") {
		name += ".md"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// SearchMemory is a stub — semantic search is no longer available.
func (a *App) SearchMemory(query string, limit int) ([]MemorySearchResult, error) {
	return nil, nil
}

// SyncMemory is a stub — memory indexing is no longer available.
func (a *App) SyncMemory() error {
	return nil
}

// GetMemoryDir returns the path to the memory directory.
func (a *App) GetMemoryDir() string {
	return filepath.Join(clawfirmDataDir(), "memory")
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func saveConfig(cfg *config.Config) error {
	dir := clawfirmDataDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return writeConfigYAML(filepath.Join(dir, "config.yml"), cfg)
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// buildRuntimeInfo returns a one-line runtime summary for the system prompt.
func buildRuntimeInfo(ac config.AgentConfig) string {
	host, _ := os.Hostname()
	info := "model=" + ac.Model
	if host != "" {
		info = "host=" + host + " | " + info
	}
	if ac.WorkspaceDir != "" {
		info += " | workspace=" + ac.WorkspaceDir
	}
	return info
}

func buildProviders(cfg *config.Config) (map[string]provider.LLMProvider, error) {
	return agentbuilder.BuildProviders(cfg)
}

func buildProvider(id string, pc config.ProviderConfig) (provider.LLMProvider, error) {
	return agentbuilder.BuildProvider(id, pc)
}

func providerEnvVar(id string) string {
	return agentbuilder.ProviderEnvVar(id)
}

func defaultModelForProvider(providerID string) string {
	return agentbuilder.DefaultModelForProvider(providerID)
}

// buildTools resolves a list of tool names to AgentTool instances.
func buildTools(names []string, memMgr *memory.Manager, cfg *config.Config, v *vault.Vault, mediaProvider provider.LLMProvider, agentRef ...agentbuilder.AgentRef) []tool.AgentTool {
	var vaultEnv func() map[string]string
	if v != nil {
		vaultEnv = func() map[string]string {
			m, _ := v.Env()
			return m
		}
	}
	return agentbuilder.BuildTools(names, memMgr, cfg, vaultEnv, mediaProvider, agentRef...)
}

func streamSummarize(ctx context.Context, prov provider.LLMProvider, model types.Model, msgs []types.Message) (string, error) {
	prompt := memory.BuildSummarizePrompt(msgs)
	req := provider.LLMRequest{
		Model: model,
		Messages: []types.Message{
			&types.UserMessage{
				Role: "user",
				Content: []types.ContentBlock{
					&types.TextContent{Type: types.ContentTypeText, Text: prompt},
				},
			},
		},
	}
	ch, err := prov.Stream(ctx, req)
	if err != nil {
		return "", fmt.Errorf("streamSummarize: %w", err)
	}
	var sb strings.Builder
	for ev := range ch {
		if ev.Type == types.StreamEventTextDelta {
			sb.WriteString(ev.Delta)
		}
		if ev.Error != nil {
			return "", fmt.Errorf("streamSummarize: %s", ev.Error.ErrorMessage)
		}
	}
	return sb.String(), nil
}

// ─── Cron Scheduler ──────────────────────────────────────────────────────────

// buildAgentForCron creates a fresh Agent for the named agent config.
// Reuses the same provider/model/tools/skills/system-prompt logic as startGateway.
func (a *App) buildAgentForCron(agentName string) (gateway.AgentRunner, error) {
	a.mu.RLock()
	cfg := a.cfg
	memMgr := a.memoryMgr
	a.mu.RUnlock()

	ac, ok := cfg.Agent(agentName)
	if !ok {
		return nil, fmt.Errorf("cron: agent %q not found in config", agentName)
	}

	providerMap, err := buildProviders(cfg)
	if err != nil {
		return nil, fmt.Errorf("cron: providers: %w", err)
	}
	prov, ok := providerMap[ac.Provider]
	if !ok {
		return nil, fmt.Errorf("cron: provider %q not found for agent %q", ac.Provider, agentName)
	}

	var mediaProvider provider.LLMProvider
	if cfg.Media.Provider != "" {
		mediaProvider = providerMap[cfg.Media.Provider]
	}

	maxTokens := ac.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}
	model := types.Model{ID: ac.Model, Provider: ac.Provider, MaxTokens: maxTokens}
	tools := buildTools(ac.Tools, memMgr, cfg, a.vault, mediaProvider, agentbuilder.AgentRef{Provider: prov, Model: ac.Model})

	skillResult := skill.Load(skill.LoadOptions{SkillPaths: appendBundledSkillsDir(ac.SkillPaths)})
	compacted := skill.CompactSkillPaths(skillResult.Skills)
	skillsPrompt, _, _ := skill.ApplySkillsPromptLimits(compacted)

	bootstrap := agent.LoadBootstrapContext(ac.WorkspaceDir)
	systemPrompt := agent.BuildSystemPrompt(agent.SystemPromptParams{
		WorkspaceDir:   ac.WorkspaceDir,
		SkillsPrompt:   skillsPrompt,
		ContextFiles:   bootstrap.ContextFiles,
		WorkspaceNotes: bootstrap.WorkspaceNotes,
		PromptMode:     agent.PromptModeFull,
		RuntimeInfo:    buildRuntimeInfo(ac),
		ExtraPrompt:    ac.SystemPrompt,
	})

	opts := []agent.AgentOption{
		agent.WithModel(model),
		agent.WithSystemPrompt(systemPrompt),
	}
	if len(tools) > 0 {
		opts = append(opts, agent.WithTools(tools))
	}
	return agent.NewAgent(prov, opts...), nil
}

// syncCronConfigToDB seeds the database with cron jobs from config.yml (by name, no duplicates).
func (a *App) syncCronConfigToDB() {
	a.mu.RLock()
	cfg := a.cfg
	db := a.db
	a.mu.RUnlock()

	if db == nil || len(cfg.CronJobs) == 0 {
		return
	}
	cronStore := db.CronJobs()
	for _, cj := range cfg.CronJobs {
		if _, found, err := cronStore.FindByName(cj.Name); err != nil {
			log.Printf("app: sync cron job %s: %v", cj.Name, err)
			continue
		} else if found {
			continue // already exists
		}
		job := &store.CronJob{
			ID:           uuid.New().String(),
			Name:         cj.Name,
			ScheduleKind: cj.Schedule.Kind,
			Schedule: store.ScheduleData{
				At:       cj.Schedule.At,
				EveryMs:  cj.Schedule.EveryMs,
				AnchorMs: cj.Schedule.AnchorMs,
				Expr:     cj.Schedule.Expr,
				Tz:       cj.Schedule.Tz,
			},
			AgentName: cj.AgentName,
			Prompt:    cj.Prompt,
			Enabled:   cj.Enabled,
		}
		if err := cronStore.Create(job); err != nil {
			log.Printf("app: sync cron job %s: create: %v", cj.Name, err)
		} else {
			log.Printf("app: seeded cron job %s from config", cj.Name)
		}
	}
}

// ListCronJobs returns all cron jobs.
func (a *App) ListCronJobs() ([]store.CronJob, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	return a.db.CronJobs().List()
}

// AddCronJob creates a new cron job and schedules it.
func (a *App) AddCronJob(job store.CronJob) error {
	if a.db == nil {
		return fmt.Errorf("database not available")
	}
	if job.Name == "" {
		return fmt.Errorf("job name is required")
	}
	if job.ScheduleKind == "at" && job.Schedule.At == "" {
		return fmt.Errorf("'at' schedule requires a date/time")
	}
	if job.ScheduleKind == "every" && job.Schedule.EveryMs <= 0 {
		return fmt.Errorf("'every' schedule requires a positive interval")
	}
	if job.ScheduleKind == "cron" && job.Schedule.Expr == "" {
		return fmt.Errorf("'cron' schedule requires an expression")
	}
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	if a.cronScheduler != nil {
		return a.cronScheduler.AddJob(&job)
	}
	return a.db.CronJobs().Create(&job)
}

// UpdateCronJob updates an existing cron job and reschedules it.
func (a *App) UpdateCronJob(job store.CronJob) error {
	if a.db == nil {
		return fmt.Errorf("database not available")
	}
	if a.cronScheduler != nil {
		return a.cronScheduler.UpdateJob(&job)
	}
	return a.db.CronJobs().Update(&job)
}

// DeleteCronJob removes a cron job.
func (a *App) DeleteCronJob(jobID string) error {
	if a.db == nil {
		return fmt.Errorf("database not available")
	}
	if a.cronScheduler != nil {
		return a.cronScheduler.RemoveJob(jobID)
	}
	return a.db.CronJobs().Delete(jobID)
}

// ToggleCronJob enables or disables a cron job.
func (a *App) ToggleCronJob(jobID string, enabled bool) error {
	if a.db == nil {
		return fmt.Errorf("database not available")
	}
	if a.cronScheduler != nil {
		return a.cronScheduler.ToggleJob(jobID, enabled)
	}
	return a.db.CronJobs().SetEnabled(jobID, enabled)
}

// GetCronJobHistory returns execution history for a specific job.
func (a *App) GetCronJobHistory(jobID string, limit int) ([]store.CronJobHistory, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	return a.db.CronJobs().ListHistory(jobID, limit)
}

// GetCronJobHistoryAll returns execution history across all jobs.
func (a *App) GetCronJobHistoryAll(limit int) ([]store.CronJobHistory, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	return a.db.CronJobs().ListHistoryAll(limit)
}

// TriggerCronJob runs a job immediately, regardless of its schedule.
func (a *App) TriggerCronJob(jobID string) error {
	a.mu.RLock()
	sched := a.cronScheduler
	a.mu.RUnlock()
	if sched == nil {
		return fmt.Errorf("cron scheduler not running")
	}
	return sched.TriggerNow(jobID)
}

// ─── WhipFlow file browsing ───────────────────────────────────────────────────

// scanAllWhipFiles recursively walks dir and returns all .whip file paths,
// skipping .bak files.
func scanAllWhipFiles(dir string) []string {
	var out []string
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if filepath.Ext(d.Name()) == ".whip" {
			out = append(out, path)
		}
		return nil
	})
	return out
}

// ListWhipFiles returns all .whip files from user dir and bundled dir.
// User files take priority (same relative path shadows bundled).
func (a *App) ListWhipFiles() ([]string, error) {
	userDir := filepath.Join(clawfirmDataDir(), "workflows")
	bDir := bundledDir("workflows")

	seen := make(map[string]bool) // relative path → already added
	var files []string

	addFrom := func(baseDir string) {
		_ = filepath.WalkDir(baseDir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() {
				return nil
			}
			if filepath.Ext(d.Name()) != ".whip" {
				return nil
			}
			rel, _ := filepath.Rel(baseDir, path)
			if !seen[rel] {
				seen[rel] = true
				files = append(files, path)
			}
			return nil
		})
	}

	addFrom(userDir)  // user first
	addFrom(bDir)     // bundled fallback

	return files, nil
}

// GetWhipFileContent returns the content of a .whip file.
func (a *App) GetWhipFileContent(path string) (string, error) {
	userDir := filepath.Join(clawfirmDataDir(), "workflows")
	bDir := bundledDir("workflows")
	// security: only allow files inside user or bundled workflows dir
	abs, err := filepath.Abs(path)
	if err != nil || (!strings.HasPrefix(abs, userDir) && !strings.HasPrefix(abs, bDir)) {
		return "", fmt.Errorf("access denied")
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// extractText returns a plain-text preview of a message for history display.
func extractText(m types.Message) string {
	switch msg := m.(type) {
	case *types.UserMessage:
		for _, b := range msg.Content {
			if t, ok := b.(*types.TextContent); ok {
				return t.Text
			}
		}
	case *types.AssistantMessage:
		for _, b := range msg.Content {
			if t, ok := b.(*types.TextContent); ok {
				return t.Text
			}
		}
	}
	return ""
}

// ─── Browser (CDP) ──────────────────────────────────────────────────────────

// BrowserStatus holds the CDP connection status returned to the frontend.
type BrowserStatus struct {
	Connected bool   `json:"connected"`
	CDPURL    string `json:"cdpURL"`
	Browser   string `json:"browser"` // e.g. "Chrome/126.0.6478.63"
	Error     string `json:"error"`
}

// BrowserTestCDP probes localhost:9222 for an active Chrome CDP endpoint.
func (a *App) BrowserTestCDP() BrowserStatus {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:9222/json/version")
	if err != nil {
		return BrowserStatus{Connected: false, Error: "CDP not reachable: " + err.Error()}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var info struct {
		Browser      string `json:"Browser"`
		WebSocketURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return BrowserStatus{Connected: false, Error: "invalid CDP response"}
	}
	return BrowserStatus{
		Connected: true,
		CDPURL:    info.WebSocketURL,
		Browser:   info.Browser,
	}
}

// BrowserLaunchChrome launches Chrome with --remote-debugging-port=9222 using a dedicated
// user-data-dir so it can coexist with a running Chrome instance.
func (a *App) BrowserLaunchChrome() BrowserStatus {
	// If CDP is already reachable, return immediately.
	if status := a.BrowserTestCDP(); status.Connected {
		return status
	}

	// Find Chrome binary.
	chromePath := findChromeBinary()
	if chromePath == "" {
		return BrowserStatus{Connected: false, Error: "Chrome not found on this system"}
	}

	// Use the social-cli chrome profile which preserves user logins/cookies.
	// Falls back to ~/.clawfirm/chrome-profile if the social-cli profile doesn't exist.
	profileDir := filepath.Join(os.Getenv("HOME"), ".social-cli", "chrome-profile")
	if _, err := os.Stat(profileDir); os.IsNotExist(err) {
		profileDir = filepath.Join(os.Getenv("HOME"), ".clawfirm", "chrome-profile")
	}

	cmd := exec.Command(chromePath,
		"--remote-debugging-port=9222",
		"--user-data-dir="+profileDir,
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-blink-features=AutomationControlled",
	)
	if err := cmd.Start(); err != nil {
		return BrowserStatus{Connected: false, Error: "failed to launch Chrome: " + err.Error()}
	}
	// Detach — don't wait for Chrome to exit.
	go func() { _ = cmd.Wait() }()

	// Poll for CDP readiness (up to 5 seconds).
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		status := a.BrowserTestCDP()
		if status.Connected {
			return status
		}
	}
	return BrowserStatus{Connected: false, Error: "Chrome launched but CDP not ready after 5s"}
}

// findChromeBinary returns the path to a Chrome/Chromium binary, or "" if not found.
func findChromeBinary() string {
	switch runtime.GOOS {
	case "darwin":
		candidates := []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	case "linux":
		for _, name := range []string{"google-chrome", "chromium-browser", "chromium"} {
			if p, err := exec.LookPath(name); err == nil {
				return p
			}
		}
	default: // windows
		candidates := []string{
			filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	}
	return ""
}

// ─── Browser Shortcuts (YAML adapters) ──────────────────────────────────────

// ShortcutInfo describes a browser automation shortcut available to the frontend.
type ShortcutInfo struct {
	Platform string   `json:"platform"`
	File     string   `json:"file"`
	Commands []string `json:"commands"`
}

// BrowserListShortcuts returns all YAML adapter shortcuts.
// User shortcuts take priority over bundled ones with the same filename.
func (a *App) BrowserListShortcuts() []ShortcutInfo {
	userDir := filepath.Join(clawfirmDataDir(), "shortcuts")
	bDir := bundledDir("shortcuts")

	seen := make(map[string]bool)
	var out []ShortcutInfo

	addFrom := func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") || seen[e.Name()] {
				continue
			}
			seen[e.Name()] = true
			fp := filepath.Join(dir, e.Name())
			adapter, err := browser.LoadAdapterYAML(fp)
			if err != nil {
				continue
			}
			cmds := make([]string, 0, len(adapter.Commands))
			for k := range adapter.Commands {
				cmds = append(cmds, k)
			}
			out = append(out, ShortcutInfo{
				Platform: adapter.Platform,
				File:     e.Name(),
				Commands: cmds,
			})
		}
	}

	addFrom(userDir)
	addFrom(bDir)
	return out
}

// BrowserRunShortcut executes a YAML adapter command and returns the result rows.
func (a *App) BrowserRunShortcut(file, command string, args []string) ([]map[string]any, error) {
	fp := filepath.Join(clawfirmDataDir(), "shortcuts", file)
	if _, err := os.Stat(fp); err != nil {
		// Fallback to bundled shortcuts.
		fp = filepath.Join(bundledDir("shortcuts"), file)
		if _, err := os.Stat(fp); err != nil {
			return nil, fmt.Errorf("shortcut file not found: %s", file)
		}
	}
	// Load adapter to detect platform and determine target type.
	adapter, err := browser.LoadAdapterYAML(fp)
	if err != nil {
		return nil, err
	}
	targetType := "chrome"
	if strings.HasPrefix(adapter.Platform, "clawfirm") || adapter.Platform == "clawfirm_test" {
		targetType = "app"
	}
	return browser.RunYAMLCommand(context.Background(), fp, command, args, 9222, nil, nil, targetType)
}
