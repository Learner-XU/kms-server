#!/bin/bash
# KMS 一键停止脚本
# 用法: ./stop.sh

KMS_DIR="$(cd "$(dirname "$0")" && pwd)"
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

log() { echo -e "${GREEN}[KMS]${NC} $1"; }

log "正在停止 KMS 服务..."

# 停止前端
FRONTEND_PID=$(lsof -ti :3456 2>/dev/null)
if [ -n "$FRONTEND_PID" ]; then
    kill "$FRONTEND_PID" 2>/dev/null
    log "前端已停止 (PID: $FRONTEND_PID)"
fi

# 停止后端
BACKEND_PID=$(lsof -ti :8000 2>/dev/null)
if [ -n "$BACKEND_PID" ]; then
    kill "$BACKEND_PID" 2>/dev/null
    log "后端已停止 (PID: $BACKEND_PID)"
fi

log "KMS 服务已停止"
