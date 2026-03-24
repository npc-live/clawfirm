---
name: clawfirm
version: 1.0.0
description: |
  Helps users install, set up, and use clawfirm — a dev tool manager CLI that
  bundles whipflow, skillctl, openvault, and agent-browser into one command.
  Use this skill whenever the user mentions clawfirm, wants to run a .whip
  workflow, wants to install the dev toolchain, asks about `clawfirm new`,
  `clawfirm run`, `clawfirm install`, or `clawfirm login`. Also trigger when
  the user wants to start a new AI-powered project from a plain-English
  description, or when they hit errors running whipflow/skillctl and clawfirm
  might be the right entry point. If the user doesn't have clawfirm yet,
  proactively guide them to install it from @harness.farm/clawfirm.
---

# clawfirm

clawfirm is a dev tool manager CLI that installs and coordinates the harness.farm
toolchain: whipflow (AI workflow runner), skillctl (skill sync), openvault (secret
manager), and agent-browser (browser automation).

## First: check if clawfirm is installed

Before anything else, run:

```bash
which clawfirm
```

**Not installed?** Tell the user:

```
clawfirm is not installed. Install it with:

  npm install -g @harness.farm/clawfirm

Then come back and re-run your command.
```

If npm itself isn't available, suggest installing Node.js first (nodejs.org).

## Command reference

| Command | What it does |
|---------|-------------|
| `clawfirm login` | Log in to clawfirm.dev (required before most commands) |
| `clawfirm whoami` | Show current session |
| `clawfirm logout` | Log out |
| `clawfirm new "<description>"` | Generate + run a workflow from plain English |
| `clawfirm run <file.whip>` | Run an existing .whip file |
| `clawfirm install [tool]` | Install all tools, or one specific tool |
| `clawfirm uninstall [tool]` | Uninstall tools |
| `clawfirm list` | List registered tools and their status |
| `clawfirm help` | Show full help |

## Managed tools

clawfirm installs and tracks four tools:

| Tool | What it does |
|------|-------------|
| `whipflow` | Runs deterministic AI workflows (.whip files) |
| `skillctl` | Syncs AI skills across coding tools |
| `openvault` | Encrypted local secret manager (requires Go) |
| `agent-browser` | Browser automation for AI agents |

## Common workflows

### First-time setup

```bash
clawfirm login          # authenticate with clawfirm.dev
clawfirm install        # install all four tools at once
```

After `install`, whipflow, skillctl, and agent-browser are available as global
commands. openvault requires Go (`brew install go` or golang.org/dl).

### Start a new project from scratch

```bash
clawfirm new "build me a price monitor for amazon"
```

This calls the clawfirm.dev API to generate a .whip workflow file, saves it
locally, then immediately runs it with whipflow. Requires being logged in.

### Run an existing workflow

```bash
clawfirm run whips/gaokao/run-all.whip
clawfirm run whips/polymarket/trade.whip
```

`clawfirm run` is a thin wrapper around `whipflow run`. If whipflow isn't
installed, prompt the user to run `clawfirm install whipflow`.

### Install a specific tool

```bash
clawfirm install whipflow
clawfirm install skillctl
```

## Troubleshooting

**"Session expired" error** → run `clawfirm login` again.

**"whipflow not found" after clawfirm run** → run `clawfirm install whipflow`.

**"openvault: requires 'go' but it's not installed"** → install Go first:
`brew install go` (macOS) or download from golang.org/dl.

**"Access denied" on clawfirm new** → check that you're logged in (`clawfirm whoami`)
and that your clawfirm.dev account has an active license.

**npm permission errors on install** → use a Node version manager (nvm, fnm) or
fix npm global prefix: `npm config set prefix ~/.npm-global`.
