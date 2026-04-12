"""
cache.py — 本地 JSON 缓存（TTL 可配置）
存储路径：~/.tavily_cache/{md5_key}.json
"""
import json
import hashlib
from datetime import datetime, timezone, timedelta
from pathlib import Path
from typing import Optional

from . import config


def _key_to_path(key: str) -> Path:
    """将 cache key 转换为安全的文件路径（纯十六进制 MD5）。"""
    hashed = hashlib.md5(key.encode("utf-8")).hexdigest()
    return config.CACHE_DIR / f"{hashed}.json"


def get(key: str) -> Optional[dict]:
    """
    读取缓存。
    返回 dict（含原始数据 + cached_at 字段）或 None（未命中/已过期）。
    """
    path = _key_to_path(key)
    if not path.exists():
        return None

    try:
        with open(path, "r", encoding="utf-8") as f:
            data = json.load(f)
    except Exception:
        return None

    cached_at_str = data.get("_cached_at")
    if not cached_at_str:
        return None

    cached_at = datetime.fromisoformat(cached_at_str)
    ttl = timedelta(hours=config.CACHE_TTL_HOURS)
    if datetime.now(timezone.utc) - cached_at > ttl:
        # 缓存已过期
        return None

    return data


def set(key: str, data: dict) -> None:
    """写入缓存，自动追加 _cached_at 时间戳。"""
    config.CACHE_DIR.mkdir(parents=True, exist_ok=True)
    path = _key_to_path(key)
    payload = {**data, "_cached_at": datetime.now(timezone.utc).isoformat()}
    try:
        with open(path, "w", encoding="utf-8") as f:
            json.dump(payload, f, ensure_ascii=False, indent=2)
    except Exception:
        pass  # 缓存写入失败不影响主流程，静默忽略


def make_key(tool: str, input_str: str) -> str:
    return f"{tool}:{input_str}"
