import { WatchButton } from '../personal/WatchButton';
import { scrollWorkIntoView, useWorkAnchor } from '../shell/workScroll';
import { Fragment, lazy, Suspense, useEffect, useRef } from 'react';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { localDate } from '../clock/labels';
import type { Snapshot } from '../data/contract';
import { FranchisePicker } from '../shell/FranchisePicker';
import { BidPlanCard, signals, usePoolData } from './BidPlans';
import { BidForm } from './BidForm';
import { SignalChip } from '../look/Slots';
import {
  planOpen, type BoardReading, type Plan, type PoolFilter, type PoolReading, type PoolRow,
} from './contract';
import { poolState, type PoolState } from './state';
import './pool.css';
import { RangeBar } from '../market/ReadingView';
const MarketReports = lazy(() => import('../reports/MarketReports'));

export function MarketFacts({ row }: { row: PoolRow }) {
  return <>{row.market?.map(f => <div key={f.label} title={f.reason}>
    <dt>{f.label}</dt><dd>{f.text}{f.range && <RangeBar range={f.range} />}</dd>
  </div>)}</>;
}

function PlayerCell({ row }: { row: PoolRow }) {
  return (
    <>
      <Act
        verb="market"
        args={{ field: 'inspect', value: `pool:${row.id}` }}
        label={`Inspect ${row.name}`}
      >
        {row.name}
      </Act>
      <span className="pool-pos">
        {row.position} · {row.team}
      </span>
    </>
  );
}

// Scrolls its card into view each time the screen asks for it (after a save, or on a click).
function useBringIntoView(wanted: boolean, seq: number) {
  const ref = useRef<HTMLDivElement>(null);
  const done = useRef(0);
  useEffect(() => {
    if (!wanted || done.current === seq) return;
    done.current = seq;
    if (ref.current) scrollWorkIntoView(ref.current);
  }, [wanted, seq]);
  return ref;
}

function OpenCard({ plan, focus, busy, state }: {
  plan: Plan; focus: PoolState['focus']; busy: string; state?: PoolState;
}) {
  const ref = useBringIntoView(focus.id === plan.id, focus.seq);
  return (
    <div ref={ref} className="bid-open-card">
      <BidPlanCard plan={plan} busy={busy} />
      {state?.form?.planId === plan.id &&
        <BidForm form={state.form} preview={state.preview} busy={busy} />}
    </div>
  );
}

// The owner's open plans sit above the 800-row table: one line each, and a click opens the card.
export function OpenPlans({ board, openId, focus, busy, state }: {
  board?: BoardReading;
  openId: string;
  focus: PoolState['focus'];
  busy: string;
  state?: PoolState;
}) {
  const all = board?.value.plans ?? [];
  const open = all.filter((p) => planOpen(p.stateCode));
  const chosen = open.find((p) => p.id === openId);
  return (
    <section className="bid-open" aria-label="Your open bid plans" data-endpoint="M-194">
      <h4>Your open bid plans</h4>
      {board?.status === 'ok' && open.length === 0 && (
        <p>No open bid plans. Press Plan a bid on a player below.</p>
      )}
      <ul className="bid-open-list">
        {open.map((plan) => (
          <li key={plan.id}>
            <Act
              verb="market"
              args={{ field: 'pool-toggle', value: plan.id }}
              label={`Open bid plan: ${plan.player}`}
              expanded={plan.id === openId}
            >
              <b>{plan.player}</b>
              <span>{plan.contract}</span>
              <span>{plan.points} points</span>
              <SignalChip signal={signals[plan.stateCode] ?? 'grey'} label={plan.state} />
            </Act>
          </li>
        ))}
      </ul>
      {chosen && <OpenCard plan={chosen} focus={focus} busy={busy} state={state} />}
    </section>
  );
}

function PoolTable({ rows, state, mine }: { rows: PoolRow[]; state?: PoolState; mine: string }) {
  return (
    <table className="pool-table" aria-label="Free agents">
      <thead>
        <tr>
          <th>Player</th>
          <th>Age</th>
          <th>Experience</th>
          <th>Minimum</th>
          <th>Market</th>
          <th>
            <span className="pool-sr">Bid</span>
          </th>
        </tr>
      </thead>
      <tbody>
        {rows.map((row) => (
          <Fragment key={row.id}><tr>
            <td>
              <PlayerCell row={row} />
              <WatchButton id={row.id} name={row.name} mine={mine} />
            </td>
            <td>{row.age}</td>
            <td>{row.experience}</td>
            <td className="pool-money">{row.minimum.text}</td>
            <td className={row.marketLines.length === 1 ? 'pool-market pool-market-none' : 'pool-market'}>
              {row.marketLines.map((line, i) =>
              <div key={i}>{line}</div>)}</td>
            <td>
              {row.plan && <span className="pool-plan">Bid plan: {row.plan}</span>}
              {row.planId ? (
                <Act
                  verb="market"
                  args={{ field: 'pool-open', value: row.planId }}
                  label={`Open bid plan on ${row.name}`}
                >
                  Open bid plan
                </Act>
              ) : (
                <Act
                  verb="market"
                  args={{ field: 'pool-plan', value: row.id }}
                  label={`Plan a bid on ${row.name}`}
                >
                  Plan a bid
                </Act>
              )}
            </td>
          </tr>
          {state?.form && !state.form.planId && state.form.playerId === row.id &&
            <tr className="pool-form-row"><td colSpan={6}>
            <BidForm form={state.form} preview={state.preview} busy={state.busy} />
          </td></tr>}</Fragment>
        ))}
      </tbody>
    </table>
  );
}

export function PoolHowCounted({ reading }: { reading: PoolReading }) {
  return (
    <details>
      <summary>How this is counted</summary>
      <p>{reading.howCounted}</p>
      <p>
        Method: {reading.method} · Build: {reading.buildStamp}
      </p>
    </details>
  );
}

export function PoolFrame({ reading, refreshing, state, mine = '' }: {
  reading?: PoolReading; refreshing: boolean; state?: PoolState; mine?: string;
}) {
  const failed = reading?.status === 'failed';
  return (
    <section className="pool-reading" aria-label="Free agents from MFL" data-endpoint="M-056">
      {!reading && <p role="status">Reading the free-agent list…</p>}
      {reading && failed && refreshing && <p role="status">Asking MFL for the free-agent list…</p>}
      {reading && failed && !refreshing && (
        <>
          <p role="alert">Failed · {reading.absenceReason}</p>
          <Act verb="market" args={{ field: 'pool-refresh', value: '' }} label="Try MFL again">
            Try MFL again
          </Act>
        </>
      )}
      {reading?.status === 'thin' && <p role="status">Empty · {reading.absenceReason}</p>}
      {reading?.status === 'not_wired' && <p role="status">Not wired · {reading.absenceReason}</p>}
      {reading?.status === 'ok' && (
        <>
          <p>{reading.value.summary}</p>
          {reading.value.rows.length === 0 ? (
            <p>No free agent matches this position.</p>
          ) : (
            <div className="pool-table-scroll" tabIndex={0} role="region" aria-label="Free agents">
              <PoolTable rows={reading.value.rows} state={state} mine={mine} />
            </div>
          )}
        </>
      )}
      {reading && (
        <>
          <p className="pool-source">
            Source: {reading.provenance.source} · As of:{' '}
            {reading.provenance.freshness.fetchedAt
              ? localDate(reading.provenance.freshness.fetchedAt)
              : 'Not recorded'}
            {reading.provenance.freshness.state === 'stale' &&
              ` · Stale: ${reading.provenance.freshness.note}`}
          </p>
          <PoolHowCounted reading={reading} />
        </>
      )}
    </section>
  );
}

function Controls({ reading, state }: { reading?: PoolReading; state: PoolState }) {
  const { scope } = state;
  const dispatch = commands.dispatch;
  return (
    <div className="pool-controls" aria-label="Pool filter and sort" data-endpoint="M-039">
      <label>
        Position
        <select
          value={scope.position}
          onChange={(e) => dispatch('market', { field: 'pool-position', value: e.target.value })}
        >
          <option value="">All positions</option>
          {reading?.value.positions.map((p) => (
            <option key={p} value={p}>
              {p}
            </option>
          ))}
        </select>
      </label>
      <label>
        Sort
        <select
          value={scope.sort}
          onChange={(e) => dispatch('market', { field: 'pool-sort', value: e.target.value })}
        >
          {reading?.value.sorts.map(({ key: id, label }) => (
            <option key={id} value={id}>
              {label}
            </option>
          ))}
        </select>
      </label>
      <Act
        verb="market"
        args={{ field: 'pool-sort', value: scope.sort }}
        label={scope.desc ? 'Sort ascending' : 'Sort descending'}
      >
        {scope.desc ? 'Descending' : 'Ascending'}
      </Act>
      <Act
        verb="market"
        args={{ field: 'pool-refresh', value: '' }}
        label="Refresh the free-agent list from MFL"
        disabled={state.refreshing}
      >
        {state.refreshing ? 'Asking MFL…' : 'Refresh from MFL'}
      </Act>
    </div>
  );
}

export function PoolScreen({ state, snapshot, franchiseId }: {
  state: PoolState;
  snapshot?: Snapshot;
  franchiseId: string;
}) {
  const root = useRef<HTMLElement>(null);
  useWorkAnchor(root, '.pool-table tbody tr:not(.pool-form-row), .bid-plan');
  // The market reports are navigation, not pool rows: they sit outside the anchored section so the
  // pool's scroll anchor never undoes endpoint.open revealing the chosen report.
  return (<>
    <section ref={root} className="pool" aria-label="Free-agent pool">
      <header>
        <h4>Free-agent pool</h4>
        <p>
          Every free agent MFL reports, with the rulebook minimum salary for his experience. A bid plan
          is yours alone: the app copies the ProBoards post text and watches the rosters for the result.
        </p>
      </header>
      {!franchiseId && snapshot && <FranchisePicker snapshot={snapshot} />}
      {state.error && <p role="alert">{state.error}</p>}
      {state.notice && <p role="status">{state.notice}</p>}
      {state.form && !state.form.planId &&
        !state.reading?.value.rows.some(row => row.id === state.form?.playerId) &&
        <BidForm form={state.form} preview={state.preview} busy={state.busy} />}
      <OpenPlans board={state.board} openId={state.openPlan} focus={state.focus}
        busy={state.busy} state={state} />
      <Controls reading={state.reading} state={state} />
      <PoolFrame reading={state.reading} refreshing={state.refreshing} state={state} mine={franchiseId} />
    </section>
    <Suspense fallback={null}><MarketReports /></Suspense>
  </>);
}

export default function Pool({ snapshot }: { snapshot?: Snapshot }) {
  const state = usePoolData(true);
  const franchiseId = commands.use().franchiseId ?? '';
  return <PoolScreen state={state} snapshot={snapshot} franchiseId={franchiseId} />;
}

export function PoolInspectorView({ row, plans, busy, reading }: {
  row?: PoolRow;
  plans: Plan[];
  busy: string;
  reading?: PoolReading;
}) {
  return (
    <section className="pool-inspector" aria-label="Free agent" data-endpoint="M-056">
      <h5>{row?.name ?? 'Free agent'}</h5>
      {!row && <p role="status">Reading the free-agent list…</p>}
      {row && (
        <dl className="kv">
          <dt>Position · team</dt>
          <dd>
            {row.position} · {row.team}
          </dd>
          <dt>Age</dt>
          <dd>{row.age}</dd>
          <dt>Experience</dt>
          <dd>{row.experience}</dd>
          <dt>Minimum salary</dt>
          <dd className="pool-money">{row.minimum.text}</dd>
          <MarketFacts row={row} />
        </dl>
      )}
      {row && reading && <PoolHowCounted reading={reading} />}
      {plans.length === 0 && <p>No open bid plan on this player.</p>}
      {plans.map((plan) => (
        <article key={plan.id} className="bid-plan" aria-label={`Bid plan ${plan.player}`}>
          <h5>{plan.player}</h5>
          <p>{plan.state} · <time dateTime={plan.history.at(-1)?.at}>{plan.history.at(-1)?.at
            ? localDate(plan.history.at(-1)!.at) : 'Time not recorded'}</time></p>
          <p>{plan.contract}</p><Act verb="market" args={{ field: 'pool-open', value: plan.id }}
            label={`Open bid plan: ${plan.player}`}>See bid plan</Act></article>
      ))}
      {row && plans.length === 0 && (
        <Act
          verb="market"
          args={{ field: 'pool-plan', value: row.id }}
          label={`Plan a bid on ${row.name}`}
        >
          Plan a bid
        </Act>
      )}
    </section>
  );
}

// The Inspector for a free agent: his row and any open bid plan on him.
const PlayerLists = lazy(() => import('../personal/PersonalLists').then(m => ({ default: m.PlayerLists })));
export function PoolInspector({ id }: { id: string }) {
  const state = usePoolData(true);
  const row = state.reading?.value.rows.find((r) => r.id === id);
  const plans = (state.board?.value.plans ?? []).filter(
    (p) => p.playerId === id && planOpen(p.stateCode),
  );
  return <><PoolInspectorView row={row} plans={plans} busy={state.busy} reading={state.reading} />
    <Suspense fallback={null}><PlayerLists id={id} /></Suspense></>;
}

export const set = (change: { field: string; value: string }) =>
  poolState.change(change as PoolFilter);
