import { useState, useEffect, useRef, useCallback } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { GetWebhookBaseURL, AbortCurrentTurn, GetHistory, GetAgentSkills, GetToolExecutionState, type SkillInfo } from "../lib/wails-shim";
import { useWebSocket, type WSMessage } from "../hooks/useWebSocket";
import { ToolPanel, type ToolExecution, isWhipflowPreview } from "./ToolPanel";
import { HtmlPreview } from "./HtmlPreview";

interface AttachedFile {
  file: File;
  preview: string; // object URL for thumbnail
  data: string;    // base64 (no prefix)
  mime: string;
}

const MAX_FILE_SIZE = 20 * 1024 * 1024; // 20 MB

function readFileAsBase64(file: File): Promise<{ data: string; mime: string }> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const result = reader.result as string;
      // strip "data:<mime>;base64," prefix
      const base64 = result.split(",")[1];
      resolve({ data: base64, mime: file.type });
    };
    reader.onerror = () => reject(reader.error);
    reader.readAsDataURL(file);
  });
}

interface Message {
  role: "user" | "assistant";
  content: string;
  thinking?: string; // accumulated native thinking (e.g. Anthropic extended thinking)
  streaming?: boolean;
  fileCount?: number;
}

interface Props {
  agentName: string;
  sessionID: string;
  onBack: () => void;
  onNewSession: () => void;
  onOpenSession?: (agentName: string, sessionID: string) => void;
}

export function ChatView({ agentName, sessionID, onBack, onNewSession, onOpenSession: _onOpenSession }: Props) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [wsURL, setWsURL] = useState<string | null>(null);
  const [isStreaming, setIsStreaming] = useState(false);
  const [isThinking, setIsThinking] = useState(false);
  const [wsStatus, setWsStatus] = useState<"connecting" | "open" | "closed">("closed");
  const [toolExecutions, setToolExecutions] = useState<ToolExecution[]>([]);
  const [agentSkills, setAgentSkills] = useState<SkillInfo[]>([]);
  const [showSkillPicker, setShowSkillPicker] = useState(false);
  const [skillQuery, setSkillQuery] = useState("");
  const [skillPickerIdx, setSkillPickerIdx] = useState(0);
  const [previewHtml, setPreviewHtml] = useState<string | null>(null);
  const [whipPlanCode, setWhipPlanCode] = useState<string | null>(null);
  const [whipPlanEditing, setWhipPlanEditing] = useState(false);
  const [whipPlanEditText, setWhipPlanEditText] = useState("");
  // ask statements extracted from whip code: [{varName, prompt}]
  const [whipAskFields, setWhipAskFields] = useState<{varName: string; prompt: string}[]>([]);
  const [whipAskValues, setWhipAskValues] = useState<Record<string, string>>({});
  const [whipAskReady, setWhipAskReady] = useState(false);
  const [activeTab, setActiveTab] = useState<"tools" | "preview">("tools");
  const [attachedFiles, setAttachedFiles] = useState<AttachedFile[]>([]);
  const [leftPanelWidth, setLeftPanelWidth] = useState(40); // percentage
  const bottomRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const isDragging = useRef(false);

  // Resize splitter drag handlers
  const handleSplitterMouseDown = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    isDragging.current = true;
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";

    const onMouseMove = (ev: MouseEvent) => {
      if (!isDragging.current || !containerRef.current) return;
      const rect = containerRef.current.getBoundingClientRect();
      const pct = ((ev.clientX - rect.left) / rect.width) * 100;
      setLeftPanelWidth(Math.min(80, Math.max(20, pct)));
    };
    const onMouseUp = () => {
      isDragging.current = false;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
      document.removeEventListener("mousemove", onMouseMove);
      document.removeEventListener("mouseup", onMouseUp);
    };
    document.addEventListener("mousemove", onMouseMove);
    document.addEventListener("mouseup", onMouseUp);
  }, []);

  // Persist current session so App.tsx can restore it on next launch.
  useEffect(() => {
    localStorage.setItem("clawfirm:lastSession", JSON.stringify({ agentName, sessionID }));
  }, [agentName, sessionID]);

  // Resolve WebSocket URL and load existing history on mount.
  useEffect(() => {
    GetWebhookBaseURL().then((base) => {
      if (base) {
        const wsBase = base.replace(/^http/, "ws");
        setWsStatus("connecting");
        setWsURL(`${wsBase}/ws/${agentName}/${sessionID}`);
      }
    });

    // Load skills for this agent.
    GetAgentSkills(agentName).then((s) => { if (s) setAgentSkills(s); }).catch(() => {});

    // Load saved messages and tool executions for this session.
    GetHistory("webchat/" + agentName, sessionID).then((history) => {
      if (!history || history.length === 0) return;
      setMessages(
        history.map((m) => ({
          role: m.role as "user" | "assistant",
          content: m.content,
        }))
      );
      // Restore HTML preview or whip plan from last assistant message.
      for (let i = history.length - 1; i >= 0; i--) {
        if (history[i].role === "assistant") {
          const html = extractHtmlBlock(history[i].content);
          if (html) { setPreviewHtml(html); setActiveTab("preview"); break; }
          const whip = extractWhipBlock(history[i].content);
          if (whip) { applyWhipPlan(whip); break; }
        }
      }
    }).catch(() => {});

    GetToolExecutionState("webchat/" + agentName, sessionID).then((execs) => {
      if (!execs || execs.length === 0) return;
      setToolExecutions(execs.map((e: any) => ({
        id: e.id,
        name: e.name,
        args: e.args,
        status: (e.status || (e.isError ? "error" : "done")) as "running" | "done" | "error" | "interrupted",
        result: e.result,
        startTime: e.startedAt || e.timestamp,
        endTime: e.endedAt || e.timestamp,
      })));
    }).catch(() => {});
  }, [agentName, sessionID]);

  // Split message content into segments: thinking blocks and regular text.
  // Handles <think>...</think> tags emitted by models like DeepSeek/QwQ.
  // Also handles unclosed <think> (streaming) as "thinking_open".
  // Strips orphan </think> tags that appear without a matching <think>.
  function parseThinkingBlocks(text: string): { type: "text" | "thinking" | "thinking_open"; content: string }[] {
    const segments: { type: "text" | "thinking" | "thinking_open"; content: string }[] = [];
    const re = /<think>([\s\S]*?)<\/think>/g;
    let lastIndex = 0;
    let match: RegExpExecArray | null;
    while ((match = re.exec(text)) !== null) {
      if (match.index > lastIndex) {
        segments.push({ type: "text", content: text.slice(lastIndex, match.index) });
      }
      segments.push({ type: "thinking", content: match[1].trim() });
      lastIndex = re.lastIndex;
    }
    if (lastIndex < text.length) {
      const remaining = text.slice(lastIndex);
      // Check for unclosed <think> tag (still streaming)
      const openIdx = remaining.indexOf("<think>");
      if (openIdx !== -1) {
        if (openIdx > 0) {
          segments.push({ type: "text", content: remaining.slice(0, openIdx) });
        }
        segments.push({ type: "thinking_open", content: remaining.slice(openIdx + 7).trim() });
      } else {
        // Strip any orphan </think> tags (e.g. model output starts with </think>)
        const cleaned = remaining.replace(/<\/think>/g, "");
        if (cleaned.trim()) {
          segments.push({ type: "text", content: cleaned });
        }
      }
    }
    return segments;
  }

  // Extract the last complete ```html code block from text.
  function extractHtmlBlock(text: string): string | null {
    const matches = [...text.matchAll(/```html\n([\s\S]*?)```/g)];
    return matches.length > 0 ? matches[matches.length - 1][1] : null;
  }

  // Extract the last complete ```whip code block from text.
  function extractWhipBlock(text: string): string | null {
    const matches = [...text.matchAll(/```whip\n([\s\S]*?)```/g)];
    return matches.length > 0 ? matches[matches.length - 1][1] : null;
  }

  // Parse ask statements from whip source: `ask varName: "prompt text"`
  function extractAskFields(source: string): {varName: string; prompt: string}[] {
    const fields: {varName: string; prompt: string}[] = [];
    const re = /^\s*ask\s+(\w+)\s*:\s*"([^"]*)"/gm;
    let m: RegExpExecArray | null;
    while ((m = re.exec(source)) !== null) {
      fields.push({ varName: m[1], prompt: m[2] });
    }
    return fields;
  }

  function applyWhipPlan(code: string) {
    setWhipPlanCode(code);
    setWhipPlanEditText(code);
    const fields = extractAskFields(code);
    setWhipAskFields(fields);
    setWhipAskValues(Object.fromEntries(fields.map(f => [f.varName, ""])));
    setWhipAskReady(fields.length === 0);
  }

  // Stable onMessage — wrapped in useCallback so identity never changes.
  const handleMessage = useCallback((msg: WSMessage) => {
    if (msg.type === "thinking") {
      setIsThinking(true);
      return;
    }
    if (msg.type === "thinking_delta" && msg.content) {
      setIsThinking(true);
      setMessages((prev) => {
        const last = prev[prev.length - 1];
        if (last?.role === "assistant" && last.streaming) {
          return [...prev.slice(0, -1), { ...last, thinking: (last.thinking || "") + msg.content }];
        }
        return [...prev, { role: "assistant", content: "", thinking: msg.content, streaming: true }];
      });
      return;
    }
    if (msg.type === "message_end") {
      setMessages((prev) => {
        const last = prev[prev.length - 1];
        if (last?.role === "assistant" && last.streaming) {
          return [...prev.slice(0, -1), { ...last, streaming: false }];
        }
        return prev;
      });
      return;
    }
    if (msg.type === "delta" && msg.content) {
      setIsThinking(false);
      setMessages((prev) => {
        const last = prev[prev.length - 1];
        let updated: Message;
        if (last?.role === "assistant" && last.streaming) {
          updated = { ...last, content: last.content + msg.content };
        } else {
          updated = { role: "assistant", content: msg.content!, streaming: true };
        }
        const html = extractHtmlBlock(updated.content);
        if (html) { setPreviewHtml(html); setActiveTab("preview"); }
        const whip = extractWhipBlock(updated.content);
        if (whip) { applyWhipPlan(whip); }
        if (last?.role === "assistant" && last.streaming) {
          return [...prev.slice(0, -1), updated];
        }
        return [...prev, updated];
      });
    } else if (msg.type === "done") {
      setIsStreaming(false);
      setIsThinking(false);
      setMessages((prev) => {
        const last = prev[prev.length - 1];
        return last?.streaming ? [...prev.slice(0, -1), { ...last, streaming: false }] : prev;
      });
    } else if (msg.type === "error") {
      setIsStreaming(false);
      setIsThinking(false);
      setMessages((prev) => [
        ...prev,
        { role: "assistant", content: `⚠️ Error: ${msg.content}` },
      ]);
    } else if (msg.type === "tool_start") {
      setToolExecutions((prev) => [...prev, {
        id: msg.tool_call_id!,
        name: msg.tool_name!,
        args: msg.tool_args,
        status: "running",
        startTime: Date.now(),
      }]);
    } else if (msg.type === "tool_update") {
      setToolExecutions((prev) => prev.map((t) => {
        if (t.id !== msg.tool_call_id) return t;
        // For whipflow_run, accumulate session steps as an array.
        if (t.name === "whipflow_run" && msg.partial_result != null) {
          const prevSteps: any[] = Array.isArray(t.partialResult) ? t.partialResult : [];
          return { ...t, partialResult: [...prevSteps, msg.partial_result] };
        }
        return { ...t, partialResult: msg.partial_result };
      }));
    } else if (msg.type === "tool_end") {
      // If the result is a whipflow_preview, apply the plan UI for ask handling.
      if (msg.tool_result && typeof msg.tool_result === "string") {
        try {
          const parsed = JSON.parse(msg.tool_result);
          if (isWhipflowPreview(parsed) && parsed.analysis.has_ask) {
            applyWhipPlan(parsed.source);
          }
        } catch { /* not JSON, ignore */ }
      }
      setToolExecutions((prev) => prev.map((t) =>
        t.id === msg.tool_call_id ? {
          ...t,
          status: msg.tool_is_error ? "error" : "done",
          result: msg.tool_result,
          endTime: Date.now(),
        } : t
      ));
    }
  }, []);

  const handleOpen = useCallback(() => setWsStatus("open"), []);
  const handleClose = useCallback(() => {
    setWsStatus("closed");
    setIsStreaming(false);
    setIsThinking(false);
  }, []);

  const { send } = useWebSocket({
    url: wsURL,
    onMessage: handleMessage,
    onOpen: handleOpen,
    onClose: handleClose,
  });

  // Re-fetch tool execution state on WebSocket reconnect.
  const prevWsStatus = useRef(wsStatus);
  useEffect(() => {
    if (prevWsStatus.current !== "open" && wsStatus === "open") {
      // Just reconnected — refresh tool states from backend.
      GetToolExecutionState("webchat/" + agentName, sessionID).then((execs) => {
        if (!execs || execs.length === 0) return;
        setToolExecutions((prev) => {
          // Merge: use backend state but preserve any live partialResult from prior WS updates.
          const byId = new Map(prev.map((t) => [t.id, t]));
          return execs.map((e: any) => {
            const existing = byId.get(e.id);
            return {
              id: e.id,
              name: e.name,
              args: existing?.args ?? e.args,
              status: (e.status || (e.isError ? "error" : "done")) as "running" | "done" | "error" | "interrupted",
              result: e.result || existing?.result,
              partialResult: existing?.partialResult,
              startTime: e.startedAt || e.timestamp,
              endTime: e.endedAt || e.timestamp,
            };
          });
        });
      }).catch(() => {});
    }
    prevWsStatus.current = wsStatus;
  }, [wsStatus, agentName, sessionID]);

  // Auto-scroll.
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const filteredSkills = agentSkills.filter((s) => {
    if (!skillQuery) return true;
    const q = skillQuery.toLowerCase();
    return s.name.toLowerCase().includes(q) || s.description.toLowerCase().includes(q);
  });

  const showNewCommand = showSkillPicker && "new".startsWith(skillQuery.toLowerCase());
  const showPlanCommand = showSkillPicker && "plan".startsWith(skillQuery.toLowerCase());

  // Revoke object URLs on unmount.
  useEffect(() => {
    return () => {
      attachedFiles.forEach((f) => URL.revokeObjectURL(f.preview));
    };
  }, []);

  async function addFiles(files: FileList | File[]) {
    const arr = Array.from(files);
    for (const f of arr) {
      if (f.size > MAX_FILE_SIZE) {
        alert(`File "${f.name}" exceeds 20 MB limit.`);
        continue;
      }
      if (!f.type.startsWith("image/") && !f.type.startsWith("video/") && !f.type.startsWith("audio/")) continue;
      const { data, mime } = await readFileAsBase64(f);
      const preview = URL.createObjectURL(f);
      setAttachedFiles((prev) => [...prev, { file: f, preview, data, mime }]);
    }
  }

  function removeFile(index: number) {
    setAttachedFiles((prev) => {
      URL.revokeObjectURL(prev[index].preview);
      return prev.filter((_, i) => i !== index);
    });
  }

  function handlePaste(e: React.ClipboardEvent<HTMLTextAreaElement>) {
    const items = e.clipboardData.items;
    const imageFiles: File[] = [];
    for (let i = 0; i < items.length; i++) {
      if (items[i].type.startsWith("image/")) {
        const file = items[i].getAsFile();
        if (file) imageFiles.push(file);
      }
    }
    if (imageFiles.length > 0) {
      e.preventDefault();
      addFiles(imageFiles);
    }
  }

  function handleInputChange(val: string) {
    setInput(val);
    const slash = val.lastIndexOf("/");
    if (slash !== -1 && (slash === 0 || val[slash - 1] === " ")) {
      setSkillQuery(val.slice(slash + 1));
      setShowSkillPicker(true);
      setSkillPickerIdx(0);
    } else {
      setShowSkillPicker(false);
    }
  }

  function selectSkill(skill: SkillInfo) {
    const prefix = input.replace(/\/\S*$/, "");
    setInput(prefix + skill.name + " ");
    setShowSkillPicker(false);
    inputRef.current?.focus();
  }

  function handleInputKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (showSkillPicker && filteredSkills.length > 0) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setSkillPickerIdx((i) => Math.min(i + 1, filteredSkills.length - 1));
        return;
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        setSkillPickerIdx((i) => Math.max(i - 1, 0));
        return;
      }
      if (e.key === "Enter" || e.key === "Tab") {
        e.preventDefault();
        selectSkill(filteredSkills[skillPickerIdx]);
        return;
      }
      if (e.key === "Escape") {
        setShowSkillPicker(false);
        return;
      }
    }
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  // Send a command as a user message — used by ToolPanel buttons.
  function sendCommand(command: string) {
    if (wsStatus !== "open" || isStreaming) return;
    setMessages((prev) => [...prev, { role: "user", content: command }]);
    setIsStreaming(true);
    setActiveTab("tools");
    send(JSON.stringify({ type: "message", content: command }));
  }

  async function handleSend() {
    const text = (input || inputRef.current?.value || "").trim();
    if (!text || isStreaming) return;
    if (text === "/new") {
      setInput("");
      setShowSkillPicker(false);
      onNewSession();
      return;
    }
    if (text.startsWith("/plan ") || text === "/plan") {
      const description = text.slice(6).trim();
      setInput("");
      setShowSkillPicker(false);
      if (wsStatus !== "open") return;
      // Build prompt — let LLM decide whether to read skill for reference.
      const prompt = `请根据以下需求设计一个 Whipflow 工作流（如需查阅语法，可用 skill 工具读取 "whipflow" skill），**只输出 \`\`\`whip 代码块**，不包含其他解释。\n\n需求：${description || "(no description provided)"}`;
      setMessages((prev) => [...prev, { role: "user", content: text }]);
      setIsStreaming(true);
      send(JSON.stringify({ type: "message", content: prompt }));
      return;
    }
    if (wsStatus !== "open") return;
    const fileCount = attachedFiles.length;
    const images = attachedFiles.map((f) => ({ data: f.data, mime: f.mime }));
    setMessages((prev) => [...prev, { role: "user", content: text, fileCount: fileCount || undefined }]);
    setInput("");
    setShowSkillPicker(false);
    // Clear attachments (revoke previews).
    attachedFiles.forEach((f) => URL.revokeObjectURL(f.preview));
    setAttachedFiles([]);
    setIsStreaming(true);
    send(JSON.stringify({ type: "message", content: text, ...(images.length > 0 ? { images } : {}) }));
  }

  function handleStop() {
    AbortCurrentTurn(agentName, sessionID);
    setIsStreaming(false);
  }

  function handleEditPreview(source: string) {
    applyWhipPlan(source);
  }

  const statusColor =
    wsStatus === "open" ? "bg-[#1e1c17]" :
    wsStatus === "connecting" ? "bg-[rgba(30,28,23,0.4)] animate-pulse" :
    "bg-[rgba(200,90,42,0.6)]";

  return (
    <div className="flex flex-col h-screen bg-[#f0ece3]">
      {/* Header */}
      <header className="flex items-center gap-3 px-4 py-2.5 border-b border-dashed border-[rgba(30,28,23,0.15)]">
        <button onClick={onBack} className="text-[rgba(30,28,23,0.4)] hover:text-[#1e1c17] transition-colors font-mono text-[13px]">←</button>
        <h1 className="font-bold text-[#1e1c17] text-[12px] tracking-widest uppercase">// {agentName}</h1>
        <button
          onClick={() => navigator.clipboard.writeText(sessionID)}
          title="Copy thread ID"
          className="text-[10px] text-[rgba(30,28,23,0.35)] hover:text-[rgba(30,28,23,0.65)] font-mono bg-[rgba(30,28,23,0.05)] hover:bg-[rgba(30,28,23,0.08)] px-1.5 py-0.5 border border-dashed border-[rgba(30,28,23,0.12)] transition-colors"
        >
          #{sessionID}
        </button>
        <span className="flex items-center gap-1.5 text-[10px] text-[rgba(30,28,23,0.4)] ml-auto tracking-wider font-mono uppercase">
          <span className={`w-1.5 h-1.5 ${statusColor}`} />
          // {wsStatus}
        </span>
      </header>

      {/* Main content: Chat (left) + Tool Panel (right) */}
      <div ref={containerRef} className="flex flex-1 min-h-0">
        {/* Left: Chat + Input */}
        <div style={{ width: `${leftPanelWidth}%` }} className="flex flex-col min-w-0 bg-[#f0ece3]">
          <div className="flex-1 overflow-y-auto px-4 py-4 space-y-3">
            {messages.length === 0 && (
              <div className="flex flex-col items-center justify-center mt-16 gap-3 select-none">
                <pre className="text-[10px] leading-[1.4] font-mono text-center" style={{color: 'transparent', backgroundImage: 'linear-gradient(135deg, #6b8cba 0%, #c85a2a 40%, #6bba8c 80%)', WebkitBackgroundClip: 'text', backgroundClip: 'text'}}>
{`  ___ _      _   __    __  ___  _  ___  __  __
 / __| |    /_\\ \\ \\  / /  | __|| || _ \\|  \\/  |
| (__| |__ / _ \\ \\ \\/ /   | _| | ||   /| |\\/| |
 \\___|____/_/ \\_\\ \\__/    |_|  |_||_|_\\|_|  |_|`}
                </pre>
                <p className="text-[10px] text-[rgba(30,28,23,0.35)] tracking-[0.2em] uppercase font-mono">// SYSTEM READY // AWAITING INPUT...</p>
              </div>
            )}
            {messages.map((msg, i) => (
              <div key={i} className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}>
                <div
                  className={`max-w-[75%] min-w-0 overflow-hidden px-3 py-2.5 text-[12px] border border-dashed ${
                    msg.role === "user"
                      ? "bg-[rgba(30,28,23,0.07)] text-[#1e1c17] border-[rgba(30,28,23,0.25)]"
                      : "bg-[rgba(30,28,23,0.03)] text-[rgba(30,28,23,0.85)] border-[rgba(30,28,23,0.1)]"
                  }`}
                >
                  <div className="prose-pre:overflow-x-auto prose-pre:max-w-full [&_pre]:overflow-x-auto [&_pre]:max-w-full [&_code]:break-all [&_pre_code]:break-normal [&_table]:border-collapse [&_table]:w-full [&_th]:border [&_th]:border-white/20 [&_th]:px-2 [&_th]:py-1 [&_td]:border [&_td]:border-white/20 [&_td]:px-2 [&_td]:py-1">
                    {msg.role === "assistant" ? (
                      <>
                        {/* Native thinking (Anthropic extended thinking) */}
                        {msg.thinking && (
                          <details className="my-2 border border-dashed border-[rgba(30,28,23,0.12)] bg-[rgba(30,28,23,0.02)] overflow-hidden"
                            open={msg.streaming && !msg.content}>
                            <summary className="cursor-pointer select-none px-3 py-2 text-[10px] text-[rgba(30,28,23,0.4)] hover:text-[rgba(30,28,23,0.6)] transition-colors flex items-center gap-1.5 tracking-wider uppercase">
                              {msg.streaming && !msg.content && (
                                <span className="w-1.5 h-1.5 bg-[rgba(200,90,42,0.5)] animate-pulse" />
                              )}
                              // {msg.streaming && !msg.content ? "THINKING..." : "THINKING"}
                            </summary>
                            <div className="px-3 pb-2 text-[11px] text-[rgba(30,28,23,0.5)] leading-relaxed">
                              <ReactMarkdown remarkPlugins={[remarkGfm]}>{msg.thinking}</ReactMarkdown>
                            </div>
                          </details>
                        )}
                        {/* Text content — parse <think> tags from models like DeepSeek/QwQ */}
                        {msg.content.includes("<think>") || msg.content.includes("</think>") ? (
                          parseThinkingBlocks(msg.content).map((seg, si) =>
                            seg.type === "thinking" ? (
                              <details key={si} className="my-2 border border-dashed border-[rgba(30,28,23,0.12)] bg-[rgba(30,28,23,0.02)] overflow-hidden">
                                <summary className="cursor-pointer select-none px-3 py-2 text-[10px] text-[rgba(30,28,23,0.4)] hover:text-[rgba(30,28,23,0.6)] transition-colors tracking-wider uppercase">
                                  // THINKING
                                </summary>
                                <div className="px-3 pb-2 text-[11px] text-[rgba(30,28,23,0.5)] leading-relaxed">
                                  <ReactMarkdown remarkPlugins={[remarkGfm]}>{seg.content}</ReactMarkdown>
                                </div>
                              </details>
                            ) : seg.type === "thinking_open" ? (
                              <details key={si} open className="my-2 border border-dashed border-[rgba(200,90,42,0.2)] bg-[rgba(200,90,42,0.02)] overflow-hidden">
                                <summary className="cursor-pointer select-none px-3 py-2 text-[10px] text-[rgba(200,90,42,0.6)] flex items-center gap-1.5 tracking-wider uppercase">
                                  <span className="w-1.5 h-1.5 bg-[rgba(200,90,42,0.5)] animate-pulse" />
                                  // THINKING...
                                </summary>
                                {seg.content && (
                                  <div className="px-3 pb-2 text-[11px] text-[rgba(30,28,23,0.5)] leading-relaxed">
                                    <ReactMarkdown remarkPlugins={[remarkGfm]}>{seg.content}</ReactMarkdown>
                                  </div>
                                )}
                              </details>
                            ) : seg.content.trim() ? (
                              <ReactMarkdown key={si} remarkPlugins={[remarkGfm]}>{seg.content}</ReactMarkdown>
                            ) : null
                          )
                        ) : msg.content ? (
                          <ReactMarkdown remarkPlugins={[remarkGfm]}>{msg.content}</ReactMarkdown>
                        ) : null}
                      </>
                    ) : (
                      <ReactMarkdown remarkPlugins={[remarkGfm]}>{msg.content}</ReactMarkdown>
                    )}
                  </div>
                  {msg.role === "user" && msg.fileCount && (
                    <div className="mt-1 text-[11px] opacity-70">
                      {msg.fileCount} file{msg.fileCount > 1 ? "s" : ""} attached
                    </div>
                  )}
                  {msg.streaming && (
                    <span className="inline-block w-1 h-4 ml-1 bg-current opacity-70 animate-pulse" />
                  )}
                </div>
              </div>
            ))}
            {isThinking && !messages.some(m => m.role === "assistant" && m.streaming) && (
              <div className="flex justify-start">
                <div className="flex items-center gap-2 px-3 py-2 border border-dashed border-[rgba(30,28,23,0.15)] bg-[rgba(30,28,23,0.02)]">
                  <span className="text-[10px] font-mono text-[rgba(30,28,23,0.4)] tracking-widest animate-pulse">// PROCESSING...</span>
                </div>
              </div>
            )}
            <div ref={bottomRef} />
          </div>
          {/* Whip Plan inline banner */}
          {whipPlanCode && (
            <div className="border-t border-dashed border-[rgba(30,28,23,0.12)] px-4 py-3 flex flex-col gap-2">
              <div className="flex items-center justify-between">
                <span className="text-[10px] text-[rgba(30,28,23,0.4)] font-mono tracking-wider">// workflow.whip</span>
                <div className="flex gap-2">
                  <button
                    onClick={() => {
                      if (whipPlanEditing) {
                        setWhipPlanCode(whipPlanEditText);
                        setWhipPlanEditing(false);
                      } else {
                        setWhipPlanEditText(whipPlanCode);
                        setWhipPlanEditing(true);
                      }
                    }}
                    className="px-2 py-0.5 text-[10px] text-[rgba(30,28,23,0.5)] hover:text-[rgba(30,28,23,0.8)] border border-dashed border-[rgba(30,28,23,0.15)] hover:border-[rgba(30,28,23,0.3)] transition-colors uppercase tracking-wider"
                  >
                    {whipPlanEditing ? "[save]" : "[edit]"}
                  </button>
                  <button
                    onClick={() => {
                      const code = whipPlanEditing ? whipPlanEditText : whipPlanCode;
                      sendCommand(`Preview this workflow:\n\`\`\`whip\n${code}\n\`\`\``);
                    }}
                    disabled={isStreaming || wsStatus !== "open"}
                    className="px-2 py-0.5 text-[10px] text-[rgba(30,28,23,0.5)] hover:text-[rgba(30,28,23,0.8)] border border-dashed border-[rgba(30,28,23,0.15)] hover:border-[rgba(30,28,23,0.3)] disabled:opacity-40 transition-colors uppercase tracking-wider"
                    title="List sessions for step-by-step debug"
                  >
                    [step]
                  </button>
                  <button
                    onClick={() => {
                      const code = whipPlanEditing ? whipPlanEditText : whipPlanCode;
                      const inputsStr = whipAskFields.length > 0
                        ? ` with inputs: ${JSON.stringify(whipAskValues)}`
                        : "";
                      sendCommand(`Execute this workflow now${inputsStr}:\n\`\`\`whip\n${code}\n\`\`\``);
                    }}
                    disabled={isStreaming || wsStatus !== "open" || !whipAskReady}
                    className="px-2 py-0.5 text-[10px] text-[#1e1c17] bg-[rgba(30,28,23,0.08)] border border-dashed border-[rgba(30,28,23,0.3)] hover:bg-[rgba(30,28,23,0.14)] disabled:opacity-40 transition-colors uppercase tracking-wider font-bold"
                  >
                    [execute]
                  </button>
                  <button
                    onClick={() => { setWhipPlanCode(null); setWhipPlanEditing(false); setWhipAskFields([]); setWhipAskReady(false); }}
                    className="px-2 py-0.5 text-[10px] text-[rgba(30,28,23,0.3)] hover:text-[rgba(30,28,23,0.6)] transition-colors"
                  >
                    [x]
                  </button>
                </div>
              </div>

              {/* Ask fields form — shown when workflow has ask statements */}
              {whipAskFields.length > 0 && !whipAskReady && (
                <div className="flex flex-col gap-2 bg-[rgba(200,90,42,0.06)] border border-[rgba(200,90,42,0.2)] rounded-lg p-3">
                  <span className="text-[11px] text-[rgba(200,90,42,0.8)] font-medium">Workflow inputs required</span>
                  {whipAskFields.map(({ varName, prompt }) => (
                    <div key={varName} className="flex flex-col gap-1">
                      <label className="text-[10px] text-[rgba(61,57,41,0.4)]">{prompt || varName}</label>
                      <input
                        type="text"
                        value={whipAskValues[varName] ?? ""}
                        onChange={(e) => setWhipAskValues((prev) => ({ ...prev, [varName]: e.target.value }))}
                        placeholder={varName}
                        className="bg-[rgba(61,57,41,0.06)] border border-[rgba(61,57,41,0.12)] rounded-md px-2.5 py-1.5 text-[11px] text-[rgba(61,57,41,0.85)] focus:outline-none focus:ring-1 focus:ring-[rgba(200,90,42,0.4)] placeholder-[rgba(61,57,41,0.3)]"
                      />
                    </div>
                  ))}
                  <button
                    onClick={() => setWhipAskReady(true)}
                    disabled={whipAskFields.some(f => !(whipAskValues[f.varName] ?? "").trim())}
                    className="mt-1 self-end px-3 py-1.5 rounded-lg text-[11px] font-medium bg-[rgba(200,90,42,0.2)] text-[rgba(200,90,42,0.9)] hover:bg-[rgba(200,90,42,0.3)] disabled:opacity-40 transition-colors"
                  >
                    Confirm inputs →
                  </button>
                </div>
              )}

              {/* Confirmed inputs summary */}
              {whipAskFields.length > 0 && whipAskReady && (
                <div className="flex flex-wrap gap-2">
                  {whipAskFields.map(({ varName }) => (
                    <span key={varName} className="text-[10px] bg-[rgba(61,57,41,0.08)] border border-[rgba(61,57,41,0.08)] rounded px-2 py-0.5 text-[rgba(61,57,41,0.5)]">
                      {varName}: <span className="text-[rgba(61,57,41,0.7)]">{whipAskValues[varName]}</span>
                    </span>
                  ))}
                  <button onClick={() => setWhipAskReady(false)} className="text-[10px] text-[rgba(200,90,42,0.6)] hover:text-[rgba(200,90,42,0.9)]">edit</button>
                </div>
              )}

              {whipPlanEditing ? (
                <textarea
                  value={whipPlanEditText}
                  onChange={(e) => setWhipPlanEditText(e.target.value)}
                  rows={6}
                  className="w-full bg-[rgba(30,28,23,0.04)] border border-dashed border-[rgba(30,28,23,0.15)] p-2.5 text-[11px] font-mono text-[rgba(30,28,23,0.85)] resize-none focus:outline-none leading-relaxed"
                  spellCheck={false}
                />
              ) : (
                <pre className="text-[11px] font-mono text-[rgba(30,28,23,0.65)] bg-[rgba(30,28,23,0.04)] border border-dashed border-[rgba(30,28,23,0.1)] p-2.5 max-h-36 overflow-y-auto whitespace-pre-wrap break-words leading-relaxed">{whipPlanCode}</pre>
              )}
            </div>
          )}
          {/* Input — inside left column */}
          <div className="px-4 py-3">
            {/* Skill Picker */}
            {showSkillPicker && (showNewCommand || showPlanCommand || filteredSkills.length > 0) && (
              <div className="mb-2 border border-dashed border-[rgba(30,28,23,0.2)] bg-[#ece9e0] overflow-hidden">
                <div className="max-h-52 overflow-y-auto">
                  {showNewCommand && (
                    <button
                      onMouseDown={(e) => { e.preventDefault(); setInput("/new"); setShowSkillPicker(false); }}
                      className="w-full text-left px-3 py-2 flex flex-col gap-0.5 transition-colors hover:bg-[rgba(30,28,23,0.05)] border-b border-dashed border-[rgba(30,28,23,0.08)]"
                    >
                      <span className="text-[11px] font-mono text-[#1e1c17]">/new</span>
                      <span className="text-[10px] text-[rgba(30,28,23,0.4)]">// start a new session</span>
                    </button>
                  )}
                  {showPlanCommand && (
                    <button
                      onMouseDown={(e) => { e.preventDefault(); setInput("/plan "); setShowSkillPicker(false); inputRef.current?.focus(); }}
                      className="w-full text-left px-3 py-2 flex flex-col gap-0.5 transition-colors hover:bg-[rgba(30,28,23,0.05)] border-b border-dashed border-[rgba(30,28,23,0.08)]"
                    >
                      <span className="text-[11px] font-mono text-[#1e1c17]">/plan</span>
                      <span className="text-[10px] text-[rgba(30,28,23,0.4)]">// generate whipflow workflow</span>
                    </button>
                  )}
                  {filteredSkills.map((s, i) => (
                    <button
                      key={s.filePath}
                      onMouseDown={(e) => { e.preventDefault(); selectSkill(s); }}
                      className={`w-full text-left px-3 py-2 flex flex-col gap-0.5 transition-colors ${
                        i === skillPickerIdx
                          ? "bg-[rgba(30,28,23,0.08)]"
                          : "hover:bg-[rgba(30,28,23,0.04)]"
                      }`}
                    >
                      <span className="text-[11px] font-mono text-[#1e1c17]">/{s.name}</span>
                      <span className="text-[10px] text-[rgba(30,28,23,0.4)] line-clamp-1">// {s.description}</span>
                    </button>
                  ))}
                </div>
              </div>
            )}
            {/* Attachment preview strip */}
            {attachedFiles.length > 0 && (
              <div className="flex gap-2 mb-2 flex-wrap">
                {attachedFiles.map((f, i) => (
                  <div key={i} className="relative group">
                    <img
                      src={f.preview}
                      alt={f.file.name}
                      className="w-14 h-14 object-cover rounded-lg border border-[rgba(61,57,41,0.12)]"
                    />
                    <button
                      onClick={() => removeFile(i)}
                      className="absolute -top-1.5 -right-1.5 w-5 h-5 rounded-full bg-[rgba(61,57,41,0.7)] text-white text-[11px] flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity hover:bg-[rgba(220,50,40,0.8)]"
                    >
                      &times;
                    </button>
                  </div>
                ))}
              </div>
            )}
            {/* Hidden file input */}
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*,video/*,audio/*"
              multiple
              className="hidden"
              onChange={(e) => { if (e.target.files) addFiles(e.target.files); e.target.value = ""; }}
            />
            {/* Terminal input */}
            <div className="border border-dashed border-[rgba(30,28,23,0.2)] bg-[rgba(30,28,23,0.02)]">
              {/* Textarea row */}
              <div className="flex items-start px-3 pt-2.5 pb-1.5">
                <span className="text-[11px] text-[rgba(30,28,23,0.3)] font-mono mt-0.5 mr-2 select-none flex-shrink-0">&gt;</span>
                <textarea
                  ref={inputRef}
                  rows={1}
                  placeholder={wsStatus === "open" ? "// input command..." : "// connecting..."}
                  value={input}
                  onChange={(e) => { handleInputChange(e.target.value); e.target.style.height = "auto"; e.target.style.height = Math.min(e.target.scrollHeight, 200) + "px"; }}
                  onKeyDown={handleInputKeyDown}
                  onPaste={handlePaste}
                  disabled={isStreaming || wsStatus !== "open"}
                  className="flex-1 text-[12px] bg-transparent text-[#1e1c17] placeholder-[rgba(30,28,23,0.25)] focus:outline-none disabled:opacity-50 resize-none overflow-y-auto leading-relaxed font-mono"
                  style={{ minHeight: "24px", maxHeight: "200px" }}
                />
              </div>
              {/* Toolbar row */}
              <div className="flex items-center gap-1 px-3 pb-2 pt-0.5 border-t border-dashed border-[rgba(30,28,23,0.08)]">
                {/* Skill picker */}
                <button
                  onClick={() => { setInput("/"); handleInputChange("/"); inputRef.current?.focus(); }}
                  className="flex items-center gap-1 px-2 py-0.5 text-[10px] text-[rgba(30,28,23,0.4)] hover:text-[rgba(30,28,23,0.7)] hover:bg-[rgba(30,28,23,0.05)] transition-colors tracking-wider uppercase border border-dashed border-transparent hover:border-[rgba(30,28,23,0.15)]"
                >
                  <span>/ {agentName}</span>
                </button>
                {/* Attach */}
                <button
                  onClick={() => fileInputRef.current?.click()}
                  disabled={isStreaming || wsStatus !== "open"}
                  className="px-2 py-0.5 text-[10px] text-[rgba(30,28,23,0.35)] hover:text-[rgba(30,28,23,0.6)] disabled:opacity-40 transition-colors uppercase tracking-wider"
                  title="添加附件"
                >
                  + attach
                </button>
                {/* Tool count */}
                <button
                  onClick={() => setActiveTab("tools")}
                  className="px-2 py-0.5 text-[10px] text-[rgba(30,28,23,0.35)] hover:text-[rgba(30,28,23,0.6)] transition-colors uppercase tracking-wider"
                >
                  tools:{agentSkills.length || 1}
                </button>
                {/* Spacer */}
                <div className="flex-1" />
                {/* Model display */}
                <span className="text-[10px] text-[rgba(30,28,23,0.3)] font-mono uppercase tracking-wider">opus-4.6</span>
                {/* Send / Stop */}
                {isStreaming ? (
                  <button
                    onClick={handleStop}
                    className="ml-2 px-2.5 py-0.5 text-[10px] text-red-400 border border-dashed border-red-300 hover:bg-[rgba(255,69,58,0.08)] transition-colors tracking-wider uppercase"
                    title="停止"
                  >
                    [stop]
                  </button>
                ) : (
                  <button
                    onClick={handleSend}
                    disabled={!input.trim() || wsStatus !== "open"}
                    className="ml-2 px-2.5 py-0.5 text-[10px] text-[rgba(30,28,23,0.5)] border border-dashed border-[rgba(30,28,23,0.2)] hover:border-[rgba(30,28,23,0.4)] hover:text-[rgba(30,28,23,0.8)] disabled:opacity-30 transition-colors tracking-wider uppercase"
                    title="发送 (⌘↵)"
                  >
                    [send]
                  </button>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Resize handle */}
        <div
          onMouseDown={handleSplitterMouseDown}
          className="w-px flex-shrink-0 cursor-col-resize bg-[rgba(30,28,23,0.1)] hover:bg-[rgba(200,90,42,0.4)] active:bg-[rgba(200,90,42,0.6)] transition-colors"
          style={{borderLeft: '1px dashed rgba(30,28,23,0.15)'}}
        />

        {/* Right: Tabbed Panel */}
        <div style={{ width: `${100 - leftPanelWidth}%` }} className="border-l border-dashed border-[rgba(30,28,23,0.12)] min-h-0 flex flex-col">
          {/* Tab bar */}
          <div className="flex border-b border-dashed border-[rgba(30,28,23,0.1)] flex-shrink-0">
            <button
              onClick={() => setActiveTab("tools")}
              className={`px-4 py-2 text-[10px] tracking-widest uppercase transition-colors ${
                activeTab === "tools"
                  ? "text-[#1e1c17] border-b border-[#1e1c17]"
                  : "text-[rgba(30,28,23,0.35)] hover:text-[rgba(30,28,23,0.55)]"
              }`}
            >
              // tool activity
            </button>
            <button
              onClick={() => previewHtml && setActiveTab("preview")}
              className={`px-4 py-2 text-[10px] tracking-widest uppercase transition-colors ${
                activeTab === "preview"
                  ? "text-[#1e1c17] border-b border-[#1e1c17]"
                  : previewHtml
                  ? "text-[rgba(30,28,23,0.35)] hover:text-[rgba(30,28,23,0.55)]"
                  : "text-[rgba(30,28,23,0.2)] cursor-not-allowed"
              }`}
            >
              // html preview
            </button>
          </div>
          {/* Tab content */}
          <div className="flex-1 min-h-0 flex flex-col">
            {activeTab === "tools" ? (
              <div className="flex-1 min-h-0">
                <ToolPanel executions={toolExecutions} onCommand={sendCommand} onEditPreview={handleEditPreview} />
              </div>
            ) : previewHtml ? (
              <div className="h-full p-3">
                <HtmlPreview html={previewHtml} />
              </div>
            ) : (
              <div className="flex items-center justify-center h-full text-[rgba(61,57,41,0.3)] text-[13px]">
                No HTML preview available
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
