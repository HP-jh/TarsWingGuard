#!/bin/bash
# WingGuard 健康检查
set -e
# 检查进程存活
if ! pgrep -f wingguard >/dev/null 2>&1; then
  echo "FAIL: WingGuard process not found"
  exit 1
fi
echo "OK: WingGuard is running"
