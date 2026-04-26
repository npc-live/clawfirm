# Raydium Pool Monitoring

## API
- Pools: `GET https://api.raydium.io/v2/ammV3/ammPools`
- Returns pool list with `mintA`, `mintB`, `tvl`, `volume24h`

## New Pool Detection
- Poll every 5-10 seconds
- Compare against seen pool set
- Filter: skip if both mints are SOL/USDC/USDT (stablecoin pairs)
- Filter: skip if TVL < minimum liquidity threshold

## Rugpull Red Flags
1. Mint authority NOT revoked — creator can mint infinite tokens
2. Freeze authority NOT revoked — creator can freeze your tokens
3. Top 10 holders own >50% supply — dump risk
4. LP not locked/burned — creator can pull liquidity
5. Contract deployed < 1 hour ago with huge initial buy
6. No social media / website / audit
