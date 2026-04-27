# Bundled Assets Layered Lookup

## Goal
- Bundled assets → `~/.clawfirm/bundled/{skills,workflows,shortcuts}/` (always overwrite)
- User assets → `~/.clawfirm/{skills,workflows,shortcuts}/` (user-managed)
- Lookup: user first, bundled fallback

## Changes
1. `app/app.go`: initUserDirs, extraction target, remove skip-if-exists, migration, ListWhipFiles, GetWhipFileContent, BrowserListShortcuts, BrowserRunShortcut, GetAllSkills, GetAgentSkills, startGateway
2. `internal/agentbuilder/builder.go`: append bundled skills dir
3. `tool/builtin/browser_shortcut.go`: layered shortcut resolution
4. `cmd/gateway/main.go`: append bundled skills dir
5. `cmd/browser-shortcut/main.go`: layered shortcut resolution
