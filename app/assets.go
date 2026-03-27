package app

import _ "embed"

// embeddedFunc holds the bundled `func` binary.
// In production builds (make app), the Makefile replaces app/assets/func
// with a real darwin/universal binary before compilation.
// During `wails dev`, this contains a small placeholder — no binary is extracted.
//
//go:embed assets/func
var embeddedFunc []byte
