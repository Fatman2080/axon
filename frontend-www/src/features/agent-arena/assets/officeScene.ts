import officeDesignSceneUrl from './office/office-design.scene.json?url';

export interface OfficeSceneCell {
  col: number;
  row: number;
}

interface OfficeSceneImageLayer {
  id: string;
  name: string;
  mode: 'image';
  kind: 'scene';
  visible: boolean;
  x: number;
  y: number;
  width: number;
  height: number;
  imageSource: string;
}

interface OfficeSceneLogicLayer {
  id: string;
  name: string;
  mode: 'marker';
  kind: 'logic';
  visible: boolean;
  markers: Array<{
    id: string;
    type: string;
    key: string | null;
    col: number;
    row: number;
  }>;
  grid: number[][];
}

interface OfficeSceneDocument {
  version: number;
  projectName: string;
  scene: {
    id: string;
    name: string;
    cols: number;
    rows: number;
    tileSize: number;
    layers: Array<OfficeSceneImageLayer | OfficeSceneLogicLayer | Record<string, unknown>>;
  };
}

export interface OfficeSceneData {
  projectName: string;
  sceneName: string;
  cols: number;
  rows: number;
  tileSize: number;
  bounds: {
    width: number;
    height: number;
  };
  backgroundLayers: OfficeSceneImageLayer[];
  foregroundLayers: OfficeSceneImageLayer[];
  spawnCell: OfficeSceneCell | null;
  slotCells: OfficeSceneCell[];
}

let officeScenePromise: Promise<OfficeSceneData> | null = null;

async function sampleVisibleCells(layer: OfficeSceneImageLayer, cols: number, rows: number, tileSize: number) {
  if (typeof document === 'undefined') {
    return new Set<string>();
  }

  const image = await new Promise<HTMLImageElement>((resolve, reject) => {
    const img = new Image();
    img.onload = () => resolve(img);
    img.onerror = () => reject(new Error(`Failed to load office scene image layer: ${layer.name}`));
    img.src = layer.imageSource;
  });

  const canvas = document.createElement('canvas');
  canvas.width = image.width;
  canvas.height = image.height;
  const context = canvas.getContext('2d', { willReadFrequently: true });
  if (!context) {
    return new Set<string>();
  }

  context.imageSmoothingEnabled = false;
  context.drawImage(image, 0, 0);
  const visibleCells = new Set<string>();

  for (let row = 0; row < rows; row += 1) {
    for (let col = 0; col < cols; col += 1) {
      const sampleX = col * tileSize + Math.floor(tileSize / 2) - layer.x;
      const sampleY = row * tileSize + Math.floor(tileSize / 2) - layer.y;
      if (sampleX < 0 || sampleY < 0 || sampleX >= image.width || sampleY >= image.height) {
        continue;
      }
      const alpha = context.getImageData(sampleX, sampleY, 1, 1).data[3];
      if (alpha > 0) {
        visibleCells.add(`${col}:${row}`);
      }
    }
  }

  return visibleCells;
}

function isImageLayer(layer: unknown): layer is OfficeSceneImageLayer {
  return Boolean(
    layer &&
      typeof layer === 'object' &&
      (layer as OfficeSceneImageLayer).kind === 'scene' &&
      (layer as OfficeSceneImageLayer).mode === 'image' &&
      typeof (layer as OfficeSceneImageLayer).imageSource === 'string',
  );
}

function isLogicLayer(layer: unknown): layer is OfficeSceneLogicLayer {
  return Boolean(
    layer &&
      typeof layer === 'object' &&
      (layer as OfficeSceneLogicLayer).kind === 'logic' &&
      (layer as OfficeSceneLogicLayer).mode === 'marker' &&
      Array.isArray((layer as OfficeSceneLogicLayer).grid),
  );
}

async function normalizeOfficeScene(document: OfficeSceneDocument): Promise<OfficeSceneData> {
  const visibleImageLayers = document.scene.layers.filter(isImageLayer).filter((layer) => layer.visible !== false);
  const backgroundLayers = visibleImageLayers.filter((layer) => layer.name !== '角色' && layer.name !== '角色前景遮挡');
  const foregroundLayers = visibleImageLayers.filter((layer) => layer.name === '角色前景遮挡');
  const logicLayer = document.scene.layers.find(isLogicLayer) || null;
  const floorLayer =
    backgroundLayers.find((layer) => layer.name === '地板') ||
    backgroundLayers[0] ||
    null;

  const boundsWidth = Math.max(...visibleImageLayers.map((layer) => layer.x + layer.width), document.scene.cols * document.scene.tileSize);
  const boundsHeight = Math.max(...visibleImageLayers.map((layer) => layer.y + layer.height), document.scene.rows * document.scene.tileSize);
  const visibleCells = floorLayer
    ? await sampleVisibleCells(floorLayer, document.scene.cols, document.scene.rows, document.scene.tileSize)
    : new Set<string>();

  const spawnMarker = logicLayer?.markers.find((marker) => marker.type === 'spawn') || null;
  const slotCells: OfficeSceneCell[] = [];

  if (logicLayer) {
    for (let row = 0; row < logicLayer.grid.length; row += 1) {
      const cells = logicLayer.grid[row];
      for (let col = 0; col < cells.length; col += 1) {
        if (cells[col] !== 0) continue;
        if (row === 0 || col === 0 || col === document.scene.cols - 1) continue;
        if (visibleCells.size > 0 && !visibleCells.has(`${col}:${row}`)) continue;
        slotCells.push({ col, row });
      }
    }
  }

  return {
    projectName: document.projectName,
    sceneName: document.scene.name,
    cols: document.scene.cols,
    rows: document.scene.rows,
    tileSize: document.scene.tileSize,
    bounds: {
      width: boundsWidth,
      height: boundsHeight,
    },
    backgroundLayers,
    foregroundLayers,
    spawnCell: spawnMarker ? { col: spawnMarker.col, row: spawnMarker.row } : null,
    slotCells,
  };
}

export function loadOfficeScene() {
  if (!officeScenePromise) {
    officeScenePromise = fetch(officeDesignSceneUrl)
      .then((response) => {
        if (!response.ok) {
          throw new Error(`Failed to load office scene: ${response.status}`);
        }
        return response.json() as Promise<OfficeSceneDocument>;
      })
      .then(normalizeOfficeScene);
  }

  return officeScenePromise;
}
