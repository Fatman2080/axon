# Axon 主网发布清单（最后一公里）

> 适用版本：`v1.0.0`  
> 适用链 ID：`axon_8210-1`

---

## 0) 发布角色分工（建议）

- 发布负责人（Release Lead）：统一口径、执行冻结/解冻
- 验证者协调员（Validator Ops）：收集 gentx、核对 genesis hash
- 基础设施负责人（Infra）：RPC/监控/告警/备份
- 安全负责人（Security）：密钥与权限审计、应急响应

---

## 1) T-24h：冻结前准备

- [ ] 冻结代码版本（仅允许阻塞级修复）
- [ ] 在发布分支确认版本号为 `v1.0.0`
- [ ] 备份所有主节点（配置、数据、密钥）
- [ ] 验证节点系统时间同步（NTP）
- [ ] 确认监控与告警可用（CPU/内存/磁盘/出块/peer 数）

参考命令：

```bash
cd axon
make build
./build/axond version --long
bash scripts/backup-node.sh --home ~/.axon-mainnet
```

---

## 2) T-12h：创世文件冻结流程

### 2.1 生成主网 genesis 模板

```bash
cd axon
bash scripts/init_mainnet.sh --home ~/.axon-mainnet
```

### 2.2 添加初始验证者并收集 gentx

```bash
./build/axond genesis add-genesis-account <validator_addr> <amount>aaxon --home ~/.axon-mainnet
./build/axond genesis gentx <key_name> <stake_amount>aaxon --chain-id axon_8210-1 --home ~/.axon-mainnet
./build/axond genesis collect-gentxs --home ~/.axon-mainnet
./build/axond genesis validate ~/.axon-mainnet/config/genesis.json
```

### 2.3 冻结 hash（必须全网一致）

```bash
shasum -a 256 ~/.axon-mainnet/config/genesis.json
# Linux 可用：sha256sum ~/.axon-mainnet/config/genesis.json
```

检查项：

- [ ] 所有验证者拿到**同一份** `genesis.json`
- [ ] 所有验证者回报的 SHA256 完全一致
- [ ] `genesis validate` 全部通过

---

## 3) T-2h：启动前健康检查

运行一键预检（建议每个验证者节点都跑）：

```bash
cd axon
bash scripts/mainnet_preflight.sh \
  --binary ./build/axond \
  --home ~/.axon-mainnet \
  --expected-chain-id axon_8210-1 \
  --expected-version v1.0.0 \
  --expected-min-gas-prices 10000000000aaxon
```

检查项：

- [ ] `FAIL=0`（有 FAIL 一律禁止上线）
- [ ] `WARN` 项已确认风险可接受
- [ ] `minimum-gas-prices` 已配置
- [ ] `seeds` 或 `persistent_peers` 至少配置其一
- [ ] 验证者密钥权限不宽松（建议 `600`）

---

## 4) T0：主网上线执行

```bash
./build/axond start --home ~/.axon-mainnet
```

上线后前 30 分钟重点观察：

- [ ] 节点持续出块（无长时间卡块）
- [ ] 区块时间接近目标（约 5 秒）
- [ ] peer 数稳定，不持续下降
- [ ] RPC/JSON-RPC 正常响应
- [ ] 内存与磁盘增长在预期范围内

---

## 5) T+1h：发布后确认

- [ ] 公布正式 Genesis SHA256
- [ ] 公布 RPC/Explorer 地址
- [ ] 收集验证者运行状态与初始高度
- [ ] 记录首小时异常日志与处理结果

---

## 6) 回滚与应急预案（必须演练）

### 6.1 启动失败（未出块）

- 触发条件：`FAIL` 类配置错误、全网未形成共识
- 处理步骤：
  1. 停止节点
  2. 从冻结备份恢复配置与 genesis
  3. 逐项复核 chain-id / genesis hash / peers / gas price
  4. 重新统一启动窗口

### 6.2 已出块但出现严重参数错误

- 触发条件：危及资产安全或网络可用性
- 处理建议：
  1. 立即发布公告并暂停新增接入
  2. 启动治理/升级流程（`x/gov` + `x/upgrade`）
  3. 产出 RCA（根因分析）与修复时间线

---

## 7) 值班 SLO 建议（首周）

- 区块可用性：`>= 99.9%`
- 验证者在线率：`>= 95%`
- 关键告警响应时间：`<= 10 分钟`
- 严重事件公告时间：`<= 30 分钟`

---

## 8) 发布日执行记录（模板）

- 发布时间：`YYYY-MM-DD HH:mm:ss UTC`
- 发布版本：`v1.0.0`
- Genesis SHA256：`<hash>`
- 首块高度与时间：`<height/time>`
- 参与验证者数：`<n>`
- 负责人签字：`<name>`

