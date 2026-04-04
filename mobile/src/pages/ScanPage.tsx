import { useState, useEffect, useRef, useCallback } from "react";
import { scan, cancel, checkPermissions, requestPermissions, Format } from "@tauri-apps/plugin-barcode-scanner";
import jsQR from "jsqr";
import {
  loadDevices,
  saveDevice,
  removeDevice,
  connectDevice,
  parseQRUrl,
  type SavedDevice,
} from "../store";
import type { Device } from "../api";

interface Props {
  onConnect: (device: Device) => void;
}

// Android WebView supports getUserMedia; iOS WKWebView does not.
const canUseWebCamera = typeof navigator !== "undefined"
  && !!navigator.mediaDevices
  && !!navigator.mediaDevices.getUserMedia;

export default function ScanPage({ onConnect }: Props) {
  const [devices, setDevices] = useState<SavedDevice[]>([]);
  const [showManual, setShowManual] = useState(false);
  const [pasteUrl, setPasteUrl] = useState("");
  const [error, setError] = useState("");
  const [scanning, setScanning] = useState(false);

  const videoRef = useRef<HTMLVideoElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const rafRef = useRef<number>(0);

  useEffect(() => {
    loadDevices().then(setDevices);
    return () => stopCamera();
  }, []);

  const stopCamera = useCallback(() => {
    cancelAnimationFrame(rafRef.current);
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((t) => t.stop());
      streamRef.current = null;
    }
  }, []);

  // ─── Scan: jsQR (Android) or native plugin (iOS) ───────────────────────
  const handleScan = async () => {
    setError("");

    if (canUseWebCamera) {
      await handleScanJsQR();
    } else {
      await handleScanNative();
    }
  };

  // Android: getUserMedia + jsQR
  const handleScanJsQR = async () => {
    setScanning(true);
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: "environment", width: { ideal: 1280 }, height: { ideal: 720 } },
      });
      streamRef.current = stream;

      const video = videoRef.current!;
      video.srcObject = stream;
      await video.play();

      const canvas = canvasRef.current!;
      const ctx = canvas.getContext("2d", { willReadFrequently: true })!;

      const tick = () => {
        if (!streamRef.current) return;
        if (video.readyState === video.HAVE_ENOUGH_DATA) {
          canvas.width = video.videoWidth;
          canvas.height = video.videoHeight;
          ctx.drawImage(video, 0, 0);
          const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
          const code = jsQR(imageData.data, imageData.width, imageData.height, {
            inversionAttempts: "dontInvert",
          });
          if (code?.data) {
            stopCamera();
            setScanning(false);
            handleDetected(code.data);
            return;
          }
        }
        rafRef.current = requestAnimationFrame(tick);
      };
      rafRef.current = requestAnimationFrame(tick);
    } catch (e) {
      setScanning(false);
      setError("Camera error: " + String(e));
    }
  };

  // iOS: native barcode-scanner plugin (AVFoundation)
  const handleScanNative = async () => {
    try {
      let perm = await checkPermissions();
      if (perm !== "granted") {
        perm = await requestPermissions();
        if (perm !== "granted") {
          setError("Camera permission denied");
          return;
        }
      }

      // Make WebView transparent so native camera shows through
      document.documentElement.classList.add("scanner-active");
      setScanning(true);

      const result = await scan({ formats: [Format.QRCode], windowed: false });

      document.documentElement.classList.remove("scanner-active");
      setScanning(false);

      if (result?.content) {
        handleDetected(result.content);
      }
    } catch (e) {
      document.documentElement.classList.remove("scanner-active");
      setScanning(false);
      setError("Scan error: " + String(e));
    }
  };

  const handleDetected = async (data: string) => {
    const device = parseQRUrl(data);
    if (!device) {
      setError("Invalid QR code: " + data);
      return;
    }
    await tryConnect(device);
  };

  const handleCancelScan = async () => {
    if (canUseWebCamera) {
      stopCamera();
    } else {
      try { await cancel(); } catch { /* ignore */ }
      document.documentElement.classList.remove("scanner-active");
    }
    setScanning(false);
  };

  const handlePasteConnect = async () => {
    setError("");
    const raw = pasteUrl.trim();
    if (!raw) { setError("Paste the URL from desktop"); return; }
    const device = parseQRUrl(raw);
    if (!device) { setError("Invalid URL — paste the full link from desktop"); return; }
    await tryConnect(device);
  };

  const tryConnect = async (device: Device) => {
    setError("");
    const ok = await connectDevice(device);
    if (ok) {
      await saveDevice(device);
      setDevices(await loadDevices());
      onConnect(device);
    } else {
      setError("Cannot connect — check URL and try again");
    }
  };

  const handleReconnect = async (device: SavedDevice) => {
    await tryConnect(device);
  };

  const handleRemove = async (url: string) => {
    await removeDevice(url);
    setDevices(await loadDevices());
  };

  // ─── Scanning view (Android only — jsQR with live camera) ──────────────
  if (scanning && canUseWebCamera) {
    return (
      <div style={{
        position: "fixed", inset: 0,
        background: "#000", zIndex: 9999,
        display: "flex", flexDirection: "column",
        alignItems: "center", justifyContent: "center",
      }}>
        <video
          ref={videoRef}
          style={{
            position: "absolute", inset: 0,
            width: "100%", height: "100%",
            objectFit: "cover",
          }}
          playsInline
          muted
        />
        <canvas ref={canvasRef} style={{ display: "none" }} />
        <div style={{
          position: "relative", zIndex: 1,
          display: "flex", flexDirection: "column",
          alignItems: "center",
        }}>
          <div style={{
            width: 250, height: 250,
            border: "3px solid rgba(255,255,255,0.8)",
            borderRadius: 24,
          }} />
          <p style={{ color: "#fff", marginTop: 24, fontSize: 16, textShadow: "0 1px 4px rgba(0,0,0,0.8)" }}>
            Point at QR code
          </p>
          <button
            onClick={handleCancelScan}
            style={{
              marginTop: 32, padding: "12px 48px",
              background: "rgba(0,0,0,0.5)",
              color: "#fff", border: "1px solid rgba(255,255,255,0.4)",
              borderRadius: 12, fontSize: 16, cursor: "pointer",
            }}
          >
            Cancel
          </button>
        </div>
      </div>
    );
  }

  // ─── iOS native scanning — show cancel overlay ─────────────────────────
  if (scanning && !canUseWebCamera) {
    return (
      <div style={{
        position: "fixed", inset: 0, zIndex: 9999,
        display: "flex", flexDirection: "column",
        alignItems: "center", justifyContent: "flex-end",
        paddingBottom: 80,
      }}>
        <button
          onClick={handleCancelScan}
          style={{
            padding: "14px 56px",
            background: "rgba(0,0,0,0.6)",
            color: "#fff", border: "1px solid rgba(255,255,255,0.4)",
            borderRadius: 12, fontSize: 17, cursor: "pointer",
            backdropFilter: "blur(8px)",
          }}
        >
          Cancel
        </button>
      </div>
    );
  }

  return (
    <div className="app-layout">
      <div className="page">
        <h1 className="page-header">clawfirm</h1>

        {/* Scan area */}
        <div className="scan-area">
          <div className="scan-icon">
            <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5">
              <rect x="3" y="3" width="7" height="7" rx="1" />
              <rect x="14" y="3" width="7" height="7" rx="1" />
              <rect x="3" y="14" width="7" height="7" rx="1" />
              <rect x="14" y="14" width="3" height="3" />
              <path d="M21 14v3a1 1 0 01-1 1h-3" />
              <path d="M14 21h3a1 1 0 001-1v-3" />
            </svg>
          </div>
          <p style={{ marginBottom: 16, color: "var(--text-dim)" }}>
            Scan QR code from clawfirm desktop
          </p>
          <button className="btn btn-primary btn-full" onClick={handleScan}>
            Scan QR Code
          </button>
          <button
            className="btn btn-outline btn-full"
            style={{ marginTop: 8 }}
            onClick={() => setShowManual(!showManual)}
          >
            Paste URL
          </button>
        </div>

        {/* Paste URL input */}
        {showManual && (
          <div className="card">
            <input
              className="input"
              placeholder="https://xxx.ngrok-free.app/remote/?token=..."
              value={pasteUrl}
              onChange={(e) => setPasteUrl(e.target.value)}
            />
            <button
              className="btn btn-primary btn-full"
              style={{ marginTop: 8 }}
              onClick={handlePasteConnect}
            >
              Connect
            </button>
          </div>
        )}

        {/* Error */}
        {error && (
          <div className="card" style={{ borderLeft: "3px solid var(--red)" }}>
            <p style={{ color: "var(--red)", fontSize: 14 }}>{error}</p>
          </div>
        )}

        {/* Saved devices */}
        {devices.length > 0 && (
          <div className="device-list">
            <p style={{ color: "var(--text-dim)", fontSize: 13, marginBottom: 8 }}>Saved Devices</p>
            {devices.map((d) => (
              <div className="device-item" key={d.url}>
                <div className="device-info" onClick={() => handleReconnect(d)} style={{ cursor: "pointer" }}>
                  <div className="device-name">{d.name}</div>
                  <div className="device-url">{d.url}</div>
                </div>
                <button
                  className="btn btn-outline"
                  style={{ padding: "6px 12px", fontSize: 13 }}
                  onClick={() => handleRemove(d.url)}
                >
                  Remove
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
