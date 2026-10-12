import { useEffect, useState } from 'react';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { ReportsView } from '../draftreports/Reports';
import { ReportView } from '../reports/ReportView';
import { researchState } from './state';
import type { Research as ResearchBoard } from './contract';
import './research.css';

function Choice({ name, field, value, options, all = false }: {
  name: string; field: string; value: string; options: { value: string; label: string }[]; all?: boolean;
}) {
  return <label>{name}{' '}<select value={value}
    onChange={e => commands.dispatch('research.read', { action: field, value: e.target.value })}>
    {all && <option value="">All</option>}
    {options.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
  </select></label>;
}
function NewsActions({ research, mine }: { research: ResearchBoard; mine: boolean }) {
  const names = mine ? new Set(research.board.reports.find(r => r.id === 'M-060')?.rows
    .map(r => r.cells[0])) : null;
  return <div className="research-articles" data-endpoint="M-059">
    <Act verb="research.read" args={{ action: 'news', value: mine ? 'mine' : 'league' }}
      disabled={mine && !research.scope.mine}>
      Refresh rostered players’ news
    </Act>
    <ul>{research.articles.filter(a => !names || names.has(a.name)).map(a =>
      <li key={`${a.id}:${a.article}`}>
        <Act verb="research.read" args={{ action: 'open', value: `${a.id}:${a.article}` }}
          label={`Open on MFL · ${a.name} · ${a.headline}`}>{a.name} · Open article on MFL</Act>
      </li>)}</ul>
  </div>;
}
export function NewsView({ research }: { research: ResearchBoard }) {
  return <div className="draft-reports research-news">
    <ReportView report={research.news} howLabel={research.board.howLabel} heading="h4" />
    <NewsActions research={research} mine={false} />
  </div>;
}
export function ResearchView({ research, selected }: {
  research: ResearchBoard; selected: readonly string[];
}) {
  const [search, setSearch] = useState(research.scope.search);
  const chosen = selected.includes('M-059') ? ['M-060'] : selected;
  const shown = research.board.reports.find(r => chosen.includes(r.id)) ?? research.board.reports[0];
  const s = research.scope;
  const options = (values: string[]) => values.map(value => ({ value, label: value }));
  const controls = <div className="market-controls" aria-label="Research choices">
    {['M-037', 'M-038', 'M-053', 'M-054', 'M-051', 'M-057'].includes(shown.id) &&
      <Choice name="Position" field="position" value={s.position} options={options(research.positions)}
        all />}
    {['M-037', 'M-038', 'M-053', 'M-054', 'M-051', 'M-057', 'M-048'].includes(shown.id) &&
      <Choice name="NFL team" field="team" value={s.team} options={options(research.teams)} all />}
    {shown.id === 'M-040' && <Choice name="Position group" field="group" value={s.group}
      options={options(['QB', 'RB', 'WR/TE', 'K', 'IDP', 'DST'])} />}
    {shown.id === 'M-052' && <Choice name="Activity" field="trend" value={s.trend}
      options={['Adds', 'Drops', 'Starters', 'Owned', 'Traded'].map((label, i) => ({ label,
        value: ['topAdds', 'topDrops', 'topStarters', 'topOwns', 'topTrades'][i] }))} />}
    {shown.id === 'M-048' && <Choice name="Depth chart" field="depth" value={s.depth}
      options={[{ value: 'team', label: 'By team and formation' }, 
        { value: 'grid', label: 'Starter grid' }]} />}
    {shown.id === 'M-057' && <Choice name="Injuries" field="injuries" value={s.injuries}
      options={[{ value: 'all', label: 'All NFL players' }, 
        { value: 'games', label: 'This week’s games' }]} />}
  </div>;
  return <section className={`research research-${s.depth}`} aria-label="Player research">
    <section data-endpoint="M-062" className="research-search" aria-label="Player search">
      <form onSubmit={e => {
        e.preventDefault();
        void commands.dispatch('research.read', { action: 'search', value: search });
      }}>
      <label>Find a player · name, position or NFL team
        <input className="search-input" type="search" value={search} maxLength={100}
          placeholder="Name, position or NFL team"
          onChange={e => setSearch(e.target.value)} />
      </label>
      <Act verb="research.read" args={{ action: 'search', value: search }} submit>Search players</Act>
      </form>
      {!s.search.trim() ? <p>Enter a name, position or NFL team to find a player.</p> : <>
        <p>{research.players.length} matches · open a player in the Inspector</p>
        <ul>{research.players.map(p => <li key={p.id}>
          <Act verb="inspector.open"
            args={{ subject: { kind: 'player', id: p.id, franchiseId: p.franchiseId } }}>
            {p.name} · {p.position} · {p.team} · {p.owner}
          </Act>
        </li>)}</ul>
      </>}
    </section>
    <ReportsView board={research.board} selected={[shown.id]} refreshVerb="research.read"
      controls={controls} />
    {shown.id === 'M-060' && <NewsActions research={research} mine />}
  </section>;
}
export default function Research({ newsOnly = false }: { newsOnly?: boolean }) {
  const state = researchState.use();
  const selected = commands.use().endpointIds;
  const mine = commands.use().franchiseId;
  const myNews = selected.includes('M-060') || selected.includes('M-059');
  useEffect(() => {
    let disposed = false;
    let stop: (() => void) | undefined;
    const load = async () => {
      await researchState.load();
      if (disposed) return;
      if (newsOnly) await researchState.load(false, true, true);
      else if (myNews && mine) await researchState.load(false, true);
      else if (!myNews) await researchState.load(true);
    };
    void load();
    void import('../../../wailsjs/runtime/runtime').then(m => {
      if (!disposed && typeof window !== 'undefined' && (window as { runtime?: unknown }).runtime)
        stop = m.EventsOn('target:season', () => { void load(); });
    });
    return () => { disposed = true; stop?.(); };
  }, [mine, newsOnly, myNews]);
  return <>
    {state.pending && <p role="status">
      {newsOnly ? 'Reading rostered player news…' : state.status || 'Reading held research copies'}
    </p>}
    {state.error && <p role="alert">{state.error}</p>}
    {state.research && (newsOnly ? <NewsView research={state.research} />
      : <ResearchView research={state.research} selected={selected} />)}
  </>;
}
