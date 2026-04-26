# ClawFirm Solana Skills

AI Agent 专用的 Solana 区块链技能包，支持钱包管理、代币交易、DeFi 操作。

## 已集成 Skills

| Skill | 功能 | 来源 |
|-------|------|------|
| **solana-payments-wallets-trading** | SOL/USDC 支付、买卖代币、价格查询、质押、收益 | ClawHub |
| **solana-easy-swap** | Jupiter 驱动的代币 swap，聊天即交易 | ClawHub |
| **solana-sniper-bot** | 新币狙击、Raydium/Jupiter 监控、LLM 风险评估 | ClawHub |
| **raphael-solana** | Solana + Polygon 钱包 + Polymarket 套利 + Raydium swap | ClawHub |
| **solana-basics** | Solana 基础操作 | ClawHub |
| **solana-connect** | Solana 钱包连接 | ClawHub |

## 安装

Skills 会自动从 `app/assets/skills/solana/` 加载。

也可以手动安装到 OpenClaw：
```bash
clawhub install solana-payments-wallets-trading
clawhub install solana-easy-swap
clawhub install raphael-solana
```

## 使用示例

### 1. 钱包操作
```
创建一个 Solana 钱包
查看我的 SOL 余额
```

### 2. 代币交易
```
用 1 SOL 换 USDC
查看 BONK 的价格
```

### 3. 新币狙击
```
监控 pump.fun 新币
设置自动买入条件
```

## ⚠️ 安全警告

- `solana-sniper-bot` 被 VirusTotal 标记为可疑，使用前请审计代码
- 建议使用独立钱包，不要存放大额资金
- 所有交易操作需要用户确认

## 返佣集成

这些 skills 可配合 Jupiter Referral Program 使用：
- 开发者可添加 0.1%-2% 手续费
- Jupiter 抽成 20%，剩余 80% 归集成方

详见：https://referral.jup.ag/

---

*Part of ClawFirm.dev - AI 一人公司合伙人*
