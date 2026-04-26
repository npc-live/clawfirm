# Solana Swap — Reference Agent Prompt

Use this as guidance for how an agent should handle swap requests when the `solana-swap` skill is installed.

## Handling a Swap Request

When a user asks to swap tokens (e.g., "swap 1 SOL for USDC", "trade 50 USDC to SOL", "buy some BONK with 0.5 SOL"):

### 1. Parse the Intent

Extract:
- **From token** — what they're selling
- **To token** — what they're buying
- **Amount** — how much they're selling

If any of these are unclear, ask. Don't guess tokens — if you don't recognize a token name, ask for the mint address.

Convert the amount to base units before calling prepare:
- SOL: × 1,000,000,000 (9 decimals)
- USDC/USDT: × 1,000,000 (6 decimals)
- BONK: × 100,000 (5 decimals)
- Other tokens: ask the user or look up decimals

### 2. Prepare the Swap

Call `prepare` with the parsed intent. If it succeeds, present the summary to the user clearly:

> **Swap Summary**
> Selling: 1 SOL
> Receiving: ~150.23 USDC (min 148.73 USDC)
> Slippage: 1%
> Price Impact: 0.01%
>
> Confirm?

**If price impact > 1%**, add a warning:
> ⚠️ Price impact is 3.2% — this is high. You'll get significantly less than market rate. Still want to proceed?

**If price impact > 5%**, strongly discourage:
> ⚠️ Price impact is 7.8% — this is very high. You'll lose a lot to slippage. Consider a smaller amount or a different pair. Proceed anyway?

Wait for explicit confirmation. Acceptable confirmations: "yes", "y", "go", "do it", "confirm", "proceed", "send it", etc.

If the user says no, cancel, or changes their mind — acknowledge and stop.

### 3. Execute the Swap

After confirmation, call `execute` with the `prepareId`.

If it returns a signature, tell the user:
> Swap submitted! Waiting for confirmation...

Then poll `status` every 2-3 seconds until the state is `confirmed`, `failed`, or `expired`.

### 4. Report the Result

On **confirmed**:
Call `receipt` and report:
> ✅ Swap complete!
> Sent: 1 SOL
> Received: 150.19 USDC
> Tx: https://solscan.io/tx/{signature}

On **failed**:
> ❌ Swap failed on-chain (likely slippage — price moved too much). No tokens were lost beyond the transaction fee.
> Want to try again with higher slippage?

On **expired** (status stays `submitted` for too long):
> ⏰ Transaction may have expired. Checking...
> If not confirmed after ~60s, suggest re-preparing.

## Error Handling

### Retryable Errors (agent can retry automatically)

| Error | Action |
|---|---|
| `BACKEND_UNAVAILABLE` | Wait 3s, retry `prepare` up to 2x. If still failing: "Jupiter is currently down. Try again in a few minutes." |
| `RPC_UNAVAILABLE` | Wait 3s, retry up to 2x. If still failing: "Solana RPC is unreachable. Try again shortly." |
| `TX_EXPIRED` | Re-run `prepare`, show new summary, re-confirm with user (prices may have changed). |
| `TX_BROADCAST_FAILED` | Retry `execute` once with same `prepareId` (if not expired). If still failing, re-prepare. |
| `PREPARE_EXPIRED` | Re-run `prepare`, show new summary, re-confirm with user. |

### Non-Retryable Errors (report to user)

| Error | Response |
|---|---|
| `INVALID_INPUT` | Report the specific issue. Ask user to fix. |
| `INSUFFICIENT_SOL` | "You don't have enough SOL. You have X, need ~Y for this swap + fees." |
| `KEYPAIR_NOT_FOUND` | "Wallet isn't configured. Set `SOLANA_KEYPAIR_PATH` to your keypair file." |
| `KEYPAIR_INVALID` | "Keypair file couldn't be read. Check it's a valid Solana keypair JSON." |
| `BACKEND_QUOTE_FAILED` | "Jupiter can't find a route for this swap. Might be no liquidity or an invalid token pair." |
| `TX_FAILED_ONCHAIN` | "Swap failed on-chain. **Do not retry automatically.** Ask user if they want to try again (maybe with higher slippage)." |
| `PREPARE_ALREADY_EXECUTED` | "This swap was already submitted. Check status instead." |

### Critical Rule: Never Auto-Retry `TX_FAILED_ONCHAIN`

This means the transaction landed on Solana and failed (e.g., slippage exceeded). The user paid a fee. Retrying sends a **new** transaction with a **new** fee. Always ask the user first.

## Edge Cases

### User says "swap all my SOL"
Don't swap literally all — they need SOL for the transaction fee. Subtract ~0.01 SOL from the amount as a buffer, and tell the user:
> Swapping 4.99 SOL (keeping ~0.01 for fees).

### User mentions a token you don't know
Ask for the mint address. Don't guess or search — wrong mint = wrong token = lost funds.

### User asks for a very small swap
Proceed normally. Jupiter may not find a route for dust amounts — if `BACKEND_QUOTE_FAILED`, explain that the amount might be too small.

### User asks about swap status later
If you have the signature from a previous swap, call `status` or `receipt`. If you don't have it, ask for the transaction signature.

### User wants to swap to a different wallet
This requires the `--allowThirdParty` flag in prepare. Confirm the destination address with the user explicitly:
> You want to send the USDC to `ABC123...`? This is a different wallet than yours. Confirm?

## What This Skill Does NOT Do

- **No best price routing** — takes Jupiter's default route
- **No limit orders** — executes immediately at market
- **No cross-chain** — Solana only
- **No wallet management** — doesn't create or import keys
- **No token discovery** — doesn't search for tokens by name
- **No portfolio tracking** — just swaps

If a user asks for any of these, let them know it's not supported and suggest alternatives if you know any.
