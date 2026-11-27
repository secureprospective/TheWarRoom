import type { NativeHQ } from '../commissioner/hqReading';
import { slotRank } from '../shell/pulseModel';
import type { Draft } from '../draftroom/draft';
import type { Game } from '../now/liveContract';
import { boardSlots, type HQLineup } from '../shell/HQBoard';
import { fitsSlot } from '../shell/hqPlacement';

// The phone Team tab's view of the hosted HQ reading: rows, states and legal moves. Presentation
// only: points, projections and locks are the server's numbers; the lineup command checks legality.

export type TeamSection = 'starters' | 'bench' | 'ir';
export type TeamState = 'live' | 'fin' | 'bye' | 'out' | '';
export type TeamRow = {
  key: string; section: TeamSection; slot: string; index: number; id?: string; name: string;
  position: string; state: TeamState; words: string; points?: number; range?: [number, number];
  locked: boolean;
};
export type TeamView = {
  starters: TeamRow[]; bench: TeamRow[]; ir: TeamRow[]; points?: number; projection?: number;
  locked: number; bye?: TeamRow; irOutsideBench: boolean;
};

export const dayTime = (at: string) => new Date(at).toLocaleString('en-US',
  { weekday: 'short', hour: 'numeric', minute: '2-digit' }).replace(',', '').toUpperCase();

/** gameWords is the row's game line: live clock, final, bye, or the kickoff the reads hold. */
export function gameWords(team: string | undefined, games: Game[] | undefined, kickoff: string | undefined):
  { state: TeamState; words: string; started: boolean } {
  const club = team || '—';
  const game = team ? games?.find(g => g.home.abbreviation === team || g.away.abbreviation === team)
    : undefined;
  if (game?.status === 'in')
    return { state: 'live', words: `${club} · LIVE Q${game.period} ${game.clock}`.trim(), started: true };
  if (game?.status === 'post') return { state: 'fin', words: `${club} · FINAL`, started: true };
  if (team && games?.length && !game) return { state: 'bye', words: 'ON BYE · 0 PTS', started: false };
  const at = game?.kickoff || kickoff;
  const opponent = game ? game.home.abbreviation === team ? `VS ${game.away.abbreviation}`
    : `AT ${game.home.abbreviation}` : '';
  const started = Boolean(at && Date.parse(at) <= Date.now());
  return { state: '', started, words: [club, at ? `${dayTime(at)}${opponent ? ` ${opponent}` : ''}` : '']
    .filter(Boolean).join(' · ') };
}

export function teamView(reading: NativeHQ, lineup: HQLineup | undefined, mine: string,
  games?: Game[]): TeamView {
  const players = reading.snapshot.players.value;
  const roster = reading.snapshot.rosters.value.find(r => r.franchiseId === mine)?.players ?? [];
  const own = reading.thisWeek?.scores.find(s => s.franchise === mine);
  const scored = new Map(own?.starters.map(p => [p.player, p.points ?? p.provisional]));
  const row = (section: TeamSection, slot: string, index: number, id?: string,
    fallback?: string): TeamRow => {
    const player = players.find(p => p.id === id);
    const position = player?.position || fallback || '—';
    const key = `${section}-${index}`;
    if (!id) return { key, section, slot, index, name: 'Empty starter spot', position: '', state: '',
      words: '', locked: false };
    const history = reading.playerHistory?.[id]?.map(p => p.points);
    const range: [number, number] | undefined = history?.length
      ? [Math.min(...history), Math.max(...history)] : undefined;
    const status = roster.find(p => p.id === id)?.rosterStatus;
    const game = gameWords(player?.team, games, reading.kickoffs?.get(id));
    const out = status === 'IR';
    return { key, section, slot, index, id, name: player?.name || id, position,
      state: out ? 'out' : game.state, words: out ? 'INJURED RESERVE' : game.words,
      points: section === 'starters' ? scored.get(id) : undefined,
      range, locked: game.started };
  };
  const starters = boardSlots(lineup, reading.settings).map((slot, i) =>
    row('starters', slot.code, i, slot.id, lineup?.starters.find(p => p.id === slot.id)?.position))
    .sort((a, b) => slotRank(a.slot) - slotRank(b.slot));
  const irIds = new Set(roster.filter(p => p.rosterStatus === 'IR').map(p => p.id));
  const bench = (lineup?.bench ?? []).filter(p => !irIds.has(p.id))
    .map((p, i) => row('bench', 'BN', i, p.id, p.position));
  const ir = [...irIds].map((id, i) => row('ir', 'IR', i, id));
  const projection = reading.projections && lineup ? lineup.starters.reduce((sum, p) =>
    sum + (reading.projections!.get(p.id) ?? 0), 0) : undefined;
  return { starters, bench, ir, points: own?.points ?? own?.provisional, projection,
    locked: starters.filter(r => r.id && r.locked).length,
    bye: starters.find(r => r.state === 'bye'), irOutsideBench: reading.settings.reserveExcluded !== false };
}

/** canMoveTo: a starter and a bench player trade places only when each fits and neither game started. */
export function canMoveTo(from: TeamRow, to: TeamRow, reading: NativeHQ): boolean {
  if (from.key === to.key || from.section === 'ir' || to.section === 'ir') return false;
  if (from.locked || to.locked) return false;
  if (from.section === to.section) return false;
  const [starter, bench] = from.section === 'starters' ? [from, to] : [to, from];
  // An open bench spot is a place to add a player from Moves, not a swap target.
  if (!bench.id) return false;
  return fitsSlot(bench.id ? { id: bench.id, position: bench.position } : undefined,
    { code: starter.slot, side: 'off' }, reading.settings);
}

/** benchStarter removes a starter id from the starting list so he moves straight to the bench. */
export function benchStarter(lineup: HQLineup, id: string): string[] {
  return lineup.starters.filter(p => p.id !== id).map(p => p.id);
}

/** moveStarters is the starter id list after the swap; the hosted lineup command checks it. */
export function moveStarters(lineup: HQLineup, from: TeamRow, to: TeamRow): string[] {
  const [starter, bench] = from.section === 'starters' ? [from, to] : [to, from];
  const ids = lineup.starters.map(p => p.id);
  if (!starter.id) return bench.id ? [...ids, bench.id] : ids;
  return ids.map(id => id === starter.id ? bench.id! : id);
}

export type TeamPick = { key: string; label: string; from: string };
/** myPicks lists the draft slots this franchise owns and has not yet used, from draft.read. */
export function myPicks(draft: Draft, mine: string): TeamPick[] {
  const used = new Set(draft.picks.filter(p => (p.extra ?? 0) === 0 && (p.player !== '' || p.forfeited ||
    (p.skipped && p.how !== 'expired'))).map(p => `${p.round}-${p.slot}`));
  const name = (id: string) => draft.owners.find(o => o.id === id)?.name || id;
  return draft.order.flatMap(round => round.picks)
    .filter(p => p.owner === mine && !used.has(`${p.round}-${p.slot}`))
    .map(p => ({ key: `${p.round}-${p.slot}`, label: `${draft.season} round ${p.round}`,
      from: p.original === mine ? 'own' : `from ${name(p.original)}` }));
}
