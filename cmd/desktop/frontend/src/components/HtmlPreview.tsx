interface Props {
  html: string;
}

const TAILWIND_CDN = `<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/tailwindcss@3/dist/tailwind.min.css">`;

export function HtmlPreview({ html }: Props) {
  // Inject Tailwind CDN into <head> if not already present
  let enriched = html;
  if (!html.includes("tailwindcss")) {
    if (html.includes("</head>")) {
      enriched = html.replace("</head>", `${TAILWIND_CDN}\n</head>`);
    } else if (html.includes("<html")) {
      enriched = html.replace(/<html([^>]*)>/, `<html$1><head>${TAILWIND_CDN}</head>`);
    } else {
      enriched = `<!DOCTYPE html><html><head>${TAILWIND_CDN}</head><body>${html}</body></html>`;
    }
  }

  return (
    <iframe
      srcDoc={enriched}
      sandbox="allow-scripts"
      title="HTML Preview"
      style={{
        width: "100%",
        height: "100%",
        border: "none",
        borderRadius: "8px",
        background: "#fff",
      }}
    />
  );
}
