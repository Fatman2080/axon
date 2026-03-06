# ClawFi Hyperliquid Trading Skill 🦅

A one-click, cross-platform installation tool designed for AI Agents operating on **ClawFi — The On-Chain Wall Street for Agents**.

This package automatically:

1. **Installs SDKs**: Fetches `hyperliquid-python-sdk` and `eth-account` for your Python environment.
2. **Deploys Documentation**: Injects the canonical `SKILL.md` (trading rules, integration limits, and API examples) directly into your agent's context directory (Global `~/.agents/skills` or a local `openclaw` project).
3. **Injects Variables**: Optionally writes your authorized keys directly into your shell profile or Windows environment.

---

## ⚡ Installation

```bash
npx clawfi-hyperliquid-skill@latest \
  --wallet=0xYourMainWalletAddress \
  --key=0xYourAgentPrivateKey
```

> **Always use `@latest`** — without it, npx may use a locally cached older version and your `SKILL.md` won't receive updates.

_(You can also run without `--wallet` / `--key` and set the variables manually afterward.)_

## 🔄 Updating

Re-running the installer always **overwrites** the existing `SKILL.md` with the latest version — no manual cleanup needed.

```bash
npx clawfi-hyperliquid-skill@latest
```

The installer prints the installed package version in the summary, so you can always verify which version of the skill is active.

---

## 🔑 Environment Variables

| Variable                | Description                                                                  |
| ----------------------- | ---------------------------------------------------------------------------- |
| `CLAWFI_WALLET_ADDRESS` | Your **main account address** that holds the actual trading balance.         |
| `CLAWFI_PRIVATE_KEY`    | Your **Agent Key** (Proxy API Key) private key. Never the main wallet's key! |

## 🛡️ Core Trading Restrictions (ClawFi Rules)

1. **Pre-Trade Confirmation**: The Agent MUST show a full order summary (asset, direction, size, leverage, TP/SL, estimated liquidation) and wait for your explicit "confirm" before placing any trade — unless you explicitly enable auto-trade mode.
2. **Leverage is Global, Set it Explicitly**: You cannot set leverage inside an order call. Call `update_leverage()` per asset before opening any position. Leverage **must be ≤ 10x**.
3. **10% Circuit Breaker**: The Agent must halt and close all positions if total drawdown reaches 10% of the initial allocated balance.
4. **No Asset Transfers**: Your Agent Key has zero withdrawal/transfer permissions by design.
5. **No Market Manipulation**: Wash trading and spoofing are strictly forbidden.

## 📚 What's Next?

After installation, tell your Agent to read its new skill documentation.

- **Global Agents**: `~/.agents/skills/clawfi-hyperliquid/SKILL.md`
- **OpenClaw Agents**: `[project_root]/.agents/skills/clawfi-hyperliquid/SKILL.md`

Your Agent will learn how to initialize the connection securely, confirm orders before execution, manage leverage correctly, and place Limit/Market/TP/SL orders safely.
