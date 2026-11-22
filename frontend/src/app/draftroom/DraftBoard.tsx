import type { Draft, Pick } from './draft';
import { DraftLauncher } from './DraftClock';
import { contractWords, franchiseName, isOnClock, pickWords, playerLine, when } from './draftWords';

const slotOf = (p: Pick) => (p.extra > 0 ? `${p.slot} (extra)` : String(p.slot));

function Result({ draft, pick }: { draft: Draft; pick: Pick }) {
  const taken = pick.player === '' ? undefined : draft.players[pick.player];
  return (
    <>
      {pick.player !== '' && <strong>{taken ? playerLine(taken) : `Player ${pick.player}`}</strong>}
      <span className="draft-how">{pickWords(draft, pick)}</span>
      {pick.contract && pick.player !== '' && <span>{contractWords(pick.contract)}</span>}
      {pick.madeAt && <time dateTime={pick.madeAt}>{when(pick.madeAt)}</time>}
    </>
  );
}

function Launchers({ draft, pick }: { draft: Draft; pick: Pick }) {
  const ref = { round: pick.round, original: pick.original, extra: pick.extra };
  const named = `round ${pick.round} pick ${slotOf(pick)}, ${franchiseName(draft, pick.owner)}`;
  const who = draft.startup ? 'player' : 'rookie';
  const open = pick.player === '' && (isOnClock(draft, pick) || pick.skipped) && !pick.forfeited;
  return (
    <>
      {open && <DraftLauncher action={{ kind: 'assign', ref }} text={`Choose a ${who}`}
        label={`Choose a ${who} for ${named}`} />}
      {pick.player !== '' && <DraftLauncher action={{ kind: 'change', ref }} text="Change"
        label={`Change ${named}`} />}
      {pick.extra === 0 && <DraftLauncher action={{ kind: 'insert', ref }} text="Add extra pick"
        label={`Add an extra pick after ${named}`} />}
    </>
  );
}

// DraftBoard lists one round: every pick with its owner and what became of it. A round that cannot be
// ordered yet says why, in Go's words.
export function DraftBoard({ draft, round, may }: { draft: Draft; round: number; may: boolean }) {
  const order = draft.order.find(o => o.round === round);
  const picks = draft.picks.filter(p => p.round === round);
  const live = may && draft.started && draft.closed === null;
  return (
    <table className="report-table draft-board" aria-label={`Round ${round}`}>
      <caption>Round {round}</caption>
      <thead><tr><th scope="col">Pick</th><th scope="col">Franchise</th><th scope="col">Result</th>
        {live && <th scope="col">Change</th>}</tr></thead>
      <tbody>
        {picks.length === 0 && (
          <tr><td colSpan={live ? 4 : 3}>{order?.words ?? 'This round has no order yet.'}</td></tr>
        )}
        {picks.map(p => (
          <tr key={`${p.original}:${p.extra}`} data-state={isOnClock(draft, p) ? 'clock' : undefined}>
            <th scope="row">{slotOf(p)}</th>
            <td>{franchiseName(draft, p.owner)}
              {p.original !== p.owner && <small> (from {franchiseName(draft, p.original)})</small>}</td>
            <td><Result draft={draft} pick={p} /></td>
            {live && <td><Launchers draft={draft} pick={p} /></td>}
          </tr>
        ))}
      </tbody>
    </table>
  );
}
