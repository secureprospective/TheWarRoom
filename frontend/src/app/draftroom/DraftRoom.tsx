import { lazy, Suspense, useEffect, useMemo, useRef, useState } from 'react';
import { leagueState } from '../commissioner/proxy';
import { surfaceActions } from '../commissioner/surfaceActions';
import { DraftBoard } from './DraftBoard';
import { DraftClock, DraftLauncher } from './DraftClock';
import { DraftOutcome } from './DraftOutcome';
import { oldestOwedPick, type Draft } from './draft';
import { franchiseName } from './draftWords';
import { createDraftRoom, draftRoom, type DraftRoomPort } from './state';
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
export default function DraftRoom({ league, franchise: viewFranchise, sessionFranchise, port, mayChangeRules,
  hideFranchiseView = false }:
  { league?: string; franchise?: string; sessionFranchise?: string; port?: DraftRoomPort;
    mayChangeRules?: boolean;
    hideFranchiseView?: boolean } = {}) {
  const session = leagueState.use();
  const hasScopedSession = sessionFranchise !== undefined;
  const { choices, pending } = session;
  const people = hasScopedSession ? undefined : session.people;
  const outcome = hasScopedSession ? undefined : session.outcome;
  const leagueError = hasScopedSession ? '' : session.error;
  const client = useMemo(() => port ? createDraftRoom(port) : draftRoom, [port]);
  const room = client.use();
  const selected = hasScopedSession ? league ?? '' : session.selected;
  const viewKey = `${hasScopedSession ? 1 : 0}\0${league ?? ''}\0${selected}\0${viewFranchise ?? ''}`;
  const [viewingState, setViewingState] = useState<{ key: string; franchise: string }>();
  const viewing = viewingState?.key === viewKey ? viewingState.franchise : '';
  const [round, setRound] = useState(0);
  const [tray, setTray] = useState<Tray>();
  const background = useRef<HTMLDivElement>(null);
  const actingFranchise = hasScopedSession ? sessionFranchise : people?.yourFranchise || '';
  const franchise = hasScopedSession ? viewing || viewFranchise || ''
    : viewing || viewFranchise || actingFranchise || people?.seats[0]?.franchise || '';
  const loaded = room.draft;
  const draft = loaded && loaded.league === selected && room.scope?.league === selected &&
    room.scope.franchise === franchise ? loaded : undefined;
  const mayChange = mayChangeRules ?? people?.mayChangeRules === true;
  const may = mayChange && !hideFranchiseView;
  const clock = draft?.onClock ?? null;
  const waiting = draft !== undefined && !draft.started && Date.parse(draft.start) > 0;
  const now = useNow((clock !== null && !clock.paused) || waiting);
  const due = clock && !clock.paused ? clock.deadline : waiting ? draft.start : '';
  const landed = outcome?.status === 'done' && outcome.form.startsWith('draft.')
    ? outcome.receipt : undefined;
  useDraftReading(selected, franchise, due, now, landed?.entry ?? '', room.startup,
    client.read);
  useEffect(() => { setViewingState(undefined); }, [viewKey]);
  useEffect(() => { if (!hasScopedSession) void leagueState.choose(league); }, [league, hasScopedSession]);
  useEffect(() => { if (landed) setTray(undefined); }, [landed?.entry]);
  useEffect(() => surfaceActions.register('draftroom', args => {
    if (args.action === 'close') setTray(undefined);
    const current = client.peek();
    if (args.action === 'pick' && actingFranchise && franchise === actingFranchise &&
      current.draft?.league === selected && current.scope?.league === selected &&
      current.scope.franchise === franchise)
      return client.pick(actingFranchise, args.value).then(() => {
        const outcome = client.peek().pick;
        if (outcome.status === 'refused') throw new Error(outcome.message);
      });
    const action = args.action === 'tray' ? parseTrayAction(args.value) : null;
    if (action && may && draft) setTray({ ...action, opener: document.activeElement as HTMLElement });
  }), [may, draft, actingFranchise, client, franchise, selected]);
  if (!hasScopedSession && !pending && choices.length === 0 && !leagueError)
    return <section className="draft-room">
    <h4>Draft room</h4>
    <p>No app league has been started. Create a league in Control Room to use the draft.</p>
  </section>;
  const owed = draft ? oldestOwedPick(draft, actingFranchise) : undefined;
  const viewingOwnTeam = franchise === actingFranchise;
  const canMakeOwed = Boolean(owed?.contract && viewingOwnTeam && !clock?.paused);
  const yourTurn = !owed && viewingOwnTeam && clock !== null && actingFranchise === clock.owner &&
    draft?.closed === null;
  const isStartup = draft?.startup ?? room.startup;
  const kind = isStartup ? 'Startup' : 'Rookie';
  return (
    <section className="draft-room" aria-label={`${kind} draft room`} aria-busy={room.pending}>
      <div ref={background} aria-hidden={tray ? true : undefined}>
        <h4>{draft ? `${draft.season} ${kind.toLowerCase()} draft` : `${kind} draft`}</h4>
        {leagueError && <p role="alert">{leagueError}</p>}
        {room.error && <p role="alert">{room.error}</p>}
        {!draft && !room.error && !leagueError && <p role="status">Reading the draft…</p>}
        {draft && (
          <>
            {!hideFranchiseView && people && people.seats.length > 1 && (
              <label>Franchise to view
                <select value={franchise}
                  onChange={e => setViewingState({ key: viewKey, franchise: e.target.value })}>
                  {people.seats.map(s => <option key={s.franchise} value={s.franchise}>{s.name}</option>)}
                </select>
              </label>
            )}
            {may && draft.startup && !draft.started && draft.closed === null && (
              <DraftLauncher action={{ kind: 'open' }} text="Open the startup draft" />
            )}
            <DraftClock draft={draft} now={now} may={may} />
            {(yourTurn || canMakeOwed) && <YourPick draft={draft} outcome={room.pick}
              owed={canMakeOwed ? owed : undefined} />}
            {draft.next && franchise === actingFranchise && !yourTurn && draft.closed === null && (
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
