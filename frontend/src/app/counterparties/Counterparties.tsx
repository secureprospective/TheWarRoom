import { lazy, Suspense, useEffect } from 'react';
import { commands } from '../commands/registry';
import { Act } from '../commands/Act';
import type { Snapshot } from '../data/contract';
import { ArchiveReading } from '../market/ReadingView';
import type { CounterpartiesReading, CounterRow, CounterWords } from './contract';
import { counterState, tiers, windows, postures, type SortKey } from './state';
import { EdgeBar, Share, TeamLink, number, percent } from './Displays';
import '../market/market.css';
import './counterparties.css';
const TeamPanel = lazy(() => import('./TeamPanel'));

export function CounterControls({ snapshot }: { snapshot?: Snapshot }) {
  const s = counterState.use();
  return (
    <section className="market-filters" aria-label="Counterparties filters">
      {(
        [
          ['Standing on trade day', 'counterpart-tier', tiers, s.scope.tiers],
          ['Time of year', 'counterpart-window', windows, s.scope.windows],
          ['What the trade did', 'counterpart-posture', postures, s.scope.postures],
        ] as const
      ).map(([heading, field, choices, selected]) => (
        <div key={field}>
          <h5>{heading}</h5>
          <div className="market-controls">
            {choices.map(([value, label]) => (
              <Act variant="button"
                key={value}
                verb="market"
                args={{ field, value }}
                active={selected.includes(value)}
                label={`Filter ${label}`}
              >
                {label}
              </Act>
            ))}
          </div>
        </div>
      ))}
      <details>
        <summary>Filter by team</summary>
        <div className="market-controls">
          <Act variant="button"
            verb="market" args={{ field: 'counterpart-team', value: '' }} active={!s.scope.team}>
            All teams
          </Act>
          {s.reading?.value?.teams.map((t) => (
            <Act variant="button"
              key={t.fid}
              verb="market"
              args={{ field: 'counterpart-team', value: t.fid }}
              active={s.scope.team === t.fid}
              label={`Filter ${snapshot?.franchises.value.find((f) => f.id === t.fid)?.name ?? t.name}`}
            >
              {snapshot?.franchises.value.find((f) => f.id === t.fid)?.name ?? t.name}
            </Act>
          ))}
        </div>
      </details>
      <div className="market-controls">
        <Act variant="button" verb="market" args={{ field: 'counterpart-reset', value: '' }}>
          Reset Counterparties filters
        </Act>
        {(s.reading?.status === 'failed' ||
          Object.values(s.panels).some((r) => r.status === 'failed')) && (
          <Act variant="button" verb="market" args={{ field: 'counterpart-retry', value: '' }}>
            Retry reading
          </Act>
        )}
      </div>
      <p>Empty selections mean every standing, window and posture. Frozen seasons 2017–2026.</p>
      {s.pending && (
        <p role="status">
          Updating filters; figures still describe the previous scope until Go replies.
        </p>
      )}
    </section>
  );
}
function sortValue(row: CounterRow, key: SortKey): string | number | null {
  switch (key) {
    case 'now':
      return row.forNow.p;
    case 'later':
      return row.forLater.p;
    case 'day':
      return row.day.total ?? null;
    case 'since':
      return row.since.total ?? null;
    case 'picks':
      return row.picks.mean;
    case 'age':
      return row.age.mean;
    case 'partners':
      return row.partnersCount;
    default:
      return row[key];
  }
}
export function sortedRows(rows: CounterRow[], key: SortKey, descending: boolean) {
  return [...rows].sort((a, b) => {
    const x = sortValue(a, key),
      y = sortValue(b, key);
    if (x == null) return y == null ? 0 : 1;
    if (y == null) return -1;
    const order =
      typeof x === 'string' && typeof y === 'string' ? x.localeCompare(y) : Number(x) - Number(y);
    return descending ? -order : order;
  });
}
export function TeamTable({
  rows,
  snapshot,
  words,
}: {
  rows: CounterRow[];
  snapshot?: Snapshot;
  words: CounterWords;
}) {
  const { sort } = counterState.use();
  const columns: readonly [SortKey, string][] = [
    ['name', words.team],
    ['n', words.sides],
    ['perSeason', words.perSeason],
    ['now', words.now],
    ['later', words.later],
    ['day', words.day],
    ['since', words.since],
    ['picks', words.picks],
    ['age', words.age],
    ['partners', words.partners],
  ];
  return (
    <section data-endpoint="L-26" className="counter-team-box">
      <h5>How each team trades</h5>
      <p>
        Sort on archive values. On the day and since are totals; ranges in parentheses are per trade.
        Since needs 2+ played seasons.
      </p>
      <div className="market-controls counter-sort" role="group" aria-label="Sort by">
        <span>Sort by</span>
        {columns.map(([key, label]) => (
          <Act variant="button" key={key} verb="market" args={{ field: 'counterpart-sort', value: key }}
            active={sort.key === key} label={`Sort by ${label}`}>
            {label}{sort.key === key ? (sort.descending ? ' ↓' : ' ↑') : ''}
          </Act>
        ))}
      </div>
      <div
        className="counter-scroll counter-team-scroll"
        tabIndex={0}
        aria-label="Teams table"
      >
        <table className="counter-table counter-team-table">
          <thead className="counter-sticky-header">
            <tr>
              {columns.map(([key, label]) => (
                <th
                  key={key}
                  data-column={key}
                  className="counter-nowrap"
                  scope="col"
                  aria-sort={
                    sort.key === key ? (sort.descending ? 'descending' : 'ascending') : 'none'
                  }
                >
                  <Act variant="button"
                    verb="market"
                    args={{ field: 'counterpart-sort', value: key }}
                    label={`Sort by ${label}`}
                  >
                    {label}
                  </Act>
                </th>
              ))}

            </tr>
          </thead>
          <tbody>
            {sortedRows(rows, sort.key, sort.descending).map((r) => (
              <tr key={r.fid}>
                <th scope="row" title={r.standingTitle}>
                  <TeamLink snapshot={snapshot} fid={r.fid} name={r.name} />
                  <small className="counter-standing">{r.standingTitle}</small>
                </th>
                <td className="counter-nowrap" data-column="n">{r.nText}</td>
                <td className="counter-nowrap" data-column="perSeason">{number(r.perSeason, 1)}</td>
                <td className="counter-nowrap" data-column="now">
                  <Share words={words} share={r.forNow} label="Bought for now" />
                </td>
                <td className="counter-nowrap" data-column="later">
                  <Share words={words} share={r.forLater} label="Built for later" />
                </td>
                <td className="counter-total" data-column="day" data-tone={r.day.tone}>
                  {r.dayText}
                </td>
                <td className="counter-total" data-column="since" data-tone={r.since.tone}>
                  {r.sinceText}
                </td>
                <td className="counter-nowrap" data-column="picks">
                  {number(r.picks.mean)}
                </td>
                <td className="counter-nowrap" data-column="age">
                  {number(r.age.mean, 1)}
                </td>
                <td data-column="partners">{r.partnersText}
                  <small className="counter-partner-label">{words.partner}</small>
                  {r.partner ? (
                    <>
                      <TeamLink snapshot={snapshot} fid={r.partner.fid} name={r.partnerName} />{' '}
                      {r.partnerTrades}
                    </>
                  ) : (
                    '—'
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
export function CounterReading({
  reading,
  snapshot,
}: {
  reading?: CounterpartiesReading;
  snapshot?: Snapshot;
}) {
  const v = reading?.value;
  return (
    <ArchiveReading reading={reading} countText={v?.countText} title="Counterparties">
      {v && (
        <>
          <p>{v.asOf}</p>
          <p>{v.summary}</p>
          <p>{v.note}</p>
          <p className="counter-legend">{v.words.legend}</p>
          {reading?.status === 'ok' && (
            <>
              <p className="market-finding">{v.headline}</p>
              <section data-endpoint="L-23">
                <h5>Trading by standing</h5>
                <div className="counter-tiers">
                  {v.cards.map((c) => (
                    <section key={c.id}>
                      <h5>{c.label}</h5>
                      <p>{c.hint}</p>
                      {c.excluded ? (
                        <p>Not in the current filter</p>
                      ) : c.thin ? (
                        <p>Too few to judge · fewer than 20 trade sides</p>
                      ) : (
                        <>
                          <p>{c.perSeasonText}</p>
                          {c.posture.map((p) => (
                            <p key={p.id} title={p.title} data-tone={p.tone}>
                              {p.label}: {percent(p.p)}{' '}
                              {p.tone === 'down' ? v.words.down : p.tone === 'up' ? v.words.up : ''}
                            </p>
                          ))}
                          {c.rows.map((r) => (
                            <div className="market-row" key={r.label} data-tone={r.tone}>
                              <b>{r.label}</b>
                              <span>{r.text}</span>
                              {r.range && <EdgeBar edge={r.range} label={r.label} />}
                            </div>
                          ))}
                        </>
                      )}
                    </section>
                  ))}
                </div>
              </section>
              <section data-endpoint="L-24">
                <h5>Earlier and later seasons</h5>
                <p>A season needs 10 trade sides before its shares are shown.</p>
                {v.trendSay.map((s) => (
                  <p key={s}>{s}</p>
                ))}
                <div className="counter-trends">
                  {v.trends.tiers.map((t) => (
                    <section key={t.tier}>
                      <h5>{v.cards.find((c) => c.id === t.tier)?.label}</h5>
                      <table className="counter-table counter-trend-table">
                        <thead>
                          <tr>
                            <th scope="col">Season</th>
                            <th scope="col">Trade sides</th>
                            <th scope="col">Bought for now</th>
                            <th scope="col">Built for later</th>
                            <th scope="col">{v.words.day}</th>
                          </tr>
                        </thead>
                        <tbody>
                          {t.rows.map((r) => (
                            <tr key={r.season}>
                              <th scope="row">{r.season}</th>
                              <td className="counter-nowrap" data-column="n">{r.nText}</td>
                              <td>
                                {r.showShares ? (
                                  <Share words={v.words} share={r.now} label="Bought for now" />
                                ) : (
                                  'Too few'
                                )}
                              </td>
                              <td>
                                {r.showShares ? (
                                  <Share words={v.words} share={r.later} label="Built for later" />
                                ) : (
                                  'Too few'
                                )}
                              </td>
                              <td>{number(r.net)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </section>
                  ))}
                </div>
              </section>
              <section data-endpoint="L-25">
                <h5>Standing against standing</h5>
                <p>
                  Read each row's gains against the column's tier. Each trade supplies two sides.
                </p>
                <div className="counter-scroll">
                  <table className="counter-table counter-matrix">
                    <thead>
                      <tr>
                        <th scope="col">Trading as</th>
                        {v.picks.map((pick) => (
                          <th key={pick.id} scope="col">{pick.with}</th>
                        ))}
                      </tr>
                    </thead>
                    <tbody>
                      {v.picks.map((pick, index) => (
                        <tr key={pick.id}>
                          <th scope="row">{pick.tradingAs}</th>
                          {v.pairs.slice(index * 3, index * 3 + 3).map((p) => (
                            <td key={p.other} title={p.title}>
                              <p>{p.trades}</p>
                              <EdgeBar edge={p.now} label={v.words.day} />
                              <EdgeBar edge={p.real} label={v.words.since} />
                            </td>
                          ))}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </section>
              <TeamTable rows={v.rows} snapshot={snapshot} words={v.words} />
            </>
          )}
        </>
      )}
    </ArchiveReading>
  );
}
export function panelEndpoint(ids: readonly string[]) {
  return ids.some((id) => ['L-27', 'L-28', 'L-29'].includes(id));
}
export default function Counterparties({ fid, snapshot }: { fid?: string; snapshot?: Snapshot }) {
  const s = counterState.use();
  const shell = commands.use();
  const wantsPanel = panelEndpoint(shell.endpointIds);
  useEffect(() => {
    if (
      !fid &&
      wantsPanel &&
      !shell.asset?.startsWith('team:') &&
      shell.franchiseId &&
      s.reading?.value?.teams.some((t) => t.fid === shell.franchiseId)
    ) {
      commands.dispatch('market', { field: 'inspect', value: `team:${shell.franchiseId}` });
    }
  }, [fid, wantsPanel, shell.asset, shell.franchiseId, s.reading]);
  useEffect(() => {
    if (!s.reading && !s.pending) void counterState.refresh();
  }, []);
  if (fid)
    return (
      <Suspense fallback={<p role="status">Opening team history…</p>}>
        <TeamPanel fid={fid} snapshot={snapshot} />
      </Suspense>
    );
  return (
    <div className="market counterparties">
      <CounterControls snapshot={snapshot} />
      {wantsPanel && !shell.asset?.startsWith('team:') && (
        <p role="status">
          Choose a team below to open its seasons, asset flows and partners in the Inspector.
        </p>
      )}
      <CounterReading reading={s.reading} snapshot={snapshot} />
    </div>
  );
}
