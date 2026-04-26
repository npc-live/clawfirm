---
name: solana-wallet
description: >
  Manage Solana and Polygon wallets, run Polymarket weather arbitrage, post to X/Twitter,
  and execute Raydium swaps — all from natural language.
version: 1.1.2
homepage: https://github.com/inspi-writer001/raphael-solana
user-invocable: true
metadata:
  openclaw:
    emoji: "🤖"
    primaryEnv: MASTER_ENCRYPTION_PASSWORD_CRYPTO
    requires:
      env:
        - MASTER_ENCRYPTION_PASSWORD_CRYPTO
        - MASTER_ENCRYPTED
        - MASTER_SALT
        - SOLANA_RPC_URL
        - X_API_KEY
        - X_API_SECRET
        - X_ACCESS_TOKEN
        - X_ACCESS_TOKEN_SECRET
        - X_BEARER_TOKEN
      anyBins:
        - node
        - tsx
    install:
      node:
        - "."
    os:
      - macos
      - linux
---

# Solana + Polymarket + X Wallet Agent Skill

**Source code:** https://github.com/inspi-writer001/raphael-solana

All code executed by this skill lives in that public repository. Review it before providing credentials or enabling live trading.

You control Solana wallets, Polygon EVM wallets, a Polymarket weather arbitrage scanner,
and an X/Twitter strategy — all from natural language.

## Setup

This skill is self-contained. After `clawhub install solana-wallet`, the source code and dependencies are installed automatically — no manual repo clone needed.

Add your credentials to `~/.openclaw/.env` (see **Environment Variables** below).

## Execution Rules

1. **The CLI is bundled with this skill. The exec prefix is:**
   ```
   node --experimental-transform-types {baseDir}/bin/solana-wallet.ts
   ```
   `{baseDir}` is resolved by OpenClaw to this skill's installation directory.
2. The CLI reads only the specific env vars listed in the **Environment Variables** section below — no others.
3. **Prefer plugin tools over CLI** when available — all 13 tools are available as direct plugin calls and require no exec.
4. The following Node.js warnings are expected and harmless: `ExperimentalWarning`, `bigint` deprecation, `punycode`. Disregard them in output parsing.

## Plugin Tools (use these first — no exec needed)

These 13 tools are registered by [`src/plugin.ts`](https://github.com/inspi-writer001/raphael-solana/blob/main/src/plugin.ts), bundled in this skill package at `{baseDir}/src/plugin.ts`. OpenClaw loads them automatically when the skill is activated.

### Wallet & Polymarket

| Tool | When to use |
|---|---|
| `create_evm_wallet` | User wants to create a Polygon wallet for Polymarket |
| `list_evm_wallets` | User asks what EVM wallets exist |
| `check_usdc_balance` | User wants to verify USDC arrived on Polygon |
| `start_weather_arb` | User wants to start the weather arb scanner |
| `stop_weather_arb` | User wants to stop the weather arb scanner |
| `get_strategy_status` | User asks about scanner status, city readings, edges, X tweet count |

### X / Twitter

| Tool | When to use |
|---|---|
| `x_post_tweet` | User wants to post a tweet |
| `x_reply` | User wants to reply to a specific tweet |
| `x_search` | User wants to search recent tweets (requires Basic+ X tier) |
| `x_get_mentions` | User wants to see recent mentions of the bot |
| `x_resolve_user` | User wants to look up a Twitter user by @handle |
| `start_x_strategy` | User wants to start mention monitoring / keyword feed / trade posting |
| `stop_x_strategy` | User wants to stop the X strategy |

## CLI Command Reference

The CLI prefix for ALL commands below is:
```
node --experimental-transform-types {baseDir}/bin/solana-wallet.ts
```

### Solana Wallet Commands

| User says | Command |
|---|---|
| Check Solana balance | `<prefix> balance <wallet-name>` |
| Create Solana wallet | `<prefix> wallet create <name> [--network devnet\|mainnet-beta]` |
| List Solana wallets | `<prefix> wallet list` |
| Transfer SOL | `<prefix> transfer sol <wallet> <to-address> <amount>` |
| Transfer SPL token | `<prefix> transfer spl <wallet> <to-address> <mint> <amount>` |
| Transfer MATIC | `<prefix> transfer matic <wallet> <to-address> <amount>` |
| Transfer ERC-20 (USDC etc.) | `<prefix> transfer erc20 <wallet> <to-address> <token-address> <amount>` |
| Swap tokens | `<prefix> swap <wallet> SOL <output-mint> <amount>` |
| Find pump.fun plays | `<prefix> find-pairs` |

### EVM / Polygon Wallet Commands

| User says | Command |
|---|---|
| Create Polygon wallet | `<prefix> evm-wallet create <name>` |
| List Polygon wallets | `<prefix> evm-wallet list` |
| Check MATIC / ERC-20 balance | `<prefix> evm-wallet balance <name> [--token <address>]` |

### X / Twitter Commands

| User says | Command |
|---|---|
| Post a tweet | `<prefix> x tweet <text>` |
| Reply to a tweet | `<prefix> x reply <tweet-id> <text>` |
| Search tweets | `<prefix> x search <query> [--max 10]` |
| Check mentions | `<prefix> x mentions [--since <tweet-id>]` |
| Look up a user | `<prefix> x resolve <handle>` |
| Start X strategy | See full command below |

**Start X strategy (full command):**
```
node --experimental-transform-types {baseDir}/bin/solana-wallet.ts scanner start x \
  --handle <bot-handle> \
  [--keywords "pump.fun,graduation"] \
  [--post-trade-updates] \
  [--auto-reply] \
  [--max-tweets-per-hour 2] \
  [--interval 60] \
  [--dry-run]
```

### Scanner Commands

| User says | Command |
|---|---|
| Start weather arb | See full command below |
| Stop scanner | `node --experimental-transform-types {baseDir}/bin/solana-wallet.ts scanner stop` |
| Check scanner status | `node --experimental-transform-types {baseDir}/bin/solana-wallet.ts scanner status` |

**Start weather arb (full command):**
```
node --experimental-transform-types {baseDir}/bin/solana-wallet.ts scanner start polymarket-weather <evm-wallet-name> \
  --amount <usdc-per-trade> \
  [--cities nyc,london,seoul,chicago,dallas,miami,paris,toronto,seattle] \
  [--max-position <usdc>] \
  [--min-edge 0.20] \
  [--min-fair-value 0.40] \
  [--interval <seconds>] \
  [--dry-run]
```

## Typical Agent Flow: Polymarket Weather Arb

1. Create EVM wallet (plugin: `create_evm_wallet` or CLI: `evm-wallet create polymarket1`)
2. Tell user: **"Send USDC (Polygon PoS network) to: `<address>`"**
3. Poll balance until funded: `check_usdc_balance { wallet_name: "polymarket1" }`
4. Start dry run: `start_weather_arb { wallet_name: "polymarket1", trade_amount_usdc: 5, dry_run: true }`
5. Check readings after 2 minutes: `get_strategy_status`
6. If edges look reasonable, restart without dry run: `start_weather_arb { ..., dry_run: false }`

## Typical Agent Flow: X / Twitter

1. Confirm X credentials are set: `X_API_KEY`, `X_API_SECRET`, `X_ACCESS_TOKEN`, `X_ACCESS_TOKEN_SECRET`, `X_BEARER_TOKEN`
2. Start in dry-run to verify: `start_x_strategy { handle: "mybot", dry_run: true, post_trade_updates: true }`
3. Check status: `get_strategy_status` — shows tweets sent this hour
4. Once confirmed working, restart without dry run

## Supported Cities for Weather Arb

| Key | City |
|---|---|
| `nyc` | New York City |
| `london` | London |
| `seoul` | Seoul |
| `chicago` | Chicago |
| `dallas` | Dallas |
| `miami` | Miami |
| `paris` | Paris |
| `toronto` | Toronto |
| `seattle` | Seattle |

## Environment Variables

This is the **complete list** of every environment variable the skill reads. It reads no others.

### Required

| Variable | What it is | Used by |
|---|---|---|
| `MASTER_ENCRYPTION_PASSWORD_CRYPTO` | Your chosen password — held in memory only, never written to disk | Wallet decryption |
| `MASTER_ENCRYPTED` | AES-256-GCM encrypted master key blob (generated by `pnpm setup`) | Wallet decryption |
| `MASTER_SALT` | PBKDF2 salt for key derivation (generated by `pnpm setup`) | Wallet decryption |

### Optional

| Variable | Default | What it is | Used by |
|---|---|---|---|
| `SOLANA_RPC_URL` | `https://api.devnet.solana.com` | Solana JSON-RPC endpoint | Balances, transfers, swaps |
| `RAPHAEL_DATA_DIR` | `~/.raphael` | Directory for encrypted wallet files and scanner state | Wallet store, PID files |
| `WALLET_STORE_PATH` | `$RAPHAEL_DATA_DIR/wallets.json` | Path to encrypted Solana wallet JSON | Wallet store |
| `PUMPPORTAL_WS` | `wss://pumpportal.fun/api/data` | pump.fun WebSocket (public, no key needed) | pump.fun scanner |
| `X_API_KEY` | — | OAuth 1.0a consumer key | X writes (tweets, replies) |
| `X_API_SECRET` | — | OAuth 1.0a consumer secret | X writes |
| `X_ACCESS_TOKEN` | — | OAuth 1.0a user access token | X writes |
| `X_ACCESS_TOKEN_SECRET` | — | OAuth 1.0a user access token secret | X writes |
| `X_BEARER_TOKEN` | — | OAuth 2.0 app-only bearer token | X reads (search, timelines) |

X features are fully optional — the skill operates without any `X_*` vars. Obtain them from [developer.x.com](https://developer.x.com) → Projects & Apps → Keys and Tokens with **Read and Write** permissions.

### Wallet encryption model

Private keys are never stored in plaintext. The skill uses two-layer AES-256-GCM encryption entirely on your local machine — no keys or wallet data are sent to any remote server.

```
MASTER_ENCRYPTION_PASSWORD_CRYPTO  (your password, memory only)
  ↓ PBKDF2 — 100,000 iterations, SHA-256
MASTER_ENCRYPTED + MASTER_SALT     (encrypted blob — useless without the password)
  ↓ AES-256-GCM decrypt → master key
wallet private key                 (AES-256-GCM, per-wallet salt → ~/.raphael/)
```

`MASTER_ENCRYPTED` and `MASTER_SALT` are outputs of `pnpm setup` — they are specific to your password and machine. Sharing them without the password reveals nothing.

## Rules

- Always confirm before live trades (unless user explicitly says "just do it" or "no dry run")
- Always suggest `--dry-run` / `dry_run: true` for first-time scanner and X strategy starts
- Report Solana Explorer URL after Solana transactions
- Never display private keys
- For Polymarket: USDC must be on **Polygon PoS network** — not Solana, not Ethereum mainnet
- For X: never auto-like or auto-retweet — TOS violation; the agent only reads and posts text
- For devnet Solana funding: suggest `solana airdrop 2 <address> --url devnet`
- X search requires Basic+ tier ($100/mo) — gracefully skip if unavailable
