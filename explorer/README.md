# Axon Block Explorer

基于 [Blockscout](https://github.com/blockscout/blockscout) 的 Axon 链区块浏览器。

## 快速启动

```bash
# 确保 axond 正在运行（JSON-RPC 在 localhost:8545）
cd explorer
docker-compose up -d
```

浏览器访问: http://localhost:4000

## 服务

| 服务 | 端口 | 说明 |
|------|------|------|
| Blockscout UI | 4000 | 区块浏览器主界面 |
| PostgreSQL | 5432 | 数据库（内部） |
| Redis | 6379 | 缓存（内部） |
| Smart Contract Verifier | 8043 | 合约验证服务 |

## 停止

```bash
docker-compose down        # 停止服务
docker-compose down -v     # 停止并清除数据
```

## 连接配置

默认连接到 `localhost:8545`（axond JSON-RPC）。修改 `docker-compose.yml` 中的 `ETHEREUM_JSONRPC_HTTP_URL` 可以指向其他节点。
