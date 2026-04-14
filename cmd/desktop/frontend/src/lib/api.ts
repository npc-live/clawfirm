// HTTP API client for gateway Web UI mode (non-Wails environment).
// Mirrors the subset of wailsjs/go/app/App exports needed for chat.

import type { ChannelInfo, HistoryMessage } from "../wailsjs/go/app/App";

const BASE = "/api";

async function fetchJSON<T>(path: string): Promise<T> {
  const res = await fetch(BASE + path);
  if (!res.ok) throw new Error(`API ${path}: ${res.status}`);
  return res.json();
}

export async function IsFirstRun(): Promise<boolean> {
  const data = await fetchJSON<{ firstRun: boolean }>("/system/first-run");
  return data.firstRun;
}

export async function GetChannels(): Promise<ChannelInfo[]> {
  return fetchJSON<ChannelInfo[]>("/channels");
}

export async function GetChatSessions(agentName: string): Promise<string[]> {
  return fetchJSON<string[]>(`/channels/${encodeURIComponent(agentName)}/sessions`);
}

export async function GetHistory(channelID: string, userID: string): Promise<HistoryMessage[]> {
  // channelID is "webchat/<agentName>", extract agentName
  const agentName = channelID.replace(/^webchat\//, "");
  return fetchJSON<HistoryMessage[]>(
    `/channels/${encodeURIComponent(agentName)}/sessions/${encodeURIComponent(userID)}/history`
  );
}

export async function GetWebhookBaseURL(): Promise<string> {
  const data = await fetchJSON<{ url: string }>("/webhook/base-url");
  return data.url;
}

export async function GetVersion(): Promise<string> {
  const data = await fetchJSON<{ version: string }>("/system/version");
  return data.version;
}
