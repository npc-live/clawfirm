# 环境配置 TODO 清单

> 生成时间：2026-03-19
> 平台：macOS 15.6.1 (Sequoia)

---

## 待办项

### ❌ TODO 1 — 安装 Tauri CLI

**状态：** 缺失
**原因：** `cargo tauri` 命令不存在

**安装命令：**
```bash
cargo install tauri-cli
```

**验证命令：**
```bash
cargo tauri --version
```

> 注意：需要从源码编译，预计耗时 5–10 分钟。

---

### ⚠️ TODO 2 — 确认 pandas 是否已安装

**状态：** 未确认
**原因：** 检测时未返回版本信息

**确认命令：**
```bash
pip3 show pandas
```

**若未安装，执行：**
```bash
pip3 install pandas
```

---

## 已通过项（无需操作）

| 组件 | 版本 | 备注 |
|---|---|---|
| Rust / cargo | 1.92.0 | ✅ |
| Node.js | v20.19.2 | ✅ >= 18 |
| pnpm | 10.29.3 | ✅ |
| npm | 10.8.2 | ✅ |
| Python3 | 3.13.2 | ✅ |
| requests | 2.32.3 | ✅ |
| numpy | 2.2.6 | ✅ |
| WebKit / WebView | macOS 内置 | ✅ |

---

## 快速修复脚本

```bash
# 1. 安装 Tauri CLI
cargo install tauri-cli

# 2. 确认 pandas
pip3 show pandas || pip3 install pandas

# 3. 验证 Tauri CLI
cargo tauri --version
```
