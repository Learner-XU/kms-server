#!/bin/bash
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

info() { echo -e "${GREEN}[INFO]${NC} $1"; }

if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}[ERROR]${NC} 请使用 root: sudo ./uninstall.sh"
    exit 1
fi

info "停止服务 ..."
systemctl stop kms-server 2>/dev/null || true
systemctl disable kms-server 2>/dev/null || true

info "删除服务文件 ..."
rm -f /etc/systemd/system/kms-server.service
systemctl daemon-reload

read -p "是否删除 /opt/kms 及所有数据？[y/N] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    info "删除安装目录 ..."
    rm -rf /opt/kms
    userdel kms 2>/dev/null || true
    info "已完全卸载"
else
    info "保留数据目录 /opt/kms"
fi

info "卸载完成"
