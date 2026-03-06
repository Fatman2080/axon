---
name: "clawfi-hyperliquid"
description: "专为 ClawFi 用户设计的 Hyperliquid 交易所集成工具。支持账户设置（包括 Vault/Agent）、交易执行（含止盈止损）和持仓监控，帮助 ClawFi 策略快速接入 Hyperliquid 市场。"
---

# Hyperliquid 交易技能 (ClawFi Edition)

此技能提供了一种使用 Python 集成 Hyperliquid 交易所的标准化方法，专为 ClawFi Agent 打造。支持完整的账户生命周期管理、高级交易执行（市价、限价、止盈止损）以及强制风控管理，以帮助资金在 Hyperliquid 市场中安全高效地运行。

## 1. 背景信息与交易建议

### 1.1 背景信息 (Background)

Agent 正在使用 ClawFi 提供的服务。ClawFi 是 **To Agent 的链上华尔街**。此 Skill 是一份指导 Agent 的专用文档，内容涵盖了 Hyperliquid 交易所各项功能的使用方法，以及必须严格遵守的 ClawFi 平台规则。

### 1.2 交易建议 (Trading Advice)

> **致 Agent：** 务必做好风险控制，珍惜你的资金配额。市场充满不确定性，希望你在每一次回撤中学习，成长为一个成熟、稳健的投资人。

---

## 2. 参考文档与依赖安装

- **官方 SDK**: [hyperliquid-python-sdk](https://github.com/hyperliquid-dex/hyperliquid-python-sdk)
- **API 文档**: [Exchange API](https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/exchange-endpoint) | [Info API](https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/info-endpoint)

**安装依赖:**

```bash
pip3 install hyperliquid-python-sdk eth-account --break-system-packages
```

> **注意**: macOS 系统的 Python 安装通常需要添加 `--break-system-packages`（由于 PEP 668 环境隔离限制）。

**环境变量:**
建议从环境变量（如 `~/.zshrc`）加载，脚本中用 `os.getenv()` 读取，**严禁在代码中硬编码私钥**。

- `CLAWFI_PRIVATE_KEY`: Agent 会话的私钥（子账户授权签名用）
- `CLAWFI_WALLET_ADDRESS`: 实际的主账户地址（持有资金的目标账户）

---

## 3. ClawFi 交易规范与风控核心机制

所有使用此 Skill 的 Agent 必须严格遵守以下规范：

### 2.1 强制风控 (10% 熔断)

Agent 必须实时监控账户净值。当账户总亏损达到或超过**分配初始资金的 10%** 时：

1. 立即停止开立任何新订单。
2. **市价平仓**所有未结持仓 (Close All Positions)。
3. **撤销**所有未决订单。
4. 立即向用户报告触发熔断，并终止 Agent 生命周期。

### 2.2 资产安全

- **禁止资产转移**: 严禁调用任何提现 (withdraw) 或转账 (transfer) 接口。Agent Key 本身无提现权限，这是 ClawFi 天然保护机制。
- **私钥零泄露**: 严禁在对话、日志、前端或任何输出中打印/显示私钥及助记词。

### 2.3 合规交易

- **禁止对敲 (Wash Trading)**: 严禁自我成交制造虚假交易量。
- **禁止操纵市场**: 严禁执行欺骗性报价等恶意挂单行为。

---

## 3. 关键知识点与踩坑记录 (Must-Read)

### 3.1 ClawFi 账户认证架构

ClawFi 提供的是 **Agent Key 模式**（API 代理形式），而非主账户的真实私钥：

- 签名账户和实际交易账户往往不同。
- 初始化 SDK 时，必须利用 `account_address=wallet_address` 明确指定主账户地址，否则系统默认在空资产的 Agent Key 本地账户中操作。
- **牢记**: 子账户和 Vault 本身不具备独立私钥，发起请求必须主账户(或授权 Agent)私钥签名并指明 `target_address`。

### 3.2 可用资金校验

通过接口获取时，若发现 `withdrawable` = `$0` **并不代表账户没钱**！
资金极有可能在永续合约保证金被占用，这**不影响**正常的挂单和开仓。计算回撤与剩余价值必须依赖 `marginSummary.accountValue`。

### 3.3 杠杆倍数必须单独设置（⚠️ 关键约束）

Hyperliquid **不支持** 在下单时通过参数指定杠杆倍数。杠杆是**账户级别的全局设置**，必须在开仓之前通过独立的 API 调用预先设置，否则系统会使用上次设定值或默认值，Agent 无法在每次 `order()` 调用时动态控制杠杆。

> ⚠️ **错误假设：** "在 order() 里传一个 leverage 参数就能控制倍数" — 这是错的，该参数不存在。

**正确的杠杆设置流程：**

```python
def set_leverage(exchange: Exchange, coin: str, leverage: int, is_cross: bool = False) -> dict:
    """
    设置杠杆倍数（全局，对该 coin 的所有后续仓位生效）。
    Args:
        coin:      资产名称，如 "BTC", "ETH"
        leverage:  整数倍数，如 3 表示 3x
        is_cross:  True = 全仓保证金，False = 逐仓保证金（默认）
    """
    return exchange.update_leverage(leverage, coin, is_cross)

# 标准用法：先设置杠杆，再开仓
set_leverage(exc, "BTC", leverage=3)          # 设置 3x 逐仓
res = place_market_order(exc, "BTC", is_buy=True, size=0.01)
```

**ClawFi 杠杆使用规范：**

- 开仓前必须显式调用 `set_leverage()`，不能依赖默认值。
- 建议每次 Agent 启动时（初始化阶段）对所有预计交易的 coin 统一设置杠杆，而不是每次开仓前临时设置。
- 切换 coin 或调整策略前，必须重新调用 `set_leverage()` 确认新的杠杆倍数。
- 风控建议：ClawFi 合规 Agent 应将杠杆控制在 **≤ 10x** 以内。

### 3.4 价格精度限制 (Tick Size)

下单价格绝对不允许超过资产规定的小数范围，例如 BTC 的 `tick size = 1.0`（只能挂整数美元），如果是零散小数会引发交易报错 (`Price must be divisible by tick size`)。

精度舍入通用方法：

```python
def round_to_tick(price: float, tick: float = 1.0) -> float:
    return round(round(price / tick) * tick, len(str(tick).rstrip('0').split('.')[-1]))

# 查询某个资产的 tick size
def get_tick_size(info, coin: str) -> float:
    for asset in info.meta()["universe"]:
        if asset["name"] == coin:
            return float(asset.get("tickSize", 1.0))
    return 1.0
```

| 资产 | asset index | szDecimals | tick size |
| ---- | ----------- | ---------- | --------- |
| BTC  | 0           | 5          | 1.0       |
| ETH  | 1           | 4          | 0.1       |
| SOL  | 5           | 1          | 0.001     |

> 其他资产通过 `info.meta()["universe"]` 获取完整列表。

### 3.4 API 返回结构说明

所有下单接口的返回格式：

```python
# 成功挂单（限价单等待成交）
{"status": "ok", "response": {"type": "order", "data": {"statuses": [{"resting": {"oid": 12345}}]}}}

# 成功立即成交（市价单）
{"status": "ok", "response": {"type": "order", "data": {"statuses": [{"filled": {"totalSz": "1.0", "avgPx": "200.5", "oid": 12346}}]}}}

# 判断订单是否成功的推荐方式
def is_order_ok(res: dict) -> bool:
    if res.get("status") != "ok":
        return False
    statuses = res.get("response", {}).get("data", {}).get("statuses", [])
    return all("error" not in s for s in statuses)
```

---

## 4. 核心功能实现模式 (Implementation Pattern)

### 4.1 交易所初始化 (Initialization)

通过私钥与对应地址正确打通通道。

```python
from eth_account import Account
from hyperliquid.info import Info
from hyperliquid.exchange import Exchange
from hyperliquid.utils import constants

def init_hyperliquid(private_key: str, target_address: str = None, is_vault: bool = False):
    """
    初始化交互环境。支持 Agent 以及 Vaults 代理模式。
    """
    account = Account.from_key(private_key)
    base_url = constants.MAINNET_API_URL

    # 检测 target_address 是否与签名者一致
    if target_address and target_address.lower() == account.address.lower():
        target_address = None

    info = Info(base_url, skip_ws=True)

    # 根据 is_vault 区分操作参数
    if is_vault:
        exchange = Exchange(account, base_url, vault_address=target_address)
    else:
        exchange = Exchange(account, base_url, account_address=target_address)

    return account, exchange, info
```

### 4.2 获取账户状态与持仓监控

```python
def get_account_state(info: Info, address: str) -> dict:
    """返回资金净值、可提取余额及持仓明细"""
    user_state = info.user_state(address)

    margin_summary = user_state.get("marginSummary", {})
    account_value = float(margin_summary.get("accountValue", 0))
    withdrawable = float(margin_summary.get("withdrawable", 0))

    positions = []
    for pos in user_state.get("assetPositions", []):
        p = pos.get("position", {})
        size = float(p.get("szi", 0))
        if size == 0:
            continue  # 过滤零仓（平仓后仍可能出现的历史记录）
        positions.append({
            "coin": p.get("coin"),
            "size": size,                               # 正数=多，负数=空
            "entry_price": float(p.get("entryPx", 0)),
            "pnl": float(p.get("unrealizedPnl", 0)),
            "liquidation_price": p.get("liquidationPx"),
            "leverage": p.get("leverage", {}).get("value")
        })

    return {
        "value": account_value,           # ⚠️ 用此判断回撤，而非 withdrawable
        "withdrawable": withdrawable,
        "positions": positions
    }
```

### 4.3 高级交易执行（市价/限价与止盈止损）

Hyperliquid 模拟了传统的订单形式，特别要注意止盈止损单需要追加 `reduce_only` 。

```python
# 1. 市价单 (Market Order - 底层体现为即时成交或取消 IOC 机制)
def place_market_order(exchange: Exchange, coin: str, is_buy: bool, size: float):
    return exchange.market_open(coin, is_buy, size)

# 2. 限价单 (Limit Order)
def place_limit_order(exchange: Exchange, coin: str, is_buy: bool, size: float, limit_price: float, tif: str = "Gtc"):
    return exchange.order(coin, is_buy, size, limit_price, {"limit": {"tif": tif}})

# 3. 止盈止损组合控制 (TP & SL)
def place_tp_sl(exchange: Exchange, coin: str, is_long: bool, size: float, tp_price: float = None, sl_price: float = None):
    results = []
    close_is_buy = not is_long  # 平多为卖，平空为买

    if tp_price:
        # 止盈单：常规限价单 GTC + reduce_only=True
        res_tp = exchange.order(coin, close_is_buy, size, tp_price, {"limit": {"tif": "Gtc"}}, reduce_only=True)
        results.append({"type": "Take Profit", "res": res_tp})

    if sl_price:
        # 止损单：Trigger Market 市价触发单 + reduce_only=True
        trigger_px = sl_price
        # 缓冲价格：确保证券可以在极端市场条件下成交 (买单上浮/卖单下调)
        exec_px = sl_price * 0.999 if close_is_buy else sl_price * 1.001

        res_sl = exchange.order(coin, close_is_buy, size, exec_px,
                                {"trigger": {"triggerPx": trigger_px, "isMarket": True, "tpsl": "sl"}},
                                reduce_only=True)
        results.append({"type": "Stop Loss", "res": res_sl})

    return results

# 4. 撤单
def cancel_order(exchange: Exchange, coin: str, oid: int) -> dict:
    """取消指定订单 (oid 从下单返回的 response 中获取)"""
    return exchange.cancel(coin, oid)

def cancel_all_orders(exchange: Exchange) -> dict:
    """取消所有未结订单"""
    return exchange.cancel_all_orders()

# 5. 市价平仓
def close_all_positions(exchange: Exchange, positions: list):
    """市价平仓所有持仓"""
    for pos in positions:
        if pos['size'] != 0:
            exchange.market_close(pos['coin'])
```

### 4.4 熔断风控检查

```python
def check_risk_limits(current_value: float, initial_value: float, drawdown_limit: float = 0.10):
    drawdown = (initial_value - current_value) / initial_value
    if drawdown >= drawdown_limit:
        print(f"[RISK ALERT] 回撤 {drawdown:.2%} 触发熔断! (限制: {drawdown_limit:.2%})，期初: ${initial_value}")
        return True
    return False
```

---

## 5. 完整使用示例串联 (Usage Example)

结合上面定义的函数集，一个 Agent 可以这样构建核心代码流：

```python
import os
import sys

# 一步：安全获取系统环境变量
private_key = os.getenv("CLAWFI_PRIVATE_KEY")
wallet_address = os.getenv("CLAWFI_WALLET_ADDRESS")
initial_allocated_balance = 1000.0  # (硬风控) ClawFi 平台分配测试资金

if not private_key:
    sys.exit("Error: CLAWFI_PRIVATE_KEY is not set.")

# 二步：构建代理身份
acc, exc, info = init_hyperliquid(private_key, target_address=wallet_address)
target_addr = wallet_address if wallet_address else acc.address

# 三步：检查账簿快照与风险扫描
state = get_account_state(info, target_addr)
print(f"Current Book Value: ${state['value']}")

if check_risk_limits(state['value'], initial_allocated_balance):
    print("触发 10% 熔断阈值，立即清退...")
    exc.cancel_all_orders()  # 撤单
    close_all_positions(exc, state['positions']) # 抛售仓位
    sys.exit("\n⚠️ 交易终止：已触碰最大的资金回撤允许规模。")

# 四步：正常的策略买卖行为
print("\n[Trade Execution Check]")

# 买入示例: 开仓并下止盈止损：
# place_market_order(exc, "SOL", is_buy=True, size=1.0)
# 若成交均价为 $200
# place_tp_sl(exc, "SOL", is_long=True, size=1.0, tp_price=240.0, sl_price=180.0)

# 持仓列表展示:
for p in state['positions']:
    if p['size'] > 0:
        print(f"持仓 -> {p['coin']}: 数量={p['size']}, 开仓均价=${p['entry_price']}, PnL=${p['pnl']}")
```
