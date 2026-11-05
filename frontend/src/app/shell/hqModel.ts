import type { LineupReading, Snapshot } from '../data/contract';
import type { Report } from '../reports/contract';
import type { Readings } from '../contracts/contract';

export const unavailable = 'Not available yet';
// A read still in flight has not failed and has not come back empty. Say it is still reading, and
// keep the unavailable words for a read that answered without the fact. Every empty state that can
// be reached while a read is running goes through absent, so no region has to decide this itself.
export const stillReading = 'Still reading…';
export function absent(pending?: boolean): string { return pending ? stillReading : unavailable; }
// Nothing selected is not a failed read. The detail panels say what to do next, in one calm line,
// and keep the unavailable words for a read that actually failed.
export const pickPlayer = 'Pick a player to see their details';
export type HQSettings = { maxYears?: number; active?: number; ir?: number; taxi?: number; starters?: number;
  offence?: number; defence?: number; reserveExcluded?: boolean; positions: { code: string; min: number;
    max?: number }[] };
export type HQCap = { season: number; committed?: number; dead?: number; room?: number; max?: number; ir?:
  number };
export function money(value?: number | null): string {
  return value == null ? unavailable : `$${(value / 100_000_000).toFixed(2)}M`;
}
export function rosterContract(contract?: { salary: number | null; yearsRemaining?: number }): string {
  return contract ? `${money(contract.salary)} · ${contract.yearsRemaining === undefined ? '—' :
    `${contract.yearsRemaining}y`}` : unavailable;
}
export function settingsFromReport(report?: Report): HQSettings {
  const rows = report?.status === 'ok' || report?.status === 'stale' ? report.rows : [];
  const count = (label: string) => {
    const value = rows.find(r => r.cells[0] === label)?.cells[1];
    return value && /^\d+(?: players)?$/.test(value) ? Number.parseInt(value, 10) : undefined;
  };
  const positions = rows.filter(r => r.group === 'Starters').flatMap(r => {
    const match = /^(?:Exactly (\d+)|(\d+) to (\d+))$/.exec(r.cells[1]);
    if (!match) return [];
    return [{ code: r.cells[0] === 'PK' ? 'K' : r.cells[0],
      min: Number(match[1] ?? match[2]), max: Number(match[1] ?? match[3]) }];
  });
  return { active: count('Roster size'), ir: count('Injured reserve'), taxi: count('Taxi squad'),
    starters: count('Starters each week'), offence: count("Offensive starters (MFL's count)"),
    defence: count("Defensive starters (MFL's count)"), positions };
}
export function capFromReads(snapshot: Snapshot, mine: string, contracts?: Readings): HQCap {
  const franchise = snapshot.franchises.value.find(f => f.id === mine);
  const cap = contracts?.cap.value;
  // Signed salary adjustments are not necessarily dead money. Never relabel their sum as dead cap.
  return { season: cap?.season ?? snapshot.league.value.season,
    max: cap?.cap?.cents ?? snapshot.league.value.salaryCap,
    room: cap?.room?.cents ?? franchise?.capRoom };
}
export function lineupCheck(reading?: LineupReading, error?: string): string {
  if (!reading || error || reading.rulesSource.freshness.state === 'fail') return unavailable;
  if (!reading.check.full || !reading.check.legal)
    return reading.check.problems.map(p => p.message).join(' · ') || 'Lineup is partial';
  return 'Lineup is full';
}
// A league whose games are not ranked yet has no record and no place. Say that plainly, and keep the
// standings read's own words in the hover, rather than calling a fact that does not exist unavailable.
export function placeLine(record?: string, standing?: string, note?: string): string {
  const held = [record, standing].filter(Boolean).join(' · ');
  return held || (note ? 'No games ranked yet' : unavailable);
}
export function capacity(count: number, limit?: number): string {
  return `${count} / ${limit ?? unavailable}`;
}
