/**
 * Polymarket Weather Trading Script
 * 
 * 使用方法:
 * 1. 设置环境变量: PRIVATE_KEY, POLY_API_KEY, POLY_SECRET, POLY_PASSPHRASE
 * 2. 运行: node scripts/weather-trader.js scan    (扫描机会)
 * 3. 运行: node scripts/weather-trader.js trade   (执行交易)
 */

import { createPublicClient, createWalletClient, http } from 'viem';
import { polygon } from 'viem/chains';
import { privateKeyToAccount } from 'viem/accounts';

// ============== 配置 ==============
const CONFIG = {
  // Polymarket API
  GAMMA_API: 'https://gamma-api.polymarket.com',
  CLOB_API: 'https://clob.polymarket.com',
  
  // 气象 API
  WEATHER_API: 'https://ensemble-api.open-meteo.com/v1/ensemble',
  
  // 交易参数
  MIN_EDGE: 0.05,        // 最低 edge 5%
  HALF_KELLY: 0.5,       // Half Kelly
  MAX_POSITION_PCT: 0.10, // 单仓上限 10%
  MAX_EXPOSURE_PCT: 0.80, // 总敞口上限 80%
  MIN_LIQUIDITY: 1000,   // 最低流动性 $1000
  
  // 监控城市
  CITIES: [
    { name: 'Shanghai', slug: 'shanghai', lat: 31.23, lon: 121.47, tz: 'Asia/Shanghai' },
    { name: 'Tokyo', slug: 'tokyo', lat: 35.69, lon: 139.69, tz: 'Asia/Tokyo' },
    { name: 'Seoul', slug: 'seoul', lat: 37.46, lon: 126.44, tz: 'Asia/Seoul' },
    { name: 'NYC', slug: 'nyc', lat: 40.71, lon: -74.01, tz: 'America/New_York' },
    { name: 'London', slug: 'london', lat: 51.51, lon: -0.13, tz: 'Europe/London' },
  ],
};

// ============== 工具函数 ==============
async function fetchJson(url) {
  const res = await fetch(url);
  if (!res.ok) throw new Error(`HTTP ${res.status}: ${res.statusText}`);
  return res.json();
}

function formatDate(date) {
  return date.toISOString().split('T')[0].replace(/-/g, '-');
}

function getTomorrowDateSlug() {
  const tomorrow = new Date();
  tomorrow.setDate(tomorrow.getDate() + 1);
  const months = ['january', 'february', 'march', 'april', 'may', 'june', 
                  'july', 'august', 'september', 'october', 'november', 'december'];
  return `${months[tomorrow.getMonth()]}-${tomorrow.getDate()}`;
}

// ============== 气象数据 ==============
async function getWeatherForecast(city) {
  const url = `${CONFIG.WEATHER_API}?latitude=${city.lat}&longitude=${city.lon}&hourly=temperature_2m&models=icon_seamless&forecast_days=3`;
  const data = await fetchJson(url);
  
  // 获取明天 14:00 的集合预报
  const hourlyData = data.hourly;
  const times = hourlyData.time;
  const tomorrowIdx = 38; // 约 38 小时后
  
  // 收集所有成员的温度
  const temps = [];
  for (const [key, values] of Object.entries(hourlyData)) {
    if (key.startsWith('temperature_2m') && Array.isArray(values)) {
      if (values[tomorrowIdx] !== undefined) {
        temps.push(values[tomorrowIdx]);
      }
    }
  }
  
  // 计算概率分布
  const bins = {};
  for (const temp of temps) {
    const bin = Math.round(temp);
    bins[bin] = (bins[bin] || 0) + 1;
  }
  
  // 转换为概率
  const total = temps.length;
  const distribution = {};
  for (const [bin, count] of Object.entries(bins)) {
    distribution[bin] = count / total;
  }
  
  return {
    city: city.name,
    temps,
    min: Math.min(...temps),
    max: Math.max(...temps),
    avg: temps.reduce((a, b) => a + b, 0) / temps.length,
    distribution,
  };
}

// ============== 市场数据 ==============
async function getMarketData(city, dateSlug) {
  const slug = `highest-temperature-in-${city.slug}-on-${dateSlug}-2026`;
  const url = `${CONFIG.GAMMA_API}/events?slug=${slug}`;
  
  try {
    const events = await fetchJson(url);
    if (!events || events.length === 0) return null;
    
    const event = events[0];
    const markets = event.markets.filter(m => !m.closed);
    
    return {
      city: city.name,
      slug,
      liquidity: event.liquidity,
      volume: event.volume,
      markets: markets.map(m => {
        const prices = JSON.parse(m.outcomePrices);
        const tokenIds = JSON.parse(m.clobTokenIds);
        return {
          title: m.groupItemTitle,
          yesPrice: parseFloat(prices[0]),
          noPrice: parseFloat(prices[1]),
          liquidity: m.liquidityNum,
          tokenId: tokenIds[0],
          conditionId: m.conditionId,
        };
      }),
    };
  } catch (e) {
    console.error(`获取 ${city.name} 市场数据失败:`, e.message);
    return null;
  }
}

// ============== Edge 计算 ==============
function calculateEdge(forecast, market) {
  const opportunities = [];
  
  for (const m of market.markets) {
    // 解析温度区间
    let tempBin;
    const title = m.title;
    if (title.includes('or below')) {
      tempBin = parseInt(title);
    } else if (title.includes('or higher')) {
      tempBin = parseInt(title);
    } else {
      tempBin = parseInt(title.replace('°C', '').replace('°F', ''));
    }
    
    if (isNaN(tempBin)) continue;
    
    // 估算全天最高温（14:00 温度 + 2-3°C）
    const adjustedBin = tempBin - 2; // 模型预测需要向下调整
    
    // 计算模型概率
    let modelProb = 0;
    for (let t = adjustedBin - 1; t <= adjustedBin + 1; t++) {
      modelProb += forecast.distribution[t] || 0;
    }
    
    // 计算 edge
    const edge = modelProb - m.yesPrice;
    
    if (edge >= CONFIG.MIN_EDGE && m.liquidity >= CONFIG.MIN_LIQUIDITY) {
      // 计算 Kelly 仓位
      const kelly = edge / (1 - m.yesPrice);
      const halfKelly = kelly * CONFIG.HALF_KELLY;
      
      opportunities.push({
        city: market.city,
        temp: title,
        marketPrice: m.yesPrice,
        modelProb,
        edge,
        kelly: halfKelly,
        liquidity: m.liquidity,
        tokenId: m.tokenId,
        conditionId: m.conditionId,
        payout: 1 / m.yesPrice,
      });
    }
  }
  
  return opportunities.sort((a, b) => b.edge - a.edge);
}

// ============== 主扫描函数 ==============
async function scan() {
  console.log('='.repeat(60));
  console.log('  Polymarket Weather Scanner');
  console.log('  ' + new Date().toISOString());
  console.log('='.repeat(60));
  console.log('');
  
  const dateSlug = getTomorrowDateSlug();
  console.log(`扫描日期: ${dateSlug}-2026\n`);
  
  const allOpportunities = [];
  
  for (const city of CONFIG.CITIES) {
    console.log(`\n📍 ${city.name}`);
    console.log('-'.repeat(40));
    
    // 获取气象预报
    const forecast = await getWeatherForecast(city);
    console.log(`气象预报: ${forecast.min.toFixed(1)}°C - ${forecast.max.toFixed(1)}°C (avg: ${forecast.avg.toFixed(1)}°C)`);
    
    // 获取市场数据
    const market = await getMarketData(city, dateSlug);
    if (!market) {
      console.log('  ❌ 未找到市场');
      continue;
    }
    console.log(`市场流动性: $${market.liquidity.toFixed(0)}`);
    
    // 计算 edge
    const opportunities = calculateEdge(forecast, market);
    
    if (opportunities.length > 0) {
      console.log('\n套利机会:');
      for (const opp of opportunities) {
        console.log(`  ✅ ${opp.temp}: Edge ${(opp.edge * 100).toFixed(1)}% | 市场 ${(opp.marketPrice * 100).toFixed(1)}% | 模型 ${(opp.modelProb * 100).toFixed(1)}% | 赔率 ${opp.payout.toFixed(1)}x`);
        allOpportunities.push(opp);
      }
    } else {
      console.log('  ⚪ 无套利机会');
    }
  }
  
  // 汇总
  console.log('\n' + '='.repeat(60));
  console.log('  汇总');
  console.log('='.repeat(60));
  console.log(`\n找到 ${allOpportunities.length} 个套利机会\n`);
  
  if (allOpportunities.length > 0) {
    console.log('Top 5 机会:');
    for (const opp of allOpportunities.slice(0, 5)) {
      console.log(`  ${opp.city} ${opp.temp}: Edge ${(opp.edge * 100).toFixed(1)}% | Kelly ${(opp.kelly * 100).toFixed(1)}%`);
    }
  }
  
  // 保存结果
  const outputPath = './data/opportunities.json';
  const { mkdirSync, writeFileSync } = await import('node:fs');
  mkdirSync('./data', { recursive: true });
  writeFileSync(outputPath, JSON.stringify({
    ts: new Date().toISOString(),
    dateSlug,
    opportunities: allOpportunities,
  }, null, 2));
  console.log(`\n结果已保存到: ${outputPath}`);
  
  return allOpportunities;
}

// ============== 交易函数 (占位) ==============
async function trade() {
  console.log('='.repeat(60));
  console.log('  Polymarket Weather Trader');
  console.log('='.repeat(60));
  console.log('');
  
  // 检查环境变量
  const requiredEnv = ['PRIVATE_KEY', 'POLY_API_KEY', 'POLY_SECRET', 'POLY_PASSPHRASE'];
  const missing = requiredEnv.filter(k => !process.env[k]);
  
  if (missing.length > 0) {
    console.error('❌ 缺少环境变量:', missing.join(', '));
    console.log('\n请设置以下环境变量:');
    console.log('  $env:PRIVATE_KEY = "0x..."');
    console.log('  $env:POLY_API_KEY = "..."');
    console.log('  $env:POLY_SECRET = "..."');
    console.log('  $env:POLY_PASSPHRASE = "..."');
    process.exit(1);
  }
  
  // 读取机会
  const { readFileSync, existsSync } = await import('node:fs');
  if (!existsSync('./data/opportunities.json')) {
    console.error('❌ 请先运行 scan 命令');
    process.exit(1);
  }
  
  const data = JSON.parse(readFileSync('./data/opportunities.json', 'utf8'));
  console.log(`读取到 ${data.opportunities.length} 个机会 (扫描时间: ${data.ts})`);
  
  // TODO: 实现 EIP-712 签名和订单提交
  console.log('\n⚠️ 交易功能开发中...');
  console.log('请使用 Polymarket 网页手动下单，或等待后续更新。');
}

// ============== 主入口 ==============
const command = process.argv[2];

if (command === 'scan') {
  scan().catch(console.error);
} else if (command === 'trade') {
  trade().catch(console.error);
} else {
  console.log(`
Polymarket Weather Trading Script

Usage:
  node scripts/weather-trader.js scan    扫描套利机会
  node scripts/weather-trader.js trade   执行交易 (需要设置环境变量)

环境变量:
  PRIVATE_KEY       钱包私钥
  POLY_API_KEY      Polymarket API Key
  POLY_SECRET       Polymarket API Secret
  POLY_PASSPHRASE   Polymarket API Passphrase
`);
}
