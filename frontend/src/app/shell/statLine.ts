/** A starter's recorded statistics for one week, by league stat id. */
export type StatLine = Record<string, number>;

const num = (value: number) => String(Math.round(value * 10) / 10);

/** segments are the short phrases a stat line is made of; each is absent when nothing was recorded. */
function segments(line: StatLine) {
  const q = (...ids: string[]) => ids.reduce((sum, id) => sum + (line[id] ?? 0), 0);
  const when = (count: number, words: string) => (count > 0 ? words : undefined);
  return {
    touchdowns: when(q('passing_td', 'rushing_td', 'receiving_td', 'interception_return_td',
      'fumble_recovery_tds'), `${num(q('passing_td', 'rushing_td', 'receiving_td',
      'interception_return_td', 'fumble_recovery_tds'))} TD`),
    passes: when(q('pass_attempt'), `${num(q('pass_completion'))}/${num(q('pass_attempt'))}`),
    passYards: when(q('pass_attempt'), `${num(q('passing_yards'))} yds`),
    rush: when(q('rushing_attempt'), `${num(q('rushing_attempt'))}-${num(q('rushing_yards'))} rush`),
    carries: when(q('rushing_attempt'), `${num(q('rushing_attempt'))} car, ${num(q('rushing_yards'))} yds`),
    catches: when(q('reception', 'target'), `${num(q('reception'))}-${num(q('receiving_yards'))} rec`),
    catchYards: when(q('reception'), `${num(q('reception'))} rec, ${num(q('receiving_yards'))} yds`),
    targets: when(q('target'), `${num(q('target'))} tgt`),
    fieldGoals: when(q('fg_attempt'), `${num(q('fg_made'))}/${num(q('fg_attempt'))} FG`),
    extraPoints: when(q('pat_made'), `${num(q('pat_made'))} XP`),
    tackles: when(q('tackle', 'assist'), `${num(q('tackle', 'assist'))} tkl`),
    sacks: when(q('sack'), `${num(q('sack'))} sack`),
    turnovers: [when(q('interception_thrown'), `${num(q('interception_thrown'))} INT`),
      when(q('fumble_lost'), `${num(q('fumble_lost'))} FUM`),
      when(q('interception_caught'), `${num(q('interception_caught'))} INT`),
      when(q('forced_fumble'), `${num(q('forced_fumble'))} FF`)],
  };
}

const said = (parts: (string | undefined)[]) => parts.filter((part): part is string => part !== undefined);

/** compactLine is the line under a starter: touchdowns first, then the two phrases that say most
 *  about the position played. */
export function compactLine(line: StatLine | undefined, slot: string): string[] {
  if (!line) return [];
  const s = segments(line);
  const rest = slot === 'QB' ? [s.passes, s.passYards, s.rush]
    : slot === 'RB' ? [s.rush, s.catches]
    : slot === 'WR' || slot === 'TE' ? [s.catches, s.targets, s.rush]
    : slot === 'K' || slot === 'PK' ? [s.fieldGoals, s.extraPoints]
    : [s.tackles, s.sacks, ...s.turnovers.slice(2)];
  return said([s.touchdowns, ...said(rest).slice(0, 2)]);
}

/** boxLine is a starter's row in the fantasy box score: two lines of production, then scores and
 *  turnovers. A cell with nothing recorded reads as a dash. */
export function boxLine(line: StatLine | undefined, slot: string): [string, string, string] {
  if (!line) return ['–', '–', '–'];
  const s = segments(line);
  const join = (parts: (string | undefined)[], by: string) => said(parts).join(by) || '–';
  const [one, two] = slot === 'QB' ? [join([s.passes, s.passYards], ', '), s.carries]
    : slot === 'RB' ? [s.carries, s.catchYards]
    : slot === 'WR' || slot === 'TE' ? [join([s.catchYards, s.targets], ' / '), s.carries]
    : slot === 'K' || slot === 'PK' ? [s.fieldGoals, s.extraPoints]
    : [s.tackles, s.sacks];
  return [one ?? '–', two ?? '–', join([s.touchdowns, ...s.turnovers], ' · ')];
}
