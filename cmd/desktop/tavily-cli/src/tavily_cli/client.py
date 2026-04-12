"""
client.py — Tavily HTTP 客户端封装，统一错误分类
"""
import time
import httpx
from . import config


class TavilyAPIError(Exception):
    """基础 API 错误"""
    def __init__(self, message: str, status_code: int = 0):
        super().__init__(message)
        self.status_code = status_code


class APIKeyError(TavilyAPIError):
    """401 — API Key 无效或未授权"""


class RateLimitError(TavilyAPIError):
    """429 — 请求频率超限"""


class ClientError(TavilyAPIError):
    """4xx — 客户端请求错误"""


class ServerError(TavilyAPIError):
    """5xx — 服务端错误"""


def post(endpoint: str, payload: dict, timeout: int = 30) -> dict:
    """
    向 Tavily API 发起 POST 请求。
    - endpoint: 例如 "/search" 或 "/crawl"
    - payload: 请求体 dict
    - timeout: 超时秒数
    返回解析后的响应 dict，失败时抛出对应异常。
    """
    api_key = config.get_api_key()
    url = f"{config.API_BASE_URL}{endpoint}"
    headers = {
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json",
    }

    try:
        with httpx.Client(timeout=timeout) as client:
            resp = client.post(url, json=payload, headers=headers)
    except httpx.TimeoutException as e:
        raise TavilyAPIError(f"请求超时（{timeout}s）：{e}") from e
    except httpx.ConnectError as e:
        raise TavilyAPIError(f"连接失败（DNS/网络错误）：{e}") from e
    except httpx.RequestError as e:
        raise TavilyAPIError(f"请求异常：{e}") from e

    # 状态码分类
    code = resp.status_code
    if code == 200:
        try:
            return resp.json()
        except Exception as e:
            raise TavilyAPIError(f"响应 JSON 解析失败：{e}") from e
    elif code == 401:
        raise APIKeyError("API Key 无效或未授权（401）", status_code=code)
    elif code == 429:
        raise RateLimitError("请求频率超限（429），请稍后重试", status_code=code)
    elif 400 <= code < 500:
        raise ClientError(f"客户端错误（{code}）：{resp.text[:200]}", status_code=code)
    else:
        raise ServerError(f"服务端错误（{code}）：{resp.text[:200]}", status_code=code)
