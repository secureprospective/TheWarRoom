import { fixes, pickActs, situations, synonyms } from './guide';
import type { Room } from './room';
import { openFix, openPickAct, openSituation, openTeam } from './openers';

export type Hit = { type: string; label: string; words: string; extra?: string; open(room: Room): void };

/** Everything the find box can reach: the league's rules (current values), its jobs, the situations, the
 *  fixes, its franchises and the pick actions. */
export function searchIndex(room: Room): Hit[] {
  const out: Hit[] = [];
  for (const section of room.state.rules?.sections ?? [])
    for (const r of section.rows)
      out.push({ type: 'RULE', label: r.label, words: `${section.label} · ${r.value}`,
        open: x => x.go('rules', () => x.rule(r.field)) });
  for (const j of room.state.jobs?.jobs ?? [])
    out.push({ type: 'JOB', label: j.name, words: j.words, open: x => x.go('jobs') });
  situations.forEach((s, i) => out.push({ type: 'SITUATION', label: s[1], words: s[0], extra: synonyms[i] ?? '',
    open: x => x.go('sit', () => openSituation(x, i)) }));
  fixes.forEach((f, i) => out.push({ type: 'FIX', label: f[0], words: f[1],
    open: x => x.go('fix', () => openFix(x, i)) }));
  for (const s of room.state.people?.seats ?? [])
    out.push({ type: 'TEAM', label: s.name, words: 'seat · picks · roster',
      open: x => x.go('people', () => openTeam(x, s.franchise)) });
  pickActs.forEach((p, i) => out.push({ type: 'PICKS', label: p[0], words: p[1],
    open: x => x.go('picks', () => openPickAct(x, i)) }));
  return out;
}

/** The mock's match: at least half the words, best first, twelve at most. */
export function runSearch(room: Room, query: string): Hit[] {
  const words = query.trim().toLowerCase().split(/\s+/).filter(Boolean);
  const need = Math.ceil(words.length / 2);
  return searchIndex(room)
    .map(h => {
      const text = `${h.type} ${h.label} ${h.words} ${h.extra ?? ''}`.toLowerCase();
      return [h, words.filter(w => text.includes(w)).length] as const;
    })
    .filter(([, n]) => n >= need)
    .sort((a, b) => b[1] - a[1])
    .map(([h]) => h)
    .slice(0, 12);
}
