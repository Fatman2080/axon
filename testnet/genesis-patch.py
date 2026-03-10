#!/usr/bin/env python3
"""Patch genesis.json for Axon public testnet."""
import json, sys

genesis_path = sys.argv[1]
denom = sys.argv[2] if len(sys.argv) > 2 else "aaxon"

with open(genesis_path) as f:
    genesis = json.load(f)

app = genesis["app_state"]

# Staking
app["staking"]["params"]["bond_denom"] = denom
app["staking"]["params"]["max_validators"] = 100
app["staking"]["params"]["unbonding_time"] = "1209600s"  # 14 days

# Mint — disabled (Axon uses custom block rewards)
app["mint"]["params"]["mint_denom"] = denom
app["mint"]["params"]["inflation_max"] = "0.000000000000000000"
app["mint"]["params"]["inflation_min"] = "0.000000000000000000"
app["mint"]["params"]["inflation_rate_change"] = "0.000000000000000000"
app["mint"]["minter"]["inflation"] = "0.000000000000000000"

# EVM
app["evm"]["params"]["evm_denom"] = denom
if "extended_denom_options" in app["evm"]["params"]:
    app["evm"]["params"]["extended_denom_options"]["extended_denom"] = denom

# Axon precompiles
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

# Fee market — low fees for testnet
app["feemarket"]["params"]["no_base_fee"] = True
app["feemarket"]["params"]["min_gas_price"] = "0.000000000000000000"

# Bank denom metadata
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

# Gov — use aaxon
if "params" in app.get("gov", {}):
    for dep in app["gov"]["params"].get("min_deposit", []):
        dep["denom"] = denom

# Agent params
if "agent" in app:
    app["agent"]["params"]["min_stake"] = "100000000000000000000"  # 100 AXON
    app["agent"]["params"]["heartbeat_interval"] = "720"
    app["agent"]["params"]["epoch_length"] = "720"

# Block params
genesis["consensus"]["params"]["block"]["max_gas"] = "40000000"
genesis["consensus"]["params"]["block"]["time_iota_ms"] = "1000"

with open(genesis_path, "w") as f:
    json.dump(genesis, f, indent=2)

print("Genesis patched for Axon testnet")
