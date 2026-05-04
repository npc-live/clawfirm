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
  GetConfig,
  SaveChannelConfig,
  GetSoulPrompt,
  SaveSoulPrompt,
} from "../lib/wails-shim";
import type { MemoryFile, MemorySearchResult } from "../lib/wails-shim";

type Tab = "files" | "search" | "prompts";

export function MemoryPane() {
  const [tab, setTab] = useState<Tab>("files");
  const [memDir, setMemDir] = useState("");

  useEffect(() => {
    GetMemoryDir().then(setMemDir).catch(() => {});
  }, []);

  return (
    <div className="p-6 h-full flex flex-col">
      <header className="mb-4 flex-shrink-0 flex items-start justify-between border-b border-dashed border-[rgba(30,28,23,0.1)] pb-4">
        <div>
          <h2 className="text-[11px] font-bold text-[#1e1c17] tracking-widest uppercase">// memory</h2>
          <p className="text-[10px] text-[rgba(30,28,23,0.3)] mt-0.5 font-mono">{memDir}</p>
        </div>
        <SyncButton />
      </header>

      {/* Tabs */}
      <div className="flex gap-0 mb-4 flex-shrink-0 border-b border-dashed border-[rgba(30,28,23,0.1)]">
        {(["files", "search", "prompts"] as const).map((t) => (
          <button key={t} onClick={() => setTab(t)}
            className={`px-4 py-1.5 text-[10px] font-mono uppercase tracking-widest transition-colors ${
              tab === t
                ? "text-[#1e1c17] border-b border-[#1e1c17]"
                : "text-[rgba(30,28,23,0.4)] hover:text-[rgba(30,28,23,0.7)]"
            }`}>
            // {t}
          </button>
        ))}
      </div>

      <div className="flex-1 min-h-0 overflow-hidden">
        {tab === "files" && <FilesTab />}
        {tab === "search" && <SearchTab />}
        {tab === "prompts" && <PromptsTab />}
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
      className="text-[10px] px-2.5 py-1 border border-dashed border-[rgba(30,28,23,0.2)] text-[rgba(30,28,23,0.45)] hover:text-[#1e1c17] hover:border-[rgba(30,28,23,0.4)] disabled:opacity-40 transition-colors font-mono uppercase tracking-wider">
      {syncing ? "[syncing]" : ok ? "[ok]" : "[sync]"}
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
    <div className="flex gap-3 h-full">
      {/* File list */}
      <div className="w-48 flex-shrink-0 flex flex-col gap-px overflow-y-auto border-r border-dashed border-[rgba(30,28,23,0.1)] pr-3">
        <button onClick={() => setCreating(true)}
          className="w-full text-[10px] px-2 py-1.5 border border-dashed border-[rgba(30,28,23,0.25)] text-[rgba(30,28,23,0.55)] hover:text-[#1e1c17] hover:border-[rgba(30,28,23,0.4)] transition-colors font-mono uppercase tracking-wider mb-1.5 flex-shrink-0">
          [+ new file]
        </button>

        {creating && (
          <div className="flex gap-1 mb-1.5 flex-shrink-0">
            <input autoFocus value={newName} onChange={(e) => setNewName(e.target.value)}
              onKeyDown={(e) => { if (e.key === "Enter") handleCreate(); if (e.key === "Escape") { setCreating(false); setNewName(""); } }}
              placeholder="filename.md"
              className="flex-1 min-w-0 px-2 py-1 text-[11px] font-mono bg-[rgba(30,28,23,0.05)] border border-dashed border-[rgba(30,28,23,0.25)] text-[#1e1c17] placeholder-[rgba(30,28,23,0.25)] focus:outline-none" />
            <button onClick={handleCreate} className="px-2 py-1 text-[10px] font-mono border border-dashed border-[rgba(30,28,23,0.3)] text-[rgba(30,28,23,0.6)] hover:text-[#1e1c17] uppercase">[ok]</button>
          </div>
        )}

        {loading ? (
          <p className="text-[10px] text-[rgba(30,28,23,0.3)] px-1 font-mono">// loading...</p>
        ) : files.length === 0 ? (
          <p className="text-[10px] text-[rgba(30,28,23,0.3)] px-1 font-mono">// no files</p>
        ) : (
          files.map((f) => (
            <button key={f.path} onClick={() => openFile(f)}
              className={`w-full text-left px-2 py-1.5 transition-colors group flex items-center justify-between gap-1 border-l-2 ${
                selected?.path === f.path
                  ? "bg-[rgba(30,28,23,0.07)] border-[#1e1c17]"
                  : "border-transparent hover:bg-[rgba(30,28,23,0.04)] hover:border-[rgba(30,28,23,0.2)]"
              }`}>
              <div className="min-w-0">
                <p className={`text-[11px] font-mono truncate ${selected?.path === f.path ? "text-[#1e1c17]" : "text-[rgba(30,28,23,0.65)]"}`}>{f.name}</p>
                <p className="text-[9px] text-[rgba(30,28,23,0.25)] mt-0.5 font-mono">{f.chunkCount} chunks</p>
              </div>
              <button onClick={(e) => { e.stopPropagation(); handleDelete(f); }}
                className="opacity-0 group-hover:opacity-100 text-[10px] text-[rgba(200,50,30,0.6)] hover:text-red-400 transition-all px-1 flex-shrink-0 font-mono">
                [x]
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
              <p className="text-[10px] text-[rgba(30,28,23,0.35)] font-mono truncate">{selected.path}</p>
              <div className="flex items-center gap-2 flex-shrink-0">
                {dirty && <span className="text-[10px] text-[rgba(30,28,23,0.5)] font-mono">// unsaved</span>}
                <button onClick={handleSave} disabled={saving || !dirty}
                  className="text-[10px] px-2.5 py-1 border border-dashed border-[rgba(30,28,23,0.25)] text-[rgba(30,28,23,0.55)] hover:text-[#1e1c17] hover:border-[rgba(30,28,23,0.4)] disabled:opacity-40 transition-colors font-mono uppercase tracking-wider">
                  {saving ? "[saving]" : "[save]"}
                </button>
              </div>
            </div>
            <textarea
              ref={editorRef}
              value={content}
              onChange={(e) => { setContent(e.target.value); setDirty(true); }}
              spellCheck={false}
              className="flex-1 min-h-0 w-full px-3 py-2.5 text-[12px] font-mono bg-[rgba(30,28,23,0.03)] border border-dashed border-[rgba(30,28,23,0.12)] focus:outline-none focus:border-[rgba(30,28,23,0.3)] resize-none leading-relaxed text-[rgba(30,28,23,0.82)] placeholder-[rgba(30,28,23,0.2)]"
              placeholder="// write markdown here..."
            />
            {error && <p className="text-[10px] text-red-400 mt-2 flex-shrink-0 font-mono">{error}</p>}
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center text-[rgba(30,28,23,0.25)]">
            <p className="text-[10px] font-mono tracking-widest uppercase">// select a file</p>
          </div>
        )}
      </div>
    </div>
  );
}

// ─── Prompts tab ─────────────────────────────────────────────────────────────

function PromptsTab() {
  const [systemPrompt, setSystemPrompt] = useState("");
  const [soulPrompt, setSoulPrompt] = useState("");
  const [systemDirty, setSystemDirty] = useState(false);
  const [soulDirty, setSoulDirty] = useState(false);
  const [saving, setSaving] = useState<"system" | "soul" | null>(null);
  const [error, setError] = useState("");
  const [agentConfig, setAgentConfig] = useState<any>(null);

  useEffect(() => {
    GetConfig().then((cfg: any) => {
      if (cfg?.agents?.length > 0) {
        const ac = cfg.agents[0];
        setAgentConfig(ac);
        setSystemPrompt(ac.system_prompt ?? "");
      }
    }).catch(() => {});
    GetSoulPrompt().then((s) => setSoulPrompt(s ?? "")).catch(() => {});
  }, []);

  async function saveSystem() {
    if (!agentConfig) return;
    setSaving("system"); setError("");
    try {
      await SaveChannelConfig({ ...agentConfig, system_prompt: systemPrompt });
      setAgentConfig({ ...agentConfig, system_prompt: systemPrompt });
      setSystemDirty(false);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    } finally { setSaving(null); }
  }

  async function saveSoul() {
    setSaving("soul"); setError("");
    try {
      await SaveSoulPrompt(soulPrompt);
      setSoulDirty(false);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    } finally { setSaving(null); }
  }

  return (
    <div className="flex flex-col gap-5 h-full overflow-y-auto">
      {error && <p className="text-[10px] text-red-400 font-mono flex-shrink-0">{error}</p>}

      {/* System Prompt */}
      <div className="flex flex-col gap-2 flex-1 min-h-0">
        <div className="flex items-center justify-between flex-shrink-0">
          <div>
            <h3 className="text-[10px] font-bold text-[#1e1c17] tracking-widest uppercase font-mono">// system prompt</h3>
            <p className="text-[9px] text-[rgba(30,28,23,0.3)] mt-0.5 font-mono">config.yml → agents[0].system_prompt</p>
          </div>
          <div className="flex items-center gap-2">
            {systemDirty && <span className="text-[10px] text-[rgba(30,28,23,0.5)] font-mono">// unsaved</span>}
            <button onClick={saveSystem} disabled={saving !== null || !systemDirty}
              className="text-[10px] px-2.5 py-1 border border-dashed border-[rgba(30,28,23,0.25)] text-[rgba(30,28,23,0.55)] hover:text-[#1e1c17] hover:border-[rgba(30,28,23,0.4)] disabled:opacity-40 transition-colors font-mono uppercase tracking-wider">
              {saving === "system" ? "[saving]" : "[save]"}
            </button>
          </div>
        </div>
        <textarea
          value={systemPrompt}
          onChange={(e) => { setSystemPrompt(e.target.value); setSystemDirty(true); }}
          spellCheck={false}
          placeholder="// 在此输入系统提示词，会追加到 agent 的系统提示末尾…"
          className="flex-1 min-h-[120px] w-full px-3 py-2.5 text-[12px] font-mono bg-[rgba(30,28,23,0.03)] border border-dashed border-[rgba(30,28,23,0.12)] focus:outline-none focus:border-[rgba(30,28,23,0.3)] resize-none leading-relaxed text-[rgba(30,28,23,0.82)] placeholder-[rgba(30,28,23,0.2)]"
        />
      </div>

      {/* Soul Prompt */}
      <div className="flex flex-col gap-2 flex-1 min-h-0">
        <div className="flex items-center justify-between flex-shrink-0">
          <div>
            <h3 className="text-[10px] font-bold text-[#1e1c17] tracking-widest uppercase font-mono">// soul prompt (SOUL.md)</h3>
            <p className="text-[9px] text-[rgba(30,28,23,0.3)] mt-0.5 font-mono">定义 agent 的人格和角色，会注入系统提示词</p>
          </div>
          <div className="flex items-center gap-2">
            {soulDirty && <span className="text-[10px] text-[rgba(30,28,23,0.5)] font-mono">// unsaved</span>}
            <button onClick={saveSoul} disabled={saving !== null || !soulDirty}
              className="text-[10px] px-2.5 py-1 border border-dashed border-[rgba(30,28,23,0.25)] text-[rgba(30,28,23,0.55)] hover:text-[#1e1c17] hover:border-[rgba(30,28,23,0.4)] disabled:opacity-40 transition-colors font-mono uppercase tracking-wider">
              {saving === "soul" ? "[saving]" : "[save]"}
            </button>
          </div>
        </div>
        <textarea
          value={soulPrompt}
          onChange={(e) => { setSoulPrompt(e.target.value); setSoulDirty(true); }}
          spellCheck={false}
          placeholder="// 在此定义 agent 的人格、语气、角色设定…"
          className="flex-1 min-h-[120px] w-full px-3 py-2.5 text-[12px] font-mono bg-[rgba(30,28,23,0.03)] border border-dashed border-[rgba(30,28,23,0.12)] focus:outline-none focus:border-[rgba(30,28,23,0.3)] resize-none leading-relaxed text-[rgba(30,28,23,0.82)] placeholder-[rgba(30,28,23,0.2)]"
        />
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
        <div className="flex-1 flex items-center border border-dashed border-[rgba(30,28,23,0.25)] bg-[rgba(30,28,23,0.03)]">
          <span className="pl-2 text-[11px] font-mono text-[rgba(30,28,23,0.3)]">&gt;</span>
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="// query..."
            className="flex-1 px-2 py-1.5 text-[11px] font-mono bg-transparent focus:outline-none text-[#1e1c17] placeholder-[rgba(30,28,23,0.25)]"
          />
        </div>
        <button type="submit" disabled={searching || !query.trim()}
          className="px-3 py-1.5 text-[10px] font-mono border border-dashed border-[rgba(30,28,23,0.3)] text-[rgba(30,28,23,0.6)] hover:text-[#1e1c17] hover:border-[rgba(30,28,23,0.5)] disabled:opacity-40 transition-colors uppercase tracking-wider">
          {searching ? "[...]" : "[search]"}
        </button>
      </form>

      {error && <p className="text-[10px] text-red-400 mb-3 flex-shrink-0 font-mono">{error}</p>}

      <div className="flex-1 min-h-0 overflow-y-auto space-y-2">
        {searched && results.length === 0 && (
          <div className="py-12 text-center">
            <pre className="text-[10px] font-mono text-[rgba(30,28,23,0.2)] leading-tight">{`// NO RESULTS FOUND\n// TRY DIFFERENT QUERY`}</pre>
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
    <div className="border border-dashed border-[rgba(30,28,23,0.15)] bg-[rgba(30,28,23,0.02)] overflow-hidden">
      <button className="w-full text-left px-3 py-2 hover:bg-[rgba(30,28,23,0.04)] transition-colors"
        onClick={() => setExpanded(!expanded)}>
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0 flex-1">
            <p className="text-[10px] font-mono text-[rgba(30,28,23,0.4)] truncate">
              // {result.filePath.split("/").pop()} : {result.startLine}–{result.endLine}
            </p>
            <p className="text-[11px] font-mono text-[rgba(30,28,23,0.75)] mt-0.5 line-clamp-2 leading-snug">
              {result.content.slice(0, 160)}{result.content.length > 160 ? "…" : ""}
            </p>
          </div>
          <div className="flex-shrink-0 flex items-center gap-2 mt-0.5">
            <ScoreBadge pct={pct} />
            <span className="text-[rgba(30,28,23,0.3)] text-[10px] font-mono">{expanded ? "[^]" : "[v]"}</span>
          </div>
        </div>
      </button>
      {expanded && (
        <div className="border-t border-dashed border-[rgba(30,28,23,0.1)] px-3 py-2">
          <p className="text-[10px] text-[rgba(30,28,23,0.3)] font-mono mb-2 truncate">// {result.filePath}</p>
          <pre className="text-[11px] text-[rgba(30,28,23,0.65)] whitespace-pre-wrap break-words font-mono leading-relaxed max-h-48 overflow-y-auto">
            {result.content}
          </pre>
        </div>
      )}
    </div>
  );
}

function ScoreBadge({ pct }: { pct: number }) {
  const cls =
    pct >= 70 ? "border-[rgba(30,150,60,0.4)] text-[rgba(30,150,60,0.8)]" :
    pct >= 40 ? "border-[rgba(180,130,20,0.4)] text-[rgba(180,130,20,0.8)]" :
               "border-[rgba(30,28,23,0.2)] text-[rgba(30,28,23,0.35)]";
  return <span className={`text-[10px] font-mono px-1 py-0.5 border border-dashed ${cls}`}>{pct}%</span>;
}
