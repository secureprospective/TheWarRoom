import { array, choice, id, integer, object, text, timestamp } from '../data/parseValues';
import { report, type Report } from '../reports/contract';

export type ListKind = 'watch' | 'bait' | 'note' | 'draft';
export type ListChange = {
  id: string; entity: 'player' | 'franchise'; action: 'add' | 'remove' | 'up' | 'down'; text?: string;
};
export type Entry = {
  kind: ListKind; entity: 'player' | 'franchise'; id: string; text: string; at: string; order: number;
};
export type PersonalReading = {
  entries: Entry[]; bait: Report; players: { id: string; status: string; market: string[] }[] };
export function parsePersonal(value: unknown): PersonalReading {
  const r = object(value, 'Private lists', ['entries', 'bait', 'players']);
  return { bait: report(r.bait, 'League trade bait'), players: array(r.players, 'Players', (v, p) => {
    const player = object(v, p, ['id', 'status', 'market']);
    return { id: id(player.id, p, true), status: text(player.status, p), 
      market: array(player.market, p, text) };
  }), entries: array(r.entries, 'Entries', (v, p) => {
    const e = object(v, p, ['kind', 'entity', 'id', 'text', 'at', 'order']);
    const entity = choice(e.entity, p, ['player', 'franchise']);
    return { kind: choice(e.kind, p, ['watch', 'bait', 'note', 'draft']), entity,
      id: id(e.id, p, entity === 'player'), text: text(e.text, p), at: timestamp(e.at, p),
      order: integer(e.order, p, 1) };
  }) };
}
