import { leagueState } from './proxy';
import { parsePeople } from './people';
import { parseRules } from './rules';
import type { App } from './commissionerDom';
import ledgerWire from './wire.testdata.json';
import peopleWire from './people.wire.testdata.json';
import rulesWire from './rules.wire.testdata.json';
import previewWire from './preview.wire.testdata.json';

// The Go bindings as the forms see them: readings are the Go-written fixtures, and every execute call
// is recorded so a test can read exactly what was sent.
export const rules = parseRules(rulesWire);
export const commissionerPeople = parsePeople(peopleWire.commissioner);
export const ownerPeople = parsePeople(peopleWire.owner);
export const receipt = (over: object = {}) => ({
  entry: 'act.new', league: rulesWire.league, revision: 4, recordedAt: '2026-10-13T12:00:00Z',
  message: 'Recorded in the league ledger.', ...over,
});
export type Sent = { command: string; league: string; request: string; reason: string; expected: number;
  effective: string; visibility: string; args: Record<string, unknown> };
type Reading = unknown | (() => unknown);
const read = (value: Reading) => (typeof value === 'function' ? (value as () => unknown)() : value);
export function stub(over: Partial<App> = {}, readings: { people?: Reading; rules?: Reading } = {}) {
  const calls = { sent: [] as Sent[], ledger: [] as string[], rules: [] as string[], people: [] as string[] };
  const app: Partial<App> = {
    TargetCommissionerPreview: async body => {
      const sent = JSON.parse(body) as Sent;
      const source = sent.effective === 'now' ? previewWire.reason : previewWire.hold;
      return { ...source, target: { ...source.target, league: sent.league, version: sent.expected } };
    },
    TargetCommissionerExecute: async body => { calls.sent.push(JSON.parse(body)); return receipt(); },
    TargetCommissionerLedger: async league => { calls.ledger.push(league); return ledgerWire; },
    TargetCommissionerLeagues: async () => [{ id: rulesWire.league, name: rulesWire.name }],
    TargetCommissionerRules: async league => {
      calls.rules.push(league);
      return read(readings.rules ?? rulesWire);
    },
    TargetCommissionerPeople: async league => {
      calls.people.push(league);
      return read(readings.people ?? peopleWire.commissioner);
    },
    ...over,
  };
  return { app, calls };
}
export async function open(league = rulesWire.league) {
  leagueState.forget();
  await leagueState.choose(league);
}
