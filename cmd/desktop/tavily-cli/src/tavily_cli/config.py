"""
config.py — 读取环境变量与全局常量
API Key 必须通过 TAVILY_API_KEY 环境变量提供，不得硬编码。
"""
import os
from pathlib import Path
from dotenv import load_dotenv

# 尝试加载 .env（本地开发用，生产环境忽略）
load_dotenv()

API_BASE_URL = "https://api.tavily.com"
CACHE_DIR = Path.home() / ".tavily_cache"
DEFAULT_SEARCH_TIMEOUT = 30
DEFAULT_CRAWL_TIMEOUT = 60
CACHE_TTL_HOURS = int(os.getenv("TAVILY_CACHE_TTL_HOURS", "24"))


def get_api_key() -> str:
    """读取并校验 TAVILY_API_KEY，若未设置则抛出 EnvironmentError。"""
    key = os.getenv("TAVILY_API_KEY", "").strip()
    if not key:
        raise EnvironmentError(
            "[CONFIG] ❌ 环境变量 TAVILY_API_KEY 未设置或为空。\n"
            "请执行：export TAVILY_API_KEY=your_key_here"
        )
    return key
