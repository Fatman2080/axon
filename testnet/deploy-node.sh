#!/bin/bash
set -e

# ============================================================
# Axon Testnet — Single Validator Node Deployment Script
#
# Run on a fresh Ubuntu 22.04+ server:
#   curl -sSL https://raw.githubusercontent.com/Fatman2080/axon/main/testnet/deploy-node.sh | bash
#
# Or download and customize:
#   wget https://raw.githubusercontent.com/Fatman2080/axon/main/testnet/deploy-node.sh
#   chmod +x deploy-node.sh
#   MONIKER="my-node" SEEDS="..." ./deploy-node.sh
# ============================================================

AXON_HOME="${AXON_HOME:-/opt/axon}"
CHAIN_ID="${CHAIN_ID:-axon_9001-1}"
MONIKER="${MONIKER:-axon-validator-$(hostname -s)}"
SEEDS="${SEEDS:-}"
GENESIS_URL="${GENESIS_URL:-}"
GO_VERSION="1.23.4"
DENOM="aaxon"

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

log() { echo -e "${CYAN}[AXON]${NC} $1"; }
ok()  { echo -e "${GREEN}[OK]${NC} $1"; }
err() { echo -e "${RED}[ERR]${NC} $1"; exit 1; }

# ── 1. System dependencies ──────────────────────────────────
log "Installing system dependencies..."
sudo apt-get update -qq
sudo apt-get install -y -qq build-essential git curl jq python3 \
    ufw fail2ban > /dev/null 2>&1
ok "System dependencies installed"

# ── 2. Install Go ────────────────────────────────────────────
if ! command -v go &>/dev/null || [[ "$(go version)" != *"$GO_VERSION"* ]]; then
    log "Installing Go $GO_VERSION..."
    wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
    rm "go${GO_VERSION}.linux-amd64.tar.gz"

    echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' | sudo tee /etc/profile.d/go.sh > /dev/null
    export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin
    ok "Go $(go version | awk '{print $3}') installed"
else
    ok "Go already installed: $(go version | awk '{print $3}')"
fi

# ── 3. Build axond ───────────────────────────────────────────
log "Building axond from source..."
REPO_DIR="/tmp/axon-build"
rm -rf "$REPO_DIR"
git clone --depth 1 https://github.com/Fatman2080/axon.git "$REPO_DIR"
cd "$REPO_DIR"
make build
sudo cp build/axond /usr/local/bin/axond
ok "axond built and installed: $(axond version 2>/dev/null || echo 'ok')"

# ── 4. Initialize node ──────────────────────────────────────
log "Initializing node (moniker=$MONIKER)..."
sudo mkdir -p "$AXON_HOME"
sudo chown "$(whoami)" "$AXON_HOME"

if [ -f "$AXON_HOME/config/genesis.json" ]; then
    log "Node already initialized, skipping init."
else
    axond init "$MONIKER" --chain-id "$CHAIN_ID" --home "$AXON_HOME" 2>/dev/null
    ok "Node initialized at $AXON_HOME"
fi

# ── 5. Download or patch genesis ─────────────────────────────
if [ -n "$GENESIS_URL" ]; then
    log "Downloading genesis from $GENESIS_URL..."
    curl -sSL "$GENESIS_URL" -o "$AXON_HOME/config/genesis.json"
    ok "Genesis downloaded"
else
    log "Patching local genesis for testnet..."
    python3 "$REPO_DIR/testnet/genesis-patch.py" "$AXON_HOME/config/genesis.json" "$DENOM"
    ok "Genesis patched"
fi

# ── 6. Configure peers ──────────────────────────────────────
CONFIG="$AXON_HOME/config/config.toml"
APP_TOML="$AXON_HOME/config/app.toml"

if [ -n "$SEEDS" ]; then
    log "Setting seeds: ${SEEDS:0:60}..."
    sed -i "s|seeds = \"\"|seeds = \"$SEEDS\"|" "$CONFIG"
fi

# CometBFT RPC listens on localhost only — use a reverse proxy for public access
sed -i 's|laddr = "tcp://0.0.0.0:26657"|laddr = "tcp://127.0.0.1:26657"|' "$CONFIG"
sed -i 's|addr_book_strict = true|addr_book_strict = false|' "$CONFIG"
sed -i 's|prometheus = false|prometheus = true|' "$CONFIG"
sed -i 's|address = "tcp://localhost:1317"|address = "tcp://0.0.0.0:1317"|' "$APP_TOML"
sed -i 's|address = "localhost:9090"|address = "0.0.0.0:9090"|' "$APP_TOML"
sed -i "s|minimum-gas-prices = \"\"|minimum-gas-prices = \"10000000000${DENOM}\"|" "$APP_TOML"

ok "Node configured"

# ── 7. Firewall ──────────────────────────────────────────────
log "Configuring firewall..."
sudo ufw --force reset > /dev/null 2>&1
sudo ufw default deny incoming > /dev/null 2>&1
sudo ufw default allow outgoing > /dev/null 2>&1
sudo ufw allow 22/tcp    > /dev/null 2>&1   # SSH
sudo ufw allow 26656/tcp > /dev/null 2>&1   # P2P
sudo ufw allow 8545/tcp  > /dev/null 2>&1   # JSON-RPC
sudo ufw allow 8546/tcp  > /dev/null 2>&1   # WebSocket
sudo ufw allow 1317/tcp  > /dev/null 2>&1   # REST API
sudo ufw allow 9090/tcp  > /dev/null 2>&1   # gRPC
sudo ufw allow 26660/tcp > /dev/null 2>&1   # Prometheus
sudo ufw --force enable  > /dev/null 2>&1
ok "Firewall configured (SSH, P2P, JSON-RPC, API open; CometBFT RPC localhost-only)"

# ── 8. Systemd service ──────────────────────────────────────
log "Creating systemd service..."

sudo tee /etc/systemd/system/axond.service > /dev/null << SVCEOF
[Unit]
Description=Axon Chain Node
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$(whoami)
ExecStart=/usr/local/bin/axond start \\
    --home $AXON_HOME \\
    --chain-id $CHAIN_ID \\
    --json-rpc.enable \\
    --json-rpc.address 0.0.0.0:8545 \\
    --json-rpc.ws-address 0.0.0.0:8546 \\
    --json-rpc.api eth,txpool,net,web3 \\
    --api.enable \\
    --api.address tcp://0.0.0.0:1317
Restart=always
RestartSec=5
LimitNOFILE=65535

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=axond

[Install]
WantedBy=multi-user.target
SVCEOF

sudo systemctl daemon-reload
sudo systemctl enable axond
ok "Systemd service created (axond.service)"

# ── 9. Summary ───────────────────────────────────────────────
NODE_ID=$(axond comet show-node-id --home "$AXON_HOME" 2>/dev/null \
          || axond tendermint show-node-id --home "$AXON_HOME" 2>/dev/null \
          || echo "unknown")

IP=$(curl -s ifconfig.me 2>/dev/null || hostname -I | awk '{print $1}')

echo ""
echo -e "${GREEN}============================================${NC}"
echo -e "${GREEN}  Axon Node Deployed!${NC}"
echo -e "${GREEN}============================================${NC}"
echo ""
echo "  Chain ID:     $CHAIN_ID"
echo "  Moniker:      $MONIKER"
echo "  Home:         $AXON_HOME"
echo "  Node ID:      $NODE_ID"
echo "  Public IP:    $IP"
echo ""
echo "  Peer address: ${NODE_ID}@${IP}:26656"
echo ""
echo "  Endpoints:"
echo "    P2P:        $IP:26656"
echo "    CometBFT:   http://127.0.0.1:26657 (localhost only)"
echo "    JSON-RPC:   http://$IP:8545"
echo "    WebSocket:  ws://$IP:8546"
echo "    REST API:   http://$IP:1317"
echo "    gRPC:       $IP:9090"
echo "    Prometheus: http://$IP:26660"
echo ""
echo "  Commands:"
echo "    sudo systemctl start axond    # Start node"
echo "    sudo systemctl stop axond     # Stop node"
echo "    sudo journalctl -fu axond     # View logs"
echo ""
echo "  To create a validator (after syncing):"
echo "    axond keys add mykey --home $AXON_HOME"
echo "    axond tx staking create-validator \\"
echo "      --amount=10000000000000000000000000aaxon \\"
echo "      --pubkey=\$(axond comet show-validator --home $AXON_HOME) \\"
echo "      --moniker=$MONIKER \\"
echo "      --chain-id=$CHAIN_ID \\"
echo "      --commission-rate=0.10 \\"
echo "      --commission-max-rate=0.20 \\"
echo "      --commission-max-change-rate=0.01 \\"
echo "      --min-self-delegation=1 \\"
echo "      --from=mykey \\"
echo "      --home=$AXON_HOME"
echo ""

# Clean up
rm -rf "$REPO_DIR"
