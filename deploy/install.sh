#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Check root
if [[ $EUID -ne 0 ]]; then
    error "请使用 root 运行此脚本: sudo ./install.sh"
fi

INSTALL_DIR="/opt/kms"
SERVICE_USER="kms"
SERVICE_GROUP="kms"
BINARY="kms-server"
SERVICE_FILE="kms-server.service"
ENV_FILE=".env.example"

info "KMS 知识管理系统 - SUSE 安装脚本"
echo "=================================="

# Create user/group
if ! id "$SERVICE_USER" &>/dev/null; then
    info "创建用户 $SERVICE_USER ..."
    useradd -r -s /sbin/nologin -d "$INSTALL_DIR" "$SERVICE_USER"
fi

# Create directory
info "创建安装目录 $INSTALL_DIR ..."
mkdir -p "$INSTALL_DIR"

# Copy binary
info "安装二进制文件 ..."
cp "$BINARY" "$INSTALL_DIR/$BINARY"
chmod 755 "$INSTALL_DIR/$BINARY"

# Copy env example if .env doesn't exist
if [[ ! -f "$INSTALL_DIR/.env" ]]; then
    if [[ -f "$ENV_FILE" ]]; then
        cp "$ENV_FILE" "$INSTALL_DIR/.env"
        chmod 600 "$INSTALL_DIR/.env"
        warn "已复制 .env.example → .env，请编辑配置: $INSTALL_DIR/.env"
    fi
fi

# Set permissions
chown -R "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR"

# Install systemd service
info "安装 systemd 服务 ..."
cp "$SERVICE_FILE" /etc/systemd/system/
systemctl daemon-reload

# Enable and start
info "启用服务 ..."
systemctl enable kms-server

info "安装完成！"
echo ""
echo "后续步骤:"
echo "  1. 编辑配置:  vi $INSTALL_DIR/.env"
echo "  2. 启动服务:  systemctl start kms-server"
echo "  3. 查看状态:  systemctl status kms-server"
echo "  4. 查看日志:  journalctl -u kms-server -f"
echo ""
