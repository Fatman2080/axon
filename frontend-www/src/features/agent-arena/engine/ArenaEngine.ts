import {
  Application,
  Assets,
  Container,
  Graphics,
  Sprite,
  Text,
  Texture,
} from 'pixi.js';
import { Rectangle } from 'pixi.js';
import { arenaAssetManifest, avatarSheetMeta } from '../assets/manifest';
import { loadOfficeScene, type OfficeSceneCell, type OfficeSceneData } from '../assets/officeScene';
import type { ArenaAgentEntity, ArenaEvent, ArenaPerfStats } from '../types';

type AnimationKind = 'idle' | 'run' | 'phone';

type LoadedAssets = {
  avatars: Record<string, Record<AnimationKind, Texture>>;
  officeScene: OfficeSceneData;
  backgroundLayers: LoadedSceneLayer[];
  foregroundLayers: LoadedSceneLayer[];
};

type LoadedSceneLayer = {
  id: string;
  name: string;
  texture: Texture;
  x: number;
  y: number;
  width: number;
  height: number;
};

type AgentView = {
  container: Container;
  sprite: Sprite;
  shadow: Graphics;
  selection: Graphics;
  deathMarker: Graphics;
  agentId: string | null;
  entity: ArenaAgentEntity | null;
  targetX: number;
  targetY: number;
  velocityX: number;
  velocityY: number;
  animationKind: AnimationKind;
  animationFrame: number;
  animationElapsed: number;
  animBoostUntil: number;
  spawnPulseUntil: number;
  deathUntil: number;
};

type FloatingText = {
  text: Text;
  active: boolean;
  agentId: string | null;
  life: number;
  duration: number;
  velocityX: number;
  velocityY: number;
  kind: 'profit' | 'loss';
};

type RingEffect = {
  graphic: Graphics;
  active: boolean;
  life: number;
  duration: number;
};

const OFFICE_SCENE_SCALE = 4;
const MIN_WORLD_WIDTH = 920;
const MIN_WORLD_HEIGHT = 720;
const WORLD_MARGIN_X = 28;
const WORLD_MARGIN_Y = 28;
const INITIAL_CAMERA_ZOOM = 1.35;
const MIN_CAMERA_ZOOM = 0.9;
const MAX_CAMERA_ZOOM = 3.2;
const SLOT_CLUSTER_OFFSETS = [
  { x: 0, y: 0 },
  { x: 8, y: 1 },
  { x: -8, y: 1 },
  { x: 0, y: 8 },
  { x: 10, y: 9 },
  { x: -10, y: 9 },
] as const;

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value));
}

function hashString(input: string) {
  let value = 0;
  for (let index = 0; index < input.length; index += 1) {
    value = (value * 31 + input.charCodeAt(index)) >>> 0;
  }
  return value;
}

function createAnimationFrames(base: Texture, frameWidth: number, frameHeight: number, frameCount: number, row = 0) {
  const frames: Texture[] = [];
  for (let index = 0; index < frameCount; index += 1) {
    frames.push(
      new Texture({
        source: base.source,
        frame: new Rectangle(index * frameWidth, row * frameHeight, frameWidth, frameHeight),
      }),
    );
  }
  return frames;
}

async function loadArenaAssets(): Promise<LoadedAssets> {
  const [officeScene, avatarEntries] = await Promise.all([
    loadOfficeScene(),
    Promise.all(
    Object.entries(arenaAssetManifest.avatars).map(async ([variant, urls]) => {
      const idle = (await Assets.load(urls.idle)) as Texture;
      const run = (await Assets.load(urls.run)) as Texture;
      const phone = (await Assets.load(urls.phone)) as Texture;
      return [variant, { idle, run, phone }] as const;
    }),
    ),
  ]);

  const backgroundLayers = await Promise.all(
    officeScene.backgroundLayers.map(async (layer) => ({
      id: layer.id,
      name: layer.name,
      texture: (await Assets.load(layer.imageSource)) as Texture,
      x: layer.x,
      y: layer.y,
      width: layer.width,
      height: layer.height,
    })),
  );

  const foregroundLayers = await Promise.all(
    officeScene.foregroundLayers.map(async (layer) => ({
      id: layer.id,
      name: layer.name,
      texture: (await Assets.load(layer.imageSource)) as Texture,
      x: layer.x,
      y: layer.y,
      width: layer.width,
      height: layer.height,
    })),
  );

  return {
    avatars: Object.fromEntries(avatarEntries),
    officeScene,
    backgroundLayers,
    foregroundLayers,
  };
}

export class ArenaEngine {
  private app: Application | null = null;

  private host: HTMLElement | null = null;

  private canvasElement: HTMLCanvasElement | null = null;

  private world = new Container();

  private backgroundLayer = new Container();

  private agentLayer = new Container();

  private foregroundLayer = new Container();

  private effectLayer = new Container();

  private assets: LoadedAssets | null = null;

  private worldWidth = MIN_WORLD_WIDTH;

  private worldHeight = MIN_WORLD_HEIGHT;

  private officeOrigin = { x: 0, y: 0 };

  private slotPositions: Array<{ x: number; y: number }> = [];

  private spawnPoint: { x: number; y: number } | null = null;

  private animationTextures = new Map<string, Record<AnimationKind, Texture[]>>();

  private entities = new Map<string, ArenaAgentEntity>();

  private agentViews = new Map<string, AgentView>();

  private agentPool: AgentView[] = [];

  private floatingTextPool: FloatingText[] = [];

  private ringPool: RingEffect[] = [];

  private pendingEvents: ArenaEvent[] = [];

  private selectedAgentId: string | null = null;

  private lastFrameAt = performance.now();

  private fps = 60;

  private eventsThisSecond = 0;

  private eventsPerSecond = 0;

  private secondWindowStartedAt = performance.now();

  private lastStatsAt = 0;

  private fitScale = 1;

  private cameraZoom = INITIAL_CAMERA_ZOOM;

  private cameraPanX = 0;

  private cameraPanY = 0;

  private isDragging = false;

  private dragPointerId: number | null = null;

  private dragStartClientX = 0;

  private dragStartClientY = 0;

  private dragStartPanX = 0;

  private dragStartPanY = 0;

  private readonly handleResize = () => {
    this.layoutWorld();
  };

  private readonly handleWheel = (event: WheelEvent) => {
    event.preventDefault();
    const direction = event.deltaY > 0 ? -1 : 1;
    const factor = direction > 0 ? 1.12 : 1 / 1.12;
    this.zoomTo(this.cameraZoom * factor);
  };

  private readonly handlePointerDown = (event: PointerEvent) => {
    if (event.button !== 0) return;
    this.isDragging = true;
    this.dragPointerId = event.pointerId;
    this.dragStartClientX = event.clientX;
    this.dragStartClientY = event.clientY;
    this.dragStartPanX = this.cameraPanX;
    this.dragStartPanY = this.cameraPanY;
    this.canvasElement?.setPointerCapture?.(event.pointerId);
    if (this.canvasElement) {
      this.canvasElement.style.cursor = 'grabbing';
    }
  };

  private readonly handlePointerMove = (event: PointerEvent) => {
    if (!this.isDragging || event.pointerId !== this.dragPointerId) return;
    this.cameraPanX = this.dragStartPanX + (event.clientX - this.dragStartClientX);
    this.cameraPanY = this.dragStartPanY + (event.clientY - this.dragStartClientY);
    this.layoutWorld();
  };

  private readonly handlePointerUp = (event: PointerEvent) => {
    if (event.pointerId !== this.dragPointerId) return;
    this.isDragging = false;
    this.dragPointerId = null;
    this.canvasElement?.releasePointerCapture?.(event.pointerId);
    if (this.canvasElement) {
      this.canvasElement.style.cursor = 'grab';
    }
  };

  constructor(
    private readonly callbacks: {
      onSelectAgent?: (agent: ArenaAgentEntity | null) => void;
      onStats?: (stats: ArenaPerfStats) => void;
    } = {},
  ) {}

  async init(host: HTMLElement) {
    this.host = host;
    this.assets = await loadArenaAssets();
    this.buildAnimationTextures();
    this.configureSceneLayout();

    const app = new Application();
    await app.init({
      antialias: false,
      backgroundAlpha: 0,
      autoDensity: true,
      resizeTo: host,
      resolution: Math.min(window.devicePixelRatio || 1, 2),
      powerPreference: 'high-performance',
    });

    this.app = app;
    this.canvasElement = app.canvas as HTMLCanvasElement;
    if (this.canvasElement.parentElement !== host) {
      host.appendChild(this.canvasElement);
    }
    this.canvasElement.style.cursor = 'grab';
    this.canvasElement.style.touchAction = 'none';
    host.addEventListener('wheel', this.handleWheel, { passive: false });
    this.canvasElement.addEventListener('pointerdown', this.handlePointerDown);
    this.canvasElement.addEventListener('pointermove', this.handlePointerMove);
    this.canvasElement.addEventListener('pointerup', this.handlePointerUp);
    this.canvasElement.addEventListener('pointercancel', this.handlePointerUp);

    this.world.sortableChildren = true;
    this.backgroundLayer.zIndex = 0;
    this.agentLayer.zIndex = 10;
    this.foregroundLayer.zIndex = 15;
    this.effectLayer.zIndex = 20;
    this.world.addChild(this.backgroundLayer, this.agentLayer, this.foregroundLayer, this.effectLayer);
    app.stage.addChild(this.world);

    this.buildStaticBackground();
    this.bootstrapPools();
    this.layoutWorld();

    app.ticker.add(this.onTick);
    window.addEventListener('resize', this.handleResize);
  }

  dispose() {
    window.removeEventListener('resize', this.handleResize);
    this.host?.removeEventListener('wheel', this.handleWheel);
    this.canvasElement?.removeEventListener('pointerdown', this.handlePointerDown);
    this.canvasElement?.removeEventListener('pointermove', this.handlePointerMove);
    this.canvasElement?.removeEventListener('pointerup', this.handlePointerUp);
    this.canvasElement?.removeEventListener('pointercancel', this.handlePointerUp);
    if (this.app) {
      this.app.ticker.remove(this.onTick);
      this.app.destroy(true);
      this.app = null;
    }
    if (this.host && this.canvasElement && this.host.contains(this.canvasElement)) {
      this.host.removeChild(this.canvasElement);
    }
    this.canvasElement = null;
    this.agentViews.clear();
    this.agentPool = [];
    this.entities.clear();
    this.pendingEvents = [];
    this.floatingTextPool = [];
    this.ringPool = [];
  }

  enqueue(events: ArenaEvent[]) {
    this.pendingEvents.push(...events);
  }

  getQueuedEvents() {
    return this.pendingEvents.length;
  }

  zoomIn() {
    this.zoomTo(this.cameraZoom * 1.15);
  }

  zoomOut() {
    this.zoomTo(this.cameraZoom / 1.15);
  }

  resetCamera() {
    this.cameraZoom = INITIAL_CAMERA_ZOOM;
    this.cameraPanX = 0;
    this.cameraPanY = 0;
    this.layoutWorld();
  }

  private zoomTo(nextZoom: number) {
    this.cameraZoom = clamp(nextZoom, MIN_CAMERA_ZOOM, MAX_CAMERA_ZOOM);
    this.layoutWorld();
  }

  private buildAnimationTextures() {
    if (!this.assets) return;
    Object.entries(this.assets.avatars).forEach(([variant, textures]) => {
      const idleMeta = avatarSheetMeta.idle;
      const runMeta = avatarSheetMeta.run;
      const phoneMeta = avatarSheetMeta.phone;
      this.validateAvatarSheet(`${variant}:idle`, textures.idle, idleMeta.frameWidth, idleMeta.frameHeight, idleMeta.frameCount);
      this.validateAvatarSheet(`${variant}:run`, textures.run, runMeta.frameWidth, runMeta.frameHeight, runMeta.frameCount);
      this.validateAvatarSheet(`${variant}:phone`, textures.phone, phoneMeta.frameWidth, phoneMeta.frameHeight, phoneMeta.frameCount);

      this.animationTextures.set(variant, {
        idle: createAnimationFrames(textures.idle, idleMeta.frameWidth, idleMeta.frameHeight, idleMeta.frameCount),
        run: createAnimationFrames(textures.run, runMeta.frameWidth, runMeta.frameHeight, runMeta.frameCount),
        phone: createAnimationFrames(textures.phone, phoneMeta.frameWidth, phoneMeta.frameHeight, phoneMeta.frameCount),
      });
    });
  }

  private validateAvatarSheet(label: string, texture: Texture, frameWidth: number, frameHeight: number, frameCount: number) {
    const width = texture.width;
    const height = texture.height;
    const expectedWidth = frameWidth * frameCount;
    if (width !== expectedWidth || height !== frameHeight) {
      console.warn(`[agent-arena] avatar sheet mismatch for ${label}`, {
        width,
        height,
        expectedWidth,
        expectedHeight: frameHeight,
        frameWidth,
        frameHeight,
        frameCount,
      });
    }
  }

  private bootstrapPools() {
    for (let index = 0; index < 128; index += 1) {
      const text = new Text({
        text: '',
        style: {
          fontFamily: 'JetBrains Mono, monospace',
          fontSize: 16,
          fontWeight: '700',
          stroke: { color: 0x000000, width: 4 },
        },
      });
      text.anchor.set(0.5);
      text.visible = false;
      this.effectLayer.addChild(text);
      this.floatingTextPool.push({
        text,
        active: false,
        agentId: null,
        life: 0,
        duration: 0,
        velocityX: 0,
        velocityY: 0,
        kind: 'profit',
      });
    }

    for (let index = 0; index < 32; index += 1) {
      const graphic = new Graphics();
      graphic.visible = false;
      this.effectLayer.addChild(graphic);
      this.ringPool.push({
        graphic,
        active: false,
        life: 0,
        duration: 0,
      });
    }
  }

  private configureSceneLayout() {
    if (!this.assets) return;

    const { officeScene } = this.assets;
    const scaledWidth = officeScene.bounds.width * OFFICE_SCENE_SCALE;
    const scaledHeight = officeScene.bounds.height * OFFICE_SCENE_SCALE;

    this.worldWidth = Math.max(MIN_WORLD_WIDTH, Math.ceil(scaledWidth + WORLD_MARGIN_X * 2));
    this.worldHeight = Math.max(MIN_WORLD_HEIGHT, Math.ceil(scaledHeight + WORLD_MARGIN_Y * 2));
    this.officeOrigin = {
      x: Math.round((this.worldWidth - scaledWidth) / 2),
      y: Math.round((this.worldHeight - scaledHeight) / 2),
    };
    this.slotPositions = officeScene.slotCells.map((cell) => this.cellToScenePoint(cell));
    this.spawnPoint = officeScene.spawnCell ? this.cellToScenePoint(officeScene.spawnCell) : null;
  }

  private cellToScenePoint(cell: OfficeSceneCell) {
    if (!this.assets) {
      return { x: 0, y: 0 };
    }

    const tileSpan = this.assets.officeScene.tileSize * OFFICE_SCENE_SCALE;
    return {
      x: this.officeOrigin.x + (cell.col + 0.5) * tileSpan,
      y: this.officeOrigin.y + (cell.row + 0.88) * tileSpan,
    };
  }

  private resolveSlotPosition(slotIndex: number, agentId: string) {
    if (this.slotPositions.length === 0) {
      return this.spawnPoint ?? { x: this.worldWidth / 2, y: this.worldHeight / 2 };
    }

    const normalizedSlot = Math.max(0, slotIndex);
    const base = this.slotPositions[normalizedSlot % this.slotPositions.length];
    const clusterIndex = Math.floor(normalizedSlot / this.slotPositions.length);
    const clusterOffset = SLOT_CLUSTER_OFFSETS[clusterIndex % SLOT_CLUSTER_OFFSETS.length];
    const jitterSeed = hashString(agentId);

    return {
      x: base.x + clusterOffset.x + ((jitterSeed % 5) - 2),
      y: base.y + clusterOffset.y + (((jitterSeed >> 4) % 5) - 2),
    };
  }

  private buildStaticBackground() {
    if (!this.assets) return;
    this.backgroundLayer.removeChildren();
    this.foregroundLayer.removeChildren();

    const background = new Graphics();
    background.beginFill(0x090b11);
    background.drawRect(0, 0, this.worldWidth, this.worldHeight);
    background.endFill();
    this.backgroundLayer.addChild(background);

    this.assets.backgroundLayers.forEach((layer) => {
      const sprite = new Sprite(layer.texture);
      sprite.x = this.officeOrigin.x + layer.x * OFFICE_SCENE_SCALE;
      sprite.y = this.officeOrigin.y + layer.y * OFFICE_SCENE_SCALE;
      sprite.scale.set(OFFICE_SCENE_SCALE);
      sprite.roundPixels = true;
      this.backgroundLayer.addChild(sprite);
    });

    this.assets.foregroundLayers.forEach((layer) => {
      const sprite = new Sprite(layer.texture);
      sprite.x = this.officeOrigin.x + layer.x * OFFICE_SCENE_SCALE;
      sprite.y = this.officeOrigin.y + layer.y * OFFICE_SCENE_SCALE;
      sprite.scale.set(OFFICE_SCENE_SCALE);
      sprite.roundPixels = true;
      this.foregroundLayer.addChild(sprite);
    });
  }

  private createAgentView(): AgentView {
    const container = new Container();
    container.sortableChildren = true;
    container.eventMode = 'static';
    container.cursor = 'pointer';

    const shadow = new Graphics();
    shadow.beginFill(0x000000, 0.28);
    shadow.drawEllipse(0, 9, 10, 4);
    shadow.endFill();
    shadow.zIndex = 0;

    const sprite = new Sprite(Texture.WHITE);
    sprite.anchor.set(0.5, 0.75);
    sprite.scale.set(2);
    sprite.tint = 0xffffff;
    sprite.zIndex = 5;

    const selection = new Graphics();
    selection.visible = false;
    selection.zIndex = 2;

    const deathMarker = new Graphics();
    deathMarker.visible = false;
    deathMarker.zIndex = 8;

    container.addChild(selection, shadow, sprite, deathMarker);
    this.agentLayer.addChild(container);

    const view: AgentView = {
      container,
      sprite,
      shadow,
      selection,
      deathMarker,
      agentId: null,
      entity: null,
      targetX: 0,
      targetY: 0,
      velocityX: 0,
      velocityY: 0,
      animationKind: 'idle',
      animationFrame: 0,
      animationElapsed: 0,
      animBoostUntil: 0,
      spawnPulseUntil: 0,
      deathUntil: 0,
    };

    container.on('pointertap', () => {
      if (!view.entity) return;
      this.selectAgent(view.entity.id);
    });

    return view;
  }

  private acquireAgentView() {
    const pooled = this.agentPool.pop();
    if (pooled) {
      pooled.container.visible = true;
      pooled.container.alpha = 1;
      pooled.container.rotation = 0;
      pooled.container.scale.set(1);
      pooled.shadow.visible = true;
      pooled.sprite.visible = true;
      pooled.selection.visible = false;
      pooled.deathMarker.visible = false;
      return pooled;
    }
    return this.createAgentView();
  }

  private releaseAgentView(agentId: string) {
    const view = this.agentViews.get(agentId);
    if (!view) return;
    view.agentId = null;
    view.entity = null;
    view.container.visible = false;
    view.container.alpha = 0;
    this.agentViews.delete(agentId);
    this.agentPool.push(view);
  }

  private selectAgent(agentId: string | null) {
    this.entities.forEach((entity) => {
      entity.selected = entity.id === agentId;
    });
    this.selectedAgentId = agentId;
    this.agentViews.forEach((view) => {
      const selected = !!agentId && view.agentId === agentId;
      view.selection.clear();
      if (!selected) {
        view.selection.visible = false;
        return;
      }
      view.selection.visible = true;
      view.selection.lineStyle(2, 0x00ff41, 0.95);
      view.selection.drawCircle(0, -4, 18);
    });

    const entity = agentId ? this.entities.get(agentId) || null : null;
    this.callbacks.onSelectAgent?.(entity);
  }

  private onTick = () => {
    if (!this.app) return;
    const now = performance.now();
    const deltaMs = clamp(now - this.lastFrameAt, 8, 48);
    this.lastFrameAt = now;
    this.fps = 1000 / deltaMs;

    if (now - this.secondWindowStartedAt >= 1000) {
      this.eventsPerSecond = this.eventsThisSecond;
      this.eventsThisSecond = 0;
      this.secondWindowStartedAt = now;
    }

    this.flushEvents();
    this.updateAgents(deltaMs, now);
    this.updateFloatingTexts(deltaMs);
    this.updateRings(deltaMs);

    if (now - this.lastStatsAt >= 250) {
      this.lastStatsAt = now;
      const liveAgents = Array.from(this.entities.values()).filter((entity) => !entity.isDead).length;
      const deadAgents = this.entities.size - liveAgents;
      this.callbacks.onStats?.({
        fps: Number(this.fps.toFixed(1)),
        liveAgents,
        deadAgents,
        renderedAgents: this.agentViews.size,
        pooledAgents: this.agentPool.length,
        pooledFloaters: this.floatingTextPool.filter((item) => !item.active).length,
        activeFloaters: this.floatingTextPool.filter((item) => item.active).length,
        queuedEvents: this.pendingEvents.length,
        eventsPerSecond: this.eventsPerSecond,
      });
    }
  };

  private flushEvents() {
    if (this.pendingEvents.length === 0) return;
    const batch = this.pendingEvents.splice(0, this.pendingEvents.length);
    this.eventsThisSecond += batch.length;
    batch.forEach((event) => this.applyEvent(event));
  }

  private applyEvent(event: ArenaEvent) {
    if (event.type === 'agent_snapshot_reset') {
      this.entities.clear();
      Array.from(this.agentViews.keys()).forEach((agentId) => this.releaseAgentView(agentId));
      event.agents.forEach((agent) => {
        this.entities.set(agent.id, { ...agent });
        this.materializeAgent(agent, 'joined');
      });
      this.selectAgent(null);
      return;
    }

    if (event.type === 'agent_joined') {
      this.entities.set(event.agent.id, { ...event.agent });
      this.materializeAgent(event.agent, 'joined');
      return;
    }

    if (event.type === 'agent_removed') {
      this.entities.delete(event.agentId);
      if (this.selectedAgentId === event.agentId) {
        this.selectAgent(null);
      }
      this.releaseAgentView(event.agentId);
      return;
    }

    const entity = this.entities.get(event.agentId);
    if (!entity) return;

    if (event.type === 'agent_state_changed') {
      entity.status = event.status;
      entity.isDead = false;
      entity.updatedAt = event.emittedAt;
      entity.lastEvent = `state -> ${event.status}`;
      this.syncView(entity, true);
      return;
    }

    if (event.type === 'agent_pnl_realized') {
      entity.totalPnl += event.pnlDelta;
      entity.lastRealizedPnl = event.pnlDelta;
      entity.updatedAt = event.emittedAt;
      entity.lastEvent = `${event.pnlDelta >= 0 ? '+' : ''}${event.pnlDelta.toFixed(2)} realized`;
      entity.status = event.pnlDelta >= 0 ? 'running' : entity.status === 'phone' ? 'phone' : 'running';
      this.spawnFloatingText(entity.id, event.pnlDelta);
      this.syncView(entity, true);
      return;
    }

    if (event.type === 'agent_died') {
      entity.isDead = true;
      entity.status = 'dead';
      entity.updatedAt = event.emittedAt;
      entity.lastEvent = 'melted';
      this.syncView(entity, true);
      return;
    }

    if (event.type === 'agent_revived') {
      entity.isDead = false;
      entity.status = 'idle';
      entity.updatedAt = event.emittedAt;
      entity.lastEvent = 'revived';
      this.syncView(entity, true);
    }
  }

  private materializeAgent(entity: ArenaAgentEntity, withSpawnPulse = 'joined') {
    const view = this.acquireAgentView();
    const position = this.resolveSlotPosition(entity.slotIndex, entity.id);
    const spawnPoint = this.spawnPoint ?? position;
    view.agentId = entity.id;
    view.entity = entity;
    view.targetX = position.x;
    view.targetY = position.y;
    view.container.x = spawnPoint.x;
    view.container.y = spawnPoint.y;
    view.container.alpha = 0;
    view.container.scale.set(0.25);
    view.animationKind = entity.status === 'phone' ? 'phone' : entity.status === 'running' ? 'run' : 'idle';
    view.animationFrame = 0;
    view.animationElapsed = 0;
    view.animBoostUntil = performance.now() + 5000;
    view.spawnPulseUntil = performance.now() + 550;
    view.deathUntil = 0;
    this.updateViewTexture(view, entity);
    this.paintDeathMarker(view, entity.isDead);
    this.agentViews.set(entity.id, view);
    if (withSpawnPulse) {
      this.spawnRing(spawnPoint.x, spawnPoint.y - 6, 0x00ff41);
    }
  }

  private syncView(entity: ArenaAgentEntity, activeBoost = false) {
    let view = this.agentViews.get(entity.id);
    if (!view) {
      this.materializeAgent(entity);
      view = this.agentViews.get(entity.id) || null;
    }
    if (!view) return;
    view.entity = entity;
    const position = this.resolveSlotPosition(entity.slotIndex, entity.id);
    view.targetX = position.x;
    view.targetY = position.y;
    if (activeBoost) {
      view.animBoostUntil = performance.now() + 4200;
    }
    if (entity.isDead) {
      view.deathUntil = performance.now() + 650;
    }
    this.updateViewTexture(view, entity);
    this.paintDeathMarker(view, entity.isDead);
    if (this.selectedAgentId === entity.id) {
      this.callbacks.onSelectAgent?.(entity);
    }
  }

  private updateViewTexture(view: AgentView, entity: ArenaAgentEntity) {
    const textures = this.animationTextures.get(entity.avatarVariant);
    if (!textures) return;
    const animationKind: AnimationKind =
      entity.status === 'phone' ? 'phone' : entity.status === 'running' ? 'run' : 'idle';
    view.animationKind = animationKind;
    const nextTextures = textures[animationKind];
    view.sprite.texture = nextTextures[0] || Texture.WHITE;
    view.sprite.tint = entity.isDead ? 0xff6b6b : 0xffffff;
  }

  private updateAgents(deltaMs: number, now: number) {
    this.agentViews.forEach((view) => {
      if (!view.entity) return;
      const entity = view.entity;
      const dx = view.targetX - view.container.x;
      const dy = view.targetY - view.container.y;
      view.container.x += dx * 0.12;
      view.container.y += dy * 0.12;

      if (view.spawnPulseUntil > now) {
        const progress = 1 - (view.spawnPulseUntil - now) / 550;
        const scale = 0.25 + progress * 0.75;
        view.container.scale.set(scale);
        view.container.alpha = progress;
      } else if (entity.isDead && view.deathUntil > now) {
        const progress = 1 - (view.deathUntil - now) / 650;
        view.container.alpha = 1 - progress * 0.45;
        view.container.rotation = progress * 0.28;
        view.container.scale.set(1 - progress * 0.15);
      } else {
        view.container.alpha += (1 - view.container.alpha) * 0.18;
        view.container.rotation *= 0.8;
        const wobble = view.agentId ? (hashString(view.agentId) % 6) * 0.01 : 0;
        view.container.scale.set(entity.isDead ? 0.86 : 1 + wobble);
      }

      const shouldAnimate = !entity.isDead && (view.animBoostUntil > now || view.agentId === this.selectedAgentId);
      const textures = this.animationTextures.get(entity.avatarVariant)?.[view.animationKind];
      if (textures && textures.length > 0) {
        if (shouldAnimate) {
          view.animationElapsed += deltaMs;
          if (view.animationElapsed >= 110) {
            view.animationElapsed = 0;
            view.animationFrame = (view.animationFrame + 1) % textures.length;
          }
          view.sprite.texture = textures[view.animationFrame];
        } else {
          view.animationFrame = 0;
          view.sprite.texture = textures[0];
        }
      }

      view.shadow.alpha = entity.isDead ? 0.15 : 0.28;
      view.sprite.y = entity.isDead ? 6 : 0;
    });
  }

  private spawnRing(x: number, y: number, color: number) {
    const available = this.ringPool.find((ring) => !ring.active) || this.ringPool[0];
    if (!available) return;
    available.active = true;
    available.life = 0;
    available.duration = 420;
    available.graphic.visible = true;
    available.graphic.clear();
    available.graphic.lineStyle(2, color, 0.9);
    available.graphic.drawCircle(0, 0, 6);
    available.graphic.x = x;
    available.graphic.y = y;
    available.graphic.alpha = 1;
    available.graphic.scale.set(0.2);
  }

  private updateRings(deltaMs: number) {
    this.ringPool.forEach((ring) => {
      if (!ring.active) return;
      ring.life += deltaMs;
      const progress = clamp(ring.life / ring.duration, 0, 1);
      ring.graphic.alpha = 1 - progress;
      ring.graphic.scale.set(0.2 + progress * 2.1);
      if (progress >= 1) {
        ring.active = false;
        ring.graphic.visible = false;
      }
    });
  }

  private spawnFloatingText(agentId: string, pnlDelta: number) {
    const view = this.agentViews.get(agentId);
    if (!view) return;

    const sameAgent = this.floatingTextPool.find((item) => item.active && item.agentId === agentId);
    const available = sameAgent || this.floatingTextPool.find((item) => !item.active) || this.floatingTextPool[0];
    if (!available) return;

    available.active = true;
    available.agentId = agentId;
    available.life = 0;
    available.duration = pnlDelta >= 0 ? 1200 : 1400;
    available.velocityX = pnlDelta >= 0 ? 0.018 : (Math.random() > 0.5 ? 0.03 : -0.03);
    available.velocityY = pnlDelta >= 0 ? -0.09 : -0.07;
    available.kind = pnlDelta >= 0 ? 'profit' : 'loss';
    available.text.text = `${pnlDelta >= 0 ? '+' : ''}$${Math.abs(pnlDelta).toFixed(2)}`;
    available.text.style.fill = pnlDelta >= 0 ? 0x00ff66 : 0xff5f56;
    available.text.x = view.container.x;
    available.text.y = view.container.y - 22;
    available.text.visible = true;
    available.text.alpha = 1;
    available.text.scale.set(1);
  }

  private updateFloatingTexts(deltaMs: number) {
    this.floatingTextPool.forEach((item) => {
      if (!item.active) return;
      item.life += deltaMs;
      item.text.x += item.velocityX * deltaMs;
      item.text.y += item.velocityY * deltaMs;
      if (item.kind === 'profit') {
        item.text.scale.set(1 + clamp(item.life / item.duration, 0, 1) * 0.28);
      } else {
        const shake = Math.sin(item.life / 36) * 0.6;
        item.text.x += shake;
      }
      item.text.alpha = 1 - clamp(item.life / item.duration, 0, 1);
      if (item.life >= item.duration) {
        item.active = false;
        item.agentId = null;
        item.text.visible = false;
      }
    });
  }

  private paintDeathMarker(view: AgentView, isDead: boolean) {
    view.deathMarker.clear();
    view.deathMarker.visible = isDead;
    if (!isDead) return;
    view.deathMarker.lineStyle(3, 0xff4d4f, 0.95);
    view.deathMarker.moveTo(-9, -16);
    view.deathMarker.lineTo(9, 2);
    view.deathMarker.moveTo(9, -16);
    view.deathMarker.lineTo(-9, 2);
    view.deathMarker.beginFill(0x151515, 0.88);
    view.deathMarker.drawRoundedRect(-9, 6, 18, 7, 3);
    view.deathMarker.endFill();
  }

  private layoutWorld() {
    if (!this.app) return;
    const screenWidth = this.app.screen.width;
    const screenHeight = this.app.screen.height;
    this.fitScale = Math.min(screenWidth / this.worldWidth, screenHeight / this.worldHeight);
    const scale = this.fitScale * this.cameraZoom;
    const scaledWidth = this.worldWidth * scale;
    const scaledHeight = this.worldHeight * scale;
    const centeredX = (screenWidth - scaledWidth) / 2;
    const centeredY = (screenHeight - scaledHeight) / 2;

    if (scaledWidth <= screenWidth) {
      this.cameraPanX = 0;
    } else {
      const minPanX = screenWidth - scaledWidth - centeredX;
      const maxPanX = -centeredX;
      this.cameraPanX = clamp(this.cameraPanX, minPanX, maxPanX);
    }

    if (scaledHeight <= screenHeight) {
      this.cameraPanY = 0;
    } else {
      const minPanY = screenHeight - scaledHeight - centeredY;
      const maxPanY = -centeredY;
      this.cameraPanY = clamp(this.cameraPanY, minPanY, maxPanY);
    }

    this.world.scale.set(scale);
    this.world.x = centeredX + this.cameraPanX;
    this.world.y = centeredY + this.cameraPanY;
  }
}
