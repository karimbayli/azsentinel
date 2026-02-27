#!/usr/bin/env bash
# ============================================================
# Sentinel V2 — Probe Node Deployment Script
# Usage: ./deploy-node.sh <node-id> <region> <country> <central-url>
# ============================================================
set -euo pipefail

NODE_ID="${1:?Usage: $0 <node-id> <region> <country> <central-url>}"
REGION="${2:?Usage: $0 <node-id> <region> <country> <central-url>}"
COUNTRY="${3:?Usage: $0 <node-id> <region> <country> <central-url>}"
CENTRAL_URL="${4:?Usage: $0 <node-id> <region> <country> <central-url>}"

echo "🚀 Deploying Sentinel V2 Probe Agent: $NODE_ID ($REGION, $COUNTRY)"

# Create sentinel user if not exists
if ! id sentinel &>/dev/null; then
    echo "👤 Creating sentinel user..."
    sudo useradd -r -s /bin/false sentinel
fi

# Create data directory
sudo mkdir -p /var/lib/sentinel
sudo chown sentinel:sentinel /var/lib/sentinel

# Copy binary (assumes it's built locally or from Docker)
echo "📦 Building probe agent..."
CGO_ENABLED=1 go build -ldflags="-s -w" -o /tmp/sentinel-probe ./cmd/probe-agent/
sudo cp /tmp/sentinel-probe /usr/local/bin/sentinel-probe
sudo chmod +x /usr/local/bin/sentinel-probe

# Create config directory
sudo mkdir -p /etc/sentinel

# Generate config
echo "📝 Generating configuration..."
cat > /tmp/probe.yaml <<EOF
node_id: "${NODE_ID}"
region: "${REGION}"
country: "${COUNTRY}"
central_url: "${CENTRAL_URL}"
hmac_secret: "${HMAC_SECRET:-change-me-in-production}"
probe_interval: 60s
log_level: "info"

buffer:
  db_path: "/var/lib/sentinel/buffer.db"
  max_size: 10000

targets:
  - url: "https://e-gov.az"
    category: "GOV"
    criticality: 10
    enabled: true
    display_name: "E-Government Portal"
  - url: "https://asan.gov.az"
    category: "GOV"
    criticality: 10
    enabled: true
    display_name: "ASAN Service"
  - url: "https://taxes.gov.az"
    category: "GOV"
    criticality: 8
    enabled: true
    display_name: "Tax Service"
  - url: "https://president.az"
    category: "GOV"
    criticality: 7
    enabled: true
    display_name: "President of Azerbaijan"
  - url: "https://abb-bank.az"
    category: "BANK"
    criticality: 9
    enabled: true
    display_name: "ABB Bank"
  - url: "https://pashabank.az"
    category: "BANK"
    criticality: 9
    enabled: true
    display_name: "PASHA Bank"
  - url: "https://kapitalbank.az"
    category: "BANK"
    criticality: 8
    enabled: true
    display_name: "Kapital Bank"
  - url: "https://rabitabank.az"
    category: "BANK"
    criticality: 7
    enabled: true
    display_name: "Rabitabank"
  - url: "https://delta.az"
    category: "ISP"
    criticality: 7
    enabled: true
    display_name: "Delta Telecom"
  - url: "https://aztelekom.az"
    category: "ISP"
    criticality: 7
    enabled: true
    display_name: "Aztelekom"
  - url: "https://bakcell.az"
    category: "ISP"
    criticality: 6
    enabled: true
    display_name: "Bakcell"
  - url: "https://nar.az"
    category: "ISP"
    criticality: 6
    enabled: true
    display_name: "Nar Mobile"
  - url: "https://oxu.az"
    category: "MEDIA"
    criticality: 6
    enabled: true
    display_name: "Oxu.az"
  - url: "https://report.az"
    category: "MEDIA"
    criticality: 6
    enabled: true
    display_name: "Report.az"
  - url: "https://1news.az"
    category: "MEDIA"
    criticality: 5
    enabled: true
    display_name: "1news.az"
  - url: "https://1.1.1.1"
    category: "ANCHOR"
    criticality: 0
    enabled: true
    display_name: "Cloudflare DNS"
  - url: "https://google.com"
    category: "ANCHOR"
    criticality: 0
    enabled: true
    display_name: "Google"
  - url: "https://cloudflare.com"
    category: "ANCHOR"
    criticality: 0
    enabled: true
    display_name: "Cloudflare"
EOF
sudo cp /tmp/probe.yaml /etc/sentinel/probe.yaml

# Install systemd service
echo "⚙️  Installing systemd service..."
sudo cp deployments/probe-agent.service /etc/systemd/system/sentinel-probe.service
sudo systemctl daemon-reload
sudo systemctl enable sentinel-probe
sudo systemctl start sentinel-probe

echo ""
echo "✅ Probe agent deployed as systemd service!"
echo "   Check status: sudo systemctl status sentinel-probe"
echo "   View logs:    sudo journalctl -u sentinel-probe -f"
