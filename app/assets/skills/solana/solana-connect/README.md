# OpenClaw Solana Connect v3.0

> Secure toolkit for AI agents to interact with Solana blockchain

A purpose-built toolkit that enables autonomous AI agents running on OpenClaw to interact with the Solana blockchain **securely**.

## 🛡️ Security First

### Private Key Protection
**Private keys are NEVER exposed to the agent.**

- `generateWallet()` returns only the address
- `connectWallet()` returns only the address
- Transactions are signed internally without exposing the raw private key

### Security Features

| Feature | Description |
|---------|-------------|
| Max Limits | `MAX_SOL_PER_TX` - Prevents large losses |
| Dry-Run | Simulation by default - safe testing |
| Human Confirmation | Required for large transactions |
| Testnet Default | Safe by default |

## Installation

```bash
# Via ClawHub
clawhub install solana-connect

# Or manually
git clone https://github.com/Seenfinity/openclaw-solana-connect.git
cd openclaw-solana-connect
npm install
```

## Configuration

```bash
# RPC endpoint (testnet by default)
export SOLANA_RPC_URL=https://api.testnet.solana.com

# Max SOL per transaction (default: 10)
export MAX_SOL_PER_TX=10

# Max tokens per transaction (default: 10000)
export MAX_TOKENS_PER_TX=10000

# Human confirmation threshold in SOL (default: 1)
export HUMAN_CONFIRMATION_THRESHOLD=1
```

## Quick Start

```javascript
const { generateWallet, getBalance, sendSol, getConfig } = require('./scripts/solana.js');

// Generate wallet (private key protected)
const wallet = generateWallet();
console.log('Address:', wallet.address);

// Check balance
const balance = await getBalance(wallet.address);
console.log('SOL:', balance.sol);

// Dry-run simulation (default - safe)
const simulation = await sendSol(privateKey, toAddress, 0.5, { dryRun: true });
console.log('Simulation:', simulation);

// Send real transaction
const tx = await sendSol(privateKey, toAddress, 0.5, { dryRun: false, skipConfirmation: true });
console.log('Signature:', tx.signature);
```

## API Reference

### Wallet Functions
```javascript
generateWallet()       // Generate new wallet (returns address only)
connectWallet(privateKey) // Validate address from private key
```

### Query Functions
```javascript
getBalance(address)              // Get SOL balance
getTransactions(address, limit)  // Get transaction history
getTokenAccounts(address)        // Get token holdings
```

### Transaction Functions
```javascript
sendSol(privateKey, toAddress, amount, options)
```

Options:
- `dryRun: boolean` - Simulation only (default: true)
- `skipConfirmation: boolean` - Skip human confirmation (default: false)

## Security Usage

```javascript
// ✅ Good - Always dry-run first
const sim = await sendSol(key, to, 1.0, { dryRun: true });
if (sim.success) {
  console.log('Simulation OK, sending real tx...');
  const tx = await sendSol(key, to, 1.0, { dryRun: false });
}

// ✅ Good - Set limits
export MAX_SOL_PER_TX=5

// ✅ Good - Human confirmation for large amounts
export HUMAN_CONFIRMATION_THRESHOLD=0.5

// ❌ Bad - Never hardcode keys
const key = '4z...'; // DON'T
```

## Testing

```bash
npm install
node test.js
```

All tests pass:
- ✅ Generate wallet
- ✅ Configuration
- ✅ Get balance
- ✅ Get transactions
- ✅ Dry-run sendSol
- ✅ Max limit enforcement

## GitHub

[github.com/Seenfinity/openclaw-solana-connect](https://github.com/Seenfinity/openclaw-solana-connect)

---

MIT © 2026 Seenfinity
