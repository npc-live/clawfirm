import { useState, useEffect, useCallback } from "react";
import { Routes, Route, Navigate, useLocation } from "react-router-dom";
import { loadDevices, getActiveDeviceUrl, connectDevice } from "./store";
import { api, type Device } from "./api";
import BottomNav from "./components/BottomNav";
import ScanPage from "./pages/ScanPage";
import ChatsPage from "./pages/ChatsPage";
import ChatView from "./pages/ChatView";
import CanvasPage from "./pages/CanvasPage";
import CanvasView from "./pages/CanvasView";
import ChannelsPage from "./pages/ChannelsPage";

export default function App() {
  const [connected, setConnected] = useState(false);
  const [loading, setLoading] = useState(true);

  // Try to auto-reconnect to the last device on launch.
  useEffect(() => {
    (async () => {
      try {
        const activeUrl = await getActiveDeviceUrl();
        if (!activeUrl) {
          setLoading(false);
          return;
        }
        const devices = await loadDevices();
        const device = devices.find((d) => d.url === activeUrl);
        if (device) {
          const ok = await connectDevice(device);
          setConnected(ok);
        }
      } catch {
        // ignore
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const handleConnect = useCallback((device: Device) => {
    api.setDevice(device);
    setConnected(true);
  }, []);

  const handleDisconnect = useCallback(() => {
    api.setDevice(null);
    setConnected(false);
  }, []);

  if (loading) {
    return (
      <div className="app-layout">
        <div className="page" style={{ display: "flex", alignItems: "center", justifyContent: "center" }}>
          <p style={{ color: "var(--text-dim)" }}>Loading...</p>
        </div>
      </div>
    );
  }

  if (!connected) {
    return <ScanPage onConnect={handleConnect} />;
  }

  return <MainLayout onDisconnect={handleDisconnect} />;
}

function MainLayout({ onDisconnect }: { onDisconnect: () => void }) {
  const location = useLocation();
  // Hide bottom nav in chat view (full-screen chat)
  const isChatView = /^\/chats\/[^/]+\/[^/]+/.test(location.pathname);

  return (
    <div className="app-layout">
      <Routes>
        <Route path="/" element={<Navigate to="/chats" replace />} />
        <Route path="/chats" element={<ChatsPage />} />
        <Route path="/chats/:agentName/:sessionId" element={<ChatView />} />
        <Route path="/canvas" element={<CanvasPage />} />
        <Route path="/canvas/:name" element={<CanvasView />} />
        <Route path="/channels" element={<ChannelsPage onDisconnect={onDisconnect} />} />
        <Route path="*" element={<Navigate to="/chats" replace />} />
      </Routes>
      {!isChatView && <BottomNav />}
    </div>
  );
}
