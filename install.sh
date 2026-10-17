#!/usr/bin/env bash
set -euo pipefail

# TarsWingGuard One-Line Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/HP-jh/TarsWingGuard/main/install.sh | bash

REPO="HP-jh/TarsWingGuard"
VERSION="${VERSION:-0.1.0}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect platform
ARCH=$(uname -m)
OS=$(uname -s | tr '[:upper:]' '[:lower:]')

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

BINARY="twg-v${VERSION}-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${BINARY}"

echo "========================================"
echo "  TarsWingGuard Installer v${VERSION}"
echo "========================================"
echo ""

# Step 1: welcome
echo "[1/6] 欢迎使用 TarsWingGuard 安装向导"
echo "      平台: ${OS}/${ARCH}"
echo ""

# Step 2: check dependencies
echo "[2/6] 检查依赖..."
MISSING=""
command -v curl >/dev/null 2>&1 || MISSING="$MISSING curl"
if [ -n "$MISSING" ]; then
  echo "      缺少: $MISSING"
  echo "      请先安装缺失的依赖"
  exit 1
fi
echo "      依赖检查通过"
echo ""

# Step 3: download
echo "[3/6] 下载 ${BINARY}..."
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -fsSL -o "${TMPDIR}/twg" "$URL" || {
  echo "      下载失败，尝试备用地址..."
  curl -fsSL -o "${TMPDIR}/twg" "${URL}.tar.gz" && tar -xzf "${TMPDIR}/twg" -C "$TMPDIR"
}
chmod +x "${TMPDIR}/twg"
echo "      下载完成"
echo ""

# Step 4: verify checksum (if available)
echo "[4/6] 校验文件..."
CHECKSUM_URL="https://github.com/${REPO}/releases/download/v${VERSION}/SHA256SUMS.txt"
if curl -fsSL -o "${TMPDIR}/SHA256SUMS.txt" "$CHECKSUM_URL" 2>/dev/null; then
  cd "$TMPDIR"
  if sha256sum -c SHA256SUMS.txt --ignore-missing 2>/dev/null | grep -q "OK"; then
    echo "      校验通过"
  else
    echo "      警告: 校验和验证失败，继续安装（风险自负）"
  fi
  cd - >/dev/null
else
  echo "      无校验和文件，跳过"
fi
echo ""

# Step 5: install
echo "[5/6] 安装到 ${INSTALL_DIR}..."
if [ -w "$INSTALL_DIR" ]; then
  cp "${TMPDIR}/twg" "${INSTALL_DIR}/twg"
  cp "${TMPDIR}/twg" "${INSTALL_DIR}/wing-guard" 2>/dev/null || true
else
  echo "      需要管理员权限..."
  sudo cp "${TMPDIR}/twg" "${INSTALL_DIR}/twg"
  sudo cp "${TMPDIR}/twg" "${INSTALL_DIR}/wing-guard" 2>/dev/null || true
fi
echo "      安装完成"
echo ""

# Step 6: setup config
echo "[6/6] 初始化配置..."
CONFIG_DIR="${HOME}/.config/tarswingguard"
mkdir -p "$CONFIG_DIR"
if [ ! -f "${CONFIG_DIR}/wing.yaml" ]; then
  cat > "${CONFIG_DIR}/wing.yaml" <<'EOF'
version: "0.1.0"
schema: wing/v1
core:
  name: TarsWingCore
  description: 我的网关
  version: "0.1.0"
modules:
  - id: wing-guard
    name: "羽翼防护"
    version: "0.1.0"
    tier: pro
    transport: stdio
    command: wing-guard
    deps: [core-waf]
EOF
  echo "      已创建默认配置: ${CONFIG_DIR}/wing.yaml"
else
  echo "      配置已存在，跳过"
fi
echo ""

echo "========================================"
echo "  安装完成!"
echo "========================================"
echo ""
echo "  快速开始:"
echo "    twg -list              列出所有模块"
echo "    twg -resolve pro       查看 pro 套餐模块"
echo "    twg -tiers             查看套餐对比"
echo "    wing-guard -version    查看羽翼防护版本"
echo ""
echo "  启用羽翼防护:"
echo "    在 TSG config.json 中添加:"
echo '    {"wing": {"mode": "tier", "tier": "pro"}}'
echo ""
echo "  文档: https://github.com/${REPO}"
