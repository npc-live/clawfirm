/**
 * Polymarket Weather Trading Script v2.0
 * 
 * 修复内容:
 * 1. 使用 bestAsk 作为买入价格（而不是 mid price）
 * 2. 使用 bestBid 作为卖出价格
 * 3. 添加价格上限警告
 * 4. 添加流动性检查
 * 5. 添加价差 (spread) 警告
 * 
 * 使用方法:
 *   node scripts/weather-trader-v2.js scan     扫描机会
 *   node scripts/weather-trader-v2.js check    检查特定市场
 */

const GAMMA_API = 'https://gamma-api.polymarket.com';
const WEATHER_API = 'https://ensemble-api.open-meteo.com/v1/ensemble';

// ============== 风控配置 ==============
const CONFIG = {
  // 价格限制
  MAX_BUY_PRICE: 0.10,      // 最高买入价 10%（超过不买）
  MIN_LIQUIDITY: 500,       // 最低流动性 $500
  MAX_SPREAD_PCT: 0.50,     // 最大价差 50%（超过警告）
  
  // 策略参数
  MIN_EDGE: 0.10,           // 最低 edge 10%
  HALF_KELLY: 0.5,          // Half Kelly
  
  // 监控城市
  CITIES: [
    { name: 'Shanghai', slug: 'shanghai', lat: 31.23, lon: 121.47 },
    { name: 'Tokyo', slug: 'tokyo', lat: 35.69, lon: 139.69 },
    { name: 'Seoul', slug: 'seoul', lat: 37.46, lon: 126.44 },
    { name: 'London', slug: 'london', lat: 51.51, lon: -0.13 },
    { name: 'NYC', slug: 'nyc', lat: 40.71, lon: -74.01 },
  ],
};

// ============== 工具函数 ==============
async function fetchJson(url) {
  const res = await fetch(url);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

function getTomorrowSlug() {
  const tomorrow = new Date();
  tomorrow.setDate(tomorrow.getDate() + 1);
  const months = ['january', 'february', 'march', 'april', 'may', 'june', 
                  'july', 'august', 'september', 'october', 'november', 'december'];
  return `${months[tomorrow.getMonth()]}-${tomorrow.getDate()}`;
}

function formatPct(value) {
  return `${(value * 100).toFixed(1)}%`;
}

// ============== 获取市场数据 (修复版) ==============
async function getMarketData(city, dateSlug) {
  const slug = `highest-temperature-in-${city.slug}-on-${dateSlug}-2026`;
  const url = `${GAMMA_API}/events?slug=${slug}`;
  
  try {
    const events = await fetchJson(url);
    if (!events || events.length === 0) return null;
    
    const event = events[0];
    
    return {
      city: city.name,
      slug,
      liquidity: event.liquidity,
      volume24h: event.volume24hr,
      markets: event.markets
        .filter(m => !m.closed)
        .map(m => {
          const prices = JSON.parse(m.outcomePrices);
          const midPrice = parseFloat(prices[0]);
          const bestBid = parseFloat(m.bestBid) || 0;
          const bestAsk = parseFloat(m.bestAsk) || midPrice * 1.1;
          const spread = bestAsk - bestBid;
          const spreadPct = bestBid > 0 ? spread / bestBid : 0;
          
          return {
            title: m.groupItemTitle,
            // 价格数据
            midPrice,
            bestBid,        // 卖出价
            bestAsk,        // 买入价 ← 重要！
            spread,
            spreadPct,
            // 流动性
            liquidity: m.liquidityNum || 0,
            // Token ID (用于下单)
            tokenId: JSON.parse(m.clobTokenIds)[0],
          };
        }),
    };
  } catch (e) {
    console.error(`获取 ${city.name} 失败:`, e.message);
    return null;
  }
}

// ============== 获取气象预报 ==============
async function getWeatherForecast(city) {
  const url = `${WEATHER_API}?latitude=${city.lat}&longitude=${city.lon}&hourly=temperature_2m&models=icon_seamless&forecast_days=3`;
  const data = await fetchJson(url);
  
  const temps = [];
  for (const [key, values] of Object.entries(data.hourly)) {
    if (key.startsWith('temperature_2m') && Array.isArray(values)) {
      if (values[38] !== undefined) temps.push(values[38]);
    }
  }
  
  // 计算概率分布
  const distribution = {};
  for (const temp of temps) {
    const bin = Math.round(temp);
    distribution[bin] = (distribution[bin] || 0) + 1 / temps.length;
  }
  
  // 估算全天最高温 = 14:00温度 + 3-4°C
  const avg = temps.reduce((a, b) => a + b, 0) / temps.length;
  const estimatedMax = avg + 3.5;
  
  return {
    temps,
    avg,
    min: Math.min(...temps),
    max: Math.max(...temps),
    estimatedDailyMax: estimatedMax,
    distribution,
  };
}

// ============== 扫描机会 (修复版) ==============
async function scan() {
  console.log('='.repeat(70));
  console.log('  Polymarket Weather Scanner v2.0 (Fixed)');
  console.log('  ' + new Date().toISOString());
  console.log('='.repeat(70));
  console.log('');
  
  const dateSlug = getTomorrowSlug();
  console.log(`扫描日期: ${dateSlug}-2026\n`);
  
  console.log('⚠️  风控参数:');
  console.log(`   最高买入价: ${formatPct(CONFIG.MAX_BUY_PRICE)}`);
  console.log(`   最低流动性: $${CONFIG.MIN_LIQUIDITY}`);
  console.log(`   最大价差: ${formatPct(CONFIG.MAX_SPREAD_PCT)}`);
  console.log(`   最低 Edge: ${formatPct(CONFIG.MIN_EDGE)}`);
  console.log('');
  
  const opportunities = [];
  
  for (const city of CONFIG.CITIES) {
    console.log(`\n${'━'.repeat(50)}`);
    console.log(`📍 ${city.name}`);
    console.log('━'.repeat(50));
    
    // 获取市场数据
    const market = await getMarketData(city, dateSlug);
    if (!market) {
      console.log('  ❌ 未找到市场');
      continue;
    }
    
    console.log(`流动性: $${market.liquidity.toFixed(0)} | 24h量: $${market.volume24h.toFixed(0)}\n`);
    
    // 获取气象预报
    const forecast = await getWeatherForecast(city);
    console.log(`气象预报: 14:00温度 ${forecast.min.toFixed(1)}-${forecast.max.toFixed(1)}°C`);
    console.log(`全天最高估算: ~${forecast.estimatedDailyMax.toFixed(0)}°C\n`);
    
    // 显示市场价格
    console.log('温度      Mid     Bid     Ask    Spread   流动性   状态');
    console.log('-'.repeat(65));
    
    for (const m of market.markets) {
      const tempNum = parseInt(m.title.match(/\d+/)?.[0] || '0');
      
      // 计算模型概率 (基于全天最高温估算)
      let modelProb = 0;
      for (let t = tempNum - 1; t <= tempNum + 1; t++) {
        modelProb += forecast.distribution[t - 4] || 0; // 调整偏移
      }
      
      // 计算 edge (基于 bestAsk，即实际买入价)
      const edge = modelProb - m.bestAsk;
      
      // 状态判断
      let status = '';
      const warnings = [];
      
      if (m.bestAsk > CONFIG.MAX_BUY_PRICE) {
        warnings.push('价格过高');
      }
      if (m.liquidity < CONFIG.MIN_LIQUIDITY) {
        warnings.push('流动性低');
      }
      if (m.spreadPct > CONFIG.MAX_SPREAD_PCT) {
        warnings.push('价差大');
      }
      
      if (edge >= CONFIG.MIN_EDGE && warnings.length === 0) {
        status = `✅ Edge ${formatPct(edge)}`;
        opportunities.push({
          city: city.name,
          temp: m.title,
          bestAsk: m.bestAsk,
          modelProb,
          edge,
          liquidity: m.liquidity,
          payout: 1 / m.bestAsk,
        });
      } else if (warnings.length > 0) {
        status = `⚠️ ${warnings.join(', ')}`;
      } else {
        status = '—';
      }
      
      console.log(
        `${m.title.padEnd(10)}` +
        `${formatPct(m.midPrice).padStart(6)} ` +
        `${formatPct(m.bestBid).padStart(6)} ` +
        `${formatPct(m.bestAsk).padStart(6)} ` +
        `${formatPct(m.spreadPct).padStart(7)} ` +
        `$${m.liquidity.toFixed(0).padStart(6)} ` +
        `${status}`
      );
    }
  }
  
  // 汇总
  console.log('\n' + '='.repeat(70));
  console.log('  汇总');
  console.log('='.repeat(70));
  
  if (opportunities.length === 0) {
    console.log('\n⚠️  没有找到符合条件的机会');
    console.log('可能原因:');
    console.log('  - 所有低价选项都不符合 edge 要求');
    console.log('  - 流动性不足');
    console.log('  - 价差过大');
  } else {
    console.log(`\n找到 ${opportunities.length} 个机会:\n`);
    
    for (const opp of opportunities.slice(0, 5)) {
      console.log(`  ${opp.city} ${opp.temp}`);
      console.log(`    买入价(Ask): ${formatPct(opp.bestAsk)} | 模型: ${formatPct(opp.modelProb)} | Edge: ${formatPct(opp.edge)}`);
      console.log(`    赔率: ${opp.payout.toFixed(1)}x | 流动性: $${opp.liquidity.toFixed(0)}`);
      console.log('');
    }
  }
  
  // 保存结果
  const { mkdirSync, writeFileSync } = await import('node:fs');
  mkdirSync('./data', { recursive: true });
  writeFileSync('./data/opportunities-v2.json', JSON.stringify({
    ts: new Date().toISOString(),
    config: CONFIG,
    opportunities,
  }, null, 2));
  
  console.log('结果已保存到: ./data/opportunities-v2.json');
}

// ============== 检查特定市场 ==============
async function check(cityName) {
  console.log(`\n检查 ${cityName || '所有'} 市场的实时价格...\n`);
  
  const dateSlug = getTomorrowSlug();
  const cities = cityName 
    ? CONFIG.CITIES.filter(c => c.name.toLowerCase() === cityName.toLowerCase())
    : CONFIG.CITIES;
  
  for (const city of cities) {
    const market = await getMarketData(city, dateSlug);
    if (!market) continue;
    
    console.log(`\n📍 ${city.name} (${market.slug})`);
    console.log(`   总流动性: $${market.liquidity.toFixed(0)}`);
    console.log('');
    console.log('   温度       Bid(卖)   Ask(买)   Spread');
    console.log('   ' + '-'.repeat(45));
    
    market.markets
      .filter(m => m.bestAsk > 0.001)
      .sort((a, b) => b.midPrice - a.midPrice)
      .slice(0, 8)
      .forEach(m => {
        console.log(
          `   ${m.title.padEnd(12)}` +
          `${formatPct(m.bestBid).padStart(8)} ` +
          `${formatPct(m.bestAsk).padStart(8)} ` +
          `${formatPct(m.spreadPct).padStart(8)}`
        );
      });
  }
}

// ============== 主入口 ==============
const command = process.argv[2];
const arg = process.argv[3];

if (command === 'scan') {
  scan().catch(console.error);
} else if (command === 'check') {
  check(arg).catch(console.error);
} else {
  console.log(`
Polymarket Weather Trading Script v2.0

修复内容:
  - 使用 bestAsk 作为买入价格
  - 添加价格上限检查 (默认 10%)
  - 添加流动性检查 (默认 $500)
  - 添加价差警告 (默认 50%)

使用方法:
  node scripts/weather-trader-v2.js scan          扫描所有市场
  node scripts/weather-trader-v2.js check         检查实时价格
  node scripts/weather-trader-v2.js check Tokyo   检查特定城市

风控参数 (可在代码中修改):
  MAX_BUY_PRICE: ${formatPct(CONFIG.MAX_BUY_PRICE)} (买入价格上限)
  MIN_LIQUIDITY: $${CONFIG.MIN_LIQUIDITY} (最低流动性)
  MAX_SPREAD_PCT: ${formatPct(CONFIG.MAX_SPREAD_PCT)} (最大价差)
  MIN_EDGE: ${formatPct(CONFIG.MIN_EDGE)} (最低 edge)
`);
}
