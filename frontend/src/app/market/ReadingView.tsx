import type { ReactNode } from 'react';
import { Act } from '../commands/Act';
import type { MarketLine, MarketSection, MarketRange, ReadingShell, SurfaceReading } from './contract';

export function countNoun(count: number, noun: string): string {
  return `${count} ${noun}${count === 1 ? '' : 's'}`;
}

export function RangeBar({ range }: { range: MarketRange }) {
  const x = (v: number) =>
    12 + (range.max ? Math.max(0, Math.min(1, v / range.max)) :
      Math.max(0, Math.min(1, Math.log(Math.max(0.1, v) / 0.1) / Math.log(100)))) * 276;
  return (
    <svg className="market-range" viewBox="0 0 300 26" role="img" aria-label={range.text}>
      <line className="axis" x1="12" x2="288" y1="13" y2="13" />
      {!range.max && <line className="reference" x1={x(1)} x2={x(1)} y1="2" y2="24" />}
      <line className="range" x1={x(range.lo)} x2={x(range.hi)} y1="13" y2="13" />
      <circle cx={x(range.value)} cy="13" r="4" />
    </svg>
  );
}
export function HowCounted({ reading, countText, observed = false }: {
  reading: ReadingShell; countText?: string; observed?: boolean;
}) {
  return (
    <details className="market-how" data-endpoint="L-33">
      <summary>How this is counted</summary>
      <p>
        {observed
          ? 'The source dates show which history is available, not the whole live market.'
          : 'This is frozen archive history through the cutoff, not today’s live market.'}
        The scope and counting method below define which records are visible.
      </p>
      <p>
        {reading.howCounted || 'No archive reading is available; no numbers have been supplied.'}
      </p>
      <dl className="kv">
        <dt>Count / threshold</dt>
        <dd>
          {countText ?? reading.count} / {reading.threshold}
        </dd>
        <dt>Baseline</dt>
        <dd>{reading.baseline || 'Unavailable'}</dd>
        <dt>Archive cutoff</dt>
        <dd>{reading.cutoff || 'Unavailable'}</dd>
        <dt>Method</dt>
        <dd>{reading.method || 'Unavailable'}</dd>
        <dt>Source</dt>
        <dd>{reading.provenance?.source || 'Unavailable'}</dd>
        <dt>Input SHA-256</dt>
        <dd>{reading.inputSHA || 'Unavailable'}</dd>
        <dt>Build</dt>
        <dd>{reading.buildStamp || 'Unavailable'}</dd>
      </dl>
    </details>
  );
}
export function ArchiveReading({
  reading,
  title,
  children,
  countText,
  observed = false,
}: {
  reading?: ReadingShell;
  title: string;
  children?: ReactNode;
  countText?: string;
  observed?: boolean;
}) {
  if (!reading)
    return (
      <section aria-label={title}>
        <h4>{title}</h4>
        <p>Archive reading requested. No figures yet.</p>
      </section>
    );
  return (
    <section className="market-reading" aria-label={title}>
      <h4>{title}</h4>
      {reading.status !== 'ok' && (
        <p role={reading.status === 'failed' ? 'alert' : 'status'}>
          {reading.status === 'thin'
            ? 'Too few to judge'
            : reading.status === 'not_wired'
              ? 'Not wired'
              : 'Failed'}{' '}
          · {reading.absenceReason} ·{' '}
          {countText
            ? `${countText} record${reading.count === 1 ? '' : 's'}`
            : countNoun(reading.count, 'record')}{' '}
          / {reading.threshold} required
        </p>
      )}
      {(reading.status === 'ok' || reading.status === 'thin') && children}
      <HowCounted reading={reading} countText={countText} observed={observed} />
    </section>
  );
}
export function ReadingView({
  reading,
  title,
  inspect = false,
  disclose = false,
}: {
  reading?: SurfaceReading;
  title: string;
  inspect?: boolean;
  disclose?: boolean;
}) {
  return (
    <ArchiveReading reading={reading} title={title}>
      {reading?.value && (
        <>
          <p className="market-finding">{reading.value.summary}</p>
          <div className="market-sections">
            {reading.value.sections.map((section) => (
              <SurfaceSection key={section.id} section={section} inspect={inspect} disclose={disclose} />
            ))}
          </div>
        </>
      )}
    </ArchiveReading>
  );
}

function SurfaceRows({ rows, inspect }: { rows: MarketLine[]; inspect: boolean }) {
  return <>
    {rows.length === 0 && <p>No asset in view clears chance on this side.</p>}
    {rows.map((row, index) => (
      <div className="market-row" key={`${row.key}:${index}`} data-mark={row.mark || undefined}>
        {inspect && row.key ? (
          <Act variant="text" verb="market" args={{ field: 'inspect', value: row.key }}
            label={`Inspect ${row.label}`}>{row.label}</Act>
        ) : <b>{row.label}</b>}
        {row.range && <><span>{row.range.text}</span><RangeBar range={row.range} /></>}
        <small>{row.detail}{row.count > 0 ? ` · ${countNoun(row.count, 'record')}` : ''}</small>
      </div>
    ))}
  </>;
}
function SurfaceSection({ section, inspect, disclose }: {
  section: MarketSection; inspect: boolean; disclose: boolean;
}) {
  const content = <>
    <p>{section.note}</p>
    <SurfaceRows rows={section.rows} inspect={inspect} />
    {section.breakdowns?.map(b => (
      <details className="market-breakdown" key={b.title}>
        <summary>{b.title} · {countNoun(b.count, 'judged pick')}</summary>
        <SurfaceRows rows={b.rows} inspect={inspect} />
      </details>
    ))}
  </>;
  return disclose && !section.breakdowns?.length ? (
    <details className="market-section" data-endpoint={section.id} tabIndex={-1}>
      <summary>{section.title}</summary>{content}
    </details>
  ) : (
    <section data-endpoint={section.id} tabIndex={-1}>
      <h5>{section.title}</h5>{content}
    </section>
  );
}
