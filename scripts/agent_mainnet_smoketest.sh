#!/bin/bash
set -euo pipefail

# Axon Agent mainnet smoke + accounting checks
# - register debit threshold check
# - tx success / event checks
# - heartbeat progression check
# - optional reward growth check over N blocks
#
# Example:
#   bash scripts/agent_mainnet_smoketest.sh \
#     --node https://rpc.axon.network:443 \
#     --chain-id axon_8210-1 \
#     --from mykey \
#     --address axon1... \
#     --run-register \
#     --run-heartbeat \
#     --check-reward-blocks 30

BINARY="${BINARY:-axond}"
NODE=""
CHAIN_ID="axon_8210-1"
FROM=""
ADDRESS=""

CAPABILITIES="ops,monitoring"
MODEL="smoketest-bot"
STAKE="100000000000000000000aaxon"
CHECK_REWARD_BLOCKS=0
FORBID_AGENT_ADDRESS=""

RUN_REGISTER=false
RUN_HEARTBEAT=false

PASS_COUNT=0
WARN_COUNT=0
FAIL_COUNT=0

pass() { echo "PASS: $1"; PASS_COUNT=$((PASS_COUNT + 1)); }
warn() { echo "WARN: $1"; WARN_COUNT=$((WARN_COUNT + 1)); }
fail() { echo "FAIL: $1"; FAIL_COUNT=$((FAIL_COUNT + 1)); }
info() { echo "INFO: $1"; }

usage() {
  cat <<EOF
Usage:
  bash scripts/agent_mainnet_smoketest.sh --node <rpc> --from <key> --address <bech32> [options]

Required:
  --node <rpc>                 RPC endpoint, e.g. https://rpc.axon.network:443
  --from <key>                 local key name used by axond
  --address <bech32>           account bech32 address of --from

Optional:
  --binary <path>              axond binary path (default: axond)
  --chain-id <id>              chain id (default: axon_8210-1)
  --capabilities <v>           register capabilities (default: ops,monitoring)
  --model <v>                  register model (default: smoketest-bot)
  --stake <coin>               register stake coin (default: 100000000000000000000aaxon)
  --run-register               execute register tx
  --run-heartbeat              execute heartbeat tx
  --check-reward-blocks <N>    wait N blocks and check aaxon balance growth
  --forbid-agent-address <a>   optional: assert this address is NOT registered as agent
  --help                       show this help
EOF
}

require_python() {
  if ! command -v python3 >/dev/null 2>&1; then
    echo "python3 is required"
    exit 1
  fi
}

extract_aaxon_balance() {
  local json="$1"
  python3 - "$json" <<'PY'
import json, sys
data = json.loads(sys.argv[1])
amt = 0
for c in data.get("balances", []):
    if c.get("denom") == "aaxon":
        try:
            amt = int(c.get("amount", "0"))
        except ValueError:
            amt = 0
        break
print(amt)
PY
}

extract_stake_amount() {
  local coin="$1"
  python3 - "$coin" <<'PY'
import re, sys
m = re.match(r'^([0-9]+)aaxon$', sys.argv[1])
print(m.group(1) if m else "0")
PY
}

query_balance_json() {
  "$BINARY" query bank balances "$1" --node "$NODE" --output json
}

query_latest_height() {
  "$BINARY" status --node "$NODE" | python3 -c 'import json,sys; print(int(json.load(sys.stdin)["SyncInfo"]["latest_block_height"]))'
}

query_agent_json() {
  "$BINARY" query agent agent "$1" --node "$NODE" --output json
}

extract_agent_last_heartbeat() {
  local json="$1"
  python3 - "$json" <<'PY'
import json, sys
data = json.loads(sys.argv[1])
agent = data.get("agent", {})
v = agent.get("last_heartbeat", "0")
try:
    print(int(v))
except Exception:
    print(0)
PY
}

extract_txhash() {
  local json="$1"
  python3 - "$json" <<'PY'
import json, sys
data = json.loads(sys.argv[1])
print(data.get("txhash",""))
PY
}

extract_tx_code() {
  local json="$1"
  python3 - "$json" <<'PY'
import json, sys
data = json.loads(sys.argv[1])
print(int(data.get("code", 0)))
PY
}

has_register_event() {
  local tx_json="$1"
  python3 - "$tx_json" <<'PY'
import json, sys
data = json.loads(sys.argv[1])
logs = data.get("tx_response", {}).get("logs", [])
for lg in logs:
    for ev in lg.get("events", []):
        if ev.get("type") == "agent_registered":
            print("1")
            raise SystemExit(0)
print("0")
PY
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --binary) BINARY="$2"; shift 2 ;;
    --node) NODE="$2"; shift 2 ;;
    --chain-id) CHAIN_ID="$2"; shift 2 ;;
    --from) FROM="$2"; shift 2 ;;
    --address) ADDRESS="$2"; shift 2 ;;
    --capabilities) CAPABILITIES="$2"; shift 2 ;;
    --model) MODEL="$2"; shift 2 ;;
    --stake) STAKE="$2"; shift 2 ;;
    --check-reward-blocks) CHECK_REWARD_BLOCKS="$2"; shift 2 ;;
    --forbid-agent-address) FORBID_AGENT_ADDRESS="$2"; shift 2 ;;
    --run-register) RUN_REGISTER=true; shift 1 ;;
    --run-heartbeat) RUN_HEARTBEAT=true; shift 1 ;;
    --help|-h) usage; exit 0 ;;
    *)
      echo "Unknown arg: $1"
      usage
      exit 1
      ;;
  esac
done

if [[ -z "$NODE" || -z "$FROM" || -z "$ADDRESS" ]]; then
  usage
  exit 1
fi

if ! command -v "$BINARY" >/dev/null 2>&1 && [[ ! -x "$BINARY" ]]; then
  echo "Binary not found: $BINARY"
  exit 1
fi

require_python

echo "=== Axon Agent Mainnet Smoke Test ==="
echo "Binary:              $BINARY"
echo "Node:                $NODE"
echo "Chain ID:            $CHAIN_ID"
echo "From:                $FROM"
echo "Address:             $ADDRESS"
echo "Stake:               $STAKE"
echo "Register TX:         $RUN_REGISTER"
echo "Heartbeat TX:        $RUN_HEARTBEAT"
echo "Reward Check Blocks: $CHECK_REWARD_BLOCKS"
echo

if "$BINARY" query agent params --node "$NODE" >/dev/null 2>&1; then
  pass "agent params query ok"
else
  fail "agent params query failed"
fi

BAL_BEFORE_JSON="$(query_balance_json "$ADDRESS" || true)"
if [[ -n "$BAL_BEFORE_JSON" ]]; then
  BAL_BEFORE_AAXON="$(extract_aaxon_balance "$BAL_BEFORE_JSON")"
  pass "queried balance before: $BAL_BEFORE_AAXON aaxon"
else
  BAL_BEFORE_AAXON=0
  fail "failed to query balance before"
fi

AGENT_BEFORE_EXISTS=0
AGENT_BEFORE_JSON=""
if AGENT_BEFORE_JSON="$(query_agent_json "$ADDRESS" 2>/dev/null)"; then
  AGENT_BEFORE_EXISTS=1
  pass "agent exists before test"
else
  warn "agent not found before test (ok for first registration)"
fi

HB_BEFORE=0
if [[ $AGENT_BEFORE_EXISTS -eq 1 ]]; then
  HB_BEFORE="$(extract_agent_last_heartbeat "$AGENT_BEFORE_JSON")"
fi

if [[ -n "$FORBID_AGENT_ADDRESS" ]]; then
  if query_agent_json "$FORBID_AGENT_ADDRESS" >/dev/null 2>&1; then
    fail "forbidden address is registered as agent: $FORBID_AGENT_ADDRESS"
  else
    pass "forbidden address is not an agent: $FORBID_AGENT_ADDRESS"
  fi
fi

REGISTER_TXHASH=""
if [[ "$RUN_REGISTER" == true ]]; then
  info "submitting register tx"
  REG_OUT="$("$BINARY" tx agent register "$CAPABILITIES" "$MODEL" "$STAKE" \
    --from "$FROM" --chain-id "$CHAIN_ID" --node "$NODE" --yes --output json 2>/dev/null || true)"
  if [[ -z "$REG_OUT" ]]; then
    fail "register tx returned empty output"
  else
    REG_CODE="$(extract_tx_code "$REG_OUT")"
    REGISTER_TXHASH="$(extract_txhash "$REG_OUT")"
    if [[ "$REG_CODE" -eq 0 ]]; then
      pass "register tx accepted, txhash=$REGISTER_TXHASH"
    else
      fail "register tx failed with code=$REG_CODE, txhash=$REGISTER_TXHASH"
    fi
  fi

  if [[ -n "$REGISTER_TXHASH" ]]; then
    set +e
    TX_JSON="$("$BINARY" query tx "$REGISTER_TXHASH" --node "$NODE" --output json 2>/dev/null)"
    TX_RC=$?
    set -e
    if [[ $TX_RC -eq 0 && -n "$TX_JSON" ]]; then
      EVT="$(has_register_event "$TX_JSON")"
      if [[ "$EVT" == "1" ]]; then
        pass "register event detected in tx logs"
      else
        warn "register event not found in tx logs (check node indexer settings)"
      fi
    else
      warn "could not query register tx by hash (may be indexer delay)"
    fi
  fi
fi

if [[ "$RUN_HEARTBEAT" == true ]]; then
  info "submitting heartbeat tx"
  HB_OUT="$("$BINARY" tx agent heartbeat \
    --from "$FROM" --chain-id "$CHAIN_ID" --node "$NODE" --yes --output json 2>/dev/null || true)"
  if [[ -z "$HB_OUT" ]]; then
    fail "heartbeat tx returned empty output"
  else
    HB_CODE="$(extract_tx_code "$HB_OUT")"
    HB_TXHASH="$(extract_txhash "$HB_OUT")"
    if [[ "$HB_CODE" -eq 0 ]]; then
      pass "heartbeat tx accepted, txhash=$HB_TXHASH"
    else
      fail "heartbeat tx failed with code=$HB_CODE, txhash=$HB_TXHASH"
    fi
  fi
fi

BAL_AFTER_JSON="$(query_balance_json "$ADDRESS" || true)"
if [[ -n "$BAL_AFTER_JSON" ]]; then
  BAL_AFTER_AAXON="$(extract_aaxon_balance "$BAL_AFTER_JSON")"
  pass "queried balance after: $BAL_AFTER_AAXON aaxon"
else
  BAL_AFTER_AAXON=0
  fail "failed to query balance after"
fi

if [[ "$RUN_REGISTER" == true ]]; then
  STAKE_AAXON="$(extract_stake_amount "$STAKE")"
  DEBIT=$((BAL_BEFORE_AAXON - BAL_AFTER_AAXON))
  if [[ "$DEBIT" -ge "$STAKE_AAXON" ]]; then
    pass "register debit check ok: debit=$DEBIT >= stake=$STAKE_AAXON (fees included)"
  else
    fail "register debit too small: debit=$DEBIT < stake=$STAKE_AAXON"
  fi
fi

AGENT_AFTER_JSON=""
if AGENT_AFTER_JSON="$(query_agent_json "$ADDRESS" 2>/dev/null)"; then
  pass "agent query after test ok"
else
  fail "agent not found after test"
fi

if [[ "$RUN_HEARTBEAT" == true && -n "$AGENT_AFTER_JSON" ]]; then
  HB_AFTER="$(extract_agent_last_heartbeat "$AGENT_AFTER_JSON")"
  if [[ "$HB_AFTER" -ge "$HB_BEFORE" ]]; then
    pass "last_heartbeat progressed or unchanged as expected (before=$HB_BEFORE, after=$HB_AFTER)"
  else
    fail "last_heartbeat decreased unexpectedly (before=$HB_BEFORE, after=$HB_AFTER)"
  fi
fi

if [[ "$CHECK_REWARD_BLOCKS" -gt 0 ]]; then
  START_H="$(query_latest_height)"
  START_BAL="$BAL_AFTER_AAXON"
  TARGET_H=$((START_H + CHECK_REWARD_BLOCKS))
  info "waiting for reward window: current=$START_H target=$TARGET_H"
  while true; do
    CUR_H="$(query_latest_height)"
    if [[ "$CUR_H" -ge "$TARGET_H" ]]; then
      break
    fi
    sleep 5
  done
  END_BAL_JSON="$(query_balance_json "$ADDRESS" || true)"
  END_BAL="$(extract_aaxon_balance "$END_BAL_JSON")"
  if [[ "$END_BAL" -gt "$START_BAL" ]]; then
    pass "reward growth observed over $CHECK_REWARD_BLOCKS blocks: +$((END_BAL - START_BAL)) aaxon"
  else
    warn "no balance growth over $CHECK_REWARD_BLOCKS blocks; reward may be pooled/distributed later"
  fi
fi

echo
echo "=== Smoke Test Result ==="
echo "PASS: $PASS_COUNT"
echo "WARN: $WARN_COUNT"
echo "FAIL: $FAIL_COUNT"

if [[ $FAIL_COUNT -gt 0 ]]; then
  echo "Result: FAILED"
  exit 1
fi

if [[ $WARN_COUNT -gt 0 ]]; then
  echo "Result: PASS WITH WARNINGS"
else
  echo "Result: PASS"
fi
