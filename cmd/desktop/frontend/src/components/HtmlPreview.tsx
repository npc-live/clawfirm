import { marked } from "marked";

interface Props {
  html: string;
}

const TAILWIND_CDN = `<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/tailwindcss@3/dist/tailwind.min.css">`;
const GITHUB_MD_STYLE = `<style>
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; padding: 16px 24px; color: #24292f; line-height: 1.6; }
  h1,h2,h3,h4,h5,h6 { margin: 1em 0 0.4em; font-weight: 600; }
  h1 { font-size: 1.8em; border-bottom: 1px solid #d0d7de; padding-bottom: 0.3em; }
  h2 { font-size: 1.4em; border-bottom: 1px solid #d0d7de; padding-bottom: 0.2em; }
  p { margin: 0.6em 0; }
  pre { background: #f6f8fa; border-radius: 6px; padding: 12px 16px; overflow-x: auto; }
  code { background: #f6f8fa; border-radius: 4px; padding: 2px 6px; font-size: 0.88em; font-family: 'SFMono-Regular', Consolas, monospace; }
  pre code { background: none; padding: 0; }
  blockquote { border-left: 4px solid #d0d7de; margin: 0; padding: 0 12px; color: #57606a; }
  ul, ol { padding-left: 1.5em; }
  table { border-collapse: collapse; width: 100%; }
  th, td { border: 1px solid #d0d7de; padding: 6px 12px; }
  th { background: #f6f8fa; font-weight: 600; }
  a { color: #0969da; text-decoration: none; }
  a:hover { text-decoration: underline; }
  img { max-width: 100%; }
  hr { border: none; border-top: 1px solid #d0d7de; margin: 1em 0; }
</style>`;

function isMarkdown(content: string): boolean {
  const trimmed = content.trim();
  // If it looks like an HTML document, treat as HTML
  if (/^<!doctype\s+html/i.test(trimmed) || /^<html/i.test(trimmed)) return false;
  // If it starts with an HTML tag, treat as HTML
  if (/^<[a-z][a-z0-9]*[\s>]/i.test(trimmed)) return false;
  return true;
}

function markdownToHtml(md: string): string {
  const body = marked.parse(md) as string;
  return `<!DOCTYPE html><html><head><meta charset="utf-8">${GITHUB_MD_STYLE}</head><body>${body}</body></html>`;
}

export function HtmlPreview({ html }: Props) {
  let enriched: string;

  if (isMarkdown(html)) {
    enriched = markdownToHtml(html);
  } else {
    enriched = html;
    const fillStyle = `<style>html,body{min-height:100%;}</style>`;
    if (!html.includes("tailwindcss")) {
      if (html.includes("</head>")) {
        enriched = html.replace("</head>", `${TAILWIND_CDN}\n${fillStyle}\n</head>`);
      } else if (html.includes("<html")) {
        enriched = html.replace(/<html([^>]*)>/, `<html$1><head>${TAILWIND_CDN}${fillStyle}</head>`);
      } else {
        enriched = `<!DOCTYPE html><html><head>${TAILWIND_CDN}${fillStyle}</head><body>${html}</body></html>`;
      }
    } else {
      if (html.includes("</head>")) {
        enriched = html.replace("</head>", `${fillStyle}\n</head>`);
      }
    }
  }

  return (
    <iframe
      srcDoc={enriched}
      sandbox="allow-scripts"
      title="Preview"
      style={{
        width: "100%",
        height: "100%",
        border: "none",
        borderRadius: "8px",
        background: "transparent",
      }}
    />
  );
}
