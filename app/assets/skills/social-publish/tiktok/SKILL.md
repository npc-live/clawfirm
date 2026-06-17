---
name: tiktok
version: 1.0.0
description: |
  TikTok 国际版内容创作全套。包括英文短视频文案（Hook-first 策略）、
  竖屏/横屏格式规范、hashtag 策略、For You Page 算法适配、
  Community Guidelines 合规、完播率优化、封面设计策略。
  建议配合 copywriting-base 使用以获取通用文案能力。
  触发词：TikTok、海外短视频、TikTok运营、TikTok marketing。
---

# TikTok Content Creation

## Caption Style

### Core Tone: Authentic, Hook-first, Trend-aware

- **Authentic**: Sound like a real person, not a brand. Casual, conversational
- **Hook-first**: First line must stop the scroll — 1-2 seconds to capture attention
- **Trend-aware**: Reference trending sounds, formats, and cultural moments
- **Value-driven**: Every video must deliver on its promise
- **Inclusive**: Write for global audiences — simple English, avoid regional slang overuse

### Caption Templates

```
Hook + Value + CTA:
  "POV: you just discovered [hack/trick/fact]"
  "Stop scrolling if you [pain point]"
  "3 things I wish I knew about [topic] sooner"
  "This [method] changed everything for me"
  "Nobody talks about this but [insight]"

Story-driven:
  "I tried [X] for [time] — here's what happened"
  "Day [N] of [challenge/experiment]"
  "What I learned from [experience]"

Engagement bait (compliant):
  "Save this for later"
  "Share with someone who needs this"
  "Comment [emoji] if you agree"
  "Which one are you? 1, 2, or 3?"
```

### Writing Rules

```
DO:
  - Start with a hook (question, bold claim, or relatable scenario)
  - Use line breaks for readability
  - Include 1 clear CTA (follow, save, share, comment)
  - Write in first person where possible
  - Use numbers for specificity ("5 tips" > "some tips")
  - Match caption energy to video energy

DON'T:
  - Start with "Hey guys" or "Welcome back" (wastes the hook)
  - Write walls of text (keep under 150 words in caption)
  - Use all caps excessively (looks spammy)
  - Promise what the video doesn't deliver (reduces retention)
  - Use banned/restricted words (see content-rules.md)
  - ⛔ Insert ANY external URLs in caption (GitHub, website, shop links, etc.)
  - ⛔ Put QR codes in video frames
  - External links ONLY allowed in Bio — use "Link in bio" to redirect
```

---

## Format Specs

### Video Specifications

| Item | Spec |
|------|------|
| Aspect Ratio | 9:16 vertical (primary), 16:9 horizontal, 1:1 square |
| Resolution | 1080×1920 (vertical) / 1920×1080 (horizontal) |
| Format | MP4 / MOV / WebM |
| Size | ≤10GB (web upload), ≤287MB (mobile) |
| Frame Rate | 30fps (recommended) / 60fps |
| Codec | H.264 recommended |
| Duration | 15s — 10min (up to 60min for some accounts) |
| Audio | AAC, 44100Hz |

### Duration Sweet Spots

| Duration | Use Case | Difficulty |
|----------|----------|------------|
| 7-15s | Trending sounds, quick tips, memes | ★ |
| 15-60s | Tutorials, storytelling, POV | ★★ |
| 1-3min | In-depth how-to, reviews, vlogs | ★★★ |
| 3-10min | Series content, full tutorials | ★★★★ |

### Cover Specifications

```
Vertical Cover: 1080×1920 (9:16)
  - Full-screen display on profile grid
  - Primary cover format for most content

Horizontal Cover: 1920×1080 (16:9)
  - Used for embed/share previews
  - Desktop browser display

Design Rules:
  - Bold text overlay (3-5 words max)
  - High contrast colors
  - Face visible = higher CTR (+25-35%)
  - Bright, eye-catching backgrounds
  - Text safe zone: center 80% of frame

Avoid:
  - Small/unreadable text
  - Cluttered designs
  - Logos without context
  - Black/dark backgrounds (blends with UI)
```

### Subtitle/Text Overlay

```
Position: Center or lower third (avoid bottom 15% — UI overlay zone)
Font: Bold sans-serif, white with black outline/shadow
Size: ≥36pt (visible on mobile)
Timing: Sync with speech, 1-2 lines max
Style: Highlight keywords in accent color (yellow, red, green)
```

---

## Caption Rules

> **TikTok 没有独立标题字段。** 只有 Caption（描述文案），即发布时的文字输入区。Caption 兼具标题和正文功能，前 80 字符决定信息传达效率。不要在 Agent 调用中传递 title 参数。

### Caption Writing

| Rule | Detail |
|------|--------|
| Max Length | 4,000 characters (as of 2025) |
| Recommended | 50-150 characters for best engagement |
| Visible | First ~80 characters before "more" truncation |
| Keywords | Place primary keyword in first 40 characters |
| Line Breaks | Use for readability — TikTok renders them |

### Caption Formula

```
[Hook line — grab attention]

[Value/Content summary — 1-2 sentences]

[CTA — follow, save, share, or comment prompt]

[Hashtags — 3-5 relevant tags]
```

### Example Captions

```
Tutorial:
  "The Excel trick your boss doesn't know about 👀
   This saved me 3 hours every week.
   Save this & try it tomorrow.
   #ExcelTips #Productivity #WorkHack #LearnOnTikTok"

Storytelling:
  "I quit my 9-5 to build an app. Month 3 update.
   Revenue: $0 → $4,200. Here's exactly what I did.
   Follow for month 4 🔔
   #BuildInPublic #StartupLife #Entrepreneur"

Trend/Entertainment:
  "Tell me you're a developer without telling me 💀
   Comment yours ⬇️
   #DevLife #ProgrammerHumor #Tech #fyp"
```

---

## Hashtag Strategy

### Hashtag Best Practices

```
Count: 3-5 hashtags (sweet spot for reach)
Placement: End of caption (keeps caption clean)
Mix:
  1 broad/discovery tag:  #fyp #foryou #viral
  1-2 niche/topic tags:   #CodingTips #StartupLife #AITools
  1 trend/challenge tag:  Current trending hashtag
  0-1 branded tag:        Your own recurring tag

Max: TikTok allows up to 100 characters for all hashtags combined
```

### High-Performance Tags

```
Discovery:
  #fyp #foryou #foryoupage #viral #trending

Engagement:
  #LearnOnTikTok #TikTokMadeMeBuyIt #tutorial

Niche (examples):
  Tech: #TechTok #CodingTips #AI #WebDev
  Business: #SmallBusiness #Entrepreneur #SideHustle
  Finance: #MoneyTok #Investing #CryptoTok
  Lifestyle: #LifeHack #Productivity #Motivation

Avoid:
  - Banned/restricted hashtags (check before using)
  - Irrelevant trending tags (algorithm penalizes mismatch)
  - Too many tags (>8 looks spammy, dilutes signal)
```

---

## Posting Schedule

### Best Times (UTC)

| Time (UTC) | Audience | Traffic |
|------------|----------|---------|
| 13:00-15:00 | US morning + EU afternoon | ★★★★ |
| 17:00-19:00 | US afternoon + EU evening | ★★★★★ |
| 22:00-01:00 | US evening prime time | ★★★★★ |
| 06:00-08:00 | EU morning + Asia evening | ★★★ |

### Posting Frequency

```
Recommended:
  - 1-3 videos per day (consistency > volume)
  - At least 4 per week minimum
  - Post at same times to train audience

Growth Phase:
  - 2-3 per day for first 30 days
  - Test different times, track analytics

Stable Phase:
  - 1 per day, high quality
  - Supplement with Stories/Photos
```

---

## Content Compliance

See → `references/content-rules.md`

### Community Guidelines Summary

| Category | Risk | Action |
|----------|------|--------|
| Copyright music | High | Use TikTok Commercial Music Library or original |
| Health claims | High | No medical advice without "not professional advice" |
| Income claims | High | No "guaranteed income" / "make $X fast" |
| Political content | Medium | Labeled and may have limited distribution |
| Branded content | Medium | Must use Branded Content toggle |
| External links | Medium | Only in bio; no QR codes in videos |
| Minors | Critical | No content featuring/targeting minors inappropriately |

### Shadowban Indicators

```
Signs:
  - 0 views after 1 hour (normally 200-500 initial push)
  - Videos not showing in hashtag feeds
  - Sudden engagement drop (>80% decrease)

Common Causes:
  - Posting copyrighted content
  - Spam behavior (mass following/unfollowing)
  - Using banned hashtags
  - Repeated Community Guidelines violations
  - Bot-like activity patterns

Recovery:
  - Stop posting for 24-72 hours
  - Remove flagged content
  - Post original, high-quality content
  - Avoid external links for 1-2 weeks
```

---

## Algorithm Optimization

See → `references/algorithm-guide.md`

### For You Page (FYP) Ranking Signals

```
Weight (estimated):
  1. Watch Time / Completion Rate — #1 signal, videos watched to end get boosted
  2. Re-watches — Users watching again = strong positive signal
  3. Shares — Highest-weight engagement action
  4. Comments — High-weight, especially lengthy ones
  5. Saves/Favorites — "Bookmark" indicates high-value content
  6. Likes — Base engagement, lower weight than above
  7. Profile Visits — from video = interest signal
  8. Follow-from-video — Strong conversion signal
```

### Content Lifecycle

```
Phase 1: Initial Push (0-1h)
  - Shown to 200-500 users (small test pool)
  - Algorithm measures: completion rate, engagement rate
  - Target: >50% completion, >5% engagement

Phase 2: Expansion (1-24h)
  - If Phase 1 metrics pass threshold → 5K-50K pool
  - New signals: share rate, comment quality
  - Target: maintain engagement ratios

Phase 3: Viral Potential (24h-7d)
  - Strong Phase 2 → 100K+ pool
  - Content enters broader FYP
  - Can continue growing for days/weeks

Phase 4: Evergreen (7d+)
  - High-performing content resurfaces periodically
  - Search-optimized content has longest shelf life
```

### Completion Rate Tactics

| Position | Strategy |
|----------|----------|
| 0-1s | Pattern interrupt: unexpected visual/sound/text |
| 1-3s | Hook: promise value ("Watch till the end for...") |
| 3-10s | Deliver first value point quickly |
| Mid | Maintain tension: "But here's where it gets interesting" |
| Pre-end | Tease payoff: "The last one is the best" |
| End | Reward + CTA: deliver promise, ask for follow/save |

---

## Output Format

```
🎬 TikTok Post:
Caption:
[Hook line]
[Value summary]
[CTA]
[#tag1 #tag2 #tag3]

Estimated Duration: {X}s

🖼️ Cover Text: [3-5 word overlay text]
🎵 Sound Suggestion: [trending sound / original / music type]

⏰ Best Post Time: [time window in UTC]

⚠️ Compliance Check:
  - Banned words: ✅ None
  - Copyright risk: ✅ Clear
  - Branded content: ✅ Toggled if needed
  - External links: ✅ No URLs in caption or video (GitHub, website, etc.)
  - QR codes: ✅ None in video frames
```
