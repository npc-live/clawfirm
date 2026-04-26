#!/usr/bin/env tsx
import "dotenv/config"
import { startPumpListener, getBufferStats } from "../src/screener.ts"

console.log("[test] Starting pump.fun listener for 30s — logging raw events...\n")

startPumpListener(
  (token) => console.log("[GRADUATION]", JSON.stringify(token, null, 2)),
  true  // verbose: log raw events
)

await new Promise<void>(r => setTimeout(r, 30_000))

const stats = getBufferStats()
console.log("\n[test] Done. Buffer stats:", stats)
process.exit(0)
