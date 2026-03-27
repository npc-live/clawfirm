"""
printer.py — 状态打印工具（状态到 stderr，结果到 stdout）
"""
import sys
import json
from rich.console import Console
from rich.syntax import Syntax

_err_console = Console(stderr=True)
_out_console = Console()


def status(tool: str, step: str, message: str) -> None:
    """打印当前步骤状态到 stderr，格式：[TOOL:STEP] message"""
    _err_console.print(f"[bold cyan][{tool.upper()}:{step}][/bold cyan] {message}")


def success(tool: str, message: str) -> None:
    _err_console.print(f"[bold green][{tool.upper()}:DONE][/bold green] ✅ {message}")


def warn(tool: str, message: str) -> None:
    _err_console.print(f"[bold yellow][{tool.upper()}:WARN][/bold yellow] ⚠️  {message}")


def error(tool: str, message: str) -> None:
    _err_console.print(f"[bold red][{tool.upper()}:ERROR][/bold red] ❌ {message}")


def output_json(data_json: str) -> None:
    """将 JSON 结果输出到 stdout，带语法高亮。"""
    syntax = Syntax(data_json, "json", theme="monokai", line_numbers=False)
    _out_console.print(syntax)
