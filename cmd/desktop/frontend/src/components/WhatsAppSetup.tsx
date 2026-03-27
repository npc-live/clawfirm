import { useEffect, useState } from "react";
import { GetWhatsAppQR, GetWhatsAppStatus, LogoutWhatsApp } from "../wailsjs/go/app/App";

const POLL_MS = 2000;

export function WhatsAppSetup() {
  const [status, setStatus] = useState<string>("disconnected");
  const [qrURL, setQRURL] = useState<string>("");

  useEffect(() => {
    let active = true;

    async function poll() {
      try {
        const s = await GetWhatsAppStatus();
        if (!active) return;
        setStatus(s);

        if (s === "qr_pending") {
          const qr = await GetWhatsAppQR();
          if (active) setQRURL(qr ?? "");
        } else {
          setQRURL("");
        }
      } catch {
        // Gateway may not be ready yet; ignore errors during polling.
      }
    }

    poll();
    const id = setInterval(poll, POLL_MS);
    return () => {
      active = false;
      clearInterval(id);
    };
  }, []);

  async function handleLogout() {
    try {
      await LogoutWhatsApp();
      setStatus("disconnected");
      setQRURL("");
    } catch (e) {
      console.error("WhatsApp logout failed:", e);
    }
  }

  return (
    <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-5">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <span className="text-lg font-semibold text-gray-900">WhatsApp</span>
          <StatusBadge status={status} />
        </div>
        {status === "connected" && (
          <button
            onClick={handleLogout}
            className="text-sm text-red-500 hover:text-red-700 transition-colors"
          >
            Disconnect
          </button>
        )}
      </div>

      {status === "qr_pending" && qrURL && (
        <div className="flex flex-col items-center gap-2">
          <p className="text-sm text-gray-500">
            Scan this QR code in WhatsApp → Linked Devices → Link a Device
          </p>
          <img
            src={qrURL}
            alt="WhatsApp QR code"
            className="w-48 h-48 rounded-lg border border-gray-200"
          />
        </div>
      )}

      {status === "qr_pending" && !qrURL && (
        <p className="text-sm text-gray-400 italic">Generating QR code…</p>
      )}

      {status === "disabled" && (
        <p className="text-sm text-gray-400 italic">
          未启用。在 config.yml 中设置 <code className="font-mono">whatsapp.enabled: true</code> 开启。
        </p>
      )}

      {(status === "disconnected" || status === "logged_out") && (
        <p className="text-sm text-gray-400 italic">
          {status === "logged_out" ? "Logged out. Restart the app to pair again." : "Waiting for gateway…"}
        </p>
      )}
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { label: string; cls: string }> = {
    connected:    { label: "Connected",    cls: "bg-green-100 text-green-700" },
    qr_pending:   { label: "Scan QR",      cls: "bg-yellow-100 text-yellow-700" },
    disconnected: { label: "Disconnected", cls: "bg-gray-100 text-gray-500" },
    logged_out:   { label: "Logged out",   cls: "bg-red-100 text-red-600" },
    disabled:     { label: "未启用",        cls: "bg-gray-100 text-gray-400" },
  };
  const { label, cls } = map[status] ?? map["disconnected"];
  return (
    <span className={`text-xs font-medium px-2 py-0.5 rounded-full ${cls}`}>
      {label}
    </span>
  );
}
