#!/usr/bin/env bash
set -Eeuo pipefail

APP_NAME="XboardNode-Plus"
REPO="xboardnext999/XboardNode-Plus"
INSTALL_ROOT="/etc/XboardNode-Plus"
CONFIG_FILE="${INSTALL_ROOT}/config.yml"
CREDENTIALS_FILE="${INSTALL_ROOT}/credentials.env"
META_FILE="${INSTALL_ROOT}/install-meta.json"
BINARY_PATH="/usr/local/bin/xboard-node"
CLI_PATH="/usr/local/bin/xbctl"
SERVICE_NAME="xboard-node.service"
SERVICE_PATH="/etc/systemd/system/${SERVICE_NAME}"

MODE="machine"
PANEL_URL=""
TOKEN=""
MACHINE_ID=""
NODE_ID=""
NODE_TYPE=""
DOMAIN=""
VERSION="latest"
KERNEL="auto"
HEALTH_PORT="65530"
TEMPLATE="random"
FALLBACK_TARGET="127.0.0.1:8080"
TROJAN_UPSTREAM="127.0.0.1:10443"
WEB_UPSTREAM="127.0.0.1:8443"
SKIP_NGINX=0

log() { printf '[%s] %s\n' "$1" "$2"; }
die() { log ERROR "$1" >&2; exit 1; }

usage() {
    cat <<'HELP'
XboardNode-Plus easy installer

Usage:
  sudo bash easy-install.sh --panel URL --token TOKEN --machine-id ID --domain DOMAIN

Options:
  --mode machine|node          Install mode (default: machine)
  --panel URL                  Panel URL
  --token TOKEN                Machine or node token
  --machine-id ID              Machine ID for machine mode
  --node-id ID                 Node ID for node mode
  --node-type TYPE             Optional node type for node mode
  --domain DOMAIN              Public Trojan SNI / camouflage domain
  --version VERSION            Release version or latest (default: latest)
  --kernel auto|singbox|xray   Kernel type (default: auto)
  --health-port PORT           Health port, 0 to disable (default: 65530)
  --template NAME              Camouflage template: random, cloud, studio, docs, commerce, status
  --fallback-target HOST:PORT  Local camouflage HTTP target (default: 127.0.0.1:8080)
  --trojan-upstream HOST:PORT  Local Trojan TLS inbound (default: 127.0.0.1:10443)
  --web-upstream HOST:PORT     Local HTTPS upstream for unmatched SNI (default: 127.0.0.1:8443)
  --skip-nginx                 Install node only, skip nginx camouflage/SNI setup
  --help                       Show help

Example:
  curl -fL https://github.com/xboardnext999/XboardNode-Plus/releases/latest/download/easy-install.sh | sudo bash -s -- \
    --panel https://www.baiyunfast.com \
    --token YOUR_TOKEN \
    --machine-id 12 \
    --domain sg3.oone.us
HELP
}

while [ $# -gt 0 ]; do
    case "$1" in
        --mode) MODE="$2"; shift 2 ;;
        --panel|-a) PANEL_URL="$2"; shift 2 ;;
        --token|-t) TOKEN="$2"; shift 2 ;;
        --machine-id) MACHINE_ID="$2"; shift 2 ;;
        --node-id|-n) NODE_ID="$2"; shift 2 ;;
        --node-type|-T) NODE_TYPE="$2"; shift 2 ;;
        --domain) DOMAIN="$2"; shift 2 ;;
        --version) VERSION="$2"; shift 2 ;;
        --kernel|-k) KERNEL="$2"; shift 2 ;;
        --health-port) HEALTH_PORT="$2"; shift 2 ;;
        --template) TEMPLATE="$2"; shift 2 ;;
        --fallback-target) FALLBACK_TARGET="$2"; shift 2 ;;
        --trojan-upstream) TROJAN_UPSTREAM="$2"; shift 2 ;;
        --web-upstream) WEB_UPSTREAM="$2"; shift 2 ;;
        --skip-nginx) SKIP_NGINX=1; shift ;;
        --help|-h) usage; exit 0 ;;
        *) die "Unknown option: $1" ;;
    esac
done

[ "$(id -u)" -eq 0 ] || die "Please run with sudo or as root"
[ -n "$PANEL_URL" ] || die "--panel is required"
[ -n "$TOKEN" ] || die "--token is required"

case "$MODE" in
    machine)
        [ -n "$MACHINE_ID" ] || die "--machine-id is required in machine mode"
        ;;
    node)
        [ -n "$NODE_ID" ] || die "--node-id is required in node mode"
        ;;
    *)
        die "--mode must be machine or node"
        ;;
esac

if [ "$SKIP_NGINX" -eq 0 ]; then
    [ -n "$DOMAIN" ] || die "--domain is required unless --skip-nginx is used"
fi

ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) die "Unsupported architecture: $ARCH" ;;
esac

if [ "$VERSION" = "latest" ]; then
    ASSET_BASE="https://github.com/${REPO}/releases/latest/download"
else
    ASSET_BASE="https://github.com/${REPO}/releases/download/${VERSION}"
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

download_asset() {
    local name="$1"
    local dest="$2"
    log STEP "Downloading ${name}"
    curl -fL --retry 5 --retry-delay 2 -o "$dest" "${ASSET_BASE}/${name}"
    chmod +x "$dest"
}

install_packages() {
    if command -v apt-get >/dev/null 2>&1; then
        export DEBIAN_FRONTEND=noninteractive
        apt-get update
        apt-get install -y ca-certificates curl nginx libnginx-mod-stream
    else
        log WARN "apt-get not found; please install curl, nginx, and nginx stream module manually"
    fi
}

ensure_nginx_stream_include() {
    [ "$SKIP_NGINX" -eq 0 ] || return 0
    mkdir -p /etc/nginx/stream.d
    if command -v nginx >/dev/null 2>&1 && nginx -T 2>/dev/null | grep -q '/etc/nginx/stream.d/\*.conf'; then
        return 0
    fi
    if grep -qE '^[[:space:]]*stream[[:space:]]*\{' /etc/nginx/nginx.conf 2>/dev/null; then
        log WARN "nginx.conf already has a stream block; ensure it includes /etc/nginx/stream.d/*.conf"
        return 0
    fi
    cat >> /etc/nginx/nginx.conf <<'EOF_NGINX_STREAM'

stream {
    include /etc/nginx/stream.d/*.conf;
}
EOF_NGINX_STREAM
}

write_service() {
    cat > "$SERVICE_PATH" <<EOF_SERVICE
[Unit]
Description=Xboard Node Backend
Documentation=https://github.com/${REPO}
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_ROOT}
EnvironmentFile=-${CREDENTIALS_FILE}
ExecStart=${BINARY_PATH} -c ${CONFIG_FILE}
Restart=always
RestartSec=5
LimitNOFILE=1048576
NoNewPrivileges=true
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF_SERVICE
}

render_config() {
    local args=(
        config init
        --mode "$MODE"
        --panel-url "$PANEL_URL"
        --kernel "$KERNEL"
        --health-port "$HEALTH_PORT"
        --token "$TOKEN"
        --version "$VERSION"
        --output "$TMP_DIR/config.yml"
        --credentials-out "$TMP_DIR/credentials.env"
        --meta "$TMP_DIR/install-meta.json"
        --install-root "$INSTALL_ROOT"
    )

    [ -f "$CONFIG_FILE" ] && args+=(--config "$CONFIG_FILE")
    [ -f "$CREDENTIALS_FILE" ] && args+=(--credentials-in "$CREDENTIALS_FILE")

    if [ "$MODE" = "machine" ]; then
        args+=(--machine-id "$MACHINE_ID")
    else
        args+=(--node-id "$NODE_ID")
        [ -n "$NODE_TYPE" ] && args+=(--node-type "$NODE_TYPE")
    fi

    "$TMP_DIR/xbctl" "${args[@]}"
}

install_node() {
    mkdir -p "$INSTALL_ROOT"
    download_asset "xboard-node-linux-${ARCH}" "$TMP_DIR/xboard-node"
    download_asset "xbctl-linux-${ARCH}" "$TMP_DIR/xbctl"

    render_config

    systemctl stop "$SERVICE_NAME" >/dev/null 2>&1 || true
    install -m 755 "$TMP_DIR/xboard-node" "$BINARY_PATH"
    install -m 755 "$TMP_DIR/xbctl" "$CLI_PATH"
    ln -sf "$CLI_PATH" /usr/bin/xbctl 2>/dev/null || true
    install -m 600 "$TMP_DIR/config.yml" "$CONFIG_FILE"
    install -m 600 "$TMP_DIR/credentials.env" "$CREDENTIALS_FILE"
    install -m 644 "$TMP_DIR/install-meta.json" "$META_FILE"
    write_service

    systemctl daemon-reload
    systemctl enable --now "$SERVICE_NAME"
}

configure_nginx() {
    [ "$SKIP_NGINX" -eq 0 ] || return 0
    log STEP "Configuring camouflage site"
    "$CLI_PATH" nginx camouflage-site \
        --domain "$DOMAIN" \
        --listen "$FALLBACK_TARGET" \
        --template "$TEMPLATE" \
        --write --force --test --reload

    log STEP "Configuring Trojan fallback"
    "$CLI_PATH" config trojan-fallback --target "$FALLBACK_TARGET" --restart

    log STEP "Configuring nginx SNI passthrough"
    "$CLI_PATH" nginx trojan-sni \
        --domain "$DOMAIN" \
        --trojan-upstream "$TROJAN_UPSTREAM" \
        --web-upstream "$WEB_UPSTREAM" \
        --write --force --test --reload
}

install_packages
ensure_nginx_stream_include
install_node
configure_nginx

log INFO "Installation complete"
log INFO "Service: ${SERVICE_NAME}"
log INFO "Config: ${CONFIG_FILE}"
log INFO "Health: http://127.0.0.1:${HEALTH_PORT}/healthz"
if [ "$SKIP_NGINX" -eq 0 ]; then
    log INFO "Camouflage: https://${DOMAIN}/"
fi
