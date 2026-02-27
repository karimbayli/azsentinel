#!/usr/bin/env bash
# ============================================================
# Sentinel V2 — Generate HMAC secrets for node communication
# ============================================================
set -euo pipefail

echo "🔐 Generating HMAC shared secret..."
SECRET=$(openssl rand -hex 32)
echo ""
echo "HMAC_SECRET=$SECRET"
echo ""
echo "Add this to:"
echo "  1. Central server .env file"
echo "  2. Each probe agent .env or config file"
echo "  3. Keep it in a password manager"
echo ""
echo "⚠️  This secret must be IDENTICAL on all nodes and the central server."
