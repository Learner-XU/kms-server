#!/bin/bash
# KMS 一键启动脚本
# 用法: ./start.sh

KMS_DIR="$(cd "$(dirname "$0")" && pwd)"
WEB_DIR="$KMS_DIR/../kms-web"
BACKEND_PORT=8000
FRONTEND_PORT=3456

# 自动检测局域网 IP
LAN_IP=$(ipconfig getifaddr en0 2>/dev/null || ipconfig getifaddr en1 2>/dev/null || echo "127.0.0.1")

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() { echo -e "${GREEN}[KMS]${NC} $1"; }
warn() { echo -e "${YELLOW}[KMS]${NC} $1"; }
err() { echo -e "${RED}[KMS]${NC} $1"; }

# --- 清理已有进程 ---
cleanup_port() {
    local port=$1
    local pid
    pid=$(lsof -ti :"$port" 2>/dev/null || true)
    if [ -n "$pid" ]; then
        warn "端口 $port 被进程 $pid 占用，正在停止..."
        kill "$pid" 2>/dev/null || true
        sleep 1
        kill -9 "$pid" 2>/dev/null || true
    fi
}

log "========== KMS 启动 =========="

# --- 清理 ---
log "检查并清理旧进程..."
cleanup_port $BACKEND_PORT
cleanup_port $FRONTEND_PORT
log "清理完成"

# --- 启动后端 ---
log "启动后端 (:$BACKEND_PORT)..."
cd "$KMS_DIR"
if [ ! -f ".env" ]; then
    err "未找到 .env 文件，请先配置"
    exit 1
fi
nohup ./kms-server > "$KMS_DIR/kms-server.log" 2>&1 &
BACKEND_PID=$!
sleep 2

if kill -0 "$BACKEND_PID" 2>/dev/null; then
    log "后端已启动 (PID: $BACKEND_PID)"
else
    err "后端启动失败，查看日志: $KMS_DIR/kms-server.log"
    exit 1
fi

# --- 启动前端 ---
log "启动前端 (:$FRONTEND_PORT)..."
cd "$WEB_DIR"
if [ ! -d ".next" ]; then
    log "首次运行，先构建前端..."
    npx next build
fi
nohup npx next start -p $FRONTEND_PORT -H 0.0.0.0 > "$KMS_DIR/kms-web.log" 2>&1 &
FRONTEND_PID=$!
sleep 3

if kill -0 "$FRONTEND_PID" 2>/dev/null; then
    log "前端已启动 (PID: $FRONTEND_PID)"
else
    err "前端启动失败，查看日志: $KMS_DIR/kms-web.log"
    exit 1
fi

# --- 完成 ---
echo ""
log "========== KMS 启动完成 =========="
log "后端: http://$LAN_IP:$BACKEND_PORT"
log "前端: http://$LAN_IP:$FRONTEND_PORT"
log "后端PID: $BACKEND_PID | 前端PID: $FRONTEND_PID"
log "停止服务: $KMS_DIR/stop.sh"
echo ""
