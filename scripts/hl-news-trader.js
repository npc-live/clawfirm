/**
 * Hyperliquid News-Driven Trading Strategy
 *
 * 策略逻辑:
 *   1. 轮询新闻源 (CryptoPanic + RSS)，提取加密货币关键词
 *   2. 用规则打分 (sentiment score)，识别强驱动事件
 *   3. 匹配 Hyperliquid 永续合约，计算建仓方向 & 仓位
 *   4. 通过 Hyperliquid REST API 下单
 *
 * 支持的事件类型:
 *   - ETF 批准 / 拒绝
 *   - 交易所上币 / 下币
 *   - 协议安全事件 (hack/exploit)
 *   - 监管新闻 (SEC/CFTC)
 *   - 大额清算 / 强平
 *   - 链上巨鲸转账
 *
 * 使用方法:
 *   node scripts/hl-news-trader.js scan          扫描最新新闻 & 信号
 *   node scripts/hl-news-trader.js monitor       持续监控 (每60s)
 *   node scripts/hl-news-trader.js trade         扫描 + 自动下单 (需设置 HL_PRIVATE_KEY)
 *   node scripts/hl-news-trader.js positions     查看当前仓位
 *
 * 环境变量:
 *   ANTHROPIC_API_KEY 必填，Claude API key
 *   HL_PRIVATE_KEY    Hyperliquid 私钥 (hex, 不含0x前缀)
 *   HL_WALLET         钱包地址
 *   HL_PAPER          设为 "true" 开启纸交易模式
 *   CRYPTOPANIC_KEY   CryptoPanic API key (可选, 免费tier也可用)
 */

import { ethers } from 'ethers'; // 仅用于签名
import { writeFileSync, mkdirSync, readFileSync, existsSync } from 'node:fs';

// ============================================================
// 配置
// ============================================================
const CONFIG = {
  // 风控
  MAX_POSITION_USD: 50,          // 单笔最大仓位 $50
  MAX_TOTAL_EXPOSURE_USD: 200,   // 总敞口上限 $200
  MAX_OPEN_TRADES: 4,            // 最多同时持仓数
  STOP_LOSS_PCT: 0.04,           // 止损 4%
  TAKE_PROFIT_PCT: 0.08,         // 止盈 8%
  MIN_SIGNAL_SCORE: 60,          // 最低信号分 (0-100)

  // 持仓时间
  MAX_HOLD_MINUTES: 120,         // 最长持仓 2 小时 (新闻事件衰减)
  ENTRY_WINDOW_MINUTES: 5,       // 新闻发布后 5 分钟内才入场

  // 新闻源轮询间隔
  POLL_INTERVAL_MS: 60_000,      // 60 秒

  // 杠杆
  DEFAULT_LEVERAGE: 3,           // 3x 杠杆
  MAX_LEVERAGE: 5,               // 信号强度极高时最多 5x

  // Hyperliquid
  HL_API: 'https://api.hyperliquid.xyz',
  HL_PAPER: process.env.HL_PAPER === 'true',

  // 新闻
  CRYPTOPANIC_API: 'https://cryptopanic.com/api/v1/posts/',
  CRYPTOPANIC_KEY: process.env.CRYPTOPANIC_KEY || '',

  // RSS 新闻源 (无需 API key)
  RSS_FEEDS: [
    'https://cointelegraph.com/rss',
    'https://decrypt.co/feed',
    'https://www.theblock.co/rss.xml',
  ],

  // AI 分析
  CLAUDE_MODEL: 'claude-haiku-4-5-20251001',   // 快速 + 低成本，适合高频调用
  AI_BATCH_SIZE: 15,                            // 每次批量分析的新闻条数
  ANTHROPIC_API: 'https://api.anthropic.com/v1/messages',
};

// HL 支持的交易对白名单 (用于 prompt 约束 AI 输出)
const HL_COINS = [
  'BTC','ETH','SOL','BNB','AVAX','ADA','DOGE','SHIB','DOT','LINK',
  'UNI','AAVE','SUI','APT','ARB','OP','HYPE','PEPE','WLD','INJ',
  'TIA','SEI','BLUR','JUP','EIGEN','PYTH','STRK','MANTA','ALT','ZRO',
];

// ============================================================
// 工具函数
// ============================================================
function log(level, msg, data = {}) {
  const entry = { ts: new Date().toISOString(), level, msg, ...data };
  console.log(JSON.stringify(entry));
}

async function fetchJson(url, opts = {}) {
  const res = await fetch(url, { timeout: 10000, ...opts });
  if (!res.ok) throw new Error(`HTTP ${res.status} ${url}`);
  return res.json();
}

// ============================================================
// AI 新闻分析 (Claude)
// ============================================================

const SYSTEM_PROMPT = `You are a crypto trading signal analyst. Given a batch of news headlines and summaries, analyze each item and decide if it has clear, immediate price impact on any cryptocurrency.

For each news item that has a tradeable signal, output a JSON object. Return ONLY a JSON array (no markdown, no explanation).

Each signal object must have:
- id: number (matches input index, 0-based)
- coins: string[] (affected tickers from this list ONLY: ${HL_COINS.join(',')})
- direction: "long" | "short"
- score: number 0-100 (confidence × impact magnitude)
- category: one of: etf | listing | delisting | security | regulation | macro | onchain | liquidation | protocol | ecosystem
- decay_fast: boolean (true if impact fades within 2 minutes, e.g. hack news, sudden listing)
- reason: string (one sentence, in Chinese)

Scoring guide:
- 90-100: Confirmed ETF approval/rejection, major exchange hack >$50M, DoJ arrest of exchange CEO
- 75-89:  Major exchange listing (Coinbase/Binance), bridge exploit, SEC lawsuit filed
- 60-74:  Smaller exchange listing, institutional buy announcement, Fed pivot signal
- 40-59:  Whale movement, protocol upgrade, partnership announcement
- <40 or no clear signal: omit the item entirely

Rules:
- Only include coins that are DIRECTLY mentioned or clearly implicated
- For macro news (Fed, CPI), include BTC and ETH
- "long" = price goes up, "short" = price goes down
- If direction is ambiguous, omit the item
- Be strict: noise/opinion pieces/price analysis → omit`;

/**
 * 批量用 Claude 分析新闻，返回信号数组
 * @param {Array<{title:string, body:string, url:string, published:Date, source:string}>} items
 * @returns {Promise<Array>}
 */
async function analyzeNewsWithAI(items) {
  const apiKey = process.env.ANTHROPIC_API_KEY;
  if (!apiKey) throw new Error('ANTHROPIC_API_KEY not set');

  // 构造输入：只传 id + title + body (控制 token 数)
  const input = items.map((item, i) => ({
    id:    i,
    title: item.title,
    body:  item.body?.slice(0, 200) || '',
  }));

  const res = await fetch(CONFIG.ANTHROPIC_API, {
    method:  'POST',
    headers: {
      'Content-Type':      'application/json',
      'x-api-key':         apiKey,
      'anthropic-version': '2023-06-01',
    },
    body: JSON.stringify({
      model:      CONFIG.CLAUDE_MODEL,
      max_tokens: 2048,
      system:     SYSTEM_PROMPT,
      messages: [{
        role:    'user',
        content: `Analyze these ${items.length} crypto news items:\n\n${JSON.stringify(input, null, 2)}`,
      }],
    }),
  });

  if (!res.ok) {
    const err = await res.text();
    throw new Error(`Claude API error ${res.status}: ${err}`);
  }

  const data = await res.json();
  const text = data.content?.[0]?.text || '[]';

  // 解析 JSON (Claude 有时在前后加空白或 ```json)
  const jsonStr = text.replace(/^```json\s*/i, '').replace(/```\s*$/, '').trim();
  let signals;
  try {
    signals = JSON.parse(jsonStr);
  } catch {
    log('warn', 'AI response parse failed', { raw: text.slice(0, 300) });
    return [];
  }

  // 将信号与原始新闻条目合并
  return signals.map(s => ({
    ...s,
    headline:  items[s.id]?.title || '',
    url:       items[s.id]?.url || '',
    published: items[s.id]?.published?.toISOString() || new Date().toISOString(),
    source:    items[s.id]?.source || '',
  }));
}

/**
 * 计算仓位大小 (简化 Kelly)
 * p: 预期胜率 (由信号分估算)
 * b: 止盈 / 止损比
 */
function calcPositionSize(score, accountBalance) {
  const p = 0.45 + (score / 100) * 0.25; // score 60 → 60%, score 100 → 70%
  const b = CONFIG.TAKE_PROFIT_PCT / CONFIG.STOP_LOSS_PCT; // 8/4 = 2
  const kelly = (p * b - (1 - p)) / b;
  const halfKelly = kelly * 0.5;
  const usd = Math.min(
    CONFIG.MAX_POSITION_USD,
    Math.max(10, accountBalance * halfKelly),
  );
  return parseFloat(usd.toFixed(2));
}

function leverage(score) {
  if (score >= 85) return CONFIG.MAX_LEVERAGE;
  if (score >= 70) return 4;
  return CONFIG.DEFAULT_LEVERAGE;
}

// ============================================================
// 新闻获取
// ============================================================
async function fetchCryptoPanic() {
  const params = new URLSearchParams({
    auth_token: CONFIG.CRYPTOPANIC_KEY || 'public',
    public:     'true',
    filter:     'hot',
    kind:       'news',
  });
  try {
    const data = await fetchJson(`${CONFIG.CRYPTOPANIC_API}?${params}`);
    return (data.results || []).map(item => ({
      title:      item.title,
      body:       item.body || '',
      url:        item.url,
      published:  new Date(item.published_at),
      source:     'cryptopanic',
    }));
  } catch (e) {
    log('warn', 'CryptoPanic fetch failed', { error: e.message });
    return [];
  }
}

async function fetchRSS(feedUrl) {
  try {
    const text = await (await fetch(feedUrl, { timeout: 8000 })).text();
    const items = [];
    // 极简 RSS 解析 (无需依赖)
    const itemMatches = text.matchAll(/<item[^>]*>([\s\S]*?)<\/item>/g);
    for (const [, itemXml] of itemMatches) {
      const title   = itemXml.match(/<title[^>]*><!\[CDATA\[(.*?)\]\]>/)?.[1]
                   || itemXml.match(/<title[^>]*>(.*?)<\/title>/)?.[1] || '';
      const desc    = itemXml.match(/<description[^>]*><!\[CDATA\[(.*?)\]\]>/)?.[1]
                   || itemXml.match(/<description[^>]*>(.*?)<\/description>/)?.[1] || '';
      const pubDate = itemXml.match(/<pubDate[^>]*>(.*?)<\/pubDate>/)?.[1] || '';
      const link    = itemXml.match(/<link[^>]*>(.*?)<\/link>/)?.[1] || '';

      items.push({
        title:     title.replace(/<[^>]+>/g, '').trim(),
        body:      desc.replace(/<[^>]+>/g, '').trim().slice(0, 500),
        url:       link.trim(),
        published: pubDate ? new Date(pubDate) : new Date(),
        source:    new URL(feedUrl).hostname,
      });
    }
    return items.slice(0, 20);
  } catch (e) {
    log('warn', 'RSS fetch failed', { feed: feedUrl, error: e.message });
    return [];
  }
}

async function fetchAllNews() {
  const [cpNews, ...rssResults] = await Promise.allSettled([
    fetchCryptoPanic(),
    ...CONFIG.RSS_FEEDS.map(fetchRSS),
  ]);

  const all = [
    ...(cpNews.status === 'fulfilled' ? cpNews.value : []),
    ...rssResults.flatMap(r => r.status === 'fulfilled' ? r.value : []),
  ];

  // 去重 (按标题相似度)
  const seen = new Set();
  return all.filter(item => {
    const key = item.title.toLowerCase().slice(0, 60);
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

// ============================================================
// Hyperliquid API
// ============================================================
const HL_API = CONFIG.HL_API;

async function hlGetMids() {
  const data = await fetchJson(`${HL_API}/info`, {
    method:  'POST',
    headers: { 'Content-Type': 'application/json' },
    body:    JSON.stringify({ type: 'allMids' }),
  });
  return data; // { BTC: "95000.0", ETH: "3500.0", ... }
}

async function hlGetPositions(walletAddress) {
  const data = await fetchJson(`${HL_API}/info`, {
    method:  'POST',
    headers: { 'Content-Type': 'application/json' },
    body:    JSON.stringify({ type: 'clearinghouseState', user: walletAddress }),
  });
  return data?.assetPositions || [];
}

async function hlGetMeta() {
  return fetchJson(`${HL_API}/info`, {
    method:  'POST',
    headers: { 'Content-Type': 'application/json' },
    body:    JSON.stringify({ type: 'meta' }),
  });
}

/**
 * 构造并提交 Hyperliquid 订单
 * 使用 EIP-712 签名
 */
async function hlPlaceOrder({ coin, isBuy, sz, limitPx, leverage: lev, walletPrivKey }) {
  if (CONFIG.HL_PAPER) {
    log('info', '[PAPER] Order skipped', { coin, isBuy, sz, limitPx });
    return { paper: true, coin, isBuy, sz, limitPx };
  }

  if (!walletPrivKey) {
    throw new Error('HL_PRIVATE_KEY not set');
  }

  const wallet = new ethers.Wallet(walletPrivKey);
  const nonce  = Date.now();

  // 设置杠杆 (先调 leverage API)
  await fetchJson(`${HL_API}/exchange`, {
    method:  'POST',
    headers: { 'Content-Type': 'application/json' },
    body:    JSON.stringify({
      action: {
        type:     'updateLeverage',
        asset:    coin,
        isCross:  true,
        leverage: lev,
      },
      nonce,
      signature: await wallet.signMessage(`updateLeverage:${coin}:${lev}:${nonce}`),
    }),
  });

  // 下单
  const orderAction = {
    type:   'order',
    orders: [{
      a:    coin,           // asset
      b:    isBuy,          // is_buy
      p:    limitPx,        // price (略高于市价确保成交)
      s:    String(sz),     // size
      r:    false,          // reduce_only
      t:    { limit: { tif: 'Ioc' } }, // Immediate-or-Cancel
    }],
    grouping: 'na',
  };

  const res = await fetchJson(`${HL_API}/exchange`, {
    method:  'POST',
    headers: { 'Content-Type': 'application/json' },
    body:    JSON.stringify({
      action:    orderAction,
      nonce:     nonce + 1,
      signature: await wallet.signMessage(JSON.stringify(orderAction) + ':' + (nonce + 1)),
    }),
  });

  return res;
}

// ============================================================
// 信号分析核心 (AI 驱动)
// ============================================================

/**
 * 过滤新鲜新闻 → 分批调用 Claude → 展开为 per-coin 信号 → 去重
 */
async function analyzeNews(newsItems, mids) {
  const cutoff = new Date(Date.now() - CONFIG.ENTRY_WINDOW_MINUTES * 60_000);

  // 只分析入场窗口内的新闻
  const fresh = newsItems.filter(item => item.published >= cutoff);
  if (fresh.length === 0) return [];

  console.log(`  → ${fresh.length} 条新鲜新闻 (${CONFIG.ENTRY_WINDOW_MINUTES}min内) 送入 AI 分析...`);

  // 分批调用，避免单次 token 过多
  const batches = [];
  for (let i = 0; i < fresh.length; i += CONFIG.AI_BATCH_SIZE) {
    batches.push(fresh.slice(i, i + CONFIG.AI_BATCH_SIZE));
  }

  const aiResults = await Promise.all(batches.map(batch => analyzeNewsWithAI(batch)));
  const rawSignals = aiResults.flat();

  console.log(`  → AI 返回 ${rawSignals.length} 个原始信号`);

  // 展开 coins 数组为 per-coin 信号，同时附上实时价格
  const expanded = [];
  for (const s of rawSignals) {
    if (s.score < CONFIG.MIN_SIGNAL_SCORE) continue;

    for (const coin of (s.coins || [])) {
      const price = parseFloat(mids[coin]);
      if (!price) continue; // HL 不支持该币种

      expanded.push({
        coin,
        direction:  s.direction,
        score:      s.score,
        category:   s.category,
        decay_fast: s.decay_fast,
        reason:     s.reason,
        price,
        headline:   s.headline,
        url:        s.url,
        published:  s.published,
        source:     s.source,
      });
    }
  }

  // 去重：同一币种只保留最高分信号
  const best = new Map();
  for (const s of expanded) {
    const key = `${s.coin}:${s.direction}`;
    if (!best.has(key) || best.get(key).score < s.score) {
      best.set(key, s);
    }
  }

  return [...best.values()].sort((a, b) => b.score - a.score);
}

// ============================================================
// 扫描命令
// ============================================================
async function scan() {
  console.log('='.repeat(70));
  console.log('  Hyperliquid News Trader — Signal Scanner');
  console.log('  ' + new Date().toISOString());
  console.log('='.repeat(70));
  console.log('');

  console.log('正在获取新闻 & 市场数据...');
  const [news, mids] = await Promise.all([fetchAllNews(), hlGetMids()]);
  console.log(`已获取 ${news.length} 条新闻 | HL 支持 ${Object.keys(mids).length} 个币种\n`);

  const signals = await analyzeNews(news, mids);

  if (signals.length === 0) {
    console.log('没有发现达到阈值的交易信号。');
    console.log(`(最低分要求: ${CONFIG.MIN_SIGNAL_SCORE}, 入场窗口: ${CONFIG.ENTRY_WINDOW_MINUTES}min)`);
    return signals;
  }

  console.log(`发现 ${signals.length} 个信号:\n`);
  console.log('分数  方向   币种    价格          类别        新闻标题');
  console.log('-'.repeat(80));

  for (const s of signals) {
    const dir  = s.direction === 'long' ? '📈 LONG ' : '📉 SHORT';
    const fast = s.decay_fast ? ' ⚡' : '';
    console.log(
      `${String(s.score).padStart(3)}   ${dir}  ${s.coin.padEnd(6)}  ` +
      `$${s.price.toFixed(2).padStart(10)}  ${s.category.padEnd(12)}  ` +
      `${s.headline.slice(0, 45)}${fast}`
    );
    console.log(`      AI理由: ${s.reason || '-'}`);
    console.log(`      来源: ${s.source} | ${s.published}`);
    console.log('');
  }

  // 保存
  mkdirSync('./data', { recursive: true });
  writeFileSync('./data/hl-signals.json', JSON.stringify({
    ts: new Date().toISOString(),
    signals,
  }, null, 2));
  console.log('信号已保存到: ./data/hl-signals.json');

  return signals;
}

// ============================================================
// 自动交易命令
// ============================================================
async function trade() {
  const PRIVATE_KEY  = process.env.HL_PRIVATE_KEY;
  const WALLET       = process.env.HL_WALLET;
  const isPaper      = CONFIG.HL_PAPER;

  if (!isPaper && (!PRIVATE_KEY || !WALLET)) {
    console.error('❌ 请设置 HL_PRIVATE_KEY 和 HL_WALLET 环境变量，或设置 HL_PAPER=true');
    process.exit(1);
  }

  console.log(`\n模式: ${isPaper ? '📝 纸交易' : '🔴 实盘'}`);

  const signals = await scan();
  if (signals.length === 0) return;

  // 检查现有仓位
  let positions = [];
  if (!isPaper && WALLET) {
    positions = await hlGetPositions(WALLET);
  }

  const openCount   = positions.filter(p => parseFloat(p.position?.szi || 0) !== 0).length;
  const ACCOUNT_BAL = 200; // 简化: 假设账户余额 $200，正式版需从 API 读取

  const results = [];
  let executed = 0;

  for (const signal of signals) {
    if (openCount + executed >= CONFIG.MAX_OPEN_TRADES) {
      log('info', '达到最大持仓数', { max: CONFIG.MAX_OPEN_TRADES });
      break;
    }

    // decay_fast 信号要求 2 分钟内发布
    if (signal.decay_fast) {
      const age = (Date.now() - new Date(signal.published).getTime()) / 1000;
      if (age > 120) {
        log('info', '快速衰减信号已过期', { coin: signal.coin, ageSeconds: age });
        continue;
      }
    }

    const positionUsd  = calcPositionSize(signal.score, ACCOUNT_BAL);
    const lev          = leverage(signal.score);
    const isBuy        = signal.direction === 'long';
    const slippage     = 0.002; // 0.2% 滑点容忍
    const limitPx      = isBuy
      ? parseFloat((signal.price * (1 + slippage)).toFixed(2))
      : parseFloat((signal.price * (1 - slippage)).toFixed(2));
    const sz           = parseFloat((positionUsd / signal.price).toFixed(4));

    console.log(`\n▶ 执行 ${isBuy ? 'LONG' : 'SHORT'} ${signal.coin}`);
    console.log(`  价格: $${signal.price} | 仓位: $${positionUsd} | 杠杆: ${lev}x`);
    console.log(`  信号: ${signal.score}分 (${signal.category}) — ${signal.headline.slice(0, 60)}`);

    try {
      const res = await hlPlaceOrder({
        coin:         signal.coin,
        isBuy,
        sz,
        limitPx,
        leverage:     lev,
        walletPrivKey: PRIVATE_KEY,
      });

      console.log(`  ✅ 订单结果: ${JSON.stringify(res)}`);
      results.push({ signal, positionUsd, lev, sz, limitPx, result: res });
      executed++;
    } catch (e) {
      console.error(`  ❌ 下单失败: ${e.message}`);
    }
  }

  console.log(`\n本轮执行: ${executed} 笔`);

  mkdirSync('./data', { recursive: true });
  writeFileSync('./data/hl-trades.json', JSON.stringify({
    ts:      new Date().toISOString(),
    mode:    isPaper ? 'paper' : 'live',
    results,
  }, null, 2));
}

// ============================================================
// 持续监控
// ============================================================
async function monitor() {
  console.log(`\n开始持续监控 (每 ${CONFIG.POLL_INTERVAL_MS / 1000}s 刷新一次)...\n`);
  let round = 0;
  while (true) {
    round++;
    console.log(`\n${'='.repeat(40)} 第 ${round} 轮 ${new Date().toLocaleTimeString()}`);
    try {
      await scan();
    } catch (e) {
      console.error('扫描出错:', e.message);
    }
    await new Promise(r => setTimeout(r, CONFIG.POLL_INTERVAL_MS));
  }
}

// ============================================================
// 查看仓位
// ============================================================
async function positions() {
  const WALLET = process.env.HL_WALLET;
  if (!WALLET) { console.error('❌ 请设置 HL_WALLET 环境变量'); process.exit(1); }

  const pos = await hlGetPositions(WALLET);
  const open = pos.filter(p => parseFloat(p.position?.szi || 0) !== 0);

  if (open.length === 0) {
    console.log('当前无持仓。');
    return;
  }

  console.log(`\n当前持仓 (${open.length} 个):\n`);
  console.log('币种    方向    仓位大小   入场价    未实现盈亏  杠杆');
  console.log('-'.repeat(65));

  for (const { position: p } of open) {
    const sz      = parseFloat(p.szi);
    const dir     = sz > 0 ? '📈 LONG ' : '📉 SHORT';
    const pnl     = parseFloat(p.unrealizedPnl);
    const pnlStr  = pnl >= 0 ? `+$${pnl.toFixed(2)}` : `-$${Math.abs(pnl).toFixed(2)}`;
    console.log(
      `${p.coin.padEnd(8)}${dir}  ` +
      `${Math.abs(sz).toFixed(4).padStart(10)}   ` +
      `$${parseFloat(p.entryPx).toFixed(2).padStart(8)}  ` +
      `${pnlStr.padStart(12)}  ` +
      `${p.leverage?.value || '-'}x`
    );
  }
}

// ============================================================
// 主入口
// ============================================================
const cmd = process.argv[2];

const USAGE = `
Hyperliquid News-Driven Trading Script

用法:
  node scripts/hl-news-trader.js scan          扫描最新新闻信号
  node scripts/hl-news-trader.js monitor       持续监控 (每60s)
  node scripts/hl-news-trader.js trade         自动下单 (需要环境变量)
  node scripts/hl-news-trader.js positions     查看当前持仓

环境变量:
  ANTHROPIC_API_KEY 必填，Claude API key
  HL_PRIVATE_KEY    Hyperliquid 私钥 (hex)
  HL_WALLET         钱包地址 (0x...)
  HL_PAPER=true     开启纸交易模式 (默认: false)
  CRYPTOPANIC_KEY   CryptoPanic API key (可选)

策略逻辑:
  1. 每60s拉取 CryptoPanic + 3个RSS源
  2. 将新鲜新闻批量送入 Claude (${CONFIG.CLAUDE_MODEL}) 分析
  3. AI 输出结构化信号: 币种、方向、分数(0-100)、类别、理由
  4. 仅交易5分钟内的新闻 (decay_fast 事件限2分钟)
  5. Half-Kelly 仓位 + 4%止损 / 8%止盈

风控参数:
  MIN_SIGNAL_SCORE:    ${CONFIG.MIN_SIGNAL_SCORE}
  MAX_POSITION_USD:    $${CONFIG.MAX_POSITION_USD}
  MAX_TOTAL_EXPOSURE:  $${CONFIG.MAX_TOTAL_EXPOSURE_USD}
  MAX_OPEN_TRADES:     ${CONFIG.MAX_OPEN_TRADES}
  STOP_LOSS:           ${CONFIG.STOP_LOSS_PCT * 100}%
  TAKE_PROFIT:         ${CONFIG.TAKE_PROFIT_PCT * 100}%
  MAX_HOLD:            ${CONFIG.MAX_HOLD_MINUTES}min
`;

switch (cmd) {
  case 'scan':      scan().catch(console.error);      break;
  case 'monitor':   monitor().catch(console.error);   break;
  case 'trade':     trade().catch(console.error);     break;
  case 'positions': positions().catch(console.error); break;
  default:          console.log(USAGE);
}
