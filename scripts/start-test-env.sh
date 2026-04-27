#!/bin/bash
# scripts/start-test-env.sh
# 启动测试环境：gateway(:9988) + vite dev(:5173)
# 用法：./scripts/start-test-env.sh

set -e
cd "$(dirname "$0")/.."

echo "=== 检查 :9988 是否已被占用 ==="
if lsof -i :9988 -t > /dev/null 2>&1; then
  echo "⚠  :9988 已被占用，跳过启动 gateway"
else
  echo "▶  启动 gateway on :9988 ..."
  go run ./cmd/gateway > /tmp/gateway.log 2>&1 &
  GATEWAY_PID=$!
  echo $GATEWAY_PID > /tmp/gateway.pid
  echo "   PID=$GATEWAY_PID  log=/tmp/gateway.log"

  # 等 gateway 就绪
  for i in $(seq 1 20); do
    if curl -sf http://localhost:9988/ > /dev/null 2>&1; then
      echo "✔  gateway ready"
      break
    fi
    sleep 0.5
  done
fi

echo ""
echo "=== 检查 :5173 是否已被占用 ==="
if lsof -i :5173 -t > /dev/null 2>&1; then
  echo "⚠  :5173 已被占用，跳过启动 vite"
else
  echo "▶  启动 vite dev server on :5173 ..."
  cd cmd/desktop/frontend
  npm run dev > /tmp/vite.log 2>&1 &
  VITE_PID=$!
  echo $VITE_PID > /tmp/vite.pid
  cd ../../..
  echo "   PID=$VITE_PID  log=/tmp/vite.log"

  # 等 vite 就绪
  for i in $(seq 1 30); do
    if curl -sf http://localhost:5173/ > /dev/null 2>&1; then
      echo "✔  vite ready"
      break
    fi
    sleep 0.5
  done
fi

echo ""
echo "✔  测试环境就绪"
echo "   Web UI: http://localhost:5173"
echo "   Gateway: http://localhost:9988"
echo ""
echo "运行场景："
echo "  go run ./cmd/browser-shortcut ~/.clawfirm/shortcuts/clawfirm_test.yaml basic_chat '你好'"
echo "  go run ./cmd/browser-shortcut ~/.clawfirm/shortcuts/clawfirm_test.yaml send_then_stop"
