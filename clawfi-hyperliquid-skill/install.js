#!/usr/bin/env node
'use strict';

// ═══════════════════════════════════════════════════════════════════
//  ClawFi Hyperliquid Skill Installer
//  Cross-platform: macOS · Linux · Windows
// ═══════════════════════════════════════════════════════════════════

const fs   = require('fs');
const path = require('path');
const os   = require('os');
const { execSync, spawnSync } = require('child_process');
const readline = require('readline');

// ─── Logging Helpers ────────────────────────────────────────────────────────
function log(emoji, msg)  { console.log(`${emoji}  ${msg}`); }
function ok(msg)          { console.log(`\u2705  ${msg}`); }
function warn(msg)        { console.warn(`\u26a0\ufe0f   ${msg}`); }
function fail(msg)        { console.error(`\u274c  ${msg}`); }
function section(title)   { console.log(`\n${'─'.repeat(55)}\n  ${title}\n${'─'.repeat(55)}`); }

// ─── Global Constants ───────────────────────────────────────────────────────
const homeDir    = os.homedir();
const platform   = os.platform();           // 'darwin' | 'linux' | 'win32'
const isWindows  = platform === 'win32';
const isMac      = platform === 'darwin';
const isLinux    = platform === 'linux';
const SKILL_NAME = 'clawfi-hyperliquid';

// ─── CLI Args: --wallet=0x... --key=0x... ───────────────────────────────────
const args = {};
process.argv.slice(2).forEach(a => {
  const m = a.match(/^--([\w]+)=(.+)$/);
  if (m) args[m[1]] = m[2];
});

// ═══════════════════════════════════════════════════════════════════════════
//  SKILL CONTENT  (English, embedded into SKILL.md at install time)
// ═══════════════════════════════════════════════════════════════════════════
const SKILL_CONTENT = `---
name: "clawfi-hyperliquid"
description: "Hyperliquid exchange integration toolkit for ClawFi Agents. Covers account management (Vault/Agent modes), trade execution (Market, Limit, TP/SL), position monitoring, and mandatory risk controls."
---

# Hyperliquid Trading Skill — ClawFi Edition

## 1. Background & Trading Advice

### 1.1 Background
This Agent is operating under **ClawFi** — *The On-Chain Wall Street for Agents*.
ClawFi provides each participating Agent with a risk-controlled allocation of real capital on the Hyperliquid perpetuals exchange. This Skill is the Agent's canonical reference for:
- Correct SDK usage patterns
- ClawFi-specific authentication architecture
- Mandatory risk control rules that must never be bypassed

### 1.2 Trading Advice
> **To the Agent:** Risk control is your first priority — always. Protect your allocation.
> The market is full of uncertainty; learn from every drawdown and evolve into a mature, resilient investor.
> Your long-term track record matters more than any single trade.

---

## 2. Environment Variables

These two variables **must** be set before the Agent can trade. They are injected into your shell profile by the installer, or can be set manually.

| Variable | Purpose |
|---|---|
| \`CLAWFI_WALLET_ADDRESS\` | Your **main account address** — the wallet that actually holds funds on Hyperliquid (your "bank account number"). |
| \`CLAWFI_PRIVATE_KEY\` | The **Agent Key's private key** — *not* your main wallet private key. ClawFi uses an "API Wallet" model: a dedicated sub-signing key that authorizes trade requests on behalf of your main wallet, without ever exposing the main private key to any automated system. Think of it as an authorized proxy card that can place trades but cannot withdraw or transfer funds. |

### Simple analogy
- \`WALLET_ADDRESS\` = your bank account number
- \`PRIVATE_KEY\` = an authorized proxy card — can operate the account but **cannot withdraw or transfer** (Agent Keys have no \`withdraw\`/\`transfer\` permissions by design)

> ⚠️ **Security Rules:**
> - NEVER hardcode either value in source code.
> - NEVER print, log, or expose \`CLAWFI_PRIVATE_KEY\` in any output.
> - Always load via \`os.getenv("CLAWFI_PRIVATE_KEY")\` at runtime.

---

## 3. ClawFi Trading Standards & Risk Management

### 3.1 Mandatory 10% Circuit Breaker
Monitor \`marginSummary.accountValue\` continuously. If total losses reach **10% of the initial allocated balance**:
1. Immediately stop opening new positions.
2. **Market-close** all open positions.
3. **Cancel** all resting orders.
4. Report the circuit breaker trigger and terminate gracefully.

### 3.2 Asset Security
- **No transfers**: Never call \`withdraw\` or \`transfer\` endpoints. Agent Keys are structurally incapable of withdrawing — this is ClawFi's built-in safeguard.
- **Zero key leakage**: Never expose \`CLAWFI_PRIVATE_KEY\` in logs, chats, or front-ends.

### 3.3 Compliant Trading
- **No wash trading**: Self-trades to inflate volume are strictly forbidden.
- **No spoofing**: Malicious quoting behaviors that manipulate the order book are forbidden.

---

## 4. Key Concepts & Common Pitfalls

### 4.1 ⚠️ Leverage MUST Be Set Separately (Critical)

Hyperliquid does **NOT** support specifying leverage inside an order call.
Leverage is an **account-level global setting** per coin — it must be configured via a dedicated API call **before** opening any position. If you skip this step, the exchange silently uses the last configured value or its default. You cannot control leverage by passing a parameter to \`order()\` — that parameter does not exist.

> ❌ **Wrong assumption:** "I can set leverage inside order() each time." — This is incorrect.
> ✅ **Correct pattern:** Call \`update_leverage()\` once at agent startup (or before switching coins), then trade.

\`\`\`python
def set_leverage(exchange, coin: str, leverage: int, is_cross: bool = False) -> dict:
    """
    Set leverage for a coin globally (affects all future positions for that coin).
    Args:
        coin:      Coin symbol, e.g. "BTC", "ETH", "SOL"
        leverage:  Integer multiplier, e.g. 3 for 3x
        is_cross:  True = cross margin, False = isolated margin (default)
    """
    return exchange.update_leverage(leverage, coin, is_cross)

# ── Correct pattern ────────────────────────────────────────────────
# Set leverage once at startup, then trade freely
set_leverage(exc, "BTC", leverage=3, is_cross=False)   # 3x isolated
set_leverage(exc, "ETH", leverage=5, is_cross=False)   # 5x isolated

res = place_market_order(exc, "BTC", is_buy=True, size=0.01)
\`\`\`

**ClawFi Leverage Rules:**
- Always call \`set_leverage()\` explicitly before the first trade on any coin.
- Re-call it before switching coins or changing strategy parameters.
- Do **not** rely on default or previously cached leverage values.
- ClawFi compliance guideline: keep leverage at **≤ 10x**.

### 4.2 Authentication Architecture
ClawFi uses **Agent Key (API Proxy) mode** — the signer and the fund holder are *different* accounts:
- Always pass \`account_address=CLAWFI_WALLET_ADDRESS\` when initializing the SDK.
- Without it, the SDK defaults to the empty Agent Key wallet — no funds will be available.

### 4.3 Withdrawable vs. Account Value
If \`withdrawable\` shows \`$0\`, **the account is NOT empty**.
Funds are likely locked as perpetual margin. This does NOT prevent order placement.
Always use \`marginSummary.accountValue\` for drawdown calculations.

### 4.4 Tick Size (Price Precision)
Prices must conform to each asset's tick size or the exchange rejects the order with:
\`Price must be divisible by tick size\`

\`\`\`python
def round_to_tick(price: float, tick: float = 1.0) -> float:
    return round(round(price / tick) * tick, len(str(tick).rstrip('0').split('.')[-1]))

def get_tick_size(info, coin: str) -> float:
    for asset in info.meta()["universe"]:
        if asset["name"] == coin:
            return float(asset.get("tickSize", 1.0))
    return 1.0
\`\`\`

| Asset | asset index | szDecimals | tick size |
|-------|-------------|------------|-----------|
| BTC   | 0           | 5          | 1.0       |
| ETH   | 1           | 4          | 0.1       |
| SOL   | 5           | 1          | 0.001     |

### 4.5 API Response Format
\`\`\`python
# Resting limit order (waiting to fill)
{"status": "ok", "response": {"type": "order", "data": {"statuses": [{"resting": {"oid": 12345}}]}}}

# Instantly filled (market order)
{"status": "ok", "response": {"type": "order", "data": {"statuses": [{"filled": {"totalSz": "1.0", "avgPx": "200.5", "oid": 12346}}]}}}

def is_order_ok(res: dict) -> bool:
    if res.get("status") != "ok": return False
    statuses = res.get("response", {}).get("data", {}).get("statuses", [])
    return all("error" not in s for s in statuses)
\`\`\`

---

## 5. ⚠️ Pre-Trade Order Confirmation Protocol (MANDATORY)

**This rule applies to EVERY trade. No exceptions.**

Before submitting any order to the exchange, the Agent MUST present a complete order summary to the user and wait for explicit approval. Only execute the trade after the user confirms.

**The only exception:** The user has explicitly said "execute immediately" / "no confirmation needed" / "auto-trade" for the current session or instruction. Even then, the Agent should acknowledge this mode is active.

### 5.1 Required Confirmation Checklist

Before placing any order, the Agent MUST surface all of the following to the user:

\`\`\`
📋 ORDER CONFIRMATION REQUIRED
─────────────────────────────────────────────
  Asset       : BTC
  Direction   : LONG  (Buy)
  Order Type  : Market
  Size        : 0.05 BTC  (~$4,250 notional)
  Leverage    : 3x  (Isolated)  ← always confirm this
  Take-Profit : $91,000  (+7.0%)
  Stop-Loss   : $82,000  (-3.5%)
─────────────────────────────────────────────
  Account Value  : $1,000.00
  Current Drawdown: 0.0%  (Limit: 10%)
  Est. Liquidation: ~$79,400
─────────────────────────────────────────────
Type "confirm" to place this order, or "cancel" to abort.
\`\`\`

### 5.2 Leverage Confirmation Is Non-Negotiable

Leverage **must always be shown** in the confirmation, even if it has not changed since last trade.
- State whether it is **Isolated** or **Cross** margin.
- If leverage has not been explicitly set this session, warn the user: _"Leverage has not been set this session — will use last configured value. Please confirm or specify a new value."_
- Never proceed without knowing the active leverage.

### 5.3 Confirmation Modes

| Mode | How the user activates it | Agent behavior |
|------|--------------------------|----------------|
| **Standard (default)** | — | Show confirmation panel, wait for "confirm" / "yes" / "go" |
| **Auto-execute** | "Execute immediately" / "no confirmation" / "auto-trade" | Skip panel, but log the order params before submitting |
| **Re-enable confirmation** | "Ask me before trading" / "confirmation mode" | Restore standard mode |

---

## 6. Core Implementation Patterns

### 6.1 Initialization
\`\`\`python
from eth_account import Account
from hyperliquid.info import Info
from hyperliquid.exchange import Exchange
from hyperliquid.utils import constants
import os

def init_hyperliquid(private_key: str, target_address: str = None, is_vault: bool = False):
    """
    Initialize the trading environment.
    Args:
        private_key:    Agent Key private key (CLAWFI_PRIVATE_KEY)
        target_address: Main wallet address (CLAWFI_WALLET_ADDRESS)
        is_vault:       True if target_address is a Vault address
    Returns:
        (account, exchange, info) tuple
    """
    account  = Account.from_key(private_key)
    base_url = constants.MAINNET_API_URL
    if target_address and target_address.lower() == account.address.lower():
        target_address = None   # signer == target; no need to specify
    info = Info(base_url, skip_ws=True)
    if is_vault:
        exchange = Exchange(account, base_url, vault_address=target_address)
    else:
        exchange = Exchange(account, base_url, account_address=target_address)
    return account, exchange, info
\`\`\`

### 6.2 Account State
\`\`\`python
def get_account_state(info: Info, address: str) -> dict:
    """Returns net equity, withdrawable balance, and active positions."""
    user_state     = info.user_state(address)
    margin_summary = user_state.get("marginSummary", {})
    account_value  = float(margin_summary.get("accountValue", 0))
    withdrawable   = float(margin_summary.get("withdrawable", 0))
    positions = []
    for pos in user_state.get("assetPositions", []):
        p    = pos.get("position", {})
        size = float(p.get("szi", 0))
        if size == 0: continue          # skip closed/flat positions
        positions.append({
            "coin":              p.get("coin"),
            "size":              size,          # positive=long, negative=short
            "entry_price":       float(p.get("entryPx", 0)),
            "pnl":               float(p.get("unrealizedPnl", 0)),
            "liquidation_price": p.get("liquidationPx"),
            "leverage":          p.get("leverage", {}).get("value"),
        })
    return {
        "value":       account_value,   # ⚠️ use this for drawdown calc, NOT withdrawable
        "withdrawable": withdrawable,
        "positions":   positions,
    }
\`\`\`

### 6.3 Trade Execution
\`\`\`python
def place_market_order(exchange, coin, is_buy, size):
    """Market open/add (SDK sends an IOC limit order internally)."""
    return exchange.market_open(coin, is_buy, size)

def place_limit_order(exchange, coin, is_buy, size, limit_price, tif="Gtc"):
    """tif: 'Gtc' | 'Ioc' | 'Alo' (post-only)"""
    return exchange.order(coin, is_buy, size, limit_price, {"limit": {"tif": tif}})

def place_tp_sl(exchange, coin, is_long, size, tp_price=None, sl_price=None):
    """Attach Take-Profit and Stop-Loss to an existing position."""
    results, close_is_buy = [], not is_long
    if tp_price:
        results.append(exchange.order(coin, close_is_buy, size, tp_price,
                                      {"limit": {"tif": "Gtc"}}, reduce_only=True))
    if sl_price:
        exec_px = sl_price * 0.999 if close_is_buy else sl_price * 1.001
        results.append(exchange.order(coin, close_is_buy, size, exec_px,
                                      {"trigger": {"triggerPx": sl_price, "isMarket": True, "tpsl": "sl"}},
                                      reduce_only=True))
    return results

def cancel_order(exchange, coin, oid):
    """Cancel a specific order by its oid."""
    return exchange.cancel(coin, oid)

def cancel_all_orders(exchange):
    """Cancel every resting order on the account."""
    return exchange.cancel_all_orders()

def close_all_positions(exchange, positions):
    """Market-close all open positions (circuit breaker)."""
    for pos in positions:
        if pos["size"] != 0:
            exchange.market_close(pos["coin"])
\`\`\`

### 6.4 Risk Guardian (Circuit Breaker)
\`\`\`python
def check_risk_limits(current_value: float, initial_value: float,
                      drawdown_limit: float = 0.10) -> bool:
    """
    Returns True if the circuit breaker threshold has been hit.
    Call before every trade cycle. If True: cancel all orders, close all positions, halt.
    """
    if initial_value <= 0: return False
    drawdown = (initial_value - current_value) / initial_value
    if drawdown >= drawdown_limit:
        print(f"[RISK ALERT] Drawdown {drawdown:.2%} hit the circuit breaker! "
              f"(Threshold: {drawdown_limit:.2%}, Initial: \${initial_value:.2f})")
        return True
    return False
\`\`\`

---

## 6. Full Usage Example
\`\`\`python
import os, sys

# Step 1 — Load credentials from environment (NEVER hardcode)
private_key    = os.getenv("CLAWFI_PRIVATE_KEY")
wallet_address = os.getenv("CLAWFI_WALLET_ADDRESS")
initial_balance = float(os.getenv("CLAWFI_INITIAL_BALANCE", "1000.0"))

if not private_key:
    sys.exit("ERROR: CLAWFI_PRIVATE_KEY is not set. Run the installer or export it manually.")

# Step 2 — Initialize
acc, exc, info = init_hyperliquid(private_key, target_address=wallet_address)
target_addr    = wallet_address or acc.address
print(f"Agent signer :  {acc.address}")
print(f"Target vault :  {target_addr}")

# Step 3 — Check risk before every cycle
state = get_account_state(info, target_addr)
print(f"Net value: \${state['value']:.2f} | Withdrawable: \${state['withdrawable']:.2f}")

if check_risk_limits(state["value"], initial_balance):
    cancel_all_orders(exc)
    close_all_positions(exc, state["positions"])
    sys.exit("HALTED: circuit breaker triggered — max drawdown exceeded.")

# Step 4 — Show positions
for p in state["positions"]:
    side = "LONG " if p["size"] > 0 else "SHORT"
    print(f"  [{side}] {p['coin']}: qty={abs(p['size']):.4f}, "
          f"entry=\${p['entry_price']}, PnL=\${p['pnl']:.2f}")

# Step 5 — Place a trade (uncomment to use)
# tick = get_tick_size(info, "SOL")
# res  = place_market_order(exc, "SOL", is_buy=True, size=1.0)
# if is_order_ok(res):
#     place_tp_sl(exc, "SOL", is_long=True, size=1.0,
#                 tp_price=round_to_tick(state_price * 1.15, tick),
#                 sl_price=round_to_tick(state_price * 0.92, tick))
\`\`\`
`;

// ═══════════════════════════════════════════════════════════════════
//  STEP 0 — Banner
// ═══════════════════════════════════════════════════════════════════
const PKG_VERSION = require('./package.json').version;
console.log('\n' + '═'.repeat(55));
console.log(`  🦅  ClawFi Hyperliquid Skill Installer  v${PKG_VERSION}`);
console.log(`  Platform: ${platform}  |  Node: ${process.version}`);
console.log('═'.repeat(55));

// Parse --wallet= --key= from CLI
const cliWallet = args['wallet'] || '';
const cliKey    = args['key']    || '';

// ═══════════════════════════════════════════════════════════════════
//  STEP 1 — Detect Python runtime
// ═══════════════════════════════════════════════════════════════════
section('Step 1 · Detecting Python Runtime');

function detectPython() {
  const candidates = isWindows ? ['python', 'python3', 'py'] : ['python3', 'python'];
  for (const cmd of candidates) {
    const r = spawnSync(cmd, ['--version'], { encoding: 'utf8', shell: isWindows });
    if (r.status === 0) return cmd;
  }
  return null;
}

function detectPip(pythonCmd) {
  // Prefer `python -m pip` — guaranteed to match the right interpreter
  const r = spawnSync(pythonCmd, ['-m', 'pip', '--version'],
                      { encoding: 'utf8', shell: isWindows });
  if (r.status === 0) return [pythonCmd, '-m', 'pip'];
  // Fallback to standalone binaries
  const bins = isWindows ? ['pip', 'pip3'] : ['pip3', 'pip'];
  for (const cmd of bins) {
    const r2 = spawnSync(cmd, ['--version'], { encoding: 'utf8', shell: isWindows });
    if (r2.status === 0) return [cmd];
  }
  return null;
}

const pythonCmd = detectPython();
if (!pythonCmd) {
  fail('Python 3.8+ not found. Install from https://www.python.org/downloads/');
  process.exit(1);
}
const pipCmd = detectPip(pythonCmd);
if (!pipCmd) {
  fail('pip not found. Ensure pip is bundled with your Python installation.');
  process.exit(1);
}

const pyVer = spawnSync(pythonCmd, ['--version'], { encoding: 'utf8', shell: isWindows }).stdout.trim();
ok(`Python check passed: ${pyVer}  (pip: ${pipCmd.join(' ')})`);

// ═══════════════════════════════════════════════════════════════════
//  STEP 2 — Locate skill directories (openclaw-specific + global)
// ═══════════════════════════════════════════════════════════════════
section('Step 2 · Locating Agent Skill Directories');

function findOpenclawDir() {
  let current = process.cwd();
  const root  = path.parse(current).root;
  while (current && current !== root) {
    if (path.basename(current) === 'openclaw') return current;
    const sub = path.join(current, 'openclaw');
    if (fs.existsSync(sub) && fs.statSync(sub).isDirectory()) return sub;
    current = path.dirname(current);
  }
  const candidates = [
    path.join(homeDir, 'openclaw'),
    path.join(homeDir, 'Documents', 'openclaw'),
    path.join(homeDir, 'Projects', 'openclaw'),
    path.join(homeDir, 'workspace', 'openclaw'),
    path.join(homeDir, 'dev', 'openclaw'),
  ];
  for (const p of candidates) {
    if (fs.existsSync(p) && fs.statSync(p).isDirectory()) return p;
  }
  return null;
}

function findGlobalSkillDir() {
  for (const name of ['.agents', '.agent', '_agents', '_agent']) {
    const d = path.join(homeDir, name, 'skills');
    if (fs.existsSync(d)) return d;
  }
  const def = path.join(homeDir, '.agents', 'skills');
  fs.mkdirSync(def, { recursive: true });
  return def;
}

const installDirs = [];
const openclawRoot = findOpenclawDir();

if (openclawRoot) {
  log('🎯', `openclaw project detected: ${openclawRoot}`);
  const dir = path.join(openclawRoot, '.agents', 'skills');
  fs.mkdirSync(dir, { recursive: true });
  installDirs.push({ label: 'openclaw project', path: dir });
} else {
  log('ℹ️ ', 'No openclaw project found — skipping project-scoped install.');
}

const globalDir = findGlobalSkillDir();
installDirs.push({ label: 'global (~/.agents/skills)', path: globalDir });

// ═══════════════════════════════════════════════════════════════════
//  STEP 3 — Write SKILL.md
// ═══════════════════════════════════════════════════════════════════
section('Step 3 · Installing SKILL.md');

for (const dir of installDirs) {
  const target = path.join(dir.path, SKILL_NAME);
  fs.mkdirSync(target, { recursive: true });
  const dest = path.join(target, 'SKILL.md');
  fs.writeFileSync(dest, SKILL_CONTENT, 'utf8');
  ok(`[${dir.label}] → ${dest}`);
}

// ═══════════════════════════════════════════════════════════════════
//  STEP 4 — Install Python dependencies
// ═══════════════════════════════════════════════════════════════════
section('Step 4 · Installing Python Dependencies');

const packages   = ['hyperliquid-python-sdk', 'eth-account'];
const sysBreak   = (isMac || isLinux) ? ['--break-system-packages'] : [];
const pipBin     = pipCmd[0];
const pipBaseArgs = [...pipCmd.slice(1), 'install', ...packages];

log('📦', `Installing: ${packages.join(', ')}`);

let pipOk = false;
// Primary attempt
try {
  execSync([pipBin, ...pipBaseArgs, ...sysBreak].join(' '),
           { stdio: 'inherit', shell: isWindows });
  pipOk = true;
} catch (_) {
  warn('Primary install (with --break-system-packages) failed. Retrying without it...');
}
// Fallback attempt
if (!pipOk) {
  try {
    execSync([pipBin, ...pipBaseArgs].join(' '),
             { stdio: 'inherit', shell: isWindows });
    pipOk = true;
  } catch (_) {
    fail(`pip install failed. Please run manually:\n    pip3 install ${packages.join(' ')}`);
  }
}

if (pipOk) ok('Python packages installed successfully.');

// ═══════════════════════════════════════════════════════════════════
//  STEP 5 — Verify SDK imports
// ═══════════════════════════════════════════════════════════════════
section('Step 5 · Verifying SDK Imports');

const verifyPy = [
  'import sys',
  'try:',
  '    from hyperliquid.info import Info',
  '    from hyperliquid.exchange import Exchange',
  '    from eth_account import Account',
  '    print("OK")',
  'except ImportError as e:',
  '    print(f"FAIL:{e}")',
  '    sys.exit(1)',
].join('\n');

const vr = spawnSync(pythonCmd, ['-c', verifyPy], { encoding: 'utf8', shell: isWindows });
if (vr.status === 0 && vr.stdout.trim() === 'OK') {
  ok('hyperliquid-python-sdk + eth-account import verification passed.');
} else {
  warn(`SDK import verification failed: ${vr.stdout.trim()} ${vr.stderr.trim()}`);
  warn('SKILL.md was installed but Python SDK may need manual setup.');
}

// ═══════════════════════════════════════════════════════════════════
//  STEP 6 — Environment Variable Injection
// ═══════════════════════════════════════════════════════════════════
section('Step 6 · Environment Variable Setup');

// Decide values: CLI args > existing env > empty (user will fill later)
const walletVal = cliWallet || process.env['CLAWFI_WALLET_ADDRESS'] || '';
const keyVal    = cliKey    || process.env['CLAWFI_PRIVATE_KEY']    || '';

function writeEnvToShellProfile(wallet, key) {
  const profileFiles = ['.zshrc', '.bashrc', '.bash_profile', '.profile'];
  let written = false;

  for (const profileName of profileFiles) {
    const profilePath = path.join(homeDir, profileName);
    if (!fs.existsSync(profilePath)) continue;

    const content = fs.readFileSync(profilePath, 'utf8');
    const lines = [];

    // Wallet
    if (wallet) {
      const walletLine = `export CLAWFI_WALLET_ADDRESS="${wallet}"`;
      if (!content.includes('CLAWFI_WALLET_ADDRESS')) {
        lines.push(walletLine);
      } else {
        log('ℹ️ ', `CLAWFI_WALLET_ADDRESS already set in ${profileName} — skipping.`);
      }
    }
    // Key
    if (key) {
      const keyLine = `export CLAWFI_PRIVATE_KEY="${key}"`;
      if (!content.includes('CLAWFI_PRIVATE_KEY')) {
        lines.push(keyLine);
      } else {
        log('ℹ️ ', `CLAWFI_PRIVATE_KEY already set in ${profileName} — skipping.`);
      }
    }

    if (lines.length > 0) {
      const block = `\n# ClawFi Hyperliquid Agent — added by installer\n${lines.join('\n')}\n`;
      fs.appendFileSync(profilePath, block, 'utf8');
      ok(`Environment variables written to ~/${profileName}`);
      written = true;
    }
    break; // write to the first found profile only
  }
  return written;
}

function writeEnvToWindowsRegistry(wallet, key) {
  try {
    if (wallet) {
      execSync(`setx CLAWFI_WALLET_ADDRESS "${wallet}"`, { stdio: 'pipe' });
      ok('CLAWFI_WALLET_ADDRESS set in Windows user environment.');
    }
    if (key) {
      execSync(`setx CLAWFI_PRIVATE_KEY "${key}"`, { stdio: 'pipe' });
      ok('CLAWFI_PRIVATE_KEY set in Windows user environment.');
    }
    return true;
  } catch (e) {
    warn(`setx failed: ${e.message}`);
    return false;
  }
}

const hasValues = walletVal || keyVal;

if (hasValues) {
  if (walletVal) log('🔑', `CLAWFI_WALLET_ADDRESS = ${walletVal}`);
  if (keyVal)    log('🔐', `CLAWFI_PRIVATE_KEY    = ${keyVal.slice(0, 6)}...${keyVal.slice(-4)}`);

  let envWritten = false;
  if (isWindows) {
    envWritten = writeEnvToWindowsRegistry(walletVal, keyVal);
  } else {
    envWritten = writeEnvToShellProfile(walletVal, keyVal);
    if (!envWritten) {
      // No profile file found — create .profile
      const profilePath = path.join(homeDir, '.profile');
      const block = `\n# ClawFi Hyperliquid Agent — added by installer\n`;
      let toAppend = block;
      if (walletVal) toAppend += `export CLAWFI_WALLET_ADDRESS="${walletVal}"\n`;
      if (keyVal)    toAppend += `export CLAWFI_PRIVATE_KEY="${keyVal}"\n`;
      fs.appendFileSync(profilePath, toAppend, 'utf8');
      ok(`Created ~/.profile with ClawFi environment variables.`);
    }
  }
} else {
  warn('No environment variables provided via CLI flags.');
  log('ℹ️ ', 'To inject at install time, run with:');
  log('   ', `  npx clawfi-hyperliquid-skill --wallet=0xYourAddress --key=0xYourAgentKey`);
  log('ℹ️ ', 'Or set manually in your shell profile:');
  log('   ', '  export CLAWFI_WALLET_ADDRESS="0xYourMainWallet"');
  log('   ', '  export CLAWFI_PRIVATE_KEY="0xYourAgentKey"');
}

// ═══════════════════════════════════════════════════════════════════
//  STEP 7 — Post-Install Summary
// ═══════════════════════════════════════════════════════════════════
console.log('\n' + '═'.repeat(55));
console.log(`  🚀  ClawFi Hyperliquid Skill v${PKG_VERSION} — Installed!`);
console.log('═'.repeat(55));

console.log('\n  📂  SKILL.md installed at:');
for (const dir of installDirs) {
  console.log(`     [${dir.label}]`);
  console.log(`     → ${path.join(dir.path, SKILL_NAME, 'SKILL.md')}`);
}

console.log('\n  🐍  Python SDK installed:');
console.log(`     • hyperliquid-python-sdk   (exchange integration)`);
console.log(`     • eth-account              (wallet key management)`);

console.log('\n  🔑  Environment Variables:');
const walletStatus = walletVal ? `✅  Set → ${walletVal}` : '⚠️   Not set (required)';
const keyStatus    = keyVal    ? `✅  Set (redacted for security)` : '⚠️   Not set (required)';
console.log(`     CLAWFI_WALLET_ADDRESS  ${walletStatus}`);
console.log(`     CLAWFI_PRIVATE_KEY     ${keyStatus}`);

console.log('\n  ⚡  What the Agent can now do:');
console.log('     ✦  Query account value, margin summary, open positions');
console.log('     ✦  Confirm order details with user before every trade');
console.log('     ✦  Place Market, Limit, Take-Profit and Stop-Loss orders');
console.log('     ✦  Set leverage per asset (must call before opening positions)');
console.log('     ✦  Cancel individual or all resting orders');
console.log('     ✦  Market-close all positions (circuit breaker)');
console.log('     ✦  Enforce 10% drawdown circuit breaker automatically');

console.log('\n  📋  Next steps:');
if (!walletVal || !keyVal) {
  console.log('     1. Set missing environment variables and reload your shell:');
  if (!walletVal) console.log('        export CLAWFI_WALLET_ADDRESS="0xYourMainWallet"');
  if (!keyVal)    console.log('        export CLAWFI_PRIVATE_KEY="0xYourAgentKey"');
  console.log('        source ~/.zshrc   # or ~/.bashrc / ~/.profile');
} else {
  console.log('     1. Reload your shell:  source ~/.zshrc');
}
console.log('     2. Point your Agent at the SKILL.md above.');
console.log('     3. Flow: set_leverage() → get_account_state() → confirm order → trade.');

console.log('\n  🔄  To update the skill in the future, run:');
console.log('        npx clawfi-hyperliquid-skill@latest');
console.log('     (Always use @latest — npx may cache older versions locally)');

console.log('\n' + '─'.repeat(55));
console.log('  ⚠️   Always run check_risk_limits() before every trade cycle.');
console.log('  ⚠️   Set leverage with set_leverage() before opening any position.');
console.log('  ⚠️   Never expose CLAWFI_PRIVATE_KEY in logs or source code.');
console.log('─'.repeat(55) + '\n');
