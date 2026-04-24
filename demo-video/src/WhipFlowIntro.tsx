import React from "react";
import {
  AbsoluteFill,
  Sequence,
  useCurrentFrame,
  useVideoConfig,
  interpolate,
  spring,
} from "remotion";

// ─── Design Tokens ───
const BG = "#09090b";
const BG_SURFACE = "#18181b";
const BORDER = "#27272a";
const TEXT = "#f4f4f5";
const MUTED = "#71717a";
const GREEN = "#22c55e";
const BLUE = "#3b82f6";
const PURPLE = "#a855f7";
const ORANGE = "#f97316";

const FONT = "system-ui, -apple-system, sans-serif";
const MONO = "'SF Mono', 'Fira Code', 'Cascadia Code', monospace";

const CLAMP = {
  extrapolateLeft: "clamp" as const,
  extrapolateRight: "clamp" as const,
};

// ─── Helpers ───
function useTypewriter(text: string, frame: number, charDelay = 2): string {
  const chars = Math.floor(frame / charDelay);
  return text.slice(0, Math.min(chars, text.length));
}

function BottomTicker({
  text,
  frame,
  delay = 10,
}: {
  text: string;
  frame: number;
  delay?: number;
}) {
  return (
    <div
      style={{
        position: "absolute",
        bottom: 30,
        left: 0,
        right: 0,
        display: "flex",
        justifyContent: "center",
      }}
    >
      <div
        style={{
          background: BG_SURFACE,
          border: `1px solid ${BORDER}`,
          borderRadius: 8,
          padding: "8px 24px",
          fontSize: 16,
          fontFamily: MONO,
          color: MUTED,
          opacity: interpolate(frame, [delay, delay + 5], [0, 1], CLAMP),
        }}
      >
        {text}
      </div>
    </div>
  );
}

// ─── Scene 1: Hook — Chat Prompt ───
const Scene1Hook: React.FC = () => {
  const frame = useCurrentFrame();

  const promptText = '> whipflow run daily-publish.whip';
  const displayPrompt = useTypewriter(promptText, frame, 1);
  const showCursor = frame < promptText.length * 1 + 10;

  const responseDelay = 40;
  const responseOpacity = interpolate(
    frame,
    [responseDelay, responseDelay + 10],
    [0, 1],
    CLAMP
  );

  const glowOpacity = interpolate(frame, [0, 45, 60], [0, 0, 0.15], CLAMP);

  return (
    <AbsoluteFill
      style={{
        background: BG,
        display: "flex",
        flexDirection: "column",
        justifyContent: "center",
        alignItems: "center",
        padding: 60,
      }}
    >
      {/* Glow */}
      <div
        style={{
          position: "absolute",
          width: 600,
          height: 600,
          borderRadius: "50%",
          background: `radial-gradient(circle, ${GREEN}, transparent 70%)`,
          opacity: glowOpacity,
          top: "20%",
          left: "30%",
        }}
      />

      {/* Terminal */}
      <div
        style={{
          background: BG_SURFACE,
          border: `1px solid ${BORDER}`,
          borderRadius: 12,
          padding: 40,
          width: 900,
          fontFamily: MONO,
        }}
      >
        {/* Terminal dots */}
        <div style={{ display: "flex", gap: 8, marginBottom: 24 }}>
          <div
            style={{
              width: 12,
              height: 12,
              borderRadius: "50%",
              background: "#ef4444",
            }}
          />
          <div
            style={{
              width: 12,
              height: 12,
              borderRadius: "50%",
              background: "#eab308",
            }}
          />
          <div
            style={{
              width: 12,
              height: 12,
              borderRadius: "50%",
              background: GREEN,
            }}
          />
        </div>

        {/* Command */}
        <div style={{ fontSize: 28, color: GREEN, marginBottom: 16 }}>
          {displayPrompt}
          {showCursor && (
            <span
              style={{
                display: "inline-block",
                width: 14,
                height: 28,
                background: GREEN,
                marginLeft: 2,
                verticalAlign: "middle",
                opacity: Math.sin(frame * 0.4) > 0 ? 1 : 0,
              }}
            />
          )}
        </div>

        {/* Response */}
        <div style={{ opacity: responseOpacity, color: MUTED, fontSize: 20 }}>
          ✓ Loading 6 agents... orchestrating workflow
        </div>
      </div>

      <BottomTicker text="WhipFlow DSL" frame={frame} delay={5} />
    </AbsoluteFill>
  );
};

// ─── Scene 2: Code Reveal ───
const Scene2Code: React.FC = () => {
  const frame = useCurrentFrame();

  const codeLines = [
    { text: 'ask topic: "请输入主题"', color: BLUE, delay: 0 },
    { text: "", color: TEXT, delay: 8 },
    { text: "agent analyst:", color: PURPLE, delay: 12 },
    { text: '  tools: ["browser", "read"]', color: MUTED, delay: 18 },
    { text: "", color: TEXT, delay: 24 },
    {
      text: 'let data = session: analyst',
      color: GREEN,
      delay: 28,
    },
    { text: '  prompt: "分析 {topic}"', color: TEXT, delay: 36 },
    { text: "", color: TEXT, delay: 42 },
    { text: "print data", color: ORANGE, delay: 48 },
  ];

  return (
    <AbsoluteFill
      style={{
        background: BG,
        display: "flex",
        justifyContent: "center",
        alignItems: "center",
        padding: 60,
      }}
    >
      <div
        style={{
          background: BG_SURFACE,
          border: `1px solid ${BORDER}`,
          borderRadius: 12,
          padding: "40px 50px",
          width: 1000,
          fontFamily: MONO,
        }}
      >
        {/* Header */}
        <div
          style={{
            fontSize: 14,
            color: MUTED,
            textTransform: "uppercase" as const,
            letterSpacing: 2,
            marginBottom: 24,
            opacity: interpolate(frame, [0, 5], [0, 1], CLAMP),
          }}
        >
          example.whip
        </div>

        {/* Code lines */}
        {codeLines.map((line, i) => {
          const lineOpacity = interpolate(
            frame,
            [line.delay, line.delay + 6],
            [0, 1],
            CLAMP
          );
          const slideX = interpolate(
            frame,
            [line.delay, line.delay + 6],
            [-30, 0],
            CLAMP
          );
          return (
            <div
              key={i}
              style={{
                opacity: lineOpacity,
                transform: `translateX(${slideX}px)`,
                fontSize: 24,
                lineHeight: 1.8,
                color: line.color,
                minHeight: 20,
              }}
            >
              {line.text}
            </div>
          );
        })}
      </div>

      <BottomTicker text=".whip syntax" frame={frame} delay={5} />
    </AbsoluteFill>
  );
};

// ─── Scene 3: Agent Roles ───
const Scene3Agents: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const agents = [
    {
      name: "analyst",
      icon: "🔍",
      tools: "browser, read",
      color: BLUE,
      desc: "Data Collection",
    },
    {
      name: "writer",
      icon: "✍️",
      tools: "bash, write",
      color: PURPLE,
      desc: "Content Creation",
    },
    {
      name: "publisher",
      icon: "🚀",
      tools: "browser, bash",
      color: GREEN,
      desc: "Multi-Platform",
    },
  ];

  return (
    <AbsoluteFill
      style={{
        background: BG,
        display: "flex",
        flexDirection: "column",
        justifyContent: "center",
        alignItems: "center",
        padding: 60,
      }}
    >
      {/* Title */}
      <div
        style={{
          fontSize: 14,
          color: MUTED,
          fontFamily: MONO,
          textTransform: "uppercase" as const,
          letterSpacing: 2,
          marginBottom: 40,
          opacity: interpolate(frame, [0, 5], [0, 1], CLAMP),
        }}
      >
        Specialized AI Agents
      </div>

      {/* Agent cards */}
      <div style={{ display: "flex", gap: 40 }}>
        {agents.map((agent, i) => {
          const delay = i * 12;
          const cardScale = spring({
            frame: Math.max(0, frame - delay),
            fps,
            config: { damping: 12, stiffness: 100 },
          });
          return (
            <div
              key={agent.name}
              style={{
                transform: `scale(${cardScale})`,
                background: BG_SURFACE,
                border: `1px solid ${BORDER}`,
                borderRadius: 16,
                padding: "36px 40px",
                width: 280,
                textAlign: "center" as const,
              }}
            >
              <div style={{ fontSize: 48, marginBottom: 16 }}>{agent.icon}</div>
              <div
                style={{
                  fontSize: 24,
                  fontWeight: 700,
                  color: agent.color,
                  fontFamily: MONO,
                  marginBottom: 8,
                }}
              >
                {agent.name}
              </div>
              <div
                style={{
                  fontSize: 18,
                  color: TEXT,
                  marginBottom: 12,
                  fontFamily: FONT,
                }}
              >
                {agent.desc}
              </div>
              <div
                style={{
                  fontSize: 14,
                  color: MUTED,
                  fontFamily: MONO,
                }}
              >
                [{agent.tools}]
              </div>
            </div>
          );
        })}
      </div>

      <BottomTicker text="agent definitions" frame={frame} delay={5} />
    </AbsoluteFill>
  );
};

// ─── Scene 4: Parallel Execution ───
const Scene4Parallel: React.FC = () => {
  const frame = useCurrentFrame();

  const tasks = [
    { label: "session: analyst", status: "analyzing...", color: BLUE },
    { label: "session: writer", status: "composing...", color: PURPLE },
    { label: "session: publisher", status: "publishing...", color: GREEN },
  ];

  const headerOpacity = interpolate(frame, [0, 8], [0, 1], CLAMP);

  return (
    <AbsoluteFill
      style={{
        background: BG,
        display: "flex",
        flexDirection: "column",
        justifyContent: "center",
        alignItems: "center",
        padding: 60,
      }}
    >
      {/* parallel: keyword */}
      <div
        style={{
          opacity: headerOpacity,
          fontSize: 32,
          fontWeight: 700,
          color: ORANGE,
          fontFamily: MONO,
          marginBottom: 40,
        }}
      >
        parallel:
      </div>

      {/* Parallel lanes */}
      <div style={{ display: "flex", gap: 30 }}>
        {tasks.map((task, i) => {
          const delay = 10 + i * 3;
          const laneOpacity = interpolate(
            frame,
            [delay, delay + 8],
            [0, 1],
            CLAMP
          );
          const barWidth = interpolate(
            frame,
            [delay + 10, delay + 60],
            [0, 100],
            CLAMP
          );
          const pulse = Math.sin((frame - delay) * 0.15) * 0.3 + 0.7;

          return (
            <div
              key={task.label}
              style={{
                opacity: laneOpacity,
                background: BG_SURFACE,
                border: `1px solid ${BORDER}`,
                borderRadius: 12,
                padding: "24px 30px",
                width: 280,
              }}
            >
              <div
                style={{
                  fontSize: 18,
                  fontFamily: MONO,
                  color: task.color,
                  marginBottom: 12,
                }}
              >
                {task.label}
              </div>

              {/* Progress bar */}
              <div
                style={{
                  height: 6,
                  borderRadius: 3,
                  background: BORDER,
                  marginBottom: 12,
                  overflow: "hidden",
                }}
              >
                <div
                  style={{
                    width: `${barWidth}%`,
                    height: "100%",
                    borderRadius: 3,
                    background: task.color,
                    opacity: pulse,
                  }}
                />
              </div>

              <div style={{ fontSize: 14, color: MUTED, fontFamily: MONO }}>
                {barWidth < 100 ? task.status : "✓ done"}
              </div>
            </div>
          );
        })}
      </div>

      <BottomTicker text="concurrent execution" frame={frame} delay={5} />
    </AbsoluteFill>
  );
};

// ─── Scene 5: Real Pipeline — 6 Platforms ───
const Scene5Pipeline: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const platforms = [
    { name: "小红书", icon: "📕", color: "#ff2442" },
    { name: "Twitter", icon: "𝕏", color: BLUE },
    { name: "B站", icon: "📺", color: "#00a1d6" },
    { name: "微博", icon: "🔴", color: "#ff8200" },
    { name: "Telegram", icon: "✈️", color: "#0088cc" },
    { name: "Binance", icon: "🟡", color: "#f0b90b" },
  ];

  return (
    <AbsoluteFill
      style={{
        background: BG,
        display: "flex",
        flexDirection: "column",
        justifyContent: "center",
        alignItems: "center",
        padding: 60,
      }}
    >
      {/* Title */}
      <div
        style={{
          fontSize: 14,
          color: MUTED,
          fontFamily: MONO,
          textTransform: "uppercase" as const,
          letterSpacing: 2,
          marginBottom: 12,
          opacity: interpolate(frame, [0, 5], [0, 1], CLAMP),
        }}
      >
        daily-publish.whip
      </div>
      <div
        style={{
          fontSize: 28,
          fontWeight: 700,
          color: TEXT,
          fontFamily: FONT,
          marginBottom: 40,
          opacity: interpolate(frame, [3, 10], [0, 1], CLAMP),
        }}
      >
        One Script → 6 Platforms
      </div>

      {/* Platform grid */}
      <div
        style={{
          display: "flex",
          flexWrap: "wrap" as const,
          gap: 20,
          justifyContent: "center",
          maxWidth: 960,
        }}
      >
        {platforms.map((p, i) => {
          const delay = 8 + i * 6;
          const cardScale = spring({
            frame: Math.max(0, frame - delay),
            fps,
            config: { damping: 12, stiffness: 120 },
          });
          const checkDelay = delay + 30;
          const checkOpacity = interpolate(
            frame,
            [checkDelay, checkDelay + 5],
            [0, 1],
            CLAMP
          );

          return (
            <div
              key={p.name}
              style={{
                transform: `scale(${cardScale})`,
                background: BG_SURFACE,
                border: `1px solid ${BORDER}`,
                borderRadius: 12,
                padding: "20px 28px",
                width: 280,
                display: "flex",
                alignItems: "center",
                gap: 16,
              }}
            >
              <div style={{ fontSize: 32 }}>{p.icon}</div>
              <div>
                <div
                  style={{
                    fontSize: 20,
                    fontWeight: 600,
                    color: p.color,
                    fontFamily: FONT,
                  }}
                >
                  {p.name}
                </div>
                <div
                  style={{
                    fontSize: 14,
                    color: GREEN,
                    fontFamily: MONO,
                    opacity: checkOpacity,
                  }}
                >
                  ✓ published
                </div>
              </div>
            </div>
          );
        })}
      </div>

      <BottomTicker text="multi-platform publishing" frame={frame} delay={5} />
    </AbsoluteFill>
  );
};

// ─── Scene 6: Grand Finale ───
const Scene6Finale: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  const logoScale = spring({
    frame,
    fps,
    config: { damping: 10, stiffness: 80 },
  });

  const taglineOpacity = interpolate(frame, [25, 35], [0, 1], CLAMP);
  const urlOpacity = interpolate(frame, [40, 50], [0, 1], CLAMP);

  const glowOpacity = interpolate(frame, [0, 30], [0, 0.2], CLAMP);

  return (
    <AbsoluteFill
      style={{
        background: BG,
        display: "flex",
        flexDirection: "column",
        justifyContent: "center",
        alignItems: "center",
      }}
    >
      {/* Background glow */}
      <div
        style={{
          position: "absolute",
          width: 800,
          height: 800,
          borderRadius: "50%",
          background: `radial-gradient(circle, ${GREEN}, transparent 60%)`,
          opacity: glowOpacity,
        }}
      />

      {/* Logo text */}
      <div
        style={{
          transform: `scale(${logoScale})`,
          display: "flex",
          alignItems: "baseline",
          gap: 12,
          marginBottom: 20,
        }}
      >
        <span
          style={{
            fontSize: 72,
            fontWeight: 800,
            color: TEXT,
            fontFamily: FONT,
          }}
        >
          Whip
        </span>
        <span
          style={{
            fontSize: 72,
            fontWeight: 800,
            color: GREEN,
            fontFamily: FONT,
          }}
        >
          Flow
        </span>
      </div>

      {/* Tagline */}
      <div
        style={{
          opacity: taglineOpacity,
          fontSize: 28,
          color: MUTED,
          fontFamily: FONT,
          marginBottom: 16,
        }}
      >
        AI Workflow Orchestration DSL
      </div>

      {/* Brand */}
      <div
        style={{
          opacity: urlOpacity,
          fontSize: 18,
          color: PURPLE,
          fontFamily: MONO,
          letterSpacing: 2,
        }}
      >
        by ClawFirm
      </div>
    </AbsoluteFill>
  );
};

// ─── Bilingual Subtitle Layer ───
type SubtitleEntry = {
  startFrame: number;
  endFrame: number;
  en: string;
  zh: string;
  style?: "default" | "emphasis" | "brand";
};

const SUBTITLES: SubtitleEntry[] = [
  {
    startFrame: 0,
    endFrame: 88,
    en: "What if AI agents could follow a script?",
    zh: "如果 AI 能按脚本协作？",
    style: "emphasis",
  },
  {
    startFrame: 90,
    endFrame: 178,
    en: "Define tasks in plain DSL",
    zh: "用简洁的 DSL 定义任务",
  },
  {
    startFrame: 180,
    endFrame: 268,
    en: "Assign specialized roles",
    zh: "分配专业角色",
  },
  {
    startFrame: 270,
    endFrame: 358,
    en: "Run them in parallel",
    zh: "并行执行，效率翻倍",
    style: "emphasis",
  },
  {
    startFrame: 360,
    endFrame: 448,
    en: "Ship to 6 platforms at once",
    zh: "一键发布 6 大平台",
  },
  {
    startFrame: 450,
    endFrame: 538,
    en: "WhipFlow — by ClawFirm",
    zh: "WhipFlow — AI 工作流引擎",
    style: "brand",
  },
];

const SubtitleLayer: React.FC = () => {
  const frame = useCurrentFrame();
  const current = SUBTITLES.find(
    (s) => frame >= s.startFrame && frame < s.endFrame
  );
  if (!current) return null;

  const fadeIn = interpolate(
    frame,
    [current.startFrame, current.startFrame + 8],
    [0, 1],
    CLAMP
  );
  const fadeOut = interpolate(
    frame,
    [current.endFrame - 8, current.endFrame],
    [1, 0],
    CLAMP
  );
  const opacity = Math.min(fadeIn, fadeOut);

  const isEmphasis = current.style === "emphasis";
  const isBrand = current.style === "brand";

  return (
    <div
      style={{
        position: "absolute",
        bottom: 80,
        left: 0,
        right: 0,
        display: "flex",
        justifyContent: "center",
        zIndex: 100,
      }}
    >
      <div
        style={{
          opacity,
          background: "rgba(0,0,0,0.75)",
          backdropFilter: "blur(8px)",
          borderRadius: 10,
          padding: isBrand ? "14px 48px" : "10px 36px",
          maxWidth: 1200,
          textAlign: "center" as const,
          display: "flex",
          flexDirection: "column",
          gap: 4,
        }}
      >
        {/* English */}
        <span
          style={{
            fontSize: isBrand ? 30 : isEmphasis ? 26 : 22,
            fontWeight: isBrand ? 800 : isEmphasis ? 700 : 500,
            color: isBrand ? GREEN : TEXT,
            fontFamily: FONT,
            letterSpacing: isBrand ? 1 : 0,
          }}
        >
          {current.en}
        </span>
        {/* Chinese */}
        <span
          style={{
            fontSize: isBrand ? 22 : 18,
            fontWeight: 400,
            color: isBrand ? MUTED : "rgba(244,244,245,0.6)",
            fontFamily: FONT,
          }}
        >
          {current.zh}
        </span>
      </div>
    </div>
  );
};

// ─── Main Composition ───
export const WhipFlowIntro: React.FC = () => {
  const SCENE_DURATION = 90;
  const TOTAL = SCENE_DURATION * 6; // 540 frames = 18s

  return (
    <AbsoluteFill style={{ background: BG }}>
      <Sequence from={0} durationInFrames={SCENE_DURATION}>
        <Scene1Hook />
      </Sequence>
      <Sequence from={SCENE_DURATION} durationInFrames={SCENE_DURATION}>
        <Scene2Code />
      </Sequence>
      <Sequence from={SCENE_DURATION * 2} durationInFrames={SCENE_DURATION}>
        <Scene3Agents />
      </Sequence>
      <Sequence from={SCENE_DURATION * 3} durationInFrames={SCENE_DURATION}>
        <Scene4Parallel />
      </Sequence>
      <Sequence from={SCENE_DURATION * 4} durationInFrames={SCENE_DURATION}>
        <Scene5Pipeline />
      </Sequence>
      <Sequence from={SCENE_DURATION * 5} durationInFrames={SCENE_DURATION}>
        <Scene6Finale />
      </Sequence>

      {/* Subtitle layer on top of everything */}
      <Sequence from={0} durationInFrames={TOTAL}>
        <SubtitleLayer />
      </Sequence>
    </AbsoluteFill>
  );
};
