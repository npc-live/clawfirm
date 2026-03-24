---
name: remotion-video
version: 1.0.0
description: |
  Smart video factory that auto-generates Remotion-based demo/promo videos.
  Can analyze a GitHub repo (deep scan) OR use conversation context + Claude memory
  to auto-generate a video script with scenes, timing, visuals, and burned-in subtitles.
  Produces a self-contained TSX composition, registers it, and renders to mp4.

  Activate when: user wants to create a demo video, promo video, product showcase,
  explainer video, or any programmatic video using Remotion — especially when they
  share a GitHub repo URL or have been discussing a product in conversation.
---

# Remotion Video Skill

Smart video factory: takes a GitHub repo OR conversation context, auto-generates a
video script (scenes, timing, visuals, burned-in subtitles), and outputs a complete,
renderable Remotion TSX composition.

---

## Prerequisites

The project must have a Remotion setup. If `demo-video/` does not exist, scaffold it:

```bash
mkdir -p demo-video/src
cd demo-video
npm init -y
npm install remotion @remotion/cli @remotion/player react react-dom typescript @types/react
```

Create `demo-video/src/index.ts`:
```tsx
import { registerRoot } from "remotion";
import { RemotionRoot } from "./Root";
registerRoot(RemotionRoot);
```

Create `demo-video/tsconfig.json`:
```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "jsx": "react-jsx",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true
  },
  "include": ["src"]
}
```

---

## Workflow Overview

The skill runs a **4-phase automated pipeline**:

```
Phase 1: Gather Context    → GitHub repo deep scan OR conversation context OR ask user
Phase 2: Generate Script   → Auto-produce VideoScript with scenes + subtitles
Phase 3: Generate TSX      → Build composition file with SubtitleLayer + scene components
Phase 4: Register & Render → Root.tsx, package.json script, tsc check, render mp4
```

If the user provides a **manual video script** (explicit scenes, timing, visuals), skip
Phases 1–2 and go directly to Phase 3 — backward compatible with the old workflow.

---

## Phase 1: Context Gathering

Only **ONE** input source is needed. Pick the first available:

| Priority | Source | When |
|----------|--------|------|
| 1 | GitHub repo URL | User provides a repo link |
| 2 | Conversation context + Claude memory | Product has been discussed in chat |
| 3 | Ask the user | Neither of the above |

### Path A: GitHub Repo (Deep Scan)

When the user provides a repo URL:

1. **Clone / fetch** the repo via `gh repo clone` or `git clone`
2. **Read core files**: README.md, package.json (or Cargo.toml, pyproject.toml, go.mod, etc.), main config files
3. **Scan source**: detect tech stack, key modules, UI components, API endpoints, entry points
4. **Extract** into a `ProductProfile`:

```typescript
interface ProductProfile {
  name: string;           // product/project name
  tagline: string;        // one-line description
  techStack: string[];    // e.g. ["Next.js", "TypeScript", "Prisma"]
  coreFeatures: string[]; // 3–6 key features
  userFlows: string[];    // key user journeys detected from code
  repoUrl: string;
}
```

5. **Auto-infer a usage story** from the feature set — imagine the ideal user going through the product for the first time

**What to scan in the repo:**
- `README.md` → product name, tagline, feature list, screenshots
- `package.json` / equivalent → name, description, dependencies (infer tech stack)
- `src/` or `app/` → route structure, key components, API handlers
- Config files (`.env.example`, `docker-compose.yml`) → infrastructure signals
- `docs/` → additional context if available

### Path B: Conversation Context + Memory

When no repo URL but conversation has product context:

1. **Scan current conversation** for: user goals, product usage demos, feature requests, pain points, "wow moments", product names
2. **Check Claude memory** for: user profile, project context, previous conversations about this product
3. **Extract** into a `UsageStory`:

```typescript
interface UsageStory {
  productName: string;
  steps: {
    action: string;      // what the user does
    result: string;      // what happens
    wowFactor?: string;  // why this is impressive
  }[];
  valueProp: string;     // the core value proposition
  beforeAfter?: {
    before: string;      // pain point / old way
    after: string;       // with the product
  };
}
```

4. **Infer a ProductProfile** from the conversation context

### Path C: Fallback Ask

When neither A nor B is available, ask:

- "What product is this video for? (GitHub URL or description)"
- "What's the key user journey to showcase?"
- "Any specific tagline or brand message?"
- "Target duration? (default: 15–25s)"

---

## Phase 2: Script Generation

From `ProductProfile` + `UsageStory`, generate a `VideoScript`.

### Type Definitions

```typescript
interface VideoScript {
  title: string;
  duration: number;       // total seconds
  fps: 30;
  width: 1920;
  height: 1080;
  brand: {
    name: string;
    tagline: string;
    url?: string;
  };
  scenes: Scene[];
  subtitles: SubtitleEntry[];
}

interface Scene {
  name: string;
  archetype: string;      // chat | dashboard | code | chart | flywheel | data | trading | finale
  startFrame: number;
  durationFrames: number;
  story: string;          // one-line narrative for this scene
  visualSpec: Record<string, any>; // archetype-specific config (texts, data, layout details)
  ticker: string;         // bottom ticker text
}

interface SubtitleEntry {
  startFrame: number;
  endFrame: number;
  text: string;
  style?: "default" | "emphasis" | "brand";
}
```

### Script Generation Heuristics

**Scene count & timing:**
- Target: `ceil(duration / 3)` scenes, each 2–4 seconds
- Minimum 3 scenes, maximum 12 scenes
- Quick impact scenes: 60–75 frames (2–2.5s)
- Standard scenes: 75–105 frames (2.5–3.5s)
- Complex/data scenes: 90–120 frames (3–4s)
- Finale: 75–90 frames (2.5–3s)

**Scene ordering:**
1. **Opening** — always a Chat/Prompt archetype showing the product's "spark moment" (user's first interaction)
2. **Middle scenes** — map 1:1 to core features extracted from repo or usage story. Choose archetype by feature type:
   - User interaction → Chat
   - Data/analytics output → Dashboard or Data/Research
   - Code generation → Code Generation
   - Metrics/KPIs → Bar Chart or Trading
   - Iterative process → Flywheel
3. **Closing** — always Grand Finale with brand reveal

**Subtitle pacing:**
- ~3 words per second, max ~12 words per subtitle entry
- One subtitle per visual beat / scene transition
- First subtitle appears at frame 0
- Last subtitle should be the brand tagline with `style: "brand"`
- Use `style: "emphasis"` for key value propositions or "wow" moments
- Gaps between subtitles: 0–5 frames (keep it tight)

**Archetype selection guide:**
| Feature type | Best archetype |
|-------------|----------------|
| Chat / prompt / command | Chat/Prompt Scene |
| Research / document output | Data/Research Scene |
| Multi-metric view | Multi-Column Dashboard |
| Code gen / file output | Code Generation Scene |
| Feedback loop / iteration | Circular/Flywheel Scene |
| Financial / dense data | Trading/Terminal Scene |
| Comparison / revenue | Bar Chart / Revenue Scene |
| Brand close / CTA | Grand Finale |

---

## Phase 3: Code Generation

Build a single TSX composition file containing everything.

### Design System

All videos MUST use this consistent dark-mode design system. Never invent new base colors
unless the user explicitly requests a different theme.

#### Color Tokens

```tsx
const BG = "#09090b";           // zinc-950 — main background
const BG_SURFACE = "#18181b";   // zinc-900 — cards, panels
const BORDER = "#27272a";       // zinc-800 — borders, dividers
const TEXT = "#f4f4f5";         // zinc-100 — primary text
const MUTED = "#71717a";        // zinc-500 — secondary text, labels
const GREEN = "#22c55e";        // success, positive, active
const BLUE = "#3b82f6";         // info, links, primary accent
const PURPLE = "#a855f7";       // AI/creative accent
const ORANGE = "#f97316";       // warning, trading accent
const RED = "#ef4444";          // error, negative, loss
const YELLOW = "#eab308";       // caution, medium-severity
```

#### Typography

```tsx
const FONT = "system-ui, -apple-system, sans-serif";
const MONO = "'SF Mono', 'Fira Code', 'Cascadia Code', monospace";
```

- Headings: FONT, bold (700–800)
- Body text: FONT, regular (400–500)
- Code, numbers, labels: MONO
- Section labels: MONO, uppercase, letterSpacing: 2, color: MUTED, fontSize: 14

#### Spacing & Layout

- Standard padding: 60px horizontal, 24–50px vertical
- Card border-radius: 8–16px
- Card pattern: `background: BG_SURFACE, border: 1px solid ${BORDER}, borderRadius: 12`
- Column dividers: `borderRight: 1px solid ${BORDER}`

---

### Reusable Component Patterns

#### 1. BottomTicker

Every scene should have a bottom ticker — a centered pill at the bottom showing context.

```tsx
function BottomTicker({ text, frame, delay = 10 }: { text: string; frame: number; delay?: number }) {
  return (
    <div style={{
      position: "absolute", bottom: 30, left: 0, right: 0,
      display: "flex", justifyContent: "center",
    }}>
      <div style={{
        background: BG_SURFACE,
        border: `1px solid ${BORDER}`,
        borderRadius: 8,
        padding: "8px 24px",
        fontSize: 16,
        fontFamily: MONO,
        color: MUTED,
        opacity: interpolate(frame, [delay, delay + 5], [0, 1], CLAMP),
      }}>
        {text}
      </div>
    </div>
  );
}
```

#### 2. SectionLabel

Uppercase mono label for panel headers.

```tsx
function SectionLabel({ text, frame, delay = 0 }: { text: string; frame: number; delay?: number }) {
  return (
    <div style={{
      fontSize: 14, color: MUTED, fontFamily: MONO,
      textTransform: "uppercase", letterSpacing: 2, marginBottom: 16,
      opacity: interpolate(frame, [delay, delay + 5], [0, 1], CLAMP),
    }}>
      {text}
    </div>
  );
}
```

#### 3. AnimatingNumber

Counter that animates from 0 to a target value.

```tsx
function AnimatingNumber({
  frame, delay, duration, target,
  prefix = "", suffix = "", decimals = 0,
  fontSize = 28, color = TEXT,
}: {
  frame: number; delay: number; duration: number; target: number;
  prefix?: string; suffix?: string; decimals?: number;
  fontSize?: number; color?: string;
}) {
  const value = interpolate(frame, [delay, delay + duration], [0, target], CLAMP);
  const formatted = decimals > 0 ? value.toFixed(decimals) : Math.floor(value).toLocaleString();
  return (
    <span style={{ fontSize, fontWeight: 700, color, fontFamily: MONO }}>
      {prefix}{formatted}{suffix}
    </span>
  );
}
```

#### 4. ChatBubble

User/AI conversation bubble with typewriter effect.

```tsx
function ChatBubble({ text, isUser, frame, delay = 0, typeSpeed = 2 }: {
  text: string; isUser: boolean; frame: number; delay?: number; typeSpeed?: number;
}) {
  const localFrame = frame - delay;
  if (localFrame < 0) return null;
  const displayText = useTypewriter(text, localFrame, typeSpeed);
  const opacity = interpolate(localFrame, [0, 6], [0, 1], CLAMP);
  const showCursor = localFrame < text.length * typeSpeed;
  return (
    <div style={{ opacity, display: "flex", justifyContent: isUser ? "flex-end" : "flex-start", marginBottom: 16 }}>
      <div style={{
        background: isUser ? BLUE : BG_SURFACE,
        border: isUser ? "none" : `1px solid ${BORDER}`,
        borderRadius: isUser ? "20px 20px 4px 20px" : "20px 20px 20px 4px",
        padding: "14px 24px", maxWidth: 700,
        fontSize: 28, color: TEXT, fontFamily: FONT, lineHeight: 1.5,
      }}>
        {displayText}
        {showCursor && (
          <span style={{
            display: "inline-block", width: 2, height: 28,
            background: isUser ? "rgba(255,255,255,0.8)" : GREEN,
            marginLeft: 2, verticalAlign: "middle",
            opacity: Math.sin(localFrame * 0.4) > 0 ? 1 : 0,
          }} />
        )}
      </div>
    </div>
  );
}
```

#### 5. Typewriter Hook

```tsx
function useTypewriter(text: string, frame: number, charDelay = 2): string {
  const chars = Math.floor(frame / charDelay);
  return text.slice(0, Math.min(chars, text.length));
}
```

#### 6. Clamp Helper

```tsx
const CLAMP = { extrapolateLeft: "clamp" as const, extrapolateRight: "clamp" as const };
```

#### 7. SubtitleLayer (NEW)

Burned-in subtitle overlay that sits on top of all scenes. Spans the entire video duration.

```tsx
const SUBTITLES: SubtitleEntry[] = [
  // Generated from VideoScript.subtitles — example:
  // { startFrame: 0, endFrame: 89, text: "Your product is live.", style: "default" },
  // { startFrame: 90, endFrame: 164, text: "Users give feedback. AI iterates.", style: "emphasis" },
];

interface SubtitleEntry {
  startFrame: number;
  endFrame: number;
  text: string;
  style?: "default" | "emphasis" | "brand";
}

function SubtitleLayer() {
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
          background: "rgba(0,0,0,0.7)",
          backdropFilter: "blur(8px)",
          borderRadius: 8,
          padding: isBrand ? "12px 40px" : "8px 32px",
          maxWidth: 1200,
          textAlign: "center",
        }}
      >
        <span
          style={{
            fontSize: isBrand ? 32 : isEmphasis ? 28 : 24,
            fontWeight: isBrand ? 800 : isEmphasis ? 700 : 500,
            color: isBrand ? GREEN : TEXT,
            fontFamily: FONT,
            letterSpacing: isBrand ? 1 : 0,
          }}
        >
          {current.text}
        </span>
      </div>
    </div>
  );
}
```

**Position**: `bottom: 80` places it above the BottomTicker (which is at `bottom: 30`).
**Z-index**: 100 ensures it renders above all scene content.

---

### Animation Patterns

#### Fade In
```tsx
const opacity = interpolate(frame, [delay, delay + 8], [0, 1], CLAMP);
```

#### Slide In (from left)
```tsx
const slideX = interpolate(frame, [delay, delay + 8], [-200, 0], CLAMP);
// apply: transform: `translateX(${slideX}px)`
```

#### Spring Scale (pop in)
```tsx
const scale = spring({
  frame: Math.max(0, frame - delay),
  fps,
  config: { damping: 12, stiffness: 100 },
});
// apply: transform: `scale(${scale})`
```

#### Flash Effect (highlight then fade)
```tsx
const flash = interpolate(frame, [delay, delay + 3, delay + 8], [0, 1, 0.15], CLAMP);
// apply as background overlay opacity
```

#### Pulsing Indicator
```tsx
const pulse = Math.sin(frame * 0.3) > 0 ? 1 : 0.3;
// apply to a small dot: opacity: pulse
```

#### Scrolling Content
```tsx
const scrollY = interpolate(frame, [startFrame, endFrame], [0, -totalScroll], CLAMP);
// apply: transform: `translateY(${scrollY}px)`
// add gradient fade at bottom:
// background: `linear-gradient(transparent, ${BG})`
```

#### Number Counter
```tsx
const value = interpolate(frame, [delay, delay + duration], [0, targetValue], CLAMP);
// display: Math.floor(value).toLocaleString()
```

---

### Scene Archetypes

These are proven scene patterns. Mix and match for any video.

#### 1. Chat/Prompt Scene
User sends a message, AI responds. Good for "idea" or "command" moments.
- Layout: centered or split-panel
- Components: ChatBubble, terminal chrome dots

#### 2. Data/Research Scene
Scrolling document or data output. Good for showing AI-generated analysis.
- Layout: left (prompt) + right (scrolling output)
- Components: SectionLabel, scrolling lines, BottomTicker with byte count

#### 3. Multi-Column Dashboard
Show parallel information streams. Good for "distribution", "comparison", "monitoring".
- Layout: 2–3 columns with border dividers
- Components: SectionLabel per column, cards, counters, checkmarks

#### 4. Code Generation Scene
File tree + code streaming + terminal output.
- Layout: 3-pane (explorer | code | terminal)
- Components: file tree with folder icons, syntax-colored code lines, terminal output

#### 5. Circular/Flywheel Scene
Show a cyclical process. Good for feedback loops, iterative processes.
- Layout: centered SVG circle with nodes positioned using trigonometry
- Components: rotating arc, spring-animated nodes, center metric

#### 6. Trading/Terminal Scene
Dense data display. Good for financial, analytics, monitoring.
- Layout: left (signal feed) + right (chart + stats grid)
- Components: SVG polyline chart with fill, flash-animated signal lines, 2x2 stats grid

#### 7. Bar Chart / Revenue Scene
Show comparative metrics across categories.
- Layout: left (source/variants) + right (bar chart + total)
- Components: animated-width bars, per-bar labels, grand total pill

#### 8. Grand Finale / Brand Reveal
Cards converge, total counter, brand text.
- Layout: centered, phased animation
- Phase 1: cards slide in from edges
- Phase 2: cards compress up, divider line, total counter
- Phase 3: brand name + subtitle + tagline fade in

---

### Composition Structure (Updated with SubtitleLayer)

Every video composition follows this pattern:

```tsx
export const MyVideo: React.FC = () => {
  const totalFrames = /* sum of all scene durations */;

  return (
    <AbsoluteFill style={{ background: BG }}>
      {/* Scene sequences */}
      <Sequence from={0} durationInFrames={90}>
        <Scene1 />
      </Sequence>
      <Sequence from={90} durationInFrames={75}>
        <Scene2 />
      </Sequence>
      {/* ... more scenes */}

      {/* Subtitle layer spans entire video — always last so it renders on top */}
      <Sequence from={0} durationInFrames={totalFrames}>
        <SubtitleLayer />
      </Sequence>
    </AbsoluteFill>
  );
};
```

Each scene is a standalone function component using `useCurrentFrame()` internally.
Frame 0 inside each scene is always the start of that scene (Sequence handles offset).

The `SubtitleLayer` is placed as the **last Sequence** so it renders above all scenes.
Its `SUBTITLES` array uses **absolute frame numbers** (not scene-local), matching
the `VideoScript.subtitles` entries directly.

---

## Phase 4: Register & Render

### Step 1: Register in Root.tsx

Add import and `<Composition>` entry with correct id, frames, fps, dimensions:

```tsx
import { Composition } from "remotion";
import { MyVideo } from "./MyVideo";

export const RemotionRoot: React.FC = () => {
  return (
    <>
      <Composition
        id="MyVideo"
        component={MyVideo}
        durationInFrames={totalFrames}
        fps={30}
        width={1920}
        height={1080}
      />
    </>
  );
};
```

### Step 2: Add render script

Add to `demo-video/package.json`:
```json
"render:<name>": "npx remotion render src/index.ts <CompositionId> out/<name>.mp4"
```

### Step 3: Verify

1. `npx tsc --noEmit` — no type errors
2. `npm run preview` — visual check in Remotion Studio
3. `npm run render:<name>` — render to mp4
4. Verify total duration matches spec
5. Verify subtitles are visible and properly timed

---

## Timing Guidelines

- **30 fps** standard (use 60fps only if user requests smooth slow-motion)
- **1920x1080** resolution (16:9)
- Scene durations:
  - Quick impact scene: 1–2s (30–60 frames)
  - Standard scene: 2–3.5s (60–105 frames)
  - Complex scene with lots of data: 3–4s (90–120 frames)
  - Finale/reveal: 2–3s (60–90 frames)
- Animation timing:
  - Fade in: 5–8 frames
  - Typewriter: 1–2 frames per character
  - Number counter: 30–60 frames for full animation
  - Staggered items: 6–12 frame delay between each
  - Spring pop: damping 12, stiffness 80–120
- Subtitle timing:
  - ~3 words/second reading pace
  - 8-frame fade in, 8-frame fade out
  - Max 12 words per subtitle entry
  - Subtitle `bottom: 80` sits above BottomTicker at `bottom: 30`

---

## Important Rules

1. **One file per composition** — don't split scenes into separate files
2. **All animation via `interpolate()` and `spring()`** — no CSS transitions or keyframes
3. **Use `Sequence` for scene timing** — not manual frame offset math in components
4. **Every scene gets a `BottomTicker`** — it's the visual signature
5. **Every video gets a `SubtitleLayer`** — burned-in subtitles spanning the full duration
6. **Numbers always animate** — never show final values statically
7. **Stagger everything** — items should appear sequentially, not all at once
8. **Dark theme only** — unless user explicitly requests otherwise
9. **No external assets** — no images, fonts, or media files; everything is code-drawn
10. **Keep each scene self-contained** — scene functions should only use `useCurrentFrame()` and `useVideoConfig()`, no cross-scene state
11. **English text by default** — unless user specifies another language
12. **Only one context source needed** — don't require both a repo URL and a manual script; one input path is enough
13. **Subtitle frame numbers are absolute** — they reference the global timeline, not scene-local frames
14. **SubtitleLayer is always the last Sequence** — ensures it renders above all scene content
