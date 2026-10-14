import { object, text, timestamp } from '../data/parseValues';

// Who set a value, when and with which recorded change, as Go reads it from the league ledger. Any screen
// that shows a value the commissioner set, a commissioner's or an owner's, shows it with the glyph.
export type Governed = { entry: string; who: string; when: string; change: string };

export function parseGoverned(v: unknown, path: string): Governed {
  const r = object(v, path, ['entry', 'who', 'when', 'change']);
  return { entry: text(r.entry, path), who: text(r.who, path), when: timestamp(r.when, `${path}.when`),
    change: text(r.change, path) };
}
