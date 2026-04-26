---
name: solana-connect
description: OpenClaw Solana Connect — Secure toolkit for AI agents to interact with Solana blockchain. Features private key protection, max limits, dry-run mode, and human confirmation for large transactions.
metadata:
  {
    "openclaw":
      {
        "requires":
          {
            "env": ["SOLANA_RPC_URL", "MAX_SOL_PER_TX", "MAX_TOKENS_PER_TX", "HUMAN_CONFIRMATION_THRESHOLD"],
          },
        "install":
          [
            {
              "id": "npm",
              "kind": "npm",
              "package": "@solana/web3.js",
              "label": "Install Solana Web3.js",
            },
            {
              "id": "npm",
              "kind": "npm", 
              "package": "tweetnacl",
              "label": "Install TweetNaCl",
            },
            {
              "id": "npm",
              "kind": "npm",
              "package": "bs58",
              "label": "Install bs58",
            },
          ],
      },
  }
---

# 🔗 OpenClaw Solana Connect v3.0

> Secure toolkit for AI agents to interact with Solana blockchain

## 🛡️ Security Features

- **Private Key Protection** - Keys never exposed to agent
- **Max Limits** - Configurable transaction limits
- **Dry-Run Mode** - Simulate before sending (default)
- **Human Confirmation** - Required for large transactions
- **Testnet Default** - Safe by default

## What Works

| Function | Status | Description |
|----------|--------|-------------|
| `generateWallet()` | ✅ Works | Generate wallet addresses |
| `connectWallet()` | ✅ Works | Validate wallet addresses |
| `getBalance()` | ✅ Works | Read SOL/token balances |
| `getTransactions()` | ✅ Works | Read transaction history |
| `getTokenAccounts()` | ✅ Works | Read token holdings |
| `sendSol()` | ✅ Works | Send SOL (with security) |

## Installation

```bash
clawhub install solana-connect
```

## Environment Variables

- `SOLANA_RPC_URL` - RPC endpoint (default: testnet)
- `MAX_SOL_PER_TX` - Max SOL per transaction (default: 10)
- `MAX_TOKENS_PER_TX` - Max tokens per transaction (default: 10000)
- `HUMAN_CONFIRMATION_THRESHOLD` - SOL amount requiring human confirmation (default: 1)

## Usage

```javascript
const { generateWallet, getBalance, sendSol, getConfig } = require('./scripts/solana.js');

// Generate wallet (address only - private key protected)
const wallet = generateWallet();
console.log('Address:', wallet.address);

// Check balance
const balance = await getBalance(wallet.address);

// Send SOL (DRY-RUN by default - simulation only)
const result = await sendSol(privateKey, toAddress, 0.5, { dryRun: true });
console.log('Simulation:', result);

// Send real transaction
const tx = await sendSol(privateKey, toAddress, 0.5, { dryRun: false, skipConfirmation: true });
console.log('Signature:', tx.signature);
```

## Security Options

```javascript
// Dry-run (simulation) - safe, doesn't send
await sendSol(key, to, amount, { dryRun: true });

// Real transaction - requires explicit flag
await sendSol(key, to, amount, { dryRun: false });

// Skip human confirmation (for automated agents)
await sendSol(key, to, amount, { dryRun: false, skipConfirmation: true });
```

---

**Security:** Never hardcode private keys. Use environment variables.
