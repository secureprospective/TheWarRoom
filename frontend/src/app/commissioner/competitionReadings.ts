import type * as Bindings from '../../../wailsjs/go/main/App';
import type { ActReadings, Option } from './actReadings';
import type { FieldKind } from './actTable';

export async function readCompetitionOptions(app: typeof Bindings, league: string,
  kinds: readonly FieldKind[], season: number, revision: number, options: ActReadings['options']) {
  if (kinds.includes('scheduleGame')) {
    const schedule = await app.TargetCommissionerSchedule(league);
    if (schedule.league !== league || schedule.revision !== revision)
      throw new Error('The schedule changed. Reopen the change.');
    const name = (id: string) => options.franchise?.find(f => f.value === id)?.label ?? 'Unnamed team';
    options.scheduleGame = [
      ...schedule.draft.flatMap(w => w.games.map(g => ({
        value: JSON.stringify({ week: w.week, id: g.id, published: false }),
        label: `Draft week ${w.week}: ${name(g.home)} v ${name(g.away)}`,
      }))),
      ...schedule.published.flatMap(w => w.games.map(g => ({
        value: JSON.stringify({ week: w.week, id: g.id, published: true }),
        label: `Published week ${w.week}: ${name(g.home)} v ${name(g.away)}`,
      }))),
    ];
  }
  if (!kinds.some(k => ['bracketNode', 'bracketGame', 'bracketSlot', 'preset', 'shape'].includes(k)))
    return;
  const bracket = await app.TargetNativeBracket(league, season);
  if (bracket.league !== league || bracket.revision !== revision || bracket.season !== season)
    throw new Error('The playoffs changed. Reopen the change.');
  options.preset = [
    { value: 'league_rules', label: 'League playoff rules' },
    { value: 'legacy_nfl', label: 'Legacy NFL playoff layout' },
    ...bracket.presets.map(value => ({ value, label: `Saved layout: ${value}` })),
  ];
  options.shape = bracket.rules.shape.length ? [{ value: JSON.stringify(bracket.rules.shape),
    label: 'Current playoff layout' }] : [];
  options.bracketNode = [];
  options.bracketGame = [];
  const slots = new Map<string, Option>();
  for (const field of bracket.fields) for (const team of field.seeds) {
    slots.set(String(team.seed), { value: String(team.seed),
      label: `Seed ${team.seed} in the chosen team’s conference` });
  }
  options.bracketSlot = [...slots.values()];
  for (const round of bracket.rounds) for (const game of round.games) {
    const side = (team: typeof game.home) => team ? `${team.name} (seed ${team.seed})` : 'Team to be decided';
    const label = `${round.name}, week ${game.week}: ${side(game.home)} v ${side(game.away)}`;
    options.bracketGame.push({ value: game.id, label });
    options.bracketNode.push({ value: game.homeSlot, label: `${label} — home team` },
      { value: game.awaySlot, label: `${label} — away team` });
  }
}
