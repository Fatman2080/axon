#!/bin/bash
set -euo pipefail

# ============================================================
# Axon Mainnet Genesis Initializer
#
# Generates a production-ready genesis.json for the Axon mainnet.
# Usage: bash scripts/init_mainnet.sh [--home DIR] [--chain-id ID]
# ============================================================

BINARY="${BINARY:-axond}"
HOME_DIR="${HOME_DIR:-$HOME/.axon-mainnet}"
CHAIN_ID="${CHAIN_ID:-axon_9001-1}"
DENOM="aaxon"
MIN_VALIDATOR_STAKE="10000000000000000000000"  # 10,000 AXON
MIN_AGENT_STAKE="100"
BLOCK_TIME="5s"

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --home) HOME_DIR="$2"; shift 2 ;;
    --chain-id) CHAIN_ID="$2"; shift 2 ;;
    *) echo "Unknown: $1"; exit 1 ;;
  esac
done

echo "=== Axon Mainnet Genesis Initializer ==="
echo "Home:     $HOME_DIR"
echo "Chain ID: $CHAIN_ID"
echo ""

# Initialize
$BINARY init "axon-mainnet" --chain-id "$CHAIN_ID" --home "$HOME_DIR" 2>/dev/null

GENESIS="$HOME_DIR/config/genesis.json"

# ── Patch genesis with production parameters ──

# Use Python for JSON manipulation
python3 - "$GENESIS" << 'PYTHON_SCRIPT'
import json, sys

genesis_path = sys.argv[1]
with open(genesis_path) as f:
    genesis = json.load(f)

# ── Chain parameters ──
genesis["chain_id"] = "axon_9001-1"

# ── Consensus parameters ──
genesis["consensus"]["params"]["block"]["max_gas"] = "40000000"
genesis["consensus"]["params"]["block"]["max_bytes"] = "2097152"  # 2MB
genesis["consensus"]["params"]["block"]["time_iota_ms"] = "1000"

# ── Staking ──
staking = genesis["app_state"]["staking"]["params"]
staking["bond_denom"] = "aaxon"
staking["unbonding_time"] = "1209600s"      # 14 days
staking["max_validators"] = 100
staking["min_commission_rate"] = "0.050000000000000000"  # 5% minimum

# ── Slashing ──
slashing = genesis["app_state"]["slashing"]["params"]
slashing["signed_blocks_window"] = "10000"
slashing["min_signed_per_window"] = "0.050000000000000000"
slashing["downtime_jail_duration"] = "600s"
slashing["slash_fraction_double_sign"] = "0.050000000000000000"  # 5%
slashing["slash_fraction_downtime"] = "0.001000000000000000"     # 0.1%

# ── Governance ──
gov = genesis["app_state"]["gov"]["params"]
gov["min_deposit"] = [{"denom": "aaxon", "amount": "10000000000000000000000"}]  # 10,000 AXON
gov["max_deposit_period"] = "172800s"    # 2 days
gov["voting_period"] = "604800s"         # 7 days
gov["quorum"] = "0.334000000000000000"   # 33.4%
gov["threshold"] = "0.500000000000000000"  # 50%
gov["veto_threshold"] = "0.334000000000000000"  # 33.4%

# ── Mint: disabled (Axon uses custom agent module minting) ──
if "mint" in genesis["app_state"]:
    mint = genesis["app_state"]["mint"]
    if "params" in mint:
        mint["params"]["mint_denom"] = "aaxon"
        mint["params"]["inflation_rate_change"] = "0.000000000000000000"
        mint["params"]["inflation_max"] = "0.000000000000000000"
        mint["params"]["inflation_min"] = "0.000000000000000000"

# ── Bank ──
bank = genesis["app_state"]["bank"]["params"]
bank["default_send_enabled"] = True

# ── Distribution ──
dist = genesis["app_state"]["distribution"]["params"]
dist["community_tax"] = "0.000000000000000000"  # 0% community tax (burns handle deflation)
dist["base_proposer_reward"] = "0.000000000000000000"
dist["bonus_proposer_reward"] = "0.000000000000000000"

# ── Agent module ──
if "agent" in genesis["app_state"]:
    agent = genesis["app_state"]["agent"]
    if "params" in agent:
        params = agent["params"]
        params["min_register_stake"] = 100
        params["register_burn_amount"] = 20
        params["max_reputation"] = 100
        params["epoch_length"] = 720
        params["heartbeat_timeout"] = 720
        params["ai_challenge_window"] = 50
        params["deregister_cooldown"] = 120960  # ~7 days

# ── Fee Market (EIP-1559) ──
if "feemarket" in genesis["app_state"]:
    fm = genesis["app_state"]["feemarket"]
    if "params" in fm:
        fm["params"]["no_base_fee"] = False
        fm["params"]["base_fee"] = "1000000000"  # 1 gwei initial base fee

# ── EVM ──
if "evm" in genesis["app_state"]:
    evm = genesis["app_state"]["evm"]
    if "params" in evm:
        evm["params"]["evm_denom"] = "aaxon"

# ── Zero supply (all tokens come from mining) ──
genesis["app_state"]["bank"]["supply"] = []

with open(genesis_path, 'w') as f:
    json.dump(genesis, f, indent=2)

print("Genesis patched with mainnet parameters.")
PYTHON_SCRIPT

echo ""
echo "=== Mainnet Genesis Configuration ==="
echo ""
echo "Key Parameters:"
echo "  Chain ID:              axon_9001-1"
echo "  Block Gas Limit:       40,000,000"
echo "  Block Time:            ~5 seconds"
echo "  Validator Unbonding:   14 days"
echo "  Max Validators:        100"
echo "  Min Commission:        5%"
echo "  Slashing (double sign): 5%"
echo "  Slashing (downtime):   0.1%"
echo "  Gov Min Deposit:       10,000 AXON"
echo "  Gov Voting Period:     7 days"
echo "  Gov Quorum:            33.4%"
echo "  Agent Stake:           100 AXON (20 burned)"
echo "  Epoch Length:           720 blocks (~1 hour)"
echo "  Heartbeat Timeout:     720 blocks (~1 hour)"
echo "  EIP-1559:              Enabled"
echo "  Initial Supply:        0 (100% mined)"
echo ""
echo "Genesis file: $GENESIS"
echo ""
echo "Next steps:"
echo "  1. Add initial validators: $BINARY genesis add-genesis-account ..."
echo "  2. Collect gentxs:         $BINARY genesis collect-gentxs --home $HOME_DIR"
echo "  3. Validate genesis:       $BINARY genesis validate --home $HOME_DIR"
echo "  4. Share genesis.json with all validators"
echo "  5. Start the network:      $BINARY start --home $HOME_DIR"
