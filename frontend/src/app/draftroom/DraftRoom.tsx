import { lazy, Suspense, useEffect, useRef, useState } from 'react';
import { leagueState } from '../commissioner/proxy';
import { surfaceActions } from '../commissioner/surfaceActions';
import { DraftBoard } from './DraftBoard';
import { DraftClock, DraftLauncher } from './DraftClock';
import { DraftOutcome } from './DraftOutcome';
import type { Draft } from './draft';
import { franchiseName } from './draftWords';
import { draftRoom } from './state';
import { parseTrayAction, type TrayAction } from './trayAction';
import { useDraftReading, useNow } from './useDraftReading';
import { YourPick } from './YourPick';
import './draftroom.css';

const DraftTray = lazy(() => import('./DraftTray'));
type Tray = TrayAction & { opener: HTMLElement };

function RoundBoard({ draft, may, round, onRound }:
  { draft: Draft; may: boolean; round: number; onRound(round: number): void }) {
  const shown = round || draft.onClock?.round || 1;
  return (
    <>
      <label>Round to show
        <select value={shown} onChange={e => onRound(Number(e.target.value))}>
          {draft.order.map(o => <option key={o.round} value={o.round}>Round {o.round}</option>)}
        </select>
      </label>
      <DraftBoard draft={draft} round={shown} may={may} />
    </>
  );
}

function Rules({ draft }: { draft: Draft }) {
  return (
    <details className="draft-rules">
      <summary>How this draft runs</summary>
      <dl>{draft.rules.map(r => <div key={r.label}><dt>{r.label}</dt><dd>{r.words}</dd></div>)}</dl>
    </details>
  );
}

// DraftRoom is the rookie draft: the clock, your own pick, the board and, for a commissioner, the tray.
// Go replays the ledger into the draft and decides every refusal; this presents it.
export default function DraftRoom() {
  const { choices, selected, people, outcome, error: leagueError, pending } = leagueState.use();
  const room = draftRoom.use();
  const [viewing, setViewing] = useState('');
  const [round, setRound] = useState(0);
  const [tray, setTray] = useState<Tray>();
  const background = useRef<HTMLDivElement>(null);
  const draft = room.draft;
  const franchise = viewing || people?.yourFranchise || people?.seats[0]?.franchise || '';
  const may = people?.mayChangeRules === true;
  const clock = draft?.onClock ?? null;
  const waiting = draft !== undefined && !draft.started && Date.parse(draft.start) > 0;
  const now = useNow((clock !== null && !clock.paused) || waiting);
  const due = clock && !clock.paused ? clock.deadline : waiting ? draft.start : '';
  const landed = outcome.status === 'done' && outcome.form.startsWith('draft.')
    ? outcome.receipt : undefined;
  useDraftReading(selected, franchise, due, now, landed?.entry ?? '');
  useEffect(() => { void leagueState.choose(); }, []);
  useEffect(() => { if (landed) setTray(undefined); }, [landed?.entry]);
  useEffect(() => surfaceActions.register('draftroom', args => {
    if (args.action === 'close') setTray(undefined);
    if (args.action === 'pick' && people?.yourFranchise)
      return draftRoom.pick(people.yourFranchise, args.value).then(() => {
        const outcome = draftRoom.peek().pick;
        if (outcome.status === 'refused') throw new Error(outcome.message);
      });
    const action = args.action === 'tray' ? parseTrayAction(args.value) : null;
    if (action && may && draft) setTray({ ...action, opener: document.activeElement as HTMLElement });
  }), [may, draft, people]);
  if (!pending && choices.length === 0 && !leagueError) return null;
  const yourTurn = clock !== null && people?.yourFranchise === clock.owner && draft?.closed === null;
  return (
    <section className="draft-room" aria-label="Rookie draft room" aria-busy={room.pending}>
      <div ref={background} aria-hidden={tray ? true : undefined}>
        <h4>{draft ? `${draft.season} rookie draft` : 'Rookie draft'}</h4>
        {leagueError && <p role="alert">{leagueError}</p>}
        {room.error && <p role="alert">{room.error}</p>}
        {!draft && !room.error && !leagueError && <p role="status">Reading the draft…</p>}
        {draft && (
          <>
            {people && people.seats.length > 1 && (
              <label>Franchise to view
                <select value={franchise} onChange={e => setViewing(e.target.value)}>
                  {people.seats.map(s => <option key={s.franchise} value={s.franchise}>{s.name}</option>)}
                </select>
              </label>
            )}
            <DraftClock draft={draft} now={now} may={may} />
            {yourTurn && <YourPick draft={draft} outcome={room.pick} />}
            {draft.next && !yourTurn && draft.closed === null && (
              <p>{franchiseName(draft, franchise)}’s next pick is round {draft.next.round}, pick{' '}
                {draft.next.slot}.</p>
            )}
            {landed && <p role="status">{landed.message}</p>}
            {draft.closed && <DraftOutcome closed={draft.closed} />}
            <RoundBoard draft={draft} may={may} round={round} onRound={setRound} />
            {may && draft.started && (
              <div className="draft-launchers" role="group" aria-label="Restore or close the draft">
                {draft.checkpoints.length > 1 && <DraftLauncher action={{ kind: 'restore' }}
                  text="Restore the draft to an earlier point" />}
                {draft.closed === null && <DraftLauncher action={{ kind: 'close' }} text="Close the draft" />}
              </div>
            )}
            <Rules draft={draft} />
          </>
        )}
      </div>
      {tray && draft && (
        <Suspense fallback={<p role="status">Opening the change…</p>}>
          <DraftTray action={tray} draft={draft} opener={tray.opener} background={background.current}
            onClose={() => setTray(undefined)} />
        </Suspense>
      )}
    </section>
  );
}
