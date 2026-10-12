import { useEffect, useState } from 'react';
import { parsePlayerReading, type PlayerReading } from './contract';
import { Act } from '../commands/Act';
import './research.css';
import type * as Bindings from '../../../wailsjs/go/main/App';

export default function ResearchPlayerInspector({ id, identity = true }: { id: string; identity?: boolean }) {
  const [player, setPlayer] = useState<PlayerReading>();
  const [error, setError] = useState('');
  useEffect(() => {
    let disposed = false;
    setPlayer(undefined);
    setError('');
    const app = (window as unknown as { go?: { main: { App: typeof Bindings } } }).go?.main.App;
    if (!app) { setError('Player research requires the desktop app.'); return; }
    void app.TargetResearchPlayer(id).then(value => {
      if (!disposed) setPlayer(parsePlayerReading(value));
    }).catch(cause => { if (!disposed) setError(String(cause)); });
    return () => { disposed = true; };
  }, [id]);
  return <section aria-label="Research player Inspector">
    {error && <p role="alert">{error}</p>}
    {!player && !error && <p role="status">Reading held player reports…</p>}
    {player && <PlayerResearchView player={player} identity={identity} />}
  </section>;
}
export function PlayerResearchView({ player, identity }: { player: PlayerReading; identity: boolean }) {
  return <div className="draft-reports">
    {identity && <><h4>{player.name}</h4><p>{player.position} · {player.team} · {player.owner}</p></>}
    <section className="research-facts" data-endpoint={player.report.id} aria-label={player.report.title}>
      <h5>{player.report.title}</h5>
      <p>{player.report.source}</p>
      <dl>{player.report.rows.map((row, i) => <div key={i}>
        <dt>{row.cells[0]}</dt>
        <dd>{row.cells[1]}<small>{row.cells[2]}</small></dd>
      </div>)}</dl>
      <details><summary>{player.howLabel}</summary><p>{player.report.how}</p></details>
    </section>
    <ul>{player.articles.map(a => <li key={a.article}>
      <Act verb="research.read" args={{ action: 'open', value: `${a.id}:${a.article}` }}
        label={`Open article on MFL · ${a.headline}`}>Open article on MFL · {a.headline}</Act>
    </li>)}</ul>
  </div>;
}
