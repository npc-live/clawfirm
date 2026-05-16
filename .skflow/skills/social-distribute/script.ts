import { sh, ask, done } from "@skflow/runtime";

// ── Top-level helpers (preserved verbatim in compiled output) ──

const CONFIGS: Record<string, { yaml: string; cmd: string; build: (c: any, u: any) => string[] }> = {
  twitter: {
    yaml: "x.yaml", cmd: "post_video",
    build: (c, u) => [u.video, c.text],
  },
  bilibili: {
    yaml: "bilibili-fill.yaml", cmd: "post",
    build: (c, u) => [u.video, c.title, c.desc, u.cover_v || "", c.tags || "", u.chapters || ""],
  },
  youtube: {
    yaml: "youtube-fill.yaml", cmd: "post_video",
    build: (c, u) => [u.video, c.title, c.desc, "unlisted", u.chapters || ""],
  },
  tiktok: {
    yaml: "tiktok-fill.yaml", cmd: "post",
    build: (c, u) => [u.video, c.title, c.desc, c.tags || "", u.cover_h || "", u.cover_v || ""],
  },
  xiaohongshu: {
    yaml: "xhs-fill.yaml", cmd: "post_video",
    build: (c, u) => [u.video, c.title, c.desc, c.tags || "", u.chapters || ""],
  },
  douyin: {
    yaml: "douyin-fill.yaml", cmd: "post",
    build: (c, u) => [u.video, c.title, c.desc, u.cover_v || "", c.tags || "", u.chapters || ""],
  },
  linkedin: {
    yaml: "linkedin-fill.yaml", cmd: "post_video",
    build: (c, u) => [u.video, c.title, c.desc],
  },
  wechat_channels: {
    yaml: "channels-fill.yaml", cmd: "post",
    build: (c, u) => [u.video, c.title, c.desc, u.cover_v || ""],
  },
};

function safeParse(text: string): any {
  try { return JSON.parse(text); } catch { return null; }
}

function buildReadCmd(platforms: string[]): string {
  const dir = "app/assets/skills/social-publish";
  const parts = platforms.map(function(name) {
    return 'echo "<<CR:' + name + '>>" && cat ' + dir + '/' + name + '/references/content-rules.md 2>/dev/null; echo "<<FS:' + name + '>>" && cat ' + dir + '/' + name + '/references/format-specs.md 2>/dev/null';
  });
  return parts.join("; ") + '; echo "<<END>>"';
}

function parseRules(text: string, platforms: string[]): Record<string, { contentRules: string; formatSpecs: string }> {
  const out: Record<string, { contentRules: string; formatSpecs: string }> = {};
  let k = 0;
  while (k < platforms.length) {
    const name = platforms[k];
    const crTag = "<<CR:" + name + ">>";
    const fsTag = "<<FS:" + name + ">>";
    const crPos = text.indexOf(crTag);
    const fsPos = text.indexOf(fsTag);
    const nextCrTag = k + 1 < platforms.length ? "<<CR:" + platforms[k + 1] + ">>" : "<<END>>";
    const nextPos = text.indexOf(nextCrTag, fsPos > 0 ? fsPos : 0);
    out[name] = {
      contentRules: crPos >= 0 && fsPos >= 0 ? text.slice(crPos + crTag.length, fsPos).trim() : "",
      formatSpecs: fsPos >= 0 && nextPos >= 0 ? text.slice(fsPos + fsTag.length, nextPos).trim() : "",
    };
    k = k + 1;
  }
  return out;
}

function makeShCmd(platformName: string, copy: any, userIn: any, port: string): string {
  const cfg = CONFIGS[platformName];
  if (!cfg || !copy) return "";
  const args = cfg.build(copy, userIn);
  const payload = JSON.stringify({ file: cfg.yaml, command: cfg.cmd, args: args });
  const escaped = payload.replace(/'/g, "'\\''");
  return "CDP_PORT=" + port + " CLAWD_TOOL_INPUT='" + escaped + "' bin/browser-shortcut";
}

function summarize(arr: any[]): string {
  const ok = arr.filter(function(r) { return r.success; }).length;
  const fail = arr.filter(function(r) { return !r.success; });
  let msg = "分发完成: " + ok + "/" + arr.length + " 成功";
  if (fail.length > 0) {
    msg = msg + "，" + fail.length + " 失败: " + fail.map(function(r) { return r.platform; }).join(", ");
  }
  return msg;
}

// ── Main flow ──

export async function main() {
  // 1. Check CDP reachable — auto-launch if not running
  const cdpCheck = await sh(
    'curl -s --max-time 3 http://127.0.0.1:9333/json/version || curl -s --max-time 3 http://127.0.0.1:9222/json/version'
  );
  if (cdpCheck.code !== 0 || !cdpCheck.stdout.includes("Browser")) {
    await sh('rm -f ~/.social-cli/chrome-profile/SingletonLock ~/.social-cli/chrome-profile/SingletonCookie ~/.social-cli/chrome-profile/SingletonSocket 2>/dev/null; true');
    const launch = await sh('CDP_PORT=9222 bash scripts/launch-chrome-cdp.sh');
    if (launch.code !== 0) {
      return done({ summary: "Chrome CDP 自动启动失败: " + launch.stderr.slice(0, 300) });
    }
  }

  // 2. Detect CDP port
  const portTest = await sh('curl -s --max-time 2 http://127.0.0.1:9333/json/version');
  let cdpPort = "9222";
  if (portTest.code === 0 && portTest.stdout.includes("Browser")) {
    cdpPort = "9333";
  }

  // 3. Read input from env
  const inputRaw = await sh("echo $SOCIAL_DISTRIBUTE_INPUT");
  let userInput = safeParse(inputRaw.stdout.trim());
  if (!userInput) {
    return done({ summary: "SOCIAL_DISTRIBUTE_INPUT 环境变量为空或不是有效 JSON" });
  }
  if (!userInput.video || !userInput.topic || !userInput.platforms || userInput.platforms.length === 0) {
    return done({ summary: "缺少必需字段: video, topic, platforms" });
  }

  // 3b. Read chapters from video directory: prefer .srt, fallback to chapters*.txt
  if (!userInput.chapters) {
    let videoDir = userInput.video.replace(/\/[^/]+$/, "");
    let srtCheck = await sh('ls "' + videoDir + '"/*.srt 2>/dev/null | head -1');
    if (srtCheck.code === 0 && srtCheck.stdout.trim()) {
      let srtContent = await sh('cat "' + srtCheck.stdout.trim() + '"');
      if (srtContent.code === 0 && srtContent.stdout.trim()) {
        let chaptersFromSrt = await ask({
          prompt: "从以下 SRT 字幕中提取视频章节时间轴。输出纯文本，每行格式：MM:SS 章节标题。只提取主要话题转折点（10-20个章节），不要输出其他内容。",
          data: { srt: srtContent.stdout },
        });
        if (chaptersFromSrt) {
          userInput.chapters = chaptersFromSrt;
        }
      }
    }
    if (!userInput.chapters) {
      let txtCheck = await sh('ls "' + videoDir + '"/chapters*.txt "' + videoDir + '"/*章节*.txt 2>/dev/null | head -1');
      if (txtCheck.code === 0 && txtCheck.stdout.trim()) {
        let txtContent = await sh('cat "' + txtCheck.stdout.trim() + '"');
        if (txtContent.code === 0 && txtContent.stdout.trim()) {
          userInput.chapters = txtContent.stdout.trim();
        }
      }
    }
  }

  // 4. Batch-read all content-rules in ONE shell call
  let readCmd = buildReadCmd(userInput.platforms);
  const rulesRaw = await sh(readCmd);
  let platformRules = parseRules(rulesRaw.stdout, userInput.platforms);

  // 5. Yield to LLM: generate copywriting for all platforms at once
  const copywriting = await ask({
    prompt:
      "为以下视频生成多平台分发文案。\n\n" +
      "要求：\n" +
      "- 每个平台独立生成，严格遵守该平台的 content-rules 和 format-specs\n" +
      "- 返回纯 JSON 对象（不要 markdown code block），key 是平台名，value 包含字段\n" +
      "- Twitter: {text} — 中英双语推文\n" +
      "- Bilibili/小红书/抖音: {title, desc, tags} — tags 用逗号分隔\n" +
      "- YouTube: {title, desc} — 中英双语\n" +
      "- TikTok: {title, desc, tags} — 中英双语，tags 用逗号分隔\n" +
      "- LinkedIn: {title, desc} — 专业语气\n" +
      "- 微信视频号(wechat_channels): {title, desc} — 中文",
    data: {
      topic: userInput.topic,
      platforms: userInput.platforms,
      platformRules: platformRules,
    },
  });

  // 6. Parse LLM response
  let allCopy = safeParse(copywriting);
  if (!allCopy) {
    return done({ summary: "LLM 返回的文案不是有效 JSON", data: { raw: copywriting } });
  }

  // 7. Execute browser-shortcut per platform (while loop — skflow compiles this correctly)
  let results: any[] = [];
  let idx = 0;
  while (idx < userInput.platforms.length) {
    let pName = userInput.platforms[idx];
    let shellCmd = makeShCmd(pName, allCopy[pName], userInput, cdpPort);

    if (shellCmd) {
      let shOut = await sh(shellCmd, { timeout: 600000 });
      results.push({
        platform: pName,
        success: shOut.code === 0,
        code: shOut.code,
        error: shOut.code !== 0 ? shOut.stderr.slice(0, 500) : undefined,
      });
    } else {
      results.push({
        platform: pName,
        success: false,
        code: -1,
        error: "未知平台或 LLM 未生成文案: " + pName,
      });
    }

    idx = idx + 1;
  }

  // 8. Return summary
  return done({ summary: summarize(results), data: { results: results } });
}
