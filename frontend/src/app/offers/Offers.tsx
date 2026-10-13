import { localDate } from '../clock/labels';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import { ArchiveReading } from '../market/ReadingView';
import { type OfferReading, type Scope } from './provider';
import { offerState } from './state';
import '../market/market.css';
import './offers.css';

export function OffersScreen({ reading, scope, pending = false }: {
  reading: OfferReading; scope: Scope; pending?: boolean;
}) {
  const v = reading.value!;
  const localText = (text: string) => text.replace(
    /\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})/g,
    at => localDate(at),
  );
  const displayed = { ...reading, cutoff: localText(reading.cutoff),
    buildStamp: localText(reading.buildStamp), howCounted: localText(reading.howCounted) };
  return <section className="offers-history" aria-label={v.words.heading}>
    <header data-endpoint="L-32"><h4>{v.words.heading}</h4><p>{localText(v.disclosure)}</p></header>
    <div className="market-controls" data-endpoint="L-31" aria-label={v.words.direction}>
      {v.directions.map(p => <Act variant="button" key={p.id} verb="offers.filter"
        args={{ field: 'direction', value: p.id }} active={scope.direction === p.id} label={p.label} />)}
    </div>
    <div className="offers-filters">
      <label>{v.words.seasons} <select value={`${scope.from}:${scope.to}`} onChange={e => {
        const p = v.ranges.find(p => `${p.from}:${p.to}` === e.target.value)!;
        commands.dispatch('offers.filter', { field: 'range', value: p.id });
      }}>{v.ranges.map(p => (
        <option key={p.id} value={`${p.from}:${p.to}`}>{p.label}</option>
      ))}</select></label>
      <label>{v.words.window} <select value={scope.window} onChange={e =>
        commands.dispatch('offers.filter', { field: 'window', value: e.target.value })}>
        {v.windows.map(p => <option key={p.id} value={p.id}>{p.label}</option>)}</select></label>
      <label>{v.words.team} <select value={scope.team} onChange={e =>
        commands.dispatch('offers.filter', { field: 'team', value: e.target.value })}>
        {v.teams.map(p => <option key={p.fid} value={p.fid}>{p.name}</option>)}</select></label>
      {reading.status === 'failed' && <Act variant="button" verb="offers.filter"
        args={{ field: 'retry', value: '' }} label={v.words.retry} />}
    </div>
    {pending && <p role="status">{v.words.pending}</p>}
    <div data-endpoint="L-30"><ArchiveReading reading={displayed} title={v.title} countText={v.countText}>
      <p>{v.summary}</p>
      <ul className="offers-legend" aria-label="Offer endings">
        {v.stretches[0]?.outcomes.map(o => <li key={o.id}>
          <span className="offer-bar" data-outcome={o.id} aria-hidden="true" />{o.label}
        </li>)}
      </ul>
      <div className="offers-stretches">{v.stretches.map(s => <section key={s.id} aria-label={s.label}>
        <h5>{s.label}</h5><p>{s.countText}</p><p>{s.acceptance}</p>
        {s.finding && <p>{s.finding}</p>}
        <div className="offer-bar-track" aria-hidden="true">
          {s.outcomes.map(o => <span key={o.id} className="offer-bar"
            data-outcome={o.id} style={{ width: `${o.width}%` }} />)}
        </div>
        <p className="offer-counts">{s.outcomes.map((o, i) => <span key={o.id}>
          {i > 0 && ' · '}<span className="offer-count">{o.text}</span>
        </span>)}</p>
      </section>)}</div>
    </ArchiveReading></div>
  </section>;
}
export default function OffersHistory() {
  offerState.first();
  const s = offerState.use();
  // first() suspends until a reading exists; a server render sees the initial store.
  if (!s.reading) return null;
  return <OffersScreen reading={s.reading} scope={s.scope} pending={s.pending} />;
}
