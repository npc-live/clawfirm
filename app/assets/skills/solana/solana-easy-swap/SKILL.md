---
name: solana-easy-swap
description: Swap any Solana token from chat. Say 'swap 1 SOL for USDC' and it handles everything — quoting, signing, sending, confirming. No API keys, no wallet extensions, no setup beyond a keypair. Powered by Jupiter. Use when a user wants to swap, trade, exchange, buy, or sell Solana SPL tokens, SOL, USDC, memecoins, or any token pair on Solana.
metadata: { "openclaw": { "emoji": "🔄", "requires": { "bins": ["node"], "env": ["SOLANA_KEYPAIR_PATH"] }, "install": [{ "id": "npm", "kind": "command", "command": "cd {baseDir} && npm install --production", "label": "Install dependencies" }] } }
---

# Solana Easy Swap

Swap any Solana token from chat. Say "swap 1 SOL for USDC" and it handles everything — quoting, signing, sending, confirming. No API keys, no wallet extensions, no setup beyond a keypair. Powered by Jupiter.

## Setup

**First run:** Install dependencies (automatic if install spec is supported, otherwise manual):
```bash
cd {baseDir} && npm install --production
```

Required env var:
- `SOLANA_KEYPAIR_PATH` — path to a Solana keypair JSON file (standard `solana-keygen` format). **This skill reads your keypair to sign transactions.** Only use with a keypair you trust this skill to access.

Optional env vars:
- `SOLANA_RPC_URL` — custom RPC endpoint (default: `https://api.mainnet-beta.solana.com`)
- `OSS_DEFAULT_SLIPPAGE_BPS` — default slippage in basis points (default: `100` = 1%)
- `OSS_PRIORITY_FEE_FLOOR` — minimum priority fee in lamports (default: `50000`)

No API keys required. Jupiter is used unauthenticated.

## Common Token Mints

| Token | Mint |
|---|---|
| SOL (wrapped) | `So11111111111111111111111111111111111111112` |
| USDC | `EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v` |
| USDT | `Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB` |

For other tokens, ask the user for the mint address.

## Flow

### 1. Prepare

```bash
node {baseDir}/scripts/swap.mjs prepare \
  --from So11111111111111111111111111111111111111112 \
  --to EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v \
  --amount 1000000000 \
  --slippage 100
```

Returns JSON:
```json
{
  "prepareId": "abc123",
  "expectedOut": "150230000",
  "minOut": "148727700",
  "priceImpact": "0.01",
  "expiresAt": "2025-02-13T20:00:00Z",
  "summary": {
    "from": "1 SOL",
    "to": "~150.23 USDC",
    "minReceived": "148.73 USDC",
    "slippage": "1%",
    "priceImpact": "0.01%",
    "destination": "owner"
  }
}
```

**Always show the summary to the user and wait for confirmation before executing.**

If `priceImpact` > 1%, warn the user explicitly.

### 2. Execute

After user confirms:

```bash
node {baseDir}/scripts/swap.mjs execute --prepareId abc123
```

Returns JSON:
```json
{
  "signature": "5UzV...",
  "submittedAt": "2025-02-13T19:58:12Z"
}
```

### 3. Status (poll until confirmed)

```bash
node {baseDir}/scripts/swap.mjs status --signature 5UzV...
```

Returns JSON:
```json
{
  "state": "confirmed",
  "slot": 123456789,
  "confirmationStatus": "finalized"
}
```

States: `submitted` → `confirmed` | `failed` | `expired` | `unknown`

### 4. Receipt

```bash
node {baseDir}/scripts/swap.mjs receipt --signature 5UzV...
```

Returns JSON with actual amounts swapped, fees, and a Solscan link.

## Error Handling

All commands return JSON with `error` field on failure:

```json
{
  "error": {
    "code": "INSUFFICIENT_SOL",
    "message": "Not enough SOL for fees. Have 0.001, need ~0.006",
    "retryable": false
  }
}
```

Error codes and retry guidance:

| Code | Retry? | Action |
|---|---|---|
| `INVALID_INPUT` | No | Fix the input |
| `INSUFFICIENT_SOL` | No | Tell user they need more SOL |
| `KEYPAIR_NOT_FOUND` | No | Check `SOLANA_KEYPAIR_PATH` is set |
| `KEYPAIR_INVALID` | No | Check keypair file format |
| `PREPARE_EXPIRED` | Yes | Run `prepare` again, re-confirm with user |
| `PREPARE_ALREADY_EXECUTED` | No | This swap was already sent |
| `BACKEND_UNAVAILABLE` | Yes | Wait 3s, retry prepare up to 2x |
| `BACKEND_QUOTE_FAILED` | No | No route — tell user (bad pair or no liquidity) |
| `TX_EXPIRED` | Yes | Run `prepare` again, re-confirm with user |
| `TX_BROADCAST_FAILED` | Yes | Retry execute once (if not expired) |
| `TX_FAILED_ONCHAIN` | No | Swap failed (e.g., slippage). Do NOT retry. |
| `RPC_UNAVAILABLE` | Yes | Wait 3s, retry up to 2x |

## Agent Guidelines

1. **Always confirm before executing.** Show the user the summary from `prepare` and wait for explicit "yes" / "go" / "confirm".
2. **Never auto-retry failed onchain transactions.** If `TX_FAILED_ONCHAIN`, the tx landed and failed — retrying sends a new tx.
3. **Re-confirm on re-prepare.** If you need to prepare again (expired quote), show the new summary — prices may have changed.
4. **Handle amounts in base units.** SOL = 9 decimals (1 SOL = 1000000000), USDC = 6 decimals (1 USDC = 1000000).
5. **Ask for mint addresses** if the user mentions a token you don't recognize. Don't guess.
6. **Report the Solscan link** after confirmation: `https://solscan.io/tx/{signature}`

## Security

- This skill signs transactions using the configured keypair. It does NOT create, import, or manage keys.
- Keypair material is never logged, echoed, or included in any output.
- Third-party destinations require explicit `--allowThirdParty` flag.
- All swaps enforce slippage protection via `minOut`.
- Prepared swaps expire after 120 seconds by default.

## Limitations (v1)

- Jupiter unauthenticated API — Token2022 and pump.fun tokens may not work.
- No best-price routing — takes whatever Jupiter returns.
- Receipt amounts are best-effort (derived from pre/post balance diffs).
- Solana mainnet only.
