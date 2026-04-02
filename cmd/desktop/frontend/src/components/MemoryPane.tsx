import { useEffect, useRef, useState } from "react";
import {
  ListMemoryFiles,
  GetMemoryFileContent,
  SaveMemoryFileContent,
  DeleteMemoryFile,
  CreateMemoryFile,
  SearchMemory,
  SyncMemory,
  GetMemoryDir,
} from "../wailsjs/go/app/App";
import type { MemoryFile, MemorySearchResult } from "../wailsjs/go/app/App";

type Tab = "files" | "search";

export function MemoryPane() {
  const [tab, setTab] = useState<Tab>("files");
  const [memDir, setMemDir] = useState("");

  useEffect(() => {
    GetMemoryDir().then(setMemDir).catch(() => {});
  }, []);

  return (
    <div className="p-8 h-full flex flex-col">
      <header className="mb-5 flex-shrink-0 flex items-start justify-between">
        <div>
          <h2 className="text-[22px] font-semibold text-[#3d3929] tracking-[-0.43px]">Memory</h2>
          <p className="text-[11px] text-[rgba(61,57,41,0.2)] mt-0.5 font-mono">{memDir}</p>
        </div>
        <SyncButton />
      </header>

      {/* Tabs */}
      <div className="flex gap-1 mb-5 flex-shrink-0">
        {(["files", "search"] as const).map((t) => (
          <button key={t} onClick={() => setTab(t)}
            className={`px-4 py-1.5 rounded-lg text-[13px] font-medium transition-colors ${
              tab === t
                ? "bg-[rgba(200,90,42,0.15)] text-[#c85a2a]"
                : "text-[rgba(61,57,41,0.5)] hover:bg-[rgba(61,57,41,0.05)] hover:text-[#3d3929]"
            }`}>
            {t === "files" ? "Files" : "Search"}
          </button>
        ))}
      </div>

      <div className="flex-1 min-h-0 overflow-hidden">
        {tab === "files" && <FilesTab />}
        {tab === "search" && <SearchTab />}
      </div>
    </div>
  );
}

// ─── Sync button ─────────────────────────────────────────────────────────────

function SyncButton() {
  const [syncing, setSyncing] = useState(false);
  const [ok, setOk] = useState(false);

  async function handleSync() {
    setSyncing(true); setOk(false);
    try {
      await SyncMemory();
      setOk(true);
      setTimeout(() => setOk(false), 2000);
    } catch {}
    finally { setSyncing(false); }
  }

  return (
    <button onClick={handleSync} disabled={syncing}
      className="text-[12px] px-3 py-1.5 rounded-lg border border-[rgba(61,57,41,0.12)] text-[rgba(61,57,41,0.5)] hover:bg-[rgba(61,57,41,0.05)] hover:text-[#3d3929] disabled:opacity-40 transition-colors">
      {syncing ? "Syncing…" : ok ? "✓ Synced" : "Sync"}
    </button>
  );
}

// ─── Files tab ────────────────────────────────────────────────────────────────

function FilesTab() {
  const [files, setFiles] = useState<MemoryFile[]>([]);
  const [loading, setLoading] = useState(true);
  const [selected, setSelected] = useState<MemoryFile | null>(null);
  const [content, setContent] = useState("");
  const [saving, setSaving] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [error, setError] = useState("");
  const [creating, setCreating] = useState(false);
  const [newName, setNewName] = useState("");
  const editorRef = useRef<HTMLTextAreaElement>(null);

  async function load() {
    setLoading(true);
    try { setFiles((await ListMemoryFiles()) ?? []); }
    catch { setFiles([]); }
    finally { setLoading(false); }
  }

  useEffect(() => { load(); }, []);

  async function openFile(f: MemoryFile) {
    if (dirty && selected) {
      if (!confirm("Discard unsaved changes?")) return;
    }
    setSelected(f);
    setDirty(false);
    setError("");
    try {
      const c = await GetMemoryFileContent(f.path);
      setContent(c);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function handleSave() {
    if (!selected) return;
    setSaving(true); setError("");
    try {
      await SaveMemoryFileContent(selected.path, content);
      setDirty(false);
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    } finally { setSaving(false); }
  }

  async function handleDelete(f: MemoryFile) {
    if (!confirm(`Delete "${f.name}"?`)) return;
    try {
      await DeleteMemoryFile(f.path);
      if (selected?.path === f.path) { setSelected(null); setContent(""); setDirty(false); }
      await load();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function handleCreate() {
    const n = newName.trim();
    if (!n) return;
    try {
      const path = await CreateMemoryFile(n);
      setCreating(false); setNewName("");
      await load();
      const fresh = (await ListMemoryFiles()) ?? [];
      const f = fresh.find((x) => x.path === path);
      if (f) openFile(f);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div className="flex gap-4 h-full">
      {/* File list */}
      <div className="w-52 flex-shrink-0 flex flex-col gap-1 overflow-y-auto">
        <button onClick={() => setCreating(true)}
          className="w-full text-[12px] px-3 py-1.5 rounded-lg bg-[#c85a2a] text-white hover:bg-[#a84a22] transition-colors font-medium mb-1 flex-shrink-0">
          + New file
        </button>

        {creating && (
          <div className="flex gap-1 mb-1 flex-shrink-0">
            <input autoFocus value={newName} onChange={(e) => setNewName(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter") handleCreate(); if (e.key === "Escape") { setCreating(false); setNewName(""); } }}
              placeholder="filename.md"
              className="flex-1 min-w-0 px-2 py-1 text-[12px] bg-[rgba(61,57,41,0.1)] border border-[rgba(200,90,42,0.5)] rounded-lg text-[#3d3929] placeholder-[rgba(61,57,41,0.3)] focus:outline-none" />
            <button onClick={handleCreate} className="px-2 py-1 text-[11px] bg-[rgba(200,90,42,0.2)] text-[#c85a2a] rounded-lg hover:bg-[rgba(200,90,42,0.3)]">OK</button>
          </div>
        )}

        {loading ? (
          <p className="text-[12px] text-[rgba(61,57,41,0.3)] px-1">Loading…</p>
        ) : files.length === 0 ? (
          <p className="text-[12px] text-[rgba(61,57,41,0.3)] px-1 italic">No files yet.</p>
        ) : (
          files.map((f) => (
            <button key={f.path} onClick={() => openFile(f)}
              className={`w-full text-left px-3 py-2 rounded-lg transition-colors group flex items-center justify-between gap-1 ${
                selected?.path === f.path
                  ? "bg-[rgba(200,90,42,0.12)] border border-[rgba(200,90,42,0.25)]"
                  : "hover:bg-[rgba(61,57,41,0.05)]"
              }`}>
              <div className="min-w-0">
                <p className={`text-[12px] font-medium truncate ${selected?.path === f.path ? "text-[#c85a2a]" : "text-[rgba(61,57,41,0.8)]"}`}>{f.name}</p>
                <p className="text-[10px] text-[rgba(61,57,41,0.2)] mt-0.5">{f.chunkCount} chunk{f.chunkCount !== 1 ? "s" : ""}</p>
              </div>
              <button onClick={(e) => { e.stopPropagation(); handleDelete(f); }}
                className="opacity-0 group-hover:opacity-100 text-[10px] text-[rgba(255,80,80,0.7)] hover:text-red-400 transition-all px-1 flex-shrink-0">
                ✕
              </button>
            </button>
          ))
        )}
      </div>

      {/* Editor */}
      <div className="flex-1 min-w-0 flex flex-col">
        {selected ? (
          <>
            <div className="flex items-center justify-between mb-2 flex-shrink-0">
              <p className="text-[12px] text-[rgba(61,57,41,0.4)] font-mono truncate">{selected.path}</p>
              <div className="flex items-center gap-2 flex-shrink-0">
                {dirty && <span className="text-[11px] text-amber-400">Unsaved</span>}
                <button onClick={handleSave} disabled={saving || !dirty}
                  className="text-[12px] px-3 py-1.5 bg-[#c85a2a] text-white rounded-lg hover:bg-[#a84a22] disabled:opacity-40 transition-colors font-medium">
                  {saving ? "Saving…" : "Save"}
                </button>
              </div>
            </div>
            <textarea
              ref={editorRef}
              value={content}
              onChange={(e) => { setContent(e.target.value); setDirty(true); }}
              spellCheck={false}
              className="flex-1 min-h-0 w-full px-4 py-3 text-[13px] font-mono bg-[rgba(61,57,41,0.04)] border border-[rgba(61,57,41,0.08)] rounded-xl focus:outline-none focus:ring-2 focus:ring-[rgba(200,90,42,0.3)] resize-none leading-relaxed text-[rgba(61,57,41,0.85)] placeholder-[rgba(61,57,41,0.15)]"
              placeholder="Write Markdown here…"
            />
            {error && <p className="text-[11px] text-red-400 mt-2 flex-shrink-0">{error}</p>}
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center text-[rgba(61,57,41,0.3)]">
            <div className="text-center">
              <div className="text-3xl mb-2">🧠</div>
              <p className="text-[13px]">Select a file to edit</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// ─── Search tab ───────────────────────────────────────────────────────────────

function SearchTab() {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<MemorySearchResult[]>([]);
  const [searching, setSearching] = useState(false);
  const [searched, setSearched] = useState(false);
  const [error, setError] = useState("");

  async function handleSearch(e: React.FormEvent) {
    e.preventDefault();
    if (!query.trim()) return;
    setSearching(true); setError(""); setSearched(false);
    try {
      const r = await SearchMemory(query.trim(), 10);
      setResults(r ?? []);
      setSearched(true);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    } finally { setSearching(false); }
  }

  return (
    <div className="flex flex-col h-full">
      <form onSubmit={handleSearch} className="flex gap-2 mb-4 flex-shrink-0">
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search memory…"
          className="flex-1 px-4 py-2 text-[13px] bg-[rgba(61,57,41,0.05)] border border-[rgba(61,57,41,0.12)] rounded-xl focus:outline-none focus:ring-2 focus:ring-[rgba(200,90,42,0.4)] text-[#3d3929] placeholder-[rgba(61,57,41,0.2)]"
        />
        <button type="submit" disabled={searching || !query.trim()}
          className="px-5 py-2 bg-[#c85a2a] text-white text-[13px] font-semibold rounded-xl hover:bg-[#a84a22] disabled:opacity-40 transition-colors">
          {searching ? "…" : "Search"}
        </button>
      </form>

      {error && <p className="text-[12px] text-red-400 mb-3 flex-shrink-0">{error}</p>}

      <div className="flex-1 min-h-0 overflow-y-auto space-y-3">
        {searched && results.length === 0 && (
          <div className="text-center py-12 text-[rgba(61,57,41,0.2)]">
            <div className="text-2xl mb-2">🔍</div>
            <p className="text-[13px]">No results found.</p>
          </div>
        )}
        {results.map((r, i) => (
          <SearchResultCard key={i} result={r} />
        ))}
      </div>
    </div>
  );
}

function SearchResultCard({ result }: { result: MemorySearchResult }) {
  const [expanded, setExpanded] = useState(false);
  const pct = Math.round(result.score * 100);

  return (
    <div className="rounded-xl bg-[rgba(61,57,41,0.04)] border border-[rgba(61,57,41,0.1)] overflow-hidden">
      <button className="w-full text-left px-4 py-3 hover:bg-[rgba(61,57,41,0.04)] transition-colors"
        onClick={() => setExpanded(!expanded)}>
        <div className="flex items-center justify-between gap-3">
          <div className="min-w-0 flex-1">
            <p className="text-[12px] font-mono text-[rgba(61,57,41,0.5)] truncate">
              {result.filePath.split("/").pop()} · lines {result.startLine}–{result.endLine}
            </p>
            <p className="text-[13px] text-[rgba(61,57,41,0.8)] mt-0.5 line-clamp-2 leading-snug">
              {result.content.slice(0, 160)}{result.content.length > 160 ? "…" : ""}
            </p>
          </div>
          <div className="flex-shrink-0 flex items-center gap-2">
            <ScoreBadge pct={pct} />
            <span className="text-[rgba(61,57,41,0.3)] text-[11px]">{expanded ? "▲" : "▼"}</span>
          </div>
        </div>
      </button>
      {expanded && (
        <div className="border-t border-[rgba(61,57,41,0.08)] px-4 py-3">
          <p className="text-[11px] text-[rgba(61,57,41,0.35)] font-mono mb-2">{result.filePath}</p>
          <pre className="text-[12px] text-[rgba(61,57,41,0.7)] whitespace-pre-wrap break-words font-mono leading-relaxed max-h-48 overflow-y-auto">
            {result.content}
          </pre>
        </div>
      )}
    </div>
  );
}

function ScoreBadge({ pct }: { pct: number }) {
  const cls =
    pct >= 70 ? "bg-[rgba(52,199,89,0.15)] text-emerald-400" :
    pct >= 40 ? "bg-[rgba(255,179,64,0.15)] text-amber-400" :
               "bg-[rgba(61,57,41,0.1)] text-[rgba(61,57,41,0.4)]";
  return <span className={`text-[10px] font-mono px-1.5 py-0.5 rounded ${cls}`}>{pct}%</span>;
}
