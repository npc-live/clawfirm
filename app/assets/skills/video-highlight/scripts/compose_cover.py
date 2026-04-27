#!/usr/bin/env python3
"""
Generate YouTube-style video cover thumbnail from a real video frame.

Usage:
  python compose_cover.py --frame FRAME_PATH --text1 "主标题" --text2 "副标题" --output OUTPUT_PATH [--style bold|block|split]

Styles:
  bold   - Large white text with black stroke, yellow subtitle (default)
  block  - Text on solid colored blocks (high contrast)
  split  - Text split across upper/lower, dramatic

Requirements: pip install Pillow
"""

import sys
import argparse
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont, ImageFilter, ImageEnhance

# Font candidates — ordered by preference (bold first)
FONT_CANDIDATES = [
    # macOS system fonts
    ("/System/Library/Fonts/PingFang.ttc", 0),           # PingFang SC Semibold
    ("/System/Library/Fonts/PingFang.ttc", 3),           # PingFang SC Bold
    ("/Library/Fonts/Songti.ttc", 0),
    ("/System/Library/Fonts/STHeiti Medium.ttc", 0),
    ("/System/Library/Fonts/Supplemental/Arial Bold.ttf", -1),
    # Linux
    ("/usr/share/fonts/truetype/noto/NotoSansCJK-Bold.ttc", 0),
    ("/usr/share/fonts/opentype/noto/NotoSansCJKsc-Bold.otf", -1),
    ("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", -1),
]


def get_font(size: int) -> ImageFont.FreeTypeFont:
    for font_path, idx in FONT_CANDIDATES:
        if not Path(font_path).exists():
            continue
        try:
            if idx >= 0:
                return ImageFont.truetype(font_path, size, index=idx)
            else:
                return ImageFont.truetype(font_path, size)
        except Exception:
            continue
    return ImageFont.load_default()


def text_size(draw: ImageDraw.ImageDraw, text: str, font) -> tuple[int, int]:
    bbox = draw.textbbox((0, 0), text, font=font)
    return bbox[2] - bbox[0], bbox[3] - bbox[1]


def draw_stroked_text(draw: ImageDraw.ImageDraw, xy: tuple, text: str, font,
                       fill: tuple, stroke_fill: tuple, stroke_width: int = 5):
    """Draw text with thick outline/stroke for readability on any background."""
    x, y = xy
    # Stroke: draw in a circle pattern for smooth edges
    for dx in range(-stroke_width, stroke_width + 1):
        for dy in range(-stroke_width, stroke_width + 1):
            if dx * dx + dy * dy <= stroke_width * stroke_width + stroke_width:
                draw.text((x + dx, y + dy), text, font=font, fill=stroke_fill)
    draw.text((x, y), text, font=font, fill=fill)


def draw_text_with_shadow(draw: ImageDraw.ImageDraw, xy: tuple, text: str, font,
                           fill: tuple, shadow_offset: int = 4):
    """Draw text with drop shadow."""
    x, y = xy
    draw.text((x + shadow_offset, y + shadow_offset), text, font=font, fill=(0, 0, 0, 180))
    draw.text((x, y), text, font=font, fill=fill)


def style_bold(img: Image.Image, text1: str, text2: str) -> Image.Image:
    """
    Large white text with black stroke, yellow accent for subtitle.
    This is the classic YouTube thumbnail style.
    """
    draw = ImageDraw.Draw(img)
    w, h = img.size  # 1920×1080

    if text1:
        font_size = max(100, min(180, int(w * 0.09)))
        font = get_font(font_size)
        tw, th = text_size(draw, text1, font)
        # Adjust font size to fit width
        while tw > w * 0.9 and font_size > 60:
            font_size -= 10
            font = get_font(font_size)
            tw, th = text_size(draw, text1, font)

        x = (w - tw) // 2
        y = int(h * 0.70) if text2 else int(h * 0.78)
        draw_stroked_text(draw, (x, y), text1, font,
                           fill=(255, 255, 255), stroke_fill=(0, 0, 0), stroke_width=7)

    if text2:
        font_size2 = max(70, min(110, int(w * 0.058)))
        font2 = get_font(font_size2)
        tw2, th2 = text_size(draw, text2, font2)
        while tw2 > w * 0.85 and font_size2 > 50:
            font_size2 -= 8
            font2 = get_font(font_size2)
            tw2, th2 = text_size(draw, text2, font2)

        x2 = (w - tw2) // 2
        y2 = int(h * 0.855)
        draw_stroked_text(draw, (x2, y2), text2, font2,
                           fill=(255, 220, 0), stroke_fill=(0, 0, 0), stroke_width=5)

    return img


def style_block(img: Image.Image, text1: str, text2: str) -> Image.Image:
    """
    Text on solid colored background blocks — high contrast, works on any frame.
    """
    draw = ImageDraw.Draw(img, "RGBA")
    w, h = img.size

    entries = []
    if text1:
        entries.append((text1, (0, 90, 200, 230), (255, 255, 255)))
    if text2:
        entries.append((text2, (255, 200, 0, 230), (20, 20, 20)))

    # Stack blocks from bottom
    y_base = int(h * 0.75)
    pad_x = 32
    pad_y = 18

    for i, (text, bg_color, fg_color) in enumerate(entries):
        font_size = 100 if i == 0 else 78
        font = get_font(font_size)
        tw, th = text_size(draw, text, font)
        while tw > w * 0.85 and font_size > 50:
            font_size -= 8
            font = get_font(font_size)
            tw, th = text_size(draw, text, font)

        block_w = tw + pad_x * 2
        block_h = th + pad_y * 2
        x = (w - block_w) // 2
        y = y_base + i * (block_h + 12)

        # Draw block
        draw.rectangle([x, y, x + block_w, y + block_h], fill=bg_color)
        # Draw text
        draw.text((x + pad_x, y + pad_y), text, font=font, fill=fg_color)

    return img.convert("RGB")


def style_split(img: Image.Image, text1: str, text2: str) -> Image.Image:
    """
    Dramatic split: text1 at top-left, text2 at bottom-right.
    """
    draw = ImageDraw.Draw(img)
    w, h = img.size

    if text1:
        font = get_font(150)
        tw, th = text_size(draw, text1, font)
        while tw > w * 0.6 and True:
            font = get_font(int(font.size * 0.9))
            tw, th = text_size(draw, text1, font)
            if font.size < 80:
                break
        draw_stroked_text(draw, (60, int(h * 0.08)), text1, font,
                           fill=(255, 255, 255), stroke_fill=(0, 0, 0), stroke_width=8)

    if text2:
        font2 = get_font(120)
        tw2, th2 = text_size(draw, text2, font2)
        while tw2 > w * 0.6 and True:
            font2 = get_font(int(font2.size * 0.9))
            tw2, th2 = text_size(draw, text2, font2)
            if font2.size < 70:
                break
        x2 = w - tw2 - 60
        y2 = int(h * 0.78)
        draw_stroked_text(draw, (x2, y2), text2, font2,
                           fill=(255, 220, 0), stroke_fill=(0, 0, 0), stroke_width=7)

    return img


STYLES = {
    "bold": style_bold,
    "block": style_block,
    "split": style_split,
}


def compose(frame_path: str, text1: str, text2: str, output_path: str,
            style: str = "bold", width: int = 1920, height: int = 1080,
            blur_bg: int = 0):
    # Load frame
    img = Image.open(frame_path).convert("RGB")

    # Resize to target dimensions (crop to fill, no black bars)
    src_ratio = img.width / img.height
    tgt_ratio = width / height
    if src_ratio > tgt_ratio:
        new_h = height
        new_w = int(img.width * height / img.height)
    else:
        new_w = width
        new_h = int(img.height * width / img.width)

    img = img.resize((new_w, new_h), Image.LANCZOS)
    left = (new_w - width) // 2
    top = (new_h - height) // 2
    img = img.crop((left, top, left + width, top + height))

    # Optional: blur background to obscure busy/text-heavy video frames
    if blur_bg > 0:
        img = img.filter(ImageFilter.GaussianBlur(radius=blur_bg))
        # Slightly reduce saturation for a more atmospheric look
        img = ImageEnhance.Color(img).enhance(0.7)

    # Darken lower half for text readability
    overlay = Image.new("RGBA", img.size, (0, 0, 0, 0))
    draw_ov = ImageDraw.Draw(overlay)
    gradient_top = int(height * 0.50)
    for y in range(gradient_top, height):
        alpha = int(160 * (y - gradient_top) / (height - gradient_top))
        draw_ov.line([(0, y), (width, y)], fill=(0, 0, 0, alpha))
    img = Image.alpha_composite(img.convert("RGBA"), overlay).convert("RGB")

    # Apply style
    style_fn = STYLES.get(style, style_bold)
    img = style_fn(img, text1, text2)

    # Save
    Path(output_path).parent.mkdir(parents=True, exist_ok=True)
    img.save(output_path, "PNG")
    print(f"✓ Cover saved: {output_path}")
    print(f"  Size: {width}×{height}")
    print(f"  Style: {style}")
    if text1:
        print(f"  Text1: {text1}")
    if text2:
        print(f"  Text2: {text2}")


def main():
    parser = argparse.ArgumentParser(description="Compose YouTube-style video cover thumbnail")
    parser.add_argument("--frame", required=True, help="Path to hero frame image")
    parser.add_argument("--text1", default="", help="Main title text")
    parser.add_argument("--text2", default="", help="Subtitle text")
    parser.add_argument("--output", required=True, help="Output PNG path")
    parser.add_argument("--style", default="bold", choices=list(STYLES.keys()),
                        help="Thumbnail style")
    parser.add_argument("--width", type=int, default=1920, help="Output width (default 1920)")
    parser.add_argument("--height", type=int, default=1080, help="Output height (default 1080)")
    parser.add_argument("--blur-bg", type=int, default=0,
                        help="Gaussian blur radius for background (0=off, 8=light, 18=heavy). "
                             "Use 10-18 when video frame has busy subtitles/text you want to obscure.")
    args = parser.parse_args()

    compose(args.frame, args.text1, args.text2, args.output, args.style,
            args.width, args.height, args.blur_bg)


if __name__ == "__main__":
    main()
