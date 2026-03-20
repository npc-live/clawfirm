/**
 * clawfirm tool registry
 * Each tool needs: name, install command, and the binary it provides.
 */
export default {
  tools: [
    {
      name: "openvault",
      description: "Encrypted local secret manager",
      requires: "go",
      install: "go install github.com/npc-live/openvault@latest",
      uninstall: "rm -f $(which openvault)",
      check: "openvault",
      homepage: "https://openvault.sh",
    },
    {
      name: "skillctl",
      description: "Sync AI skills across coding tools",
      requires: "npm",
      install: "npm install -g @harness.farm/skillctl",
      uninstall: "npm uninstall -g @harness.farm/skillctl",
      check: "skillctl",
      homepage: "https://skillctl.dev",
    },
    {
      name: "whipflow",
      description: "Deterministic AI workflow runner",
      requires: "npm",
      install: "npm install -g @harness.farm/whipflow",
      uninstall: "npm uninstall -g @harness.farm/whipflow",
      check: "whipflow",
      homepage: "https://whipflow.dev",
    },
    {
      name: "agent-browser",
      description: "Browser automation for AI agents",
      requires: "npm",
      install: "npm install -g agent-browser",
      uninstall: "npm uninstall -g agent-browser",
      check: "agent-browser",
      homepage: "https://www.npmjs.com/package/agent-browser",
    },
  ],
};
