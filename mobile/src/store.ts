import { load, type Store } from "@tauri-apps/plugin-store";
import { api, type Device } from "./api";

// ─── Types ────────────────────────────────────────────────────────────────────

export interface SavedDevice extends Device {
  addedAt: number; // epoch ms
}

// ─── Device Store ─────────────────────────────────────────────────────────────

let store: Store | null = null;

async function getStore(): Promise<Store> {
  if (!store) {
    store = await load("devices.json");
  }
  return store;
}

export async function loadDevices(): Promise<SavedDevice[]> {
  const s = await getStore();
  return (await s.get<SavedDevice[]>("devices")) ?? [];
}

export async function saveDevice(device: Device): Promise<void> {
  const s = await getStore();
  const devices = await loadDevices();

  // Update existing or add new.
  const idx = devices.findIndex((d) => d.url === device.url);
  const saved: SavedDevice = { ...device, addedAt: Date.now() };
  if (idx >= 0) {
    devices[idx] = saved;
  } else {
    devices.push(saved);
  }
  await s.set("devices", devices);
}

export async function removeDevice(url: string): Promise<void> {
  const s = await getStore();
  const devices = await loadDevices();
  await s.set(
    "devices",
    devices.filter((d) => d.url !== url),
  );
}

export async function getActiveDeviceUrl(): Promise<string | null> {
  const s = await getStore();
  return (await s.get<string>("activeDeviceUrl")) ?? null;
}

export async function setActiveDeviceUrl(url: string | null): Promise<void> {
  const s = await getStore();
  await s.set("activeDeviceUrl", url);
}

// ─── Connection ───────────────────────────────────────────────────────────────

export async function connectDevice(device: Device): Promise<boolean> {
  api.setDevice(device);
  try {
    await api.getStatus();
    await setActiveDeviceUrl(device.url);
    return true;
  } catch {
    api.setDevice(null);
    return false;
  }
}

export function disconnectDevice(): void {
  api.setDevice(null);
  setActiveDeviceUrl(null);
}

// ─── Parse QR ─────────────────────────────────────────────────────────────────

/**
 * Parse a QR code URL like "http://192.168.1.5:12345/remote/?token=abc123"
 * into a Device object.
 */
export function parseQRUrl(raw: string): Device | null {
  try {
    const u = new URL(raw);
    const token = u.searchParams.get("token");
    if (!token) return null;

    // Base URL is scheme + host (no path).
    const baseUrl = `${u.protocol}//${u.host}`;
    const name = u.hostname;

    return { name, url: baseUrl, token };
  } catch {
    return null;
  }
}
