import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { api } from "../api";

export default function CanvasView() {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const [html, setHtml] = useState<string>("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!name) return;
    api.getCanvasContent(name)
      .then(setHtml)
      .catch(() => setHtml("<p>Failed to load canvas</p>"))
      .finally(() => setLoading(false));
  }, [name]);

  if (loading) return <div className="page"><div className="empty">Loading...</div></div>;

  return (
    <div style={{ height: "100%", display: "flex", flexDirection: "column" }}>
      <div style={{ padding: "12px 16px", borderBottom: "1px solid var(--border)", display: "flex", alignItems: "center", gap: 12 }}>
        <button className="back-btn" onClick={() => navigate("/canvas")} style={{ margin: 0 }}>
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M15 18l-6-6 6-6" />
          </svg>
        </button>
        <span style={{ fontWeight: 600, fontSize: 15 }}>{name}</span>
      </div>
      <iframe
        className="canvas-viewer"
        srcDoc={html}
        sandbox="allow-scripts"
        title={name}
        style={{ flex: 1, margin: 0, borderRadius: 0 }}
      />
    </div>
  );
}
