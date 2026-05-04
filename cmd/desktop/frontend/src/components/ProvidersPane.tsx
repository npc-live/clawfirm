import { useEffect, useState } from "react";
import { GetConfig, SaveConfig, TestProviderConnection } from "../lib/wails-shim";
import type { Config } from "../lib/wails-shim";

const PROVIDER_TYPES = [
  "anthropic", "minimax", "openai", "gemini", "ollama",
  "deepseek", "moonshot", "volcengine", "modelstudio", "glm", "zai",
  "groq", "openrouter", "together", "mistral", "xai", "nvidia",
  "xiaomi", "venice", "huggingface", "perplexity", "zenmux",
  "litellm", "sglang", "vllm",
];

const DEFAULT_BASE_URLS: Record<string, string> = {
  anthropic: "https://api.anthropic.com",
  minimax: "https://api.minimax.io/anthropic",
  openai: "https://api.openai.com/v1",
  deepseek: "https://api.deepseek.com/v1",
  moonshot: "https://api.moonshot.cn/v1",
  volcengine: "https://ark.cn-beijing.volces.com/api/v3",
  modelstudio: "https://dashscope.aliyuncs.com/compatible-mode/v1",
  glm: "https://open.bigmodel.cn/api/paas/v4",
  zai: "https://open.bigmodel.cn/api/paas/v4",
  groq: "https://api.groq.com/openai/v1",
  openrouter: "https://openrouter.ai/api/v1",
  together: "https://api.together.xyz/v1",
  mistral: "https://api.mistral.ai/v1",
  xai: "https://api.x.ai/v1",
  nvidia: "https://integrate.api.nvidia.com/v1",
  xiaomi: "https://api.xiaomimimo.com/v1",
  venice: "https://api.venice.ai/api/v1",
  huggingface: "https://router.huggingface.co/v1",
  perplexity: "https://api.perplexity.ai",
  litellm: "http://localhost:4000/v1",
  sglang: "http://127.0.0.1:30000/v1",
  vllm: "http://127.0.0.1:8000/v1",
  ollama: "http://localhost:11434",
};

interface ProviderEntry {
  id: string;
  type: string;
  api_key: string;
  base_url: string;
}

const inputCls =
  "w-full px-2 py-1.5 text-[11px] font-mono border border-dashed border-[rgba(30,28,23,0.25)] " +
  "focus:outline-none focus:border-[rgba(30,28,23,0.5)] " +
  "bg-[rgba(30,28,23,0.03)] text-[#1e1c17] placeholder-[rgba(30,28,23,0.25)]";

const selectCls =
  "w-full px-2 py-1.5 text-[11px] font-mono border border-dashed border-[rgba(30,28,23,0.25)] " +
  "focus:outline-none focus:border-[rgba(30,28,23,0.5)] " +
  "bg-[#f0ece3] text-[#1e1c17]";

export function ProvidersPane() {
  const [cfg, setCfg] = useState<Config | null>(null);
  const [providers, setProviders] = useState<ProviderEntry[]>([]);
  const [editing, setEditing] = useState<string | null>(null); // id or "new"
  const [form, setForm] = useState<ProviderEntry>({ id: "", type: "anthropic", api_key: "", base_url: "" });
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState<string | null>(null);
  const [testResult, setTestResult] = useState<Record<string, boolean | null>>({});
  const [error, setError] = useState("");
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

  useEffect(() => { load(); }, []);

  async function load() {
    try {
      const c = await GetConfig();
      setCfg(c);
      const list: ProviderEntry[] = Object.entries(c.providers ?? {}).map(([id, p]) => ({
        id,
        type: p.type,
        api_key: p.api_key,
        base_url: p.base_url,
      }));
      setProviders(list);
    } catch (e) {
      setError(String(e));
    }
  }

  function startAdd() {
    setForm({ id: "", type: "anthropic", api_key: "", base_url: "" });
    setEditing("new");
    setError("");
  }

  function startEdit(p: ProviderEntry) {
    setForm({ ...p });
    setEditing(p.id);
    setError("");
  }

  function cancelEdit() {
    setEditing(null);
    setError("");
  }

  function handleTypeChange(t: string) {
    setForm(f => ({
      ...f,
      type: t,
      base_url: f.base_url === "" || Object.values(DEFAULT_BASE_URLS).includes(f.base_url)
        ? (DEFAULT_BASE_URLS[t] ?? "")
        : f.base_url,
    }));
  }

  async function handleSave() {
    if (!form.id.trim()) { setError("Provider ID is required"); return; }
    if (editing === "new" && providers.find(p => p.id === form.id.trim())) {
      setError(`Provider "${form.id}" already exists`);
      return;
    }
    setSaving(true);
    setError("");
    try {
      const updated = editing === "new"
        ? [...providers, { ...form, id: form.id.trim() }]
        : providers.map(p => p.id === editing ? { ...form } : p);

      await saveProviders(updated);
      setProviders(updated);
      setEditing(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: string) {
    try {
      const updated = providers.filter(p => p.id !== id);
      await saveProviders(updated);
      setProviders(updated);
      setDeleteConfirm(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function saveProviders(list: ProviderEntry[]) {
    if (!cfg) return;
    const newProviders: Config["providers"] = {};
    for (const p of list) {
      newProviders[p.id] = { type: p.type, api_key: p.api_key, base_url: p.base_url };
    }
    const newCfg: Config = { ...cfg, providers: newProviders };
    await SaveConfig(newCfg);
    setCfg(newCfg);
  }

  async function handleTest(id: string) {
    setTesting(id);
    setTestResult(r => ({ ...r, [id]: null }));
    try {
      const ok = await TestProviderConnection(id);
      setTestResult(r => ({ ...r, [id]: ok }));
    } catch {
      setTestResult(r => ({ ...r, [id]: false }));
    } finally {
      setTesting(null);
    }
  }

  return (
    <div className="max-w-2xl">
      <header className="mb-4 flex items-start justify-between border-b border-dashed border-[rgba(30,28,23,0.3)] pb-4">
        <div>
          <h2 className="text-[11px] font-bold text-[#1e1c17] tracking-widest uppercase">// providers</h2>
          <p className="text-[10px] text-[rgba(30,28,23,0.3)] mt-0.5 font-mono">// llm api keys and endpoints</p>
        </div>
        {editing === null && (
          <button onClick={startAdd}
            className="text-[10px] px-2.5 py-1 border border-dashed border-[rgba(30,28,23,0.25)] text-[rgba(30,28,23,0.55)] hover:text-[#1e1c17] hover:border-[rgba(30,28,23,0.4)] transition-colors font-mono uppercase tracking-wider">
            [+ add]
          </button>
        )}
      </header>

      {error && (
        <div className="mb-4 px-3 py-2 border border-dashed border-[rgba(200,50,30,0.4)] text-[10px] font-mono text-red-400 flex-shrink-0">
          // err: {error}
        </div>
      )}

      {/* Add / Edit form */}
      {editing !== null && (
        <div className="mb-4 p-4 bg-[rgba(30,28,23,0.03)] border border-dashed border-[rgba(30,28,23,0.3)] flex-shrink-0">
          <p className="text-[10px] font-mono text-[rgba(30,28,23,0.45)] uppercase tracking-wider mb-3">
            // {editing === "new" ? "new provider" : `edit: ${editing}`}
          </p>
          <div className="space-y-2.5">
            {editing === "new" && (
              <div>
                <label className="block text-[10px] font-mono text-[rgba(30,28,23,0.4)] uppercase tracking-wider mb-1">// id</label>
                <input
                  className={inputCls}
                  placeholder="e.g. my-openai"
                  value={form.id}
                  onChange={e => setForm(f => ({ ...f, id: e.target.value }))}
                />
              </div>
            )}
            <div>
              <label className="block text-[10px] font-mono text-[rgba(30,28,23,0.4)] uppercase tracking-wider mb-1">// type</label>
              <select className={selectCls} value={form.type} onChange={e => handleTypeChange(e.target.value)}>
                {PROVIDER_TYPES.map(t => (
                  <option key={t} value={t}>{t}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-[10px] font-mono text-[rgba(30,28,23,0.4)] uppercase tracking-wider mb-1">// api key</label>
              <input
                className={inputCls}
                type="password"
                placeholder="sk-..."
                value={form.api_key}
                onChange={e => setForm(f => ({ ...f, api_key: e.target.value }))}
              />
            </div>
            <div>
              <label className="block text-[10px] font-mono text-[rgba(30,28,23,0.4)] uppercase tracking-wider mb-1">
                // base url <span className="normal-case">(optional)</span>
              </label>
              <input
                className={inputCls}
                placeholder={DEFAULT_BASE_URLS[form.type] ?? "https://..."}
                value={form.base_url}
                onChange={e => setForm(f => ({ ...f, base_url: e.target.value }))}
              />
            </div>
          </div>
          <div className="flex gap-2 mt-3">
            <button onClick={handleSave} disabled={saving}
              className="text-[10px] px-2.5 py-1 border border-dashed border-[rgba(30,28,23,0.35)] text-[rgba(30,28,23,0.7)] hover:text-[#1e1c17] hover:border-[rgba(30,28,23,0.5)] disabled:opacity-40 transition-colors font-mono uppercase tracking-wider">
              {saving ? "[saving]" : "[save]"}
            </button>
            <button onClick={cancelEdit}
              className="text-[10px] px-2.5 py-1 border border-dashed border-[rgba(30,28,23,0.3)] text-[rgba(30,28,23,0.4)] hover:text-[rgba(30,28,23,0.6)] transition-colors font-mono uppercase tracking-wider">
              [cancel]
            </button>
          </div>
        </div>
      )}

      {/* Provider list */}
      <div className="space-y-2">
        {providers.length === 0 && editing === null && (
          <div className="py-12 text-center">
            <pre className="text-[10px] font-mono text-[rgba(30,28,23,0.2)] leading-tight">{`// NO PROVIDERS CONFIGURED\n// CLICK [+ ADD] TO GET STARTED`}</pre>
          </div>
        )}
        {providers.map(p => (
          <div key={p.id}
            className="px-3 py-2.5 bg-[rgba(30,28,23,0.02)] border border-dashed border-[rgba(30,28,23,0.25)] hover:border-[rgba(30,28,23,0.4)] transition-colors">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="text-[12px] font-mono font-bold text-[#1e1c17]">{p.id}</span>
                  <span className="text-[10px] font-mono border border-dashed border-[rgba(30,28,23,0.25)] text-[rgba(30,28,23,0.5)] px-1.5 py-0.5 uppercase tracking-wider">
                    {p.type}
                  </span>
                  {testResult[p.id] === true && (
                    <span className="text-[10px] font-mono text-[rgba(30,150,60,0.8)]">[ok]</span>
                  )}
                  {testResult[p.id] === false && (
                    <span className="text-[10px] font-mono text-red-400">[fail]</span>
                  )}
                </div>
                <div className="mt-1 space-y-0.5">
                  <p className="text-[10px] text-[rgba(30,28,23,0.35)] font-mono truncate">
                    key: {p.api_key ? "••••••••" + p.api_key.slice(-4) : <span className="text-[rgba(200,50,30,0.6)]">not set</span>}
                  </p>
                  {p.base_url && (
                    <p className="text-[10px] text-[rgba(30,28,23,0.2)] font-mono truncate">
                      {p.base_url}
                    </p>
                  )}
                </div>
              </div>
              <div className="flex items-center gap-1 flex-shrink-0">
                <button
                  onClick={() => handleTest(p.id)}
                  disabled={testing === p.id}
                  className="text-[10px] px-2 py-1 border border-dashed border-[rgba(30,28,23,0.3)] text-[rgba(30,28,23,0.45)] hover:text-[#1e1c17] hover:border-[rgba(30,28,23,0.4)] disabled:opacity-40 transition-colors font-mono uppercase tracking-wider">
                  {testing === p.id ? "[...]" : "[test]"}
                </button>
                <button
                  onClick={() => startEdit(p)}
                  className="text-[10px] px-2 py-1 border border-dashed border-[rgba(30,28,23,0.3)] text-[rgba(30,28,23,0.45)] hover:text-[#1e1c17] hover:border-[rgba(30,28,23,0.4)] transition-colors font-mono uppercase tracking-wider">
                  [edit]
                </button>
                {deleteConfirm === p.id ? (
                  <>
                    <button onClick={() => handleDelete(p.id)}
                      className="text-[10px] px-2 py-1 border border-dashed border-[rgba(200,50,30,0.4)] text-red-400 hover:border-red-400 transition-colors font-mono uppercase tracking-wider">
                      [confirm]
                    </button>
                    <button onClick={() => setDeleteConfirm(null)}
                      className="text-[10px] px-2 py-1 border border-dashed border-[rgba(30,28,23,0.3)] text-[rgba(30,28,23,0.4)] hover:text-[rgba(30,28,23,0.6)] transition-colors font-mono uppercase tracking-wider">
                      [no]
                    </button>
                  </>
                ) : (
                  <button onClick={() => setDeleteConfirm(p.id)}
                    className="text-[10px] px-2 py-1 border border-dashed border-[rgba(30,28,23,0.3)] text-[rgba(30,28,23,0.45)] hover:border-[rgba(200,50,30,0.4)] hover:text-red-400 transition-colors font-mono uppercase tracking-wider">
                    [del]
                  </button>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
