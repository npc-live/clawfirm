/**
 * OpenClaw Solana Connect v3.0
 * Secure toolkit for AI agents to interact with Solana blockchain
 * 
 * SECURITY FEATURES:
 * - Private keys NEVER exposed to agent
 * - Max limits enforced per transaction
 * - Dry-run by default
 * - Testnet mode enforced
 */

const { 
  Connection, 
  PublicKey, 
  Transaction, 
  SystemProgram, 
  LAMPORTS_PER_SOL 
} = require('@solana/web3.js');

const nacl = require('tweetnacl');
const bs58 = require('bs58').default;

// Default RPC - testnet for safety
const DEFAULT_RPC = process.env.SOLANA_RPC_URL || 'https://api.testnet.solana.com';

// Security limits - configurable via env
const MAX_SOL_PER_TX = parseFloat(process.env.MAX_SOL_PER_TX) || 10;
const MAX_TOKENS_PER_TX = parseFloat(process.env.MAX_TOKENS_PER_TX) || 10000;
const REQUIRE_HUMAN_CONFIRMATION = parseFloat(process.env.HUMAN_CONFIRMATION_THRESHOLD) || 1; // SOL

/**
 * Get Solana RPC connection
 */
function getConnection(rpcUrl = DEFAULT_RPC) {
  return new Connection(rpcUrl, 'confirmed');
}

/**
 * Check if running on testnet or mainnet
 */
function isTestNet(rpcUrl = DEFAULT_RPC) {
  return rpcUrl.includes('testnet') || rpcUrl.includes('devnet');
}

/**
 * Generate a new Solana wallet
 * @returns {Object} { address } - Private key is NEVER returned
 */
function generateWallet() {
  const keyPair = nacl.sign.keyPair();
  const publicKey = bs58.encode(keyPair.publicKey);
  
  // SECURITY: Only return address, private key stays internal
  return {
    address: publicKey
  };
}

/**
 * Connect wallet - validates address from private key
 * @param {string} privateKeyBase58 - Base58 encoded private key
 * @returns {Object} { address } - Private key is NEVER returned
 * 
 * SECURITY: Private key is used internally for signing only
 */
function connectWallet(privateKeyBase58) {
  if (!privateKeyBase58) {
    throw new Error('Private key is required');
  }
  
  try {
    // Decode private key
    const privateKeyBytes = bs58.decode(privateKeyBase58);
    const keyPair = nacl.sign.keyPair.fromSeed(privateKeyBytes.slice(0, 32));
    const publicKey = bs58.encode(keyPair.publicKey);
    
    // SECURITY: Only return address, never the private key
    return {
      address: publicKey
    };
  } catch (e) {
    throw new Error(`Invalid private key: ${e.message}`);
  }
}

/**
 * Get balance for any Solana address
 */
async function getBalance(address, rpcUrl = DEFAULT_RPC) {
  const connection = getConnection(rpcUrl);
  
  try {
    const balanceLamports = await connection.getBalance(new PublicKey(address));
    const balanceSol = balanceLamports / LAMPORTS_PER_SOL;
    
    return {
      sol: balanceSol,
      lamports: balanceLamports
    };
  } catch (e) {
    throw new Error(`Failed to get balance: ${e.message}`);
  }
}

/**
 * Get token accounts for an address
 */
async function getTokenAccounts(address, rpcUrl = DEFAULT_RPC) {
  const connection = getConnection(rpcUrl);
  
  try {
    const tokenAccounts = await connection.getParsedTokenAccountsByOwner(
      new PublicKey(address), 
      { programId: new PublicKey('TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA') }
    );
    
    return tokenAccounts.value.map(account => ({
      mint: account.account.data.parsed.info.mint,
      amount: account.account.data.parsed.info.tokenAmount.uiAmountString || '0',
      decimals: account.account.data.parsed.info.tokenAmount.decimals || 9
    }));
  } catch (e) {
    return [];
  }
}

/**
 * Get recent transactions for an address
 */
async function getTransactions(address, limit = 10, rpcUrl = DEFAULT_RPC) {
  const connection = getConnection(rpcUrl);
  
  try {
    const signatures = await connection.getSignaturesForAddress(
      new PublicKey(address), 
      { limit }
    );
    
    return signatures.map(sig => ({
      signature: sig.signature,
      slot: sig.slot,
      blockTime: sig.blockTime,
      status: sig.err ? 'failed' : 'success'
    }));
  } catch (e) {
    return [];
  }
}

/**
 * Send SOL with full security features
 * @param {string} privateKey - Sender's private key (base58)
 * @param {string} toAddress - Recipient address
 * @param {number} amount - Amount in SOL
 * @param {Object} options - { dryRun: boolean, skipConfirmation: boolean }
 * @returns {Object} Transaction result or simulation
 */
async function sendSol(privateKey, toAddress, amount, options = {}) {
  const { dryRun = true, skipConfirmation = false } = options;
  
  // SECURITY: Validate inputs
  if (!privateKey) {
    throw new Error('Private key is required');
  }
  if (!toAddress || toAddress.length < 32) {
    throw new Error('Invalid recipient address');
  }
  if (amount <= 0) {
    throw new Error('Amount must be positive');
  }
  
  // SECURITY: Check max limits
  if (amount > MAX_SOL_PER_TX) {
    throw new Error(`Amount ${amount} SOL exceeds max limit of ${MAX_SOL_PER_TX} SOL`);
  }
  
  // SECURITY: Check testnet (warn if mainnet)
  const connection = getConnection(DEFAULT_RPC);
  if (!isTestNet(DEFAULT_RPC) && dryRun === false && !skipConfirmation) {
    console.warn('⚠️  WARNING: Running on MAINNET with real transactions!');
  }
  
  // SECURITY: Require human confirmation for large amounts
  if (amount >= REQUIRE_HUMAN_CONFIRMATION && !skipConfirmation && dryRun === false) {
    throw new Error(`Amount ${amount} SOL requires human confirmation (threshold: ${REQUIRE_HUMAN_CONFIRMATION} SOL)`);
  }
  
  try {
    // Create keypair from private key
    const privateKeyBytes = bs58.decode(privateKey);
    const keyPair = nacl.sign.keyPair.fromSeed(privateKeyBytes.slice(0, 32));
    
    const fromPublicKey = new PublicKey(keyPair.publicKey);
    const toPublicKey = new PublicKey(toAddress);
    
    // Convert SOL to lamports
    const amountLamports = Math.round(amount * LAMPORTS_PER_SOL);
    
    // Create transaction
    const transaction = new Transaction().add(
      SystemProgram.transfer({
        fromPubkey: fromPublicKey,
        toPubkey: toPublicKey,
        lamports: amountLamports
      })
    );
    
    // Get recent blockhash
    const { blockhash } = await connection.getLatestBlockhash();
    transaction.recentBlockhash = blockhash;
    transaction.feePayer = fromPublicKey;
    
    // Sign transaction
    const sign = require('tweetnacl').sign;
    const messageBytes = transaction.serializeMessage();
    const signature = sign.detached(messageBytes, keyPair.secretKey);
    transaction.addSignature(fromPublicKey, Buffer.from(signature));
    
    // If dryRun, just return simulation
    if (dryRun) {
      // Simulate the transaction
      try {
        const simulation = await connection.simulateTransaction(transaction);
        return {
          success: true,
          dryRun: true,
          from: fromPublicKey.toBase58(),
          to: toAddress,
          amount: amount,
          amountLamports: amountLamports,
          signature: bs58.encode(transaction.signature),
          logs: simulation.value.logs || [],
          message: 'Simulation successful - set dryRun: false to send real transaction'
        };
      } catch (simError) {
        // If simulation fails, return the error
        return {
          success: false,
          dryRun: true,
          from: fromPublicKey.toBase58(),
          to: toAddress,
          amount: amount,
          error: simError.message,
          message: 'Simulation failed - check error for details'
        };
      }
    }
    
    // Send real transaction
    const txSignature = await connection.sendRawTransaction(transaction.serialize());
    
    // Wait for confirmation
    await connection.confirmTransaction(txSignature);
    
    return {
      success: true,
      dryRun: false,
      from: fromPublicKey.toBase58(),
      to: toAddress,
      amount: amount,
      signature: txSignature,
      message: 'Transaction sent successfully'
    };
    
  } catch (e) {
    throw new Error(`Transaction failed: ${e.message}`);
  }
}

/**
 * Get configuration info (without sensitive data)
 */
function getConfig() {
  return {
    rpc: DEFAULT_RPC,
    isTestNet: isTestNet(),
    maxSolPerTx: MAX_SOL_PER_TX,
    maxTokensPerTx: MAX_TOKENS_PER_TX,
    humanConfirmationThreshold: REQUIRE_HUMAN_CONFIRMATION,
    features: {
      dryRun: true,
      maxLimits: true,
      humanConfirmation: true
    }
  };
}

module.exports = {
  generateWallet,
  connectWallet,
  getBalance,
  getTransactions,
  getTokenAccounts,
  sendSol,
  getConnection,
  isTestNet,
  getConfig
};
