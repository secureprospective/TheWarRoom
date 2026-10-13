import { lazy, Suspense, useEffect } from 'react';
import type { Snapshot } from '../data/contract';
import { Act } from '../commands/Act';
import { ReadingView } from './ReadingView';
import { draftState, segments, nflRounds } from './draftState';
import type { MarketFilter, SurfaceReading } from './contract';

const PersonalLists = lazy(() => import('../personal/PersonalLists'));
const NFLDraft = lazy(() => import('../personal/RosterReports'));
const Reports = lazy(() => import('../draftreports/Reports'));

type Option = readonly [string, string, string];
function Choices({
  field,
  options,
  selected,
}: {
  field: MarketFilter['field'];
  options: readonly Option[];
  selected: readonly string[];
}) {
  return (
    <div className="market-controls">
      {options.map(([value, shown, label]) => (
        <Act variant="button"
          key={value}
          verb="market"
          args={{ field, value }}
          active={selected.includes(value)}
          label={label}
        >
          {shown}
        </Act>
      ))}
    </div>
  );
}
export function DraftControls() {
  const s = draftState.use();
  return (
    <section className="market-filters" aria-label="Draft filters">
      <h5>Draft classes</h5>
      <Choices
        field="draft-from"
        options={[
          ['all', '2017–2026', 'Use classes 2017–2026'],
          ['recent', '2024–2026', 'Use classes 2024–2026'],
        ]}
        selected={[s.scope.from === 2017 ? 'all' : 'recent']}
      />
      <h5>Judge on</h5>
      <Choices
        field="draft-seasons"
        options={[
          ['2', 'First 2 seasons', 'Judge on first 2 seasons'],
          ['3', 'First 3 seasons', 'Judge on first 3 seasons'],
          ['4', 'First 4 seasons', 'Judge on first 4 seasons'],
        ]}
        selected={[String(s.scope.seasons)]}
      />
      <h5>Draft stretches</h5>
      <Choices
        field="draft-segment"
        options={segments.map(([id, label]) => [id, label, `Filter draft stretch ${label}`] as const)}
        selected={s.scope.segments}
      />
      <h5>NFL draft round that spring</h5>
      <Choices
        field="draft-nfl"
        options={nflRounds.map(
          ([id, label]) =>
            [id, label.startsWith('NFL ') ? label.slice(4) : label, `Filter ${label}`] as const,
        )}
        selected={s.scope.nfl}
      />
      <h5>Position profile</h5>
      <Choices
        field="draft-position"
        options={['QB', 'RB', 'WR', 'TE', 'PK', 'DL', 'LB', 'DB'].map(
          (id) => [id, id, `Draft profile ${id}`] as const,
        )}
        selected={[s.scope.group]}
      />
      <Act variant="button"
        verb="market" args={{ field: 'draft-reset', value: '' }} label="Reset Draft filters">
        Every stretch and NFL round · reset
      </Act>
      {s.reading?.status === 'failed' && (
        <Act variant="button"
          verb="market"
          args={{ field: 'draft-retry', value: '' }}
          label="Retry Draft archive reading"
        >
          Retry reading
        </Act>
      )}
      {s.pending && (
        <p role="status">
          Updating Draft filters; any figures below still describe the previous scope until Go replies.
        </p>
      )}
    </section>
  );
}
export function DraftReading({ reading, kind }: { reading?: SurfaceReading; kind?: string }) {
  const shown =
    kind && reading?.value
      ? {
          ...reading,
          value: {
            ...reading.value,
            sections: reading.value.sections.filter((s) => ['L-19', 'L-20', 'L-21'].includes(s.id)),
          },
        }
      : reading;
  return (
    <ReadingView
      reading={shown}
      title={kind ? `Draft pick card · ${kind}` : 'When to take each position'}
      inspect
    />
  );
}
export default function Draft({ kind, snapshot }: { kind?: string; snapshot?: Snapshot }) {
  const s = draftState.use();
  useEffect(() => {
    if (kind) void draftState.pick(kind);
    else if (!s.reading && !s.pending) void draftState.refresh();
  }, [kind, s.scope]);
  return (
    <div className="market">
      {!kind && snapshot && <Suspense fallback={null}><PersonalLists snapshot={snapshot} draft /></Suspense>}
      {!kind && <Suspense fallback={null}><Reports /><NFLDraft mine="" draft /></Suspense>}
      <h4>{kind ? 'What this round has returned' : 'Draft lab'}</h4>
      <p>
        Frozen draft history, not live prospect grades. 100% is the same slot in other classes. Empty
        stretch or NFL selections mean every pick.
      </p>
      {kind ? (
        <p>
          Historical round cohort for the position and seasons named in the reading below. A future pick
          has no known slot or player; these are not individual predictions.
        </p>
      ) : (
        <DraftControls />
      )}
      {kind && s.picks[kind]?.status === 'failed' && (
        <Act variant="button"
          verb="market" args={{ field: 'draft-retry', value: '' }} label="Retry draft pick history">
          Retry reading
        </Act>
      )}
      {kind && s.pending && <p role="status">Updating; this card still shows the previous scope.</p>}
      <DraftReading reading={kind ? s.picks[kind] : s.reading} kind={kind} />
    </div>
  );
}
export const set = draftState.change;
