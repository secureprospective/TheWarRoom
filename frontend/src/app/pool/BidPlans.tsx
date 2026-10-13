import { useEffect } from 'react';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { localDate } from '../clock/labels';
import type { Signal } from '../look/channels';
import { SignalChip } from '../look/Slots';
import { RailStrip } from '../shell/EnvelopeRail';
import type { BidAction, BoardReading, Plan, PoolFilter, Rule } from './contract';
import { poolState, type PoolState } from './state';
import './pool.css';

export const signals: Record<string, Signal> = {
  landed: 'green', awarded_off_app: 'grey', blocked: 'red', failed: 'red', draft: 'grey',
  ready: 'blue', handed_off: 'blue', bid_pending: 'amber', not_yet_done: 'amber',
  not_verified: 'amber', stale: 'amber', withdrawn: 'grey',
};
const verbFields: Record<string, PoolFilter['field']> = {
  'hand-off': 'pool-handoff', revise: 'pool-revise', withdraw: 'pool-withdraw', posted: 'pool-posted',
  copy: 'pool-copy', unposted: 'pool-unposted',
};
// The post is shown while the owner still has something to do with it.
const showsPost = new Set(['blocked', 'ready', 'handed_off', 'not_yet_done', 'not_verified', 'bid_pending']);

function RuleList({ rules }: { rules: Rule[] }) {
  return (
    <ul className="bid-rules">
      {rules.map((r) => (
        <li key={r.rule} data-verdict={r.verdict}>
          <b>{r.label}</b>
          <span className="bid-mark">{r.mark}</span>
          <p>{r.reason}</p>
        </li>
      ))}
    </ul>
  );
}

// Failures and notes show in full; the rules that pass fold into Go's one line.
export function Rules({ rules, summary }: { rules: Rule[]; summary: string }) {
  const passing = rules.filter((r) => r.verdict === 'pass');
  const rest = rules.filter((r) => r.verdict !== 'pass');
  return (
    <div className="bid-rules-box" aria-label="Bid rules">
      {rest.length > 0 && <RuleList rules={rest} />}
      {passing.length > 0 && (
        <details className="bid-pass">
          <summary>{summary}</summary>
          <RuleList rules={passing} />
        </details>
      )}
    </div>
  );
}

function ActionButton({ plan, action, busy }: { plan: Plan; action: BidAction; busy: string }) {
  return (
    <>
      <Act variant="button"
        verb="market"
        args={{ field: verbFields[action.id], value: plan.id }}
        label={`${action.label}: ${plan.player}`}
        disabled={action.disabled || busy === plan.id}
      >
        {action.label}
      </Act>
      {action.disabled && action.reason && (
        <p className="bid-why">{`${action.label} is off: ${action.reason}`}</p>
      )}
    </>
  );
}

export function BidPlanCard({ plan, busy = '' }: { plan: Plan; busy?: string }) {
  const stages = plan.history.filter((entry) => entry.state !== 'draft');
  const finished = ['withdrawn', 'landed', 'failed', 'awarded_off_app'].includes(plan.stateCode);
  return (
    <article className="bid-plan" data-state={plan.stateCode} aria-label={`Bid plan ${plan.player}`}>
      <header>
        <h5>{plan.player}</h5>
      </header>
      {!finished && <p className="bid-line">
        {plan.contract} · {plan.points} points · {plan.beatText}
      </p>}
      <RailStrip receipt={plan.receipt} />
      {!finished && plan.rules.length > 0 && <Rules rules={plan.rules} summary={plan.ruleSummary} />}
      {showsPost.has(plan.stateCode) && (
        <pre className="bid-post" aria-label="ProBoards post">
          {plan.post}
        </pre>
      )}
      {!finished && plan.evidence && <p className="bid-evidence">Evidence: {plan.evidence}</p>}
      <div className="bid-actions">
        {!finished && <Act variant="button" verb="acquisition" args={{ mine: plan.franchiseId, bid: plan.id }}
          label="What does the add cost me?" />}
        {plan.actions.map((action) => (
          <ActionButton key={action.id} plan={plan} action={action} busy={busy} />
        ))}
      </div>
      <details className="bid-details">
        <summary>History</summary>
        {finished && <pre className="bid-post" aria-label="ProBoards post">{plan.post}</pre>}
        <ol className="bid-history">
          {stages.map((entry) => (
            <li key={`${entry.at}${entry.state}`}>
              <time dateTime={entry.at}>{localDate(entry.at)}</time> {entry.note}
            </li>
          ))}
        </ol>
      </details>
    </article>
  );
}

type BoardProps = {
  board?: BoardReading;
  busy: string;
  limit?: number;
  // Which plans this board lists; the pool lists only the finished ones below its table.
  show?: (plan: Plan) => boolean;
  title?: string;
};

export function BidBoardView({ board, busy, limit, show = () => true, title = 'Bid plans' }: BoardProps) {
  const plans = (board?.value.plans ?? []).filter(show);
  return (
    <section className="bid-board" aria-label={title} data-endpoint="M-194">
      <h4>{title}</h4>
      {!board && <p role="status">Reading your bid plans…</p>}
      {board && board.status !== 'ok' && (
        <p role="alert">Failed · {board.absenceReason}</p>
      )}
      {board?.status === 'ok' && (
        <>
          <p>{board.value.summary}. Only your own plans are kept; nothing about other franchises.</p>
          {plans.length === 0 && <p>No bid plans yet. Plan a bid from the free-agent pool.</p>}
          {plans.slice(0, limit).map((plan) => (
            <BidPlanCard key={plan.id} plan={plan} busy={busy} />
          ))}
          {limit !== undefined && plans.length > limit && (
            <p>{plans.length - limit} older bid plans not shown</p>
          )}
          <details className="bid-details">
            <summary>How this is counted</summary>
            <p>{board.howCounted}</p>
          </details>
        </>
      )}
    </section>
  );
}

// Keeps the plan store fresh while a screen that shows plans is open.
export function usePoolData(withPool: boolean): PoolState {
  const franchiseId = commands.use().franchiseId ?? '';
  useEffect(() => {
    void (withPool ? poolState.open() : poolState.loadBoard());
    return commands.onMovesChange(() => void (withPool ? poolState.reload() : poolState.loadBoard()));
  }, [franchiseId, withPool]);
  return poolState.use();
}

export default function BidPlans() {
  const s = usePoolData(false);
  return <BidBoardView board={s.board} busy={s.busy} limit={15} />;
}
