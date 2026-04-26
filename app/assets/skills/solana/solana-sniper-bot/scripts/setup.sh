#!/bin/bash
set -e
echo "=== Solana Sniper Bot Setup ==="
pip install solana==0.35.0 solders==0.21.0 httpx==0.27.0 aiohttp==3.9.5 python-dotenv==1.0.1 base58==2.1.1
echo "Done. Create .env with SOLANA_PRIVATE_KEY and LLM_API_KEY"
