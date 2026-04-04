import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { api } from "../api";

export default function CanvasPage() {
  const navigate = useNavigate();
  const [files, setFiles] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getCanvasList()
      .then(setFiles)
      .catch(() => setFiles([]))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="page"><div className="empty">Loading...</div></div>;

  return (
    <div className="page">
      <h1 className="page-header">Canvas</h1>
      {files.length === 0 ? (
        <div className="empty">No canvas files</div>
      ) : (
        files.map((name) => (
          <div
            key={name}
            className="card"
            onClick={() => navigate(`/canvas/${name}`)}
            style={{ cursor: "pointer" }}
          >
            <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" strokeWidth="2">
                <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
                <polyline points="14 2 14 8 20 8" />
              </svg>
              <span className="card-title" style={{ margin: 0 }}>{name}</span>
            </div>
          </div>
        ))
      )}
    </div>
  );
}
