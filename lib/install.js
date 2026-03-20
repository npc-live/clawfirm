import { execSync, spawnSync } from "node:child_process";

const GREEN = "\x1b[32m";
const YELLOW = "\x1b[33m";
const RED = "\x1b[31m";
const BLUE = "\x1b[34m";
const DIM = "\x1b[2m";
const NC = "\x1b[0m";

const ok = (msg) => console.log(`${GREEN}  ✓${NC}  ${msg}`);
const info = (msg) => console.log(`${BLUE}  →${NC}  ${msg}`);
const warn = (msg) => console.log(`${YELLOW}  !${NC}  ${msg}`);
const fail = (msg) => console.log(`${RED}  ✗${NC}  ${msg}`);

function isInstalled(cmd) {
  const result = spawnSync("which", [cmd], { encoding: "utf8" });
  return result.status === 0;
}

function hasRequirement(req) {
  return isInstalled(req);
}

function installTool(tool) {
  info(`Installing ${tool.name}${DIM} — ${tool.description}${NC}`);

  if (!hasRequirement(tool.requires)) {
    fail(`${tool.name}: requires '${tool.requires}' but it's not installed`);
    return false;
  }

  try {
    execSync(tool.install, { stdio: "inherit" });
    ok(`${tool.name} updated`);
    return true;
  } catch {
    fail(`${tool.name}: install failed`);
    return false;
  }
}

function uninstallTool(tool) {
  info(`Uninstalling ${tool.name}${DIM} — ${tool.description}${NC}`);

  if (!isInstalled(tool.check)) {
    ok(`${tool.name} not installed, skipping`);
    return true;
  }

  if (!tool.uninstall) {
    fail(`${tool.name}: no uninstall command defined`);
    return false;
  }

  try {
    execSync(tool.uninstall, { stdio: "inherit", shell: true });
    ok(`${tool.name} uninstalled`);
    return true;
  } catch {
    fail(`${tool.name}: uninstall failed`);
    return false;
  }
}

export async function runUninstall(config, target) {
  const tools = target
    ? config.tools.filter((t) => t.name === target)
    : config.tools;

  if (target && tools.length === 0) {
    fail(`Unknown tool: ${target}`);
    console.log(
      `\n  Available: ${config.tools.map((t) => t.name).join(", ")}\n`
    );
    process.exit(1);
  }

  console.log(`\n  clawfirm uninstall\n`);

  const failed = [];
  for (const tool of tools) {
    const success = uninstallTool(tool);
    if (!success) failed.push(tool.name);
    console.log();
  }

  if (failed.length > 0) {
    warn(`Failed: ${failed.join(", ")}`);
    process.exit(1);
  }

  console.log(`  Done.\n`);
}

export async function runInstall(config, target) {
  const tools = target
    ? config.tools.filter((t) => t.name === target)
    : config.tools;

  if (target && tools.length === 0) {
    fail(`Unknown tool: ${target}`);
    console.log(
      `\n  Available: ${config.tools.map((t) => t.name).join(", ")}\n`
    );
    process.exit(1);
  }

  console.log(`\n  clawfirm install\n`);

  const failed = [];
  for (const tool of tools) {
    const ok = installTool(tool);
    if (!ok) failed.push(tool.name);
    console.log();
  }

  if (failed.length > 0) {
    warn(`Failed: ${failed.join(", ")}`);
    process.exit(1);
  }

  // Download encrypted skills (stored as .enc, not yet decrypted)
  if (!target) {
    const { loadSession } = await import("./auth.js");
    const { installSkills } = await import("./skills.js");
    const session = loadSession();
    if (session) {
      await installSkills(session);
      console.log();
    }
  }

  console.log(`  All tools ready.\n`);
}
