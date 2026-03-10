#!/bin/bash
set -e

BINARY="./build/axond"
CHAIN_ID="axon_9001-1"
MONIKER="axon-local"
HOME_DIR="$HOME/.axond"
DENOM="aaxon"
KEY_NAME="validator"
KEYRING="test"
GENESIS="$HOME_DIR/config/genesis.json"

if [ ! -f "$BINARY" ]; then
    echo "Building axond..."
    go build -o build/axond ./cmd/axond/
fi

rm -rf "$HOME_DIR"

echo "==> Initializing chain..."
$BINARY init $MONIKER --chain-id $CHAIN_ID --home "$HOME_DIR" 2>/dev/null

echo "==> Creating validator key..."
$BINARY keys add $KEY_NAME --keyring-backend $KEYRING --home "$HOME_DIR" 2>/dev/null
VALIDATOR_ADDR=$($BINARY keys show $KEY_NAME -a --keyring-backend $KEYRING --home "$HOME_DIR")
echo "    Validator address: $VALIDATOR_ADDR"

echo "==> Patching genesis.json..."
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

# Activate Axon custom precompiles alongside defaults
axon_precompiles = [
    "0x0000000000000000000000000000000000000801",  # IAgentRegistry
    "0x0000000000000000000000000000000000000802",  # IAgentReputation
    "0x0000000000000000000000000000000000000803",  # IAgentWallet
]
existing = app["evm"]["params"].get("active_static_precompiles", [])
for pc in axon_precompiles:
    if pc not in existing:
        existing.append(pc)
app["evm"]["params"]["active_static_precompiles"] = existing

app["feemarket"]["params"]["no_base_fee"] = True
app["feemarket"]["params"]["min_gas_price"] = "0.000000000000000000"

# Bank: register aaxon denom metadata (required by EVM)
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

echo "==> Adding genesis account (1B AXON = 1e27 aaxon)..."
$BINARY genesis add-genesis-account "$VALIDATOR_ADDR" "1000000000000000000000000000${DENOM}" \
    --keyring-backend $KEYRING --home "$HOME_DIR"

echo "==> Creating genesis transaction (10M AXON stake)..."
$BINARY genesis gentx $KEY_NAME "10000000000000000000000000${DENOM}" \
    --chain-id $CHAIN_ID \
    --moniker $MONIKER \
    --keyring-backend $KEYRING \
    --home "$HOME_DIR"

echo "==> Collecting genesis transactions..."
$BINARY genesis collect-gentxs --home "$HOME_DIR" 2>/dev/null

echo "==> Validating genesis..."
$BINARY genesis validate-genesis --home "$HOME_DIR"

echo "==> Configuring node..."
if [[ "$OSTYPE" == "darwin"* ]]; then
    SED_CMD="sed -i ''"
else
    SED_CMD="sed -i"
fi

$SED_CMD 's/minimum-gas-prices = ""/minimum-gas-prices = "0'$DENOM'"/' "$HOME_DIR/config/app.toml"

echo ""
echo "============================================"
echo "  Axon local testnet ready!"
echo "  Chain ID:    $CHAIN_ID"
echo "  Validator:   $VALIDATOR_ADDR"
echo "  Home:        $HOME_DIR"
echo "============================================"
echo ""
echo "Start with:"
echo "  $BINARY start --home $HOME_DIR --chain-id $CHAIN_ID --json-rpc.enable"
echo ""
echo "JSON-RPC: http://localhost:8545"
echo "CometBFT: http://localhost:26657"
