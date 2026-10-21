import { useEffect, useState } from 'react';
import { type PlayerReading } from './contract';
import { NewsList } from './Research';
import './research.css';
import { researchAPI, researchAvailable, subscribeResearchPlayer } from './api';

export default function ResearchPlayerInspector({ id, identity = true }: { id: string; identity?: boolean }) {
  const [player, setPlayer] = useState<PlayerReading>();
  const [error, setError] = useState('');
  useEffect(() => {
    let disposed = false;
    setPlayer(undefined);
    setError('');
    if (!researchAvailable()) { setError('Player research requires the desktop app.'); return; }
    let revision = 0;
    const read = () => {
      const request = ++revision;
      void researchAPI.player!(id).then(value => {
      if (!disposed && request === revision) setPlayer(value);
      }).catch(cause => { if (!disposed && request === revision) setError(String(cause)); });
    };
    read();
    const stop = subscribeResearchPlayer(read);
    return () => { disposed = true; stop(); };
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
      <dl>{player.report.rows.filter(row => !row.cells[0].startsWith('News · '))
        .map((row, i) => <div key={i}>
        <dt>{row.cells[0]}</dt>
        <dd>{row.cells[1]}<small>{row.cells[2]}</small></dd>
      </div>)}</dl>
      <details><summary>{player.howLabel}</summary><p>{player.report.how}</p></details>
    </section>
    {player.articles.length > 0 && <NewsList pageKey={`player:${player.id}`}
      report={{ ...player.report, id: 'M-058', title: 'Player news',
        columns: ['Posted', 'Headline', 'Feed time'], summary: [],
        rows: player.report.rows.filter(row => row.cells[0].startsWith('News · '))
          .map(row => ({ ...row, cells: [row.cells[0].slice(7), ...row.cells.slice(1)] })) }}
      articles={player.articles} howLabel={player.howLabel} />}

  </div>;
}
