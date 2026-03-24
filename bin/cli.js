#!/usr/bin/env node
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";
import { resolve, dirname } from "node:path";

const __dirname = dirname(fileURLToPath(import.meta.url));

async function loadConfig() {
  const configPath = resolve(__dirname, "../clawfirm.config.js");
  const { default: config } = await import(configPath);
  return config;
}

function printHelp(config) {
  console.log(`
  clawfirm — dev tool manager

  Usage:
    clawfirm login             Log in to clawfirm.dev
    clawfirm whoami            Show current session
    clawfirm logout            Log out
    clawfirm new "<description>"  Start a project from natural language
    clawfirm run <file.whip>   Run a whipflow workflow
    clawfirm install [tool]    Install all tools (or a specific one)
    clawfirm uninstall [tool]  Uninstall all tools (or a specific one)
    clawfirm list              List registered tools
    clawfirm <name> [args]     Dispatch to clawfirm-<name>
    clawfirm help              Show this message

  Tools in registry:
${config.tools.map((t) => `    ${t.name.padEnd(14)} ${t.description}`).join("\n")}
`);
}

function printList(config) {
  console.log(`\n  Registered tools:\n`);
  for (const t of config.tools) {
    console.log(`    ${t.name.padEnd(14)} ${t.description}`);
    console.log(`    ${"".padEnd(14)} ${t.homepage}\n`);
  }
}

async function main() {
  const [, , command, ...rest] = process.argv;

  const config = await loadConfig();

  if (!command || command === "help" || command === "--help" || command === "-h") {
    printHelp(config);
    return;
  }

  if (command === "list") {
    printList(config);
    return;
  }

  if (command === "login") {
    const { runLogin } = await import("../lib/login.js");
    await runLogin();
    return;
  }

  if (command === "whoami") {
    const { runWhoami } = await import("../lib/login.js");
    await runWhoami();
    return;
  }

  if (command === "logout") {
    const { runLogout } = await import("../lib/login.js");
    await runLogout();
    return;
  }

  // Commands below require a valid license
  const { checkLicense } = await import("../lib/auth.js");
  checkLicense();

  if (command === "new") {
    const { runNew } = await import("../lib/new.js");
    await runNew(rest.join(" "));
    return;
  }

  if (command === "install") {
    const { runInstall } = await import("../lib/install.js");
    await runInstall(config, rest[0]);
    return;
  }

  if (command === "uninstall") {
    const { runUninstall } = await import("../lib/install.js");
    await runUninstall(config, rest[0]);
    return;
  }

  if (command === "run") {
    const { spawnSync } = await import("node:child_process");
    const result = spawnSync("whipflow", ["run", ...rest], { stdio: "inherit" });
    if (result.error?.code === "ENOENT") {
      console.error("\n  clawfirm: 'whipflow' not found. Install it with: npm i -g whipflow\n");
      process.exit(1);
    }
    process.exit(result.status ?? 0);
    return;
  }

  // Plugin dispatch: clawfirm <name> → clawfirm-<name>
  const { dispatch } = await import("../lib/dispatch.js");
  dispatch(command, rest);
}

main().catch((err) => {
  console.error(err.message);
  process.exit(1);
});
