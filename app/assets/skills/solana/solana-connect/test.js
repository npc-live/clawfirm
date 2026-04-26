/**
 * OpenClaw Solana Connect - Test Suite v3.0
 */

const { generateWallet, connectWallet, getBalance, getTransactions, getTokenAccounts, getConnection, isTestNet, sendSol, getConfig } = require('./scripts/solana.js');

console.log('🧪 OpenClaw Solana Connect v3.0 - Test Suite\n');

let passed = 0;
let failed = 0;

// Test 1: Generate new wallet
function test1() {
  console.log('Test 1: Generate new wallet...');
  try {
    const wallet = generateWallet();
    if (wallet.address && wallet.address.length > 30) {
      console.log(`  ✅ PASSED - Address: ${wallet.address.slice(0,8)}... (private key protected)`);
      passed++;
    } else {
      console.log('  ❌ FAILED - Invalid address');
      failed++;
    }
  } catch (e) {
    console.log(`  ❌ FAILED - ${e.message}`);
    failed++;
  }
}

// Test 2: Get config
function test2() {
  console.log('Test 2: Get configuration...');
  try {
    const config = getConfig();
    if (config.isTestNet && config.maxSolPerTx && config.features.dryRun) {
      console.log(`  ✅ PASSED - Testnet: ${config.isTestNet}, Max SOL: ${config.maxSolPerTx}`);
      passed++;
    } else {
      console.log('  ❌ FAILED - Invalid config');
      failed++;
    }
  } catch (e) {
    console.log(`  ❌ FAILED - ${e.message}`);
    failed++;
  }
}

// Test 3: Get balance
async function test3() {
  console.log('Test 3: Get balance...');
  try {
    const testAddr = 'GeD4JLVBYCGYGV3dnrVRvCjKC2X4wCJMxRDrgMHTJpH';
    const balance = await getBalance(testAddr);
    if (balance.sol !== undefined) {
      console.log(`  ✅ PASSED - Balance: ${balance.sol} SOL`);
      passed++;
    } else {
      console.log('  ❌ FAILED - Invalid balance');
      failed++;
    }
  } catch (e) {
    console.log(`  ❌ FAILED - ${e.message}`);
    failed++;
  }
}

// Test 4: Get transactions
async function test4() {
  console.log('Test 4: Get transactions...');
  try {
    const testAddr = 'GeD4JLVBYCGYGV3dnrVRvCjKC2X4wCJMxRDrgMHTJpH';
    const txs = await getTransactions(testAddr, 3);
    console.log(`  ✅ PASSED - Found ${txs.length} transactions`);
    passed++;
  } catch (e) {
    console.log(`  ❌ FAILED - ${e.message}`);
    failed++;
  }
}

// Test 5: Dry-run sendSol
async function test5() {
  console.log('Test 5: Dry-run sendSol (simulation)...');
  try {
    const wallet = generateWallet();
    const result = await sendSol(wallet.address, 'GeD4JLVBYCGYGV3dnrVRvCjKC2X4wCJMxRDrgMHTJpH', 0.001, { dryRun: true });
    if (result.success && result.dryRun && result.amount === 0.001) {
      console.log(`  ✅ PASSED - Simulation: ${result.amount} SOL, sig: ${result.signature.slice(0,16)}...`);
      passed++;
    } else {
      console.log('  ❌ FAILED - Invalid result');
      failed++;
    }
  } catch (e) {
    console.log(`  ❌ FAILED - ${e.message}`);
    failed++;
  }
}

// Test 6: Security - Max limit check
async function test6() {
  console.log('Test 6: Security - Max limit enforcement...');
  try {
    const wallet = generateWallet();
    await sendSol(wallet.address, 'GeD4JLVBYCGYGV3dnrVRvCjKC2X4wCJMxRDrgMHTJpH', 100, { dryRun: true });
    console.log('  ❌ FAILED - Should have thrown error for amount > max');
    failed++;
  } catch (e) {
    if (e.message.includes('exceeds max limit')) {
      console.log(`  ✅ PASSED - Correctly rejected: ${e.message.slice(0,50)}...`);
      passed++;
    } else {
      console.log(`  ❌ FAILED - Wrong error: ${e.message}`);
      failed++;
    }
  }
}

// Run tests
(async () => {
  test1();
  test2();
  await test3();
  await test4();
  await test5();
  await test6();
  
  console.log('\n========================================');
  console.log(`📊 Results: ${passed} passed, ${failed} failed`);
  console.log('========================================\n');
  
  if (failed > 0) {
    console.log('⚠️  Some tests failed. Check errors above.');
    process.exit(1);
  } else {
    console.log('🎉 All tests passed!\n');
    process.exit(0);
  }
})();
