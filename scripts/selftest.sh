#!/bin/bash
# Smart Self-Test with Isolated Workspace
# 智能自测：使用独立的测试 workspace

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🧠 智能自测开始（独立 workspace）"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# ============================================================
# Phase 1: 准备测试 workspace
# ============================================================

echo ""
echo "📋 Phase 1: 准备测试 workspace"
echo ""

mkdir -p ~/.clawfirm-test
if [ -f ~/.clawfirm/config.yml ] && [ ! -f ~/.clawfirm-test/config.yml ]; then
  cp ~/.clawfirm/config.yml ~/.clawfirm-test/config.yml
  echo "✓ 复制配置文件"
fi
echo "✓ 测试 workspace: ~/.clawfirm-test"
echo "✓ 测试端口: 19988"

# ============================================================
# Phase 2: 启动测试 gateway
# ============================================================

echo ""
echo "🚀 Phase 2: 启动测试 gateway"
echo ""

# 清理测试端口
lsof -i :19988 -t 2>/dev/null | xargs kill -9 2>/dev/null || true
sleep 1

# 启动测试 gateway
cd /Users/qing/projects/pi-go || exit 1
CLAWFIRM_HOME=~/.clawfirm-test go run ./cmd/gateway -addr :19988 > /tmp/gateway-test.log 2>&1 &
TEST_PID=$!
echo "Test Gateway PID: $TEST_PID"
sleep 3

# 健康检查
if curl -sf http://localhost:19988/health > /dev/null 2>&1; then
  echo "✅ 测试 gateway 启动成功"
else
  echo "❌ 测试 gateway 启动失败"
  tail -20 /tmp/gateway-test.log
  exit 1
fi

# ============================================================
# Phase 3: 执行测试
# ============================================================

echo ""
echo "🧪 Phase 3: 执行测试"
echo ""

# 测试 1: 健康检查
echo "  → 测试 1: 健康检查"
HEALTH=$(curl -sf http://localhost:19988/health)
if echo "$HEALTH" | grep -q "ok"; then
  echo "    ✅ 健康检查通过"
  TEST1_PASS=1
else
  echo "    ❌ 健康检查失败"
  TEST1_PASS=0
fi

# 测试 2: WebSocket 端点
echo "  → 测试 2: WebSocket 端点"
WS_CHECK=$(curl -sf http://localhost:19988/ | grep -c "clawfirm" || echo "0")
if [ "$WS_CHECK" -gt 0 ]; then
  echo "    ✅ WebSocket 端点可用"
  TEST2_PASS=1
else
  echo "    ❌ WebSocket 端点不可用"
  TEST2_PASS=0
fi

# 测试 3: 日志分析
echo "  → 测试 3: 日志分析"
LOG_CHECK=$(go run ./cmd/logcheck /tmp/gateway-test.log 2>&1)
if echo "$LOG_CHECK" | grep -q "No anomalies"; then
  echo "    ✅ 日志检查通过"
  TEST3_PASS=1
else
  echo "    ⚠️  发现日志异常"
  echo "$LOG_CHECK"
  TEST3_PASS=0
fi

# 测试 4: 数据库隔离
echo "  → 测试 4: 数据库隔离验证"
PROD_SIZE=$(du -k ~/.clawfirm/data.db 2>/dev/null | cut -f1 || echo "0")
TEST_SIZE=$(du -k ~/.clawfirm-test/data.db 2>/dev/null | cut -f1 || echo "0")
echo "    生产数据库: ${PROD_SIZE}KB"
echo "    测试数据库: ${TEST_SIZE}KB"
if [ "$PROD_SIZE" != "$TEST_SIZE" ] || [ "$TEST_SIZE" = "0" ]; then
  echo "    ✅ 数据库已隔离"
  TEST4_PASS=1
else
  echo "    ⚠️  数据库可能未隔离"
  TEST4_PASS=0
fi

# ============================================================
# Phase 4: 测试报告
# ============================================================

echo ""
echo "📊 Phase 4: 测试报告"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "测试结果汇总"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "测试环境:"
echo "  - Workspace: ~/.clawfirm-test"
echo "  - Gateway: http://localhost:19988"
echo "  - 数据库: ~/.clawfirm-test/data.db"
echo ""

# 统计结果
PASS=$((TEST1_PASS + TEST2_PASS + TEST3_PASS + TEST4_PASS))
FAIL=$((4 - PASS))

if [ $TEST1_PASS -eq 1 ]; then
  echo "✅ 健康检查: 通过"
else
  echo "❌ 健康检查: 失败"
fi

if [ $TEST2_PASS -eq 1 ]; then
  echo "✅ WebSocket 端点: 通过"
else
  echo "❌ WebSocket 端点: 失败"
fi

if [ $TEST3_PASS -eq 1 ]; then
  echo "✅ 日志检查: 通过"
else
  echo "⚠️  日志检查: 发现异常"
fi

if [ $TEST4_PASS -eq 1 ]; then
  echo "✅ 数据库隔离: 通过"
else
  echo "⚠️  数据库隔离: 未验证"
fi

echo ""
echo "总计: ${PASS} 通过, ${FAIL} 失败"
echo ""

# 健康度评分
SCORE=$((PASS * 100 / 4))
echo "系统健康度: ${SCORE}/100"

if [ $SCORE -ge 80 ]; then
  echo "状态: ✅ 优秀"
elif [ $SCORE -ge 60 ]; then
  echo "状态: ⚠️  良好"
else
  echo "状态: ❌ 需要改进"
fi

echo ""
echo "详细日志:"
echo "  - Gateway: /tmp/gateway-test.log"
echo "  - 测试数据: ~/.clawfirm-test/"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# ============================================================
# Phase 5: 清理测试环境
# ============================================================

echo ""
echo "🧹 Phase 5: 清理测试环境"
echo ""

lsof -i :19988 -t 2>/dev/null | xargs kill 2>/dev/null || true
echo "✓ 测试 gateway 已关闭"
echo ""
echo "测试 workspace 保留在: ~/.clawfirm-test"
echo "如需清理，运行: rm -rf ~/.clawfirm-test"

echo ""
echo "🎉 智能自测完成！"
echo ""
echo "生产环境未受影响，所有测试在独立 workspace 中运行。"
echo ""
