package builtin

// ConcurrencySafe implementations for all builtin tools.
// Safe tools (read-only, no side effects) return true.
// Unsafe tools (writes, commands, external mutations) return false.

// --- Safe tools ---

func (r *Read) ConcurrencySafe() bool           { return true }
func (g *Grep) ConcurrencySafe() bool            { return true }
func (f *Find) ConcurrencySafe() bool            { return true }
func (l *Ls) ConcurrencySafe() bool              { return true }
func (f *Fetch) ConcurrencySafe() bool           { return true }
func (t *GetCurrentTime) ConcurrencySafe() bool  { return true }
func (s *SessionsList) ConcurrencySafe() bool    { return true }
func (e *Echo) ConcurrencySafe() bool            { return true }
func (n *Noop) ConcurrencySafe() bool            { return true }
func (w *WebSearch) ConcurrencySafe() bool        { return true }
func (s *SubAgent) ConcurrencySafe() bool         { return true }
func (m *MediaUnderstand) ConcurrencySafe() bool  { return true }
func (m *MediaGen) ConcurrencySafe() bool         { return true }

// --- Unsafe tools ---

func (w *Write) ConcurrencySafe() bool       { return false }
func (e *Edit) ConcurrencySafe() bool        { return false }
func (a *ApplyPatch) ConcurrencySafe() bool  { return false }
func (b *Bash) ConcurrencySafe() bool        { return false }
func (e *Exec) ConcurrencySafe() bool        { return false }
func (p *Process) ConcurrencySafe() bool     { return false }
func (w *WhipflowRun) ConcurrencySafe() bool { return false }
func (s *Skill) ConcurrencySafe() bool       { return false }
func (a *AskUser) ConcurrencySafe() bool     { return false }

// ShouldDefer — core tools always sent to LLM, low-frequency tools deferred.
// Deferred tools are excluded from LLM tool schemas until activated via tool_search.

// --- Core tools (not deferred) ---

func (r *Read) ShouldDefer() bool          { return false }
func (w *Write) ShouldDefer() bool         { return false }
func (e *Edit) ShouldDefer() bool          { return false }
func (g *Grep) ShouldDefer() bool          { return false }
func (f *Find) ShouldDefer() bool          { return false }
func (l *Ls) ShouldDefer() bool            { return false }
func (b *Bash) ShouldDefer() bool          { return false }
func (e *Exec) ShouldDefer() bool          { return false }
func (s *Skill) ShouldDefer() bool         { return false }

// --- Deferred tools (low-frequency, activated via tool_search) ---

func (a *ApplyPatch) ShouldDefer() bool    { return true }
func (f *Fetch) ShouldDefer() bool         { return true }
func (t *GetCurrentTime) ShouldDefer() bool { return true }
func (p *Process) ShouldDefer() bool       { return true }
func (s *SessionsList) ShouldDefer() bool  { return true }
func (w *WhipflowRun) ShouldDefer() bool   { return true }
func (e *Echo) ShouldDefer() bool          { return true }
func (n *Noop) ShouldDefer() bool          { return true }
func (w *WebSearch) ShouldDefer() bool     { return true }
func (a *AskUser) ShouldDefer() bool       { return true }
func (s *SubAgent) ShouldDefer() bool      { return true }
func (m *MediaUnderstand) ShouldDefer() bool { return true }
func (m *MediaGen) ShouldDefer() bool        { return true }
