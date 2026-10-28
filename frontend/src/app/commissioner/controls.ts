import { array, integer, object, text } from '../data/parseValues';
import { commissionerControls } from '../registry/commissioner.gen';

// The Go controls reading (TargetCommissionerControls): every census row, grouped, and whether a person
// can use it today. Each row's plain name and question come from the registry, which is generated from
// the census, so no wording is copied by hand here.
type ControlField = { field: string; label: string };
export type Control = { id: string; built: boolean; note: string; fields: ControlField[] };
export type ControlGroup = { id: string; label: string; controls: Control[] };
// A preset is a named bundle of setting values from Go's catalog; the start form lists exactly these.
export type StartingPreset = {
  id: string; label: string; description: string; minTeams: number; maxTeams: number;
};
export type Controls = { groups: ControlGroup[]; presets: StartingPreset[] };

function flag(v: unknown, path: string): boolean {
  if (typeof v !== 'boolean') throw new Error(`${path}: expected true or false`);
  return v;
}
export function parseControls(v: unknown): Controls {
  const r = object(v, 'League controls', ['groups', 'presets']);
  return {
    presets: array(r.presets, 'Starting presets', (x, p) => {
      const o = object(x, p, ['id', 'label', 'description', 'minTeams', 'maxTeams']);
      return { id: text(o.id, p), label: text(o.label, p), description: text(o.description, p),
        minTeams: integer(o.minTeams, `${p}.minTeams`, 1),
        maxTeams: integer(o.maxTeams, `${p}.maxTeams`, 1) };
    }),
    groups: array(r.groups, 'Control groups', (g, p) => {
      const o = object(g, p, ['id', 'label', 'controls']);
      return {
        id: text(o.id, p), label: text(o.label, p),
        controls: array(o.controls, `${p}.controls`, (c, q) => {
          const x = object(c, q, ['id', 'built', 'note', 'fields']);
          return {
            id: text(x.id, q), built: flag(x.built, `${q}.built`), note: text(x.note, q),
            fields: array(x.fields, `${q}.fields`, (f, w) => {
              const y = object(f, w, ['field', 'label']);
              return { field: text(y.field, w), label: text(y.label, w) };
            }),
          };
        }),
      };
    }),
  };
}

type Words = ReadonlyMap<string, { name: string; question: string }>;
const registryWords: Words = new Map(
  commissionerControls.map(row => [row.id, { name: row.name, question: row.gm_question }]));

// Entry is one control with its group and its words, ready to list and to search.
export type Entry = Control & { group: string; groupLabel: string; name: string; question: string };
export function entriesOf(controls: Controls, words: Words = registryWords): Entry[] {
  return controls.groups.flatMap(group => group.controls.map(control => {
    const said = words.get(control.id);
    if (!said) throw new Error(`The league controls list a row the registry does not: ${control.id}`);
    return { ...control, group: group.id, groupLabel: group.label, ...said };
  }));
}

// Hit is a search result: the control, and the rule it was found by when a rule's own label matched.
type Hit = { entry: Entry; via?: ControlField };
const wordsOf = (query: string) => query.toLowerCase().split(/\s+/).filter(Boolean);

// searchControls finds a control by its plain question or name, or by the label of a rule it owns.
// Question and name matches come first, in census order; rule label matches follow.
export function searchControls(entries: Entry[], query: string): Hit[] {
  const words = wordsOf(query);
  if (words.length === 0) return [];
  const all = (haystack: string) => words.every(w => haystack.toLowerCase().includes(w));
  const hit = (entry: Entry): Hit => ({ entry,
    via: entry.fields.find(f => f.label.toLowerCase() === query.trim().toLowerCase())
      ?? entry.fields.find(f => all(f.label)),
  });
  const asked = entries.filter(e => all(`${e.question} ${e.name}`)).map(hit);
  const named = entries.filter(e => !all(`${e.question} ${e.name}`)).map(hit).filter(h => h.via);
  return [...asked, ...named];
}
