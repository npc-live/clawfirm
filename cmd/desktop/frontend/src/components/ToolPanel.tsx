import { useState, useEffect, useRef } from "react";

export interface ToolExecution {
  id: string;
  name: string;
  args?: any;
  status: "running" | "done" | "error" | "interrupted";
  result?: any;
  partialResult?: any;
  startTime: number;
  endTime?: number;
}

// ---------------------------------------------------------------------------
// WhipFlow preview types (returned by mode="auto" / "preview")
// ---------------------------------------------------------------------------

interface WhipflowStepPreview {
  index: number;
  name?: string;
  agent?: string;
  prompt: string;
}

interface WhipflowComplexityAnalysis {
  tier: "simple" | "medium" | "complex";
  session_count: number;
  has_parallel: boolean;
  has_loops: boolean;
  has_choice: boolean;
  has_ask: boolean;
  steps: WhipflowStepPreview[];
}

export interface WhipflowPreview {
  type: "whipflow_preview";
  analysis: WhipflowComplexityAnalysis;
  source: string;
}

export function isWhipflowPreview(v: any): v is WhipflowPreview {
  return v !== null && typeof v === "object" && v.type === "whipflow_preview" && v.analysis;
}

interface Props {
  executions: ToolExecution[];
  onCommand: (command: string) => void;
  onEditPreview?: (source: string) => void;
}

// ---------------------------------------------------------------------------
// WhipFlow session step types
// ---------------------------------------------------------------------------

interface WhipflowSessionStep {
  index: number;
  name?: string;
  provider?: string;
  prompt: string;
  done: boolean;
  output?: string;
  duration_ms?: number;
  error?: string;
  stream_text?: string;
  has_history?: boolean; // message history was saved to DB — can continue conversation
  messages?: any[]; // conversation turns from NativeProvider sessions
}

function isWhipflowStep(v: any): v is WhipflowSessionStep {
  return v !== null && typeof v === "object" && typeof v.index === "number";
}

// Accumulate session steps from a stream of partial_result updates.
// Each update is a single WhipflowSessionStep; we merge them into a map
// keyed by index so start (done=false) and end (done=true) merge cleanly.
// stream_text deltas are appended (not replaced) to build the full text.
function mergeSessionSteps(partial: any): Map<number, WhipflowSessionStep> {
  const map = new Map<number, WhipflowSessionStep>();
  if (!partial) return map;
  // partial may be the latest single update or an array of all updates
  const items: any[] = Array.isArray(partial) ? partial : [partial];
  for (const item of items) {
    if (!isWhipflowStep(item)) continue;
    const existing = map.get(item.index);
    if (existing) {
      // Append stream_text delta to accumulated text
      if (item.stream_text) {
        const merged = { ...existing };
        merged.stream_text = (existing.stream_text || "") + item.stream_text;
        // Also apply any other fields from the update (e.g. done, output)
        if (item.done) { merged.done = true; merged.output = item.output; merged.duration_ms = item.duration_ms; merged.error = item.error; }
        map.set(item.index, merged);
      } else {
        map.set(item.index, { ...existing, ...item });
      }
    } else {
      map.set(item.index, item);
    }
  }
  return map;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// Try to parse a tool result string as JSON; return the parsed object or the original value.
function tryParseJSON(v: any): any {
  if (typeof v !== "string") return v;
  try { return JSON.parse(v); } catch { return v; }
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function CollapsibleJSON({ label, data }: { label: string; data: any }) {
  const [open, setOpen] = useState(false);
  if (data === undefined || data === null) return null;

  const text = typeof data === "string" ? data : JSON.stringify(data, null, 2);

  return (
    <div className="mt-1.5">
      <button
        onClick={() => setOpen(!open)}
        className="text-[11px] text-[rgba(61,57,41,0.35)] hover:text-[rgba(61,57,41,0.55)] flex items-center gap-1 transition-colors"
      >
        <span className={`transition-transform ${open ? "rotate-90" : ""}`}>▶</span>
        {label}
      </button>
      {open && (
        <pre className="mt-1 text-[11px] text-[rgba(61,57,41,0.6)] bg-[rgba(61,57,41,0.04)] border border-[rgba(61,57,41,0.08)] rounded-lg p-2 overflow-x-auto max-h-48 overflow-y-auto whitespace-pre-wrap break-all font-mono">
          {text}
        </pre>
      )}
    </div>
  );
}

function StatusIndicator({ status }: { status: ToolExecution["status"] }) {
  if (status === "running") {
    return <span className="inline-block w-2 h-2 rounded-full bg-[#c85a2a] animate-pulse" />;
  }
  if (status === "done") {
    return <span className="text-emerald-400 text-[13px]">&#10003;</span>;
  }
  if (status === "interrupted") {
    return <span className="text-amber-400 text-[13px]">&#9888;</span>;
  }
  return <span className="text-red-400 text-[13px]">&#10007;</span>;
}

function Spinner() {
  return (
    <svg className="animate-spin w-4 h-4 text-[#c85a2a]" viewBox="0 0 24 24" fill="none">
      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="3" />
      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
    </svg>
  );
}

function formatDuration(start: number, end?: number): string {
  const ms = (end ?? Date.now()) - start;
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

// ---------------------------------------------------------------------------
// WhipFlow session steps panel
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// WhipFlow source viewer
// ---------------------------------------------------------------------------

function WhipflowSource({ exec }: { exec: ToolExecution }) {
  const [open, setOpen] = useState(false);
  const args = exec.args as { source?: string; file?: string } | undefined;
  if (!args) return null;

  const label = args.file ? `source: ${args.file}` : "source";
  const content = args.source ?? null;

  return (
    <div className="mt-1.5">
      <button
        onClick={() => setOpen(!open)}
        className="text-[11px] text-[rgba(61,57,41,0.35)] hover:text-[rgba(61,57,41,0.55)] flex items-center gap-1 transition-colors"
      >
        <span className={`transition-transform ${open ? "rotate-90" : ""}`}>▶</span>
        {label}
      </button>
      {open && content && (
        <pre className="mt-1 text-[11px] text-[rgba(61,57,41,0.6)] bg-[rgba(61,57,41,0.04)] border border-[rgba(61,57,41,0.08)] rounded-lg p-2 overflow-x-auto max-h-64 overflow-y-auto whitespace-pre-wrap break-all font-mono leading-relaxed">
          {content}
        </pre>
      )}
      {open && !content && args.file && (
        <p className="mt-1 text-[11px] text-[rgba(61,57,41,0.3)] font-mono">{args.file}</p>
      )}
    </div>
  );
}

function WhipflowSessionSteps({ exec, onCommand }: { exec: ToolExecution; onCommand: (cmd: string) => void }) {
  // Collect all session step updates from partialResult (latest snapshot).
  // We accumulate updates in ChatView as an array in partialResult.
  const stepsMap = mergeSessionSteps(exec.partialResult);
  const steps = Array.from(stepsMap.values()).sort((a, b) => a.index - b.index);

  if (steps.length === 0) return null;

  const doneCount = steps.filter((s) => s.done && !s.error).length;
  const errorCount = steps.filter((s) => s.done && !!s.error).length;
  const runningCount = steps.filter((s) => !s.done).length;

  return (
    <div className="mt-2 space-y-1.5">
      {/* Summary row */}
      <div className="flex items-center gap-2 text-[11px] text-[rgba(61,57,41,0.4)]">
        <span className="font-mono">{doneCount + errorCount}/{steps.length} sessions</span>
        {runningCount > 0 && <span className="text-[#c85a2a] animate-pulse">· {runningCount} running</span>}
        {errorCount > 0 && <span className="text-red-400">· {errorCount} failed</span>}
      </div>
      {/* Per-session cards */}
      {steps.map((step) => (
        <SessionStepCard
          key={step.index}
          step={step}
          onRetry={() => onCommand(`Retry session ${step.index}, keep results of sessions 0-${step.index - 1}.`)}
          onContinue={step.has_history ? () => onCommand(`Open session ${step.index} conversation for follow-up.`) : undefined}
        />
      ))}
    </div>
  );
}

// Renders conversation messages from a NativeProvider session.
function SessionMessages({ messages }: { messages: any[] }) {
  const [open, setOpen] = useState(false);

  // Extract displayable turns: user/assistant text + tool use/result.
  const turns: { role: string; text: string; isToolUse?: boolean; isToolResult?: boolean; toolName?: string }[] = [];
  for (const msg of messages) {
    const role = msg.role ?? "unknown";
    const content = msg.content;
    if (typeof content === "string") {
      turns.push({ role, text: content });
    } else if (Array.isArray(content)) {
      for (const block of content) {
        if (block.type === "text" && block.text) {
          turns.push({ role, text: block.text });
        } else if (block.type === "tool_use") {
          turns.push({ role, text: JSON.stringify(block.input ?? {}, null, 2), isToolUse: true, toolName: block.name });
        } else if (block.type === "tool_result") {
          const resultText = Array.isArray(block.content)
            ? block.content.map((c: any) => c.text ?? "").join("")
            : String(block.content ?? "");
          turns.push({ role, text: resultText, isToolResult: true });
        }
      }
    }
  }

  if (turns.length === 0) return null;

  return (
    <div className="mt-1">
      <button
        onClick={() => setOpen((o) => !o)}
        className="text-[10px] text-[rgba(61,57,41,0.35)] hover:text-[rgba(61,57,41,0.55)] flex items-center gap-1 transition-colors"
      >
        <span className={`transition-transform ${open ? "rotate-90" : ""}`}>▶</span>
        Conversation · {turns.filter(t => !t.isToolUse && !t.isToolResult).length} messages
        {turns.some(t => t.isToolUse) && <span className="text-[rgba(200,90,42,0.5)]"> · {turns.filter(t => t.isToolUse).length} tool calls</span>}
      </button>
      {open && (
        <div className="mt-1.5 space-y-1.5 max-h-64 overflow-y-auto">
          {turns.map((turn, i) => (
            <div key={i} className={`rounded-md px-2.5 py-1.5 text-[11px] ${
              turn.isToolUse
                ? "bg-[rgba(200,90,42,0.08)] border border-[rgba(200,90,42,0.15)]"
                : turn.isToolResult
                ? "bg-[rgba(61,57,41,0.04)] border border-[rgba(61,57,41,0.08)]"
                : turn.role === "user"
                ? "bg-[rgba(200,90,42,0.06)] border border-[rgba(200,90,42,0.1)] ml-4"
                : "bg-[rgba(61,57,41,0.05)] border border-[rgba(61,57,41,0.08)]"
            }`}>
              <div className="flex items-center gap-1.5 mb-0.5">
                {turn.isToolUse ? (
                  <span className="text-[9px] font-mono text-[rgba(200,90,42,0.7)] uppercase tracking-wider">⚙ {turn.toolName}</span>
                ) : turn.isToolResult ? (
                  <span className="text-[9px] font-mono text-[rgba(61,57,41,0.4)] uppercase tracking-wider">↩ result</span>
                ) : (
                  <span className="text-[9px] font-mono text-[rgba(61,57,41,0.4)] uppercase tracking-wider">{turn.role}</span>
                )}
              </div>
              <pre className="whitespace-pre-wrap break-words font-mono text-[rgba(61,57,41,0.7)] leading-relaxed line-clamp-4">
                {turn.text.length > 300 ? turn.text.slice(0, 300) + "…" : turn.text}
              </pre>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function SessionStepCard({ step, onRetry, onContinue }: { step: WhipflowSessionStep; onRetry?: () => void; onContinue?: () => void }) {
  const [open, setOpen] = useState(false);
  const [elapsed, setElapsed] = useState(0);
  const startRef = useRef(Date.now());
  const lastStreamRef = useRef(Date.now());
  const isRunning = !step.done;
  const isError = step.done && !!step.error;
  const isDone = step.done && !step.error;

  // Update lastStream timestamp whenever stream_text changes
  useEffect(() => {
    if (step.stream_text) lastStreamRef.current = Date.now();
  }, [step.stream_text]);

  // Elapsed timer — ticks every second while running
  useEffect(() => {
    if (!isRunning) return;
    startRef.current = Date.now() - elapsed * 1000;
    const id = setInterval(() => {
      setElapsed(Math.floor((Date.now() - startRef.current) / 1000));
    }, 1000);
    return () => clearInterval(id);
  }, [isRunning]); // eslint-disable-line react-hooks/exhaustive-deps

  const silentSecs = isRunning ? Math.floor((Date.now() - lastStreamRef.current) / 1000) : 0;
  const isStuck = isRunning && silentSecs >= 30;

  const borderCls = isRunning
    ? "border-[rgba(200,90,42,0.3)] bg-[rgba(200,90,42,0.04)]"
    : isError
    ? "border-[rgba(239,68,68,0.3)] bg-[rgba(239,68,68,0.04)]"
    : "border-[rgba(61,57,41,0.08)] bg-[rgba(61,57,41,0.04)]";

  const label = step.name
    ? `#${step.index + 1} · ${step.name}`
    : `Session ${step.index + 1}`;

  return (
    <div className={`rounded-lg border overflow-hidden ${borderCls}`}>
      <button
        className="w-full flex items-center gap-2 px-3 py-2 text-left hover:bg-[rgba(61,57,41,0.04)] transition-colors"
        onClick={() => setOpen((o) => !o)}
      >
        {/* Status dot */}
        {isRunning ? (
          <span className="w-2 h-2 rounded-full bg-[#c85a2a] animate-pulse flex-shrink-0" />
        ) : isDone ? (
          <span className="text-emerald-400 text-[12px] flex-shrink-0">✓</span>
        ) : (
          <span className="text-red-400 text-[12px] flex-shrink-0">✗</span>
        )}
        {/* Label */}
        <span className="flex-1 text-[12px] text-[rgba(61,57,41,0.8)] font-medium truncate">
          {label}
        </span>
        {/* Provider badge */}
        {step.provider && (
          <span className="text-[10px] font-mono text-[rgba(61,57,41,0.3)] flex-shrink-0">
            {step.provider}
          </span>
        )}
        {/* Elapsed / Duration */}
        {isRunning ? (
          <span className={`text-[10px] font-mono flex-shrink-0 ${isStuck ? "text-amber-400" : "text-[rgba(61,57,41,0.3)]"}`}>
            {isStuck ? `⚠ ${elapsed}s` : `${elapsed}s`}
          </span>
        ) : step.duration_ms != null && step.duration_ms > 0 ? (
          <span className="text-[10px] text-[rgba(61,57,41,0.2)] font-mono flex-shrink-0">
            {step.duration_ms < 1000 ? `${step.duration_ms}ms` : `${(step.duration_ms / 1000).toFixed(1)}s`}
          </span>
        ) : null}
        {/* Continue conversation button */}
        {isDone && onContinue && (
          <button
            onClick={(e) => { e.stopPropagation(); onContinue(); }}
            className="text-[10px] px-1.5 py-0.5 rounded bg-[rgba(61,57,41,0.12)] text-[rgba(61,57,41,0.6)] hover:bg-[rgba(61,57,41,0.2)] transition-colors flex-shrink-0"
            title="Open this session's conversation to continue"
          >
            Continue
          </button>
        )}
        {/* Retry button */}
        {(isDone || isError) && onRetry && (
          <button
            onClick={(e) => { e.stopPropagation(); onRetry(); }}
            className="text-[10px] px-1.5 py-0.5 rounded bg-[rgba(200,90,42,0.15)] text-[rgba(200,90,42,0.8)] hover:bg-[rgba(200,90,42,0.25)] transition-colors flex-shrink-0"
          >
            Retry from here
          </button>
        )}
        {/* Expand toggle */}
        <span className={`text-[rgba(61,57,41,0.3)] text-[10px] flex-shrink-0 transition-transform ${open ? "rotate-180" : ""}`}>▾</span>
      </button>

      {/* Streaming text preview (collapsed, while running) */}
      {!open && isRunning && (
        <div className="px-3 pb-2 space-y-1">
          {isStuck && (
            <p className="text-[11px] text-amber-400">
              ⚠ No updates for {silentSecs}s — may be stuck
            </p>
          )}
          {step.stream_text && (
            <pre className="text-[11px] text-[rgba(61,57,41,0.5)] bg-[rgba(61,57,41,0.03)] rounded p-1.5 whitespace-pre-wrap break-words max-h-16 overflow-hidden font-mono leading-relaxed line-clamp-3">
              {step.stream_text.length > 200 ? "…" + step.stream_text.slice(-200) : step.stream_text}
            </pre>
          )}
        </div>
      )}

      {open && (
        <div className="px-3 pb-3 pt-1 space-y-2 border-t border-[rgba(61,57,41,0.08)]">
          {/* Prompt */}
          {step.prompt && (
          <div>
            <p className="text-[10px] font-semibold text-[rgba(61,57,41,0.3)] uppercase tracking-wider mb-1">Prompt</p>
            <pre className="text-[11px] text-[rgba(61,57,41,0.55)] bg-[rgba(61,57,41,0.04)] rounded-lg p-2 whitespace-pre-wrap break-words max-h-40 overflow-y-auto font-mono leading-relaxed">
              {step.prompt}
            </pre>
          </div>
          )}
          {/* Streaming text (while running) */}
          {isRunning && step.stream_text && (
            <div>
              <p className="text-[10px] font-semibold text-[rgba(200,90,42,0.5)] uppercase tracking-wider mb-1">Generating…</p>
              <pre className="text-[11px] text-[rgba(61,57,41,0.6)] bg-[rgba(61,57,41,0.04)] rounded-lg p-2 whitespace-pre-wrap break-words max-h-48 overflow-y-auto font-mono leading-relaxed">
                {step.stream_text}
              </pre>
            </div>
          )}
          {/* Output */}
          {step.output && (
            <div>
              <p className="text-[10px] font-semibold text-[rgba(61,57,41,0.3)] uppercase tracking-wider mb-1">Output</p>
              <pre className="text-[11px] text-[rgba(61,57,41,0.7)] bg-[rgba(61,57,41,0.04)] rounded-lg p-2 whitespace-pre-wrap break-words max-h-48 overflow-y-auto font-mono leading-relaxed">
                {step.output}
              </pre>
            </div>
          )}
          {/* Error */}
          {step.error && (
            <div>
              <p className="text-[10px] font-semibold text-red-400 uppercase tracking-wider mb-1">Error</p>
              <pre className="text-[11px] text-red-300 bg-[rgba(239,68,68,0.05)] rounded-lg p-2 whitespace-pre-wrap break-words font-mono">
                {step.error}
              </pre>
            </div>
          )}
          {/* Conversation messages (NativeProvider sessions) */}
          {step.messages && step.messages.length > 0 && (
            <SessionMessages messages={step.messages} />
          )}
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// WhipFlow step preview card (shown for auto/preview mode results)
// ---------------------------------------------------------------------------

function StepPreviewCard({
  preview,
  onCommand,
  onEdit,
}: {
  preview: WhipflowPreview;
  onCommand: (cmd: string) => void;
  onEdit?: () => void;
}) {
  const [showSource, setShowSource] = useState(false);
  const [confirmed, setConfirmed] = useState(false);
  const { analysis } = preview;

  const tierLabel =
    analysis.tier === "simple" ? "Simple" :
    analysis.tier === "medium" ? "Multi-step" : "Complex";

  const tierColor =
    analysis.tier === "simple" ? "text-emerald-500" :
    analysis.tier === "medium" ? "text-amber-500" : "text-red-400";

  return (
    <div className="mt-2 rounded-xl border border-[rgba(200,90,42,0.2)] bg-[rgba(200,90,42,0.04)] overflow-hidden">
      {/* Header */}
      <div className="px-3 py-2.5 flex items-center gap-2 border-b border-[rgba(200,90,42,0.1)]">
        <span className={`text-[11px] font-semibold ${tierColor}`}>{tierLabel}</span>
        <span className="text-[10px] text-[rgba(61,57,41,0.4)] font-mono">
          {analysis.session_count} session{analysis.session_count !== 1 ? "s" : ""}
          {analysis.has_parallel && " · parallel"}
          {analysis.has_loops && " · loops"}
          {analysis.has_choice && " · choice"}
          {analysis.has_ask && " · user input"}
        </span>
        <div className="flex-1" />
        {!confirmed && (
          <button
            onClick={() => { setConfirmed(true); onCommand("Execute this workflow now."); }}
            className="px-2.5 py-1 rounded-lg text-[11px] font-medium bg-[#c85a2a] text-white hover:bg-[#a84a22] transition-colors"
          >
            Run
          </button>
        )}
        {confirmed && (
          <span className="flex items-center gap-1 text-[11px] text-[rgba(200,90,42,0.7)]">
            <svg className="animate-spin w-3 h-3" viewBox="0 0 24 24" fill="none">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="3" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
            </svg>
            Running…
          </span>
        )}
        {!confirmed && (
          <button
            onClick={() => { setConfirmed(true); onCommand("Execute session 0 only (step-by-step mode)."); }}
            className="px-2.5 py-1 rounded-lg text-[11px] font-medium bg-[rgba(61,57,41,0.1)] text-[rgba(61,57,41,0.5)] hover:bg-[rgba(61,57,41,0.18)] transition-colors"
            title="Execute one session at a time"
          >
            Next Step
          </button>
        )}
        {onEdit && !confirmed && (
          <button
            onClick={onEdit}
            className="px-2.5 py-1 rounded-lg text-[11px] font-medium bg-[rgba(61,57,41,0.1)] text-[rgba(61,57,41,0.5)] hover:bg-[rgba(61,57,41,0.15)] transition-colors"
          >
            Edit
          </button>
        )}
        <button
          onClick={() => setShowSource((s) => !s)}
          className="px-2.5 py-1 rounded-lg text-[11px] font-medium bg-[rgba(61,57,41,0.04)] text-[rgba(61,57,41,0.35)] hover:bg-[rgba(61,57,41,0.08)] transition-colors"
        >
          {showSource ? "Steps" : "Code"}
        </button>
      </div>

      {showSource ? (
        /* Source code view */
        <pre className="px-3 py-2 text-[11px] font-mono text-[rgba(61,57,41,0.7)] whitespace-pre-wrap break-words max-h-64 overflow-y-auto leading-relaxed">
          {preview.source}
        </pre>
      ) : (
        /* Step list */
        <div className="px-3 py-2 space-y-0.5">
          {(analysis.steps ?? []).map((step) => (
            <div key={step.index} className="flex items-center gap-2 py-1 group">
              <span className="text-[11px] font-mono text-[rgba(200,90,42,0.6)] flex-shrink-0 w-5 text-right">
                {step.index + 1}.
              </span>
              <div className="flex-1 min-w-0">
                {step.name && (
                  <span className="text-[11px] font-medium text-[rgba(61,57,41,0.7)] mr-1.5">{step.name}</span>
                )}
                <span className="text-[11px] text-[rgba(61,57,41,0.45)] truncate">{step.prompt}</span>
              </div>
              {step.agent && (
                <span className="text-[9px] font-mono text-[rgba(61,57,41,0.3)] flex-shrink-0">{step.agent}</span>
              )}
              {!confirmed && (
                <button
                  onClick={() => { setConfirmed(true); onCommand(`Run sessions 0 to ${step.index} (stop after session ${step.index}).`); }}
                  className="opacity-0 group-hover:opacity-100 flex-shrink-0 text-[10px] px-1.5 py-0.5 rounded bg-[rgba(200,90,42,0.15)] text-[rgba(200,90,42,0.8)] hover:bg-[rgba(200,90,42,0.28)] transition-all"
                  title={`Run sessions 1–${step.index + 1} only`}
                >
                  ▶ Run to here
                </button>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Task summary bar
// ---------------------------------------------------------------------------

function TaskSummaryBar({ executions }: { executions: ToolExecution[] }) {
  const [collapsed, setCollapsed] = useState(false);

  if (executions.length === 0) return null;

  const doneCount = executions.filter((e) => e.status === "done").length;
  const errorCount = executions.filter((e) => e.status === "error").length;
  const total = executions.length;
  const hasRunning = executions.some((e) => e.status === "running");
  const currentTool = executions.find((e) => e.status === "running");
  const allDone = !hasRunning && total > 0;

  const label = currentTool
    ? currentTool.name
    : allDone
    ? "All tasks completed"
    : "Waiting...";

  return (
    <div className="flex-shrink-0 border-b border-[rgba(61,57,41,0.08)]">
      <button
        onClick={() => setCollapsed(!collapsed)}
        className="w-full flex items-center gap-3 px-4 py-2.5 hover:bg-[rgba(61,57,41,0.04)] transition-colors"
      >
        {/* Status icon */}
        {hasRunning ? (
          <Spinner />
        ) : errorCount > 0 ? (
          <span className="text-red-400 text-[14px]">&#10007;</span>
        ) : (
          <span className="text-emerald-400 text-[14px]">&#10003;</span>
        )}
        {/* Task label */}
        <span className="flex-1 text-left text-[13px] text-[rgba(61,57,41,0.85)] truncate">
          {label}
        </span>
        {/* Progress counter */}
        <span className="text-[12px] text-[rgba(61,57,41,0.4)] font-mono tabular-nums">
          {doneCount + errorCount}/{total}
        </span>
        {/* Collapse toggle */}
        <span className={`text-[rgba(61,57,41,0.3)] text-[11px] transition-transform ${collapsed ? "" : "rotate-180"}`}>
          ▾
        </span>
      </button>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main ToolPanel
// ---------------------------------------------------------------------------

export function ToolPanel({ executions, onCommand, onEditPreview }: Props) {
  return (
    <div className="flex flex-col h-full bg-[#ece5d8] text-[#3d3929]">
      {/* Task summary bar */}
      <TaskSummaryBar executions={executions} />

      {/* Tool execution cards */}
      <div className="flex-1 overflow-y-auto p-3 space-y-2">
        {executions.length === 0 && (
          <div className="text-center text-[rgba(61,57,41,0.3)] mt-8 text-[13px]">No tool activity</div>
        )}
        {executions.map((exec) => (
          <div
            key={exec.id}
            className={`rounded-xl p-3 text-[13px] border ${
              exec.status === "running"
                ? "border-[rgba(200,90,42,0.25)] bg-[rgba(200,90,42,0.05)]"
                : exec.status === "error"
                ? "border-[rgba(255,69,58,0.25)] bg-[rgba(255,69,58,0.05)]"
                : exec.status === "interrupted"
                ? "border-[rgba(245,158,11,0.25)] bg-[rgba(245,158,11,0.05)]"
                : "border-[rgba(61,57,41,0.08)] bg-[rgba(61,57,41,0.05)]"
            }`}
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <StatusIndicator status={exec.status} />
                <span className="font-mono font-medium text-[#3d3929] text-[13px]">{exec.name}</span>
              </div>
              <span className="text-[11px] text-[rgba(61,57,41,0.3)]">
                {formatDuration(exec.startTime, exec.endTime)}
              </span>
            </div>

            {/* WhipFlow source + session steps — shown instead of generic partial result */}
            {exec.name === "whipflow_run" ? (
              <>
                <WhipflowSource exec={exec} />
                {/* Check if the result is a preview (from mode=auto/preview) */}
                {exec.result && isWhipflowPreview(tryParseJSON(exec.result)) ? (
                  <StepPreviewCard
                    preview={tryParseJSON(exec.result) as WhipflowPreview}
                    onCommand={onCommand}
                    onEdit={onEditPreview ? () => {
                      const p = tryParseJSON(exec.result) as WhipflowPreview;
                      onEditPreview(p.source);
                    } : undefined}
                  />
                ) : (
                  <WhipflowSessionSteps
                    exec={exec}
                    onCommand={onCommand}
                  />
                )}
              </>
            ) : (
              <>
                <CollapsibleJSON label="Arguments" data={exec.args} />
                {exec.partialResult !== undefined && exec.status === "running" && (
                  <CollapsibleJSON label="Partial Result" data={exec.partialResult} />
                )}
              </>
            )}

            {exec.result !== undefined && (
              <CollapsibleJSON label="Result" data={exec.result} />
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
