import fallback from './fallback.gen.json';
import { TargetOffers } from '../../../wailsjs/go/main/App';
import { main } from '../../../wailsjs/go/models';
import { object, text, integer, choice, array } from '../data/parseValues';
import { freshness } from '../data/parse';
import type { Reading } from '../market/contract';
export type Scope = {
    from: number;
    to: number;
    direction: string;
    window: string;
    team: string;
};
export type Pick = {
    id: string;
    label: string;
    from: number;
    to: number;
};
export type Stretch = {
    id: string;
    label: string;
    countText: string;
    acceptance: string;
    finding: string;
    outcomes: {
        id: string;
        label: string;
        text: string;
        width: number;
    }[];
};
export type OfferWords = {
    heading: string;
    direction: string;
    seasons: string;
    window: string;
    team: string;
    retry: string;
    pending: string;
};
export type Offers = {
    words: OfferWords;
    disclosure: string;
    title: string;
    summary: string;
    countText: string;
    directions: Pick[];
    windows: Pick[];
    ranges: Pick[];
    teams: {
        fid: string;
        name: string;
    }[];
    stretches: Stretch[];
};
export type OfferReading = Reading<Offers>;
function pick(value: unknown, path: string): Pick {
    const p = object(value, path, ['id', 'label', 'from', 'to']);
    return { id: text(p.id, path), label: text(p.label, path),
      from: integer(p.from, path), to: integer(p.to, path) };
}
function stretch(value: unknown, path: string): Stretch {
    const s = object(value, path, ['id', 'label', 'countText', 'acceptance', 'finding', 'outcomes']);
    return {
        id: text(s.id, path), label: text(s.label, path), countText: text(s.countText, path),
        acceptance: text(s.acceptance, path), finding: text(s.finding, path),
        outcomes: array(s.outcomes, path, (value, path) => {
            const o = object(value, path, ['id', 'label', 'text', 'width']);
            if (typeof o.width !== 'number' || !Number.isFinite(o.width) || o.width < 0 || o.width > 100)
                throw new Error(`${path}: invalid bar width`);
            return { id: text(o.id, path), label: text(o.label,
              path), text: text(o.text, path), width: o.width };
        }),
    };
}
export function parseOffers(value: unknown): OfferReading {
    const path = 'Offers';
    const r = object(value, path, ['value', 'provenance',
      'status', 'absenceReason', 'count', 'threshold', 'baseline',
      'cutoff', 'method', 'inputSHA', 'buildStamp', 'howCounted']);
      
    const status = choice(r.status, path, ['ok', 'thin', 'failed', 'not_wired']);
    const body = object(r.value, path, ['words', 'disclosure',
      'title', 'summary', 'countText', 'directions', 'windows',
      'ranges', 'teams', 'stretches']);
    const rawWords = object(body.words, path, ['heading',
      'direction', 'seasons', 'window', 'team', 'retry', 'pending']);
      
    const words: OfferWords = { heading: text(rawWords.heading,
      path), direction: text(rawWords.direction, path), seasons: text(rawWords.seasons,
      path), window: text(rawWords.window, path), team: text(rawWords.team,
      path), retry: text(rawWords.retry, path), pending: text(rawWords.pending,
      path) };
    const parsed: OfferReading = {
        status, absenceReason: r.absenceReason == null ? null : text(r.absenceReason, path),
        count: integer(r.count, path), threshold: integer(r.threshold,
          path), baseline: text(r.baseline, path),
        cutoff: text(r.cutoff, path), method: text(r.method, path), inputSHA: text(r.inputSHA, path),
        buildStamp: text(r.buildStamp, path), howCounted: text(r.howCounted, path),
        value: {
            words, disclosure: text(body.disclosure, path),
              title: text(body.title, path), summary: text(body.summary,
              path), countText: text(body.countText, path),
            directions: array(body.directions, path, pick),
              windows: array(body.windows, path, pick), ranges: array(body.ranges,
              path, pick),
            teams: array(body.teams, path, (value, path) => { const t = object(value,
              path, ['fid', 'name']); return { fid: text(t.fid, path), name: text(t.name,
              path) }; }),
            stretches: array(body.stretches, path, stretch),
        },
    };
    if (status !== 'ok' && !parsed.absenceReason)
        throw new Error('Offers: absence reason required');
    if (status === 'ok' || status === 'thin') {
        const p = object(r.provenance, path, ['source', 'kind', 'freshness']);
        parsed.provenance = { source: text(p.source, path),
          kind: choice(p.kind, path, ['archive']), freshness: freshness(p.freshness,
          path) };
    }
    return parsed;
}
export const initialScope: Scope = { from: 2013, to: 2026, direction: 'all', window: '', team: '' };
export function unavailable(reason: string): OfferReading {
    return {
        status: 'failed', absenceReason: reason, count: 0,
          threshold: 10, baseline: '', cutoff: '', method: '', inputSHA: '',
          buildStamp: '', howCounted: '',
        value: fallback,
    };
}
export async function readOffers(scope: Scope): Promise<OfferReading> {
    try {
        return parseOffers(await TargetOffers(main.RawOfferScope.createFrom(scope)));
    }
    catch (cause) {
        return unavailable(String(cause));
    }
}
