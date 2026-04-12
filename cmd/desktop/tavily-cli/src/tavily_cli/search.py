"""
search.py — Search 工具业务逻辑（9 步流程）
"""
import time
import sys
from typing import Union

from . import client, cache, printer
from .models import SearchParams, SearchOutput, SearchResult, ErrorOutput
from . import fallback as fb


def run(params: SearchParams) -> Union[SearchOutput, ErrorOutput]:
    """
    执行 Tavily Search，含完整 Fallback 链路。
    结果写入缓存，最终返回 SearchOutput 或 ErrorOutput。
    """
    tool = "search"

    # Step 3: 状态打印
    printer.status(tool, "START", f"开始搜索: '{params.query}'")

    # Step 4-5: HTTP 请求
    t0 = time.time()
    try:
        raw = client.post("/search", params.to_payload(), timeout=30)
    except Exception as e:
        printer.error(tool, f"Search API 请求失败：{e}")
        # F2: 尝试缓存
        cached = fb.fallback_to_cache(tool, params.query, e)
        if cached:
            try:
                out = SearchOutput(**{k: v for k, v in cached.items() if not k.startswith("_")})
                printer.success(tool, "使用缓存数据返回")
                return out
            except Exception:
                pass
        # F3: 错误报告
        return fb.fallback_to_error(
            tool=tool,
            input_str=params.query,
            errors={"primary": str(e), "fallback1": None, "fallback2": "缓存未命中或解析失败"},
            fallback_chain=["search_failed", "cache_miss"],
        )

    elapsed = (time.time() - t0) * 1000  # ms

    # Step 6: Schema 映射
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
        status="success",
        source="api",
        query=params.query,
        response_time_ms=round(elapsed, 2),
        answer=raw.get("answer"),
        results=results,
        images=raw.get("images", []),
    )

    # Step 7: 写缓存
    cache.set(cache.make_key(tool, params.query), raw)

    # Step 8: 完成状态
    printer.success(tool, f"搜索完成，共 {len(results)} 条结果（{elapsed:.0f}ms）")

    return output
