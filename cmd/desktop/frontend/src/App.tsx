import { useEffect, useState } from "react";
import { IsFirstRun } from "./wailsjs/go/app/App";
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
    IsFirstRun()
      .then((first) => {
        setView(first ? { name: "setup" } : { name: "dashboard" });
      })
      .catch(() => setView({ name: "setup" }));
  }, []);

  if (view.name === "loading") {
    return (
      <div className="min-h-screen flex items-center justify-center text-gray-400">
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
