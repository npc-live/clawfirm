#!/usr/bin/env node
// OpenClaw Solana Swap Skill (OSS) — CLI entrypoint
// Usage: node swap.mjs <prepare|execute|status|receipt> [options]

import { readFileSync, writeFileSync, existsSync, mkdirSync } from 'node:fs';
import { randomUUID } from 'node:crypto';
import { Connection, Keypair, VersionedTransaction, PublicKey, LAMPORTS_PER_SOL } from '@solana/web3.js';
import bs58 from 'bs58';

// ── Config ──────────────────────────────────────────────────────────────────

const RPC_URL = process.env.SOLANA_RPC_URL || 'https://api.mainnet-beta.solana.com';
const DEFAULT_SLIPPAGE = parseInt(process.env.OSS_DEFAULT_SLIPPAGE_BPS || '100', 10);
const PRIORITY_FEE_FLOOR = parseInt(process.env.OSS_PRIORITY_FEE_FLOOR || '50000', 10);
const JUPITER_BASE = 'https://lite-api.jup.ag/swap/v1';

const KNOWN_MINTS = {
  'So11111111111111111111111111111111111111112': { symbol: 'SOL', decimals: 9 },
  'EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v': { symbol: 'USDC', decimals: 6 },
  'Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB': { symbol: 'USDT', decimals: 6 },
  'DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263': { symbol: 'BONK', decimals: 5 },
};

const SOL_MINT = 'So11111111111111111111111111111111111111112';

// ── Helpers ─────────────────────────────────────────────────────────────────

function output(obj) {
  process.stdout.write(JSON.stringify(obj, null, 2) + '\n');
}

function fail(code, message, retryable = false, details) {
  output({ error: { code, message, retryable, ...(details ? { details } : {}) } });
  process.exit(1);
}

function parseArgs(argv) {
  const args = {};
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a.startsWith('--')) {
      const key = a.slice(2);
      const next = argv[i + 1];
      if (!next || next.startsWith('--')) {
        args[key] = true; // flag
      } else {
        args[key] = next;
        i++;
      }
    }
  }
  return args;
}

function validatePubkey(value, label) {
  try {
    const bytes = bs58.decode(value);
    if (bytes.length !== 32) throw new Error('not 32 bytes');
    return new PublicKey(value);
  } catch {
    fail('INVALID_INPUT', `Invalid ${label}: not a valid base58 pubkey`);
  }
}

function loadKeypair() {
  const kpPath = process.env.SOLANA_KEYPAIR_PATH;
  if (!kpPath) fail('KEYPAIR_NOT_FOUND', 'SOLANA_KEYPAIR_PATH env var is not set');
  if (!existsSync(kpPath)) fail('KEYPAIR_NOT_FOUND', `Keypair file not found: ${kpPath}`);
  try {
    const raw = readFileSync(kpPath, 'utf-8');
    return Keypair.fromSecretKey(Uint8Array.from(JSON.parse(raw)));
  } catch (e) {
    fail('KEYPAIR_INVALID', 'Could not parse keypair file');
  }
}

function ownerPubkey() {
  const kpPath = process.env.SOLANA_KEYPAIR_PATH;
  if (!kpPath) fail('KEYPAIR_NOT_FOUND', 'SOLANA_KEYPAIR_PATH env var is not set');
  if (!existsSync(kpPath)) fail('KEYPAIR_NOT_FOUND', `Keypair file not found: ${kpPath}`);
  try {
    const raw = readFileSync(kpPath, 'utf-8');
    return Keypair.fromSecretKey(Uint8Array.from(JSON.parse(raw))).publicKey;
  } catch {
    fail('KEYPAIR_INVALID', 'Could not parse keypair file');
  }
}

function prepareFilePath(id) {
  const cacheDir = new URL('../.cache', import.meta.url).pathname;
  if (!existsSync(cacheDir)) mkdirSync(cacheDir, { recursive: true });
  return `${cacheDir}/prepare-${id}.json`;
}

function formatAmount(amount, mint) {
  const info = KNOWN_MINTS[mint];
  if (!info) return amount;
  const decimals = info.decimals;
  const raw = BigInt(amount);
  const divisor = BigInt(10 ** decimals);
  const whole = raw / divisor;
  const frac = raw % divisor;
  const fracStr = frac.toString().padStart(decimals, '0');
  // Show enough decimal places: at least 2, but more if needed to show non-zero digits
  let sigDigits = 2;
  if (whole === 0n) {
    // Find first non-zero digit in fraction
    const firstNonZero = fracStr.search(/[1-9]/);
    if (firstNonZero >= 0) sigDigits = Math.max(sigDigits, firstNonZero + 2);
  }
  const truncated = fracStr.slice(0, sigDigits);
  return `${whole}.${truncated} ${info.symbol}`;
}

function getConnection() {
  return new Connection(RPC_URL, 'confirmed');
}

// ── Commands ────────────────────────────────────────────────────────────────

async function cmdPrepare(args) {
  // Validate required inputs
  if (!args.from) fail('INVALID_INPUT', 'Missing --from (input mint)');
  if (!args.to) fail('INVALID_INPUT', 'Missing --to (output mint)');
  if (!args.amount) fail('INVALID_INPUT', 'Missing --amount (base units)');

  const fromMint = args.from;
  const toMint = args.to;
  validatePubkey(fromMint, 'from mint');
  validatePubkey(toMint, 'to mint');

  const amountIn = args.amount;
  if (!/^\d+$/.test(amountIn) || BigInt(amountIn) <= 0n) {
    fail('INVALID_INPUT', 'Amount must be a positive integer (base units)');
  }

  const slippage = parseInt(args.slippage || String(DEFAULT_SLIPPAGE), 10);
  if (isNaN(slippage) || slippage < 1 || slippage > 2000) {
    fail('INVALID_INPUT', 'Slippage must be 1–2000 bps');
  }

  const priorityFee = args.priorityFee ? parseInt(args.priorityFee, 10) : PRIORITY_FEE_FLOOR;
  if (isNaN(priorityFee) || priorityFee < 0) {
    fail('INVALID_INPUT', 'Priority fee must be a non-negative integer');
  }
  const maxLamports = Math.max(priorityFee, PRIORITY_FEE_FLOOR);

  const expirySeconds = parseInt(args.expiry || '120', 10);
  if (isNaN(expirySeconds) || expirySeconds < 30 || expirySeconds > 300) {
    fail('INVALID_INPUT', 'Expiry must be 30–300 seconds');
  }

  let destination = null;
  if (args.destination) {
    if (!args.allowThirdParty) {
      fail('INVALID_INPUT', 'Third-party destination requires --allowThirdParty flag');
    }
    destination = validatePubkey(args.destination, 'destination');
  }

  // Derive owner
  const owner = ownerPubkey();
  const conn = getConnection();

  // Pre-flight SOL balance check
  try {
    const balance = await conn.getBalance(owner);
    const estimatedFee = 5000 + maxLamports; // base fee + priority
    let needed = estimatedFee;
    if (fromMint === SOL_MINT) needed += Number(BigInt(amountIn));

    if (balance < needed) {
      fail('INSUFFICIENT_SOL', `Not enough SOL for fees. Have ${(balance / LAMPORTS_PER_SOL).toFixed(4)} SOL, need ~${(needed / LAMPORTS_PER_SOL).toFixed(4)} SOL`, false, {
        have: balance,
        need: needed,
      });
    }
  } catch (e) {
    if (e.code) throw e; // re-throw structured errors
    fail('RPC_UNAVAILABLE', `RPC error checking balance: ${e.message}`, true);
  }

  // Jupiter quote
  let quoteResponse;
  try {
    const quoteUrl = `${JUPITER_BASE}/quote?inputMint=${fromMint}&outputMint=${toMint}&amount=${amountIn}&slippageBps=${slippage}`;
    const res = await fetch(quoteUrl);
    if (!res.ok) {
      const body = await res.text();
      if (res.status >= 500) fail('BACKEND_UNAVAILABLE', `Jupiter quote error: ${res.status}`, true);
      fail('BACKEND_QUOTE_FAILED', `Jupiter quote failed: ${res.status} ${body}`);
    }
    quoteResponse = await res.json();
  } catch (e) {
    if (e.code) throw e;
    fail('BACKEND_UNAVAILABLE', `Jupiter unreachable: ${e.message}`, true);
  }

  // Jupiter swap (build tx)
  let swapTxBase64;
  try {
    const swapBody = {
      quoteResponse,
      userPublicKey: owner.toBase58(),
      dynamicComputeUnitLimit: true,
      prioritizationFeeLamports: {
        priorityLevelWithMaxLamports: {
          maxLamports,
          priorityLevel: 'high',
        },
      },
    };
    if (destination) swapBody.destinationTokenAccount = destination.toBase58();

    const res = await fetch(`${JUPITER_BASE}/swap`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(swapBody),
    });
    if (!res.ok) {
      const body = await res.text();
      if (res.status >= 500) fail('BACKEND_UNAVAILABLE', `Jupiter swap build error: ${res.status}`, true);
      fail('BACKEND_QUOTE_FAILED', `Jupiter swap build failed: ${res.status} ${body}`);
    }
    const swapData = await res.json();
    swapTxBase64 = swapData.swapTransaction;
    if (!swapTxBase64) fail('BACKEND_QUOTE_FAILED', 'Jupiter returned no swap transaction');
  } catch (e) {
    if (e.code) throw e;
    fail('BACKEND_UNAVAILABLE', `Jupiter swap build unreachable: ${e.message}`, true);
  }

  // Store prepared swap
  const prepareId = randomUUID();
  const expiresAt = new Date(Date.now() + expirySeconds * 1000).toISOString();

  const prepared = {
    prepareId,
    txBase64: swapTxBase64,
    fromMint,
    toMint,
    amountIn,
    slippage,
    owner: owner.toBase58(),
    destination: destination ? destination.toBase58() : owner.toBase58(),
    expiresAt,
    executed: false,
    expectedOut: quoteResponse.outAmount || null,
    minOut: quoteResponse.otherAmountThreshold || null,
    priceImpact: quoteResponse.priceImpactPct || null,
  };

  writeFileSync(prepareFilePath(prepareId), JSON.stringify(prepared));

  // Build summary
  const summary = {
    from: formatAmount(amountIn, fromMint),
    to: prepared.expectedOut ? `~${formatAmount(prepared.expectedOut, toMint)}` : 'unknown',
    minReceived: prepared.minOut ? formatAmount(prepared.minOut, toMint) : 'unknown',
    slippage: `${slippage / 100}%`,
    priceImpact: prepared.priceImpact ? `${prepared.priceImpact}%` : 'unknown',
    destination: destination ? destination.toBase58() : 'owner',
  };

  output({
    prepareId,
    expectedOut: prepared.expectedOut,
    minOut: prepared.minOut,
    priceImpact: prepared.priceImpact,
    expiresAt,
    summary,
  });
}

async function cmdExecute(args) {
  if (!args.prepareId) fail('INVALID_INPUT', 'Missing --prepareId');

  const filePath = prepareFilePath(args.prepareId);
  if (!existsSync(filePath)) fail('INVALID_INPUT', `Unknown prepareId: ${args.prepareId}`);

  const prepared = JSON.parse(readFileSync(filePath, 'utf-8'));

  if (prepared.executed) {
    fail('PREPARE_ALREADY_EXECUTED', 'This swap has already been executed');
  }

  if (new Date(prepared.expiresAt) < new Date()) {
    fail('PREPARE_EXPIRED', 'This prepared swap has expired — run prepare again', true);
  }

  // Sign
  const keypair = loadKeypair();
  let signedTx;
  try {
    const txBuf = Buffer.from(prepared.txBase64, 'base64');
    const tx = VersionedTransaction.deserialize(txBuf);
    tx.sign([keypair]);
    signedTx = tx.serialize();
  } catch (e) {
    fail('TX_BROADCAST_FAILED', `Failed to deserialize/sign transaction: ${e.message}`, true);
  }

  // Send
  const conn = getConnection();
  let signature;
  try {
    signature = await conn.sendRawTransaction(signedTx, { skipPreflight: false });
  } catch (e) {
    if (e.message && e.message.includes('block height exceeded')) {
      fail('TX_EXPIRED', 'Transaction blockhash expired — run prepare again', true);
    }
    fail('TX_BROADCAST_FAILED', `Failed to broadcast: ${e.message}`, true);
  }

  // Mark executed
  prepared.executed = true;
  prepared.signature = signature;
  writeFileSync(filePath, JSON.stringify(prepared));

  output({
    signature,
    submittedAt: new Date().toISOString(),
  });
}

async function cmdStatus(args) {
  if (!args.signature) fail('INVALID_INPUT', 'Missing --signature');

  const conn = getConnection();
  try {
    const res = await conn.getSignatureStatuses([args.signature]);
    const status = res?.value?.[0];

    if (!status) {
      output({ state: 'unknown', signature: args.signature });
      return;
    }

    if (status.err) {
      output({
        state: 'failed',
        signature: args.signature,
        slot: status.slot,
        confirmationStatus: status.confirmationStatus,
        err: status.err,
      });
      return;
    }

    const state = status.confirmationStatus === 'finalized' || status.confirmationStatus === 'confirmed'
      ? 'confirmed' : 'submitted';

    output({
      state,
      signature: args.signature,
      slot: status.slot,
      confirmationStatus: status.confirmationStatus,
    });
  } catch (e) {
    fail('RPC_UNAVAILABLE', `RPC error: ${e.message}`, true);
  }
}

async function cmdReceipt(args) {
  if (!args.signature) fail('INVALID_INPUT', 'Missing --signature');

  const conn = getConnection();
  let txInfo;
  try {
    txInfo = await conn.getTransaction(args.signature, {
      maxSupportedTransactionVersion: 0,
    });
  } catch (e) {
    fail('RPC_UNAVAILABLE', `RPC error: ${e.message}`, true);
  }

  if (!txInfo) {
    fail('INVALID_INPUT', 'Transaction not found — it may not be confirmed yet', true);
  }

  if (txInfo.meta?.err) {
    fail('TX_FAILED_ONCHAIN', `Transaction failed onchain: ${JSON.stringify(txInfo.meta.err)}`);
  }

  // Parse token balance diffs — filter to signer's accounts only
  const pre = txInfo.meta?.preTokenBalances || [];
  const post = txInfo.meta?.postTokenBalances || [];

  // The signer is account index 0; get their pubkey from accountKeys
  const signerPubkey = txInfo.transaction?.message?.staticAccountKeys?.[0]?.toBase58?.()
    || txInfo.transaction?.message?.accountKeys?.[0]?.toBase58?.();

  // Build balance map keyed by mint, filtered to signer's token accounts
  const balances = {};
  for (const b of pre) {
    if (b.owner !== signerPubkey) continue;
    const key = b.mint;
    if (!balances[key]) balances[key] = { pre: '0', post: '0' };
    balances[key].pre = b.uiTokenAmount?.amount || '0';
  }
  for (const b of post) {
    if (b.owner !== signerPubkey) continue;
    const key = b.mint;
    if (!balances[key]) balances[key] = { pre: '0', post: '0' };
    balances[key].post = b.uiTokenAmount?.amount || '0';
  }

  // Find the token that decreased (input) and increased (output) for the signer
  let inputMint = null, outputMint = null;
  let amountInActual = null, amountOutActual = null;

  for (const [mint, bal] of Object.entries(balances)) {
    const diff = BigInt(bal.post) - BigInt(bal.pre);
    if (diff < 0n) {
      inputMint = mint;
      amountInActual = (diff * -1n).toString();
    } else if (diff > 0n) {
      outputMint = mint;
      amountOutActual = diff.toString();
    }
  }

  // SOL balance diff for fee / SOL swaps
  const preSol = txInfo.meta?.preBalances?.[0] || 0;
  const postSol = txInfo.meta?.postBalances?.[0] || 0;
  const solDiff = postSol - preSol;
  const feeLamports = (txInfo.meta?.fee || 0).toString();

  // If no token input found but SOL decreased, it's a SOL→token swap
  if (!inputMint && solDiff < 0) {
    inputMint = SOL_MINT;
    amountInActual = (Math.abs(solDiff) - Number(feeLamports)).toString();
  }
  if (!outputMint && solDiff > 0) {
    outputMint = SOL_MINT;
    amountOutActual = solDiff.toString();
  }

  output({
    signature: args.signature,
    confirmedAt: txInfo.blockTime ? new Date(txInfo.blockTime * 1000).toISOString() : null,
    inputMint,
    outputMint,
    amountInActual,
    amountOutActual,
    feeLamports,
    solscanUrl: `https://solscan.io/tx/${args.signature}`,
  });
}

// ── Main ────────────────────────────────────────────────────────────────────

const [command, ...rest] = process.argv.slice(2);
const args = parseArgs(rest);

switch (command) {
  case 'prepare': await cmdPrepare(args); break;
  case 'execute': await cmdExecute(args); break;
  case 'status':  await cmdStatus(args); break;
  case 'receipt': await cmdReceipt(args); break;
  default:
    output({
      usage: 'node swap.mjs <prepare|execute|status|receipt> [options]',
      commands: {
        prepare: '--from <mint> --to <mint> --amount <baseUnits> [--slippage <bps>] [--priorityFee <lamports>] [--destination <pubkey> --allowThirdParty] [--expiry <seconds>]',
        execute: '--prepareId <id>',
        status: '--signature <sig>',
        receipt: '--signature <sig>',
      },
    });
}
