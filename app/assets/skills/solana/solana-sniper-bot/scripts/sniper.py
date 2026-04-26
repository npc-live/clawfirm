#!/usr/bin/env python3
"""Solana Token Sniper — monitors Raydium pools, evaluates with LLM, auto-trades via Jupiter."""
import os, sys, json, time, logging, asyncio
from datetime import datetime, timezone
from pathlib import Path
from dotenv import load_dotenv
import httpx

load_dotenv()
logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
log = logging.getLogger("sniper")

PRIVATE_KEY = os.environ["SOLANA_PRIVATE_KEY"]
LLM_API_KEY = os.environ["LLM_API_KEY"]
RPC_URL = os.getenv("RPC_URL", "https://api.mainnet-beta.solana.com")
HELIUS_KEY = os.getenv("HELIUS_API_KEY", "")
BUY_AMOUNT = float(os.getenv("BUY_AMOUNT_SOL", "0.1"))
TAKE_PROFIT = float(os.getenv("TAKE_PROFIT", "2.0"))
STOP_LOSS = float(os.getenv("STOP_LOSS", "0.5"))
MAX_POSITIONS = int(os.getenv("MAX_POSITIONS", "5"))
MIN_LIQUIDITY = float(os.getenv("MIN_LIQUIDITY", "5000"))
SLIPPAGE_BPS = int(os.getenv("SLIPPAGE_BPS", "500"))
RISK_THRESHOLD = int(os.getenv("RISK_THRESHOLD", "40"))

JUPITER_QUOTE = "https://quote-api.jup.ag/v6/quote"
JUPITER_SWAP = "https://quote-api.jup.ag/v6/swap"
SOL_MINT = "So11111111111111111111111111111111111111112"
USDC_MINT = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"

TRADES_LOG = Path("trades.jsonl")
positions = {}

def get_rpc_url():
    if HELIUS_KEY:
        return f"https://mainnet.helius-rpc.com/?api-key={HELIUS_KEY}"
    return RPC_URL

async def get_token_info(mint: str) -> dict:
    """Fetch token metadata for risk assessment."""
    async with httpx.AsyncClient(timeout=15) as client:
        # Get mint account info
        rpc = get_rpc_url()
        resp = await client.post(rpc, json={
            "jsonrpc": "2.0", "id": 1, "method": "getAccountInfo",
            "params": [mint, {"encoding": "jsonParsed"}]
        })
        data = resp.json()
        info = {"mint": mint, "mint_authority": None, "freeze_authority": None, "supply": 0, "decimals": 0}
        if data.get("result", {}).get("value"):
            parsed = data["result"]["value"]["data"].get("parsed", {}).get("info", {})
            info["mint_authority"] = parsed.get("mintAuthority")
            info["freeze_authority"] = parsed.get("freezeAuthority")
            info["supply"] = int(parsed.get("supply", 0))
            info["decimals"] = parsed.get("decimals", 0)

        # Get largest holders
        resp2 = await client.post(rpc, json={
            "jsonrpc": "2.0", "id": 2, "method": "getTokenLargestAccounts",
            "params": [mint]
        })
        holders = resp2.json().get("result", {}).get("value", [])
        info["top_holders"] = holders[:10]
        total = sum(float(h.get("uiAmount", 0) or 0) for h in holders)
        top10 = sum(float(h.get("uiAmount", 0) or 0) for h in holders[:10])
        info["top10_concentration"] = (top10 / total * 100) if total > 0 else 100

        return info

def assess_risk(token_info: dict) -> int:
    """Score token risk 0-100 (lower = safer)."""
    score = 0
    # Mint authority not revoked = risky
    if token_info.get("mint_authority"): score += 25
    # Freeze authority not revoked = risky
    if token_info.get("freeze_authority"): score += 20
    # High concentration
    conc = token_info.get("top10_concentration", 100)
    if conc > 80: score += 15
    elif conc > 50: score += 10
    elif conc > 30: score += 5
    return score

async def llm_evaluate(token_info: dict, pool_info: dict) -> tuple:
    """Ask LLM for rugpull risk assessment."""
    prompt = f"""Evaluate this new Solana token for rugpull risk (0.0=safe, 1.0=scam):

Mint: {token_info['mint']}
Mint authority revoked: {token_info['mint_authority'] is None}
Freeze authority revoked: {token_info['freeze_authority'] is None}
Top 10 holder concentration: {token_info.get('top10_concentration', 'unknown'):.1f}%
Pool liquidity: ${pool_info.get('liquidity_usd', 0):.0f}

Reply with ONLY a number 0.0 to 1.0."""

    async with httpx.AsyncClient(timeout=30) as client:
        resp = await client.post("https://api.anthropic.com/v1/messages",
            headers={"x-api-key": LLM_API_KEY, "anthropic-version": "2023-06-01", "Content-Type": "application/json"},
            json={"model": "claude-3-5-haiku-20241022", "max_tokens": 20, "messages": [{"role": "user", "content": prompt}]})
        data = resp.json()
        text = data.get("content", [{}])[0].get("text", "1.0").strip()
        try:
            risk = float(text.split()[0].strip(".,"))
            return min(max(risk, 0), 1), data.get("usage", {})
        except:
            return 1.0, data.get("usage", {})

async def get_jupiter_quote(input_mint: str, output_mint: str, amount: int) -> dict:
    """Get swap quote from Jupiter."""
    async with httpx.AsyncClient(timeout=15) as client:
        resp = await client.get(JUPITER_QUOTE, params={
            "inputMint": input_mint, "outputMint": output_mint,
            "amount": str(amount), "slippageBps": str(SLIPPAGE_BPS)
        })
        return resp.json()

async def execute_swap(quote: dict) -> dict:
    """Execute swap via Jupiter."""
    from solders.keypair import Keypair
    import base58
    keypair = Keypair.from_bytes(base58.b58decode(PRIVATE_KEY))

    async with httpx.AsyncClient(timeout=30) as client:
        resp = await client.post(JUPITER_SWAP, json={
            "quoteResponse": quote,
            "userPublicKey": str(keypair.pubkey()),
            "wrapAndUnwrapSol": True
        })
        swap_data = resp.json()

        # Sign and send transaction
        from solders.transaction import VersionedTransaction
        import base64
        tx_bytes = base64.b64decode(swap_data["swapTransaction"])
        tx = VersionedTransaction.from_bytes(tx_bytes)
        signed = keypair.sign_message(tx.message.serialize())

        rpc = get_rpc_url()
        send_resp = await client.post(rpc, json={
            "jsonrpc": "2.0", "id": 1, "method": "sendTransaction",
            "params": [base64.b64encode(bytes(tx)).decode(), {"skipPreflight": True}]
        })
        return send_resp.json()

async def monitor_new_pools():
    """Monitor Raydium for new pool creation."""
    log.info("Starting Solana sniper bot...")
    log.info(f"Config: buy={BUY_AMOUNT} SOL, TP={TAKE_PROFIT}x, SL={STOP_LOSS}x, max_pos={MAX_POSITIONS}")

    # Poll Raydium new pools API
    seen_pools = set()
    while True:
        try:
            async with httpx.AsyncClient(timeout=15) as client:
                # Check for new Raydium AMM pools
                resp = await client.get("https://api.raydium.io/v2/ammV3/ammPools")
                pools = resp.json().get("data", [])

                for pool in pools[-20:]:  # Check latest 20
                    pool_id = pool.get("id", "")
                    if pool_id in seen_pools: continue
                    seen_pools.add(pool_id)

                    mint_a = pool.get("mintA", {}).get("address", "")
                    mint_b = pool.get("mintB", {}).get("address", "")
                    token_mint = mint_b if mint_a in (SOL_MINT, USDC_MINT) else mint_a
                    if token_mint in (SOL_MINT, USDC_MINT): continue

                    liquidity = float(pool.get("tvl", 0))
                    if liquidity < MIN_LIQUIDITY:
                        log.debug(f"Skip low liq: {token_mint[:12]}... ${liquidity:.0f}")
                        continue

                    log.info(f"NEW POOL: {token_mint[:20]}... liq=${liquidity:.0f}")

                    # Evaluate
                    token_info = await get_token_info(token_mint)
                    base_risk = assess_risk(token_info)
                    llm_risk, usage = await llm_evaluate(token_info, {"liquidity_usd": liquidity})
                    total_risk = int(base_risk * 0.7 + llm_risk * 100 * 0.3)

                    log.info(f"  Risk: {total_risk}/100 (base={base_risk}, llm={llm_risk:.2f})")

                    if total_risk < RISK_THRESHOLD and len(positions) < MAX_POSITIONS:
                        log.info(f"  BUY SIGNAL! Sniping {BUY_AMOUNT} SOL...")
                        amount_lamports = int(BUY_AMOUNT * 1e9)
                        try:
                            quote = await get_jupiter_quote(SOL_MINT, token_mint, amount_lamports)
                            result = await execute_swap(quote)
                            tx_id = result.get("result", "unknown")
                            log.info(f"  BOUGHT: tx={tx_id}")
                            positions[token_mint] = {
                                "entry_time": datetime.now(timezone.utc).isoformat(),
                                "amount_sol": BUY_AMOUNT,
                                "token_mint": token_mint,
                                "tx": tx_id,
                                "risk_score": total_risk
                            }
                            with open(TRADES_LOG, "a") as f:
                                f.write(json.dumps(positions[token_mint]) + "\n")
                        except Exception as e:
                            log.error(f"  Buy failed: {e}")
                    else:
                        log.info(f"  SKIP: risk={total_risk} (threshold={RISK_THRESHOLD})")

        except Exception as e:
            log.error(f"Monitor error: {e}")

        await asyncio.sleep(5)  # Poll every 5 seconds

if __name__ == "__main__":
    asyncio.run(monitor_new_pools())
