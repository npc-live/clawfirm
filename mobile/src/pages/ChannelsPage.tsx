import { useState, useEffect } from "react";
import { api, type ChannelStatus } from "../api";
import { disconnectDevice } from "../store";

interface Props {
  onDisconnect: () => void;
}

export default function ChannelsPage({ onDisconnect }: Props) {
  const [channels, setChannels] = useState<ChannelStatus[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getChannels()
      .then(setChannels)
      .catch(() => setChannels([]))
      .finally(() => setLoading(false));
  }, []);

  const handleDisconnect = () => {
    disconnectDevice();
    onDisconnect();
  };

  return (
    <div className="page">
      <h1 className="page-header">Channels</h1>

      {loading ? (
        <div className="empty">Loading...</div>
      ) : channels.length === 0 ? (
        <div className="empty">No channels active</div>
      ) : (
        channels.map((ch) => (
          <div key={ch.name} className="card">
            <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
              <div className="card-title" style={{ textTransform: "capitalize" }}>{ch.name}</div>
              <div>
                <span
                  className={`status-dot ${ch.status === "connected" ? "online" : "offline"}`}
                />
                {ch.status}
              </div>
            </div>
          </div>
        ))
      )}

      {/* Device section */}
      <div style={{ marginTop: 24 }}>
        <p style={{ color: "var(--text-dim)", fontSize: 13, marginBottom: 8 }}>Connected Device</p>
        <div className="card">
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
            <div>
              <span className="status-dot online" />
              Connected
            </div>
            <button className="btn btn-danger" style={{ padding: "6px 14px", fontSize: 13 }} onClick={handleDisconnect}>
              Disconnect
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
