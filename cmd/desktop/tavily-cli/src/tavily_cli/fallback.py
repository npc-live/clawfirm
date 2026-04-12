"""
fallback.py — 三层 Fallback 策略实现
F1: Crawl → Search 降级
F2: → 读取本地缓存
F3: → 输出错误报告 + exit(1)
"""
from urllib.parse import urlparse
from typing import Optional, Union

from . import client, cache, printer
from .models import (
    SearchParams, SearchOutput, SearchResult,
    CrawlOutput, ErrorOutput, FallbackInfo,
)


def _build_search_query_from_url(url: str) -> str:
    """从 URL 提取有意义的搜索词（不使用 site: 语法）。"""
    parsed = urlparse(url)
    domain = parsed.netloc  # e.g. "docs.tavily.com"
    path_hint = parsed.path.strip("/").replace("/", " ")  # e.g. "api reference"
    return f"{domain} {path_hint} documentation".strip()


def fallback_to_search(
    url: str,
    original_error: Exception,
    tool: str = "crawl",
) -> Optional[SearchOutput]:
    """
    F1: 当 Crawl 失败时，用 URL 构建查询词降级到 Search。
    成功返回 SearchOutput，失败返回 None。
    """
    query = _build_search_query_from_url(url)
    printer.warn(tool, f"[F1] Crawl 失败（{original_error}），降级到 Search: '{query}'")

    params = SearchParams(
        query=query,
        search_depth="advanced",
        max_results=10,
        include_answer=True,
    )
    try:
        raw = client.post("/search", params.to_payload(), timeout=30)
        results = [
            SearchResult(
                title=r.get("title", ""),
                url=r.get("url", ""),
                content=r.get("content", ""),
                score=r.get("score", 0.0),
                raw_content=r.get("raw_content"),
                published_date=r.get("published_date"),
            )
            for r in raw.get("results", [])
        ]
        output = SearchOutput(
            status="fallback",
            source="fallback_search",
            query=query,
            answer=raw.get("answer"),
            results=results,
            fallback_info=FallbackInfo(
                source="fallback_search",
                original_error=str(original_error),
            ),
        )
        printer.warn(tool, f"[F1] 降级搜索完成，共 {len(results)} 条结果")
        return output
    except Exception as e:
        printer.error(tool, f"[F1] 降级搜索也失败：{e}")
        return None


def fallback_to_cache(
    tool: str,
    input_str: str,
    original_error: Exception,
) -> Optional[dict]:
    """
    F2: 尝试读取本地缓存。
    成功返回缓存 dict，失败（过期/未命中）返回 None。
    """
    printer.warn(tool, f"[F2] 尝试读取本地缓存（key={tool}:{input_str[:60]}）")
    key = cache.make_key(tool, input_str)
    data = cache.get(key)
    if data:
        cached_at = data.get("_cached_at", "unknown")
        printer.warn(tool, f"[F2] 命中缓存 (cached_at={cached_at})，使用缓存数据")
        # 注入 fallback_info
        data["fallback_info"] = {
            "source": "local_cache",
            "original_error": str(original_error),
            "cached_at": cached_at,
        }
        data["source"] = "local_cache"
        data["status"] = "fallback"
        return data
    else:
        printer.error(tool, "[F2] 无有效缓存（未命中或已过期）")
        return None


def fallback_to_error(
    tool: str,
    input_str: str,
    errors: dict,
    fallback_chain: list,
) -> ErrorOutput:
    """
    F3: 所有降级均失败，构建并返回 ErrorOutput。
    """
    printer.error(tool, f"[F3] 所有重试均失败，输出错误报告")
    return ErrorOutput(
        tool=tool,
        input=input_str,
        fallback_chain=fallback_chain,
        errors=errors,
    )
