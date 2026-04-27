import { useEffect, useState } from "react";
import { IsFirstRun } from "./lib/wails-shim";
import { SetupWizard } from "./components/SetupWizard";
import { Dashboard } from "./components/Dashboard";
import { ChatView } from "./components/ChatView";


type View =
  | { name: "loading" }
  | { name: "setup" }
  | { name: "dashboard" }
  | { name: "chat"; agentName: string; sessionID: string };

export default function App() {
  const [view, setView] = useState<View>({ name: "loading" });

  useEffect(() => {
    let cancelled = false;
    const init = async () => {
      // Retry until OnStartup finishes (cfg may be nil for a few hundred ms).
      let first: boolean | null = null;
      for (let i = 0; i < 30; i++) {
        try {
          first = await IsFirstRun();
          break;
        } catch {
          await new Promise((r) => setTimeout(r, 200));
        }
      }
      if (cancelled) return;
      if (first === null) { setView({ name: "setup" }); return; }
      if (first) { setView({ name: "setup" }); return; }

      if (!cancelled) setView({ name: "dashboard" });
    };
    init();
    return () => { cancelled = true; };
  }, []);

  if (view.name === "loading") {
    return (
      <div className="min-h-screen flex items-center justify-center text-[rgba(61,57,41,0.4)] bg-[#f5f0e8]">
        <div className="animate-pulse text-lg">Loading…</div>
      </div>
    );
  }

  if (view.name === "setup") {
    return (
      <SetupWizard onComplete={() => setView({ name: "dashboard" })} />
    );
  }

  if (view.name === "chat") {
    return (
      <ChatView
        agentName={view.agentName}
        sessionID={view.sessionID}
        onBack={() => setView({ name: "dashboard" })}
        onNewSession={() =>
          setView({ name: "chat", agentName: view.agentName, sessionID: "s" + Date.now() })
        }
        onOpenSession={(name, sid) => setView({ name: "chat", agentName: name, sessionID: sid })}
      />
    );
  }

  return (
    <Dashboard
      onOpenChat={(agentName, sessionID) =>
        setView({ name: "chat", agentName, sessionID })
      }
    />
  );
}
