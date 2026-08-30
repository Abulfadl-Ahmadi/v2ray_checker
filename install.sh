#!/usr/bin/env bash
set -e

# ==============================================================================
# V2Ray Checker & Health Service - One-Click Installer for Linux VPS
# Repository: https://github.com/Abulfadl-Ahmadi/v2ray_checker
# ==============================================================================

REPO="Abulfadl-Ahmadi/v2ray_checker"
INSTALL_DIR="/opt/v2ray_checker"
SERVICE_NAME="v2ray_checker"

echo "========================================================"
echo "🚀 Installing V2Ray Checker Service..."
echo "========================================================"

# 1. Detect Architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)
        BINARY_NAME="v2ray_checker_linux_amd64"
        ;;
    aarch64|arm64)
        BINARY_NAME="v2ray_checker_linux_arm64"
        ;;
    *)
        echo "❌ Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

echo "ℹ️ Detected system architecture: $ARCH ($BINARY_NAME)"

# 2. Check for root privileges
if [ "$EUID" -ne 0 ]; then
    echo "❌ Please run this script as root (e.g. sudo bash install.sh)"
    exit 1
fi

# 3. Install prerequisites
echo "📦 Checking required tools (curl, wget, ca-certificates)..."
if command -v apt-get >/dev/null 2>&1; then
    apt-get update -qq && apt-get install -y -qq curl wget ca-certificates
elif command -v yum >/dev/null 2>&1; then
    yum install -y -q curl wget ca-certificates
fi

# 4. Create installation directory structure
echo "📁 Creating installation directory at $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR/data"
cd "$INSTALL_DIR"

# 5. Stop existing service if running
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "🔄 Stopping existing $SERVICE_NAME service..."
    systemctl stop "$SERVICE_NAME"
fi

# 6. Download latest binary from GitHub Releases
DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/$BINARY_NAME"
echo "⬇️ Downloading latest binary from $DOWNLOAD_URL..."
curl -fsSL -L "$DOWNLOAD_URL" -o "$INSTALL_DIR/v2ray_checker"

chmod +x "$INSTALL_DIR/v2ray_checker"

# 7. Create config.yaml if it does not exist
if [ ! -f "$INSTALL_DIR/config.yaml" ]; then
    echo "⚙️ Creating default config.yaml..."
    cat << 'EOF' > "$INSTALL_DIR/config.yaml"
server:
  port: ":8080"

worker:
  check_interval: 15m
  concurrency: 30
  timeout_sec: 4
  max_failures: 3

probe:
  urls:
    - "https://1.1.1.1/cdn-cgi/trace"
    - "https://www.google.com/generate_204"

database:
  path: "./data/v2ray.db"

collector:
  channels_file: "./channels.csv"
  subscription_urls:
    - "https://raw.githubusercontent.com/Abulfadl-Ahmadi/V2rayCollector/refs/heads/main/mixed_iran.txt"
EOF
else
    echo "ℹ️ Existing config.yaml preserved."
fi

# 8. Create empty channels.csv if missing
if [ ! -f "$INSTALL_DIR/channels.csv" ]; then
    echo "URL,AllMessagesFlag" > "$INSTALL_DIR/channels.csv"
fi

# 9. Create Systemd Service
echo "🛠️ Creating systemd service unit..."
cat << EOF > /etc/systemd/system/${SERVICE_NAME}.service
[Unit]
Description=V2Ray Automated Proxy Collector & Health Checker Service
After=network.target network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/v2ray_checker -config $INSTALL_DIR/config.yaml
Restart=always
RestartSec=5s
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

# 10. Enable & Start Service
echo "⚡ Reloading systemd and starting service..."
systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl restart "$SERVICE_NAME"

sleep 2

# 11. Verification
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "========================================================"
    echo "✅ V2Ray Checker installed and running successfully!"
    echo "========================================================"
    echo "📡 REST JSON API:   http://YOUR_SERVER_IP:8080/api/nodes"
    echo "🔗 Client Sub Link:  http://YOUR_SERVER_IP:8080/sub/all"
    echo "📊 Service Stats:    http://YOUR_SERVER_IP:8080/api/stats"
    echo ""
    echo "Useful commands:"
    echo "  - Check status:  systemctl status $SERVICE_NAME"
    echo "  - View live log: journalctl -u $SERVICE_NAME -f"
    echo "  - Restart:       systemctl restart $SERVICE_NAME"
    echo "========================================================"
else
    echo "⚠️ Service started but may have encountered an error. Check logs with:"
    echo "journalctl -u $SERVICE_NAME -n 30"
fi
