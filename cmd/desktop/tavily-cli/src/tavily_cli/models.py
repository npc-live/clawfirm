"""
models.py — Pydantic 输入/输出模型，对应执行计划 Schema F
"""
from __future__ import annotations
from typing import Any, Optional
from datetime import datetime, timezone
import json
from pydantic import BaseModel, field_validator, model_validator
from urllib.parse import urlparse


# ─────────────────────────── 输入模型 ───────────────────────────

class SearchParams(BaseModel):
    query: str
    search_depth: str = "basic"
    max_results: int = 5
    include_domains: list[str] = []
    exclude_domains: list[str] = []
    include_answer: bool = True
    include_raw_content: bool = False
    include_images: bool = False
    topic: str = "general"

    @field_validator("search_depth")
    @classmethod
    def validate_depth(cls, v: str) -> str:
        if v not in ("basic", "advanced"):
            raise ValueError("search_depth 必须是 'basic' 或 'advanced'")
        return v

    @field_validator("max_results")
    @classmethod
    def validate_max(cls, v: int) -> int:
        if not (1 <= v <= 20):
            raise ValueError("max_results 必须在 1-20 之间")
        return v

    def to_payload(self) -> dict[str, Any]:
        d: dict[str, Any] = {
            "query": self.query,
            "search_depth": self.search_depth,
            "max_results": self.max_results,
            "include_answer": self.include_answer,
            "include_raw_content": self.include_raw_content,
            "include_images": self.include_images,
            "topic": self.topic,
        }
        if self.include_domains:
            d["include_domains"] = self.include_domains
        if self.exclude_domains:
            d["exclude_domains"] = self.exclude_domains
        return d


class CrawlParams(BaseModel):
    url: str
    max_depth: int = 1
    max_breadth: int = 20
    limit: int = 50
    extract_depth: str = "basic"
    allow_external: bool = False
    select_paths: list[str] = []
    exclude_paths: list[str] = []

    @field_validator("url")
    @classmethod
    def validate_url(cls, v: str) -> str:
        parsed = urlparse(v)
        if not parsed.scheme or not parsed.netloc:
            raise ValueError(f"无效 URL（需包含 scheme 和 netloc）：{v}")
        return v

    def to_payload(self) -> dict[str, Any]:
        d: dict[str, Any] = {
            "url": self.url,
            "max_depth": self.max_depth,
            "max_breadth": self.max_breadth,
            "limit": self.limit,
            "extract_depth": self.extract_depth,
            "allow_external": self.allow_external,
        }
        if self.select_paths:
            d["select_paths"] = self.select_paths
        if self.exclude_paths:
            d["exclude_paths"] = self.exclude_paths
        return d


# ─────────────────────────── 输出模型 ───────────────────────────

class FallbackInfo(BaseModel):
    source: str  # "fallback_search" | "local_cache"
    original_error: str
    cached_at: Optional[str] = None


class SearchResult(BaseModel):
    title: str
    url: str
    content: str = ""
    score: float = 0.0
    raw_content: Optional[str] = None
    published_date: Optional[str] = None


class SearchOutput(BaseModel):
    status: str = "success"          # success | fallback | error
    source: str = "api"              # api | fallback_search | local_cache
    tool: str = "search"
    query: str
    timestamp: str = ""
    response_time_ms: float = 0.0
    answer: Optional[str] = None
    results: list[SearchResult] = []
    images: list[dict] = []
    fallback_info: Optional[FallbackInfo] = None

    def model_post_init(self, __context: Any) -> None:
        if not self.timestamp:
            self.timestamp = datetime.now(timezone.utc).isoformat()

    def to_json(self) -> str:
        return self.model_dump_json(indent=2)


class CrawlPage(BaseModel):
    url: str
    title: Optional[str] = None
    content: str = ""
    raw_content: Optional[str] = None
    depth: int = 0
    failed: bool = False
    failed_reason: Optional[str] = None


class CrawlSummary(BaseModel):
    total_pages: int = 0
    failed_pages: int = 0
    max_depth_reached: int = 0


class CrawlOutput(BaseModel):
    status: str = "success"
    source: str = "api"
    tool: str = "crawl"
    base_url: str
    timestamp: str = ""
    response_time_ms: float = 0.0
    results: list[CrawlPage] = []
    summary: CrawlSummary = CrawlSummary()
    fallback_info: Optional[FallbackInfo] = None

    def model_post_init(self, __context: Any) -> None:
        if not self.timestamp:
            self.timestamp = datetime.now(timezone.utc).isoformat()

    def to_json(self) -> str:
        return self.model_dump_json(indent=2)


class ErrorOutput(BaseModel):
    status: str = "error"
    tool: str
    input: str
    timestamp: str = ""
    fallback_chain: list[str] = []
    errors: dict[str, Optional[str]] = {}

    def model_post_init(self, __context: Any) -> None:
        if not self.timestamp:
            self.timestamp = datetime.now(timezone.utc).isoformat()

    def to_json(self) -> str:
        return self.model_dump_json(indent=2)
