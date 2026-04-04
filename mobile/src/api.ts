import { fetch } from "@tauri-apps/plugin-http";
import WebSocket from "@tauri-apps/plugin-websocket";

// ─── Types ────────────────────────────────────────────────────────────────────

export interface Device {
  name: string;
  url: string; // base URL like "http://192.168.1.5:12345" or "https://xxx.ngrok-free.app"
  token: string;
}

export interface AgentInfo {
  name: string;
  provider: string;
  model: string;
  sessions: number;
}

export interface SessionInfo {
  sessionKey: string;
  sessionId: string;
  channelId: string;
  userId: string;
  subject: string;
  model: string;
  inputTokens: number;
  outputTokens: number;
  costUsd: number;
  createdAt: number;
  updatedAt: number;
  isActive: boolean;
}

export interface HistoryMessage {
  role: string;
  content: string;
}

export interface ChannelStatus {
  name: string;
  status: string;
}

export interface ServerStatus {
  enabled: boolean;
  lanUrl: string;
  ngrokUrl: string;
  token: string;
  port: number;
  lanIP: string;
  ngrokOn: boolean;
  clients: number;
}

// ─── WebSocket message types ──────────────────────────────────────────────────

export interface WSClientMessage {
  type: "message" | "run_tool";
  content?: string;
  images?: { data: string; mime: string }[];
  tool_name?: string;
  tool_id?: string;
  tool_args?: Record<string, unknown>;
}

export interface WSServerMessage {
  type: "delta" | "done" | "error" | "tool_start" | "tool_update" | "tool_end";
  content?: string;
  stop_reason?: string;
  timestamp?: number;
  tool_call_id?: string;
  tool_name?: string;
  tool_args?: Record<string, unknown>;
  tool_result?: unknown;
  tool_is_error?: boolean;
  partial_result?: unknown;
}

// ─── HTTP Client ──────────────────────────────────────────────────────────────

class ApiClient {
  private device: Device | null = null;

  setDevice(device: Device | null) {
    this.device = device;
  }

  private async get<T>(path: string): Promise<T> {
    if (!this.device) throw new Error("No device connected");
    const url = `${this.device.url}/remote/api${path}`;
    const resp = await fetch(url, {
      method: "GET",
      headers: { "X-Remote-Token": this.device.token },
    });
    if (!resp.ok) throw new Error(`HTTP ${resp.status}: ${await resp.text()}`);
    return resp.json() as Promise<T>;
  }

  async getStatus(): Promise<ServerStatus> {
    return this.get("/status");
  }

  async getAgents(): Promise<AgentInfo[]> {
    return this.get("/agents");
  }

  async getSessions(agentName: string): Promise<SessionInfo[]> {
    return this.get(`/agents/${agentName}/sessions`);
  }

  async getHistory(agentName: string, sessionId: string): Promise<HistoryMessage[]> {
    return this.get(`/agents/${agentName}/sessions/${sessionId}/history`);
  }

  async getCanvasList(): Promise<string[]> {
    return this.get("/canvas");
  }

  async getCanvasContent(name: string): Promise<string> {
    if (!this.device) throw new Error("No device connected");
    const url = `${this.device.url}/remote/api/canvas/${name}`;
    const resp = await fetch(url, {
      method: "GET",
      headers: { "X-Remote-Token": this.device.token },
    });
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    return resp.text();
  }

  async getChannels(): Promise<ChannelStatus[]> {
    return this.get("/channels");
  }

  // ─── WebSocket ────────────────────────────────────────────────────────────

  async connectWS(
    agentName: string,
    sessionId: string,
    onMessage: (msg: WSServerMessage) => void,
    onClose?: () => void,
  ): Promise<{ send: (msg: WSClientMessage) => void; close: () => void }> {
    if (!this.device) throw new Error("No device connected");

    const baseUrl = this.device.url.replace(/^http/, "ws");
    const url = `${baseUrl}/remote/ws/${agentName}/${sessionId}?token=${this.device.token}`;

    const ws = await WebSocket.connect(url);

    ws.addListener((msg) => {
      if (typeof msg.data === "string") {
        try {
          const parsed = JSON.parse(msg.data) as WSServerMessage;
          onMessage(parsed);
        } catch {
          // ignore malformed messages
        }
      }
      if (msg.type === "Close") {
        onClose?.();
      }
    });

    return {
      send: (msg: WSClientMessage) => {
        ws.send(JSON.stringify(msg));
      },
      close: () => {
        ws.disconnect();
      },
    };
  }
}

export const api = new ApiClient();
