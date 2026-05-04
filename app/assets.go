package app

import (
	"embed"
	_ "embed"
)

// embeddedFunc holds the bundled `func` binary.
// In production builds (make app), the Makefile replaces app/assets/func
// with a real darwin/universal binary before compilation.
// During `wails dev`, this contains a small placeholder — no binary is extracted.
//
//go:embed assets/func
var embeddedFunc []byte

// embeddedClaw holds the bundled `claw` binary (claw-code Rust CLI).
// In production builds (make app), the Makefile copies the release binary here.
// During `wails dev`, this contains a small placeholder — findClawBinary falls
// back to the build tree or PATH.
//
//go:embed assets/claw
var embeddedClaw []byte

//go:embed assets/skills
var embeddedSkills embed.FS

//go:embed assets/workflows
var embeddedWorkflows embed.FS

//go:embed assets/shortcuts
var embeddedShortcuts embed.FS

// embeddedBrowserShortcut holds the bundled `browser-shortcut` binary.
// In production builds (make app), the Makefile copies the release binary here.
// During `wails dev`, this contains a small placeholder — the plugin is not registered.
//
//go:embed assets/browser-shortcut
var embeddedBrowserShortcut []byte

// embeddedWhip holds the bundled `whip` binary (WhipFlow workflow runner).
// In production builds (make app), the Makefile replaces app/assets/whip
// with a real darwin/universal binary before compilation.
// During `wails dev`, this contains a small placeholder — falls back to PATH.
//
//go:embed assets/whip
var embeddedWhip []byte
