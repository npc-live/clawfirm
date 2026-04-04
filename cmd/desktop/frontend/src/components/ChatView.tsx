import { useState, useEffect, useRef, useCallback } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { GetWebhookBaseURL, AbortCurrentTurn, GetHistory, GetAgentSkills, GetToolExecutions, type SkillInfo } from "../wailsjs/go/app/App";
import { useWebSocket, type WSMessage } from "../hooks/useWebSocket";
import { ToolPanel, type ToolExecution, type WhipflowArgs, isWhipflowPreview } from "./ToolPanel";
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
  streaming?: boolean;
  fileCount?: number;
}

interface Props {
  agentName: string;
  sessionID: string;
  onBack: () => void;
  onNewSession: () => void;
}

export function ChatView({ agentName, sessionID, onBack, onNewSession }: Props) {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState("");
  const [wsURL, setWsURL] = useState<string | null>(null);
  const [isStreaming, setIsStreaming] = useState(false);
  const [wsStatus, setWsStatus] = useState<"connecting" | "open" | "closed">("connecting");
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
  const [lastWhipflowArgs, setLastWhipflowArgs] = useState<WhipflowArgs | undefined>();
  const [activeTab, setActiveTab] = useState<"tools" | "preview">("tools");
  const [attachedFiles, setAttachedFiles] = useState<AttachedFile[]>([]);
  const bottomRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Resolve WebSocket URL and load existing history on mount.
  useEffect(() => {
    GetWebhookBaseURL().then((base) => {
      if (base) {
        const wsBase = base.replace(/^http/, "ws");
        setWsURL(`${wsBase}/ws/${agentName}/${sessionID}`);
      }
    });

    // Load skills for this agent.
    GetAgentSkills(agentName).then(setAgentSkills).catch(() => {});

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

    GetToolExecutions("webchat/" + agentName, sessionID).then((execs) => {
      if (!execs || execs.length === 0) return;
      setToolExecutions(execs.map((e) => ({
        id: e.id,
        name: e.name,
        args: e.args,
        status: e.isError ? "error" as const : "done" as const,
        result: e.result,
        startTime: e.timestamp,
        endTime: e.timestamp,
      })));
    }).catch(() => {});
  }, [agentName, sessionID]);

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
    if (msg.type === "delta" && msg.content) {
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
      setMessages((prev) => {
        const last = prev[prev.length - 1];
        return last?.streaming ? [...prev.slice(0, -1), { ...last, streaming: false }] : prev;
      });
    } else if (msg.type === "error") {
      setIsStreaming(false);
      setMessages((prev) => [
        ...prev,
        { role: "assistant", content: `⚠️ Error: ${msg.content}` },
      ]);
    } else if (msg.type === "tool_start") {
      if (msg.tool_name === "whipflow_run" && msg.tool_args) {
        setLastWhipflowArgs({
          file: msg.tool_args.file,
          source: msg.tool_args.source,
          user_inputs: msg.tool_args.user_inputs,
        });
      }
      setToolExecutions((prev) => [{
        id: msg.tool_call_id!,
        name: msg.tool_name!,
        args: msg.tool_args,
        status: "running",
        startTime: Date.now(),
      }, ...prev]);
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
  }, []);

  const { send } = useWebSocket({
    url: wsURL,
    onMessage: handleMessage,
    onOpen: handleOpen,
    onClose: handleClose,
  });

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
      if (!f.type.startsWith("image/") && !f.type.startsWith("video/")) continue;
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
    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      handleSend();
    }
  }

  async function handleSend() {
    const text = input.trim();
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

  function handleRetryFromSession(sessionIndex: number, args: WhipflowArgs) {
    if (wsStatus !== "open" || isStreaming) return;
    const callID = "retry-session-" + Date.now();
    setIsStreaming(true);
    setActiveTab("tools");
    send(JSON.stringify({
      type: "run_tool",
      tool_name: "whipflow_run",
      tool_id: callID,
      tool_args: {
        file: args.file,
        source: args.source,
        user_inputs: args.user_inputs,
        mode: "execute",
        retry_from_session: sessionIndex,
      },
    }));
  }

  function handleConfirmWhipflow(source: string, userInputs?: Record<string, string>) {
    if (wsStatus !== "open") return;
    const callID = "confirm-preview-" + Date.now();
    setIsStreaming(true);
    setActiveTab("tools");
    send(JSON.stringify({
      type: "run_tool",
      tool_name: "whipflow_run",
      tool_id: callID,
      tool_args: {
        source,
        mode: "execute",
        ...(userInputs && Object.keys(userInputs).length > 0 ? { user_inputs: userInputs } : {}),
      },
    }));
  }

  function handleEditPreview(source: string) {
    applyWhipPlan(source);
  }

  const statusColor =
    wsStatus === "open" ? "bg-emerald-400" :
    wsStatus === "connecting" ? "bg-amber-400" :
    "bg-red-400";

  return (
    <div className="flex flex-col h-screen bg-[#f5f0e8]">
      {/* Header */}
      <header className="flex items-center gap-3 px-4 py-3 border-b border-[rgba(61,57,41,0.08)]">
        <button onClick={onBack} className="text-[rgba(61,57,41,0.4)] hover:text-[#3d3929] transition-colors">←</button>
        <h1 className="font-semibold text-[#3d3929] text-[15px] tracking-tight">{agentName}</h1>
        <span className="flex items-center gap-1.5 text-[11px] text-[rgba(61,57,41,0.4)] ml-auto">
          <span className={`w-1.5 h-1.5 rounded-full ${statusColor}`} />
          {wsStatus}
        </span>
      </header>

      {/* Main content: Chat (left) + Tool Panel (right) */}
      <div className="flex flex-1 min-h-0">
        {/* Left: Chat + Input (40%) */}
        <div className="w-[40%] flex flex-col min-w-0">
          <div className="flex-1 overflow-y-auto px-4 py-4 space-y-4">
            {messages.length === 0 && (
              <div className="text-center text-[rgba(61,57,41,0.3)] mt-16 text-[13px]">Start a conversation</div>
            )}
            {messages.map((msg, i) => (
              <div key={i} className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}>
                <div
                  className={`max-w-[75%] min-w-0 overflow-hidden rounded-2xl px-4 py-3 text-[13px] ${
                    msg.role === "user"
                      ? "bg-[#c85a2a] text-white"
                      : "bg-[rgba(61,57,41,0.05)] text-[rgba(61,57,41,0.85)] border border-[rgba(61,57,41,0.08)]"
                  }`}
                >
                  <div className="prose-pre:overflow-x-auto prose-pre:max-w-full [&_pre]:overflow-x-auto [&_pre]:max-w-full [&_code]:break-all [&_pre_code]:break-normal [&_table]:border-collapse [&_table]:w-full [&_th]:border [&_th]:border-white/20 [&_th]:px-2 [&_th]:py-1 [&_td]:border [&_td]:border-white/20 [&_td]:px-2 [&_td]:py-1">
                    <ReactMarkdown remarkPlugins={[remarkGfm]}>{msg.content}</ReactMarkdown>
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
            <div ref={bottomRef} />
          </div>
          {/* Whip Plan inline banner */}
          {whipPlanCode && (
            <div className="border-t border-[rgba(61,57,41,0.08)] px-4 py-3 flex flex-col gap-2">
              <div className="flex items-center justify-between">
                <span className="text-[11px] text-[rgba(61,57,41,0.4)] font-mono">workflow.whip</span>
                <div className="flex gap-2">
                  <button
                    onClick={() => {
                      if (whipPlanEditing) {
                        // Save edits back to whipPlanCode
                        setWhipPlanCode(whipPlanEditText);
                        setWhipPlanEditing(false);
                      } else {
                        setWhipPlanEditText(whipPlanCode);
                        setWhipPlanEditing(true);
                      }
                    }}
                    className="px-2.5 py-1 rounded-lg text-[11px] font-medium bg-[rgba(61,57,41,0.1)] text-[rgba(61,57,41,0.5)] hover:bg-[rgba(61,57,41,0.1)] transition-colors"
                  >
                    {whipPlanEditing ? "Save" : "Edit"}
                  </button>
                  <button
                    onClick={() => {
                      const code = whipPlanEditing ? whipPlanEditText : whipPlanCode;
                      const callID = "plan-" + Date.now();
                      setIsStreaming(true);
                      setActiveTab("tools");
                      send(JSON.stringify({ type: "run_tool", tool_name: "whipflow_run", tool_id: callID, tool_args: { source: code, mode: "execute", user_inputs: whipAskValues } }));
                    }}
                    disabled={isStreaming || wsStatus !== "open" || !whipAskReady}
                    className="px-2.5 py-1 rounded-lg text-[11px] font-medium bg-[#c85a2a] text-white hover:bg-[#a84a22] disabled:opacity-40 transition-colors"
                  >
                    Execute
                  </button>
                  <button
                    onClick={() => { setWhipPlanCode(null); setWhipPlanEditing(false); setWhipAskFields([]); setWhipAskReady(false); }}
                    className="px-2.5 py-1 rounded-lg text-[11px] font-medium bg-[rgba(61,57,41,0.04)] text-[rgba(61,57,41,0.3)] hover:bg-[rgba(61,57,41,0.08)] transition-colors"
                  >
                    ✕
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
                  className="w-full bg-[rgba(61,57,41,0.06)] border border-[rgba(61,57,41,0.12)] rounded-lg p-2.5 text-[11px] font-mono text-[rgba(61,57,41,0.85)] resize-none focus:outline-none focus:ring-1 focus:ring-[rgba(200,90,42,0.4)] leading-relaxed"
                  spellCheck={false}
                />
              ) : (
                <pre className="text-[11px] font-mono text-[rgba(61,57,41,0.7)] bg-[rgba(61,57,41,0.06)] border border-[rgba(61,57,41,0.08)] rounded-lg p-2.5 max-h-36 overflow-y-auto whitespace-pre-wrap break-words leading-relaxed">{whipPlanCode}</pre>
              )}
            </div>
          )}
          {/* Input — inside left column */}
          <div className="border-t border-[rgba(61,57,41,0.08)] px-4 py-3">
            {/* Skill Picker */}
            {showSkillPicker && (showNewCommand || showPlanCommand || filteredSkills.length > 0) && (
              <div className="mb-2 rounded-xl border border-[rgba(61,57,41,0.12)] bg-[#ece5d8] overflow-hidden shadow-xl">
                <div className="max-h-52 overflow-y-auto">
                  {showNewCommand && (
                    <button
                      onMouseDown={(e) => { e.preventDefault(); setInput("/new"); setShowSkillPicker(false); }}
                      className="w-full text-left px-3 py-2.5 flex flex-col gap-0.5 transition-colors hover:bg-[rgba(61,57,41,0.05)] border-b border-[rgba(61,57,41,0.08)]"
                    >
                      <span className="text-[13px] font-medium text-[#3d3929]">/new</span>
                      <span className="text-[11px] text-[rgba(61,57,41,0.4)]">Start a new chat session</span>
                    </button>
                  )}
                  {showPlanCommand && (
                    <button
                      onMouseDown={(e) => { e.preventDefault(); setInput("/plan "); setShowSkillPicker(false); inputRef.current?.focus(); }}
                      className="w-full text-left px-3 py-2.5 flex flex-col gap-0.5 transition-colors hover:bg-[rgba(61,57,41,0.05)] border-b border-[rgba(61,57,41,0.08)]"
                    >
                      <span className="text-[13px] font-medium text-[#3d3929]">/plan</span>
                      <span className="text-[11px] text-[rgba(61,57,41,0.4)]">Generate a Whipflow workflow from a description</span>
                    </button>
                  )}
                  {filteredSkills.map((s, i) => (
                    <button
                      key={s.filePath}
                      onMouseDown={(e) => { e.preventDefault(); selectSkill(s); }}
                      className={`w-full text-left px-3 py-2.5 flex flex-col gap-0.5 transition-colors ${
                        i === skillPickerIdx
                          ? "bg-[rgba(200,90,42,0.2)]"
                          : "hover:bg-[rgba(61,57,41,0.05)]"
                      }`}
                    >
                      <span className="text-[13px] font-medium text-[#3d3929]">{s.name}</span>
                      <span className="text-[11px] text-[rgba(61,57,41,0.4)] line-clamp-1">{s.description}</span>
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
              accept="image/*,video/*"
              multiple
              className="hidden"
              onChange={(e) => { if (e.target.files) addFiles(e.target.files); e.target.value = ""; }}
            />
            <div className="flex gap-2 items-end">
              <button
                onClick={() => fileInputRef.current?.click()}
                disabled={isStreaming || wsStatus !== "open"}
                className="flex-shrink-0 w-[42px] h-[42px] flex items-center justify-center rounded-xl border border-[rgba(61,57,41,0.12)] bg-[rgba(61,57,41,0.05)] text-[rgba(61,57,41,0.4)] hover:text-[#3d3929] hover:bg-[rgba(61,57,41,0.12)] disabled:opacity-40 transition-colors"
                title="Attach files"
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M21.44 11.05l-9.19 9.19a6 6 0 01-8.49-8.49l9.19-9.19a4 4 0 015.66 5.66l-9.2 9.19a2 2 0 01-2.83-2.83l8.49-8.48" />
                </svg>
              </button>
              <textarea
                ref={inputRef}
                rows={1}
                placeholder={wsStatus === "open" ? "Type a message… (/ for skills, /plan for workflows, ⌘↵ to send)" : `WebSocket ${wsStatus}…`}
                value={input}
                onChange={(e) => { handleInputChange(e.target.value); e.target.style.height = "auto"; e.target.style.height = Math.min(e.target.scrollHeight, 200) + "px"; }}
                onKeyDown={handleInputKeyDown}
                onPaste={handlePaste}
                disabled={isStreaming || wsStatus !== "open"}
                className="flex-1 border border-[rgba(61,57,41,0.12)] rounded-xl px-4 py-2.5 text-[13px] bg-[rgba(61,57,41,0.05)] text-[#3d3929] placeholder-[rgba(61,57,41,0.2)] focus:outline-none focus:ring-2 focus:ring-[rgba(200,90,42,0.4)] disabled:opacity-50 resize-none overflow-y-auto leading-relaxed"
                style={{ minHeight: "42px", maxHeight: "200px" }}
              />
              {isStreaming ? (
                <button onClick={handleStop} className="px-4 py-2.5 rounded-xl bg-[rgba(255,69,58,0.15)] text-red-400 text-[13px] font-medium hover:bg-[rgba(255,69,58,0.25)] transition-colors">
                  Stop
                </button>
              ) : (
                <button
                  onClick={handleSend}
                  disabled={!input.trim() || wsStatus !== "open"}
                  className="px-4 py-2.5 rounded-xl bg-[#c85a2a] text-white text-[13px] font-medium hover:bg-[#a84a22] disabled:opacity-40 transition-colors"
                >
                  Send
                </button>
              )}
            </div>
          </div>
        </div>

        {/* Right: Tabbed Panel (60%) */}
        <div className="w-[60%] border-l border-[rgba(61,57,41,0.08)] min-h-0 flex flex-col">
          {/* Tab bar */}
          <div className="flex border-b border-[rgba(61,57,41,0.08)] flex-shrink-0">
            <button
              onClick={() => setActiveTab("tools")}
              className={`px-4 py-2.5 text-[13px] font-medium transition-colors ${
                activeTab === "tools"
                  ? "text-[#3d3929] border-b-2 border-[#c85a2a]"
                  : "text-[rgba(61,57,41,0.4)] hover:text-[rgba(61,57,41,0.55)]"
              }`}
            >
              Tool Activity
            </button>
            <button
              onClick={() => previewHtml && setActiveTab("preview")}
              className={`px-4 py-2.5 text-[13px] font-medium transition-colors ${
                activeTab === "preview"
                  ? "text-[#3d3929] border-b-2 border-[#c85a2a]"
                  : previewHtml
                  ? "text-[rgba(61,57,41,0.4)] hover:text-[rgba(61,57,41,0.55)]"
                  : "text-[rgba(61,57,41,0.3)] cursor-not-allowed"
              }`}
            >
              HTML Preview
            </button>
          </div>
          {/* Tab content */}
          <div className="flex-1 min-h-0">
            {activeTab === "tools" ? (
              <ToolPanel executions={toolExecutions} onRetryFromSession={handleRetryFromSession} onConfirmPreview={handleConfirmWhipflow} onEditPreview={handleEditPreview} whipflowArgs={lastWhipflowArgs} />
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
