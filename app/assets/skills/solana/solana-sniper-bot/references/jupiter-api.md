# Jupiter Aggregator API

## Endpoints
| Endpoint | URL |
|----------|-----|
| Quote | `GET https://quote-api.jup.ag/v6/quote` |
| Swap | `POST https://quote-api.jup.ag/v6/swap` |
| Price | `GET https://price.jup.ag/v6/price` |

## Quote Params
- `inputMint` — Source token mint address
- `outputMint` — Destination token mint address  
- `amount` — Amount in smallest unit (lamports for SOL)
- `slippageBps` — Slippage in basis points (500 = 5%)

## Swap Body
```json
{"quoteResponse": <quote>, "userPublicKey": "<wallet>", "wrapAndUnwrapSol": true}
```
Returns `swapTransaction` (base64 serialized transaction to sign and send).

## Key Mints
- SOL: `So11111111111111111111111111111111111111112`
- USDC: `EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v`
- USDT: `Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB`
