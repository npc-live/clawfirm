import React, { useEffect, useRef, useState } from "react";
import { DashedSelect } from "./DashedSelect";
import {
  GetChannels, GetChatSessions, GetHistory,
  GetConfig, SaveConfig,
  GetProviders,
  SaveChannelConfig, DeleteChannelConfig,
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
  DisableRemote, EnableNgrok, GetRemoteStatus,
} from "../lib/wails-shim";
import type { ChannelInfo, HistoryMessage, SkillInfo, RemoteSkillInfo, CronJob, CronJobHistory, Config, AgentConfig, ProviderInfo, VaultEntry, BrowserStatus, ShortcutInfo, RemoteStatus } from "../lib/wails-shim";
import { EventsOn } from "../wailsjs/runtime/runtime";
import { MemoryPane } from "./MemoryPane";
import { ProvidersPane } from "./ProvidersPane";
import { CanvasPane } from "./CanvasPane";


interface Props {
  onOpenChat: (agentName: string, sessionID: string) => void;
}
interface SessionPreview { id: string; preview: string; lastMs: number; }
type NavTab = "chats" | "canvas" | "skills" | "cron" | "memory" | "agents" | "channels" | "whipflow" | "vault" | "browser" | "settings";

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
    <div className="flex h-full bg-[#f0ece3]">
      {/* Sidebar */}
      <aside className="w-52 flex-shrink-0 bg-[#e8e4db] border-r border-dashed border-[rgba(30,28,23,0.15)] flex flex-col h-full">
        <div className="px-5 pt-6 pb-4 border-b border-dashed border-[rgba(30,28,23,0.1)]">
          <h1 className="text-[12px] font-bold text-[#1e1c17] tracking-widest uppercase">// Clawfirm</h1>
          <p className="text-[11px] text-[rgba(30,28,23,0.4)] mt-1 tracking-wider uppercase">AI Gateway</p>
        </div>
        <nav className="flex-1 px-3 py-3 space-y-0.5">
          <SidebarItem icon=">" label="chats" active={nav === "chats"} onClick={() => setNav("chats")} />
          <SidebarItem icon=">" label="canvas" active={nav === "canvas"} onClick={() => setNav("canvas")} />
          <SidebarItem icon=">" label="skills" active={nav === "skills"} onClick={() => { setNav("skills"); setSkillsKey(k => k + 1); }} />
          <SidebarItem icon=">" label="cron" active={nav === "cron"} onClick={() => setNav("cron")} />
          <SidebarItem icon=">" label="memory" active={nav === "memory"} onClick={() => setNav("memory")} />
          <SidebarItem icon=">" label="agents" active={nav === "agents"} onClick={() => setNav("agents")} />
          <SidebarItem icon=">" label="channels" active={nav === "channels"} onClick={() => setNav("channels")} />
          <SidebarItem icon=">" label="vault" active={nav === "vault"} onClick={() => setNav("vault")} />
          <SidebarItem icon=">" label="whipflow" active={nav === "whipflow"} onClick={() => setNav("whipflow")} />
          <SidebarItem icon=">" label="browser" active={nav === "browser"} onClick={() => setNav("browser")} />
          <SidebarItem icon=">" label="settings" active={nav === "settings"} onClick={() => setNav("settings")} />
        </nav>
        <div className="px-5 py-4 border-t border-dashed border-[rgba(30,28,23,0.1)]">
          <p className="text-[11px] text-[rgba(30,28,23,0.4)] font-mono uppercase tracking-wider">// {channels.length} agent{channels.length !== 1 ? "s" : ""} online</p>
        </div>
      </aside>

      {/* Content */}
      <main className={`flex-1 min-h-0 bg-[#f0ece3] ${nav === "canvas" || nav === "whipflow" ? "overflow-hidden" : "overflow-y-auto"}`}>
        {nav === "chats" && <ChatsPane sessionMap={sessionMap} channels={channels} onOpenChat={onOpenChat} />}
        {nav === "canvas" && <div className="w-full h-full"><CanvasPane /></div>}
        {nav === "skills" && <SkillsPane key={skillsKey} />}
        {nav === "cron" && <CronJobsPane />}
        {nav === "memory" && <MemoryPane />}
        {nav === "agents" && <AgentsPane onAgentsChanged={loadChannels} />}
        {nav === "channels" && <ChannelsPane />}
        {nav === "vault" && <VaultPane />}
        {nav === "whipflow" && <WhipflowPane />}
        {nav === "browser" && <BrowserPane />}
        {nav === "settings" && <SettingsPane key={Date.now()} />}
      </main>
    </div>
  );
}

function SidebarItem({ icon: _icon, label, active, onClick }: { icon: string; label: string; active: boolean; onClick: () => void }) {
  return (
    <button onClick={onClick}
      className={`w-full flex items-center gap-3 px-4 py-2 text-[12px] transition-colors text-left tracking-wider uppercase font-mono ${
        active
          ? "bg-[rgba(30,28,23,0.08)] text-[#1e1c17] border-l-2 border-[#1e1c17]"
          : "text-[rgba(30,28,23,0.4)] hover:bg-[rgba(30,28,23,0.05)] hover:text-[#1e1c17]"
      }`}>
      <span className={`text-[14px] leading-none font-bold ${active ? "text-[#1e1c17]" : "text-[rgba(30,28,23,0.35)]"}`}>&gt;</span>
      {label}
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
      <header className="mb-5 border-b border-dashed border-[rgba(30,28,23,0.1)] pb-4">
        <h2 className="text-[11px] font-bold text-[#1e1c17] tracking-widest uppercase">
          // skills
          <span className="ml-2 text-[10px] font-normal text-[rgba(30,28,23,0.4)] font-mono">
            {loading ? "[loading]" : `[${skills.length} loaded]`}
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
            className="flex-1 px-3 py-2 rounded-sm bg-[rgba(30,28,23,0.08)] border border-[rgba(30,28,23,0.12)]
              text-[13px] text-[#1e1c17] placeholder-[rgba(30,28,23,0.2)]
              focus:outline-none focus:border-[rgba(30,28,23,0.2)] transition-colors"
          />
          <button
            onClick={doSearch}
            disabled={searching || !searchQuery.trim()}
            className="px-4 py-2 rounded-sm bg-[rgba(30,28,23,0.08)] border border-[rgba(30,28,23,0.12)]
              text-[13px] text-[#1e1c17] hover:bg-[rgba(30,28,23,0.1)] transition-colors
              disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {searching ? "Searching…" : "Search"}
          </button>
        </div>

        {installMsg && (
          <p className={`mt-2 text-[12px] ${installMsg.startsWith("Error") ? "text-red-400" : "text-green-400"}`}>{installMsg}</p>
        )}

        {searchDone && searchResults.length === 0 && (
          <p className="mt-3 text-[13px] text-[rgba(30,28,23,0.5)] italic">No remote skills found.</p>
        )}
        {searchResults.length > 0 && (
          <div className="mt-3 space-y-2">
            {searchResults.map((r) => (
              <div key={r.name} className="flex items-center gap-3 px-4 py-3 rounded-sm bg-[rgba(30,28,23,0.04)] border border-[rgba(30,28,23,0.1)]">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="text-[13px] font-medium text-[#1e1c17]">{r.name}</p>
                    <span className="text-[11px] text-[rgba(30,28,23,0.35)] font-mono">v{r.version}</span>
                    {r.author && <span className="text-[11px] text-[rgba(30,28,23,0.2)]">by {r.author}</span>}
                  </div>
                  {r.description && (
                    <p className="text-[12px] text-[rgba(30,28,23,0.5)] mt-0.5 line-clamp-2">{r.description}</p>
                  )}
                </div>
                <button
                  onClick={() => doInstall(r.name)}
                  disabled={installing !== null}
                  className="flex-shrink-0 px-3 py-1.5 rounded-md bg-[rgba(200,90,42,0.15)] border border-[rgba(200,90,42,0.25)]
                    text-[12px] text-[rgba(200,90,42,0.9)] hover:bg-[rgba(200,90,42,0.25)] transition-colors
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
        <p className="text-[13px] text-[rgba(30,28,23,0.5)] italic">No skills found.</p>
      )}
      <div className="space-y-2 max-w-2xl">
        {skills.map((s) => {
          const isOpen = expanded === s.filePath;
          const content = contentMap[s.filePath] ?? "";
          const urls = isOpen ? extractUrls(content) : [];
          return (
            <div key={s.filePath} className="rounded-sm bg-[rgba(30,28,23,0.04)] border border-[rgba(30,28,23,0.1)] overflow-hidden">
              <button className="w-full text-left px-4 py-3 hover:bg-[rgba(30,28,23,0.03)] transition-colors"
                onClick={() => toggle(s.filePath)}>
                <div className="flex items-center justify-between gap-2">
                  <p className="text-[13px] font-medium text-[#1e1c17]">{s.name}</p>
                  <span className="text-[rgba(30,28,23,0.4)] text-[11px] flex-shrink-0">{isOpen ? "▲" : "▼"}</span>
                </div>
                {s.description && (
                  <p className="text-[12px] text-[rgba(30,28,23,0.5)] mt-0.5 line-clamp-2">{s.description}</p>
                )}
                <p className="text-[11px] text-[rgba(30,28,23,0.3)] font-mono mt-1">{s.filePath}</p>
              </button>

              {isOpen && (
                <div className="border-t border-[rgba(30,28,23,0.08)] px-4 py-3 space-y-3">
                  {/* URLs */}
                  {content === "" ? (
                    <p className="text-[12px] text-[rgba(30,28,23,0.4)]">Loading…</p>
                  ) : urls.length > 0 ? (
                    <div>
                      <p className="text-[11px] font-semibold text-[rgba(30,28,23,0.5)] uppercase tracking-wider mb-2">URLs</p>
                      <div className="space-y-1">
                        {urls.map((u) => (
                          <p key={u} className="text-[12px] font-mono text-[rgba(200,90,42,0.8)] break-all">{u}</p>
                        ))}
                      </div>
                    </div>
                  ) : null}
                  {/* Raw content */}
                  <div>
                    <p className="text-[11px] font-semibold text-[rgba(30,28,23,0.5)] uppercase tracking-wider mb-2">Content</p>
                    <pre className="text-[11px] text-[rgba(30,28,23,0.5)] whitespace-pre-wrap break-words max-h-64 overflow-y-auto font-mono leading-relaxed">
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

  const inputCls = "w-full px-3 py-2 text-[13px] border border-[rgba(30,28,23,0.12)] rounded focus:outline-none focus:ring-2 focus:ring-[rgba(200,90,42,0.4)] bg-[rgba(30,28,23,0.05)] text-[#1e1c17] placeholder-[rgba(30,28,23,0.3)]";

  return (
    <div className="p-8 h-full flex flex-col">
      <header className="mb-5 flex-shrink-0 flex items-center justify-between border-b border-dashed border-[rgba(30,28,23,0.1)] pb-4">
        <div>
          <h2 className="text-[11px] font-bold text-[#1e1c17] tracking-widest uppercase">
            // cron jobs
            <span className="ml-2 text-[10px] font-normal text-[rgba(30,28,23,0.4)] font-mono">
              {loading ? "[loading]" : `[${jobs.length}]`}
            </span>
          </h2>
          <p className="text-[10px] text-[rgba(30,28,23,0.4)] mt-0.5 tracking-wider uppercase">schedule agents on a timer</p>
        </div>
        {editing === null && (
          <button onClick={startNew}
            className="text-[10px] text-[#1e1c17] px-3 py-1.5 border border-dashed border-[rgba(30,28,23,0.3)] hover:bg-[rgba(30,28,23,0.06)] transition-colors font-mono uppercase tracking-wider">
            [+ new job]
          </button>
        )}
      </header>

      {/* Edit / Create form */}
      {editing !== null && (
        <GlassCard className="mb-5 flex-shrink-0 max-w-2xl">
          <h3 className="text-[13px] font-semibold text-[#1e1c17] mb-4">
            {editing === "new" ? "New Cron Job" : "Edit Cron Job"}
          </h3>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-[11px] text-[rgba(30,28,23,0.5)] mb-1">Name</label>
              <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="daily-report" className={inputCls} />
            </div>
            <div>
              <label className="block text-[11px] text-[rgba(30,28,23,0.5)] mb-1">Agent</label>
              <DashedSelect value={form.agentName} onChange={v => setForm({ ...form, agentName: v })} options={agentNames.map(n => ({ value: n, label: n }))} placeholder="— select agent —" />
            </div>
          </div>
          <div className="mt-3">
            <label className="block text-[11px] text-[rgba(30,28,23,0.5)] mb-1">Schedule Kind</label>
            <div className="flex gap-2">
              {(["cron", "every", "at"] as const).map((k) => (
                <button key={k} onClick={() => setForm({ ...form, scheduleKind: k })}
                  className={`px-3 py-1.5 rounded-sm text-[12px] font-medium transition-colors ${
                    form.scheduleKind === k
                      ? "bg-[rgba(200,90,42,0.2)] text-[#c85a2a] border border-[rgba(200,90,42,0.3)]"
                      : "bg-[rgba(30,28,23,0.05)] text-[rgba(30,28,23,0.5)] border border-[rgba(30,28,23,0.08)] hover:bg-[rgba(30,28,23,0.08)]"
                  }`}>
                  {k}
                </button>
              ))}
            </div>
          </div>
          {form.scheduleKind === "cron" && (
            <div className="grid grid-cols-2 gap-3 mt-3">
              <div>
                <label className="block text-[11px] text-[rgba(30,28,23,0.5)] mb-1">Cron Expression</label>
                <input value={form.expr} onChange={(e) => setForm({ ...form, expr: e.target.value })} placeholder="0 9 * * MON-FRI" className={inputCls} />
              </div>
              <div>
                <label className="block text-[11px] text-[rgba(30,28,23,0.5)] mb-1">Timezone (optional)</label>
                <input value={form.tz} onChange={(e) => setForm({ ...form, tz: e.target.value })} placeholder="Asia/Shanghai" className={inputCls} />
              </div>
            </div>
          )}
          {form.scheduleKind === "every" && (
            <div className="grid grid-cols-[1fr_auto] gap-2 mt-3">
              <div>
                <label className="block text-[11px] text-[rgba(30,28,23,0.5)] mb-1">Interval</label>
                <input type="number" min="1" value={form.everyVal} onChange={(e) => setForm({ ...form, everyVal: e.target.value })} placeholder="30" className={inputCls} />
              </div>
              <div>
                <label className="block text-[11px] text-[rgba(30,28,23,0.5)] mb-1">Unit</label>
                <DashedSelect value={form.everyUnit} onChange={v => setForm({ ...form, everyUnit: v as EveryUnit })} options={[
                  { value: "seconds", label: "Seconds" },
                  { value: "minutes", label: "Minutes" },
                  { value: "hours", label: "Hours" },
                  { value: "days", label: "Days" },
                ]} />
              </div>
            </div>
          )}
          {form.scheduleKind === "at" && (
            <div className="mt-3">
              <label className="block text-[11px] text-[rgba(30,28,23,0.5)] mb-1">Run At</label>
              <input type="datetime-local" value={form.at} onChange={(e) => setForm({ ...form, at: e.target.value })}
                className={`${inputCls} [color-scheme:dark]`} />
            </div>
          )}
          <div className="mt-3">
            <label className="block text-[11px] text-[rgba(30,28,23,0.5)] mb-1">Prompt</label>
            <textarea value={form.prompt} onChange={(e) => setForm({ ...form, prompt: e.target.value })} rows={3} placeholder="Generate a daily report..."
              className={`${inputCls} resize-none leading-relaxed`} />
          </div>
          <div className="mt-3 flex items-center gap-2">
            <button onClick={() => setForm({ ...form, enabled: !form.enabled })}
              className={`w-9 h-5 rounded-full transition-colors relative ${form.enabled ? "bg-[#c85a2a]" : "bg-[rgba(30,28,23,0.15)]"}`}>
              <span className={`absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform ${form.enabled ? "left-[18px]" : "left-0.5"}`} />
            </button>
            <span className="text-[12px] text-[rgba(30,28,23,0.5)]">{form.enabled ? "Enabled" : "Disabled"}</span>
          </div>
          {error && <p className="text-[11px] text-red-400 mt-2">{error}</p>}
          <div className="flex gap-2 mt-4">
            <button onClick={handleSave} disabled={saving}
              className="px-5 py-2 bg-[#c85a2a] text-white text-[13px] rounded hover:bg-[#a84a22] disabled:opacity-50 transition-colors font-semibold">
              {saving ? "Saving…" : "Save"}
            </button>
            <button onClick={cancelEdit}
              className="px-4 py-2 rounded border border-[rgba(30,28,23,0.12)] text-[13px] text-[rgba(30,28,23,0.5)] hover:bg-[rgba(30,28,23,0.05)]">
              Cancel
            </button>
          </div>
        </GlassCard>
      )}

      {/* Job list */}
      {!loading && jobs.length === 0 && editing === null && (
        <div className="text-center py-24 text-[rgba(30,28,23,0.4)]">
          <div className="text-4xl mb-3">🕐</div>
          <p className="text-[13px]">No cron jobs yet. Click "+ New Job" to create one.</p>
        </div>
      )}
      <div className="space-y-2 max-w-2xl flex-1 min-h-0 overflow-y-auto">
        {jobs.map((j) => (
          <div key={j.id} className="rounded-sm bg-[rgba(30,28,23,0.04)] border border-[rgba(30,28,23,0.1)] overflow-hidden">
            <div className="px-4 py-3 flex items-center justify-between gap-3">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <p className="text-[13px] font-medium text-[#1e1c17] truncate">{j.name}</p>
                  <span className={`text-[10px] font-mono px-1.5 py-0.5 rounded ${
                    j.scheduleKind === "cron" ? "bg-[rgba(200,90,42,0.15)] text-[#c85a2a]"
                    : j.scheduleKind === "every" ? "bg-[rgba(52,199,89,0.15)] text-emerald-400"
                    : "bg-[rgba(255,179,64,0.15)] text-amber-400"
                  }`}>
                    {j.scheduleKind}
                  </span>
                  <span className={`text-[10px] font-medium px-1.5 py-0.5 rounded-full ${
                    j.enabled ? "bg-[rgba(52,199,89,0.15)] text-emerald-400" : "bg-[rgba(30,28,23,0.08)] text-[rgba(30,28,23,0.35)]"
                  }`}>
                    {j.enabled ? "ON" : "OFF"}
                  </span>
                </div>
                <p className="text-[11px] text-[rgba(30,28,23,0.35)] mt-0.5 truncate">
                  {j.scheduleKind === "cron" && `${j.schedule.expr}${j.schedule.tz ? ` (${j.schedule.tz})` : ""}`}
                  {j.scheduleKind === "every" && j.schedule.everyMs && (() => { const h = msToHuman(j.schedule.everyMs!); return `every ${h.val} ${h.unit}`; })()}
                  {j.scheduleKind === "at" && (j.schedule.at ? new Date(j.schedule.at).toLocaleString() : "")}
                  {" · agent: "}{j.agentName}
                </p>
                <p className="text-[11px] text-[rgba(30,28,23,0.2)] mt-0.5 truncate">{j.prompt}</p>
              </div>
              <div className="flex items-center gap-2 flex-shrink-0">
                <button onClick={() => handleToggle(j.id, !j.enabled)}
                  className={`w-8 h-[18px] rounded-full transition-colors relative ${j.enabled ? "bg-[#c85a2a]" : "bg-[rgba(30,28,23,0.15)]"}`}>
                  <span className={`absolute top-[2px] w-[14px] h-[14px] rounded-full bg-white transition-transform ${j.enabled ? "left-[15px]" : "left-[2px]"}`} />
                </button>
                <button onClick={() => handleTrigger(j.id)}
                  disabled={triggering === j.id}
                  className="text-[11px] font-medium text-emerald-400 hover:text-emerald-300 disabled:opacity-50 transition-colors">
                  {triggering === j.id ? "Running…" : "▶ Run"}
                </button>
                <button onClick={() => showHistory(j.id)}
                  className="text-[11px] text-[rgba(30,28,23,0.5)] hover:text-[#1e1c17] transition-colors">
                  History
                </button>
                <button onClick={() => startEdit(j)}
                  className="text-[11px] text-[#c85a2a] hover:text-[#5aa3fb] transition-colors">
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
    <div className="border-t border-[rgba(30,28,23,0.08)] px-4 py-3">
      <div className="flex items-center justify-between mb-2">
        <p className="text-[11px] font-semibold text-[rgba(30,28,23,0.5)] uppercase tracking-wider">
          Execution History
        </p>
        <button onClick={load} className="text-[10px] text-[rgba(30,28,23,0.4)] hover:text-[rgba(30,28,23,0.7)] transition-colors">
          ↻ refresh
        </button>
      </div>

      {loading ? (
        <p className="text-[11px] text-[rgba(30,28,23,0.4)]">Loading…</p>
      ) : entries.length === 0 ? (
        <p className="text-[11px] text-[rgba(30,28,23,0.4)] italic">No executions yet.</p>
      ) : (
        <div className="space-y-1.5 max-h-[500px] overflow-y-auto pr-1">
          {entries.map((h) => {
            const isExpanded = expanded === h.id;
            const log = h.resultText || h.errorText || "";
            return (
              <div key={h.id}
                className="rounded-sm border border-[rgba(30,28,23,0.1)] overflow-hidden bg-[rgba(30,28,23,0.02)]">
                {/* Header row */}
                <button
                  onClick={() => setExpanded(isExpanded ? null : h.id)}
                  className="w-full flex items-center gap-2.5 px-3 py-2 text-left hover:bg-[rgba(30,28,23,0.03)] transition-colors">
                  <span className={`w-2 h-2 rounded-full flex-shrink-0 ${
                    h.status === "success" ? "bg-emerald-400" :
                    h.status === "error"   ? "bg-red-400" :
                    "bg-[#c85a2a] animate-pulse"
                  }`} />
                  <span className="text-[11px] font-mono text-[rgba(30,28,23,0.5)] flex-1 truncate">
                    {fmtTime(h.startedAt)}
                  </span>
                  <span className={`text-[10px] font-medium px-1.5 py-0.5 rounded ${
                    h.status === "success" ? "text-emerald-400 bg-[rgba(52,199,89,0.1)]" :
                    h.status === "error"   ? "text-red-400 bg-[rgba(255,59,48,0.1)]" :
                    "text-[#c85a2a] bg-[rgba(200,90,42,0.1)]"
                  }`}>{h.status}</span>
                  <span className="text-[10px] text-[rgba(30,28,23,0.2)] tabular-nums w-12 text-right">
                    {duration(h)}
                  </span>
                  <span className="text-[10px] text-[rgba(30,28,23,0.3)]">{isExpanded ? "▲" : "▼"}</span>
                </button>

                {/* Expanded log */}
                {isExpanded && (
                  <div className="border-t border-[rgba(30,28,23,0.05)]">
                    {log ? (
                      <pre className="px-3 py-2.5 text-[11px] font-mono leading-relaxed
                        text-[rgba(30,28,23,0.65)] whitespace-pre-wrap break-words
                        max-h-64 overflow-y-auto">
                        {log}
                      </pre>
                    ) : (
                      <p className="px-3 py-2 text-[11px] text-[rgba(30,28,23,0.2)] italic">No output.</p>
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
// New Chat dropdown button
// ─────────────────────────────────────────────────────────────────────────────

function NewChatButton({ channels, onOpenChat }: {
  channels: ChannelInfo[];
  onOpenChat: (agentName: string, sessionID: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  if (channels.length <= 1) {
    return (
      <button
        onClick={() => {
          const name = channels[0]?.name;
          if (name) onOpenChat(name, "s" + Date.now());
        }}
        className="px-3 py-1.5 text-[10px] text-[#1e1c17] border border-dashed border-[rgba(30,28,23,0.3)] hover:bg-[rgba(30,28,23,0.06)] transition-colors font-mono uppercase tracking-wider"
      >
        [+ new chat]
      </button>
    );
  }

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen((v) => !v)}
        className="px-3 py-1.5 text-[10px] text-[#1e1c17] border border-dashed border-[rgba(30,28,23,0.3)] hover:bg-[rgba(30,28,23,0.06)] transition-colors font-mono uppercase tracking-wider flex items-center gap-1.5"
      >
        [+ new chat]
        <span className={`transition-transform inline-block ${open ? "rotate-180" : ""}`}>v</span>
      </button>
      {open && (
        <div className="absolute right-0 top-full mt-1 min-w-[180px] bg-[#e8e4db] border border-dashed border-[rgba(30,28,23,0.2)] overflow-hidden z-50">
          {channels.map((c) => (
            <button
              key={c.name}
              onClick={() => {
                setOpen(false);
                onOpenChat(c.name, "s" + Date.now());
              }}
              className="w-full text-left px-3 py-2 text-[11px] text-[#1e1c17] hover:bg-[rgba(30,28,23,0.06)] transition-colors flex items-center gap-2 font-mono border-b border-dashed border-[rgba(30,28,23,0.08)] last:border-0"
            >
              <span className="text-[rgba(30,28,23,0.3)]">&gt;</span>
              <span className="flex-1 uppercase tracking-wider text-[10px]">{c.name}</span>
              <span className="text-[9px] text-[rgba(30,28,23,0.3)]">{c.model}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

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
    <div className="p-6">
      <header className="mb-5 flex items-center justify-between border-b border-dashed border-[rgba(30,28,23,0.1)] pb-4">
        <div>
          <h2 className="text-[11px] font-bold text-[#1e1c17] tracking-widest uppercase">// chats</h2>
          <p className="text-[10px] text-[rgba(30,28,23,0.4)] mt-0.5 tracking-wider uppercase">session history</p>
        </div>
        <NewChatButton channels={channels} onOpenChat={onOpenChat} />
      </header>

      {allSessions.length === 0 ? (
        <p className="text-[11px] text-[rgba(30,28,23,0.4)] font-mono">// no sessions found</p>
      ) : (
        <div className="space-y-1.5 max-w-2xl">
          {allSessions.map(({ agentName, session }) => (
            <button
              key={`${agentName}/${session.id}`}
              onClick={() => onOpenChat(agentName, session.id)}
              className="w-full text-left bg-[rgba(30,28,23,0.03)] border border-dashed border-[rgba(30,28,23,0.1)] px-3 py-2.5 hover:bg-[rgba(30,28,23,0.07)] hover:border-[rgba(30,28,23,0.2)] transition-all group"
            >
              <div className="flex items-center justify-between mb-0.5">
                <span className="text-[10px] font-mono text-[rgba(30,28,23,0.5)] uppercase tracking-wider">&gt; {agentName}</span>
                <span className="text-[10px] text-[rgba(30,28,23,0.25)] font-mono">{formatTime(session.lastMs)}</span>
              </div>
              <p className="text-[11px] text-[rgba(30,28,23,0.65)] truncate leading-snug group-hover:text-[rgba(30,28,23,0.9)] font-mono">
                {session.preview}
              </p>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// Media Tools Pane
// ─────────────────────────────────────────────────────────────────────────────

interface ToolFormEntry { provider: string; protocol: string; model: string; }

const MEDIA_TOOLS: { name: string; label: string; desc: string }[] = [
  { name: "media_gen",        label: "Media Gen",        desc: "Generate images via AI image models (Gemini, DALL-E, etc.)" },
  { name: "media_understand", label: "Media Understand",  desc: "Analyze images / video frames using a vision LLM" },
];

const TOOL_PROTOCOLS = ["google", "openai", "openai-chat", "anthropic", "gemini", "ollama"];

// DashedSelect imported from ./DashedSelect

function MediaToolsPane() {
  const [cfg, setCfg] = useState<Config | null>(null);
  const [editing, setEditing] = useState<string | null>(null);
  const [form, setForm] = useState<ToolFormEntry>({ provider: "", protocol: "", model: "" });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => { GetConfig().then(c => setCfg(c)).catch(() => {}); }, []);

  const providerIds = Object.keys(cfg?.providers ?? {});

  function startEdit(name: string) {
    const existing = cfg?.tools?.[name];
    setForm({ provider: existing?.provider ?? "", protocol: existing?.protocol ?? "", model: existing?.model ?? "" });
    setEditing(name);
    setError("");
  }

  async function handleSave() {
    if (!cfg) return;
    setSaving(true);
    setError("");
    try {
      const prev = cfg.tools?.[editing!];
      const tools = { ...(cfg.tools ?? {}), [editing!]: { provider: form.provider, protocol: form.protocol, model: form.model, api_key: prev?.api_key ?? "" } };
      const newCfg: Config = { ...cfg, tools };
      await SaveConfig(newCfg);
      setCfg(newCfg);
      setEditing(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  }

  const inputCls = "w-full px-2 py-1.5 text-[11px] font-mono border border-dashed border-[rgba(30,28,23,0.25)] focus:outline-none focus:border-[rgba(30,28,23,0.5)] bg-[rgba(30,28,23,0.03)] text-[#1e1c17] placeholder-[rgba(30,28,23,0.25)]";

  const providerOptions = [{ value: "", label: "-- select provider --" }, ...providerIds.map(id => ({ value: id, label: id }))];
  const protocolOptions = [{ value: "", label: "auto (infer from provider)" }, ...TOOL_PROTOCOLS.map(p => ({ value: p, label: p }))];

  return (
    <div className="max-w-2xl">
      <header className="mb-4 border-b border-dashed border-[rgba(30,28,23,0.3)] pb-4">
        <h2 className="text-[11px] font-bold text-[#1e1c17] tracking-widest uppercase">// tools</h2>
        <p className="text-[10px] text-[rgba(30,28,23,0.3)] mt-0.5 font-mono">// media generation &amp; understanding</p>
      </header>

      {error && (
        <div className="mb-3 px-3 py-2 border border-dashed border-[rgba(200,50,30,0.4)] text-[10px] font-mono text-red-400">
          // err: {error}
        </div>
      )}

      <div className="space-y-2">
        {MEDIA_TOOLS.map(({ name, label, desc }) => {
          const tool = cfg?.tools?.[name];
          const isEditing = editing === name;
          return (
            <div key={name} className="border border-dashed border-[rgba(30,28,23,0.2)] bg-[rgba(30,28,23,0.02)]">
              <div className="flex items-start justify-between px-3 py-2.5">
                <div className="flex-1 min-w-0">
                  <span className="text-[12px] font-bold text-[#1e1c17] font-mono">{label}</span>
                  <p className="text-[10px] text-[rgba(30,28,23,0.4)] mt-0.5 font-mono">{desc}</p>
                  {!isEditing && tool && (
                    <div className="mt-1 flex flex-wrap gap-x-3 gap-y-0.5">
                      {tool.provider && <span className="text-[10px] font-mono text-[rgba(30,28,23,0.4)]">provider: {tool.provider}</span>}
                      {tool.protocol && <span className="text-[10px] font-mono text-[rgba(30,28,23,0.4)]">protocol: {tool.protocol}</span>}
                      {tool.model    && <span className="text-[10px] font-mono text-[rgba(30,28,23,0.4)]">model: {tool.model}</span>}
                    </div>
                  )}
                </div>
                {!isEditing && (
                  <button
                    onClick={() => startEdit(name)}
                    className="ml-3 flex-shrink-0 text-[10px] px-2 py-1 border border-dashed border-[rgba(30,28,23,0.2)] text-[rgba(30,28,23,0.5)] hover:text-[#1e1c17] hover:border-[rgba(30,28,23,0.4)] transition-colors font-mono uppercase tracking-wider"
                  >
                    [edit]
                  </button>
                )}
              </div>

              {isEditing && (
                <div className="px-3 pb-3 border-t border-dashed border-[rgba(30,28,23,0.12)] pt-2.5 space-y-2">
                  <p className="text-[10px] font-mono text-[rgba(30,28,23,0.4)] uppercase tracking-wider">// edit: {name}</p>
                  <div>
                    <label className="block text-[10px] font-mono text-[rgba(30,28,23,0.4)] uppercase tracking-wider mb-1">Provider</label>
                    <DashedSelect value={form.provider} onChange={v => setForm(f => ({ ...f, provider: v }))} options={providerOptions} placeholder="-- select provider --" />
                  </div>
                  <div>
                    <label className="block text-[10px] font-mono text-[rgba(30,28,23,0.4)] uppercase tracking-wider mb-1">Protocol</label>
                    <DashedSelect value={form.protocol} onChange={v => setForm(f => ({ ...f, protocol: v }))} options={protocolOptions} placeholder="auto (infer from provider)" />
                  </div>
                  <div>
                    <label className="block text-[10px] font-mono text-[rgba(30,28,23,0.4)] uppercase tracking-wider mb-1">Model</label>
                    <input className={inputCls} placeholder="leave empty to use default" value={form.model} onChange={e => setForm(f => ({ ...f, model: e.target.value }))} />
                  </div>
                  <div className="flex gap-2 pt-1">
                    <button onClick={handleSave} disabled={saving} className="text-[10px] px-3 py-1 border border-dashed border-[rgba(30,28,23,0.4)] text-[#1e1c17] hover:bg-[rgba(30,28,23,0.06)] transition-colors font-mono uppercase tracking-wider disabled:opacity-50">
                      {saving ? "saving…" : "[save]"}
                    </button>
                    <button onClick={() => setEditing(null)} className="text-[10px] px-3 py-1 border border-dashed border-[rgba(30,28,23,0.2)] text-[rgba(30,28,23,0.45)] hover:text-[#1e1c17] transition-colors font-mono uppercase tracking-wider">
                      [cancel]
                    </button>
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

function AgentsPane({ onAgentsChanged }: { onAgentsChanged?: () => void }) {
  return (
    <div className="p-6">
      <header className="mb-5 border-b border-dashed border-[rgba(30,28,23,0.2)] pb-4">
        <h2 className="text-[11px] font-bold text-[#1e1c17] tracking-widest uppercase">// agents</h2>
        <p className="text-[10px] text-[rgba(30,28,23,0.4)] mt-0.5 tracking-wider uppercase">independent AI assistants</p>
      </header>
      <AgentsEditor onAgentsChanged={onAgentsChanged} />

      <div className="mt-10 pt-6 border-t border-dashed border-[rgba(30,28,23,0.2)]">
        <ProvidersPane />
      </div>

      <div className="mt-10 pt-6 border-t border-dashed border-[rgba(30,28,23,0.2)]">
        <MediaToolsPane />
      </div>
    </div>
  );
}

function ChannelsPane() {
  return (
    <div className="p-6">
      <header className="mb-5 border-b border-dashed border-[rgba(30,28,23,0.1)] pb-4">
        <h2 className="text-[11px] font-bold text-[#1e1c17] tracking-widest uppercase">// channels</h2>
        <p className="text-[10px] text-[rgba(30,28,23,0.4)] mt-0.5 tracking-wider uppercase">messaging integrations</p>
      </header>
      <ChannelsSettings />
    </div>
  );
}

function AgentsEditor({ onAgentsChanged }: { onAgentsChanged?: () => void }) {
  const [cfg, setCfg] = useState<Config | null>(null);
  const [providers, setProviders] = useState<ProviderInfo[]>([]);
  const [editing, setEditing] = useState<string | null>(null);
  const [form, setForm] = useState<{ provider: string; model: string; system_prompt: string }>({ provider: "", model: "", system_prompt: "" });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState<string | null>(null);
  const [newOpen, setNewOpen] = useState(false);
  const [newForm, setNewForm] = useState({ name: "", provider: "", model: "", system_prompt: "" });
  const [newSaving, setNewSaving] = useState(false);
  const [newError, setNewError] = useState("");
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

  async function reload() {
    const c = await GetConfig().catch(() => null);
    if (c) setCfg(c);
    onAgentsChanged?.();
  }

  useEffect(() => {
    reload();
    GetProviders().then(p => setProviders(p ?? [])).catch(() => {});
  }, []);

  const providerIds = providers.map(p => p.id);

  function startEdit(agent: AgentConfig) {
    setForm({ provider: agent.provider ?? "", model: agent.model ?? "", system_prompt: agent.system_prompt ?? "" });
    setEditing(agent.name);
    setError("");
    setSuccess(null);
    setDeleteConfirm(null);
  }

  async function handleSave() {
    if (!cfg || editing === null) return;
    const agent = cfg.agents.find(a => a.name === editing);
    if (!agent) return;
    setSaving(true); setError("");
    try {
      await SaveChannelConfig({ ...agent, provider: form.provider, model: form.model, system_prompt: form.system_prompt });
      await reload();
      setEditing(null);
      setSuccess(editing);
      setTimeout(() => setSuccess(null), 2500);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  }

  async function handleCreate() {
    if (!newForm.name.trim()) { setNewError("Name is required"); return; }
    if (!newForm.provider) { setNewError("Provider is required"); return; }
    if (!newForm.model.trim()) { setNewError("Model is required"); return; }
    setNewSaving(true); setNewError("");
    try {
      await SaveChannelConfig({
        name: newForm.name.trim(),
        provider: newForm.provider,
        model: newForm.model.trim(),
        system_prompt: newForm.system_prompt,
        max_tokens: 0,
      });
      await reload();
      setNewOpen(false);
      setNewForm({ name: "", provider: "", model: "", system_prompt: "" });
    } catch (e) {
      setNewError(e instanceof Error ? e.message : String(e));
    } finally {
      setNewSaving(false);
    }
  }

  async function handleDelete(name: string) {
    try {
      await DeleteChannelConfig(name);
      await reload();
      setDeleteConfirm(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  const agents = cfg?.agents ?? [];

  return (
    <div className="space-y-3 max-w-2xl">
      {/* New agent button */}
      <div className="flex justify-end mb-2">
        <button
          onClick={() => { setNewOpen(o => !o); setNewError(""); }}
          className="px-3 py-1.5 text-[12px] bg-[#c85a2a] text-white rounded-sm hover:bg-[#a84a22] transition-colors font-semibold"
        >
          {newOpen ? "Cancel" : "+ New Agent"}
        </button>
      </div>

      {/* New agent form */}
      {newOpen && (
        <div className="bg-[rgba(30,28,23,0.04)] border border-dashed border-[rgba(30,28,23,0.3)] rounded-sm p-4 space-y-3">
          {newError && <p className="text-[12px] text-red-400">{newError}</p>}
          <div>
            <label className="block text-[11px] text-[rgba(30,28,23,0.5)] mb-1">Name <span className="text-red-400">*</span></label>
            <input className={inputCls} placeholder="e.g. assistant"
              value={newForm.name} onChange={e => setNewForm(f => ({ ...f, name: e.target.value }))} />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-[11px] text-[rgba(30,28,23,0.5)] mb-1">Provider <span className="text-red-400">*</span></label>
              <DashedSelect value={newForm.provider} onChange={v => setNewForm(f => ({ ...f, provider: v }))} options={providerIds.map(id => ({ value: id, label: id }))} placeholder="— select —" />
            </div>
            <div>
              <label className="block text-[11px] text-[rgba(30,28,23,0.5)] mb-1">Model <span className="text-red-400">*</span></label>
              <input className={inputCls} placeholder="e.g. claude-sonnet-4-6"
                value={newForm.model} onChange={e => setNewForm(f => ({ ...f, model: e.target.value }))} />
            </div>
          </div>
          <div>
            <label className="block text-[11px] text-[rgba(30,28,23,0.5)] mb-1">System Prompt</label>
            <textarea className={`${inputCls} resize-none`} rows={3}
              value={newForm.system_prompt} onChange={e => setNewForm(f => ({ ...f, system_prompt: e.target.value }))} />
          </div>
          <div className="flex gap-2 pt-1">
            <button onClick={handleCreate} disabled={newSaving}
              className="px-4 py-1.5 text-[12px] bg-[#c85a2a] text-white rounded-sm hover:bg-[#a84a22] disabled:opacity-50 transition-colors font-semibold">
              {newSaving ? "Creating…" : "Create Agent"}
            </button>
            <button onClick={() => { setNewOpen(false); setNewError(""); }}
              className="px-3 py-1.5 text-[12px] bg-[rgba(30,28,23,0.08)] text-[rgba(30,28,23,0.5)] rounded-sm hover:bg-[rgba(30,28,23,0.12)] transition-colors">
              Cancel
            </button>
          </div>
        </div>
      )}

      {agents.length === 0 && !newOpen && <p className="text-[13px] text-[rgba(30,28,23,0.4)]">No agents configured.</p>}

      {agents.map(agent => (
        <div key={agent.name} className="bg-[rgba(30,28,23,0.04)] border border-dashed border-[rgba(30,28,23,0.25)] rounded-sm">
          {/* Header row */}
          <div className="flex items-center justify-between px-4 py-3">
            <div className="flex items-center gap-2">
              <span className="text-[14px] font-semibold text-[#1e1c17]">{agent.name}</span>
              {success === agent.name && <span className="text-[11px] text-emerald-400">✓ saved</span>}
              {editing !== agent.name && (
                <>
                  <span className="px-2 py-0.5 text-[11px] bg-[rgba(200,90,42,0.15)] text-[#c85a2a] rounded-full">{agent.provider || "—"}</span>
                  {agent.model && <span className="text-[11px] text-[rgba(30,28,23,0.35)]">{agent.model}</span>}
                </>
              )}
            </div>
            {editing !== agent.name ? (
              <div className="flex gap-1.5 items-center">
                <button onClick={() => startEdit(agent)}
                  className="px-3 py-1.5 text-[12px] bg-[rgba(30,28,23,0.08)] text-[rgba(30,28,23,0.5)] rounded-sm hover:bg-[rgba(30,28,23,0.12)] hover:text-[rgba(30,28,23,0.8)] transition-colors">
                  Edit
                </button>
                {deleteConfirm === agent.name ? (
                  <>
                    <button onClick={() => handleDelete(agent.name)}
                      className="px-3 py-1.5 text-[12px] bg-red-500 text-white rounded-sm hover:bg-red-600 transition-colors font-semibold">
                      Confirm
                    </button>
                    <button onClick={() => setDeleteConfirm(null)}
                      className="px-3 py-1.5 text-[12px] bg-[rgba(30,28,23,0.08)] text-[rgba(30,28,23,0.5)] rounded-sm hover:bg-[rgba(30,28,23,0.12)] transition-colors">
                      Cancel
                    </button>
                  </>
                ) : (
                  <button onClick={() => setDeleteConfirm(agent.name)}
                    className="px-3 py-1.5 text-[12px] text-red-400 hover:text-red-500 transition-colors">
                    Delete
                  </button>
                )}
              </div>
            ) : (
              <div className="flex gap-1.5">
                <button onClick={handleSave} disabled={saving}
                  className="px-3 py-1.5 text-[12px] bg-[#c85a2a] text-white rounded-sm hover:bg-[#a84a22] disabled:opacity-50 transition-colors font-semibold">
                  {saving ? "…" : "Save"}
                </button>
                <button onClick={() => { setEditing(null); setError(""); }}
                  className="px-3 py-1.5 text-[12px] bg-[rgba(30,28,23,0.08)] text-[rgba(30,28,23,0.5)] rounded-sm hover:bg-[rgba(30,28,23,0.12)] transition-colors">
                  Cancel
                </button>
              </div>
            )}
          </div>

          {/* Edit form */}
          {editing === agent.name && (
            <div className="px-4 pb-4 space-y-3 border-t border-dashed border-[rgba(30,28,23,0.2)] pt-3">
              {error && <p className="text-[12px] text-red-400">{error}</p>}
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-[11px] text-[rgba(30,28,23,0.5)] mb-1">Provider</label>
                  <DashedSelect value={form.provider} onChange={v => setForm(f => ({ ...f, provider: v }))} options={[{ value: "", label: "— none —" }, ...providerIds.map(id => ({ value: id, label: id }))]} placeholder="— none —" />
                </div>
                <div>
                  <label className="block text-[11px] text-[rgba(30,28,23,0.5)] mb-1">Model</label>
                  <input className={inputCls} placeholder="e.g. claude-sonnet-4-6"
                    value={form.model} onChange={e => setForm(f => ({ ...f, model: e.target.value }))} />
                </div>
              </div>
              <div>
                <label className="block text-[11px] text-[rgba(30,28,23,0.5)] mb-1">System Prompt</label>
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
    return <div><span className="text-[rgba(30,28,23,0.4)] italic">{line}</span>{"\n"}</div>;
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
      t.type === "kw"     ? "text-[#9333ea] font-semibold" :
      t.type === "str"    ? "text-[#16a34a]" :
      t.type === "interp" ? "text-[#c85a2a]" :
      t.type === "num"    ? "text-[#0284c7]" :
                            "text-[#1e1c17]";
    return <span key={i} className={cls}>{t.value}</span>;
  });
}

const inputCls =
  "w-full px-3 py-2 text-[11px] font-mono border border-dashed border-[rgba(30,28,23,0.15)] " +
  "focus:outline-none focus:border-[rgba(30,28,23,0.35)] " +
  "bg-[rgba(30,28,23,0.03)] text-[#1e1c17] placeholder-[rgba(30,28,23,0.25)]";


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
    <div className="p-6 max-w-2xl">
      <header className="mb-5 border-b border-dashed border-[rgba(30,28,23,0.1)] pb-4">
        <h2 className="text-[11px] font-bold text-[#1e1c17] tracking-widest uppercase">// vault</h2>
        <p className="text-[10px] text-[rgba(30,28,23,0.4)] mt-0.5 tracking-wider uppercase">env secrets for workflows</p>
      </header>

      {error && <p className="mb-4 text-[12px] text-red-400">{error}</p>}

      {/* Entry list */}
      <div className="space-y-2 mb-4">
        {entries.length === 0 && !adding && (
          <p className="text-[13px] text-[rgba(30,28,23,0.4)]">No secrets stored yet.</p>
        )}
        {entries.map(e => (
          <div key={e.key} className="bg-[rgba(30,28,23,0.04)] border border-[rgba(30,28,23,0.08)] rounded-sm overflow-hidden">
            <div className="flex items-center gap-3 px-4 py-3">
              <span className="font-mono text-[13px] text-[#1e1c17] flex-shrink-0 w-44 truncate">{e.key}</span>
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
                <span className="flex-1 font-mono text-[12px] text-[rgba(30,28,23,0.5)] truncate">
                  {revealed.has(e.key) ? e.value : "••••••••••••"}
                </span>
              )}
              <div className="flex gap-1.5 flex-shrink-0">
                {editKey === e.key ? (
                  <>
                    <button onClick={() => handleSave(e.key)} disabled={saving}
                      className="px-2.5 py-1 text-[11px] bg-[#c85a2a] text-white rounded-sm hover:bg-[#a84a22] disabled:opacity-50 font-semibold">Save</button>
                    <button onClick={() => setEditKey(null)}
                      className="px-2.5 py-1 text-[11px] bg-[rgba(30,28,23,0.08)] text-[rgba(30,28,23,0.5)] rounded-sm hover:bg-[rgba(30,28,23,0.12)]">Cancel</button>
                  </>
                ) : (
                  <>
                    <button onClick={() => toggleReveal(e.key)}
                      className="px-2.5 py-1 text-[11px] bg-[rgba(30,28,23,0.08)] text-[rgba(30,28,23,0.5)] rounded-sm hover:bg-[rgba(30,28,23,0.12)] font-mono">
                      {revealed.has(e.key) ? "hide" : "show"}
                    </button>
                    <button onClick={() => { setEditKey(e.key); setEditVal(e.value); }}
                      className="px-2.5 py-1 text-[11px] bg-[rgba(30,28,23,0.08)] text-[rgba(30,28,23,0.5)] rounded-sm hover:bg-[rgba(30,28,23,0.12)]">Edit</button>
                    <button onClick={() => handleDelete(e.key)} disabled={saving}
                      className="px-2.5 py-1 text-[11px] bg-[rgba(239,68,68,0.12)] text-red-400 rounded-sm hover:bg-[rgba(239,68,68,0.2)] disabled:opacity-50">Del</button>
                  </>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Add new */}
      {adding ? (
        <div className="bg-[rgba(30,28,23,0.04)] border border-[rgba(200,90,42,0.3)] rounded-sm p-4 space-y-3">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-[11px] text-[rgba(30,28,23,0.5)] mb-1">Key</label>
              <input autoFocus value={newKey} onChange={e => setNewKey(e.target.value)}
                placeholder="e.g. GITHUB_TOKEN" className={inputCls + " font-mono"} />
            </div>
            <div>
              <label className="block text-[11px] text-[rgba(30,28,23,0.5)] mb-1">Value</label>
              <input type="password" value={newVal} onChange={e => setNewVal(e.target.value)}
                placeholder="secret value" className={inputCls + " font-mono"}
                onKeyDown={e => { if (e.key === "Enter") handleAdd(); if (e.key === "Escape") setAdding(false); }} />
            </div>
          </div>
          <div className="flex gap-2">
            <button onClick={handleAdd} disabled={saving || !newKey.trim() || !newVal.trim()}
              className="px-4 py-1.5 text-[12px] bg-[#c85a2a] text-white rounded-sm hover:bg-[#a84a22] disabled:opacity-40 font-semibold">
              {saving ? "Saving…" : "Add Secret"}
            </button>
            <button onClick={() => { setAdding(false); setNewKey(""); setNewVal(""); }}
              className="px-4 py-1.5 text-[12px] bg-[rgba(30,28,23,0.08)] text-[rgba(30,28,23,0.5)] rounded-sm hover:bg-[rgba(30,28,23,0.12)]">
              Cancel
            </button>
          </div>
        </div>
      ) : (
        <button onClick={() => setAdding(true)}
          className="px-4 py-2 text-[13px] bg-[rgba(30,28,23,0.08)] text-[rgba(30,28,23,0.7)] rounded hover:bg-[rgba(30,28,23,0.12)] hover:text-[rgba(30,28,23,0.9)] transition-colors border border-[rgba(30,28,23,0.08)]">
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
    <div className="p-6 max-w-3xl">
      <header className="mb-5 border-b border-dashed border-[rgba(30,28,23,0.1)] pb-4">
        <h2 className="text-[11px] font-bold text-[#1e1c17] tracking-widest uppercase">// browser</h2>
        <p className="text-[10px] text-[rgba(30,28,23,0.4)] mt-0.5 tracking-wider uppercase">Chrome CDP automation</p>
      </header>

      {/* Status card */}
      <div className="bg-[rgba(30,28,23,0.04)] border border-[rgba(30,28,23,0.08)] rounded-sm p-5 mb-6">
        <div className="flex items-center gap-3 mb-3">
          <span className={`w-2.5 h-2.5 rounded-full ${status?.connected ? "bg-green-400" : "bg-red-400"}`} />
          <span className="text-[14px] font-medium text-[#1e1c17]">
            {status === null ? "Checking..." : status.connected ? "Connected" : "Disconnected"}
          </span>
        </div>

        {status?.connected && (
          <div className="space-y-1.5 mb-4">
            {status.browser && (
              <p className="text-[12px] text-[rgba(30,28,23,0.5)]">
                <span className="text-[rgba(30,28,23,0.4)]">Browser:</span> {status.browser}
              </p>
            )}
            <p className="text-[12px] text-[rgba(30,28,23,0.5)] font-mono break-all">
              <span className="text-[rgba(30,28,23,0.4)]">CDP URL:</span> {status.cdpURL}
            </p>
          </div>
        )}

        {status && !status.connected && status.error && (
          <p className="text-[12px] text-[rgba(30,28,23,0.35)] mb-4">{status.error}</p>
        )}

        <div className="flex gap-2">
          <button
            onClick={testCDP}
            disabled={testing}
            className="px-4 py-2 text-[13px] rounded-sm bg-[rgba(30,28,23,0.08)] hover:bg-[rgba(30,28,23,0.1)] text-[#1e1c17] disabled:opacity-40 transition-colors"
          >
            {testing ? "Testing..." : "Test CDP"}
          </button>
          <button
            onClick={launchChrome}
            disabled={launching}
            className="px-4 py-2 text-[13px] rounded-sm bg-[rgba(200,90,42,0.2)] hover:bg-[rgba(200,90,42,0.3)] text-[#c85a2a] disabled:opacity-40 transition-colors"
          >
            {launching ? "Launching..." : "Launch Chrome with CDP"}
          </button>
        </div>
      </div>

      {/* Shortcuts */}
      <h3 className="text-[15px] font-medium text-[#1e1c17] mb-3">Shortcuts</h3>
      <p className="text-[12px] text-[rgba(30,28,23,0.4)] mb-4">
        YAML browser automation adapters in <code className="text-[rgba(30,28,23,0.5)]">~/.clawfirm/shortcuts/</code>.
        Requires Chrome connected via CDP.
      </p>

      {/* Args input */}
      <div className="mb-4">
        <input
          type="text"
          value={cmdArgs}
          onChange={e => setCmdArgs(e.target.value)}
          placeholder="Command arguments (space-separated)..."
          className="w-full px-3 py-2 text-[13px] rounded-sm bg-[rgba(30,28,23,0.08)] border border-[rgba(30,28,23,0.08)] text-[#1e1c17] placeholder:text-[rgba(30,28,23,0.2)] outline-none focus:border-[rgba(200,90,42,0.4)]"
        />
      </div>

      {shortcuts.length === 0 ? (
        <p className="text-[13px] text-[rgba(30,28,23,0.4)]">No shortcuts found.</p>
      ) : (
        <div className="space-y-3">
          {shortcuts.map(sc => (
            <div key={sc.file} className="bg-[rgba(30,28,23,0.04)] border border-[rgba(30,28,23,0.08)] rounded-sm p-4">
              <div className="flex items-center gap-2 mb-2">
                <span className="text-[14px] font-medium text-[#1e1c17]">{sc.platform}</span>
                <span className="text-[11px] text-[rgba(30,28,23,0.2)] font-mono">{sc.file}</span>
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
                      className="px-3 py-1.5 text-[12px] rounded-md bg-[rgba(30,28,23,0.08)] hover:bg-[rgba(30,28,23,0.12)] text-[rgba(30,28,23,0.65)] disabled:opacity-30 transition-colors font-mono"
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
        <div className="mt-4 p-3 rounded-sm bg-[rgba(255,80,80,0.1)] border border-[rgba(255,80,80,0.2)]">
          <p className="text-[12px] text-red-400 font-mono whitespace-pre-wrap">{cmdError}</p>
        </div>
      )}
      {cmdResult && cmdResult.length > 0 && (
        <div className="mt-4 overflow-x-auto">
          <table className="w-full text-[12px] text-[rgba(30,28,23,0.65)]">
            <thead>
              <tr className="border-b border-[rgba(30,28,23,0.08)]">
                {Object.keys(cmdResult[0]).map(k => (
                  <th key={k} className="text-left py-2 px-2 font-medium text-[rgba(30,28,23,0.5)]">{k}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {cmdResult.map((row, i) => (
                <tr key={i} className="border-b border-[rgba(30,28,23,0.04)]">
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
      <div className="w-52 flex-shrink-0 border-r border-[rgba(30,28,23,0.1)] flex flex-col">
        <div className="px-4 py-3 border-b border-[rgba(30,28,23,0.1)]">
          <span className="text-[11px] font-semibold text-[rgba(30,28,23,0.35)] uppercase tracking-wider">Workflows</span>
          <p className="text-[10px] text-[rgba(30,28,23,0.25)] font-mono mt-0.5 break-all">~/.clawfirm/workflows/</p>
        </div>
        <div className="flex-1 overflow-y-auto py-1">
          {whipFiles.length === 0 ? (
            <p className="px-4 py-3 text-[12px] text-[rgba(30,28,23,0.2)]">No .whip files found</p>
          ) : whipFiles.map(f => (
            <button
              key={f}
              onClick={() => selectFile(f)}
              className={`w-full text-left px-4 py-2 text-[12.5px] truncate transition-colors ${
                selectedFile === f
                  ? "bg-[rgba(30,28,23,0.08)] text-[#1e1c17]"
                  : "text-[rgba(30,28,23,0.55)] hover:bg-[rgba(30,28,23,0.04)] hover:text-[#1e1c17]"
              }`}
            >
              <span className="text-[rgba(30,28,23,0.2)] mr-1">⚡</span>{basename(f)}
            </button>
          ))}
        </div>

        {/* ── Config section (collapsed by default) ── */}
        <div className="border-t border-[rgba(30,28,23,0.1)]">
          <button
            onClick={() => setCfgOpen(o => !o)}
            className="w-full flex items-center justify-between px-4 py-2.5 text-[11px] text-[rgba(30,28,23,0.35)] hover:text-[rgba(30,28,23,0.7)] transition-colors"
          >
            <span className="font-semibold uppercase tracking-wider">Config</span>
            <span>{cfgOpen ? "▲" : "▼"}</span>
          </button>
          {cfgOpen && (
            <div className="px-3 pb-3 space-y-2">
              {error && <p className="text-[11px] text-red-400">{error}</p>}
              <DashedSelect
                value={defaultProvider}
                onChange={v => setDefaultProvider(v)}
                placeholder="— default (first agent) —"
                options={[
                  ...BUILTIN_PROVIDERS.map(p => ({ value: p, label: p })),
                  ...agentNames.map(n => ({ value: n, label: n })),
                  { value: "__custom__", label: "Custom…" },
                ]}
              />
              {defaultProvider === "__custom__" && (
                <input
                  className="w-full bg-[rgba(30,28,23,0.05)] border border-[rgba(30,28,23,0.12)] rounded-sm px-2 py-1.5 text-[11.5px] text-[#1e1c17] outline-none"
                  placeholder="provider name"
                  value={customProvider}
                  onChange={e => setCustomProvider(e.target.value)}
                />
              )}
              <div className="flex items-center gap-2">
                <button onClick={handleSave} disabled={saving || !cfg}
                  className="px-3 py-1 bg-[#c85a2a] text-white text-[11px] rounded-sm hover:bg-[#a84a22] disabled:opacity-50 transition-colors font-semibold">
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
            <div className="px-5 py-3 border-b border-[rgba(30,28,23,0.1)] flex items-center gap-2">
              <span className="text-[rgba(30,28,23,0.5)] text-[12px]">⚡</span>
              <span className="text-[13px] font-medium text-[#1e1c17]">{basename(selectedFile)}</span>
              <span className="text-[11px] text-[rgba(30,28,23,0.2)] ml-1 truncate">{selectedFile}</span>
            </div>
            <div className="flex-1 overflow-auto p-5">
              {fileError ? (
                <p className="text-[12px] text-red-400">{fileError}</p>
              ) : (
                <div className="bg-[rgba(30,28,23,0.06)] rounded p-4 border border-[rgba(30,28,23,0.08)]">
                  <WhipHighlight code={fileContent} />
                </div>
              )}
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center text-[13px] text-[rgba(30,28,23,0.3)]">
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
    <div className="p-6 h-full flex flex-col">
      <header className="mb-5 flex-shrink-0 border-b border-dashed border-[rgba(30,28,23,0.1)] pb-4">
        <h2 className="text-[11px] font-bold text-[#1e1c17] tracking-widest uppercase">// settings</h2>
        <p className="text-[10px] text-[rgba(30,28,23,0.3)] mt-0.5 font-mono">{filePath}</p>
      </header>
      <textarea
        value={content}
        onChange={(e) => setContent(e.target.value)}
        spellCheck={false}
        className="flex-1 min-h-0 w-full max-w-2xl px-4 py-3 text-[13px] font-mono bg-[rgba(30,28,23,0.05)] border border-[rgba(30,28,23,0.12)] rounded-sm focus:outline-none focus:ring-2 focus:ring-[rgba(200,90,42,0.4)] resize-none leading-relaxed text-[#1e1c17] placeholder-[rgba(30,28,23,0.3)] backdrop-blur-xl"
      />
      <div className="flex items-center gap-3 mt-4 flex-shrink-0">
        <button onClick={handleSave} disabled={saving}
          className="px-5 py-2 bg-[#c85a2a] text-white text-[13px] rounded hover:bg-[#a84a22] disabled:opacity-50 transition-colors font-semibold">
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
      <RemoteCard />
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
    disconnected: { label: "Disconnected", cls: "bg-[rgba(30,28,23,0.08)] text-[rgba(30,28,23,0.5)]" },
    logged_out:   { label: "Logged out",   cls: "bg-[rgba(255,69,58,0.15)] text-red-400" },
    disabled:     { label: "Disabled",     cls: "bg-[rgba(30,28,23,0.05)] text-[rgba(30,28,23,0.2)]" },
  };

  return (
    <Card title="WhatsApp"
      badge={<Badge {...(waMap[status] ?? waMap.disconnected)} />}
      action={status === "connected"
        ? <button onClick={() => LogoutWhatsApp().catch(() => {})} className="text-[11px] text-red-400 hover:text-red-300">Disconnect</button>
        : null}>
      {status === "qr_pending" && qrURL && (
        <div className="flex flex-col items-center gap-2 pt-2">
          <p className="text-[11px] text-[rgba(30,28,23,0.5)] text-center">WhatsApp → Linked Devices → Link a Device → scan QR</p>
          <img src={qrURL} alt="QR" className="w-44 h-44 rounded border border-[rgba(30,28,23,0.12)]" />
        </div>
      )}
      {status === "qr_pending" && !qrURL && <p className="text-[11px] text-[rgba(30,28,23,0.4)] italic pt-1">Generating QR…</p>}
      {status === "disabled" && (
        <p className="text-[11px] text-[rgba(30,28,23,0.4)] pt-1">
          Set <code className="font-mono bg-[rgba(30,28,23,0.08)] px-1 rounded">whatsapp.enabled: true</code> in config.yml to enable.
        </p>
      )}
      {(status === "disconnected" || status === "logged_out") && (
        <p className="text-[11px] text-[rgba(30,28,23,0.4)] italic pt-1">
          {status === "logged_out" ? "Logged out — restart app to pair again." : "Waiting for gateway…"}
        </p>
      )}
      {status === "connected" && <p className="text-[11px] text-[rgba(30,28,23,0.4)] pt-1">Connected and receiving messages.</p>}
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
        : <Badge label="未配置" cls="bg-[rgba(30,28,23,0.08)] text-[rgba(30,28,23,0.5)]" />}
      action={configured && !editing
        ? <button onClick={() => setEditing(true)} className="text-[11px] text-[#c85a2a] hover:text-[#5aa3fb]">修改</button>
        : null}>
      {(!configured || editing) ? (
        <div className="space-y-3 pt-1">
          <p className="text-[11px] text-[rgba(30,28,23,0.4)]">企业自建应用，开启消息事件订阅（长连接）。</p>
          <div>
            <label className="block text-[11px] text-[rgba(30,28,23,0.5)] mb-1">App ID</label>
            <input type="text" placeholder="cli_xxxxxxxxxxxx" value={appID} onChange={(e) => setAppID(e.target.value)}
              className="w-full px-3 py-2 text-[13px] border border-[rgba(30,28,23,0.12)] rounded focus:outline-none focus:ring-2 focus:ring-[rgba(200,90,42,0.4)] bg-[rgba(30,28,23,0.05)] text-[#1e1c17] placeholder-[rgba(30,28,23,0.3)]" />
          </div>
          <div>
            <label className="block text-[11px] text-[rgba(30,28,23,0.5)] mb-1">App Secret</label>
            <input type="password" placeholder={secretMasked || "App Secret"} value={appSecret} onChange={(e) => setAppSecret(e.target.value)}
              className="w-full px-3 py-2 text-[13px] border border-[rgba(30,28,23,0.12)] rounded focus:outline-none focus:ring-2 focus:ring-[rgba(200,90,42,0.4)] bg-[rgba(30,28,23,0.05)] text-[#1e1c17] placeholder-[rgba(30,28,23,0.3)]" />
          </div>
          {error && <p className="text-[11px] text-red-400">{error}</p>}
          <div className="flex gap-2">
            <button onClick={handleSave} disabled={saving}
              className="flex-1 py-2 rounded bg-[#c85a2a] text-white text-[13px] font-semibold hover:bg-[#a84a22] disabled:opacity-50">
              {saving ? "保存中…" : "保存并连接"}
            </button>
            {editing && (
              <button onClick={() => { setEditing(false); setAppSecret(""); setError(""); }}
                className="px-3 py-2 rounded border border-[rgba(30,28,23,0.12)] text-[13px] text-[rgba(30,28,23,0.5)] hover:bg-[rgba(30,28,23,0.05)]">
                取消
              </button>
            )}
          </div>
        </div>
      ) : (
        <div className="pt-1">
          <p className="text-[13px] text-[rgba(30,28,23,0.5)]">App ID: <span className="font-mono text-[#1e1c17] text-[11px]">{appID}</span></p>
          <p className="mt-1 text-[11px] text-[rgba(30,28,23,0.4)]">通过 WebSocket 长连接接收消息，无需公网地址。</p>
        </div>
      )}
    </Card>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// Remote Control card
// ─────────────────────────────────────────────────────────────────────────────
function RemoteCard() {
  const [status, setStatus] = useState<RemoteStatus | null>(null);
  const [enabling, setEnabling] = useState(false);
  const [ngrokToken, setNgrokToken] = useState("");
  const [error, setError] = useState("");

  // Poll status every 2s when enabled.
  useEffect(() => {
    let active = true;
    async function poll() {
      try {
        const s = await GetRemoteStatus();
        if (active) setStatus(s);
      } catch {}
    }
    poll();
    const id = setInterval(poll, POLL_MS);
    return () => { active = false; clearInterval(id); };
  }, []);

  const enabled = status?.ngrokOn ?? false;

  async function handleEnable() {
    if (!ngrokToken.trim()) { setError("Enter your ngrok auth token first"); return; }
    setError(""); setEnabling(true);
    try {
      const s = await EnableNgrok(ngrokToken.trim());
      console.log("EnableNgrok result:", s);
      setStatus(s);
    } catch (e: unknown) {
      console.error("EnableNgrok error:", e);
      setError(String(e));
    } finally {
      setEnabling(false);
    }
  }

  async function handleDisable() {
    setError("");
    try { await DisableRemote(); setStatus(null); } catch (e: unknown) { setError(String(e)); }
  }

  const badgeProps = enabled
    ? { label: "Running", cls: "bg-[rgba(52,199,89,0.15)] text-emerald-400" }
    : { label: "Off", cls: "bg-[rgba(30,28,23,0.08)] text-[rgba(30,28,23,0.5)]" };

  return (
    <Card title="Remote"
      badge={<Badge {...badgeProps} />}
      action={enabled ? (
        <button onClick={handleDisable}
          className="text-[11px] text-red-400 hover:text-red-300">
          Disable
        </button>
      ) : undefined}>
      {!enabled && (
        <div className="space-y-3 pt-1">
          <p className="text-[11px] text-[rgba(30,28,23,0.4)]">
            Connect via ngrok so the mobile app can access your agents remotely.
          </p>
          <input type="text" placeholder="ngrok auth token" value={ngrokToken} onChange={(e) => setNgrokToken(e.target.value)}
            className="w-full px-3 py-2 text-[12px] border border-[rgba(30,28,23,0.12)] rounded focus:outline-none focus:ring-2 focus:ring-[rgba(200,90,42,0.4)] bg-[rgba(30,28,23,0.05)] text-[#1e1c17] placeholder-[rgba(30,28,23,0.3)] font-mono" />
          <button onClick={handleEnable} disabled={enabling}
            className="w-full py-2 rounded bg-[#c85a2a] text-white text-[12px] font-semibold hover:bg-[#a84a22] disabled:opacity-50">
            {enabling ? "Starting…" : "Enable Remote"}
          </button>
        </div>
      )}
      {enabled && status && (
        <div className="space-y-3 pt-1">
          {/* QR Code */}
          {status.qrCode && (
            <div className="flex flex-col items-center gap-2">
              <img src={status.qrCode} alt="Remote QR" className="w-44 h-44 rounded border border-[rgba(30,28,23,0.12)]" />
              <p className="text-[11px] text-[rgba(30,28,23,0.4)] text-center">Scan with mobile app to connect</p>
            </div>
          )}
          {/* ngrok URL */}
          <div>
            <p className="text-[11px] text-[rgba(30,28,23,0.5)]">Public URL</p>
            <p className="text-[12px] font-mono text-[#1e1c17] break-all">{status.ngrokUrl}</p>
          </div>
          {/* Connected clients */}
          <div className="flex items-center gap-2">
            <span className="text-[11px] text-[rgba(30,28,23,0.5)]">Connected devices:</span>
            <span className="text-[12px] font-semibold text-[#1e1c17]">{status.clients}</span>
          </div>
        </div>
      )}
      {error && <p className="text-[11px] text-red-400 pt-1">{error}</p>}
    </Card>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// Primitives
// ─────────────────────────────────────────────────────────────────────────────
function GlassCard({ children, className = "" }: { children: React.ReactNode; className?: string }) {
  return (
    <div className={`bg-[rgba(30,28,23,0.03)] border border-dashed border-[rgba(30,28,23,0.12)] p-4 ${className}`}>
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
          <span className="font-bold text-[#1e1c17] text-[11px] uppercase tracking-widest">// {title}</span>
          {badge}
        </div>
        {action}
      </div>
      {children}
    </GlassCard>
  );
}

function Badge({ label, cls }: { label: string; cls: string }) {
  return <span className={`text-[10px] font-mono px-1.5 py-0.5 border border-dashed uppercase tracking-wider ${cls}`}>{label}</span>;
}
