import { array, choice, integer, object, text } from '../data/parseValues';
import { hostedWorkflow } from '../commissioner/api';
import { newKey } from '../commissioner/forms';
import type { WarRoomContext } from './warroomReads';
import { finite } from './hqPlayerData';
import type { WarRoomDeal } from './warroomTradeReads';

// The trade lens is the engine's read of a deal: what each asset is worth to each side's best N
// under that side's own direction. The desk shows it; it works nothing out itself.
export const directions = ['contend', 'retool', 'rebuild'] as const;
export type Direction = typeof directions[number];
export type LensAsset = { token: string; now: number; dynasty: number; you: number; them: number;
  valued: boolean; words: string; age?: number };
export type LensNeed = { position: string; state: 'ok' | 'thin' | 'hole'; have: number; starts: number };
export type LensSide = { franchise: string; direction: Direction; set: boolean; nowWeight: number;
  before: number; after: number; net: number; out: number; in: number; needs: LensNeed[];
  spare: string[] };
export type LensSuggestion = { kind: 'trim' | 'ask' | 'sweeten'; token: string; you: number;
  them: number };
export type WarRoomLens = { preparing: boolean; starterN: number; you: LensSide; them?: LensSide;
  assets: Map<string, LensAsset>; verdict: 'empty' | 'both' | 'balk' | 'bad'; tune: LensSuggestion[];
  notes: string[] };

const flag = (value: unknown, path: string): boolean => {
  if (typeof value !== 'boolean') throw new Error(`${path}: invalid flag`);
  return value;
};
const order = ['QB', 'RB', 'WR', 'TE', 'PK', 'K', 'DT', 'DE', 'LB', 'CB', 'S'];
function side(value: unknown, path: string): LensSide | undefined {
  const s = object(value, path, ['franchise', 'direction', 'set', 'nowWeight', 'before', 'after', 'net',
    'out', 'in', 'needs', 'spare']);
  if (!s.franchise) return undefined;
  const needs = array(s.needs, `${path} needs`, (v, p): LensNeed => {
    const n = object(v, p, ['position', 'state', 'have', 'starts']);
    return { position: text(n.position, p), state: choice(n.state, p, ['ok', 'thin', 'hole']),
      have: integer(n.have, p), starts: integer(n.starts, p) };
  }).sort((a, b) => order.indexOf(a.position) - order.indexOf(b.position));
  return { franchise: text(s.franchise, path), direction: choice(s.direction, path, directions),
    set: flag(s.set, path), nowWeight: finite(s.nowWeight, path), before: finite(s.before, path),
    after: finite(s.after, path), net: finite(s.net, path), out: finite(s.out, path),
    in: finite(s.in, path), needs, spare: array(s.spare, `${path} spare`, text) };
}
export function parseWarRoomLens(value: unknown, league: string, mine: string): WarRoomLens {
  const r = object(value, 'Trade lens', ['preparing', 'league', 'season', 'starterN', 'you', 'them',
    'assets', 'verdict', 'tune', 'notes']);
  if (r.league !== league) throw new Error('Trade lens belongs to another league');
  const preparing = r.preparing === undefined ? false : flag(r.preparing, 'Preparation');
  const empty: LensSide = { franchise: mine, direction: 'retool', set: false, nowWeight: 0, before: 0,
    after: 0, net: 0, out: 0, in: 0, needs: [], spare: [] };
  const you = preparing ? empty : side(r.you, 'Your side');
  if (!you || you.franchise !== mine) throw new Error('Trade lens belongs to another franchise');
  const assets = new Map(array(r.assets, 'Lens assets', (v, p): [string, LensAsset] => {
    const a = object(v, p, ['token', 'now', 'dynasty', 'you', 'them', 'valued', 'words', 'age',
      'ageKnown']);
    const token = text(a.token, p);
    return [token, { token, now: finite(a.now, p), dynasty: finite(a.dynasty, p), you: finite(a.you, p),
      them: finite(a.them, p), valued: flag(a.valued, p), words: text(a.words, p),
      age: flag(a.ageKnown, p) ? finite(a.age, p) : undefined }];
  }));
  return { preparing, starterN: integer(r.starterN, 'Starter count'), you,
    them: preparing ? undefined : side(r.them, 'Their side'), assets,
    verdict: choice(r.verdict || 'empty', 'Verdict', ['empty', 'both', 'balk', 'bad']),
    tune: array(r.tune, 'Tune', (v, p): LensSuggestion => {
      const s = object(v, p, ['kind', 'token', 'you', 'them']);
      return { kind: choice(s.kind, p, ['trim', 'ask', 'sweeten']), token: text(s.token, p),
        you: finite(s.you, p), them: finite(s.them, p) };
    }), notes: array(r.notes, 'Lens notes', text) };
}
export async function readWarRoomLens(c: WarRoomContext, deal: WarRoomDeal) {
  if (!c.native || !c.mine) return undefined;
  const { data } = await (c.native.port ?? hostedWorkflow)({ command: 'tradelens.read', mode: 'read',
    arguments: { league: c.native.league, franchise: c.mine, partner: deal.partner,
      give: deal.partner ? deal.give : [], get: deal.partner ? deal.get : [],
      season: c.snapshot.league.value.season } });
  return parseWarRoomLens(data, c.native.league, c.mine);
}
// An owner's direction is a league record: the whole league reads it, and the lens follows it.
export async function setWarRoomDirection(c: WarRoomContext, direction: Direction, revision: number) {
  if (!c.native) throw new Error('Directions are set in an app league');
  const command = 'commish.franchise.direction.set';
  await (c.native.port ?? hostedWorkflow)({ command, mode: 'execute', arguments: {
    league: c.native.league, command, expected: revision, request: newKey(), reason: '',
    effective: 'now', visibility: 'league', args: { people: { franchise: c.mine, direction } } } });
}
export const directionLabel: Record<Direction, string> = { contend: 'Win now', retool: 'Retool',
  rebuild: 'Rebuild' };
export function lensWords(s: LensSide): string {
  const now = Math.round(s.nowWeight * 100);
  const words = s.direction === 'rebuild' ? 'rebuild: long-term value only' :
    `${directionLabel[s.direction].toLowerCase()}: ${now}% this season, ${100 - now}% long term`;
  return s.set ? words : `our read · ${words}`;
}
export const signed = (n: number) => `${n > -0.05 ? '+' : '−'}${Math.abs(n).toFixed(1)}`;
export const lensValue = (a: LensAsset | undefined, which: 'you' | 'them') =>
  a?.valued ? a[which].toFixed(1) : '—';
export function verdictWords(lens: WarRoomLens): { tag: string; className: string; words: string } {
  const n = lens.starterN, you = lens.you.net, them = lens.them?.net ?? 0;
  if (lens.verdict === 'empty') return { tag: 'EMPTY', className: '',
    words: 'Drag players or picks into the trays.' };
  if (lens.verdict === 'both') return { tag: 'BOTH WIN', className: 'both', words:
    Math.abs(you) < 0.05 && Math.abs(them) < 0.05 ? `Level: neither side's best ${n} moves.` :
      `Holds up in both sides' best ${n}. Most likely to be accepted.` };
  if (lens.verdict === 'balk') return { tag: "THEY'LL BALK", className: 'lean',
    words: `Costs them ${Math.abs(them).toFixed(1)} of their best ${n}. TUNE IT shows what closes it.` };
  return { tag: 'BAD FOR YOU', className: 'bad',
    words: `You give up ${Math.abs(you).toFixed(1)} of your own best ${n}.` };
}
export function leanWords(them: LensSide): string {
  const holes = them.needs.filter(n => n.state === 'hole').map(n => n.position);
  const short = holes.length ? holes : them.needs.filter(n => n.state === 'thin').map(n => n.position);
  const want = short.length ? ` Starters at ${short.join(', ')} move them most.` : '';
  if (them.direction === 'rebuild')
    return `Their lens counts long-term value only: youth and picks hold up, an older player's big ` +
      `season counts for less.${want}`;
  const now = Math.round(them.nowWeight * 100);
  if (them.direction === 'contend')
    return `Their lens leans on this season (${now}%): proven starters count, picks count less.${want}`;
  return `Their lens is mostly long-term (${100 - now}%), with some weight on this season.${want}`;
}
