#!/bin/bash
set -e

# ============================================================
# Axon Public Testnet Initializer
# Generates genesis, keys, and configs for N validator nodes.
# Designed to run inside a Docker init container or on the
# first boot of each cloud VM.
# ============================================================

CHAIN_ID="${CHAIN_ID:-axon_9001-1}"
DENOM="aaxon"
KEYRING="test"
NUM_VALIDATORS="${NUM_VALIDATORS:-4}"
BASE_DIR="${BASE_DIR:-/data/axon-testnet}"
BINARY="${BINARY:-axond}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

BALANCE="1000000000000000000000000000${DENOM}"  # 1B AXON
STAKE="10000000000000000000000000${DENOM}"       # 10M AXON
FAUCET_BALANCE="100000000000000000000000000${DENOM}" # 100M AXON

echo "============================================"
echo "  Axon Testnet Initializer"
echo "  Chain ID:    $CHAIN_ID"
echo "  Validators:  $NUM_VALIDATORS"
echo "  Base dir:    $BASE_DIR"
echo "============================================"

rm -rf "$BASE_DIR"
mkdir -p "$BASE_DIR"

# ── 1. Initialize all nodes ─────────────────────────────────
echo ""
echo "==> Initializing $NUM_VALIDATORS validator nodes..."

for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    NODE_DIR="$BASE_DIR/node$i"
    $BINARY init "axon-validator-$i" --chain-id "$CHAIN_ID" --home "$NODE_DIR" 2>/dev/null
    $BINARY keys add "validator$i" --keyring-backend "$KEYRING" --home "$NODE_DIR" 2>/dev/null
    echo "    node$i initialized"
done

# Create faucet key on node0
$BINARY keys add faucet --keyring-backend "$KEYRING" --home "$BASE_DIR/node0" 2>/dev/null
FAUCET_ADDR=$($BINARY keys show faucet -a --keyring-backend "$KEYRING" --home "$BASE_DIR/node0")
echo "    Faucet address: $FAUCET_ADDR"
echo "$FAUCET_ADDR" > "$BASE_DIR/faucet_address.txt"

# Export faucet private key for the faucet API
$BINARY keys unsafe-export-eth-key faucet --keyring-backend "$KEYRING" --home "$BASE_DIR/node0" \
    > "$BASE_DIR/faucet_private_key.txt" 2>/dev/null || true

# ── 2. Patch genesis ────────────────────────────────────────
echo ""
echo "==> Patching genesis.json..."
GENESIS="$BASE_DIR/node0/config/genesis.json"
python3 "$SCRIPT_DIR/genesis-patch.py" "$GENESIS" "$DENOM"

# ── 3. Genesis accounts ─────────────────────────────────────
echo ""
echo "==> Adding genesis accounts..."

for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    NODE_DIR="$BASE_DIR/node$i"
    ADDR=$($BINARY keys show "validator$i" -a --keyring-backend "$KEYRING" --home "$NODE_DIR")
    $BINARY genesis add-genesis-account "$ADDR" "$BALANCE" \
        --keyring-backend "$KEYRING" --home "$BASE_DIR/node0"
    echo "    validator$i: $ADDR"
done

# Faucet account
$BINARY genesis add-genesis-account "$FAUCET_ADDR" "$FAUCET_BALANCE" \
    --keyring-backend "$KEYRING" --home "$BASE_DIR/node0"
echo "    faucet: $FAUCET_ADDR"

# ── 4. Genesis transactions ─────────────────────────────────
echo ""
echo "==> Creating genesis transactions..."

for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    NODE_DIR="$BASE_DIR/node$i"
    cp "$BASE_DIR/node0/config/genesis.json" "$NODE_DIR/config/genesis.json"

    $BINARY genesis gentx "validator$i" "$STAKE" \
        --chain-id "$CHAIN_ID" \
        --moniker "axon-validator-$i" \
        --keyring-backend "$KEYRING" \
        --home "$NODE_DIR" 2>/dev/null

    cp "$NODE_DIR/config/gentx/"*.json "$BASE_DIR/node0/config/gentx/" 2>/dev/null || true
    echo "    gentx created for node$i"
done

# ── 5. Collect & distribute ─────────────────────────────────
echo ""
echo "==> Collecting genesis transactions..."
$BINARY genesis collect-gentxs --home "$BASE_DIR/node0" 2>/dev/null
$BINARY genesis validate-genesis --home "$BASE_DIR/node0"

for i in $(seq 1 $((NUM_VALIDATORS - 1))); do
    cp "$BASE_DIR/node0/config/genesis.json" "$BASE_DIR/node$i/config/genesis.json"
done

# ── 6. Peer configuration ───────────────────────────────────
echo ""
echo "==> Configuring peers..."

NODE_IDS=()
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    NODE_DIR="$BASE_DIR/node$i"
    NID=$($BINARY comet show-node-id --home "$NODE_DIR" 2>/dev/null \
          || $BINARY tendermint show-node-id --home "$NODE_DIR" 2>/dev/null)
    NODE_IDS+=("$NID")
done

# Build seeds string (all nodes know about each other via Docker service names)
SEEDS=""
for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    [ -n "$SEEDS" ] && SEEDS="${SEEDS},"
    SEEDS="${SEEDS}${NODE_IDS[$i]}@axon-node-${i}:26656"
done

echo "    Seeds: $SEEDS"
echo "$SEEDS" > "$BASE_DIR/seeds.txt"

# ── 7. Configure each node ──────────────────────────────────
echo ""
echo "==> Configuring nodes..."

for i in $(seq 0 $((NUM_VALIDATORS - 1))); do
    NODE_DIR="$BASE_DIR/node$i"
    CONFIG="$NODE_DIR/config/config.toml"
    APP_TOML="$NODE_DIR/config/app.toml"

    SELF_SEED="${NODE_IDS[$i]}@axon-node-${i}:26656"
    PEER_LIST=$(echo "$SEEDS" | sed "s|${SELF_SEED}||g" | sed 's|,,|,|g;s|^,||;s|,$||')

    if [[ "$OSTYPE" == "darwin"* ]]; then
        SED="sed -i ''"
    else
        SED="sed -i"
    fi

    # P2P: listen on all interfaces
    eval $SED "'s|laddr = \"tcp://0.0.0.0:26656\"|laddr = \"tcp://0.0.0.0:26656\"|'" "$CONFIG"
    eval $SED "'s|laddr = \"tcp://127.0.0.1:26657\"|laddr = \"tcp://0.0.0.0:26657\"|'" "$CONFIG"

    # Peers
    eval $SED "'s|persistent_peers = \"\"|persistent_peers = \"${PEER_LIST}\"|'" "$CONFIG"
    eval $SED "'s|addr_book_strict = true|addr_book_strict = false|'" "$CONFIG"
    eval $SED "'s|allow_duplicate_ip = false|allow_duplicate_ip = true|'" "$CONFIG"

    # Prometheus metrics
    eval $SED "'s|prometheus = false|prometheus = true|'" "$CONFIG"

    # API: listen on all interfaces
    eval $SED "'s|address = \"tcp://localhost:1317\"|address = \"tcp://0.0.0.0:1317\"|'" "$APP_TOML"
    eval $SED "'s|address = \"localhost:9090\"|address = \"0.0.0.0:9090\"|'" "$APP_TOML"
    eval $SED "'s|minimum-gas-prices = \"\"|minimum-gas-prices = \"0${DENOM}\"|'" "$APP_TOML"
    eval $SED "'s|enable = false|enable = true|g'" "$APP_TOML"

    echo "    node$i configured"
done

# ── 8. Write node info ──────────────────────────────────────
echo ""
echo "==> Writing node info..."

cat > "$BASE_DIR/network-info.json" << JSONEOF
{
  "chain_id": "$CHAIN_ID",
  "num_validators": $NUM_VALIDATORS,
  "denom": "$DENOM",
  "display_denom": "AXON",
  "seeds": "$SEEDS",
  "faucet_address": "$FAUCET_ADDR",
  "precompiles": {
    "IAgentRegistry": "0x0000000000000000000000000000000000000801",
    "IAgentReputation": "0x0000000000000000000000000000000000000802",
    "IAgentWallet": "0x0000000000000000000000000000000000000803"
  }
}
JSONEOF

echo ""
echo "============================================"
echo "  Axon Testnet Initialized!"
echo "  Chain ID:     $CHAIN_ID"
echo "  Validators:   $NUM_VALIDATORS"
echo "  Faucet:       $FAUCET_ADDR"
echo "  Data:         $BASE_DIR"
echo "============================================"
