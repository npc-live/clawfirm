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
  "w-full px-3 py-2 text-[13px] border border-[rgba(61,57,41,0.12)] rounded-xl " +
  "focus:outline-none focus:ring-2 focus:ring-[rgba(200,90,42,0.4)] " +
  "bg-[rgba(61,57,41,0.05)] text-[#3d3929] placeholder-[rgba(61,57,41,0.3)]";

const selectCls =
  "w-full px-3 py-2 text-[13px] border border-[rgba(61,57,41,0.12)] rounded-xl " +
  "focus:outline-none focus:ring-2 focus:ring-[rgba(200,90,42,0.4)] " +
  "bg-[#ece5d8] text-[#3d3929]";

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
    <div className="p-8">
      <header className="mb-6 flex items-center justify-between">
        <div>
          <h2 className="text-[22px] font-semibold text-[#3d3929] tracking-[-0.43px]">Providers</h2>
          <p className="text-[13px] text-[rgba(61,57,41,0.3)] mt-1">Manage LLM provider API keys and endpoints</p>
        </div>
        {editing === null && (
          <button onClick={startAdd}
            className="px-4 py-2 bg-[#c85a2a] text-white text-[13px] rounded-xl hover:bg-[#a84a22] transition-colors font-semibold">
            + Add Provider
          </button>
        )}
      </header>

      {error && (
        <div className="mb-4 px-4 py-3 bg-[rgba(239,68,68,0.1)] border border-[rgba(239,68,68,0.3)] rounded-xl text-[13px] text-red-400">
          {error}
        </div>
      )}

      {/* Add / Edit form */}
      {editing !== null && (
        <div className="mb-6 p-5 bg-[rgba(61,57,41,0.04)] border border-[rgba(61,57,41,0.12)] rounded-2xl max-w-xl">
          <h3 className="text-[14px] font-semibold text-[#3d3929] mb-4">
            {editing === "new" ? "New Provider" : `Edit · ${editing}`}
          </h3>
          <div className="space-y-3">
            {editing === "new" && (
              <div>
                <label className="block text-[12px] text-[rgba(61,57,41,0.4)] mb-1">Provider ID</label>
                <input
                  className={inputCls}
                  placeholder="e.g. my-openai"
                  value={form.id}
                  onChange={e => setForm(f => ({ ...f, id: e.target.value }))}
                />
              </div>
            )}
            <div>
              <label className="block text-[12px] text-[rgba(61,57,41,0.4)] mb-1">Type</label>
              <select className={selectCls} value={form.type} onChange={e => handleTypeChange(e.target.value)}>
                {PROVIDER_TYPES.map(t => (
                  <option key={t} value={t}>{t}</option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-[12px] text-[rgba(61,57,41,0.4)] mb-1">API Key</label>
              <input
                className={inputCls}
                type="password"
                placeholder="sk-..."
                value={form.api_key}
                onChange={e => setForm(f => ({ ...f, api_key: e.target.value }))}
              />
            </div>
            <div>
              <label className="block text-[12px] text-[rgba(61,57,41,0.4)] mb-1">
                Base URL <span className="text-[rgba(61,57,41,0.3)]">(optional, uses default if empty)</span>
              </label>
              <input
                className={inputCls}
                placeholder={DEFAULT_BASE_URLS[form.type] ?? "https://..."}
                value={form.base_url}
                onChange={e => setForm(f => ({ ...f, base_url: e.target.value }))}
              />
            </div>
          </div>
          <div className="flex gap-2 mt-4">
            <button onClick={handleSave} disabled={saving}
              className="px-4 py-2 bg-[#c85a2a] text-white text-[13px] rounded-xl hover:bg-[#a84a22] disabled:opacity-50 transition-colors font-semibold">
              {saving ? "Saving…" : "Save"}
            </button>
            <button onClick={cancelEdit}
              className="px-4 py-2 bg-[rgba(61,57,41,0.08)] text-[rgba(61,57,41,0.55)] text-[13px] rounded-xl hover:bg-[rgba(61,57,41,0.12)] transition-colors">
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Provider list */}
      <div className="space-y-3 max-w-2xl">
        {providers.length === 0 && editing === null && (
          <p className="text-[13px] text-[rgba(61,57,41,0.3)] py-8 text-center">
            No providers configured. Click "Add Provider" to get started.
          </p>
        )}
        {providers.map(p => (
          <div key={p.id}
            className="p-4 bg-[rgba(61,57,41,0.04)] border border-[rgba(61,57,41,0.08)] rounded-2xl hover:border-[rgba(61,57,41,0.15)] transition-colors">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2 flex-wrap">
                  <span className="text-[14px] font-semibold text-[#3d3929]">{p.id}</span>
                  <span className="px-2 py-0.5 text-[11px] bg-[rgba(200,90,42,0.15)] text-[#c85a2a] rounded-full font-medium">
                    {p.type}
                  </span>
                  {testResult[p.id] === true && (
                    <span className="text-[11px] text-emerald-400">✓ connected</span>
                  )}
                  {testResult[p.id] === false && (
                    <span className="text-[11px] text-red-400">✗ failed</span>
                  )}
                </div>
                <div className="mt-1.5 space-y-0.5">
                  <p className="text-[12px] text-[rgba(61,57,41,0.35)] font-mono truncate">
                    key: {p.api_key ? "••••••••" + p.api_key.slice(-4) : <span className="text-[rgba(255,100,100,0.6)]">not set</span>}
                  </p>
                  {p.base_url && (
                    <p className="text-[12px] text-[rgba(61,57,41,0.2)] font-mono truncate">
                      {p.base_url}
                    </p>
                  )}
                </div>
              </div>
              <div className="flex items-center gap-1.5 flex-shrink-0">
                <button
                  onClick={() => handleTest(p.id)}
                  disabled={testing === p.id}
                  className="px-3 py-1.5 text-[12px] bg-[rgba(61,57,41,0.08)] text-[rgba(61,57,41,0.5)] rounded-lg hover:bg-[rgba(61,57,41,0.12)] hover:text-[rgba(61,57,41,0.8)] disabled:opacity-50 transition-colors">
                  {testing === p.id ? "…" : "Test"}
                </button>
                <button
                  onClick={() => { startEdit(p); }}
                  className="px-3 py-1.5 text-[12px] bg-[rgba(61,57,41,0.08)] text-[rgba(61,57,41,0.5)] rounded-lg hover:bg-[rgba(61,57,41,0.12)] hover:text-[rgba(61,57,41,0.8)] transition-colors">
                  Edit
                </button>
                {deleteConfirm === p.id ? (
                  <>
                    <button onClick={() => handleDelete(p.id)}
                      className="px-3 py-1.5 text-[12px] bg-[rgba(239,68,68,0.15)] text-red-400 rounded-lg hover:bg-[rgba(239,68,68,0.25)] transition-colors">
                      Confirm
                    </button>
                    <button onClick={() => setDeleteConfirm(null)}
                      className="px-3 py-1.5 text-[12px] bg-[rgba(61,57,41,0.08)] text-[rgba(61,57,41,0.5)] rounded-lg hover:bg-[rgba(61,57,41,0.12)] transition-colors">
                      Cancel
                    </button>
                  </>
                ) : (
                  <button onClick={() => setDeleteConfirm(p.id)}
                    className="px-3 py-1.5 text-[12px] bg-[rgba(61,57,41,0.08)] text-[rgba(61,57,41,0.5)] rounded-lg hover:bg-[rgba(239,68,68,0.15)] hover:text-red-400 transition-colors">
                    Delete
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
