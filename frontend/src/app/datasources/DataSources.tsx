import { Fragment, useEffect, useState } from 'react';
import { selectProvider } from '../data/provider';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import { ReportView } from '../reports/ReportView';
import { CacheTools } from './CacheTools';
import { dataState } from './state';
import type { Sources } from './provider';
import type { main } from '../../../wailsjs/go/models';
import './sources.css';

export function RefreshScopes({ data, busy, outcomes = {} }:
  { data: Sources; busy: string; outcomes?: Record<string, string> }) {
  const [path, setPath] = useState(data.archiveDir);
  const w = data.words;
  useEffect(() => setPath(data.archiveDir), [data.archiveDir]);
  return <section data-endpoint="S-09" aria-label={w.refresh}>
    <h4>{w.refresh}</h4>
    <div className="archive-folder">
      <label htmlFor="archive-folder">{w.folder}</label>
      <input id="archive-folder" value={path} onChange={e => setPath(e.target.value)} />
      <div className="data-action">
        <Act variant="button"
          verb="data.run" args={{ kind: 'folder', path }} disabled={!!busy || !path.trim()}>
          {w.setFolder}</Act>
        <span role="status" aria-live="polite">{busy === 'folder' ? w.readProgress : outcomes.folder}</span>
      </div>
    </div>
    <dl className="refresh-scopes">{data.scopes.map(scope => <div key={scope.id}>
      <dt>{scope.label}</dt><dd><p>{scope.note}</p>
        <div className="data-action">
          <Act variant="button"
            verb="refresh.run" args={{ scope: scope.id }} disabled={!!busy || (scope.id === 'archive'
            && !data.archiveDir)}>{scope.action}</Act>
          <span role="status" aria-live="polite">
            {busy === scope.id ? scope.progress : outcomes[scope.id]}</span>
        </div>
      </dd></div>)}
      <div><dt>{w.photos}</dt><dd><p>{w.photosNote}</p>
        <div className="data-action">
          <Act variant="button"
            verb="data.run" args={{ kind: 'photos', on: !data.photosOn }} disabled={!!busy}>
            {data.photosOn ? w.photosOff : w.photosOn}</Act>
          <span role="status" aria-live="polite">{busy === 'photos' ? w.readProgress : outcomes.photos}</span>
        </div>
      </dd></div></dl>
  </section>;
}
export function Parameters({ params, busy, words: w }:
  { params: main.ParamsResult; busy: string; words: Record<string, string> }) {
  const canEdit = commands.registry['data.parameter'].roles.includes('gm');
  const [filter, setFilter] = useState('');
  const [edits, setEdits] = useState<Record<string, string>>({});
  const rows = (params.params ?? []).filter(p =>
    `${p.label} ${p.group} ${p.position}`.toLowerCase().includes(filter.toLowerCase()));
  return <section data-endpoint="C-09" aria-label={w.parameters}>
    <h4>{w.parameters}</h4>
    {!canEdit && <p>{w.permission}</p>}
    <label htmlFor="parameter-filter">{w.find}</label>
    <input id="parameter-filter" value={filter} onChange={e => setFilter(e.target.value)} />
    <div className="draft-table-scroll" role="region" aria-label={w.parameters} tabIndex={0}>
      <table><thead><tr>{[w.setting, w.value, w.default, w.override].map(h => <th
        key={h} scope="col">{h}</th>)}</tr></thead>
        <tbody>{rows.map((p, i) => {
          const id = `${p.key}@${p.position}`;
          const draft = edits[id] ?? '';
          const value = Number(draft);
          return <Fragment key={id}>
            {p.group !== rows[i - 1]?.group && <tr className="report-group">
              <th scope="colgroup" colSpan={4}>{p.group}</th></tr>}
            <tr><td>{p.label}<p>{p.position || w.league}</p><p>{w.range}: {p.min}–{p.max}</p></td>
              <td>{p.value}{p.overridden && <p>{w.overridden}</p>}</td>
              <td>{p.default}<p>{p.calibrated ? w.fitted : w.handSet}</p></td>
              <td><input type="number" step="any" min={p.min} max={p.max}
                aria-label={`${p.label} · ${p.position || w.league} · ${w.override}`}
                disabled={!canEdit || !!busy}
                value={draft} onChange={e => setEdits({ ...edits, [id]: e.target.value })} />
                <Act variant="button" verb="data.parameter" args={{ kind: 'parameter', key: p.key,
                  position: p.position, value }}
                  disabled={!canEdit || !!busy || !draft.trim() || !Number.isFinite(value) || value
                    < p.min || value > p.max}>{w.apply}</Act>
                {p.overridden && <Act variant="button"
                  verb="data.parameter" args={{ kind: 'parameter', key: p.key,
                  position: p.position, value: null }} label={w.clear} disabled={!canEdit || !!busy}>
                  {w.reset}</Act>}
              </td></tr>
          </Fragment>;
        })}</tbody></table>
    </div>
    {rows.length === 0 && <p>{w.empty}</p>}
  </section>;
}
export function HeldRead({ words: w, busy, result }:
  { words: Record<string, string>; busy: string; result?: string }) {
  return <div className="data-action"><Act variant="button" verb="data.run" args={{ kind: 'read' }}
    disabled={!!busy}>{w.read}</Act>
    <span role="status" aria-live="polite">{busy === 'read' ? w.readProgress : result}</span>
  </div>;
}
export default function DataSources() {
  const s = dataState.use();
  useEffect(() => {
    const read = () => { void dataState.run({ kind: 'read' }); };
    read();
    return selectProvider().onSeasonChange(read);
  }, []);
  const w = s.data?.words;
  return <div className="data-sources draft-reports">
    {w && <>
      <h4>{w.title}</h4>
      <HeldRead words={w} busy={s.busy} result={s.outcomes.read} />
    </>}
    {s.error && <p role="alert">{s.error} {w?.kept}</p>}
    {s.data && <>
      {[s.data.health, s.data.signals, s.data.facts, s.data.checks].map(report =>
        <ReportView key={report.id} report={report} howLabel={s.data!.words.how} heading="h4" />)}
      <RefreshScopes data={s.data} busy={s.busy} outcomes={s.outcomes} />
      <CacheTools />
    </>}
    {s.crosswalk && w && <ReportView report={s.crosswalk} howLabel={w.crosswalkHow} heading="h4" />}
    {s.params && w && <Parameters params={s.params} busy={s.busy} words={w} />}
  </div>;
}
