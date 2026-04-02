import React, { useEffect, useState } from "react";
import {
  GetChannels, GetChatSessions, GetHistory,
  GetConfig, SaveConfig,
  GetProviders,
  GetConfigRaw, SaveConfigRaw,
  GetAllSkills, GetSkillContent, SearchRemoteSkills, InstallRemoteSkill,
  GetFeishuConfig, SaveFeishuConfig,
  GetWhatsAppStatus, GetWhatsAppQR, LogoutWhatsApp,
  ListCronJobs, AddCronJob, UpdateCronJob, DeleteCronJob,
  ToggleCronJob, GetCronJobHistory, TriggerCronJob,
  GetVault, SetVaultEntry, DeleteVaultEntry,
  ListWhipFiles, GetWhipFileContent,
  BrowserTestCDP, BrowserLaunchChrome,
  BrowserListShortcuts, BrowserRunShortcut,
} from "../wailsjs/go/app/App";
import type { ChannelInfo, HistoryMessage, SkillInfo, RemoteSkillInfo, CronJob, CronJobHistory, Config, ProviderInfo, VaultEntry, BrowserStatus, ShortcutInfo } from "../wailsjs/go/app/App";
import { EventsOn } from "../wailsjs/runtime/runtime";
import { MemoryPane } from "./MemoryPane";
import { ProvidersPane } from "./ProvidersPane";
import { CanvasPane } from "./CanvasPane";


interface Props {
  onOpenChat: (agentName: string, sessionID: string) => void;
}
interface SessionPreview { id: string; preview: string; lastMs: number; }
type NavTab = "chats" | "canvas" | "skills" | "cron" | "memory" | "agents" | "channels" | "providers" | "whipflow" | "vault" | "browser" | "settings";

// ─────────────────────────────────────────────────────────────────────────────
// Root
// ─────────────────────────────────────────────────────────────────────────────
export function Dashboard({ onOpenChat }: Props) {
  const [nav, setNav] = useState<NavTab>("chats");
  const [skillsKey, setSkillsKey] = useState(0);
  const [channels, setChannels] = useState<ChannelInfo[]>([]);
  const [sessionMap, setSessionMap] = useState<Record<string, SessionPreview[]>>({});

  async function loadChannels() {
    try {
      const ch = await GetChannels();
      setChannels(ch ?? []);

      // First pass: populate session list immediately with placeholders so UI renders fast.
      const sm: Record<string, SessionPreview[]> = {};
      for (const c of ch ?? []) {
        const ids = (await GetChatSessions(c.name)) ?? [];
        sm[c.name] = ids.map((sid) => ({
          id: sid,
          lastMs: parseInt(sid.replace(/^s/, ""), 10) || 0,
          preview: "…",
        }));
      }
      setSessionMap({ ...sm });

      // Second pass: fill in previews from message history.
      for (const c of ch ?? []) {
        const previews = sm[c.name] ?? [];
        const filled: SessionPreview[] = await Promise.all(
          previews.map(async (p) => {
            try {
              const msgs: HistoryMessage[] = (await GetHistory("webchat/" + c.name, p.id)) ?? [];
              const last = msgs[msgs.length - 1];
              return {
                ...p,
                preview: last
                  ? (last.content.length > 60 ? last.content.slice(0, 60) + "…" : last.content)
                  : "(empty)",
              };
            } catch {
              return p;
            }
          })
        );
        sm[c.name] = filled;
        setSessionMap({ ...sm });
      }
    } catch (e) {
      console.error("loadChannels:", e);
    }
  }

  useEffect(() => {
    // Wails bindings may not be ready immediately on mount — retry once after a short delay.
    loadChannels();
    const t = setTimeout(() => loadChannels(), 800);
    const unsub = EventsOn("message:new", () => loadChannels());
    return () => { clearTimeout(t); unsub(); };
  }, []);

  return (
    <div className="flex h-full bg-[rgb(30,30,28)]">
      {/* Sidebar */}
      <aside className="w-52 flex-shrink-0 bg-[rgb(26,26,24)] border-r border-[rgba(255,255,255,0.08)] flex flex-col h-full">
        <div className="px-5 pt-6 pb-4">
          <h1 className="text-[15px] font-semibold text-[rgb(240,237,229)] tracking-tight">Pi Go</h1>
          <p className="text-[11px] text-[rgba(255,255,255,0.3)] mt-0.5">AI Gateway</p>
        </div>
        <nav className="flex-1 px-3 space-y-0.5">
          <SidebarItem icon="💬" label="Chats" active={nav === "chats"} onClick={() => setNav("chats")} />
          <SidebarItem icon="🎨" label="Canvas" active={nav === "canvas"} onClick={() => setNav("canvas")} />
          <SidebarItem icon="🧩" label="Skills" active={nav === "skills"} onClick={() => { setNav("skills"); setSkillsKey(k => k + 1); }} />
          <SidebarItem icon="🕐" label="Cron" active={nav === "cron"} onClick={() => setNav("cron")} />
          <SidebarItem icon="🧠" label="Memory" active={nav === "memory"} onClick={() => setNav("memory")} />
          <SidebarItem icon="🤖" label="Agents" active={nav === "agents"} onClick={() => setNav("agents")} />
          <SidebarItem icon="📡" label="Channels" active={nav === "channels"} onClick={() => setNav("channels")} />
          <SidebarItem icon="🔑" label="Providers" active={nav === "providers"} onClick={() => setNav("providers")} />
          <SidebarItem icon="🔐" label="Vault" active={nav === "vault"} onClick={() => setNav("vault")} />
          <SidebarItem icon="⚡" label="WhipFlow" active={nav === "whipflow"} onClick={() => setNav("whipflow")} />
          <SidebarItem icon="🌐" label="Browser" active={nav === "browser"} onClick={() => setNav("browser")} />
          <SidebarItem icon="⚙️" label="Settings" active={nav === "settings"} onClick={() => setNav("settings")} />
        </nav>
        <div className="px-4 py-4 border-t border-[rgba(255,255,255,0.06)]">
          <p className="text-[11px] text-[rgba(255,255,255,0.3)]">{channels.length} agent{channels.length !== 1 ? "s" : ""}</p>
        </div>
      </aside>

      {/* Content */}
      <main className={`flex-1 min-h-0 ${nav === "canvas" || nav === "whipflow" ? "overflow-hidden" : "overflow-y-auto"}`}>
        {nav === "chats" && <ChatsPane sessionMap={sessionMap} channels={channels} onOpenChat={onOpenChat} />}
        {nav === "canvas" && <div className="w-full h-full"><CanvasPane /></div>}
        {nav === "skills" && <SkillsPane key={skillsKey} />}
        {nav === "cron" && <CronJobsPane />}
        {nav === "memory" && <MemoryPane />}
        {nav === "agents" && <AgentsPane />}
        {nav === "channels" && <ChannelsPane />}
        {nav === "providers" && <ProvidersPane />}
        {nav === "vault" && <VaultPane />}
        {nav === "whipflow" && <WhipflowPane />}
        {nav === "browser" && <BrowserPane />}
        {nav === "settings" && <SettingsPane key={Date.now()} />}
      </main>
    </div>
  );
}

function SidebarItem({ icon, label, active, onClick }: { icon: string; label: string; active: boolean; onClick: () => void }) {
  return (
    <button onClick={onClick}
      className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-md text-[13px] transition-colors text-left ${
        active
          ? "bg-[rgba(38,136,249,0.15)] text-[#2688f9] font-medium"
          : "text-[rgba(240,237,229,0.65)] hover:bg-[rgba(255,255,255,0.05)] hover:text-[rgb(240,237,229)]"
      }`}>
      <span className="text-base leading-none">{icon}</span>{label}
    </button>
  );
}


// ─────────────────────────────────────────────────────────────────────────────
// Skills pane
function extractUrls(content: string): string[] {
  const found = new Set<string>();
  const re = /https?:\/\/[^\s"')<>\]]+/g;
  for (const m of content.matchAll(re)) {
    found.add(m[0].replace(/[.,;:]+$/, ""));
  }
  return [...found];
}

// ─────────────────────────────────────────────────────────────────────────────
function SkillsPane() {
  const [skills, setSkills] = useState<SkillInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [contentMap, setContentMap] = useState<Record<string, string>>({});

  // Remote search state
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState<RemoteSkillInfo[]>([]);
  const [searching, setSearching] = useState(false);
  const [searchDone, setSearchDone] = useState(false);
  const [installing, setInstalling] = useState<string | null>(null);
  const [installMsg, setInstallMsg] = useState("");

  useEffect(() => {
    setLoading(true);
    GetAllSkills()
      .then((s) => setSkills(s ?? []))
      .catch(() => setSkills([]))
      .finally(() => setLoading(false));
  }, []);

  async function toggle(filePath: string) {
    if (expanded === filePath) { setExpanded(null); return; }
    setExpanded(filePath);
    if (!contentMap[filePath]) {
      const text = await GetSkillContent(filePath).catch(() => "");
      setContentMap((m) => ({ ...m, [filePath]: text }));
    }
  }

  async function doSearch() {
    const q = searchQuery.trim();
    if (!q) return;
    setSearching(true);
    setSearchDone(false);
    setInstallMsg("");
    try {
      const results = await SearchRemoteSkills(q);
      setSearchResults(results ?? []);
    } catch {
      setSearchResults([]);
    } finally {
      setSearching(false);
      setSearchDone(true);
    }
  }

  async function doInstall(name: string) {
    setInstalling(name);
    setInstallMsg("");
    try {
      const msg = await InstallRemoteSkill(name, true);
      setInstallMsg(msg);
      // Refresh local skills list.
      const s = await GetAllSkills().catch(() => []);
      setSkills(s ?? []);
    } catch (e: any) {
      setInstallMsg("Error: " + (e?.message ?? String(e)));
    } finally {
      setInstalling(null);
    }
  }

  return (
    <div className="p-8">
      <header className="mb-6">
        <h2 className="text-[22px] font-semibold text-[rgb(240,237,229)] tracking-[-0.43px]">
          Skills
          <span className="ml-2 text-[11px] font-normal text-[rgba(255,255,255,0.5)] bg-[rgba(255,255,255,0.08)] px-2 py-0.5 rounded-full">
            {loading ? "…" : `${skills.length} loaded`}
          </span>
        </h2>
      </header>

      {/* ── Remote skill search ── */}
      <div className="mb-6 max-w-2xl">
        <div className="flex gap-2">
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && doSearch()}
            placeholder="Search remote skills (e.g. code-review, commit)…"
            className="flex-1 px-3 py-2 rounded-lg bg-[rgba(255,255,255,0.06)] border border-[rgba(255,255,255,0.1)]
              text-[13px] text-[rgb(240,237,229)] placeholder-[rgba(255,255,255,0.25)]
              focus:outline-none focus:border-[rgba(255,255,255,0.25)] transition-colors"
          />
          <button
            onClick={doSearch}
            disabled={searching || !searchQuery.trim()}
            className="px-4 py-2 rounded-lg bg-[rgba(255,255,255,0.08)] border border-[rgba(255,255,255,0.1)]
              text-[13px] text-[rgb(240,237,229)] hover:bg-[rgba(255,255,255,0.12)] transition-colors
              disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {searching ? "Searching…" : "Search"}
          </button>
        </div>

        {installMsg && (
          <p className={`mt-2 text-[12px] ${installMsg.startsWith("Error") ? "text-red-400" : "text-green-400"}`}>{installMsg}</p>
        )}

        {searchDone && searchResults.length === 0 && (
          <p className="mt-3 text-[13px] text-[rgba(255,255,255,0.4)] italic">No remote skills found.</p>
        )}
        {searchResults.length > 0 && (
          <div className="mt-3 space-y-2">
            {searchResults.map((r) => (
              <div key={r.name} className="flex items-center gap-3 px-4 py-3 rounded-lg bg-[rgba(255,255,255,0.04)] border border-[rgba(255,255,255,0.07)]">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="text-[13px] font-medium text-[rgb(240,237,229)]">{r.name}</p>
                    <span className="text-[11px] text-[rgba(255,255,255,0.35)] font-mono">v{r.version}</span>
                    {r.author && <span className="text-[11px] text-[rgba(255,255,255,0.25)]">by {r.author}</span>}
                  </div>
                  {r.description && (
                    <p className="text-[12px] text-[rgba(255,255,255,0.4)] mt-0.5 line-clamp-2">{r.description}</p>
                  )}
                </div>
                <button
                  onClick={() => doInstall(r.name)}
                  disabled={installing !== null}
                  className="flex-shrink-0 px-3 py-1.5 rounded-md bg-[rgba(96,196,255,0.15)] border border-[rgba(96,196,255,0.25)]
                    text-[12px] text-[rgba(96,196,255,0.9)] hover:bg-[rgba(96,196,255,0.25)] transition-colors
                    disabled:opacity-40 disabled:cursor-not-allowed"
                >
                  {installing === r.name ? "Installing…" : "Install"}
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* ── Local skills ── */}
      {!loading && skills.length === 0 && !searchDone && (
        <p className="text-[13px] text-[rgba(255,255,255,0.4)] italic">No skills found.</p>
      )}
      <div className="space-y-2 max-w-2xl">
        {skills.map((s) => {
          const isOpen = expanded === s.filePath;
          const content = contentMap[s.filePath] ?? "";
          const urls = isOpen ? extractUrls(content) : [];
          return (
            <div key={s.filePath} className="rounded-lg bg-[rgba(255,255,255,0.04)] border border-[rgba(255,255,255,0.07)] overflow-hidden">
              <button className="w-full text-left px-4 py-3 hover:bg-[rgba(255,255,255,0.03)] transition-colors"
                onClick={() => toggle(s.filePath)}>
                <div className="flex items-center justify-between gap-2">
                  <p className="text-[13px] font-medium text-[rgb(240,237,229)]">{s.name}</p>
                  <span className="text-[rgba(255,255,255,0.3)] text-[11px] flex-shrink-0">{isOpen ? "▲" : "▼"}</span>
                </div>
                {s.description && (
                  <p className="text-[12px] text-[rgba(255,255,255,0.4)] mt-0.5 line-clamp-2">{s.description}</p>
                )}
                <p className="text-[11px] text-[rgba(255,255,255,0.2)] font-mono mt-1">{s.filePath}</p>
              </button>

              {isOpen && (
                <div className="border-t border-[rgba(255,255,255,0.06)] px-4 py-3 space-y-3">
                  {/* URLs */}
                  {content === "" ? (
                    <p className="text-[12px] text-[rgba(255,255,255,0.3)]">Loading…</p>
                  ) : urls.length > 0 ? (
                    <div>
                      <p className="text-[11px] font-semibold text-[rgba(255,255,255,0.4)] uppercase tracking-wider mb-2">URLs</p>
                      <div className="space-y-1">
                        {urls.map((u) => (
                          <p key={u} className="text-[12px] font-mono text-[rgba(96,196,255,0.8)] break-all">{u}</p>
                        ))}
                      </div>
                    </div>
                  ) : null}
                  {/* Raw content */}
                  <div>
                    <p className="text-[11px] font-semibold text-[rgba(255,255,255,0.4)] uppercase tracking-wider mb-2">Content</p>
                    <pre className="text-[11px] text-[rgba(255,255,255,0.5)] whitespace-pre-wrap break-words max-h-64 overflow-y-auto font-mono leading-relaxed">
                      {content}
                    </pre>
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// Cron Jobs pane
// ─────────────────────────────────────────────────────────────────────────────
type EveryUnit = "seconds" | "minutes" | "hours" | "days";

type CronFormData = {
  name: string;
  scheduleKind: string;
  at: string;        // datetime-local value: "YYYY-MM-DDTHH:mm"
  everyVal: string;  // human-friendly number
  everyUnit: EveryUnit;
  expr: string;
  tz: string;
  agentName: string;
  prompt: string;
  enabled: boolean;
};

const UNIT_MS: Record<EveryUnit, number> = {
  seconds: 1_000,
  minutes: 60_000,
  hours:   3_600_000,
  days:    86_400_000,
};

// Convert ms to best-fit human value + unit.
function msToHuman(ms: number): { val: string; unit: EveryUnit } {
  if (ms > 0 && ms % UNIT_MS.days === 0) return { val: String(ms / UNIT_MS.days), unit: "days" };
  if (ms > 0 && ms % UNIT_MS.hours === 0) return { val: String(ms / UNIT_MS.hours), unit: "hours" };
  if (ms > 0 && ms % UNIT_MS.minutes === 0) return { val: String(ms / UNIT_MS.minutes), unit: "minutes" };
  return { val: String(ms / UNIT_MS.seconds), unit: "seconds" };
}

// ISO 8601 "2024-12-31T09:00:00Z" → datetime-local "2024-12-31T09:00"
function isoToLocal(iso: string): string {
  if (!iso) return "";
  try {
    const d = new Date(iso);
    const pad = (n: number) => String(n).padStart(2, "0");
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
  } catch { return ""; }
}

// datetime-local "2024-12-31T09:00" → ISO 8601 UTC
function localToIso(local: string): string {
  if (!local) return "";
  try { return new Date(local).toISOString(); }
  catch { return ""; }
}

function defaultAt(): string {
  const d = new Date();
  d.setSeconds(0, 0);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

const emptyCronForm: CronFormData = {
  name: "", scheduleKind: "cron", at: defaultAt(), everyVal: "", everyUnit: "minutes", expr: "", tz: "", agentName: "", prompt: "", enabled: true,
};

function cronFormFromJob(j: CronJob): CronFormData {
  const h = j.schedule.everyMs ? msToHuman(j.schedule.everyMs) : { val: "", unit: "minutes" as EveryUnit };
  return {
    name: j.name,
    scheduleKind: j.scheduleKind,
    at: isoToLocal(j.schedule.at ?? ""),
    everyVal: h.val,
    everyUnit: h.unit,
    expr: j.schedule.expr ?? "",
    tz: j.schedule.tz ?? "",
    agentName: j.agentName,
    prompt: j.prompt,
    enabled: j.enabled,
  };
}

function CronJobsPane() {
  const [jobs, setJobs] = useState<CronJob[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<string | null>(null); // job ID or "new"
  const [form, setForm] = useState<CronFormData>(emptyCronForm);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [historyJobId, setHistoryJobId] = useState<string | null>(null);
  const [agentNames, setAgentNames] = useState<string[]>([]);

  useEffect(() => {
    GetConfig().then(c => setAgentNames(c?.agents?.map(a => a.name) ?? [])).catch(() => {});
  }, []);

  async function loadJobs() {
    setLoading(true);
    try {
      const list = await ListCronJobs();
      setJobs(list ?? []);
    } catch { setJobs([]); }
    finally { setLoading(false); }
  }

  useEffect(() => { loadJobs(); }, []);

  function startNew() {
    setEditing("new");
    setForm({ ...emptyCronForm, at: defaultAt() });
    setError("");
  }

  function startEdit(j: CronJob) {
    setEditing(j.id);
    setForm(cronFormFromJob(j));
    setError("");
  }

  function cancelEdit() {
    setEditing(null);
    setError("");
  }

  async function handleSave() {
    if (!form.name.trim() || !form.agentName.trim() || !form.prompt.trim()) {
      setError("Name, Agent, and Prompt are required."); return;
    }
    if (form.scheduleKind === "at" && !form.at) {
      setError("Please select a date/time for the 'at' schedule."); return;
    }
    if (form.scheduleKind === "every" && (!form.everyVal || Number(form.everyVal) <= 0)) {
      setError("Please enter a valid interval for the 'every' schedule."); return;
    }
    if (form.scheduleKind === "cron" && !form.expr.trim()) {
      setError("Please enter a cron expression."); return;
    }
    setSaving(true); setError("");
    try {
      const everyMs = form.scheduleKind === "every"
        ? (Number(form.everyVal) || 0) * UNIT_MS[form.everyUnit]
        : undefined;
      const job: CronJob = {
        id: editing === "new" ? "" : editing!,
        name: form.name.trim(),
        scheduleKind: form.scheduleKind,
        schedule: {
          at: form.scheduleKind === "at" ? localToIso(form.at) : undefined,
          everyMs,
          expr: form.scheduleKind === "cron" ? form.expr : undefined,
          tz: form.scheduleKind === "cron" ? form.tz : undefined,
        },
        agentName: form.agentName.trim(),
        prompt: form.prompt.trim(),
        enabled: form.enabled,
      };
      if (editing === "new") {
        await AddCronJob(job);
      } else {
        await UpdateCronJob(job);
      }
      setEditing(null);
      await loadJobs();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    } finally { setSaving(false); }
  }

  async function handleDelete(id: string) {
    try {
      await DeleteCronJob(id);
      if (editing === id) setEditing(null);
      if (historyJobId === id) setHistoryJobId(null);
      await loadJobs();
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function handleToggle(id: string, enabled: boolean) {
    try {
      await ToggleCronJob(id, enabled);
      await loadJobs();
    } catch {}
  }

  const [triggering, setTriggering] = useState<string | null>(null);
  async function handleTrigger(id: string) {
    setTriggering(id);
    try {
      await TriggerCronJob(id);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setTimeout(() => setTriggering(t => t === id ? null : t), 2000);
    }
  }

  function showHistory(jobId: string) {
    setHistoryJobId(id => id === jobId ? null : jobId);
  }

  const inputCls = "w-full px-3 py-2 text-[13px] border border-[rgba(255,255,255,0.1)] rounded-xl focus:outline-none focus:ring-2 focus:ring-[rgba(38,136,249,0.4)] bg-[rgba(255,255,255,0.05)] text-[rgb(240,237,229)] placeholder-[rgba(255,255,255,0.2)]";

  return (
    <div className="p-8 h-full flex flex-col">
      <header className="mb-6 flex-shrink-0 flex items-center justify-between">
        <div>
          <h2 className="text-[22px] font-semibold text-[rgb(240,237,229)] tracking-[-0.43px]">
            Cron Jobs
            <span className="ml-2 text-[11px] font-normal text-[rgba(255,255,255,0.5)] bg-[rgba(255,255,255,0.08)] px-2 py-0.5 rounded-full">
              {loading ? "…" : `${jobs.length}`}
            </span>
          </h2>
          <p className="text-[13px] text-[rgba(255,255,255,0.5)] mt-1">Schedule agents to run on a timer</p>
        </div>
        {editing === null && (
          <button onClick={startNew}
            className="text-[13px] bg-[#2688f9] text-white px-4 py-2 rounded-xl hover:bg-[#1a7ae8] transition-colors font-semibold">
            + New Job
          </button>
        )}
      </header>

      {/* Edit / Create form */}
      {editing !== null && (
        <GlassCard className="mb-5 flex-shrink-0 max-w-2xl">
          <h3 className="text-[13px] font-semibold text-[rgb(240,237,229)] mb-4">
            {editing === "new" ? "New Cron Job" : "Edit Cron Job"}
          </h3>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-[11px] text-[rgba(255,255,255,0.5)] mb-1">Name</label>
              <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="daily-report" className={inputCls} />
            </div>
            <div>
              <label className="block text-[11px] text-[rgba(255,255,255,0.5)] mb-1">Agent</label>
              <select value={form.agentName} onChange={(e) => setForm({ ...form, agentName: e.target.value })} className={selectCls}>
                <option value="">— select agent —</option>
                {agentNames.map(n => <option key={n} value={n}>{n}</option>)}
              </select>
            </div>
          </div>
          <div className="mt-3">
            <label className="block text-[11px] text-[rgba(255,255,255,0.5)] mb-1">Schedule Kind</label>
            <div className="flex gap-2">
              {(["cron", "every", "at"] as const).map((k) => (
                <button key={k} onClick={() => setForm({ ...form, scheduleKind: k })}
                  className={`px-3 py-1.5 rounded-lg text-[12px] font-medium transition-colors ${
                    form.scheduleKind === k
                      ? "bg-[rgba(38,136,249,0.2)] text-[#2688f9] border border-[rgba(38,136,249,0.3)]"
                      : "bg-[rgba(255,255,255,0.05)] text-[rgba(255,255,255,0.5)] border border-[rgba(255,255,255,0.08)] hover:bg-[rgba(255,255,255,0.08)]"
                  }`}>
                  {k}
                </button>
              ))}
            </div>
          </div>
          {form.scheduleKind === "cron" && (
            <div className="grid grid-cols-2 gap-3 mt-3">
              <div>
                <label className="block text-[11px] text-[rgba(255,255,255,0.5)] mb-1">Cron Expression</label>
                <input value={form.expr} onChange={(e) => setForm({ ...form, expr: e.target.value })} placeholder="0 9 * * MON-FRI" className={inputCls} />
              </div>
              <div>
                <label className="block text-[11px] text-[rgba(255,255,255,0.5)] mb-1">Timezone (optional)</label>
                <input value={form.tz} onChange={(e) => setForm({ ...form, tz: e.target.value })} placeholder="Asia/Shanghai" className={inputCls} />
              </div>
            </div>
          )}
          {form.scheduleKind === "every" && (
            <div className="grid grid-cols-[1fr_auto] gap-2 mt-3">
              <div>
                <label className="block text-[11px] text-[rgba(255,255,255,0.5)] mb-1">Interval</label>
                <input type="number" min="1" value={form.everyVal} onChange={(e) => setForm({ ...form, everyVal: e.target.value })} placeholder="30" className={inputCls} />
              </div>
              <div>
                <label className="block text-[11px] text-[rgba(255,255,255,0.5)] mb-1">Unit</label>
                <select value={form.everyUnit} onChange={(e) => setForm({ ...form, everyUnit: e.target.value as EveryUnit })}
                  className={`${inputCls} appearance-none cursor-pointer pr-8`}>
                  <option value="seconds">Seconds</option>
                  <option value="minutes">Minutes</option>
                  <option value="hours">Hours</option>
                  <option value="days">Days</option>
                </select>
              </div>
            </div>
          )}
          {form.scheduleKind === "at" && (
            <div className="mt-3">
              <label className="block text-[11px] text-[rgba(255,255,255,0.5)] mb-1">Run At</label>
              <input type="datetime-local" value={form.at} onChange={(e) => setForm({ ...form, at: e.target.value })}
                className={`${inputCls} [color-scheme:dark]`} />
            </div>
          )}
          <div className="mt-3">
            <label className="block text-[11px] text-[rgba(255,255,255,0.5)] mb-1">Prompt</label>
            <textarea value={form.prompt} onChange={(e) => setForm({ ...form, prompt: e.target.value })} rows={3} placeholder="Generate a daily report..."
              className={`${inputCls} resize-none leading-relaxed`} />
          </div>
          <div className="mt-3 flex items-center gap-2">
            <button onClick={() => setForm({ ...form, enabled: !form.enabled })}
              className={`w-9 h-5 rounded-full transition-colors relative ${form.enabled ? "bg-[#2688f9]" : "bg-[rgba(255,255,255,0.15)]"}`}>
              <span className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${form.enabled ? "left-[18px]" : "left-0.5"}`} />
            </button>
            <span className="text-[12px] text-[rgba(255,255,255,0.5)]">{form.enabled ? "Enabled" : "Disabled"}</span>
          </div>
          {error && <p className="text-[11px] text-red-400 mt-2">{error}</p>}
          <div className="flex gap-2 mt-4">
            <button onClick={handleSave} disabled={saving}
              className="px-5 py-2 bg-[#2688f9] text-white text-[13px] rounded-xl hover:bg-[#1a7ae8] disabled:opacity-50 transition-colors font-semibold">
              {saving ? "Saving…" : "Save"}
            </button>
            <button onClick={cancelEdit}
              className="px-4 py-2 rounded-xl border border-[rgba(255,255,255,0.1)] text-[13px] text-[rgba(255,255,255,0.5)] hover:bg-[rgba(255,255,255,0.05)]">
              Cancel
            </button>
          </div>
        </GlassCard>
      )}

      {/* Job list */}
      {!loading && jobs.length === 0 && editing === null && (
        <div className="text-center py-24 text-[rgba(255,255,255,0.3)]">
          <div className="text-4xl mb-3">🕐</div>
          <p className="text-[13px]">No cron jobs yet. Click "+ New Job" to create one.</p>
        </div>
      )}
      <div className="space-y-2 max-w-2xl flex-1 min-h-0 overflow-y-auto">
        {jobs.map((j) => (
          <div key={j.id} className="rounded-lg bg-[rgba(255,255,255,0.04)] border border-[rgba(255,255,255,0.07)] overflow-hidden">
            <div className="px-4 py-3 flex items-center justify-between gap-3">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <p className="text-[13px] font-medium text-[rgb(240,237,229)] truncate">{j.name}</p>
                  <span className={`text-[10px] font-mono px-1.5 py-0.5 rounded ${
                    j.scheduleKind === "cron" ? "bg-[rgba(38,136,249,0.15)] text-[#2688f9]"
                    : j.scheduleKind === "every" ? "bg-[rgba(52,199,89,0.15)] text-emerald-400"
                    : "bg-[rgba(255,179,64,0.15)] text-amber-400"
                  }`}>
                    {j.scheduleKind}
                  </span>
                  <span className={`text-[10px] font-medium px-1.5 py-0.5 rounded-full ${
                    j.enabled ? "bg-[rgba(52,199,89,0.15)] text-emerald-400" : "bg-[rgba(255,255,255,0.08)] text-[rgba(255,255,255,0.35)]"
                  }`}>
                    {j.enabled ? "ON" : "OFF"}
                  </span>
                </div>
                <p className="text-[11px] text-[rgba(255,255,255,0.35)] mt-0.5 truncate">
                  {j.scheduleKind === "cron" && `${j.schedule.expr}${j.schedule.tz ? ` (${j.schedule.tz})` : ""}`}
                  {j.scheduleKind === "every" && j.schedule.everyMs && (() => { const h = msToHuman(j.schedule.everyMs!); return `every ${h.val} ${h.unit}`; })()}
                  {j.scheduleKind === "at" && (j.schedule.at ? new Date(j.schedule.at).toLocaleString() : "")}
                  {" · agent: "}{j.agentName}
                </p>
                <p className="text-[11px] text-[rgba(255,255,255,0.25)] mt-0.5 truncate">{j.prompt}</p>
              </div>
              <div className="flex items-center gap-2 flex-shrink-0">
                <button onClick={() => handleToggle(j.id, !j.enabled)}
                  className={`w-8 h-[18px] rounded-full transition-colors relative ${j.enabled ? "bg-[#2688f9]" : "bg-[rgba(255,255,255,0.15)]"}`}>
                  <span className={`absolute top-[2px] w-[14px] h-[14px] rounded-full bg-white transition-transform ${j.enabled ? "left-[15px]" : "left-[2px]"}`} />
                </button>
                <button onClick={() => handleTrigger(j.id)}
                  disabled={triggering === j.id}
                  className="text-[11px] font-medium text-emerald-400 hover:text-emerald-300 disabled:opacity-50 transition-colors">
                  {triggering === j.id ? "Running…" : "▶ Run"}
                </button>
                <button onClick={() => showHistory(j.id)}
                  className="text-[11px] text-[rgba(255,255,255,0.4)] hover:text-[rgb(240,237,229)] transition-colors">
                  History
                </button>
                <button onClick={() => startEdit(j)}
                  className="text-[11px] text-[#2688f9] hover:text-[#5aa3fb] transition-colors">
                  Edit
                </button>
                <button onClick={() => handleDelete(j.id)}
                  className="text-[11px] text-red-400 hover:text-red-300 transition-colors">
                  Delete
                </button>
              </div>
            </div>

            {/* History panel */}
            {historyJobId === j.id && (
              <HistoryPanel jobId={j.id} />
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// HistoryPanel — expandable execution log for a single cron job
// ─────────────────────────────────────────────────────────────────────────────
function HistoryPanel({ jobId }: { jobId: string }) {
  const [entries, setEntries] = useState<CronJobHistory[]>([]);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState<number | null>(null);

  async function load() {
    try {
      const h = await GetCronJobHistory(jobId, 20);
      setEntries(h ?? []);
    } catch { setEntries([]); }
    finally { setLoading(false); }
  }

  useEffect(() => {
    load();
  }, [jobId]);

  // Poll every 3s while any entry is "running"
  useEffect(() => {
    const hasRunning = entries.some(e => e.status === "running");
    if (!hasRunning) return;
    const t = setInterval(load, 3000);
    return () => clearInterval(t);
  }, [entries]);

  function duration(h: CronJobHistory): string {
    if (!h.finishedAt) return "…";
    const ms = new Date(h.finishedAt).getTime() - new Date(h.startedAt).getTime();
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
  }

  function fmtTime(iso: string): string {
    try { return new Date(iso).toLocaleString(); } catch { return iso; }
  }

  return (
    <div className="border-t border-[rgba(255,255,255,0.06)] px-4 py-3">
      <div className="flex items-center justify-between mb-2">
        <p className="text-[11px] font-semibold text-[rgba(255,255,255,0.4)] uppercase tracking-wider">
          Execution History
        </p>
        <button onClick={load} className="text-[10px] text-[rgba(255,255,255,0.3)] hover:text-[rgba(255,255,255,0.6)] transition-colors">
          ↻ refresh
        </button>
      </div>

      {loading ? (
        <p className="text-[11px] text-[rgba(255,255,255,0.3)]">Loading…</p>
      ) : entries.length === 0 ? (
        <p className="text-[11px] text-[rgba(255,255,255,0.3)] italic">No executions yet.</p>
      ) : (
        <div className="space-y-1.5 max-h-[500px] overflow-y-auto pr-1">
          {entries.map((h) => {
            const isExpanded = expanded === h.id;
            const log = h.resultText || h.errorText || "";
            return (
              <div key={h.id}
                className="rounded-lg border border-[rgba(255,255,255,0.07)] overflow-hidden bg-[rgba(255,255,255,0.02)]">
                {/* Header row */}
                <button
                  onClick={() => setExpanded(isExpanded ? null : h.id)}
                  className="w-full flex items-center gap-2.5 px-3 py-2 text-left hover:bg-[rgba(255,255,255,0.03)] transition-colors">
                  <span className={`w-2 h-2 rounded-full flex-shrink-0 ${
                    h.status === "success" ? "bg-emerald-400" :
                    h.status === "error"   ? "bg-red-400" :
                    "bg-[#2688f9] animate-pulse"
                  }`} />
                  <span className="text-[11px] font-mono text-[rgba(255,255,255,0.5)] flex-1 truncate">
                    {fmtTime(h.startedAt)}
                  </span>
                  <span className={`text-[10px] font-medium px-1.5 py-0.5 rounded ${
                    h.status === "success" ? "text-emerald-400 bg-[rgba(52,199,89,0.1)]" :
                    h.status === "error"   ? "text-red-400 bg-[rgba(255,59,48,0.1)]" :
                    "text-[#2688f9] bg-[rgba(38,136,249,0.1)]"
                  }`}>{h.status}</span>
                  <span className="text-[10px] text-[rgba(255,255,255,0.25)] tabular-nums w-12 text-right">
                    {duration(h)}
                  </span>
                  <span className="text-[10px] text-[rgba(255,255,255,0.2)]">{isExpanded ? "▲" : "▼"}</span>
                </button>

                {/* Expanded log */}
                {isExpanded && (
                  <div className="border-t border-[rgba(255,255,255,0.05)]">
                    {log ? (
                      <pre className="px-3 py-2.5 text-[11px] font-mono leading-relaxed
                        text-[rgba(240,237,229,0.7)] whitespace-pre-wrap break-words
                        max-h-64 overflow-y-auto">
                        {log}
                      </pre>
                    ) : (
                      <p className="px-3 py-2 text-[11px] text-[rgba(255,255,255,0.25)] italic">No output.</p>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// Channels pane
// ─────────────────────────────────────────────────────────────────────────────
// ─────────────────────────────────────────────────────────────────────────────
// Chats — session history list
// ─────────────────────────────────────────────────────────────────────────────

function ChatsPane({ sessionMap, channels, onOpenChat }: {
  sessionMap: Record<string, SessionPreview[]>;
  channels: ChannelInfo[];
  onOpenChat: (agentName: string, sessionID: string) => void;
}) {
  const allSessions: { agentName: string; session: SessionPreview }[] = [];
  for (const c of channels) {
    const sessions = sessionMap[c.name] ?? [];
    for (const s of sessions) {
      allSessions.push({ agentName: c.name, session: s });
    }
  }
  allSessions.sort((a, b) => b.session.lastMs - a.session.lastMs);

  function formatTime(ms: number) {
    if (!ms) return "";
    const diff = Date.now() - ms;
    if (diff < 60_000) return "just now";
    if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
    if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
    return new Date(ms).toLocaleDateString();
  }

  return (
    <div className="p-8">
      <header className="mb-6 flex items-center justify-between">
        <div>
          <h2 className="text-[22px] font-semibold text-[rgb(240,237,229)] tracking-[-0.43px]">Chats</h2>
          <p className="text-[13px] text-[rgba(255,255,255,0.4)] mt-1">Your conversation history</p>
        </div>
        <button
          onClick={() => {
            const agentName = channels[0]?.name;
            if (agentName) onOpenChat(agentName, "s" + Date.now());
          }}
          className="px-4 py-2 text-[13px] bg-[#2688f9] text-white rounded-xl hover:bg-[#1a7ae8] transition-colors font-medium"
        >
          + New Chat
        </button>
      </header>

      {allSessions.length === 0 ? (
        <p className="text-[13px] text-[rgba(255,255,255,0.3)]">No conversations yet.</p>
      ) : (
        <div className="space-y-2 max-w-2xl">
          {allSessions.map(({ agentName, session }) => (
            <button
              key={`${agentName}/${session.id}`}
              onClick={() => onOpenChat(agentName, session.id)}
              className="w-full text-left bg-[rgba(255,255,255,0.04)] border border-[rgba(255,255,255,0.08)] rounded-2xl px-4 py-3 hover:bg-[rgba(255,255,255,0.07)] hover:border-[rgba(255,255,255,0.14)] transition-all group"
            >
              <div className="flex items-center justify-between mb-1">
                <span className="text-[11px] font-medium text-[rgba(38,136,249,0.8)] uppercase tracking-wider">{agentName}</span>
                <span className="text-[11px] text-[rgba(255,255,255,0.25)]">{formatTime(session.lastMs)}</span>
              </div>
              <p className="text-[13px] text-[rgba(240,237,229,0.75)] truncate leading-snug group-hover:text-[rgba(240,237,229,0.9)]">
                {session.preview}
              </p>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

function AgentsPane() {
  return (
    <div className="p-8">
      <header className="mb-6">
        <h2 className="text-[22px] font-semibold text-[rgb(240,237,229)] tracking-[-0.43px]">Agents</h2>
        <p className="text-[13px] text-[rgba(255,255,255,0.4)] mt-1">Edit each agent's provider, model and system prompt</p>
      </header>
      <AgentsEditor />
    </div>
  );
}

function ChannelsPane() {
  return (
    <div className="p-8">
      <header className="mb-6">
        <h2 className="text-[22px] font-semibold text-[rgb(240,237,229)] tracking-[-0.43px]">Channels</h2>
        <p className="text-[13px] text-[rgba(255,255,255,0.4)] mt-1">Messaging integrations</p>
      </header>
      <ChannelsSettings />
    </div>
  );
}

function AgentsEditor() {
  const [cfg, setCfg] = useState<Config | null>(null);
  const [providers, setProviders] = useState<ProviderInfo[]>([]);
  const [editing, setEditing] = useState<string | null>(null);
  const [form, setForm] = useState<{ provider: string; model: string; system_prompt: string }>({ provider: "", model: "", system_prompt: "" });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    GetConfig().then(c => setCfg(c)).catch(() => {});
    GetProviders().then(p => setProviders(p ?? [])).catch(() => {});
  }, []);

  function startEdit(agent: Config["agents"][0]) {
    setForm({ provider: agent.provider ?? "", model: agent.model ?? "", system_prompt: agent.system_prompt ?? "" });
    setEditing(agent.name);
    setError("");
    setSuccess(null);
  }

  async function handleSave() {
    if (!cfg || editing === null) return;
    setSaving(true); setError(""); setSuccess(null);
    try {
      const updated: Config = {
        ...cfg,
        agents: cfg.agents.map(a =>
          a.name === editing ? { ...a, provider: form.provider, model: form.model, system_prompt: form.system_prompt } : a
        ),
      };
      await SaveConfig(updated);
      setCfg(updated);
      setEditing(null);
      setSuccess(editing);
      setTimeout(() => setSuccess(null), 2500);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  }

  const agents = cfg?.agents ?? [];
  const providerIds = providers.map(p => p.id);

  return (
    <div className="space-y-3 max-w-2xl">
      {agents.length === 0 && <p className="text-[13px] text-[rgba(255,255,255,0.3)]">No agents configured.</p>}
      {agents.map(agent => (
        <div key={agent.name} className="bg-[rgba(255,255,255,0.04)] border border-[rgba(255,255,255,0.08)] rounded-2xl">
          {/* Header row */}
          <div className="flex items-center justify-between px-4 py-3">
            <div className="flex items-center gap-2">
              <span className="text-[14px] font-semibold text-[rgb(240,237,229)]">{agent.name}</span>
              {success === agent.name && <span className="text-[11px] text-emerald-400">✓ saved</span>}
              {editing !== agent.name && (
                <>
                  <span className="px-2 py-0.5 text-[11px] bg-[rgba(38,136,249,0.15)] text-[#2688f9] rounded-full">{agent.provider || "—"}</span>
                  {agent.model && <span className="text-[11px] text-[rgba(255,255,255,0.35)]">{agent.model}</span>}
                </>
              )}
            </div>
            {editing !== agent.name ? (
              <button onClick={() => startEdit(agent)}
                className="px-3 py-1.5 text-[12px] bg-[rgba(255,255,255,0.06)] text-[rgba(255,255,255,0.5)] rounded-lg hover:bg-[rgba(255,255,255,0.1)] hover:text-[rgba(255,255,255,0.8)] transition-colors">
                Edit
              </button>
            ) : (
              <div className="flex gap-1.5">
                <button onClick={handleSave} disabled={saving}
                  className="px-3 py-1.5 text-[12px] bg-[#2688f9] text-white rounded-lg hover:bg-[#1a7ae8] disabled:opacity-50 transition-colors font-semibold">
                  {saving ? "…" : "Save"}
                </button>
                <button onClick={() => { setEditing(null); setError(""); }}
                  className="px-3 py-1.5 text-[12px] bg-[rgba(255,255,255,0.06)] text-[rgba(255,255,255,0.5)] rounded-lg hover:bg-[rgba(255,255,255,0.1)] transition-colors">
                  Cancel
                </button>
              </div>
            )}
          </div>

          {/* Edit form */}
          {editing === agent.name && (
            <div className="px-4 pb-4 space-y-3 border-t border-[rgba(255,255,255,0.06)] pt-3">
              {error && <p className="text-[12px] text-red-400">{error}</p>}
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-[11px] text-[rgba(255,255,255,0.4)] mb-1">Provider</label>
                  <select className={selectCls} value={form.provider} onChange={e => setForm(f => ({ ...f, provider: e.target.value }))}>
                    <option value="">— none —</option>
                    {providerIds.map(id => <option key={id} value={id}>{id}</option>)}
                  </select>
                </div>
                <div>
                  <label className="block text-[11px] text-[rgba(255,255,255,0.4)] mb-1">Model</label>
                  <input className={inputCls} placeholder="e.g. claude-sonnet-4-6"
                    value={form.model} onChange={e => setForm(f => ({ ...f, model: e.target.value }))} />
                </div>
              </div>
              <div>
                <label className="block text-[11px] text-[rgba(255,255,255,0.4)] mb-1">System Prompt</label>
                <textarea className={`${inputCls} resize-none`} rows={3}
                  value={form.system_prompt} onChange={e => setForm(f => ({ ...f, system_prompt: e.target.value }))} />
              </div>
            </div>
          )}
        </div>
      ))}
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// WhipFlow pane
// ─────────────────────────────────────────────────────────────────────────────

const BUILTIN_PROVIDERS = ["claude-code", "claude", "opencode", "aider", "pi"];

// Minimal WhipFlow syntax highlighter
function WhipHighlight({ code }: { code: string }) {
  const lines = code.split("\n");
  return (
    <pre className="text-[12.5px] leading-[1.6] font-mono whitespace-pre-wrap break-words">
      {lines.map((line, i) => <WhipLine key={i} line={line} />)}
    </pre>
  );
}

function WhipLine({ line }: { line: string }) {
  // Comment
  if (/^\s*#/.test(line)) {
    return <div><span className="text-[rgba(255,255,255,0.3)] italic">{line}</span>{"\n"}</div>;
  }
  // Keywords: agent, let, const, session, for, in, if, else, parallel, run
  const tokenized = tokenizeWhip(line);
  return <div>{tokenized}{"\n"}</div>;
}

function tokenizeWhip(line: string): React.ReactNode[] {
  const KEYWORDS = /\b(agent|let|const|session|for|in|if|else|parallel|run|model|prompt|end)\b/g;
  const STRING = /(["'`])(?:\\.|(?!\1)[^\\])*?\1/g;
  const INTERP = /\{[^}]+\}/g;
  const NUMBER = /\b\d+(\.\d+)?\b/g;

  type Token = { type: "kw" | "str" | "interp" | "num" | "plain"; value: string };
  const tokens: Token[] = [];
  let pos = 0;

  const allMatches: { index: number; end: number; type: Token["type"]; value: string }[] = [];

  for (const re of [
    { re: KEYWORDS, type: "kw" as const },
    { re: STRING, type: "str" as const },
    { re: INTERP, type: "interp" as const },
    { re: NUMBER, type: "num" as const },
  ]) {
    re.re.lastIndex = 0;
    let m: RegExpExecArray | null;
    while ((m = re.re.exec(line)) !== null) {
      allMatches.push({ index: m.index, end: m.index + m[0].length, type: re.type, value: m[0] });
    }
  }

  // Sort by index, remove overlaps
  allMatches.sort((a, b) => a.index - b.index);
  const deduped: typeof allMatches = [];
  let lastEnd = 0;
  for (const m of allMatches) {
    if (m.index >= lastEnd) { deduped.push(m); lastEnd = m.end; }
  }

  for (const m of deduped) {
    if (m.index > pos) tokens.push({ type: "plain", value: line.slice(pos, m.index) });
    tokens.push({ type: m.type, value: m.value });
    pos = m.end;
  }
  if (pos < line.length) tokens.push({ type: "plain", value: line.slice(pos) });

  return tokens.map((t, i) => {
    const cls =
      t.type === "kw"     ? "text-[#7dd3fc] font-semibold" :
      t.type === "str"    ? "text-[#86efac]" :
      t.type === "interp" ? "text-[#fcd34d]" :
      t.type === "num"    ? "text-[#f9a8d4]" :
                            "text-[rgba(255,255,255,0.75)]";
    return <span key={i} className={cls}>{t.value}</span>;
  });
}

const inputCls =
  "w-full px-3 py-2 text-[13px] border border-[rgba(255,255,255,0.1)] rounded-xl " +
  "focus:outline-none focus:ring-2 focus:ring-[rgba(38,136,249,0.4)] " +
  "bg-[rgba(255,255,255,0.05)] text-[rgb(240,237,229)] placeholder-[rgba(255,255,255,0.2)]";

const selectCls =
  "w-full px-3 py-2 text-[13px] border border-[rgba(255,255,255,0.1)] rounded-xl " +
  "focus:outline-none focus:ring-2 focus:ring-[rgba(38,136,249,0.4)] " +
  "bg-[rgb(40,40,38)] text-[rgb(240,237,229)]";

// ─────────────────────────────────────────────────────────────────────────────
// Vault
// ─────────────────────────────────────────────────────────────────────────────

function VaultPane() {
  const [entries, setEntries] = useState<VaultEntry[]>([]);
  const [newKey, setNewKey] = useState("");
  const [newVal, setNewVal] = useState("");
  const [adding, setAdding] = useState(false);
  const [editKey, setEditKey] = useState<string | null>(null);
  const [editVal, setEditVal] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [revealed, setRevealed] = useState<Set<string>>(new Set());

  async function load() {
    const list = await GetVault().catch(() => []);
    setEntries(list ?? []);
  }

  useEffect(() => { load(); }, []);

  async function handleAdd() {
    if (!newKey.trim() || !newVal.trim()) return;
    setSaving(true); setError("");
    try {
      await SetVaultEntry(newKey.trim(), newVal.trim());
      setNewKey(""); setNewVal(""); setAdding(false);
      await load();
    } catch (e) { setError(String(e)); }
    finally { setSaving(false); }
  }

  async function handleSave(key: string) {
    setSaving(true); setError("");
    try {
      await SetVaultEntry(key, editVal);
      setEditKey(null);
      await load();
    } catch (e) { setError(String(e)); }
    finally { setSaving(false); }
  }

  async function handleDelete(key: string) {
    setSaving(true); setError("");
    try {
      await DeleteVaultEntry(key);
      await load();
    } catch (e) { setError(String(e)); }
    finally { setSaving(false); }
  }

  function toggleReveal(key: string) {
    setRevealed(prev => {
      const next = new Set(prev);
      next.has(key) ? next.delete(key) : next.add(key);
      return next;
    });
  }

  return (
    <div className="p-8 max-w-2xl">
      <header className="mb-6">
        <h2 className="text-[22px] font-semibold text-[rgb(240,237,229)] tracking-[-0.43px]">Vault</h2>
        <p className="text-[13px] text-[rgba(255,255,255,0.4)] mt-1">
          Key-value secrets injected as environment variables when running bash, exec, process and CLI workflows.
        </p>
      </header>

      {error && <p className="mb-4 text-[12px] text-red-400">{error}</p>}

      {/* Entry list */}
      <div className="space-y-2 mb-4">
        {entries.length === 0 && !adding && (
          <p className="text-[13px] text-[rgba(255,255,255,0.3)]">No secrets stored yet.</p>
        )}
        {entries.map(e => (
          <div key={e.key} className="bg-[rgba(255,255,255,0.04)] border border-[rgba(255,255,255,0.08)] rounded-2xl overflow-hidden">
            <div className="flex items-center gap-3 px-4 py-3">
              <span className="font-mono text-[13px] text-[rgb(240,237,229)] flex-shrink-0 w-44 truncate">{e.key}</span>
              {editKey === e.key ? (
                <input
                  autoFocus
                  type="text"
                  value={editVal}
                  onChange={ev => setEditVal(ev.target.value)}
                  className={inputCls + " flex-1 font-mono text-[12px]"}
                  onKeyDown={ev => { if (ev.key === "Enter") handleSave(e.key); if (ev.key === "Escape") setEditKey(null); }}
                />
              ) : (
                <span className="flex-1 font-mono text-[12px] text-[rgba(255,255,255,0.45)] truncate">
                  {revealed.has(e.key) ? e.value : "••••••••••••"}
                </span>
              )}
              <div className="flex gap-1.5 flex-shrink-0">
                {editKey === e.key ? (
                  <>
                    <button onClick={() => handleSave(e.key)} disabled={saving}
                      className="px-2.5 py-1 text-[11px] bg-[#2688f9] text-white rounded-lg hover:bg-[#1a7ae8] disabled:opacity-50 font-semibold">Save</button>
                    <button onClick={() => setEditKey(null)}
                      className="px-2.5 py-1 text-[11px] bg-[rgba(255,255,255,0.06)] text-[rgba(255,255,255,0.5)] rounded-lg hover:bg-[rgba(255,255,255,0.1)]">Cancel</button>
                  </>
                ) : (
                  <>
                    <button onClick={() => toggleReveal(e.key)}
                      className="px-2.5 py-1 text-[11px] bg-[rgba(255,255,255,0.06)] text-[rgba(255,255,255,0.4)] rounded-lg hover:bg-[rgba(255,255,255,0.1)] font-mono">
                      {revealed.has(e.key) ? "hide" : "show"}
                    </button>
                    <button onClick={() => { setEditKey(e.key); setEditVal(e.value); }}
                      className="px-2.5 py-1 text-[11px] bg-[rgba(255,255,255,0.06)] text-[rgba(255,255,255,0.5)] rounded-lg hover:bg-[rgba(255,255,255,0.1)]">Edit</button>
                    <button onClick={() => handleDelete(e.key)} disabled={saving}
                      className="px-2.5 py-1 text-[11px] bg-[rgba(239,68,68,0.12)] text-red-400 rounded-lg hover:bg-[rgba(239,68,68,0.2)] disabled:opacity-50">Del</button>
                  </>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Add new */}
      {adding ? (
        <div className="bg-[rgba(255,255,255,0.04)] border border-[rgba(38,136,249,0.3)] rounded-2xl p-4 space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-[11px] text-[rgba(255,255,255,0.4)] mb-1">Key</label>
              <input autoFocus value={newKey} onChange={e => setNewKey(e.target.value)}
                placeholder="e.g. GITHUB_TOKEN" className={inputCls + " font-mono"} />
            </div>
            <div>
              <label className="block text-[11px] text-[rgba(255,255,255,0.4)] mb-1">Value</label>
              <input type="password" value={newVal} onChange={e => setNewVal(e.target.value)}
                placeholder="secret value" className={inputCls + " font-mono"}
                onKeyDown={e => { if (e.key === "Enter") handleAdd(); if (e.key === "Escape") setAdding(false); }} />
            </div>
          </div>
          <div className="flex gap-2">
            <button onClick={handleAdd} disabled={saving || !newKey.trim() || !newVal.trim()}
              className="px-4 py-1.5 text-[12px] bg-[#2688f9] text-white rounded-lg hover:bg-[#1a7ae8] disabled:opacity-40 font-semibold">
              {saving ? "Saving…" : "Add Secret"}
            </button>
            <button onClick={() => { setAdding(false); setNewKey(""); setNewVal(""); }}
              className="px-4 py-1.5 text-[12px] bg-[rgba(255,255,255,0.06)] text-[rgba(255,255,255,0.5)] rounded-lg hover:bg-[rgba(255,255,255,0.1)]">
              Cancel
            </button>
          </div>
        </div>
      ) : (
        <button onClick={() => setAdding(true)}
          className="px-4 py-2 text-[13px] bg-[rgba(255,255,255,0.06)] text-[rgba(255,255,255,0.6)] rounded-xl hover:bg-[rgba(255,255,255,0.1)] hover:text-[rgba(255,255,255,0.9)] transition-colors border border-[rgba(255,255,255,0.08)]">
          + Add Secret
        </button>
      )}
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// Browser (CDP)
// ─────────────────────────────────────────────────────────────────────────────
function BrowserPane() {
  const [status, setStatus] = useState<BrowserStatus | null>(null);
  const [testing, setTesting] = useState(false);
  const [launching, setLaunching] = useState(false);
  const [shortcuts, setShortcuts] = useState<ShortcutInfo[]>([]);
  const [runningCmd, setRunningCmd] = useState<string | null>(null);
  const [cmdResult, setCmdResult] = useState<Record<string, unknown>[] | null>(null);
  const [cmdError, setCmdError] = useState("");
  const [cmdArgs, setCmdArgs] = useState("");

  async function testCDP() {
    setTesting(true);
    try {
      const s = await BrowserTestCDP();
      setStatus(s);
    } catch (e) {
      setStatus({ connected: false, cdpURL: "", browser: "", error: String(e) });
    } finally {
      setTesting(false);
    }
  }

  async function launchChrome() {
    setLaunching(true);
    try {
      const s = await BrowserLaunchChrome();
      setStatus(s);
    } catch (e) {
      setStatus({ connected: false, cdpURL: "", browser: "", error: String(e) });
    } finally {
      setLaunching(false);
    }
  }

  async function loadShortcuts() {
    const list = await BrowserListShortcuts().catch(() => []);
    setShortcuts(list ?? []);
  }

  async function runCmd(file: string, command: string) {
    const key = `${file}:${command}`;
    setRunningCmd(key);
    setCmdResult(null);
    setCmdError("");
    try {
      const args = cmdArgs.trim() ? cmdArgs.trim().split(/\s+/) : [];
      const rows = await BrowserRunShortcut(file, command, args);
      setCmdResult(rows);
    } catch (e) {
      setCmdError(String(e));
    } finally {
      setRunningCmd(null);
    }
  }

  useEffect(() => { testCDP(); loadShortcuts(); }, []);

  return (
    <div className="p-8 max-w-3xl">
      <header className="mb-6">
        <h2 className="text-[22px] font-semibold text-[rgb(240,237,229)] tracking-[-0.43px]">Browser</h2>
        <p className="text-[13px] text-[rgba(255,255,255,0.4)] mt-1">
          Chrome DevTools Protocol (CDP) connection and browser automation shortcuts.
        </p>
      </header>

      {/* Status card */}
      <div className="bg-[rgba(255,255,255,0.04)] border border-[rgba(255,255,255,0.08)] rounded-2xl p-5 mb-6">
        <div className="flex items-center gap-3 mb-3">
          <span className={`w-2.5 h-2.5 rounded-full ${status?.connected ? "bg-green-400" : "bg-red-400"}`} />
          <span className="text-[14px] font-medium text-[rgb(240,237,229)]">
            {status === null ? "Checking..." : status.connected ? "Connected" : "Disconnected"}
          </span>
        </div>

        {status?.connected && (
          <div className="space-y-1.5 mb-4">
            {status.browser && (
              <p className="text-[12px] text-[rgba(255,255,255,0.5)]">
                <span className="text-[rgba(255,255,255,0.3)]">Browser:</span> {status.browser}
              </p>
            )}
            <p className="text-[12px] text-[rgba(255,255,255,0.5)] font-mono break-all">
              <span className="text-[rgba(255,255,255,0.3)]">CDP URL:</span> {status.cdpURL}
            </p>
          </div>
        )}

        {status && !status.connected && status.error && (
          <p className="text-[12px] text-[rgba(255,255,255,0.35)] mb-4">{status.error}</p>
        )}

        <div className="flex gap-2">
          <button
            onClick={testCDP}
            disabled={testing}
            className="px-4 py-2 text-[13px] rounded-lg bg-[rgba(255,255,255,0.08)] hover:bg-[rgba(255,255,255,0.12)] text-[rgb(240,237,229)] disabled:opacity-40 transition-colors"
          >
            {testing ? "Testing..." : "Test CDP"}
          </button>
          <button
            onClick={launchChrome}
            disabled={launching}
            className="px-4 py-2 text-[13px] rounded-lg bg-[rgba(38,136,249,0.2)] hover:bg-[rgba(38,136,249,0.3)] text-[#2688f9] disabled:opacity-40 transition-colors"
          >
            {launching ? "Launching..." : "Launch Chrome with CDP"}
          </button>
        </div>
      </div>

      {/* Shortcuts */}
      <h3 className="text-[15px] font-medium text-[rgb(240,237,229)] mb-3">Shortcuts</h3>
      <p className="text-[12px] text-[rgba(255,255,255,0.3)] mb-4">
        YAML browser automation adapters in <code className="text-[rgba(255,255,255,0.45)]">~/.clawfirm/shortcuts/</code>.
        Requires Chrome connected via CDP.
      </p>

      {/* Args input */}
      <div className="mb-4">
        <input
          type="text"
          value={cmdArgs}
          onChange={e => setCmdArgs(e.target.value)}
          placeholder="Command arguments (space-separated)..."
          className="w-full px-3 py-2 text-[13px] rounded-lg bg-[rgba(255,255,255,0.06)] border border-[rgba(255,255,255,0.08)] text-[rgb(240,237,229)] placeholder:text-[rgba(255,255,255,0.25)] outline-none focus:border-[rgba(38,136,249,0.4)]"
        />
      </div>

      {shortcuts.length === 0 ? (
        <p className="text-[13px] text-[rgba(255,255,255,0.3)]">No shortcuts found.</p>
      ) : (
        <div className="space-y-3">
          {shortcuts.map(sc => (
            <div key={sc.file} className="bg-[rgba(255,255,255,0.04)] border border-[rgba(255,255,255,0.08)] rounded-2xl p-4">
              <div className="flex items-center gap-2 mb-2">
                <span className="text-[14px] font-medium text-[rgb(240,237,229)]">{sc.platform}</span>
                <span className="text-[11px] text-[rgba(255,255,255,0.25)] font-mono">{sc.file}</span>
              </div>
              <div className="flex flex-wrap gap-1.5">
                {sc.commands.map(cmd => {
                  const key = `${sc.file}:${cmd}`;
                  const isRunning = runningCmd === key;
                  return (
                    <button
                      key={cmd}
                      onClick={() => runCmd(sc.file, cmd)}
                      disabled={!!runningCmd || !status?.connected}
                      className="px-3 py-1.5 text-[12px] rounded-md bg-[rgba(255,255,255,0.06)] hover:bg-[rgba(255,255,255,0.1)] text-[rgba(240,237,229,0.7)] disabled:opacity-30 transition-colors font-mono"
                    >
                      {isRunning ? `${cmd}...` : cmd}
                    </button>
                  );
                })}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Command result */}
      {cmdError && (
        <div className="mt-4 p-3 rounded-lg bg-[rgba(255,80,80,0.1)] border border-[rgba(255,80,80,0.2)]">
          <p className="text-[12px] text-red-400 font-mono whitespace-pre-wrap">{cmdError}</p>
        </div>
      )}
      {cmdResult && cmdResult.length > 0 && (
        <div className="mt-4 overflow-x-auto">
          <table className="w-full text-[12px] text-[rgba(240,237,229,0.7)]">
            <thead>
              <tr className="border-b border-[rgba(255,255,255,0.08)]">
                {Object.keys(cmdResult[0]).map(k => (
                  <th key={k} className="text-left py-2 px-2 font-medium text-[rgba(255,255,255,0.5)]">{k}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {cmdResult.map((row, i) => (
                <tr key={i} className="border-b border-[rgba(255,255,255,0.04)]">
                  {Object.values(row).map((v, j) => (
                    <td key={j} className="py-1.5 px-2 font-mono truncate max-w-[200px]">{String(v ?? "")}</td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function WhipflowPane() {
  const [cfg, setCfg] = useState<Config | null>(null);
  const [defaultProvider, setDefaultProvider] = useState("");
  const [customProvider, setCustomProvider] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [cfgOpen, setCfgOpen] = useState(false);

  // file browser
  const [whipFiles, setWhipFiles] = useState<string[]>([]);
  const [selectedFile, setSelectedFile] = useState<string | null>(null);
  const [fileContent, setFileContent] = useState<string>("");
  const [fileError, setFileError] = useState("");

  useEffect(() => { load(); loadFiles(); }, []);

  async function load() {
    try {
      const c = await GetConfig();
      setCfg(c);
      const dp = c.whipflow?.default_provider ?? "";
      const knownAgents = (c.agents ?? []).map((a: { name: string }) => a.name);
      if (dp === "" || BUILTIN_PROVIDERS.includes(dp) || knownAgents.includes(dp)) {
        setDefaultProvider(dp);
      } else {
        setDefaultProvider("__custom__");
        setCustomProvider(dp);
      }
    } catch (e) { setError(String(e)); }
  }

  async function loadFiles() {
    try {
      const files = await ListWhipFiles();
      setWhipFiles(files ?? []);
      if (files?.length > 0) {
        await selectFile(files[0]);
      }
    } catch (e) { /* ignore */ }
  }

  async function selectFile(path: string) {
    setSelectedFile(path);
    setFileError("");
    try {
      const content = await GetWhipFileContent(path);
      setFileContent(content);
    } catch (e) { setFileError(String(e)); setFileContent(""); }
  }

  async function handleSave() {
    if (!cfg) return;
    setSaving(true); setError(""); setSuccess("");
    try {
      const dp = defaultProvider === "__custom__" ? customProvider.trim() : (defaultProvider === "" ? "" : defaultProvider);
      const updated: Config = {
        ...cfg,
        whipflow: { ...(cfg.whipflow ?? {}), default_provider: dp },
      };
      await SaveConfig(updated);
      setCfg(updated);
      setSuccess("Saved.");
      setTimeout(() => setSuccess(""), 2500);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally { setSaving(false); }
  }

  const agentNames = cfg?.agents?.map(a => a.name) ?? [];
  const basename = (p: string) => p.split("/").pop() ?? p;

  return (
    <div className="flex h-full overflow-hidden">
      {/* ── Left: file list ── */}
      <div className="w-52 flex-shrink-0 border-r border-[rgba(255,255,255,0.07)] flex flex-col">
        <div className="px-4 py-3 border-b border-[rgba(255,255,255,0.07)]">
          <span className="text-[11px] font-semibold text-[rgba(255,255,255,0.35)] uppercase tracking-wider">Workflows</span>
        </div>
        <div className="flex-1 overflow-y-auto py-1">
          {whipFiles.length === 0 ? (
            <p className="px-4 py-3 text-[12px] text-[rgba(255,255,255,0.25)]">No .whip files in<br/>~/.clawfirm/workflows/</p>
          ) : whipFiles.map(f => (
            <button
              key={f}
              onClick={() => selectFile(f)}
              className={`w-full text-left px-4 py-2 text-[12.5px] truncate transition-colors ${
                selectedFile === f
                  ? "bg-[rgba(255,255,255,0.08)] text-[rgb(240,237,229)]"
                  : "text-[rgba(255,255,255,0.55)] hover:bg-[rgba(255,255,255,0.04)] hover:text-[rgb(240,237,229)]"
              }`}
            >
              <span className="text-[rgba(255,255,255,0.25)] mr-1">⚡</span>{basename(f)}
            </button>
          ))}
        </div>

        {/* ── Config section (collapsed by default) ── */}
        <div className="border-t border-[rgba(255,255,255,0.07)]">
          <button
            onClick={() => setCfgOpen(o => !o)}
            className="w-full flex items-center justify-between px-4 py-2.5 text-[11px] text-[rgba(255,255,255,0.35)] hover:text-[rgba(255,255,255,0.6)] transition-colors"
          >
            <span className="font-semibold uppercase tracking-wider">Config</span>
            <span>{cfgOpen ? "▲" : "▼"}</span>
          </button>
          {cfgOpen && (
            <div className="px-3 pb-3 space-y-2">
              {error && <p className="text-[11px] text-red-400">{error}</p>}
              <select
                className="w-full bg-[rgba(255,255,255,0.05)] border border-[rgba(255,255,255,0.1)] rounded-lg px-2 py-1.5 text-[11.5px] text-[rgb(240,237,229)] outline-none"
                value={defaultProvider}
                onChange={e => setDefaultProvider(e.target.value)}
              >
                <option value="">— default (first agent) —</option>
                <optgroup label="Built-in">
                  {BUILTIN_PROVIDERS.map(p => <option key={p} value={p}>{p}</option>)}
                </optgroup>
                {agentNames.length > 0 && (
                  <optgroup label="Agents">
                    {agentNames.map(n => <option key={n} value={n}>{n}</option>)}
                  </optgroup>
                )}
                <option value="__custom__">Custom…</option>
              </select>
              {defaultProvider === "__custom__" && (
                <input
                  className="w-full bg-[rgba(255,255,255,0.05)] border border-[rgba(255,255,255,0.1)] rounded-lg px-2 py-1.5 text-[11.5px] text-[rgb(240,237,229)] outline-none"
                  placeholder="provider name"
                  value={customProvider}
                  onChange={e => setCustomProvider(e.target.value)}
                />
              )}
              <div className="flex items-center gap-2">
                <button onClick={handleSave} disabled={saving || !cfg}
                  className="px-3 py-1 bg-[#2688f9] text-white text-[11px] rounded-lg hover:bg-[#1a7ae8] disabled:opacity-50 transition-colors font-semibold">
                  {saving ? "…" : "Save"}
                </button>
                {success && <span className="text-[11px] text-emerald-400">{success}</span>}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* ── Right: code viewer ── */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {selectedFile ? (
          <>
            <div className="px-5 py-3 border-b border-[rgba(255,255,255,0.07)] flex items-center gap-2">
              <span className="text-[rgba(255,255,255,0.4)] text-[12px]">⚡</span>
              <span className="text-[13px] font-medium text-[rgb(240,237,229)]">{basename(selectedFile)}</span>
              <span className="text-[11px] text-[rgba(255,255,255,0.25)] ml-1 truncate">{selectedFile}</span>
            </div>
            <div className="flex-1 overflow-auto p-5">
              {fileError ? (
                <p className="text-[12px] text-red-400">{fileError}</p>
              ) : (
                <div className="bg-[rgba(0,0,0,0.25)] rounded-xl p-4 border border-[rgba(255,255,255,0.06)]">
                  <WhipHighlight code={fileContent} />
                </div>
              )}
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center text-[13px] text-[rgba(255,255,255,0.2)]">
            Select a .whip file to preview
          </div>
        )}
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// Settings pane
// ─────────────────────────────────────────────────────────────────────────────
function SettingsPane() {
  const [content, setContent] = useState("");
  const [filePath, setFilePath] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  useEffect(() => {
    GetConfigRaw()
      .then((r) => { setFilePath(r.path ?? ""); setContent(r.content ?? ""); })
      .catch((e) => setError(String(e)));
  }, []);

  async function handleSave() {
    setSaving(true); setError(""); setSuccess("");
    try {
      await SaveConfigRaw(content);
      setSuccess("Saved & reloaded.");
      setTimeout(() => setSuccess(""), 3000);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    } finally { setSaving(false); }
  }

  return (
    <div className="p-8 h-full flex flex-col">
      <header className="mb-6 flex-shrink-0">
        <h2 className="text-[22px] font-semibold text-[rgb(240,237,229)] tracking-[-0.43px]">Settings</h2>
        <p className="text-[13px] text-[rgba(255,255,255,0.3)] mt-1 font-mono">{filePath}</p>
      </header>
      <textarea
        value={content}
        onChange={(e) => setContent(e.target.value)}
        spellCheck={false}
        className="flex-1 min-h-0 w-full max-w-2xl px-4 py-3 text-[13px] font-mono bg-[rgba(255,255,255,0.05)] border border-[rgba(255,255,255,0.1)] rounded-2xl focus:outline-none focus:ring-2 focus:ring-[rgba(38,136,249,0.4)] resize-none leading-relaxed text-[rgb(240,237,229)] placeholder-[rgba(255,255,255,0.2)] backdrop-blur-xl"
      />
      <div className="flex items-center gap-3 mt-4 flex-shrink-0">
        <button onClick={handleSave} disabled={saving}
          className="px-5 py-2 bg-[#2688f9] text-white text-[13px] rounded-xl hover:bg-[#1a7ae8] disabled:opacity-50 transition-colors font-semibold">
          {saving ? "Saving…" : "Save & Reload"}
        </button>
        {error && <p className="text-[13px] text-red-400">{error}</p>}
        {success && <p className="text-[13px] text-emerald-400">{success}</p>}
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// Channels settings
// ─────────────────────────────────────────────────────────────────────────────
function ChannelsSettings() {
  return (
    <div className="grid gap-5 sm:grid-cols-2 max-w-2xl">
      <WhatsAppCard />
      <FeishuCard />
    </div>
  );
}

const POLL_MS = 2000;

function WhatsAppCard() {
  const [status, setStatus] = useState("disconnected");
  const [qrURL, setQRURL] = useState("");

  useEffect(() => {
    let active = true;
    async function poll() {
      try {
        const s = await GetWhatsAppStatus();
        if (!active) return;
        setStatus(s);
        if (s === "qr_pending") { const qr = await GetWhatsAppQR(); if (active) setQRURL(qr ?? ""); }
        else setQRURL("");
      } catch {}
    }
    poll();
    const id = setInterval(poll, POLL_MS);
    return () => { active = false; clearInterval(id); };
  }, []);

  const waMap: Record<string, { label: string; cls: string }> = {
    connected:    { label: "Connected",    cls: "bg-[rgba(52,199,89,0.15)] text-emerald-400" },
    qr_pending:   { label: "Scan QR",      cls: "bg-[rgba(255,179,64,0.15)] text-amber-400" },
    disconnected: { label: "Disconnected", cls: "bg-[rgba(255,255,255,0.08)] text-[rgba(255,255,255,0.4)]" },
    logged_out:   { label: "Logged out",   cls: "bg-[rgba(255,69,58,0.15)] text-red-400" },
    disabled:     { label: "Disabled",     cls: "bg-[rgba(255,255,255,0.05)] text-[rgba(255,255,255,0.25)]" },
  };

  return (
    <Card title="WhatsApp"
      badge={<Badge {...(waMap[status] ?? waMap.disconnected)} />}
      action={status === "connected"
        ? <button onClick={() => LogoutWhatsApp().catch(() => {})} className="text-[11px] text-red-400 hover:text-red-300">Disconnect</button>
        : null}>
      {status === "qr_pending" && qrURL && (
        <div className="flex flex-col items-center gap-2 pt-2">
          <p className="text-[11px] text-[rgba(255,255,255,0.5)] text-center">WhatsApp → Linked Devices → Link a Device → scan QR</p>
          <img src={qrURL} alt="QR" className="w-44 h-44 rounded-xl border border-[rgba(255,255,255,0.1)]" />
        </div>
      )}
      {status === "qr_pending" && !qrURL && <p className="text-[11px] text-[rgba(255,255,255,0.3)] italic pt-1">Generating QR…</p>}
      {status === "disabled" && (
        <p className="text-[11px] text-[rgba(255,255,255,0.3)] pt-1">
          Set <code className="font-mono bg-[rgba(255,255,255,0.08)] px-1 rounded">whatsapp.enabled: true</code> in config.yml to enable.
        </p>
      )}
      {(status === "disconnected" || status === "logged_out") && (
        <p className="text-[11px] text-[rgba(255,255,255,0.3)] italic pt-1">
          {status === "logged_out" ? "Logged out — restart app to pair again." : "Waiting for gateway…"}
        </p>
      )}
      {status === "connected" && <p className="text-[11px] text-[rgba(255,255,255,0.3)] pt-1">Connected and receiving messages.</p>}
    </Card>
  );
}

function FeishuCard() {
  const [appID, setAppID] = useState("");
  const [appSecret, setAppSecret] = useState("");
  const [secretMasked, setSecretMasked] = useState("");
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    GetFeishuConfig()
      .then((c) => { setAppID(c.appId ?? ""); setSecretMasked(c.appSecretMasked ?? ""); })
      .catch(() => {});
  }, []);

  const configured = appID !== "" && secretMasked !== "";

  async function handleSave() {
    if (!appID.trim() || !appSecret.trim()) { setError("App ID 和 App Secret 不能为空"); return; }
    setSaving(true); setError("");
    try {
      await SaveFeishuConfig(appID.trim(), appSecret.trim());
      setSecretMasked("••••••••"); setAppSecret(""); setEditing(false);
    } catch (e: unknown) { setError(e instanceof Error ? e.message : String(e)); }
    finally { setSaving(false); }
  }

  return (
    <Card title="飞书"
      badge={configured
        ? <Badge label="已配置" cls="bg-[rgba(52,199,89,0.15)] text-emerald-400" />
        : <Badge label="未配置" cls="bg-[rgba(255,255,255,0.08)] text-[rgba(255,255,255,0.4)]" />}
      action={configured && !editing
        ? <button onClick={() => setEditing(true)} className="text-[11px] text-[#2688f9] hover:text-[#5aa3fb]">修改</button>
        : null}>
      {(!configured || editing) ? (
        <div className="space-y-3 pt-1">
          <p className="text-[11px] text-[rgba(255,255,255,0.3)]">企业自建应用，开启消息事件订阅（长连接）。</p>
          <div>
            <label className="block text-[11px] text-[rgba(255,255,255,0.5)] mb-1">App ID</label>
            <input type="text" placeholder="cli_xxxxxxxxxxxx" value={appID} onChange={(e) => setAppID(e.target.value)}
              className="w-full px-3 py-2 text-[13px] border border-[rgba(255,255,255,0.1)] rounded-xl focus:outline-none focus:ring-2 focus:ring-[rgba(38,136,249,0.4)] bg-[rgba(255,255,255,0.05)] text-[rgb(240,237,229)] placeholder-[rgba(255,255,255,0.2)]" />
          </div>
          <div>
            <label className="block text-[11px] text-[rgba(255,255,255,0.5)] mb-1">App Secret</label>
            <input type="password" placeholder={secretMasked || "App Secret"} value={appSecret} onChange={(e) => setAppSecret(e.target.value)}
              className="w-full px-3 py-2 text-[13px] border border-[rgba(255,255,255,0.1)] rounded-xl focus:outline-none focus:ring-2 focus:ring-[rgba(38,136,249,0.4)] bg-[rgba(255,255,255,0.05)] text-[rgb(240,237,229)] placeholder-[rgba(255,255,255,0.2)]" />
          </div>
          {error && <p className="text-[11px] text-red-400">{error}</p>}
          <div className="flex gap-2">
            <button onClick={handleSave} disabled={saving}
              className="flex-1 py-2 rounded-xl bg-[#2688f9] text-white text-[13px] font-semibold hover:bg-[#1a7ae8] disabled:opacity-50">
              {saving ? "保存中…" : "保存并连接"}
            </button>
            {editing && (
              <button onClick={() => { setEditing(false); setAppSecret(""); setError(""); }}
                className="px-3 py-2 rounded-xl border border-[rgba(255,255,255,0.1)] text-[13px] text-[rgba(255,255,255,0.5)] hover:bg-[rgba(255,255,255,0.05)]">
                取消
              </button>
            )}
          </div>
        </div>
      ) : (
        <div className="pt-1">
          <p className="text-[13px] text-[rgba(255,255,255,0.5)]">App ID: <span className="font-mono text-[rgb(240,237,229)] text-[11px]">{appID}</span></p>
          <p className="mt-1 text-[11px] text-[rgba(255,255,255,0.3)]">通过 WebSocket 长连接接收消息，无需公网地址。</p>
        </div>
      )}
    </Card>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// Primitives
// ─────────────────────────────────────────────────────────────────────────────
function GlassCard({ children, className = "" }: { children: React.ReactNode; className?: string }) {
  return (
    <div className={`bg-[rgba(255,255,255,0.05)] backdrop-blur-xl border border-[rgba(255,255,255,0.1)] rounded-2xl p-5 shadow-[inset_0_1px_0_rgba(255,255,255,0.06)] ${className}`}>
      {children}
    </div>
  );
}

function Card({ title, badge, action, children }: {
  title: string; badge?: React.ReactNode; action?: React.ReactNode; children: React.ReactNode;
}) {
  return (
    <GlassCard>
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <span className="font-semibold text-[rgb(240,237,229)] text-[13px]">{title}</span>
          {badge}
        </div>
        {action}
      </div>
      {children}
    </GlassCard>
  );
}

function Badge({ label, cls }: { label: string; cls: string }) {
  return <span className={`text-[11px] font-medium px-2 py-0.5 rounded-full ${cls}`}>{label}</span>;
}
