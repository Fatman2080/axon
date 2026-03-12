# Agent Arena Frontend Changes

## Goal

This document summarizes the frontend changes made for the pixel-based `agent-arena` feature.

The frontend work had two parallel goals:

- keep an isolated lab page for mock-driven stress testing
- add a formal arena page that connects to the backend snapshot + websocket service

The implementation keeps React focused on page shell and controls, while PixiJS owns real-time rendering.

## Main Frontend Changes

### 1. Added two arena routes

Defined in [App.tsx](frontend-www/src/App.tsx):

- `/lab/agent-arena`
- `/agent-arena`

Both routes are lazy-loaded with `Suspense` fallbacks so the arena bundle stays out of the initial application shell path.

### 2. Kept lab mode separate from formal mode

Implemented as two independent pages:

- [AgentArenaLab.tsx](frontend-www/src/features/agent-arena/pages/AgentArenaLab.tsx)
- [AgentArena.tsx](frontend-www/src/features/agent-arena/pages/AgentArena.tsx)

Separation of responsibilities:

- `/lab/agent-arena` remains mock-driven and is used for stress testing
- `/agent-arena` uses formal backend snapshot + websocket data

Both pages reuse the same canvas and Pixi engine.

### 3. Introduced a shared canvas boundary

Implemented in [AgentArenaCanvas.tsx](frontend-www/src/features/agent-arena/components/AgentArenaCanvas.tsx).

This component is the React-to-Pixi boundary.

Responsibilities:

- create and dispose `ArenaEngine`
- inject either mock feed or live feed
- subscribe the engine to normalized arena events
- forward selected-agent state back to React
- forward runtime performance stats back to React
- expose zoom controls for the scene viewport

Important design choice:

- React does not render one component per agent
- the canvas mounts a single Pixi engine instance
- feeds push normalized arena events into that engine

This keeps React render pressure low even when the arena contains many agents.

### 4. Added normalized feed abstraction

Defined in [types.ts](frontend-www/src/features/agent-arena/types.ts):

- `ArenaAgentEntity`
- `ArenaEvent`
- `ArenaFeed`
- `ArenaScenarioPreset`
- `ArenaPerfStats`

This lets mock and live data share the same renderer and page-level interaction model.

## Mock Feed

Implemented in [createMockArenaFeed.ts](frontend-www/src/features/agent-arena/mock/createMockArenaFeed.ts).

Behavior:

- generates synthetic agents
- emits `agent_snapshot_reset` on boot and preset change
- simulates join, remove, state, pnl, death, and revive events
- supports scenario presets:
  - `steady`
  - `pnl-storm`
  - `death-wave`

The lab page still uses this feed and remains the rendering stress-test environment.

## Live Feed

Implemented in [createLiveArenaFeed.ts](frontend-www/src/features/agent-arena/live/createLiveArenaFeed.ts).

Behavior:

- connects directly to `/api/agent-arena/ws`
- consumes initial backend snapshot from websocket
- maps backend messages into normalized `ArenaEvent`
- reconnects automatically on disconnect
- replays the latest snapshot to new local subscribers

Current runtime behavior:

- the live page initializes from the websocket snapshot
- subsequent state updates arrive through websocket event batches

## Backend API Types

Formal backend arena payloads were added in [types/index.ts](frontend-www/src/types/index.ts):

- `AgentArenaSnapshotAgent`
- `AgentArenaSnapshotResponse`
- `AgentArenaStreamEvent`
- `AgentArenaStreamMessage`

The snapshot REST client entry is exposed in [api.ts](frontend-www/src/services/api.ts).

Current usage:

- the formal page mainly relies on websocket snapshot + events
- the REST snapshot client remains available for fallback or explicit fetch use cases

## PixiJS Renderer

Implemented in [ArenaEngine.ts](frontend-www/src/features/agent-arena/engine/ArenaEngine.ts).

Main responsibilities:

- initialize the Pixi application
- load avatar spritesheets and office scene assets
- build static background and foreground layers
- manage agent view pooling
- manage floating text pooling
- manage visual ring/effect pooling
- maintain the in-memory entity map
- animate transitions and camera interaction
- compute runtime performance stats

### Rendering Strategy

The scene is layered as:

- background layer
- agent layer
- foreground occlusion layer
- effect layer

This gives the office visual depth without pushing every scene object through React.

### Current Agent Visual Behavior

The renderer supports:

- spawn pulse
- running / idle / phone animation swaps
- pnl floating text
- death marker state
- revive transition
- selection highlight

### Performance Strategy

The renderer uses:

- one `Map` for active entities
- object pooling for agent views
- object pooling for floating texts
- object pooling for ring effects
- queued event application
- minimal React updates through callback-based stats and selection reporting

The engine also supports:

- zoom in / zoom out / reset camera
- wheel zoom
- drag pan
- bounded camera movement

## Office Scene Integration

The office scene now comes from the designer-provided final scene file:

- [office-design.scene.json](frontend-www/src/features/agent-arena/assets/office/office-design.scene.json)

Scene loading and normalization are implemented in:

- [officeScene.ts](frontend-www/src/features/agent-arena/assets/officeScene.ts)

Responsibilities:

- fetch the scene JSON asset
- read image layers and logic layer
- split background and foreground occlusion layers
- detect the spawn marker
- derive agent slot cells from the logic grid
- sample visible floor cells from embedded image layers

Important result:

- agent placement is constrained to valid scene slots inside the visible office footprint
- empty black space outside the office is filtered out

## Avatar Asset Integration

The avatar manifest is defined in:

- [manifest.ts](frontend-www/src/features/agent-arena/assets/manifest.ts)

Current variants:

- `adam`
- `alex`
- `amelia`
- `bob`

Sprite assets live under:

- [characters](frontend-www/src/features/agent-arena/assets/characters)

Stable avatar mapping rules are implemented in:

- [avatarRules.ts](frontend-www/src/features/agent-arena/avatarRules.ts)

Rules:

- use a valid preferred variant if one is explicitly provided
- otherwise hash `agentId` into a stable variant

This guarantees consistent character identity across reconnects and reloads.

## Delivered Interaction Scope

The arena frontend is delivered as two coordinated surfaces:

- `/lab/agent-arena` for isolated renderer and scenario validation
- `/agent-arena` for formal backend-integrated viewing

Additional delivered characteristics:

- PixiJS is isolated behind a shared canvas boundary
- the formal page is websocket-driven for initial snapshot and incremental updates
- the office scene uses the final designer-provided scene asset
- local development supports websocket proxying through [vite.config.ts](frontend-www/vite.config.ts)

## Local Validation Support

The frontend now has a clear local validation split:

- `/lab/agent-arena` for renderer and throughput testing with mock data
- `/agent-arena` for backend-integrated websocket validation

For local backend-integrated testing, the paired backend helpers are:

- [config.arena-dev.json](config/config.arena-dev.json)
- `data/arena-dev.db`

These are not part of the production UI flow and exist only to make local validation practical.
