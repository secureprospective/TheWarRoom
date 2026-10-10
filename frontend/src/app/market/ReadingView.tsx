import type { ReactNode } from 'react';
import { Act } from '../commands/Act';
import type { MarketRange, ReadingShell, SurfaceReading } from './contract';

export function countNoun(count: number, noun: string): string {
  return `${count} ${noun}${count === 1 ? '' : 's'}`;
}

export function RangeBar({ range }: { range: MarketRange }) {
  const x = (v: number) =>
    12 + Math.max(0, Math.min(1, Math.log(Math.max(0.1, v) / 0.1) / Math.log(100))) * 276;
  return (
    <svg className="market-range" viewBox="0 0 300 26" role="img" aria-label={range.text}>
      <line className="axis" x1="12" x2="288" y1="13" y2="13" />
      <line className="reference" x1={x(1)} x2={x(1)} y1="2" y2="24" />
      <line className="range" x1={x(range.lo)} x2={x(range.hi)} y1="13" y2="13" />
      <circle cx={x(range.value)} cy="13" r="4" />
    </svg>
  );
}
export function HowCounted({ reading, countText }: { reading: ReadingShell; countText?: string }) {
  return (
    <details className="market-how" data-endpoint="L-33">
      <summary>How this is counted</summary>
      <p>
        This is the league’s trade and draft history through the archive cutoff, not today’s live
        market.
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
}: {
  reading?: ReadingShell;
  title: string;
  children?: ReactNode;
  countText?: string;
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
      <HowCounted reading={reading} countText={countText} />
    </section>
  );
}
export function ReadingView({
  reading,
  title,
  inspect = false,
}: {
  reading?: SurfaceReading;
  title: string;
  inspect?: boolean;
}) {
  return (
    <ArchiveReading reading={reading} title={title}>
      {reading?.value && (
        <>
          <p className="market-finding">{reading.value.summary}</p>
          <div className="market-sections">
            {reading.value.sections.map((section) => (
              <section key={section.id} data-endpoint={section.id}>
                <h5>{section.title}</h5>
                <p>{section.note}</p>
                {section.rows.length === 0 && <p>No asset in view clears chance on this side.</p>}
                {section.rows.map((row, index) => (
                  <div
                    className="market-row"
                    key={`${row.key}:${index}`}
                    data-mark={row.mark || undefined}
                  >
                    {inspect && row.key ? (
                      <Act
                        verb="market"
                        args={{ field: 'inspect', value: row.key }}
                        label={`Inspect ${row.label}`}
                      >
                        {row.label}
                      </Act>
                    ) : (
                      <b>{row.label}</b>
                    )}
                    {row.range && (
                      <>
                        <span>{row.range.text}</span>
                        <RangeBar range={row.range} />
                      </>
                    )}
                    <small>
                      {row.detail}
                      {row.count > 0 ? ` · ${countNoun(row.count, 'record')}` : ''}
                    </small>
                  </div>
                ))}
              </section>
            ))}
          </div>
        </>
      )}
    </ArchiveReading>
  );
}
