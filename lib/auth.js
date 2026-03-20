import { readFileSync, writeFileSync, mkdirSync, existsSync, unlinkSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";

const CONFIG_DIR = join(homedir(), ".clawfirm");
const SESSION_FILE = join(CONFIG_DIR, "session.json");

export const BASE_URL = "https://clawfirm.dev";

function ensureDir() {
  if (!existsSync(CONFIG_DIR)) mkdirSync(CONFIG_DIR, { recursive: true });
}

export function saveSession(data) {
  ensureDir();
  writeFileSync(SESSION_FILE, JSON.stringify(data, null, 2), { mode: 0o600 });
}

export function loadSession() {
  if (!existsSync(SESSION_FILE)) return null;
  try {
    return JSON.parse(readFileSync(SESSION_FILE, "utf8"));
  } catch {
    return null;
  }
}

export function clearSession() {
  if (existsSync(SESSION_FILE)) unlinkSync(SESSION_FILE);
}

export function requireSession() {
  const session = loadSession();
  if (!session) {
    console.error("\n  Not logged in. Run: clawfirm login\n");
    process.exit(1);
  }
  return session;
}

export function checkLicense() {
  const session = loadSession();

  if (!session) {
    console.error("\n  Not logged in. Run: clawfirm login\n");
    process.exit(1);
  }

  if (!session.unlocked || !session.unlockedUntil) {
    console.error("\n  Account not activated. Visit https://clawfirm.dev\n");
    process.exit(1);
  }

  const expiry = new Date(session.unlockedUntil);
  if (isNaN(expiry.getTime()) || expiry < new Date()) {
    console.error(`\n  License expired (${session.unlockedUntil}). Visit https://clawfirm.dev to renew.\n`);
    process.exit(1);
  }
}
