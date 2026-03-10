#!/bin/bash
set -e

# ============================================================
# Axon Local 4-Node Testnet
# Creates a 4-validator local network for testing
# ============================================================

BINARY="./build/axond"
CHAIN_ID="axon_9001-1"
DENOM="aaxon"
KEYRING="test"
NUM_NODES=4
BASE_DIR="$HOME/.axon-localnet"
BASE_P2P_PORT=26656
BASE_RPC_PORT=26657
BASE_API_PORT=1317
BASE_GRPC_PORT=9090
BASE_JSONRPC_PORT=8545

echo "============================================"
echo "  Axon Local 4-Node Testnet Setup"
echo "============================================"

# Build binary
if [ ! -f "$BINARY" ]; then
    echo "==> Building axond..."
    go build -o build/axond ./cmd/axond/
fi

# Clean previous state
rm -rf "$BASE_DIR"
mkdir -p "$BASE_DIR"

# ---- Step 1: Initialize all nodes ----
echo ""
echo "==> Step 1: Initializing $NUM_NODES nodes..."

for i in $(seq 0 $((NUM_NODES-1))); do
    NODE_DIR="$BASE_DIR/node$i"
    MONIKER="axon-node-$i"
    
    $BINARY init $MONIKER --chain-id $CHAIN_ID --home "$NODE_DIR" 2>/dev/null
    
    # Create validator key
    $BINARY keys add "validator$i" --keyring-backend $KEYRING --home "$NODE_DIR" 2>/dev/null
    
    echo "    Node $i initialized (moniker=$MONIKER)"
done

# ---- Step 2: Patch genesis (use node0's genesis as template) ----
echo ""
echo "==> Step 2: Patching genesis.json..."

GENESIS="$BASE_DIR/node0/config/genesis.json"

python3 - "$GENESIS" "$DENOM" <<'PYEOF'
import json, sys

genesis_path = sys.argv[1]
denom = sys.argv[2]

with open(genesis_path) as f:
    genesis = json.load(f)

app = genesis["app_state"]

app["staking"]["params"]["bond_denom"] = denom
app["staking"]["params"]["max_validators"] = 100

app["mint"]["params"]["mint_denom"] = denom
app["mint"]["params"]["inflation_max"] = "0.000000000000000000"
app["mint"]["params"]["inflation_min"] = "0.000000000000000000"
app["mint"]["params"]["inflation_rate_change"] = "0.000000000000000000"
app["mint"]["minter"]["inflation"] = "0.000000000000000000"

app["evm"]["params"]["evm_denom"] = denom
if "extended_denom_options" in app["evm"]["params"]:
    app["evm"]["params"]["extended_denom_options"]["extended_denom"] = denom

# Activate Axon custom precompiles
axon_precompiles = [
    "0x0000000000000000000000000000000000000801",
    "0x0000000000000000000000000000000000000802",
    "0x0000000000000000000000000000000000000803",
]
existing = app["evm"]["params"].get("active_static_precompiles", [])
for pc in axon_precompiles:
    if pc not in existing:
        existing.append(pc)
app["evm"]["params"]["active_static_precompiles"] = existing

app["feemarket"]["params"]["no_base_fee"] = True
app["feemarket"]["params"]["min_gas_price"] = "0.000000000000000000"

app["bank"]["denom_metadata"] = [{
    "description": "The native staking and gas token of the Axon network.",
    "denom_units": [
        {"denom": denom, "exponent": 0, "aliases": ["attoaxon"]},
        {"denom": "naxon", "exponent": 9, "aliases": ["nanoaxon"]},
        {"denom": "axon", "exponent": 18, "aliases": ["AXON"]}
    ],
    "base": denom,
    "display": "axon",
    "name": "Axon",
    "symbol": "AXON",
    "uri": "",
    "uri_hash": ""
}]

if "params" in app.get("gov", {}):
    for dep in app["gov"]["params"].get("min_deposit", []):
        dep["denom"] = denom

genesis["consensus"]["params"]["block"]["max_gas"] = "40000000"

with open(genesis_path, "w") as f:
    json.dump(genesis, f, indent=2)

print("    Genesis patched successfully")
PYEOF

# ---- Step 3: Add genesis accounts for all validators ----
echo ""
echo "==> Step 3: Adding genesis accounts..."

# Whitepaper: 0% pre-allocation. Each genesis validator receives only
# the minimum stake + gas buffer. All other tokens come from mining.
BALANCE="11000000000000000000000${DENOM}"     # 11,000 AXON = 10,000 stake + 1,000 gas
STAKE="10000000000000000000000${DENOM}"        # 10,000 AXON minimum validator stake

for i in $(seq 0 $((NUM_NODES-1))); do
    NODE_DIR="$BASE_DIR/node$i"
    ADDR=$($BINARY keys show "validator$i" -a --keyring-backend $KEYRING --home "$NODE_DIR")
    
    $BINARY genesis add-genesis-account "$ADDR" "$BALANCE" \
        --keyring-backend $KEYRING --home "$BASE_DIR/node0"
    
    echo "    Added account for node $i: $ADDR"
done

# ---- Step 4: Create gentx for all validators ----
echo ""
echo "==> Step 4: Creating genesis transactions..."

for i in $(seq 0 $((NUM_NODES-1))); do
    NODE_DIR="$BASE_DIR/node$i"
    
    # Copy the shared genesis to this node
    cp "$BASE_DIR/node0/config/genesis.json" "$NODE_DIR/config/genesis.json"
    
    $BINARY genesis gentx "validator$i" "$STAKE" \
        --chain-id $CHAIN_ID \
        --moniker "axon-node-$i" \
        --keyring-backend $KEYRING \
        --home "$NODE_DIR" 2>/dev/null
    
    # Copy gentx to node0 for collection
    cp "$NODE_DIR/config/gentx/"*.json "$BASE_DIR/node0/config/gentx/" 2>/dev/null || true
    
    echo "    Created gentx for node $i"
done

# ---- Step 5: Collect gentxs and distribute genesis ----
echo ""
echo "==> Step 5: Collecting genesis transactions..."

$BINARY genesis collect-gentxs --home "$BASE_DIR/node0" 2>/dev/null
$BINARY genesis validate-genesis --home "$BASE_DIR/node0"

# Copy final genesis to all nodes
for i in $(seq 1 $((NUM_NODES-1))); do
    cp "$BASE_DIR/node0/config/genesis.json" "$BASE_DIR/node$i/config/genesis.json"
done

echo "    Final genesis distributed to all nodes"

# ---- Step 6: Get node IDs and configure peers ----
echo ""
echo "==> Step 6: Configuring peer connections..."

NODE_IDS=()
for i in $(seq 0 $((NUM_NODES-1))); do
    NODE_DIR="$BASE_DIR/node$i"
    NODE_ID=$($BINARY comet show-node-id --home "$NODE_DIR" 2>/dev/null || $BINARY tendermint show-node-id --home "$NODE_DIR" 2>/dev/null)
    NODE_IDS+=("$NODE_ID")
    echo "    Node $i ID: $NODE_ID"
done

# Build persistent_peers string
PEERS=""
for i in $(seq 0 $((NUM_NODES-1))); do
    P2P_PORT=$((BASE_P2P_PORT + i * 100))
    if [ -n "$PEERS" ]; then
        PEERS="${PEERS},"
    fi
    PEERS="${PEERS}${NODE_IDS[$i]}@127.0.0.1:${P2P_PORT}"
done

echo "    Peers: $PEERS"

# ---- Step 7: Configure each node's ports and peers ----
echo ""
echo "==> Step 7: Configuring node ports..."

for i in $(seq 0 $((NUM_NODES-1))); do
    NODE_DIR="$BASE_DIR/node$i"
    P2P_PORT=$((BASE_P2P_PORT + i * 100))
    RPC_PORT=$((BASE_RPC_PORT + i * 100))
    API_PORT=$((BASE_API_PORT + i * 100))
    GRPC_PORT=$((BASE_GRPC_PORT + i * 100))
    JSONRPC_PORT=$((BASE_JSONRPC_PORT + i * 10))
    JSONRPC_WS_PORT=$((JSONRPC_PORT + 1))
    PPROF_PORT=$((6060 + i))
    
    APP_TOML="$NODE_DIR/config/app.toml"
    CONFIG_TOML="$NODE_DIR/config/config.toml"
    
    if [[ "$OSTYPE" == "darwin"* ]]; then
        SED="sed -i ''"
    else
        SED="sed -i"
    fi
    
    # config.toml: P2P and RPC ports
    eval $SED "'s|laddr = \"tcp://0.0.0.0:26656\"|laddr = \"tcp://0.0.0.0:${P2P_PORT}\"|'" "$CONFIG_TOML"
    eval $SED "'s|laddr = \"tcp://127.0.0.1:26657\"|laddr = \"tcp://127.0.0.1:${RPC_PORT}\"|'" "$CONFIG_TOML"
    eval $SED "'s|pprof_laddr = \"localhost:6060\"|pprof_laddr = \"localhost:${PPROF_PORT}\"|'" "$CONFIG_TOML"
    
    # Remove self from peers, set persistent_peers
    SELF_PEER="${NODE_IDS[$i]}@127.0.0.1:${P2P_PORT}"
    NODE_PEERS=$(echo "$PEERS" | sed "s|${SELF_PEER}||g" | sed 's|,,|,|g' | sed 's|^,||' | sed 's|,$||')
    eval $SED "'s|persistent_peers = \"\"|persistent_peers = \"${NODE_PEERS}\"|'" "$CONFIG_TOML"
    
    # app.toml: API, gRPC, JSON-RPC ports, min gas
    eval $SED "'s|address = \"tcp://localhost:1317\"|address = \"tcp://localhost:${API_PORT}\"|'" "$APP_TOML"
    eval $SED "'s|address = \"localhost:9090\"|address = \"localhost:${GRPC_PORT}\"|'" "$APP_TOML"
    eval $SED "'s|minimum-gas-prices = \"\"|minimum-gas-prices = \"0${DENOM}\"|'" "$APP_TOML"
    
    echo "    Node $i: P2P=$P2P_PORT RPC=$RPC_PORT API=$API_PORT JSONRPC=$JSONRPC_PORT"
done

# ---- Step 8: Generate start script ----
echo ""
echo "==> Step 8: Generating start scripts..."

START_SCRIPT="$BASE_DIR/start_all.sh"
STOP_SCRIPT="$BASE_DIR/stop_all.sh"

cat > "$START_SCRIPT" << 'STARTEOF'
#!/bin/bash
BASE_DIR="$HOME/.axon-localnet"
BINARY="./build/axond"
CHAIN_ID="axon_9001-1"

echo "Starting Axon 4-node localnet..."

for i in 0 1 2 3; do
    NODE_DIR="$BASE_DIR/node$i"
    JSONRPC_PORT=$((8545 + i * 10))
    LOG_FILE="$BASE_DIR/node${i}.log"
    
    echo "  Starting node $i (JSON-RPC: $JSONRPC_PORT)..."
    $BINARY start \
        --home "$NODE_DIR" \
        --chain-id "$CHAIN_ID" \
        --json-rpc.enable \
        --json-rpc.address "0.0.0.0:${JSONRPC_PORT}" \
        --json-rpc.ws-address "0.0.0.0:$((JSONRPC_PORT + 1))" \
        > "$LOG_FILE" 2>&1 &
    
    echo $! > "$BASE_DIR/node${i}.pid"
done

echo ""
echo "============================================"
echo "  Axon Localnet Running!"
echo "============================================"
echo ""
echo "  Node 0: RPC=26657  P2P=26656  JSON-RPC=8545"
echo "  Node 1: RPC=26757  P2P=26756  JSON-RPC=8555"
echo "  Node 2: RPC=26857  P2P=26856  JSON-RPC=8565"
echo "  Node 3: RPC=26957  P2P=26956  JSON-RPC=8575"
echo ""
echo "  Logs: $BASE_DIR/node*.log"
echo "  Stop: $BASE_DIR/stop_all.sh"
echo ""
STARTEOF

cat > "$STOP_SCRIPT" << 'STOPEOF'
#!/bin/bash
BASE_DIR="$HOME/.axon-localnet"

echo "Stopping Axon localnet..."
for i in 0 1 2 3; do
    PID_FILE="$BASE_DIR/node${i}.pid"
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if kill -0 "$PID" 2>/dev/null; then
            kill "$PID"
            echo "  Stopped node $i (PID $PID)"
        fi
        rm -f "$PID_FILE"
    fi
done
echo "All nodes stopped."
STOPEOF

chmod +x "$START_SCRIPT" "$STOP_SCRIPT"

echo ""
echo "============================================"
echo "  Axon 4-Node Localnet Ready!"
echo "============================================"
echo ""
echo "  Base dir:  $BASE_DIR"
echo "  Chain ID:  $CHAIN_ID"
echo "  Nodes:     $NUM_NODES"
echo ""
echo "  Start:     $BASE_DIR/start_all.sh"
echo "  Stop:      $BASE_DIR/stop_all.sh"
echo ""
echo "  Node ports:"
echo "    Node 0: RPC=26657  P2P=26656  JSON-RPC=8545"
echo "    Node 1: RPC=26757  P2P=26756  JSON-RPC=8555"
echo "    Node 2: RPC=26857  P2P=26856  JSON-RPC=8565"
echo "    Node 3: RPC=26957  P2P=26956  JSON-RPC=8575"
echo ""
