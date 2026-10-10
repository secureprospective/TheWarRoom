import {
  TargetMarketLists,
  TargetMarketStretches,
  TargetAssetProfile,
  TargetPickClock,
} from '../../../wailsjs/go/main/App';
import type { main } from '../../../wailsjs/go/models';
import { freshness } from '../data/parse';
import { object, text, integer, choice, array } from '../data/parseValues';

import type {
  MarketLine,
  MarketRange,
  MarketSection,
  MarketScope,
  MarketProvider,
  SurfaceReading,
} from './contract';

function finite(value: unknown, path: string): number {
  if (typeof value !== 'number' || !Number.isFinite(value))
    throw new Error(`${path}: expected finite number`);
  return value;
}
function range(value: unknown, path: string): MarketRange | null {
  if (value == null) return null;
  const r = object(value, path, ['value', 'lo', 'hi', 'text']);
  const parsed = {
    value: finite(r.value, path),
    lo: finite(r.lo, path),
    hi: finite(r.hi, path),
    text: text(r.text, path),
  };
  if (parsed.lo > parsed.hi) throw new Error(`${path}: reversed range`);
  return parsed;
}
function line(value: unknown, path: string): MarketLine {
  const r = object(value, path, ['key', 'label', 'detail', 'count', 'range', 'mark']);
  return {
    key: text(r.key, path),
    label: text(r.label, path),
    detail: text(r.detail, path),
    count: integer(r.count, path),
    range: range(r.range, path),
    mark: choice(r.mark, path, ['', 'cheapest', 'dearest']),
  };
}
function section(value: unknown, path: string): MarketSection {
  const r = object(value, path, ['id', 'title', 'note', 'rows']);
  return {
    id: text(r.id, path),
    title: text(r.title, path),
    note: text(r.note, path),
    rows: array(r.rows, path, line),
  };
}
export function parseMarket(value: unknown): SurfaceReading {
  const path = 'Market';
  const r = object(value, path, [
    'value',
    'provenance',
    'status',
    'absenceReason',
    'count',
    'threshold',
    'baseline',
    'cutoff',
    'method',
    'inputSHA',
    'buildStamp',
    'howCounted',
  ]);
  const status = choice(r.status, path, ['ok', 'thin', 'failed', 'not_wired']);
  const parsed: SurfaceReading = {
    status,
    absenceReason: r.absenceReason == null ? null : text(r.absenceReason, path),
    count: integer(r.count, path),
    threshold: integer(r.threshold, path),
    baseline: text(r.baseline, path),
    cutoff: text(r.cutoff, path),
    method: text(r.method, path),
    inputSHA: text(r.inputSHA, path),
    buildStamp: text(r.buildStamp, path),
    howCounted: text(r.howCounted, path),
  };
  if (status !== 'ok' && !parsed.absenceReason) throw new Error('Market: absence reason required');
  if (status === 'ok' || status === 'thin') {
    const source = object(r.provenance, path, ['source', 'kind', 'freshness']);
    parsed.provenance = {
      source: text(source.source, path),
      kind: choice(source.kind, path, ['archive']),
      freshness: freshness(source.freshness, path),
    };
    const body = object(r.value, path, ['summary', 'sections']);
    parsed.value = {
      summary: text(body.summary, path),
      sections: array(body.sections, path, section),
    };
    if (!parsed.cutoff || !parsed.howCounted || !/^[a-f0-9]{64}$/.test(parsed.inputSHA)) {
      throw new Error('Market: archive disclosure missing');
    }
  }
  return parsed;
}
export function unavailable(
  reason: string,
  status: 'failed' | 'not_wired' = 'failed',
): SurfaceReading {
  return {
    status,
    absenceReason: reason,
    count: 0,
    threshold: 80,
    baseline: '',
    cutoff: '',
    method: '',
    inputSHA: '',
    buildStamp: '',
    howCounted: '',
  };
}
async function read(call: () => Promise<unknown>): Promise<SurfaceReading> {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go) {
    return unavailable(
      'Historical Market needs the desktop app; no browser fixtures are substituted',
      'not_wired',
    );
  }
  try {
    return parseMarket(await call());
  } catch (cause) {
    return unavailable(`Market read failed: ${String(cause)}. Retry the held archive reading.`);
  }
}
const raw = (scope: MarketScope): main.RawMarketScope => ({
  ...scope,
  phases: scope.phases ?? [],
  assets: scope.assets ?? [],
});
export const marketProvider: MarketProvider = {
  lists: (scope) => read(() => TargetMarketLists(raw(scope))),
  clock: (scope) => read(() => TargetPickClock(raw(scope))),
  stretches: (scope) => read(() => TargetMarketStretches(raw(scope))),
  profile: (scope, kind) =>
    read(() => TargetAssetProfile({ scope: raw(scope), kind } as main.RawMarketAsset)),
};
