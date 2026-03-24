# clawfirm

Dev tool manager for AI-powered workflows.

## Install

```bash
npm install -g clawfirm
```

## Usage

```
clawfirm login                    Log in to clawfirm.dev
clawfirm whoami                   Show current session
clawfirm logout                   Log out

clawfirm install [tool]           Install all tools (or a specific one)
clawfirm uninstall [tool]         Uninstall all tools (or a specific one)
clawfirm list                     List registered tools

clawfirm new "<description>"      Start a project from natural language
clawfirm <name> [args]            Dispatch to clawfirm-<name>
```

## Tools

| Tool | Description |
|------|-------------|
| `openvault` | Encrypted local secret manager |
| `skillctl` | Sync AI skills across coding tools |
| `whipflow` | Deterministic AI workflow runner |
| `agent-browser` | Browser automation for AI agents |

## Quick Start

```bash
clawfirm login
clawfirm install
```

`install` installs all tools and syncs your skills to Claude Code.
