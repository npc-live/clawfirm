import { spawnSync } from "node:child_process";
import { writeFileSync } from "node:fs";
import { join } from "node:path";
import { cwd } from "node:process";
import { BASE_URL, requireSession } from "./auth.js";

const POLL_INTERVAL = 3000;

export async function runNew(description) {
  if (!description) {
    console.error("\n  Usage: clawfirm new \"<project description>\"\n");
    process.exit(1);
  }

  const session = requireSession();
  const headers = { "Content-Type": "application/json", Cookie: session.cookie };

  // Step 1: submit job
  let jobId;
  try {
    const res = await fetch(`${BASE_URL}/api/generate-whip`, {
      method: "POST",
      headers,
      body: JSON.stringify({ description, save: false }),
    });

    const body = await res.json().catch(() => ({}));

    if (res.status === 401) {
      console.error("\n  Session expired. Run: clawfirm login\n");
      process.exit(1);
    }
    if (res.status === 402 || res.status === 403) {
      console.error(`\n  ${body.error ?? "Access denied. Visit https://clawfirm.dev"}\n`);
      process.exit(1);
    }
    if (!res.ok) {
      console.error(`\n  API error: ${body.error ?? res.statusText}\n`);
      process.exit(1);
    }

    jobId = body.jobId;
  } catch (e) {
    console.error(`\n  Network error: ${e.message}\n`);
    process.exit(1);
  }

  // Step 2: poll until done
  const startTime = Date.now();
  let whipContent, whipKey;

  while (true) {
    const elapsed = ((Date.now() - startTime) / 1000).toFixed(0);
    process.stdout.write(`\r  Generating... ${elapsed}s `);

    await new Promise(r => setTimeout(r, POLL_INTERVAL));

    let poll;
    try {
      const res = await fetch(`${BASE_URL}/api/generate-whip?job=${jobId}`, { headers });
      poll = await res.json().catch(() => ({}));

      if (res.status === 401) {
        process.stdout.write("\r                    \r");
        console.error("\n  Session expired. Run: clawfirm login\n");
        process.exit(1);
      }
      if (!res.ok) {
        process.stdout.write("\r                    \r");
        console.error(`\n  API error: ${poll.error ?? res.statusText}\n`);
        process.exit(1);
      }
    } catch (e) {
      process.stdout.write("\r                    \r");
      console.error(`\n  Network error: ${e.message}\n`);
      process.exit(1);
    }

    if (poll.status === "completed") {
      whipKey = poll.key ?? `${jobId}/workflow-${Date.now()}.whip`;
      if (poll.whip) {
        whipContent = poll.whip;
      } else {
        // content delivered separately via /api/whips/{jobId}/{filename}
        const dlRes = await fetch(`${BASE_URL}/api/whips/${whipKey}`, { headers });
        whipContent = await dlRes.text();
        if (!dlRes.ok || !whipContent) {
          process.stdout.write("\r                    \r");
          console.error(`\n  Could not fetch whip content (${dlRes.status}): ${whipContent}\n`);
          process.exit(1);
        }
      }
      break;
    }
    if (poll.status === "error") {
      process.stdout.write("\r                    \r");
      console.error(`\n  Generation failed: ${poll.error ?? "unknown error"}\n`);
      process.exit(1);
    }
    // status === "pending" | "running" → keep polling
  }

  process.stdout.write("\r                    \r");

  const whipFile = join(cwd(), whipKey.split("/").pop());
  writeFileSync(whipFile, whipContent, "utf8");
  console.log(`  Saved: ${whipFile}\n`);

  const result = spawnSync("whipflow", ["run", whipFile], { stdio: "inherit" });

  if (result.error?.code === "ENOENT") {
    console.error("\n  whipflow not found. Run: clawfirm install\n");
    process.exit(1);
  }

  process.exit(result.status ?? 0);
}
