# Agent Arena Backend Changes

## Goal

This document summarizes the backend changes made to support the formal `agent-arena` service while keeping the existing sync and storage pipeline intact.

The backend implementation intentionally followed a minimal-intrusion strategy:

- keep the current `runSyncRound()` persistence flow unchanged
- derive arena state from existing tables instead of creating arena-specific tables
- expose a dedicated arena read model through snapshot + websocket APIs
- use an in-process shared hub so multiple frontend viewers receive the same updates

## Main Backend Changes

### 1. Added formal arena APIs

Registered in [handlers_public.go](src/handlers_public.go):

- `GET /api/agent-arena/snapshot`
- `GET /api/agent-arena/ws`

Current behavior:

- `snapshot` returns the current arena snapshot
- `ws` sends one initial snapshot, then streams incremental arena events

### 2. Added arena protocol models

Defined in [models_arena.go](src/models_arena.go):

- `AgentArenaLatestFill`
- `AgentArenaAgent`
- `AgentArenaSnapshotResponse`
- `AgentArenaEvent`
- `AgentArenaWSMessage`

This keeps the arena protocol separate from the existing market, vault, and dashboard DTOs.

### 3. Added arena-specific store query

Implemented in [store_arena.go](src/store_arena.go):

- `listLatestAgentFillsForAssignedAgents()`

This helper derives arena-specific fields from existing persisted data:

- `lastRealizedPnl`
- `lastFillId`
- `lastFillTime`

No new database tables or schema migrations were introduced.

### 4. Added arena snapshot builder and diff logic

Implemented in [handlers_public_arena.go](src/handlers_public_arena.go):

- `handleAgentArenaSnapshot`
- `handleAgentArenaWS`
- `buildAgentArenaSnapshot`
- `diffAgentArenaSnapshots`
- `reconcileAgentArenaState`

Responsibilities:

- build arena-facing agent state from `agent_accounts`, `agent_snapshots`, and `agent_fills`
- infer visual statuses such as `running`, `phone`, `idle`, and `dead`
- diff old vs new snapshots into a compact event stream
- push those events into the shared hub

### 5. Added shared websocket broadcast hub

Implemented in [arena_hub.go](src/arena_hub.go).

The hub:

- stores the latest arena snapshot in memory
- tracks active websocket subscribers
- subscribes a connection atomically with the current snapshot
- broadcasts one reconciled event batch to all clients
- drops slow subscribers instead of blocking the broadcaster

This replaced the earlier per-connection polling + diff approach.

## Existing Files Changed

### [main.go](src/main.go)

Changes:

- initializes `arenaHub`
- performs one initial `reconcileAgentArenaState()` during startup

This ensures the first viewer can receive a valid snapshot immediately after boot.

### [handlers_public.go](src/handlers_public.go)

Changes:

- `Server` now holds `arenaHub *agentArenaHub`
- public routes now register `agent-arena/snapshot` and `agent-arena/ws`
- user-facing assignment flows trigger arena reconciliation after state changes

### [agent_service.go](src/agent_service.go)

Changes:

- triggers `reconcileAgentArenaState()` after sync round completion

This keeps arena state aligned with the existing persisted sync output without rewriting the sync pipeline.

### [handlers_admin.go](src/handlers_admin.go)

Changes:

- added reconciliation after admin operations that can affect arena membership or status

Covered paths now include:

- batch delete agent accounts
- batch delete users
- revoke invite
- revoke agent
- reassign agent
- revoke user agent

## Runtime Architecture

The current backend data flow is:

1. existing business logic updates `agent_accounts`, `agent_snapshots`, and `agent_fills`
2. selected state-changing paths call `reconcileAgentArenaState()`
3. reconciliation rebuilds a fresh arena snapshot from existing tables
4. the hub diffs old vs new snapshots
5. the hub broadcasts the resulting event batch to all websocket subscribers

This is a projection/read-model approach, not a separate execution system.

## Snapshot And Websocket Behavior

Current delivery behavior:

- initial page load uses one full snapshot
- websocket connection also starts with one full snapshot
- new subscribers receive the current snapshot at subscription time
- subscribers then receive future event batches through the shared hub
- `GET /api/agent-arena/snapshot` serves the current arena snapshot as a read-only API

## Arena Event Types

Current diff output includes:

- `agent_joined`
- `agent_removed`
- `agent_state_changed`
- `agent_pnl_realized`
- `agent_died`
- `agent_revived`

These are produced by comparing the previous in-memory snapshot with the newly rebuilt snapshot.

## Status Inference

Arena status is inferred from persisted business data rather than stored as a dedicated domain field.

Current rules:

- stale synced agents become `dead`
- active agents are normally `running`
- agents with recent fills temporarily become `phone`
- everything else falls back to `idle`

## Reconciliation Triggers

Arena reconciliation currently happens after:

- server startup
- sync round completion
- user assignment and claim flows
- admin delete, revoke, and reassign flows that affect assigned agents

## Delivered Interaction Scope

The arena backend is delivered as an additive capability on top of the existing service:

- formal arena APIs are exposed under the public API group
- websocket fan-out is handled by the shared in-process hub
- arena state is derived from existing persisted data
- important business state changes are reflected into the arena through explicit reconciliation triggers

## Local Validation Support

For local validation only, the following helper assets were added:

- [config.arena-dev.json](config/config.arena-dev.json)
- `data/arena-dev.db`

They provide a local dataset for snapshot and websocket verification without changing the normal production startup path.
