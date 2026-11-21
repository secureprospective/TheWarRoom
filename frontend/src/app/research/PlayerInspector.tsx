import { useEffect, useState } from 'react';
import { type PlayerReading } from './contract';
import { Act } from '../commands/Act';
import type { HostedPort } from '../commissioner/hostedRoom';
import { hostedPlayerReadingUnavailable } from '../control/nativeApi';
import { ReportView } from '../reports/ReportView';
import { parseUsage, type Usage } from '../shell/hqPlayerData';
import { researchState } from './state';
import type { Research as ResearchBoard } from './contract';
import './research.css';
import { readPlayerUsage, researchAPI, researchAvailable, subscribeResearchPlayer } from './api';

export default function ResearchPlayerInspector({
  id, identity = true, league, port, position, usage: suppliedUsage, usageDone,
}: {
  id: string; identity?: boolean; league?: string; port?: HostedPort; position?: string;
  usage?: Usage; usageDone?: boolean;
}) {
  const [player, setPlayer] = useState<PlayerReading>();
  const [error, setError] = useState('');
  const [fetchedUsage, setFetchedUsage] = useState<{ ready: boolean; value?: Usage }>({ ready: false });
  const hostedOnly = Boolean(league) && !researchAvailable();
  useEffect(() => {
    let disposed = false;
    setPlayer(undefined);
    setError('');
    if (!researchAvailable()) {
      if (!league) setError('Player research requires the desktop app.');
      return;
    }
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
  }, [id, league]);
  useEffect(() => {
    if (!hostedOnly || usageDone !== undefined) return;
    let disposed = false;
    setFetchedUsage({ ready: false });
    void readPlayerUsage(id, league, port).then(raw => {
      if (disposed) return;
      const value = hostedPlayerReadingUnavailable(raw) ? undefined : parseUsage(raw);
      setFetchedUsage({ ready: true, value });
    }).catch(() => { if (!disposed) setFetchedUsage({ ready: true }); });
    return () => { disposed = true; };
  }, [hostedOnly, id, league, port, usageDone]);
  if (hostedOnly) {
    const usage = usageDone !== undefined ? suppliedUsage : fetchedUsage.value;
    const ready = usageDone ?? fetchedUsage.ready;
    return <section aria-label="Research player Inspector">
      <HostedResearchView usage={usage} ready={ready} position={position} />
    </section>;
  }
  return <section aria-label="Research player Inspector">
    {error && <p role="alert">{error}</p>}
    {!player && !error && <p role="status">Reading held player reports…</p>}
    {player && <PlayerResearchView player={player} identity={identity} />}
  </section>;
}

function HostedResearchView({ usage, ready, position }: {
  usage?: Usage; ready: boolean; position?: string;
}) {
  const shareLabel = position === 'RB' ? 'Carry share, last 4'
    : ['QB', 'DT', 'DE', 'LB', 'CB', 'S'].includes(position ?? '') ? 'Snap share, last 4'
      : 'Target share, last 4';
  const share = position === 'RB' ? usage?.carryShare
    : ['QB', 'DT', 'DE', 'LB', 'CB', 'S'].includes(position ?? '') ? usage?.snapShare
      : usage?.targetShare;
  return <div className="draft-reports">
    <section className="research-facts" aria-label="Player news">
      <h5>Player news</h5>
      <p>Not available</p>
    </section>
    <section className="research-facts" aria-label="Player research facts">
      <h5>Research facts</h5>
      {!ready ? <p role="status">Reading player usage…</p> : !usage ? <p>Not available</p> : <dl>
        <div><dt>Opponent</dt><dd>{usage.opponent || 'Not available'}</dd></div>
        <div><dt>Matchup vs {position ?? '—'}</dt><dd>{usage.label && usage.allowedRank !== undefined
          ? `${usage.label} · #${usage.allowedRank}` : 'Not available'}</dd></div>
        <div><dt>{shareLabel}</dt><dd>{share !== undefined
          ? `${(share * 100).toFixed(0)}%` : 'Not available'}
          <small>{usage.weeksUsed} weeks used · {usage.reason}</small></dd></div>
      </dl>}
    </section>
  </div>;
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

export function NewsList({ report, articles, howLabel, pageKey = report.id }: {
  report: ResearchBoard['news']; articles: ResearchBoard['articles']; howLabel: string; pageKey?: string;
}) {
  const limit = researchState.use().pages[pageKey] ?? 25;
  const visible = { ...report, rows: report.rows.slice(0, limit) };
  const headline = report.columns.indexOf('Headline');
  return <>
    <ReportView report={visible} howLabel={howLabel} heading="h4"
      renderCell={(cell, row, column) => column === headline && articles[row] ?
        <Act variant="text" verb="research.read" args={{ action: 'open',
          value: `${articles[row].id}:${articles[row].article}` }}
          label={`Open article on MFL · ${cell}`}>{cell}</Act> : cell} />
    {report.rows.length > limit && <Act variant="button" verb="research.read"
      args={{ action: 'more', value: pageKey }}
      label="Show 25 more">Show 25 more</Act>}
  </>;
}
