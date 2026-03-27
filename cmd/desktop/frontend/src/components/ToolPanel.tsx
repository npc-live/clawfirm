import { useState } from "react";

export interface ToolExecution {
  id: string;
  name: string;
  args?: any;
  status: "running" | "done" | "error";
  result?: any;
  partialResult?: any;
  startTime: number;
  endTime?: number;
}

interface Props {
  executions: ToolExecution[];
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
}

function isWhipflowStep(v: any): v is WhipflowSessionStep {
  return v !== null && typeof v === "object" && typeof v.prompt === "string" && typeof v.index === "number";
}

// Accumulate session steps from a stream of partial_result updates.
// Each update is a single WhipflowSessionStep; we merge them into a map
// keyed by index so start (done=false) and end (done=true) merge cleanly.
function mergeSessionSteps(partial: any): Map<number, WhipflowSessionStep> {
  const map = new Map<number, WhipflowSessionStep>();
  if (!partial) return map;
  // partial may be the latest single update or an array of all updates
  const items: any[] = Array.isArray(partial) ? partial : [partial];
  for (const item of items) {
    if (!isWhipflowStep(item)) continue;
    const existing = map.get(item.index);
    map.set(item.index, existing ? { ...existing, ...item } : item);
  }
  return map;
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
        className="text-[11px] text-[rgba(255,255,255,0.35)] hover:text-[rgba(255,255,255,0.6)] flex items-center gap-1 transition-colors"
      >
        <span className={`transition-transform ${open ? "rotate-90" : ""}`}>▶</span>
        {label}
      </button>
      {open && (
        <pre className="mt-1 text-[11px] text-[rgba(240,237,229,0.65)] bg-[rgba(255,255,255,0.03)] border border-[rgba(255,255,255,0.06)] rounded-lg p-2 overflow-x-auto max-h-48 overflow-y-auto whitespace-pre-wrap break-all font-mono">
          {text}
        </pre>
      )}
    </div>
  );
}

function StatusIndicator({ status }: { status: ToolExecution["status"] }) {
  if (status === "running") {
    return <span className="inline-block w-2 h-2 rounded-full bg-[#2688f9] animate-pulse" />;
  }
  if (status === "done") {
    return <span className="text-emerald-400 text-[13px]">&#10003;</span>;
  }
  return <span className="text-red-400 text-[13px]">&#10007;</span>;
}

function Spinner() {
  return (
    <svg className="animate-spin w-4 h-4 text-[#2688f9]" viewBox="0 0 24 24" fill="none">
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
        className="text-[11px] text-[rgba(255,255,255,0.35)] hover:text-[rgba(255,255,255,0.6)] flex items-center gap-1 transition-colors"
      >
        <span className={`transition-transform ${open ? "rotate-90" : ""}`}>▶</span>
        {label}
      </button>
      {open && content && (
        <pre className="mt-1 text-[11px] text-[rgba(240,237,229,0.65)] bg-[rgba(255,255,255,0.03)] border border-[rgba(255,255,255,0.06)] rounded-lg p-2 overflow-x-auto max-h-64 overflow-y-auto whitespace-pre-wrap break-all font-mono leading-relaxed">
          {content}
        </pre>
      )}
      {open && !content && args.file && (
        <p className="mt-1 text-[11px] text-[rgba(255,255,255,0.3)] font-mono">{args.file}</p>
      )}
    </div>
  );
}

function WhipflowSessionSteps({ exec }: { exec: ToolExecution }) {
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
      <div className="flex items-center gap-2 text-[11px] text-[rgba(255,255,255,0.4)]">
        <span className="font-mono">{doneCount + errorCount}/{steps.length} sessions</span>
        {runningCount > 0 && <span className="text-[#2688f9] animate-pulse">· {runningCount} running</span>}
        {errorCount > 0 && <span className="text-red-400">· {errorCount} failed</span>}
      </div>
      {/* Per-session cards */}
      {steps.map((step) => (
        <SessionStepCard key={step.index} step={step} />
      ))}
    </div>
  );
}

function SessionStepCard({ step }: { step: WhipflowSessionStep }) {
  const [open, setOpen] = useState(false);
  const isRunning = !step.done;
  const isError = step.done && !!step.error;
  const isDone = step.done && !step.error;

  const borderCls = isRunning
    ? "border-[rgba(38,136,249,0.3)] bg-[rgba(38,136,249,0.04)]"
    : isError
    ? "border-[rgba(239,68,68,0.3)] bg-[rgba(239,68,68,0.04)]"
    : "border-[rgba(255,255,255,0.08)] bg-[rgba(255,255,255,0.03)]";

  const label = step.name
    ? `#${step.index + 1} · ${step.name}`
    : `Session ${step.index + 1}`;

  return (
    <div className={`rounded-lg border overflow-hidden ${borderCls}`}>
      <button
        className="w-full flex items-center gap-2 px-3 py-2 text-left hover:bg-[rgba(255,255,255,0.03)] transition-colors"
        onClick={() => setOpen((o) => !o)}
      >
        {/* Status dot */}
        {isRunning ? (
          <span className="w-2 h-2 rounded-full bg-[#2688f9] animate-pulse flex-shrink-0" />
        ) : isDone ? (
          <span className="text-emerald-400 text-[12px] flex-shrink-0">✓</span>
        ) : (
          <span className="text-red-400 text-[12px] flex-shrink-0">✗</span>
        )}
        {/* Label */}
        <span className="flex-1 text-[12px] text-[rgba(240,237,229,0.8)] font-medium truncate">
          {label}
        </span>
        {/* Provider badge */}
        {step.provider && (
          <span className="text-[10px] font-mono text-[rgba(255,255,255,0.3)] flex-shrink-0">
            {step.provider}
          </span>
        )}
        {/* Duration */}
        {step.duration_ms != null && step.duration_ms > 0 && (
          <span className="text-[10px] text-[rgba(255,255,255,0.25)] font-mono flex-shrink-0">
            {step.duration_ms < 1000 ? `${step.duration_ms}ms` : `${(step.duration_ms / 1000).toFixed(1)}s`}
          </span>
        )}
        {/* Expand toggle */}
        <span className={`text-[rgba(255,255,255,0.2)] text-[10px] flex-shrink-0 transition-transform ${open ? "rotate-180" : ""}`}>▾</span>
      </button>

      {open && (
        <div className="px-3 pb-3 pt-1 space-y-2 border-t border-[rgba(255,255,255,0.06)]">
          {/* Prompt */}
          <div>
            <p className="text-[10px] font-semibold text-[rgba(255,255,255,0.3)] uppercase tracking-wider mb-1">Prompt</p>
            <pre className="text-[11px] text-[rgba(240,237,229,0.6)] bg-[rgba(255,255,255,0.03)] rounded-lg p-2 whitespace-pre-wrap break-words max-h-40 overflow-y-auto font-mono leading-relaxed">
              {step.prompt}
            </pre>
          </div>
          {/* Output */}
          {step.output && (
            <div>
              <p className="text-[10px] font-semibold text-[rgba(255,255,255,0.3)] uppercase tracking-wider mb-1">Output</p>
              <pre className="text-[11px] text-[rgba(240,237,229,0.75)] bg-[rgba(255,255,255,0.03)] rounded-lg p-2 whitespace-pre-wrap break-words max-h-48 overflow-y-auto font-mono leading-relaxed">
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
    <div className="flex-shrink-0 border-b border-[rgba(255,255,255,0.08)]">
      <button
        onClick={() => setCollapsed(!collapsed)}
        className="w-full flex items-center gap-3 px-4 py-2.5 hover:bg-[rgba(255,255,255,0.03)] transition-colors"
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
        <span className="flex-1 text-left text-[13px] text-[rgba(240,237,229,0.85)] truncate">
          {label}
        </span>
        {/* Progress counter */}
        <span className="text-[12px] text-[rgba(255,255,255,0.4)] font-mono tabular-nums">
          {doneCount + errorCount}/{total}
        </span>
        {/* Collapse toggle */}
        <span className={`text-[rgba(255,255,255,0.3)] text-[11px] transition-transform ${collapsed ? "" : "rotate-180"}`}>
          ▾
        </span>
      </button>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main ToolPanel
// ---------------------------------------------------------------------------

export function ToolPanel({ executions }: Props) {
  return (
    <div className="flex flex-col h-full bg-[rgb(26,26,24)] text-[rgb(240,237,229)]">
      {/* Task summary bar */}
      <TaskSummaryBar executions={executions} />

      {/* Tool execution cards */}
      <div className="flex-1 overflow-y-auto p-3 space-y-2">
        {executions.length === 0 && (
          <div className="text-center text-[rgba(255,255,255,0.2)] mt-8 text-[13px]">No tool activity</div>
        )}
        {executions.map((exec) => (
          <div
            key={exec.id}
            className={`rounded-xl p-3 text-[13px] border ${
              exec.status === "running"
                ? "border-[rgba(38,136,249,0.25)] bg-[rgba(38,136,249,0.05)]"
                : exec.status === "error"
                ? "border-[rgba(255,69,58,0.25)] bg-[rgba(255,69,58,0.05)]"
                : "border-[rgba(255,255,255,0.08)] bg-[rgba(255,255,255,0.05)]"
            }`}
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <StatusIndicator status={exec.status} />
                <span className="font-mono font-medium text-[rgb(240,237,229)] text-[13px]">{exec.name}</span>
              </div>
              <span className="text-[11px] text-[rgba(255,255,255,0.3)]">
                {formatDuration(exec.startTime, exec.endTime)}
              </span>
            </div>

            {/* WhipFlow source + session steps — shown instead of generic partial result */}
            {exec.name === "whipflow_run" ? (
              <>
                <WhipflowSource exec={exec} />
                <WhipflowSessionSteps exec={exec} />
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
