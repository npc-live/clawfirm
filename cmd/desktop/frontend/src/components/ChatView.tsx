import { useState, useEffect, useRef, useCallback } from "react";
import ReactMarkdown from "react-markdown";
import { GetWebhookBaseURL, AbortCurrentTurn, GetHistory, GetAgentSkills, GetToolExecutions, type SkillInfo } from "../wailsjs/go/app/App";
import { useWebSocket, type WSMessage } from "../hooks/useWebSocket";
import { ToolPanel, type ToolExecution } from "./ToolPanel";
import { HtmlPreview } from "./HtmlPreview";

interface Message {
  role: "user" | "assistant";
  content: string;
  streaming?: boolean;
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
  const [activeTab, setActiveTab] = useState<"tools" | "preview">("tools");
  const bottomRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

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
      // Restore HTML preview from last assistant message containing ```html block.
      for (let i = history.length - 1; i >= 0; i--) {
        if (history[i].role === "assistant") {
          const html = extractHtmlBlock(history[i].content);
          if (html) {
            setPreviewHtml(html);
            setActiveTab("preview");
            break;
          }
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
        if (html) {
          setPreviewHtml(html);
          setActiveTab("preview");
        }
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

  function handleSend() {
    const text = input.trim();
    if (!text || isStreaming) return;
    if (text === "/new") {
      setInput("");
      setShowSkillPicker(false);
      onNewSession();
      return;
    }
    if (wsStatus !== "open") return;
    setMessages((prev) => [...prev, { role: "user", content: text }]);
    setInput("");
    setShowSkillPicker(false);
    setIsStreaming(true);
    send(JSON.stringify({ type: "message", content: text }));
  }

  function handleStop() {
    AbortCurrentTurn(agentName, sessionID);
    setIsStreaming(false);
  }

  const statusColor =
    wsStatus === "open" ? "bg-emerald-400" :
    wsStatus === "connecting" ? "bg-amber-400" :
    "bg-red-400";

  return (
    <div className="flex flex-col h-screen bg-[rgb(30,30,28)]">
      {/* Header */}
      <header className="flex items-center gap-3 px-4 py-3 border-b border-[rgba(255,255,255,0.08)]">
        <button onClick={onBack} className="text-[rgba(255,255,255,0.4)] hover:text-[rgb(240,237,229)] transition-colors">←</button>
        <h1 className="font-semibold text-[rgb(240,237,229)] text-[15px] tracking-tight">{agentName}</h1>
        <span className="flex items-center gap-1.5 text-[11px] text-[rgba(255,255,255,0.4)] ml-auto">
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
              <div className="text-center text-[rgba(255,255,255,0.2)] mt-16 text-[13px]">Start a conversation</div>
            )}
            {messages.map((msg, i) => (
              <div key={i} className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}>
                <div
                  className={`max-w-[75%] min-w-0 overflow-hidden rounded-2xl px-4 py-3 text-[13px] ${
                    msg.role === "user"
                      ? "bg-[#2688f9] text-white"
                      : "bg-[rgba(255,255,255,0.05)] text-[rgba(240,237,229,0.85)] border border-[rgba(255,255,255,0.08)]"
                  }`}
                >
                  {msg.role === "assistant" ? (
                    <div className="prose-pre:overflow-x-auto prose-pre:max-w-full [&_pre]:overflow-x-auto [&_pre]:max-w-full [&_code]:break-all [&_pre_code]:break-normal">
                      <ReactMarkdown>{msg.content}</ReactMarkdown>
                    </div>
                  ) : (
                    msg.content
                  )}
                  {msg.streaming && (
                    <span className="inline-block w-1 h-4 ml-1 bg-current opacity-70 animate-pulse" />
                  )}
                </div>
              </div>
            ))}
            <div ref={bottomRef} />
          </div>
          {/* Input — inside left column */}
          <div className="border-t border-[rgba(255,255,255,0.08)] px-4 py-3">
            {/* Skill Picker */}
            {showSkillPicker && (showNewCommand || filteredSkills.length > 0) && (
              <div className="mb-2 rounded-xl border border-[rgba(255,255,255,0.1)] bg-[rgb(40,40,38)] overflow-hidden shadow-xl">
                <div className="max-h-52 overflow-y-auto">
                  {showNewCommand && (
                    <button
                      onMouseDown={(e) => { e.preventDefault(); setInput("/new"); setShowSkillPicker(false); }}
                      className="w-full text-left px-3 py-2.5 flex flex-col gap-0.5 transition-colors hover:bg-[rgba(255,255,255,0.05)] border-b border-[rgba(255,255,255,0.06)]"
                    >
                      <span className="text-[13px] font-medium text-[rgb(240,237,229)]">/new</span>
                      <span className="text-[11px] text-[rgba(255,255,255,0.4)]">Start a new chat session</span>
                    </button>
                  )}
                  {filteredSkills.map((s, i) => (
                    <button
                      key={s.filePath}
                      onMouseDown={(e) => { e.preventDefault(); selectSkill(s); }}
                      className={`w-full text-left px-3 py-2.5 flex flex-col gap-0.5 transition-colors ${
                        i === skillPickerIdx
                          ? "bg-[rgba(38,136,249,0.2)]"
                          : "hover:bg-[rgba(255,255,255,0.05)]"
                      }`}
                    >
                      <span className="text-[13px] font-medium text-[rgb(240,237,229)]">{s.name}</span>
                      <span className="text-[11px] text-[rgba(255,255,255,0.4)] line-clamp-1">{s.description}</span>
                    </button>
                  ))}
                </div>
              </div>
            )}
            <div className="flex gap-2 items-end">
              <textarea
                ref={inputRef}
                rows={1}
                placeholder={wsStatus === "open" ? "Type a message… (/ for skills, ⌘↵ to send)" : `WebSocket ${wsStatus}…`}
                value={input}
                onChange={(e) => { handleInputChange(e.target.value); e.target.style.height = "auto"; e.target.style.height = Math.min(e.target.scrollHeight, 200) + "px"; }}
                onKeyDown={handleInputKeyDown}
                disabled={isStreaming || wsStatus !== "open"}
                className="flex-1 border border-[rgba(255,255,255,0.1)] rounded-xl px-4 py-2.5 text-[13px] bg-[rgba(255,255,255,0.05)] text-[rgb(240,237,229)] placeholder-[rgba(255,255,255,0.25)] focus:outline-none focus:ring-2 focus:ring-[rgba(38,136,249,0.4)] disabled:opacity-50 resize-none overflow-y-auto leading-relaxed"
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
                  className="px-4 py-2.5 rounded-xl bg-[#2688f9] text-white text-[13px] font-medium hover:bg-[#1a7ae8] disabled:opacity-40 transition-colors"
                >
                  Send
                </button>
              )}
            </div>
          </div>
        </div>

        {/* Right: Tabbed Panel (60%) */}
        <div className="w-[60%] border-l border-[rgba(255,255,255,0.08)] min-h-0 flex flex-col">
          {/* Tab bar */}
          <div className="flex border-b border-[rgba(255,255,255,0.08)] flex-shrink-0">
            <button
              onClick={() => setActiveTab("tools")}
              className={`px-4 py-2.5 text-[13px] font-medium transition-colors ${
                activeTab === "tools"
                  ? "text-[rgb(240,237,229)] border-b-2 border-[#2688f9]"
                  : "text-[rgba(255,255,255,0.4)] hover:text-[rgba(255,255,255,0.6)]"
              }`}
            >
              Tool Activity
            </button>
            <button
              onClick={() => previewHtml && setActiveTab("preview")}
              className={`px-4 py-2.5 text-[13px] font-medium transition-colors ${
                activeTab === "preview"
                  ? "text-[rgb(240,237,229)] border-b-2 border-[#2688f9]"
                  : previewHtml
                  ? "text-[rgba(255,255,255,0.4)] hover:text-[rgba(255,255,255,0.6)]"
                  : "text-[rgba(255,255,255,0.2)] cursor-not-allowed"
              }`}
            >
              HTML Preview
            </button>
          </div>
          {/* Tab content */}
          <div className="flex-1 min-h-0">
            {activeTab === "tools" ? (
              <ToolPanel executions={toolExecutions} />
            ) : previewHtml ? (
              <div className="h-full p-3">
                <HtmlPreview html={previewHtml} />
              </div>
            ) : (
              <div className="flex items-center justify-center h-full text-[rgba(255,255,255,0.2)] text-[13px]">
                No HTML preview available
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
