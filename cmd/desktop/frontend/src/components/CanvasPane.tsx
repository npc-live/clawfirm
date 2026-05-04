import { useState, useRef, useCallback, useEffect } from "react";
import ReactMarkdown from "react-markdown";
import { GetWebhookBaseURL, AbortCurrentTurn, GetConfig, ReadCanvasFile, ListCanvasFiles } from "../lib/wails-shim";
import { useWebSocket, type WSMessage } from "../hooks/useWebSocket";
import { ToolPanel, type ToolExecution } from "./ToolPanel";
import { HtmlPreview } from "./HtmlPreview";

// ─── Types ────────────────────────────────────────────────────────────────────

interface CanvasNode {
  id: string;
  x: number;
  y: number;
  width: number;
  height: number;
  agentName: string;
  sessionId: string;
  minimized: boolean;
  // File-backed preview: watches ~/.clawfirm/canvas/{htmlFile}.html and auto-refreshes
  htmlFile?: string;
}

interface Message {
  role: "user" | "assistant";
  content: string;
  streaming?: boolean;
}

// ─── Constants ────────────────────────────────────────────────────────────────

const NODE_W = 620;
const NODE_H = 520;
const MIN_W = 340;
const MIN_H = 280;
const STORAGE_KEY = "pi-canvas-v1";

// ─── Persistence helpers ──────────────────────────────────────────────────────

interface CanvasState {
  nodes: CanvasNode[];
  pan: { x: number; y: number };
  zoom: number;
}

function loadCanvasState(): CanvasState {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) return JSON.parse(raw);
  } catch {}
  return { nodes: [], pan: { x: 0, y: 0 }, zoom: 1 };
}

function saveCanvasState(state: CanvasState) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {}
}

function newNode(x: number, y: number, agentName: string, htmlFile?: string): CanvasNode {
  return {
    id: "n" + Date.now(),
    x, y,
    width: NODE_W,
    height: NODE_H,
    agentName,
    sessionId: "s" + Date.now(),
    minimized: false,
    htmlFile,
  };
}

// ─── Main pane ────────────────────────────────────────────────────────────────

export function CanvasPane() {
  const init = loadCanvasState();
  const [nodes, setNodes] = useState<CanvasNode[]>(init.nodes);
  const [pan, setPan] = useState(init.pan);
  const [zoom, setZoom] = useState(init.zoom);
  const [agentNames, setAgentNames] = useState<string[]>([]);
  const [wsBase, setWsBase] = useState<string | null>(null);
  const [focusId, setFocusId] = useState<string | null>(null);
  const [newFilePrompt, setNewFilePrompt] = useState(false);
  const [fileNameInput, setFileNameInput] = useState("");
  const [canvasFiles, setCanvasFiles] = useState<string[]>([]);

  const canvasRef = useRef<HTMLDivElement>(null);
  const panRef = useRef(pan);
  const zoomRef = useRef(zoom);
  panRef.current = pan;
  zoomRef.current = zoom;

  // Save whenever nodes / pan / zoom change (debounced)
  const saveTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(() => {
    if (saveTimer.current) clearTimeout(saveTimer.current);
    saveTimer.current = setTimeout(() => {
      saveCanvasState({ nodes, pan, zoom });
    }, 400);
    return () => { if (saveTimer.current) clearTimeout(saveTimer.current); };
  }, [nodes, pan, zoom]);

  useEffect(() => {
    GetConfig().then(c => setAgentNames(c?.agents?.map(a => a.name) ?? [])).catch(() => {});
    GetWebhookBaseURL().then(base => {
      if (base) setWsBase(base.replace(/^http/, "ws"));
    }).catch(() => {});
  }, []);

  // ── Pan drag ──────────────────────────────────────────────────────────────
  const panDragging = useRef(false);
  const panStart = useRef({ mx: 0, my: 0, px: 0, py: 0 });

  function onBgPointerDown(e: React.PointerEvent) {
    if ((e.target as HTMLElement) !== canvasRef.current) return;
    panDragging.current = true;
    panStart.current = { mx: e.clientX, my: e.clientY, px: panRef.current.x, py: panRef.current.y };
    (e.target as HTMLElement).setPointerCapture(e.pointerId);
    e.preventDefault();
  }
  function onBgPointerMove(e: React.PointerEvent) {
    if (!panDragging.current) return;
    setPan({
      x: panStart.current.px + (e.clientX - panStart.current.mx),
      y: panStart.current.py + (e.clientY - panStart.current.my),
    });
  }
  function onBgPointerUp() { panDragging.current = false; }

  // ── Wheel zoom ────────────────────────────────────────────────────────────
  function onWheel(e: React.WheelEvent) {
    e.preventDefault();
    const factor = e.deltaY < 0 ? 1.08 : 0.93;
    const rect = canvasRef.current!.getBoundingClientRect();
    const mx = e.clientX - rect.left;
    const my = e.clientY - rect.top;
    setZoom(z => {
      const nz = Math.min(3, Math.max(0.2, z * factor));
      setPan(p => ({
        x: mx - (mx - p.x) * (nz / z),
        y: my - (my - p.y) * (nz / z),
      }));
      return nz;
    });
  }

  // ── Double-click to create ────────────────────────────────────────────────
  function onBgDblClick(e: React.MouseEvent) {
    if ((e.target as HTMLElement) !== canvasRef.current) return;
    if (agentNames.length === 0) return; // not ready yet
    const rect = canvasRef.current!.getBoundingClientRect();
    const cx = (e.clientX - rect.left - pan.x) / zoom;
    const cy = (e.clientY - rect.top - pan.y) / zoom;
    addNode(cx - NODE_W / 2, cy - NODE_H / 2, agentNames[0]);
  }

  function addNode(x: number, y: number, agentName: string, htmlFile?: string) {
    const n = newNode(x, y, agentName, htmlFile);
    setNodes(prev => [...prev, n]);
    setFocusId(n.id);
  }

  function addFileCell(fileName: string) {
    if (!fileName.trim()) return;
    const name = fileName.trim().replace(/\.html$/, "");
    // Check if a cell with this htmlFile already exists — reuse it
    const existing = nodes.find(n => n.htmlFile === name);
    if (existing) {
      setFocusId(existing.id);
      return;
    }
    const cx = (-pan.x + 100) / zoom;
    const cy = (-pan.y + 100) / zoom;
    addNode(cx, cy, agentNames[0] ?? "assistant", name);
  }

  function removeNode(id: string) {
    setNodes(prev => prev.filter(n => n.id !== id));
    if (focusId === id) setFocusId(null);
  }

  function updateNode(id: string, patch: Partial<CanvasNode>) {
    setNodes(prev => prev.map(n => n.id === id ? { ...n, ...patch } : n));
  }

  function handleAddFromToolbar() {
    if (agentNames.length === 0) return;
    const cx = (-pan.x + 100) / zoom;
    const cy = (-pan.y + 100) / zoom;
    addNode(cx, cy, agentNames[0]);
  }

  const sorted = [...nodes].sort((a, b) =>
    a.id === focusId ? 1 : b.id === focusId ? -1 : 0
  );

  const ready = agentNames.length > 0;

  return (
    <div className="relative w-full h-full overflow-hidden bg-[rgb(22,22,20)] select-none">
      {/* Dot grid */}
      <svg className="absolute inset-0 w-full h-full pointer-events-none" style={{ zIndex: 0 }}>
        <defs>
          <pattern id="dots" x={pan.x % (20 * zoom)} y={pan.y % (20 * zoom)}
            width={20 * zoom} height={20 * zoom} patternUnits="userSpaceOnUse">
            <circle cx={1} cy={1} r={0.8} fill="rgba(61,57,41,0.1)" />
          </pattern>
        </defs>
        <rect width="100%" height="100%" fill="url(#dots)" />
      </svg>

      {/* Toolbar */}
      <div className="absolute top-4 left-1/2 -translate-x-1/2 z-50 flex items-center gap-2
        bg-[rgb(28,28,26)] border border-dashed border-[rgba(200,190,150,0.12)] px-3 py-1.5">
        <button
          onClick={handleAddFromToolbar}
          disabled={!ready}
          className="text-[10px] px-2.5 py-1 font-mono uppercase tracking-wider border border-dashed
            border-[rgba(200,190,150,0.25)] text-[rgba(200,190,150,0.6)]
            hover:text-[rgba(200,190,150,0.9)] hover:border-[rgba(200,190,150,0.45)]
            disabled:opacity-40 transition-colors">
          [+ chat]
        </button>
        <button
          onClick={() => {
            setNewFilePrompt(true);
            setFileNameInput("");
            ListCanvasFiles().then(files => setCanvasFiles(files ?? [])).catch(() => setCanvasFiles([]));
          }}
          className="text-[10px] px-2.5 py-1 font-mono uppercase tracking-wider border border-dashed
            border-[rgba(168,85,247,0.35)] text-[rgba(168,85,247,0.65)]
            hover:text-purple-400 hover:border-[rgba(168,85,247,0.55)] transition-colors"
          title="Bind a cell to ~/.clawfirm/canvas/<name>.html">
          [+ file]
        </button>
        <span className="text-[rgba(200,190,150,0.2)] text-[10px] font-mono">// dbl-click</span>
        <div className="w-px h-3 bg-[rgba(200,190,150,0.08)]" />
        <span className="text-[10px] text-[rgba(200,190,150,0.3)] font-mono tabular-nums">{Math.round(zoom * 100)}%</span>
        <button
          onClick={() => { setPan({ x: 0, y: 0 }); setZoom(1); }}
          className="text-[10px] font-mono text-[rgba(200,190,150,0.3)] hover:text-[rgba(200,190,150,0.6)] transition-colors">
          [reset]
        </button>
        <span className="text-[10px] text-[rgba(200,190,150,0.25)] font-mono">{nodes.length}c</span>
      </div>

      {/* File cell name dialog */}
      {newFilePrompt && (
        <div className="absolute inset-0 z-[200] flex items-center justify-center bg-black/50"
          onClick={() => setNewFilePrompt(false)}>
          <div className="bg-[rgb(28,28,26)] border border-dashed border-[rgba(168,85,247,0.3)] p-5 w-80"
            onClick={e => e.stopPropagation()}>
            <p className="text-[10px] font-mono text-[rgba(168,85,247,0.7)] uppercase tracking-widest mb-1">// bind file cell</p>
            <p className="text-[10px] text-[rgba(200,190,150,0.35)] font-mono mb-3">
              watches ~/.clawfirm/canvas/<em className="text-purple-400">name</em>.html
            </p>
            {canvasFiles.length > 0 && (
              <div className="mb-3">
                <div className="text-[10px] text-[rgba(200,190,150,0.3)] font-mono uppercase tracking-wider mb-1.5">// existing</div>
                <div className="flex flex-col gap-1 max-h-36 overflow-y-auto">
                  {canvasFiles.map(f => (
                    <button
                      key={f}
                      onClick={() => { addFileCell(f); setNewFilePrompt(false); }}
                      className="text-left px-2 py-1 text-[11px] font-mono text-purple-400
                        border border-dashed border-[rgba(168,85,247,0.2)]
                        hover:border-[rgba(168,85,247,0.5)] transition-colors truncate">
                      &gt; {f}
                    </button>
                  ))}
                </div>
                <div className="my-3 border-t border-dashed border-[rgba(200,190,150,0.08)]" />
              </div>
            )}
            <input
              autoFocus
              value={fileNameInput}
              onChange={e => setFileNameInput(e.target.value)}
              onKeyDown={e => {
                if (e.key === "Enter") { addFileCell(fileNameInput); setNewFilePrompt(false); }
                if (e.key === "Escape") setNewFilePrompt(false);
              }}
              placeholder="// new name..."
              className="w-full bg-[rgba(200,190,150,0.05)] border border-dashed border-[rgba(168,85,247,0.25)]
                px-2 py-1.5 text-[11px] font-mono text-[rgba(200,190,150,0.8)] placeholder-[rgba(200,190,150,0.2)]
                focus:outline-none focus:border-[rgba(168,85,247,0.5)] mb-3"
            />
            <div className="flex gap-2 justify-end">
              <button onClick={() => setNewFilePrompt(false)}
                className="text-[10px] px-2.5 py-1 font-mono border border-dashed border-[rgba(200,190,150,0.15)] text-[rgba(200,190,150,0.35)] hover:text-[rgba(200,190,150,0.6)] transition-colors uppercase tracking-wider">
                [cancel]
              </button>
              <button
                onClick={() => { addFileCell(fileNameInput); setNewFilePrompt(false); }}
                disabled={!fileNameInput.trim()}
                className="text-[10px] px-2.5 py-1 font-mono border border-dashed border-[rgba(168,85,247,0.4)] text-purple-400
                  hover:border-[rgba(168,85,247,0.6)] disabled:opacity-40 transition-colors uppercase tracking-wider">
                [create]
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Canvas */}
      <div
        ref={canvasRef}
        className="absolute inset-0 cursor-grab active:cursor-grabbing"
        style={{ zIndex: 1 }}
        onPointerDown={onBgPointerDown}
        onPointerMove={onBgPointerMove}
        onPointerUp={onBgPointerUp}
        onWheel={onWheel}
        onDoubleClick={onBgDblClick}
      >
        {sorted.map(node => (
          <CanvasNodeCard
            key={node.id}
            node={node}
            pan={pan}
            zoom={zoom}
            wsBase={wsBase}
            agentNames={agentNames}
            focused={focusId === node.id}
            onFocus={() => setFocusId(node.id)}
            onUpdate={patch => updateNode(node.id, patch)}
            onRemove={() => removeNode(node.id)}
          />
        ))}
      </div>

      {nodes.length === 0 && (
        <div className="absolute inset-0 flex items-center justify-center pointer-events-none" style={{ zIndex: 2 }}>
          <div className="text-center">
            <pre className="text-[10px] font-mono leading-tight text-[rgba(200,190,150,0.15)]">{`+--[CANVAS]--+\n|            |\n|   EMPTY    |\n|            |\n+------------+`}</pre>
            <p className="text-[10px] font-mono text-[rgba(200,190,150,0.25)] mt-3 tracking-widest uppercase">
              {ready ? "// dbl-click to create · scroll zoom · drag pan" : "// loading agents..."}
            </p>
          </div>
        </div>
      )}
    </div>
  );
}

// ─── Node card ────────────────────────────────────────────────────────────────

interface NodeCardProps {
  node: CanvasNode;
  pan: { x: number; y: number };
  zoom: number;
  wsBase: string | null;
  agentNames: string[];
  focused: boolean;
  onFocus: () => void;
  onUpdate: (patch: Partial<CanvasNode>) => void;
  onRemove: () => void;
}

function CanvasNodeCard({ node, pan, zoom, wsBase, agentNames, focused, onFocus, onUpdate, onRemove }: NodeCardProps) {
  const [tab, setTab] = useState<"chat" | "tools" | "preview">(() =>
    node.htmlFile ? "preview" : "chat"
  );
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [isStreaming, setIsStreaming] = useState(false);
  const [isThinking, setIsThinking] = useState(false);
  const [wsStatus, setWsStatus] = useState<"connecting" | "open" | "closed">("connecting");
  const [toolExecutions, setToolExecutions] = useState<ToolExecution[]>([]);
  const [previewHtml, setPreviewHtml] = useState<string | null>(null);

  // wsURL via useEffect — mirrors ChatView pattern, connects only when wsBase is ready
  const [wsURL, setWsURL] = useState<string | null>(null);
  useEffect(() => {
    if (wsBase && node.agentName) {
      setWsURL(`${wsBase}/ws/${node.agentName}/${node.sessionId}`);
    } else {
      setWsURL(null);
    }
  }, [wsBase, node.agentName, node.sessionId]);

  // File polling: if this cell is bound to an htmlFile, poll every 5s
  useEffect(() => {
    if (!node.htmlFile) return;
    let cancelled = false;
    async function poll() {
      if (cancelled) return;
      try {
        const html = await ReadCanvasFile(node.htmlFile!);
        if (html) setPreviewHtml(html);
      } catch {}
    }
    poll(); // immediate first fetch
    const timer = setInterval(poll, 5000);
    return () => { cancelled = true; clearInterval(timer); };
  }, [node.htmlFile]);

  const bottomRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  // Drag
  const dragging = useRef(false);
  const dragStart = useRef({ mx: 0, my: 0, nx: 0, ny: 0 });

  // Resize
  const resizing = useRef(false);
  const resizeStart = useRef({ mx: 0, my: 0, w: 0, h: 0 });

  function extractPreviewContent(text: string): string | null {
    // Prefer ```html blocks
    const htmlMatches = [...text.matchAll(/```html\n([\s\S]*?)```/g)];
    if (htmlMatches.length > 0) return htmlMatches[htmlMatches.length - 1][1];
    // Fallback: ```markdown blocks
    const mdMatches = [...text.matchAll(/```markdown\n([\s\S]*?)```/g)];
    if (mdMatches.length > 0) return mdMatches[mdMatches.length - 1][1];
    return null;
  }

  const handleMessage = useCallback((msg: WSMessage) => {
    if (msg.type === "thinking") {
      setIsThinking(true);
      return;
    }
    if (msg.type === "delta" && msg.content) {
      setIsThinking(false);
      setMessages(prev => {
        const last = prev[prev.length - 1];
        let updated: Message;
        if (last?.role === "assistant" && last.streaming) {
          updated = { ...last, content: last.content + msg.content };
        } else {
          updated = { role: "assistant", content: msg.content!, streaming: true };
        }
        const preview = extractPreviewContent(updated.content);
        if (preview) { setPreviewHtml(preview); setTab("preview"); }
        return last?.role === "assistant" && last.streaming
          ? [...prev.slice(0, -1), updated]
          : [...prev, updated];
      });
    } else if (msg.type === "done") {
      setIsStreaming(false);
      setIsThinking(false);
      setMessages(prev => {
        const last = prev[prev.length - 1];
        return last?.streaming ? [...prev.slice(0, -1), { ...last, streaming: false }] : prev;
      });
    } else if (msg.type === "error") {
      setIsStreaming(false);
      setIsThinking(false);
      setMessages(prev => [...prev, { role: "assistant", content: `⚠️ ${msg.content}` }]);
    } else if (msg.type === "tool_start") {
      setToolExecutions(prev => [{
        id: msg.tool_call_id!, name: msg.tool_name!, args: msg.tool_args,
        status: "running", startTime: Date.now(),
      }, ...prev]);
      setTab("tools");
    } else if (msg.type === "tool_update") {
      setToolExecutions(prev => prev.map(t => {
        if (t.id !== msg.tool_call_id) return t;
        if (t.name === "whipflow_run" && msg.partial_result != null) {
          const s: any[] = Array.isArray(t.partialResult) ? t.partialResult : [];
          return { ...t, partialResult: [...s, msg.partial_result] };
        }
        return { ...t, partialResult: msg.partial_result };
      }));
    } else if (msg.type === "tool_end") {
      setToolExecutions(prev => prev.map(t =>
        t.id === msg.tool_call_id
          ? { ...t, status: msg.tool_is_error ? "error" : "done", result: msg.tool_result, endTime: Date.now() }
          : t
      ));
    }
  }, []);

  const handleOpen = useCallback(() => setWsStatus("open"), []);
  const handleClose = useCallback(() => { setWsStatus("closed"); setIsStreaming(false); setIsThinking(false); }, []);

  const { send } = useWebSocket({ url: wsURL, onMessage: handleMessage, onOpen: handleOpen, onClose: handleClose });

  // Listen for postMessage from File Cell iframe (sandbox="allow-scripts").
  // Canvas HTML buttons use: window.parent.postMessage({type:'run-whip', file:'...'}, '*')
  useEffect(() => {
    function onIframeMessage(e: MessageEvent) {
      if (!e.data || e.data.type !== "run-whip") return;
      const file: string = e.data.file ?? "";
      const source: string = e.data.source ?? "";
      if (!file && !source) return;
      if (wsStatus !== "open" || isStreaming) return;
      const content = file
        ? `运行 whipflow 文件: ${file}`
        : `运行 whipflow:\n\`\`\`\n${source}\n\`\`\``;
      setMessages(prev => [...prev, { role: "user", content }]);
      setIsStreaming(true);
      send(JSON.stringify({ type: "message", content }));
      setTab("chat");
    }
    window.addEventListener("message", onIframeMessage);
    return () => window.removeEventListener("message", onIframeMessage);
  }, [wsStatus, isStreaming, send]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  function handleSend() {
    const text = input.trim();
    if (!text || isStreaming || wsStatus !== "open") return;
    setMessages(prev => [...prev, { role: "user", content: text }]);
    setInput("");
    setIsStreaming(true);
    send(JSON.stringify({ type: "message", content: text }));
    setTab("chat");
  }

  function handleStop() {
    AbortCurrentTurn(node.agentName, node.sessionId);
    setIsStreaming(false);
  }

  // ── Header drag ────────────────────────────────────────────────────────────
  function onHeaderPointerDown(e: React.PointerEvent) {
    if ((e.target as HTMLElement).closest("button,select,input,textarea")) return;
    onFocus();
    dragging.current = true;
    dragStart.current = { mx: e.clientX, my: e.clientY, nx: node.x, ny: node.y };
    (e.target as HTMLElement).setPointerCapture(e.pointerId);
    e.stopPropagation();
  }
  function onHeaderPointerMove(e: React.PointerEvent) {
    if (!dragging.current) return;
    onUpdate({
      x: dragStart.current.nx + (e.clientX - dragStart.current.mx) / zoom,
      y: dragStart.current.ny + (e.clientY - dragStart.current.my) / zoom,
    });
  }
  function onHeaderPointerUp() { dragging.current = false; }

  // ── Resize handle ──────────────────────────────────────────────────────────
  function onResizePointerDown(e: React.PointerEvent) {
    e.stopPropagation(); e.preventDefault();
    onFocus();
    resizing.current = true;
    resizeStart.current = { mx: e.clientX, my: e.clientY, w: node.width, h: node.height };
    (e.target as HTMLElement).setPointerCapture(e.pointerId);
  }
  function onResizePointerMove(e: React.PointerEvent) {
    if (!resizing.current) return;
    onUpdate({
      width: Math.max(MIN_W, resizeStart.current.w + (e.clientX - resizeStart.current.mx) / zoom),
      height: Math.max(MIN_H, resizeStart.current.h + (e.clientY - resizeStart.current.my) / zoom),
    });
  }
  function onResizePointerUp() { resizing.current = false; }

  const left = node.x * zoom + pan.x;
  const top = node.y * zoom + pan.y;

  const statusText =
    wsStatus === "open" ? "[ws:ok]" :
    wsStatus === "connecting" ? "[ws:...]" : "[ws:off]";
  const statusCls =
    wsStatus === "open" ? "text-[rgba(30,150,60,0.7)]" :
    wsStatus === "connecting" ? "text-[rgba(180,130,20,0.7)]" : "text-[rgba(200,50,30,0.7)]";

  const tabBtn = (t: typeof tab, label: string, badge?: number) => (
    <button key={t} onClick={() => setTab(t)}
      className={`px-3 py-1 text-[10px] font-mono uppercase tracking-widest transition-colors ${
        tab === t
          ? "text-[rgba(200,190,150,0.9)] border-b border-[rgba(200,190,150,0.5)]"
          : "text-[rgba(200,190,150,0.3)] hover:text-[rgba(200,190,150,0.6)]"
      }`}>
      // {label}{badge != null && badge > 0 ? `(${badge})` : ""}
    </button>
  );

  const nodeW = node.minimized ? NODE_W : node.width;
  const nodeH = node.minimized ? 40 : node.height;

  return (
    <div
      style={{
        position: "absolute",
        left, top,
        width: nodeW * zoom,
        height: nodeH * zoom,
        zIndex: focused ? 100 : 10,
      }}
      onPointerDown={onFocus}
    >
      {/* Inner div scaled by zoom */}
      <div
        className="flex flex-col overflow-hidden"
        style={{
          width: nodeW,
          height: nodeH,
          transform: `scale(${zoom})`,
          transformOrigin: "top left",
          border: focused
            ? "1px dashed rgba(200,190,150,0.4)"
            : "1px dashed rgba(200,190,150,0.12)",
          background: "rgb(24,24,22)",
        }}
      >
        {/* Header */}
        <div
          className="flex items-center gap-2 px-3 h-9 bg-[rgb(30,30,28)] border-b border-dashed border-[rgba(200,190,150,0.1)]
            cursor-grab active:cursor-grabbing flex-shrink-0"
          onPointerDown={onHeaderPointerDown}
          onPointerMove={onHeaderPointerMove}
          onPointerUp={onHeaderPointerUp}
        >
          <span className={`text-[9px] font-mono flex-shrink-0 ${node.htmlFile ? "text-purple-400" : statusCls}`}>
            {node.htmlFile ? "[file]" : statusText}
          </span>
          {node.htmlFile ? (
            <span className="flex-1 text-[11px] font-mono text-purple-300 truncate">
              {node.htmlFile}.html
            </span>
          ) : (
            <select
              value={node.agentName}
              onChange={e => onUpdate({ agentName: e.target.value })}
              onClick={e => e.stopPropagation()}
              className="flex-1 bg-transparent text-[11px] font-mono text-[rgba(200,190,150,0.7)] outline-none cursor-pointer min-w-0"
            >
              {agentNames.map(n => <option key={n} value={n}>{n}</option>)}
              {agentNames.length === 0 && <option value={node.agentName}>{node.agentName}</option>}
            </select>
          )}
          <span className="text-[9px] text-[rgba(200,190,150,0.2)] font-mono flex-shrink-0">
            #{node.sessionId.slice(1, 7)}
          </span>
          <button
            onClick={e => { e.stopPropagation(); onUpdate({ minimized: !node.minimized }); }}
            className="text-[rgba(200,190,150,0.3)] hover:text-[rgba(200,190,150,0.7)] transition-colors text-[10px] font-mono px-1">
            {node.minimized ? "[+]" : "[-]"}
          </button>
          <button
            onClick={e => { e.stopPropagation(); onRemove(); }}
            className="text-[rgba(200,190,150,0.3)] hover:text-red-400 transition-colors text-[10px] font-mono px-1">
            [x]
          </button>
        </div>

        {!node.minimized && (
          <>
            {/* Tab bar */}
            <div className="flex items-center gap-0 px-2 py-0.5 bg-[rgb(26,26,24)] border-b border-dashed border-[rgba(200,190,150,0.08)] flex-shrink-0">
              {tabBtn("chat", "chat")}
              {tabBtn("tools", "tools", toolExecutions.length)}
              {tabBtn("preview", "preview")}
            </div>

            {/* Body */}
            <div className="flex-1 min-h-0 overflow-hidden relative">
              {/* Chat */}
              <div className={`flex flex-col absolute inset-0 ${tab === "chat" ? "" : "hidden"}`}>
                <div className="flex-1 overflow-y-auto px-3 py-3 space-y-2.5">
                  {messages.length === 0 && (
                    <p className="text-center text-[10px] font-mono text-[rgba(200,190,150,0.25)] mt-10 uppercase tracking-widest">
                      {wsStatus === "connecting" ? "// connecting..." : wsStatus === "closed" ? "// disconnected" : "// awaiting input"}
                    </p>
                  )}
                  {messages.map((msg, i) => (
                    <div key={i} className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}>
                      <div className={`max-w-[85%] px-2.5 py-2 text-[11px] font-mono leading-relaxed ${
                        msg.role === "user"
                          ? "border border-dashed border-[rgba(200,190,150,0.35)] text-[rgba(200,190,150,0.85)] bg-[rgba(200,190,150,0.05)]"
                          : "border border-dashed border-[rgba(200,190,150,0.15)] text-[rgba(200,190,150,0.7)]"
                      }`}>
                        {msg.role === "assistant"
                          ? <div className="[&_pre]:overflow-x-auto [&_pre]:max-w-full [&_code]:break-all [&_pre_code]:break-normal">
                              <ReactMarkdown>{msg.content}</ReactMarkdown>
                            </div>
                          : msg.content
                        }
                        {msg.streaming && <span className="inline-block w-1 h-3 ml-1 bg-current opacity-70 animate-pulse" />}
                      </div>
                    </div>
                  ))}
                  {isThinking && !messages.some(m => m.role === "assistant" && m.streaming) && (
                    <div className="flex justify-start">
                      <span className="text-[10px] font-mono text-[rgba(200,190,150,0.4)] animate-pulse">// PROCESSING...</span>
                    </div>
                  )}
                  <div ref={bottomRef} />
                </div>
                <div className="border-t border-dashed border-[rgba(200,190,150,0.08)] px-3 py-2 flex gap-2 items-end flex-shrink-0">
                  <div className="flex-1 flex items-start border border-dashed border-[rgba(200,190,150,0.2)] bg-[rgba(200,190,150,0.03)]">
                    <span className="pt-1.5 pl-2 text-[10px] font-mono text-[rgba(200,190,150,0.3)] flex-shrink-0">&gt;</span>
                    <textarea
                      ref={inputRef}
                      rows={1}
                      value={input}
                      placeholder="// message... (⌘↵)"
                      onChange={e => {
                        setInput(e.target.value);
                        e.target.style.height = "auto";
                        e.target.style.height = Math.min(e.target.scrollHeight, 100) + "px";
                      }}
                      onKeyDown={e => {
                        if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) { e.preventDefault(); handleSend(); }
                      }}
                      className="flex-1 bg-transparent px-2 py-1.5 text-[11px] font-mono text-[rgba(200,190,150,0.8)] placeholder-[rgba(200,190,150,0.2)]
                        focus:outline-none resize-none overflow-y-auto"
                      style={{ minHeight: "32px", maxHeight: "100px" }}
                    />
                  </div>
                  {isStreaming ? (
                    <button onClick={handleStop}
                      className="px-2.5 py-1.5 text-[10px] font-mono border border-dashed border-[rgba(200,50,30,0.4)] text-red-400
                        hover:border-red-400 transition-colors flex-shrink-0 uppercase tracking-wider">
                      [stop]
                    </button>
                  ) : (
                    <button onClick={handleSend} disabled={!input.trim() || wsStatus !== "open"}
                      className="px-2.5 py-1.5 text-[10px] font-mono border border-dashed border-[rgba(200,190,150,0.25)] text-[rgba(200,190,150,0.5)]
                        hover:text-[rgba(200,190,150,0.85)] hover:border-[rgba(200,190,150,0.45)] disabled:opacity-35 transition-colors flex-shrink-0 uppercase tracking-wider">
                      [send]
                    </button>
                  )}
                </div>
              </div>

              {/* Tools */}
              <div className={`absolute inset-0 overflow-y-auto ${tab === "tools" ? "" : "hidden"}`}>
                {toolExecutions.length === 0
                  ? <p className="text-center text-[10px] font-mono text-[rgba(200,190,150,0.25)] mt-10 uppercase tracking-widest">// no tool activity</p>
                  : <ToolPanel executions={toolExecutions} onCommand={() => {}} />
                }
              </div>

              {/* Preview */}
              <div className={`absolute inset-0 ${tab === "preview" ? "" : "hidden"}`}>
                {previewHtml
                  ? <HtmlPreview html={previewHtml} />
                  : (
                    <div className="flex flex-col items-center justify-center h-full gap-2">
                      {node.htmlFile ? (
                        <>
                          <p className="text-[10px] font-mono text-[rgba(168,85,247,0.5)] uppercase tracking-widest">// waiting for file</p>
                          <p className="text-[10px] font-mono text-purple-400/60 truncate px-4 text-center">
                            ~/.clawfirm/canvas/{node.htmlFile}.html
                          </p>
                          <p className="text-[10px] font-mono text-[rgba(200,190,150,0.2)]">// polling 5s</p>
                        </>
                      ) : (
                        <p className="text-[10px] font-mono text-[rgba(200,190,150,0.25)] uppercase tracking-widest">// no preview</p>
                      )}
                    </div>
                  )
                }
              </div>
            </div>

            {/* Resize handle */}
            <div
              className="absolute bottom-0 right-0 w-4 h-4 cursor-se-resize z-10"
              style={{ background: "linear-gradient(135deg, transparent 50%, rgba(200,190,150,0.1) 50%)" }}
              onPointerDown={onResizePointerDown}
              onPointerMove={onResizePointerMove}
              onPointerUp={onResizePointerUp}
            />
          </>
        )}
      </div>
    </div>
  );
}
