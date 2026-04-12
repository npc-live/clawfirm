"""
crawl.py — Crawl 工具业务逻辑（9 步流程，含三层 Fallback）
"""
import time
import json
from typing import Union

from . import client, cache, printer
from .models import CrawlParams, CrawlOutput, CrawlPage, CrawlSummary, SearchOutput, ErrorOutput
from . import fallback as fb


def run(params: CrawlParams) -> Union[CrawlOutput, SearchOutput, ErrorOutput]:
    """
    执行 Tavily Crawl，含完整三层 Fallback 链路。
    """
    tool = "crawl"
    errors: dict = {}

    # Step 3: 状态打印
    printer.status(tool, "START", f"开始抓取: '{params.url}'")

    # Step 4-5: HTTP 请求
    t0 = time.time()
    raw = None
    primary_error = None

    try:
        raw = client.post("/crawl", params.to_payload(), timeout=60)
    except Exception as e:
        primary_error = e
        errors["primary"] = str(e)
        printer.error(tool, f"Crawl API 请求失败：{e}")

    elapsed = (time.time() - t0) * 1000

    if raw is not None:
        # Step 6: Schema 映射
        raw_results = raw.get("results", [])
        pages = [
            CrawlPage(
                url=r.get("url", ""),
                title=r.get("title"),
                content=r.get("content") or r.get("raw_content", ""),
                raw_content=r.get("raw_content"),
                depth=r.get("depth", 0),
                failed=r.get("failed", False),
                failed_reason=r.get("failed_reason"),
            )
            for r in raw_results
        ]
        failed_count = sum(1 for p in pages if p.failed)
        max_depth = max((p.depth for p in pages), default=0)

        output = CrawlOutput(
            status="success",
            source="api",
            base_url=params.url,
            response_time_ms=round(elapsed, 2),
            results=pages,
            summary=CrawlSummary(
                total_pages=len(pages),
                failed_pages=failed_count,
                max_depth_reached=max_depth,
            ),
        )

        # Step 7: 写缓存
        cache.set(cache.make_key(tool, params.url), raw)

        # Step 8: 完成状态
        printer.success(
            tool,
            f"抓取完成，共 {len(pages)} 个页面，失败 {failed_count} 个（{elapsed:.0f}ms）",
        )
        return output

    # ── F1: Search 降级 ──────────────────────────────────────────
    fallback_chain = ["crawl_failed"]
    f1_result = fb.fallback_to_search(params.url, primary_error, tool=tool)
    if f1_result is not None:
        return f1_result

    errors["fallback1"] = "Search 降级也失败"
    fallback_chain.append("search_failed")

    # ── F2: 读取缓存 ─────────────────────────────────────────────
    cached = fb.fallback_to_cache(tool, params.url, primary_error)
    if cached:
        try:
            # 尝试还原为 CrawlOutput
            cached_clean = {k: v for k, v in cached.items() if not k.startswith("_")}
            out = CrawlOutput(**cached_clean)
            printer.success(tool, "使用缓存数据返回")
            return out
        except Exception:
            pass

    errors["fallback2"] = "缓存未命中或已过期"
    fallback_chain.append("cache_miss")

    # ── F3: 错误报告 ──────────────────────────────────────────────
    return fb.fallback_to_error(
        tool=tool,
        input_str=params.url,
        errors=errors,
        fallback_chain=fallback_chain,
    )
