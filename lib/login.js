import { createInterface } from "node:readline";
import { BASE_URL, saveSession, clearSession, loadSession } from "./auth.js";

const GREEN = "\x1b[32m";
const YELLOW = "\x1b[33m";
const RED = "\x1b[31m";
const BLUE = "\x1b[34m";
const DIM = "\x1b[2m";
const NC = "\x1b[0m";

const ok = (msg) => console.log(`${GREEN}  ✓${NC}  ${msg}`);
const info = (msg) => console.log(`${BLUE}  →${NC}  ${msg}`);
const fail = (msg) => console.log(`${RED}  ✗${NC}  ${msg}`);

function ask(question) {
  const rl = createInterface({ input: process.stdin, output: process.stdout });
  return new Promise((resolve) => {
    rl.question(question, (ans) => { rl.close(); resolve(ans.trim()); });
  });
}

function askPassword(question) {
  return new Promise((resolve) => {
    process.stdout.write(question);
    let password = "";

    if (process.stdin.isTTY) process.stdin.setRawMode(true);
    process.stdin.resume();
    process.stdin.setEncoding("utf8");

    function onData(char) {
      if (char === "\n" || char === "\r" || char === "\u0004") {
        if (process.stdin.isTTY) process.stdin.setRawMode(false);
        process.stdin.pause();
        process.stdin.removeListener("data", onData);
        process.stdout.write("\n");
        resolve(password);
      } else if (char === "\u0003") {
        process.exit();
      } else if (char === "\u007f" || char === "\b") {
        password = password.slice(0, -1);
      } else {
        password += char;
      }
    }

    process.stdin.on("data", onData);
  });
}

async function fetchMe(cookie) {
  const res = await fetch(`${BASE_URL}/api/auth/me`, {
    headers: { Cookie: cookie },
  });
  if (!res.ok) return null;
  return res.json();
}

export async function runLogin() {
  console.log(`\n  clawfirm login\n`);

  const email = await ask("  Email: ");
  const password = await askPassword("  Password: ");

  info("Signing in...");

  let res;
  try {
    res = await fetch(`${BASE_URL}/api/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });
  } catch (e) {
    fail(`Network error: ${e.message}`);
    process.exit(1);
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    fail(`Login failed: ${body.error ?? res.statusText}`);
    process.exit(1);
  }

  // Extract session cookie from Set-Cookie header
  const setCookie = res.headers.get("set-cookie");
  if (!setCookie) {
    fail("No session cookie returned");
    process.exit(1);
  }
  const cookie = setCookie.split(";")[0];

  // Fetch user info
  const me = await fetchMe(cookie);
  if (!me) {
    fail("Could not verify session");
    process.exit(1);
  }

  saveSession({
    cookie,
    email: me.email,
    unlocked: me.unlocked ?? false,
    unlockedUntil: me.unlockedUntil ?? null,
  });

  ok(`Logged in as ${me.email}`);
  if (me.unlocked && me.unlockedUntil) {
    ok(`Active until ${me.unlockedUntil}`);
  } else {
    console.log(`\n  ${YELLOW}Account not activated.${NC}\n`);
  }
  console.log();
}

export async function runWhoami() {
  const session = loadSession();
  if (!session) {
    console.log(`\n  Not logged in.\n`);
    return;
  }

  console.log(`\n  ${session.email}`);
  if (session.unlocked && session.unlockedUntil) {
    console.log(`  Active until: ${GREEN}${session.unlockedUntil}${NC}`);
  } else {
    console.log(`  Activated: ${YELLOW}no${NC}`);
  }
  console.log();
}

export async function runLogout() {
  const { removeSkills } = await import("./skills.js");
  removeSkills();
  clearSession();
  ok("Logged out");
  console.log();
}
