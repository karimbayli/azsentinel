#!/usr/bin/env bash
# ============================================================
# Sentinel V2 — Probe Node Quick Deploy (Docker)
# Usage: ./deploy-probe.sh
# Reads configuration from .env file in current directory
# ============================================================
set -euo pipefail

# Colors
RED='\033[0;31m'; GREEN='\033[0;32m'; CYAN='\033[0;36m'; NC='\033[0m'

echo -e "${CYAN}╔══════════════════════════════════════╗${NC}"
echo -e "${CYAN}║  Sentinel V2 — Probe Agent Deploy    ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════╝${NC}"

# ── Check prerequisites ──
command -v docker >/dev/null 2>&1 || {
    echo -e "${RED}Docker not found. Installing...${NC}"
    curl -fsSL https://get.docker.com | sh
    sudo usermod -aG docker "$USER"
    echo -e "${GREEN}Docker installed. Please log out and back in, then re-run this script.${NC}"
    exit 1
}

# ── Check .env exists ──
if [ ! -f .env ]; then
    echo -e "${RED}Error: .env file not found. Create it first:${NC}"
    cat <<'EXAMPLE'

cat > .env << 'EOF'
NODE_ID=node-az
REGION=az-baku
COUNTRY=AZ
CENTRAL_URL=https://updown.az
HMAC_SECRET=<your-shared-secret>
EOF

EXAMPLE
    exit 1
fi

# ── Load and validate .env ──
source .env
echo -e "\n${CYAN}Configuration:${NC}"
echo "  Node ID:     ${NODE_ID:-not set}"
echo "  Region:      ${REGION:-not set}"
echo "  Country:     ${COUNTRY:-not set}"
echo "  Central URL: ${CENTRAL_URL:-not set}"
echo ""

[ -z "${NODE_ID:-}" ] && echo -e "${RED}NODE_ID is required${NC}" && exit 1
[ -z "${HMAC_SECRET:-}" ] && echo -e "${RED}HMAC_SECRET is required${NC}" && exit 1
[ -z "${CENTRAL_URL:-}" ] && echo -e "${RED}CENTRAL_URL is required${NC}" && exit 1

# ── Create docker-compose.yml ──
cat > docker-compose.yml <<'COMPOSE'
services:
  probe:
    image: ghcr.io/${PROBE_IMAGE:-karimbayli/azsentinel/probe}:latest
    container_name: sentinel-probe-${NODE_ID:-probe}
    restart: unless-stopped
    env_file: .env
    environment:
      SENTINEL_NODE_ID: ${NODE_ID}
      SENTINEL_REGION: ${REGION}
      SENTINEL_COUNTRY: ${COUNTRY}
      SENTINEL_CENTRAL_URL: ${CENTRAL_URL}
      SENTINEL_HMAC_SECRET: ${HMAC_SECRET}
    volumes:
      - probe_data:/var/lib/sentinel
    deploy:
      resources:
        limits:
          memory: 128M
          cpus: '0.25'
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"

volumes:
  probe_data:
COMPOSE

# ── Deploy ──
echo -e "${CYAN}Pulling latest probe image...${NC}"
docker compose pull 2>/dev/null || true

echo -e "${CYAN}Starting probe agent...${NC}"
docker compose up -d

echo ""
echo -e "${GREEN}╔══════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  Probe agent deployed successfully!  ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════╝${NC}"
echo ""
echo "  Check logs:   docker compose logs -f"
echo "  Check status: docker compose ps"
echo "  Stop:         docker compose down"
echo ""
