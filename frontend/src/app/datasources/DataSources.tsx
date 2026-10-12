import { useEffect, useState } from 'react';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import { ReportView } from '../reports/ReportView';
import { dataState } from './state';
import type { Sources } from './provider';
import type { main } from '../../../wailsjs/go/models';
import './sources.css';

export function RefreshScopes({ data, busy }: { data: Sources; busy: string }) {
  const [path, setPath] = useState(data.archiveDir);
  useEffect(() => setPath(data.archiveDir), [data.archiveDir]);
  return <section data-endpoint="S-09" aria-label="Data refresh">
    <h4>Data refresh</h4>
    <div className="archive-folder">
      <label htmlFor="archive-folder">League archive folder (raw)</label>
      <input id="archive-folder" value={path} onChange={e => setPath(e.target.value)} />
      <Act verb="data.run" args={{ kind: 'folder', path }} disabled={!!busy || !path.trim()}>Set folder</Act>
    </div>
    <dl className="refresh-scopes">{data.scopes.map(scope => <div key={scope.id}>
      <dt>{scope.label}</dt><dd><p>{scope.note}</p>
        <Act verb="refresh.run" args={{ scope: scope.id }} disabled={!!busy || (scope.id === 'archive'
          && !data.archiveDir)}>
          {busy === scope.id ? `Refreshing ${scope.label}…` : `Refresh ${scope.label}`}
        </Act>
      </dd></div>)}</dl>
  </section>;
}
export function Parameters({ params, busy }: { params: main.ParamsResult; busy: string }) {
  const canEdit = commands.registry['data.parameter'].roles.includes('gm');
  const [filter, setFilter] = useState('');
  const [edits, setEdits] = useState<Record<string, string>>({});
  const rows = (params.params ?? []).filter(p =>
    `${p.key} ${p.position} ${p.description}`.toLowerCase().includes(filter.toLowerCase()));
  return <section data-endpoint="C-09" aria-label="Engine parameters">
    <h4>Engine parameters</h4>
    <p>Changes require the admin role. Running as GM: values are read-only.</p>
    <label htmlFor="parameter-filter">Find a setting</label>
    <input id="parameter-filter" value={filter} onChange={e => setFilter(e.target.value)} />
    <div className="draft-table-scroll" role="region" aria-label="Engine parameter values" tabIndex={0}>
      <table><thead><tr>{['Position', 'Setting', 'Value', 'Default', 'Range', 'Override'].map(h => <th
        key={h} scope="col">{h}</th>)}</tr></thead>
        <tbody>{rows.map(p => {
          const id = `${p.key}@${p.position}`;
          const draft = edits[id] ?? '';
          const value = Number(draft);
          return <tr key={id}><td>{p.position || 'League'}</td><td>{p.key}<p>{p.description}</p></td>
            <td>{p.value}{p.overridden && <p>Overridden</p>}</td><td>{p.default}<p>{p.calibrated
              ? 'Fitted' : 'Hand-set'}</p></td><td>{p.min}–{p.max}</td>
            <td><input type="number" step="any" min={p.min} max={p.max}
              aria-label={`${p.key} ${p.position} override`}
              disabled={!canEdit || !!busy}
              value={draft} onChange={e => setEdits({ ...edits, [id]: e.target.value })} />
              <Act verb="data.parameter" args={{ kind: 'parameter', key: p.key, position: p.position, value }}
                disabled={!canEdit || !!busy || !draft.trim() || !Number.isFinite(value) || value
                  < p.min || value > p.max}>Apply</Act>
              {p.overridden && <Act verb="data.parameter" args={{ kind: 'parameter', key: p.key,
                position: p.position, value: null }}
                  label="Clear the Admin value and follow the shipped default" disabled={!canEdit
                    || !!busy}>Reset</Act>}
            </td></tr>;
        })}</tbody></table>
    </div>
    {rows.length === 0 && <p>No settings match.</p>}
  </section>;
}
export default function DataSources() {
  const s = dataState.use();
  useEffect(() => { void dataState.run({ kind: 'read' }); }, []);
  return <div className="data-sources draft-reports">
    <h4>Data and sources</h4>
    <div className="market-controls"><Act verb="data.run" args={{ kind: 'read' }}
      disabled={!!s.busy}>Read held records</Act></div>
    {s.busy && <p role="status" aria-live="polite">{s.busy === 'read' ? 'Reading held data…'
      : `Working: ${s.busy}…`}</p>}
    {s.error && <p role="alert">{s.error}{s.data && ' The last reading is kept.'}</p>}
    {s.result && <p role="status">{s.result}</p>}
    {s.data && <>
      <ReportView report={s.data.health} howLabel="How this is counted" heading="h4" />
      <ReportView report={s.data.signals} howLabel="How this is counted" heading="h4" />
      <ReportView report={s.data.facts} howLabel="How this is counted" heading="h4" />
      <ReportView report={s.data.checks} howLabel="How this is counted" heading="h4" />
      <RefreshScopes data={s.data} busy={s.busy} />
    </>}
    {s.crosswalk && <ReportView report={s.crosswalk} howLabel="How players are linked" heading="h4" />}
    {s.params && <Parameters params={s.params} busy={s.busy} />}
  </div>;
}
