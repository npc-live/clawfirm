import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { api, type AgentInfo, type SessionInfo } from "../api";

export default function ChatsPage() {
  const navigate = useNavigate();
  const [agents, setAgents] = useState<AgentInfo[]>([]);
  const [selectedAgent, setSelectedAgent] = useState<string | null>(null);
  const [sessions, setSessions] = useState<SessionInfo[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getAgents().then((a) => {
      setAgents(a);
      if (a.length > 0) {
        setSelectedAgent(a[0].name);
      }
      setLoading(false);
    }).catch(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (!selectedAgent) return;
    api.getSessions(selectedAgent).then(setSessions).catch(() => setSessions([]));
  }, [selectedAgent]);

  const openChat = (agentName: string, sessionId: string) => {
    navigate(`/chats/${agentName}/${sessionId}`);
  };

  const newChat = () => {
    if (!selectedAgent) return;
    const id = `mobile-${Date.now()}`;
    navigate(`/chats/${selectedAgent}/${id}`);
  };

  const formatTime = (ms: number) => {
    if (!ms) return "";
    const d = new Date(ms);
    const now = new Date();
    if (d.toDateString() === now.toDateString()) {
      return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
    }
    return d.toLocaleDateString([], { month: "short", day: "numeric" });
  };

  if (loading) return <div className="page"><div className="empty">Loading...</div></div>;

  return (
    <div className="page">
      <h1 className="page-header">Chats</h1>

      {/* Agent tabs */}
      {agents.length > 1 && (
        <div style={{ display: "flex", gap: 8, marginBottom: 16, overflowX: "auto" }}>
          {agents.map((a) => (
            <button
              key={a.name}
              className={`btn ${selectedAgent === a.name ? "btn-primary" : "btn-outline"}`}
              style={{ padding: "6px 14px", fontSize: 13, whiteSpace: "nowrap" }}
              onClick={() => setSelectedAgent(a.name)}
            >
              {a.name}
            </button>
          ))}
        </div>
      )}

      {/* Selected agent info */}
      {selectedAgent && (
        <div className="card" style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <div className="card-title">{selectedAgent}</div>
            <div className="card-subtitle">
              {agents.find((a) => a.name === selectedAgent)?.model ?? ""}
            </div>
          </div>
          <button className="btn btn-primary" style={{ padding: "8px 16px" }} onClick={newChat}>
            + New
          </button>
        </div>
      )}

      {/* Session list */}
      {sessions.length === 0 ? (
        <div className="empty">No sessions yet</div>
      ) : (
        sessions.map((s) => (
          <div
            key={s.sessionKey}
            className="card"
            onClick={() => openChat(selectedAgent!, s.userId)}
            style={{ cursor: "pointer" }}
          >
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start" }}>
              <div style={{ flex: 1 }}>
                <div className="card-title" style={{ fontSize: 14 }}>
                  {s.subject || s.userId}
                </div>
                <div className="card-subtitle">
                  {s.inputTokens + s.outputTokens > 0 && (
                    <span>{(s.inputTokens + s.outputTokens).toLocaleString()} tokens</span>
                  )}
                  {s.costUsd > 0 && <span> · ${s.costUsd.toFixed(4)}</span>}
                </div>
              </div>
              <div style={{ fontSize: 12, color: "var(--text-dim)", whiteSpace: "nowrap" }}>
                {s.isActive && <span className="status-dot online" />}
                {formatTime(s.updatedAt)}
              </div>
            </div>
          </div>
        ))
      )}
    </div>
  );
}
