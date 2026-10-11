import { object, choice, text, array } from '../data/parseValues';
import { railStates } from './moveLabels';

export function parseRailReceipt(v: unknown, p: string) {
  const r = object(v, p, ['correlationId', 'spec', 'state', 'audit']);
  const spec = object(r.spec, p, [
    'intent', 'leagueId', 'franchiseId', 'subject', 'expected', 'gravity', 'undo', 'target', 'deadline',
  ]);
  const state = choice(r.state, p, railStates);
  const audit = array(r.audit, p, (e, ep) => {
    const entry = object(e, ep, ['at', 'from', 'event', 'to', 'note']);
    return {
      at: text(entry.at, ep), from: choice(entry.from, ep, railStates),
      to: choice(entry.to, ep, railStates), note: text(entry.note, ep),
    };
  });
  return {
    id: text(r.correlationId, p), state, franchiseId: text(spec.franchiseId, p),
    history: audit.map((entry) => ({ at: entry.at, state: entry.to, note: entry.note })),
    rail: { state, spec: { intent: text(spec.intent, p) }, audit },
  };
}
