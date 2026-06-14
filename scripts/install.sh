#!/usr/bin/env bash
#
# frp-panel 一键部署脚本 (Linux · systemd)
# One-click installer for frp-panel.
#
#   curl -fsSL https://raw.githubusercontent.com/MengStar-L/frp-panel/master/scripts/install.sh | sudo bash
#
# 可用环境变量覆盖默认值 / overridable via env vars:
#   INSTALL_DIR  安装目录   (默认 /opt/frp-panel)
#   PORT         监听端口   (默认 8088)
#   VERSION      指定版本   (默认 latest,如 v1.0.0)
#   REPO         GitHub 仓库 (默认 MengStar-L/frp-panel)
#
set -euo pipefail

REPO="${REPO:-MengStar-L/frp-panel}"
INSTALL_DIR="${INSTALL_DIR:-/opt/frp-panel}"
PORT="${PORT:-8088}"
VERSION="${VERSION:-}"
SERVICE="frp-panel"

say()  { printf '\033[36m▸\033[0m %s\n' "$*"; }
err()  { printf '\033[31m✗ %s\033[0m\n' "$*" >&2; }
die()  { err "$*"; exit 1; }

# --- 前置检查 / pre-flight -------------------------------------------------
[ "$(uname -s)" = "Linux" ] || die "此脚本仅支持 Linux。"
[ "$(id -u)" -eq 0 ]        || die "请用 root 运行(例如: curl ... | sudo bash)。"
command -v curl >/dev/null  || die "需要 curl,请先安装。"
command -v tar  >/dev/null  || die "需要 tar,请先安装。"
command -v systemctl >/dev/null || die "未检测到 systemd (systemctl),无法注册服务。"

# --- 识别架构 / detect arch ------------------------------------------------
case "$(uname -m)" in
  x86_64|amd64)   ARCH=amd64 ;;
  aarch64|arm64)  ARCH=arm64 ;;
  *) die "不支持的 CPU 架构: $(uname -m)(仅支持 x86_64 / arm64)。" ;;
esac

# --- 解析版本 / resolve version --------------------------------------------
if [ -z "$VERSION" ]; then
  say "查询最新版本 ..."
  VERSION="$(curl -fsSLI -o /dev/null -w '%{url_effective}' \
    "https://github.com/${REPO}/releases/latest" | sed -E 's#.*/tag/##' || true)"
fi
[ -n "$VERSION" ] || die "无法获取版本号,可手动指定: VERSION=v1.0.0 ..."

ASSET="frp-panel_${VERSION}_linux_${ARCH}.tar.gz"
BASE="https://github.com/${REPO}/releases/download/${VERSION}"

say "版本   : ${VERSION}"
say "架构   : linux/${ARCH}"
say "安装到 : ${INSTALL_DIR}"
say "端口   : ${PORT}"

# --- 升级前停止旧服务 / stop existing service ------------------------------
if systemctl list-unit-files | grep -q "^${SERVICE}.service"; then
  say "停止已存在的 ${SERVICE} 服务 ..."
  systemctl stop "${SERVICE}" 2>/dev/null || true
fi

# --- 下载 + 校验 + 解压 / download, verify, extract ------------------------
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

say "下载   : ${BASE}/${ASSET}"
curl -fsSL "${BASE}/${ASSET}" -o "${TMP}/${ASSET}" || die "下载失败。"

if command -v sha256sum >/dev/null; then
  if curl -fsSL "${BASE}/checksums.txt" -o "${TMP}/checksums.txt"; then
    say "校验 SHA256 ..."
    ( cd "$TMP" && grep "\*${ASSET}\$" checksums.txt | sha256sum -c - >/dev/null ) \
      || die "SHA256 校验失败,文件可能损坏或被篡改。"
    say "SHA256 校验通过。"
  else
    err "未取到 checksums.txt,跳过校验。"
  fi
fi

tar -xzf "${TMP}/${ASSET}" -C "$TMP"
[ -f "${TMP}/frp-panel" ] || die "压缩包内未找到 frp-panel 可执行文件。"

mkdir -p "$INSTALL_DIR"
install -m 0755 "${TMP}/frp-panel" "${INSTALL_DIR}/frp-panel"

# --- 注册 systemd 服务 / register systemd unit -----------------------------
say "写入 systemd 服务 ..."
cat > "/etc/systemd/system/${SERVICE}.service" <<EOF
[Unit]
Description=frp-panel - frp management panel
Documentation=https://github.com/${REPO}
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/frp-panel -dir ${INSTALL_DIR} -addr :${PORT}
Restart=always
RestartSec=5s
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now "${SERVICE}" \
  || die "服务启动失败,请排查: journalctl -u ${SERVICE} -e"

# --- 完成提示 / done -------------------------------------------------------
IP="$(hostname -I 2>/dev/null | awk '{print $1}' || true)"; IP="${IP:-<服务器IP>}"
printf '\n\033[32m✅ 安装完成!\033[0m\n'
echo "   控制台 : http://${IP}:${PORT}"
echo "   状态   : systemctl status ${SERVICE}"
echo "   日志   : journalctl -u ${SERVICE} -f"
echo "   重启   : systemctl restart ${SERVICE}"
