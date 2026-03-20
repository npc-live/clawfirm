import { spawnSync } from "node:child_process";

export function dispatch(name, args) {
  const cmd = `clawfirm-${name}`;
  const result = spawnSync(cmd, args, { stdio: "inherit" });

  if (result.error?.code === "ENOENT") {
    console.error(`\n  clawfirm: '${name}' not found (tried: ${cmd})\n`);
    process.exit(127);
  }

  process.exit(result.status ?? 0);
}
