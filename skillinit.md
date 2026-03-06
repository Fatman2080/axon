---
name: "clawfi-hyperliquid"
description: "专为 ClawFi 用户设计的 Hyperliquid 交易所集成工具。支持账户设置（包括 Vault/Agent）、交易执行（含止盈止损）和持仓监控，帮助 ClawFi 策略快速接入 Hyperliquid 市场。"
---

# Hyperliquid 交易技能 (ClawFi Edition)

## 参考文档

- 官方 SDK：https://github.com/hyperliquid-dex/hyperliquid-python-sdk
- Exchange API 文档：https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/exchange-endpoint
- Info API 文档：https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/info-endpoint

## 环境变量（已配置）

CLAWFI_PRIVATE_KEY # Agent 私钥（子账户签名用）
CLAWFI_WALLET_ADDRESS # 主账户地址（实际操作的目标账户）

从 ~/.zshrc 加载，脚本中用 os.getenv() 读取。

## 安装依赖

pip3 install hyperliquid-python-sdk eth-account --break-system-packages

> macOS 系统 Python 需要加 --break-system-packages（PEP 668 限制）

## 关键知识点（踩坑记录）

### 1. ClawFi 账户结构

ClawFi 提供的是 Agent Key 模式，不是主账户私钥：

- CLAWFI_PRIVATE_KEY → Agent 的私钥，对应签名地址（如 0x6EcD...）
- CLAWFI_WALLET_ADDRESS → 主账户地址（如 0x4852...），是实际持有资金的账户
- 两个地址不同，这是正常的 Agent 架构

初始化时必须用 account_address=wallet_address 指定目标账户，否则操作的是空的 Agent 账户。

### 2. 子账户没有私钥（重要！）

官方文档明确：Subaccounts 和 Vaults 没有私钥。
通过子账户交易，必须用主账户私钥签名，并设置 vault_address。
ClawFi 的 Agent Key 是另一种机制（API Wallet），与子账户不同。

### 3. `withdrawable` 为 $0 不等于没钱

账户净值 $100、withdrawable $0 说明资金在永续合约保证金中被占用，不影响开仓。实际可用保证金通过 marginSummary.accountValue 判断。

### 4. 价格精度（tick size）

BTC 的 tick size = 1.0（整数美元），下单价格必须是整数，否则报错：
Price must be divisible by tick size. asset=0

其他资产 tick size 不同，通用写法：
def round_to_tick(price, tick=1.0):
return round(round(price / tick) \* tick, 1)

### 5. 市价单实现方式

Hyperliquid 没有真正的"市价单"API，用 IOC 限价单 模拟：

- 价格设为当前价上浮 0.5%（买入）或下浮 0.5%（卖出）
- tif = "Ioc"：立即成交，未成交部分取消

### 6. 止盈止损单

止损用 trigger order，止盈用普通限价 GTC，都需要设 reduce_only=True：

# 止盈（限价 GTC）

exchange.order("BTC", False, size, tp_price, {"limit": {"tif": "Gtc"}}, reduce_only=True)

# 止损（trigger market）

exchange.order("BTC", False, size, sl_trigger \* 0.999,
{"trigger": {"triggerPx": sl_trigger, "isMarket": True, "tpsl": "sl"}},
reduce_only=True)

## ClawFi 交易规范与风控

### 强制风控（10% 熔断）

当账户总亏损 ≥ 分配初始资金的 10%：

1. 立即市价平仓所有持仓
2. 停止开新单
3. 向用户报告并终止

### 资产安全

- 严禁调用 withdraw / transfer 接口
- 严禁在日志或对话中显示私钥
- Agent Key 本身无法提现（天然保护）

### 合规交易

- 严禁对敲（wash trading）
- 严禁价格操纵性挂单

## 完整交易流程（带止盈止损）

[见原文代码块，含初始化/查账户/获取价格/开仓/止盈止损完整示例]

## 查看持仓

state = info.user_state(wallet_address)
for pos in state.get("assetPositions", []):
p = pos["position"]
print(f"{p['coin']}: size={p['szi']}, entry=${p['entryPx']}, PnL=${p['unrealizedPnl']}, liq=${p['liquidationPx']}")

## 平仓

# 市价平仓（IOC 反向单）

exchange.order("BTC", False, size,
round_to_tick(btc_price \* 0.995),
{"limit": {"tif": "Ioc"}}, reduce_only=True)

# 或者用 market_close

exchange.market_close("BTC")

## 风控检查

def check_risk(current_value, initial_value, limit=0.10):
drawdown = (initial_value - current_value) / initial_value
if drawdown >= limit:
print(f"[RISK] 回撤 {drawdown:.2%} 超限，触发熔断")
return True
return False

## 资产查询（BTC 精度参考）

| 资产 | asset index | szDecimals | tick size |
| ---- | ----------- | ---------- | --------- |
| BTC  | 0           | 5          | 1.0       |
| ETH  | 1           | 4          | 0.1       |
| SOL  | 5           | 1          | 0.001     |

其他资产通过 info.meta()["universe"] 获取完整列表。
