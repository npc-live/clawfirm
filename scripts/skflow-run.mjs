#!/usr/bin/env node
import { pathToFileURL } from "node:url";
import { resolve } from "node:path";
import { run, resume, RuntimeError } from "@skflow/runtime";

const [,, command, ...args] = process.argv;

if (command === "run") {
  const name = args[0];
  if (!name) { console.error("Usage: skflow-run.mjs run <name>"); process.exit(1); }
  const scriptPath = resolve(`.skflow/skills/${name}/script.compiled.js`);
  const mod = await import(pathToFileURL(scriptPath).href);
  try {
    const result = run({ scriptName: name, scriptPath, step: mod.step });
    process.stdout.write(JSON.stringify(result) + "\n");
  } catch (err) {
    if (err instanceof RuntimeError) {
      process.stdout.write(JSON.stringify(err.errorMessage) + "\n");
      process.exit(1);
    }
    throw err;
  }
} else if (command === "resume") {
  const sessionId = args[0];
  const answerArg = args.find(a => a.startsWith("--answer="));
  if (!sessionId || !answerArg) {
    console.error("Usage: skflow-run.mjs resume <session-id> --answer='<text>'");
    process.exit(1);
  }
  const answer = answerArg.slice("--answer=".length);
  const { loadMeta } = await import("@skflow/runtime/session");
  const meta = loadMeta(sessionId);
  const mod = await import(pathToFileURL(meta.scriptPath).href);
  try {
    const result = resume({ sessionId, answer, step: mod.step });
    process.stdout.write(JSON.stringify(result) + "\n");
  } catch (err) {
    if (err instanceof RuntimeError) {
      process.stdout.write(JSON.stringify(err.errorMessage) + "\n");
      process.exit(1);
    }
    throw err;
  }
} else {
  console.error("Usage: skflow-run.mjs <run|resume> ...");
  process.exit(1);
}
