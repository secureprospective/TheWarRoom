import { useEffect } from 'react';
import { Act } from '../commands/Act';
import type { Report } from '../reports/contract';
import { controlState } from './state';
import './control.css';

// NewsArticles reads MFL's site news as articles: headline and date, then MFL's own paragraphs.
// The newest opens; older articles open on demand.
export function NewsArticles({ news }: { news: Report }) {
  return (
    <section className="site-news" data-endpoint={news.id} aria-label={news.title}>
      <h4>{news.title}</h4>
      <p className="report-source">{news.source} · {news.asOf}</p>
      {news.note && <p role={news.status === 'failed' ? 'alert' : 'status'}>{news.note}</p>}
      {news.rows.map((row, i) => (
        <details key={i} open={i === 0}>
          <summary><span>{row.group}</span> <time>{row.cells[0]}</time></summary>
          {row.cells[1].split('\n').map((line, k) => <p key={k}>{line}</p>)}
        </details>
      ))}
      <details className="market-how"><summary>Where this comes from</summary><p>{news.how}</p></details>
    </section>
  );
}

// WhatsNew is MFL's site news in Control Room › App (M-141).
export function WhatsNew() {
  const s = controlState.use();
  useEffect(() => { void controlState.choose('news'); }, []);
  return (
    <section className="control-room" aria-busy={s.pending}>
      {s.error && <p role="alert">{s.error}</p>}
      {s.news && <NewsArticles news={s.news} />}
      <Act verb="control.read" args={{ action: 'news', value: '' }}>Refresh from MFL</Act>
    </section>
  );
}
