import type { NativeRequest } from '../commands/native';
import type { HostedPort } from '../commissioner/hostedRoom';

// The Home grid as the locked mock (docs/ui/mocks/home-mock-v5.html) packs it: three columns by two
// rows, tiles placed first-fit in layout order. Hosted layouts are account preferences; local mode
// retains its per-league localStorage key.
export type Span = '1x1' | '2x1' | '1x2';
export type TileId = 'due' | 'mu' | 'race' | 'head' | 'news' | 'wire' | 'sc' | 'pwr' | 'plan' | 'picks'
  | 'cap' | 'bids' | 'form';
export type Placed = { id: TileId; span: Span; row: number; col: number; w: number; h: number };
export type Phase = 'in' | 'off';
export type Layout = {
  mode: 'default' | 'custom'; tiles: [TileId, Span][]; over: TileId[]; pins: Partial<Record<TileId, true>>;
};

export const SPANS: Record<TileId, Span[]> = {
  due: ['1x1', '1x2'], mu: ['1x1', '2x1', '1x2'], race: ['1x1', '2x1'], head: ['1x1', '2x1', '1x2'],
  news: ['1x2', '2x1', '1x1'], wire: ['1x1', '2x1', '1x2'], sc: ['1x1', '2x1'], pwr: ['1x1', '1x2'],
  plan: ['1x1', '1x2'], picks: ['1x1'], cap: ['1x1'], bids: ['1x1'], form: ['1x1', '2x1'],
};
export const DEFAULTS: Record<Phase, [TileId, Span][]> = {
  in: [['mu', '1x1'], ['news', '1x2'], ['wire', '1x1'], ['race', '1x1'], ['sc', '1x1']],
  off: [['pwr', '1x1'], ['news', '1x2'], ['wire', '1x1'], ['picks', '1x1'], ['sc', '1x1']],
};

export function pack(tiles: readonly (readonly [TileId, Span])[]): Placed[] | null {
  const g = [[0, 0, 0], [0, 0, 0]];
  const out: Placed[] = [];
  for (const [id, span] of tiles) {
    const [w, h] = span.split('x').map(Number);
    let ok = false;
    for (let r = 0; r <= 2 - h && !ok; r++) for (let c = 0; c <= 3 - w && !ok; c++) {
      let free = true;
      for (let y = r; y < r + h; y++) for (let x = c; x < c + w; x++) if (g[y][x]) free = false;
      if (free) {
        for (let y = r; y < r + h; y++) for (let x = c; x < c + w; x++) g[y][x] = 1;
        out.push({ id, span, row: r, col: c, w, h }); ok = true;
      }
    }
    if (!ok) return null;
  }
  return out;
}

export const defaultLayout = (): Layout => ({ mode: 'default', tiles: [], over: [], pins: {} });
export function current(l: Layout, phase: Phase): [TileId, Span][] {
  return l.mode === 'default' ? DEFAULTS[phase].map(t => [...t] as [TileId, Span]) : l.tiles;
}
function custom(l: Layout, phase: Phase): Layout {
  return l.mode === 'default' ? { ...l, mode: 'custom', tiles: current(l, phase) } : l;
}
/** place puts a tile on the grid at a size. A full grid returns undefined: the owner picks a tile
 *  to bump to overflow instead (bump). */
export function place(l: Layout, phase: Phase, id: TileId, span: Span): Layout | undefined {
  const c = custom(l, phase);
  const i = c.tiles.findIndex(t => t[0] === id);
  const next = c.tiles.filter(t => t[0] !== id);
  next.splice(i < 0 ? next.length : i, 0, [id, span]);
  return pack(next) ? { ...c, tiles: next, over: c.over.filter(x => x !== id) } : undefined;
}
export function bump(l: Layout, phase: Phase, pick: { id: TileId; span: Span }, victim: TileId):
  Layout | undefined {
  const c = custom(l, phase);
  const i = c.tiles.findIndex(t => t[0] === victim);
  const next = c.tiles.filter(t => t[0] !== victim && t[0] !== pick.id);
  next.splice(Math.min(i, next.length), 0, [pick.id, pick.span]);
  if (!pack(next)) return undefined;
  return { ...c, tiles: next, over: [...c.over.filter(x => x !== pick.id), victim] };
}
export function remove(l: Layout, phase: Phase, id: TileId): Layout {
  const c = custom(l, phase);
  return { ...c, tiles: c.tiles.filter(t => t[0] !== id), over: [...c.over, id] };
}
export function swap(l: Layout, phase: Phase, a: TileId, b: TileId): Layout | undefined {
  const c = custom(l, phase);
  const i = c.tiles.findIndex(t => t[0] === a), j = c.tiles.findIndex(t => t[0] === b);
  if (i < 0 || j < 0) return undefined;
  const next = c.tiles.map(t => [...t] as [TileId, Span]);
  [next[i], next[j]] = [next[j], next[i]];
  return pack(next) ? { ...c, tiles: next } : undefined;
}

const ids = Object.keys(SPANS) as TileId[];
const isId = (v: unknown): v is TileId => typeof v === 'string' && (ids as string[]).includes(v);
/** parseLayout accepts only a layout that still packs; anything else is the phase default. */
export function parseLayout(raw: string | null | undefined): Layout {
  try {
    const v = JSON.parse(raw ?? '') as Layout;
    if (v.mode !== 'default' && v.mode !== 'custom') return defaultLayout();
    const tiles = v.tiles.filter(t => isId(t[0]) && SPANS[t[0]].includes(t[1]));
    const over = v.over.filter(isId);
    const pins = Object.fromEntries(Object.keys(v.pins ?? {}).filter(isId).map(k => [k, true]));
    if (!pack(tiles)) return defaultLayout();
    return { mode: v.mode, tiles, over, pins };
  } catch { return defaultLayout(); }
}
export const layoutKey = (league: string, franchise: string) =>
  `thewarroom.home.layout:${league}:${franchise}`;

type LayoutReply = { layout: Layout; revision: number; conflict: boolean };
export type HomeLayoutResult = LayoutReply & { offline: boolean; cached: boolean };
type LayoutStorage = Pick<Storage, 'getItem' | 'setItem'>;

function record(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}
function readLayoutReply(value: unknown): LayoutReply {
  if (!record(value) || Object.keys(value).some(key => !['layout', 'revision', 'conflict'].includes(key))
    || !Number.isSafeInteger(value.revision) || Number(value.revision) < 0
    || ('conflict' in value && typeof value.conflict !== 'boolean') || !record(value.layout))
    throw new Error('The saved Home layout could not be read.');
  const raw = value.layout;
  const parsed = parseLayout(JSON.stringify(raw));
  const pins = record(raw.pins) ? raw.pins : {};
  if (parsed.mode !== raw.mode || !Array.isArray(raw.tiles) || parsed.tiles.length !== raw.tiles.length
    || !Array.isArray(raw.over) || parsed.over.length !== raw.over.length
    || Object.keys(parsed.pins).length !== Object.keys(pins).length)
    throw new Error('The saved Home layout is invalid.');
  return { layout: parsed, revision: Number(value.revision), conflict: value.conflict === true };
}
function saveLayout(storage: LayoutStorage | undefined, key: string, layout: Layout): boolean {
  try {
    if (!storage) return false;
    storage.setItem(key, JSON.stringify(layout));
    return true;
  } catch { return false; }
}

/** Reads the signed-in account's layout first, with the old local key as an offline-only fallback. */
export function createHomeLayoutStore(port: HostedPort, key: string, storage?: LayoutStorage) {
  let revision = 0;
  let currentLayout = defaultLayout();
  let initialized = false;
  let writeSequence = 0;
  let queue: Promise<void> = Promise.resolve();
  let reading: Promise<HomeLayoutResult> | undefined;

  function read(): Promise<HomeLayoutResult> {
    if (reading) return reading;
    const operation = load();
    reading = operation;
    void operation.then(() => { if (reading === operation) reading = undefined; },
      () => { if (reading === operation) reading = undefined; });
    return operation;
  }

  async function load(): Promise<HomeLayoutResult> {
    try {
      const response = await port({ command: 'home.layout.read', mode: 'read', arguments: {} });
      const result = readLayoutReply(response.data);
      revision = result.revision;
      currentLayout = result.layout;
      initialized = true;
      const cached = saveLayout(storage, key, result.layout);
      return { ...result, offline: false, cached };
    } catch {
      let cached = false;
      try {
        const raw = storage?.getItem(key);
        cached = raw !== null && raw !== undefined;
        currentLayout = parseLayout(raw);
      } catch { currentLayout = defaultLayout(); }
      revision = 0;
      initialized = true;
      return { layout: currentLayout, revision, conflict: false, offline: true, cached };
    }
  }

  function write(layout: Layout): Promise<HomeLayoutResult> {
    const sequence = ++writeSequence;
    currentLayout = layout;
    const operation = queue.then(async (): Promise<HomeLayoutResult> => {
      if (!initialized) await read();
      if (sequence !== writeSequence)
        return { layout: currentLayout, revision, conflict: false, offline: false, cached: false };
      try {
        const request = { command: 'home.layout.set', mode: 'execute',
          arguments: { layout, expectedRevision: revision } } satisfies NativeRequest;
        const response = await port(request);
        const result = readLayoutReply(response.data);
        revision = result.revision;
        currentLayout = result.layout;
        const cached = saveLayout(storage, key, result.layout);
        if (result.conflict) writeSequence++;
        return { ...result, offline: false, cached };
      } catch {
        if (sequence === writeSequence) {
          currentLayout = layout;
        }
        const cached = sequence === writeSequence ? saveLayout(storage, key, layout) : false;
        return { layout: currentLayout, revision, conflict: false, offline: true, cached };
      }
    });
    queue = operation.then(() => undefined, () => undefined);
    return operation;
  }
  return { read, write };
}
