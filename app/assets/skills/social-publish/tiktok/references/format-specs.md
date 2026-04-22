# TikTok Format Specifications

## Video Parameters

### Core Specs

| Item | Specification |
|------|---------------|
| Aspect Ratio | 9:16 (vertical, primary), 16:9 (horizontal), 1:1 (square) |
| Resolution | 1080×1920 (vertical) / 1920×1080 (horizontal) |
| Format | MP4, MOV, WebM |
| Max Size | 10GB (web upload), 287MB (mobile upload) |
| Frame Rate | 30fps recommended, 60fps supported |
| Codec | H.264 (recommended), H.265 supported |
| Bitrate | ≥5Mbps for 1080p |
| Audio | AAC, 44100Hz, ≥128kbps |

### Duration Limits

| Duration | Availability | Recommended For |
|----------|-------------|-----------------|
| ≤15s | All accounts | Trending sounds, quick tips, memes |
| ≤60s | All accounts | Tutorials, storytelling, POV skits |
| ≤3min | All accounts | In-depth tutorials, reviews |
| ≤10min | Most accounts | Full tutorials, series |
| ≤60min | Select accounts | Long-form content (2025+ rollout) |

---

## Cover / Thumbnail

### Vertical Cover (Primary)

```
Size: 1080×1920 (9:16)
Format: JPG, PNG
Max Size: 5MB
Display: Profile grid, FYP preview

Design:
  - Bold text: 3-5 words max
  - Font: Bold sans-serif, high contrast
  - Face visible when possible (higher CTR)
  - Clean background, not cluttered
  - Text safe zone: center 80% (avoid edges)
```

### Horizontal Cover

```
Size: 1920×1080 (16:9)
Format: JPG, PNG
Max Size: 5MB
Display: Desktop browser, embed/share previews, search results

Design:
  - Same principles as vertical
  - Center important elements (may be cropped on mobile)
  - Ensure text readable at small sizes
```

### Cover Best Practices

```
DO:
  - Use bright, saturated colors
  - Include 1 clear focal point
  - Match cover style across videos (brand consistency)
  - Test different styles and track CTR
  - Use numbers in text ("5 Tips", "Day 30")

DON'T:
  - Use small/unreadable text
  - Overcrowd with multiple elements
  - Use pure black or very dark backgrounds
  - Include other platform logos/watermarks
  - Put text in bottom 15% (UI overlap)
```

---

## Text / Subtitle Overlay

### Text Specs

| Item | Specification |
|------|---------------|
| Position | Center or lower third; avoid bottom 15% (TikTok UI) |
| Font | Bold sans-serif (Impact, Montserrat, etc.) |
| Color | White + black outline/drop shadow |
| Size | ≥36pt (must be readable on 4" phone screen) |
| Max Lines | 2 lines per screen |
| Duration | 2-4 seconds per text card |

### Safe Zones

```
Vertical (1080×1920):
  Top safe: ≥150px from top (status bar + back button)
  Bottom safe: ≥280px from bottom (caption, buttons, nav bar)
  Left safe: ≥40px
  Right safe: ≥100px (like/comment/share buttons)

Text placement:
  Best: Y 400-1400px (center area)
  Avoid: Y 0-150px (top UI), Y 1640-1920px (bottom UI)
```

---

## Caption Format

### Text

| Item | Limit |
|------|-------|
| Max Characters | 4,000 (including hashtags and mentions) |
| Recommended | 50-150 characters for engagement |
| Visible Before Fold | ~80 characters (before "more" link) |
| Line Breaks | Supported and rendered |
| Emoji | Supported, use sparingly (1-3 per caption) |
| Mentions | @username supported |
| Hashtags | # supported, count toward character limit |

### Caption Template

```
[Hook — first 80 chars visible]
[blank line]
[Value/detail — 1-2 sentences]
[blank line]
[CTA — save/share/follow]
[blank line]
#tag1 #tag2 #tag3 #tag4 #tag5
```

---

## Audio

### Background Music

```
Sources (safe for commercial use):
  - TikTok Commercial Music Library
  - Epidemic Sound / Artlist (with license)
  - Royalty-free libraries (Pixabay, Free Music Archive)
  - Original compositions

Volume Mix:
  - Voice: 70-80% of total
  - Background music: 20-30% of total
  - Ensure voice is always clearly audible

Trending Sounds:
  - Using trending sounds boosts discoverability
  - Check TikTok Creative Center for current trends
  - Personal accounts: wider sound library access
  - Business accounts: limited to Commercial Music Library
```

### Voice Recording

```
Recommended:
  - Lavalier mic or directional mic
  - Sample rate: 44100Hz / 48000Hz
  - Bit depth: 16-bit
  - Noise reduction applied
  - Consistent volume throughout

TikTok Voice Features:
  - Text-to-speech (multiple voices available)
  - Voice effects (various filters)
  - Original sound creation
```

---

## Photo / Carousel Mode

### Photo Post Specs

| Item | Specification |
|------|---------------|
| Max Photos | 35 per post |
| Format | JPG, PNG |
| Aspect Ratio | 9:16 recommended |
| Resolution | 1080×1920 recommended |
| Display | Auto-slideshow with music |
