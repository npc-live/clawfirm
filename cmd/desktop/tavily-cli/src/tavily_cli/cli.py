"""
cli.py — typer CLI 入口，注册 tavily-search 和 tavily-crawl 命令
"""
import sys
from typing import Optional
import typer

from . import printer
from .models import SearchParams, CrawlParams, ErrorOutput
from . import search as search_mod
from . import crawl as crawl_mod

# ─── Search App ─────────────────────────────────────────────────
search_app = typer.Typer(help="Tavily Search CLI — 通过 Tavily API 搜索网页")

@search_app.command()
def search_cmd(
    query: str = typer.Argument(..., help="搜索词"),
    search_depth: str = typer.Option("basic", "--search-depth", "-d", help="搜索深度: basic | advanced"),
    max_results: int = typer.Option(5, "--max-results", "-n", min=1, max=20, help="最大结果数 (1-20)"),
    include_answer: bool = typer.Option(True, "--include-answer/--no-answer", help="是否包含 AI 摘要答案"),
    include_raw_content: bool = typer.Option(False, "--raw-content/--no-raw-content", help="是否包含原始内容"),
    include_images: bool = typer.Option(False, "--images/--no-images", help="是否包含图片"),
    topic: str = typer.Option("general", "--topic", "-t", help="话题类型: general | news"),
) -> None:
    try:
        params = SearchParams(
            query=query,
            search_depth=search_depth,
            max_results=max_results,
            include_answer=include_answer,
            include_raw_content=include_raw_content,
            include_images=include_images,
            topic=topic,
        )
    except Exception as e:
        printer.error("search", f"参数错误：{e}")
        raise typer.Exit(code=2)

    result = search_mod.run(params)
    printer.output_json(result.to_json())

    if isinstance(result, ErrorOutput):
        raise typer.Exit(code=1)


def search_main() -> None:
    search_app()


# ─── Crawl App ──────────────────────────────────────────────────
crawl_app = typer.Typer(help="Tavily Crawl CLI — 通过 Tavily API 抓取网页内容")

@crawl_app.command()
def crawl_cmd(
    url: str = typer.Argument(..., help="目标 URL（需包含 http:// 或 https://）"),
    max_depth: int = typer.Option(1, "--max-depth", help="爬取深度"),
    max_breadth: int = typer.Option(20, "--max-breadth", help="同层最大链接数"),
    limit: int = typer.Option(50, "--limit", "-l", help="最大页面数"),
    extract_depth: str = typer.Option("basic", "--extract-depth", help="提取深度: basic | advanced"),
    allow_external: bool = typer.Option(False, "--allow-external/--no-external", help="是否允许爬取外部链接"),
) -> None:
    try:
        params = CrawlParams(
            url=url,
            max_depth=max_depth,
            max_breadth=max_breadth,
            limit=limit,
            extract_depth=extract_depth,
            allow_external=allow_external,
        )
    except Exception as e:
        printer.error("crawl", f"参数错误：{e}")
        raise typer.Exit(code=2)

    result = crawl_mod.run(params)
    printer.output_json(result.to_json())

    if isinstance(result, ErrorOutput):
        raise typer.Exit(code=1)


def crawl_main() -> None:
    crawl_app()
