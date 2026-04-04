import { useState, useEffect, useRef, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { api, type HistoryMessage, type WSServerMessage, type WSClientMessage } from "../api";

type DisplayItem =
  | { kind: "message"; role: string; content: string }
  | { kind: "tool"; id: string; name: string; status: "running" | "done" | "failed" };

export default function ChatView() {
  const { agentName, sessionId } = useParams<{ agentName: string; sessionId: string }>();
  const navigate = useNavigate();
  const [items, setItems] = useState<DisplayItem[]>([]);
  const [input, setInput] = useState("");
  const [streaming, setStreaming] = useState(false);
  const [streamText, setStreamText] = useState("");
  const messagesEnd = useRef<HTMLDivElement>(null);
  const wsRef = useRef<{ send: (msg: WSClientMessage) => void; close: () => void } | null>(null);
  const streamTextRef = useRef("");

  const scrollToBottom = useCallback(() => {
    messagesEnd.current?.scrollIntoView({ behavior: "smooth" });
  }, []);

  // Flush any accumulated streaming text into a message item.
  const flushStream = useCallback(() => {
    const text = streamTextRef.current;
    if (text) {
      setItems((prev) => [...prev, { kind: "message", role: "assistant", content: text }]);
      streamTextRef.current = "";
      setStreamText("");
    }
  }, []);

  // Load history + connect WebSocket.
  useEffect(() => {
    if (!agentName || !sessionId) return;

    api.getHistory(agentName, sessionId).then((history: HistoryMessage[]) => {
      setItems(history.map((h) => ({ kind: "message" as const, role: h.role, content: h.content })));
      setTimeout(scrollToBottom, 100);
    }).catch(() => {});

    let closed = false;
    api.connectWS(
      agentName,
      sessionId,
      (msg: WSServerMessage) => {
        switch (msg.type) {
          case "delta":
            setStreaming(true);
            streamTextRef.current += msg.content ?? "";
            setStreamText(streamTextRef.current);
            scrollToBottom();
            break;

          case "tool_start":
            // Flush any text before the tool call.
            if (streamTextRef.current) {
              const text = streamTextRef.current;
              streamTextRef.current = "";
              setStreamText("");
              setItems((prev) => [
                ...prev,
                { kind: "message", role: "assistant", content: text },
                { kind: "tool", id: msg.tool_call_id ?? "", name: msg.tool_name ?? "", status: "running" },
              ]);
            } else {
              setItems((prev) => [
                ...prev,
                { kind: "tool", id: msg.tool_call_id ?? "", name: msg.tool_name ?? "", status: "running" },
              ]);
            }
            scrollToBottom();
            break;

          case "tool_end":
            setItems((prev) =>
              prev.map((item) =>
                item.kind === "tool" && item.id === msg.tool_call_id
                  ? { ...item, status: msg.tool_is_error ? "failed" : "done" }
                  : item,
              ),
            );
            break;

          case "done":
            // Flush remaining streamed text.
            if (streamTextRef.current) {
              const text = streamTextRef.current;
              streamTextRef.current = "";
              setStreamText("");
              setItems((prev) => [...prev, { kind: "message", role: "assistant", content: text }]);
            }
            setStreaming(false);
            scrollToBottom();
            break;

          case "error":
            streamTextRef.current = "";
            setStreamText("");
            setStreaming(false);
            if (msg.content) {
              setItems((prev) => [...prev, { kind: "message", role: "assistant", content: `Error: ${msg.content}` }]);
            }
            break;
        }
      },
      () => {
        if (!closed) {
          // WebSocket closed.
        }
      },
    ).then((ws) => {
      wsRef.current = ws;
    });

    return () => {
      closed = true;
      wsRef.current?.close();
      wsRef.current = null;
    };
  }, [agentName, sessionId, scrollToBottom, flushStream]);

  const sendMessage = () => {
    const text = input.trim();
    if (!text || !wsRef.current) return;
    setItems((prev) => [...prev, { kind: "message", role: "user", content: text }]);
    setInput("");
    wsRef.current.send({ type: "message", content: text });
    scrollToBottom();
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  return (
    <div className="chat-fullscreen">
      {/* Header */}
      <div className="chat-header">
        <button className="back-btn" onClick={() => navigate("/chats")} style={{ margin: 0 }}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M15 18l-6-6 6-6" />
          </svg>
        </button>
        <div>
          <div style={{ fontWeight: 600, fontSize: 15 }}>{agentName}</div>
          <div style={{ fontSize: 12, color: "var(--text-dim)" }}>{sessionId}</div>
        </div>
      </div>

      {/* Messages + tool events (interleaved) */}
      <div className="chat-messages">
        {items.map((item, i) =>
          item.kind === "message" ? (
            <div key={i} className={`message ${item.role}`}>
              <div className="message-role">{item.role}</div>
              <div className="message-bubble">{item.content}</div>
            </div>
          ) : (
            <div key={item.id || i} className={`tool-event tool-${item.status}`}>
              <span className="tool-name">{item.name}</span>
              {item.status === "running" && <span className="tool-spinner" />}
              {item.status === "done" && <span className="tool-status-done"> done</span>}
              {item.status === "failed" && <span className="tool-status-failed"> failed</span>}
            </div>
          ),
        )}

        {/* Streaming text */}
        {streaming && streamText && (
          <div className="message assistant">
            <div className="message-role">assistant</div>
            <div className="message-bubble">{streamText}<span className="cursor-blink">|</span></div>
          </div>
        )}
        <div ref={messagesEnd} />
      </div>

      {/* Input bar */}
      <div className="chat-input-bar">
        <input
          className="input"
          placeholder="Message..."
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={streaming}
        />
        <button className="btn btn-primary" onClick={sendMessage} disabled={streaming || !input.trim()}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <line x1="22" y1="2" x2="11" y2="13" />
            <polygon points="22 2 15 22 11 13 2 9 22 2" />
          </svg>
        </button>
      </div>
    </div>
  );
}
