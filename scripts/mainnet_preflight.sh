#!/bin/bash
set -euo pipefail

# Axon mainnet preflight checker.
# Usage:
#   bash scripts/mainnet_preflight.sh --home ~/.axon-mainnet --binary ./build/axond

BINARY="${BINARY:-axond}"
HOME_DIR="${HOME_DIR:-$HOME/.axon-mainnet}"
GENESIS_PATH=""
EXPECTED_CHAIN_ID="${EXPECTED_CHAIN_ID:-axon_8210-1}"
EXPECTED_VERSION="${EXPECTED_VERSION:-v1.0.0}"
EXPECTED_MIN_GAS_PRICES="${EXPECTED_MIN_GAS_PRICES:-10000000000aaxon}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --home) HOME_DIR="$2"; shift 2 ;;
    --binary) BINARY="$2"; shift 2 ;;
    --genesis) GENESIS_PATH="$2"; shift 2 ;;
    --expected-chain-id) EXPECTED_CHAIN_ID="$2"; shift 2 ;;
    --expected-version) EXPECTED_VERSION="$2"; shift 2 ;;
    --expected-min-gas-prices) EXPECTED_MIN_GAS_PRICES="$2"; shift 2 ;;
    *)
      echo "Unknown argument: $1"
      exit 1
      ;;
  esac
done

if [[ -z "$GENESIS_PATH" ]]; then
  GENESIS_PATH="$HOME_DIR/config/genesis.json"
fi

PASS_COUNT=0
WARN_COUNT=0
FAIL_COUNT=0

pass() {
  echo "PASS: $1"
  PASS_COUNT=$((PASS_COUNT + 1))
}

warn() {
  echo "WARN: $1"
  WARN_COUNT=$((WARN_COUNT + 1))
}

fail() {
  echo "FAIL: $1"
  FAIL_COUNT=$((FAIL_COUNT + 1))
}

sha256_file() {
  local file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
  else
    echo ""
    return 1
  fi
}

echo "=== Axon Mainnet Preflight ==="
echo "Binary:   $BINARY"
echo "Home:     $HOME_DIR"
echo "Genesis:  $GENESIS_PATH"
echo "Chain ID: $EXPECTED_CHAIN_ID"
echo "Min Gas:  $EXPECTED_MIN_GAS_PRICES"
echo ""

if [[ -x "$BINARY" ]] || command -v "$BINARY" >/dev/null 2>&1; then
  pass "Binary is available"
else
  fail "Binary not found or not executable: $BINARY"
fi

if [[ -f "$GENESIS_PATH" ]]; then
  pass "Genesis file exists"
else
  fail "Genesis file not found: $GENESIS_PATH"
fi

if [[ -f "$GENESIS_PATH" ]]; then
  if "$BINARY" genesis validate "$GENESIS_PATH" >/dev/null 2>&1; then
    pass "Genesis validation passed"
  else
    fail "Genesis validation failed"
  fi
fi

if [[ -f "$GENESIS_PATH" ]]; then
  GENESIS_CHAIN_ID=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1])).get("chain_id",""))' "$GENESIS_PATH")
  if [[ "$GENESIS_CHAIN_ID" == "$EXPECTED_CHAIN_ID" ]]; then
    pass "Genesis chain_id matches expected ($EXPECTED_CHAIN_ID)"
  else
    fail "Genesis chain_id mismatch: got '$GENESIS_CHAIN_ID', expected '$EXPECTED_CHAIN_ID'"
  fi
fi

if [[ -f "$GENESIS_PATH" ]]; then
  GENESIS_SHA256=$(sha256_file "$GENESIS_PATH" || true)
  if [[ -n "$GENESIS_SHA256" ]]; then
    pass "Genesis SHA256 computed: $GENESIS_SHA256"
  else
    warn "Cannot compute SHA256 (sha256sum/shasum not found)"
  fi
fi

VERSION_OUTPUT=$("$BINARY" version --long 2>/dev/null || "$BINARY" version 2>/dev/null || true)
if [[ "$VERSION_OUTPUT" == *"$EXPECTED_VERSION"* ]]; then
  pass "Binary version contains expected string ($EXPECTED_VERSION)"
else
  warn "Binary version does not contain expected string ($EXPECTED_VERSION)"
fi

APP_TOML="$HOME_DIR/config/app.toml"
if [[ -f "$APP_TOML" ]]; then
  MIN_GAS_LINE=$(python3 -c '
import re,sys
p=sys.argv[1]
for line in open(p, "r", encoding="utf-8", errors="ignore"):
    m=re.match(r"\s*minimum-gas-prices\s*=\s*\"([^\"]*)\"", line)
    if m:
        print(m.group(1))
        break
' "$APP_TOML")
  if [[ -n "$MIN_GAS_LINE" ]]; then
    if [[ "$MIN_GAS_LINE" == "$EXPECTED_MIN_GAS_PRICES" ]]; then
      pass "app.toml minimum-gas-prices matches expected ($MIN_GAS_LINE)"
    else
      fail "app.toml minimum-gas-prices mismatch: got '$MIN_GAS_LINE', expected '$EXPECTED_MIN_GAS_PRICES'"
    fi
  else
    fail "app.toml minimum-gas-prices is empty"
  fi
else
  fail "app.toml not found: $APP_TOML"
fi

CONFIG_TOML="$HOME_DIR/config/config.toml"
if [[ -f "$CONFIG_TOML" ]]; then
  P2P_LINES=$(python3 -c '
import re,sys
cfg={"seeds":"","persistent_peers":""}
for line in open(sys.argv[1], "r", encoding="utf-8", errors="ignore"):
    for k in ("seeds","persistent_peers"):
        m=re.match(rf"\s*{k}\s*=\s*\"([^\"]*)\"", line)
        if m:
            cfg[k]=m.group(1)
print(cfg["seeds"])
print(cfg["persistent_peers"])
' "$CONFIG_TOML")
  SEEDS=$(printf "%s\n" "$P2P_LINES" | awk 'NR==1 {print}')
  PEERS=$(printf "%s\n" "$P2P_LINES" | awk 'NR==2 {print}')
  if [[ -n "$SEEDS" || -n "$PEERS" ]]; then
    pass "P2P bootstrap is configured (seeds/persistent_peers)"
  else
    warn "Both seeds and persistent_peers are empty"
  fi
else
  warn "config.toml not found: $CONFIG_TOML"
fi

PV_KEY="$HOME_DIR/config/priv_validator_key.json"
if [[ -f "$PV_KEY" ]]; then
  PERM=""
  if stat -f "%Lp" "$PV_KEY" >/dev/null 2>&1; then
    PERM=$(stat -f "%Lp" "$PV_KEY")
  elif stat -c "%a" "$PV_KEY" >/dev/null 2>&1; then
    PERM=$(stat -c "%a" "$PV_KEY")
  fi
  if [[ -n "$PERM" ]]; then
    if [[ "$PERM" == "600" || "$PERM" == "400" ]]; then
      pass "priv_validator_key.json permissions look strict ($PERM)"
    else
      warn "priv_validator_key.json permissions are broader than recommended ($PERM)"
    fi
  else
    warn "Cannot read permissions for priv_validator_key.json"
  fi
else
  warn "priv_validator_key.json not found (expected for validator nodes)"
fi

AVAIL_KB=$(df -Pk "$HOME_DIR" 2>/dev/null | awk 'NR==2 {print $4}')
if [[ -n "${AVAIL_KB:-}" ]]; then
  if [[ "$AVAIL_KB" -ge 20971520 ]]; then
    pass "Free disk space >= 20 GiB"
  else
    warn "Free disk space < 20 GiB (current: ${AVAIL_KB} KiB)"
  fi
else
  warn "Cannot determine free disk space"
fi

echo ""
echo "=== Preflight Result ==="
echo "PASS: $PASS_COUNT"
echo "WARN: $WARN_COUNT"
echo "FAIL: $FAIL_COUNT"

if [[ $FAIL_COUNT -gt 0 ]]; then
  echo "Result: BLOCKED (fix FAIL items before mainnet launch)"
  exit 1
fi

if [[ $WARN_COUNT -gt 0 ]]; then
  echo "Result: READY WITH WARNINGS (review WARN items)"
else
  echo "Result: READY"
fi

