import { useEffect, useMemo, useState } from 'react';
import { createHostedRoom, type HostedPort } from '../commissioner/hostedRoom';
import { newKey } from '../commissioner/forms';
import { HoldToFire } from '../commissioner/HoldToFire';
import { HQAct } from './HQAct';
import type { HQLineup } from './HQBoard';

export function useNativeLineup(league: string, mine: string, lineup?: HQLineup, port?: HostedPort) {
  const room = useMemo(() => createHostedRoom(league, mine, port), [league, mine, port]);
  const state = room.use();
  const [ids, setIds] = useState<string[]>();
  const [version, reset] = useState(0);
  useEffect(() => { void room.refresh(lineup?.week); return room.dispose; }, [room, lineup?.week]);
  const players = lineup?.starters.concat(lineup.bench) ?? [];
  const current = ids ?? lineup?.starters.map(p => p.id) ?? [];
  const edited = lineup && { ...lineup, starters: current.flatMap(id => players.filter(p => p.id === id)),
    bench: players.filter(p => !current.includes(p.id)) };
  const changed = current.filter(id => !lineup?.starters.some(p => p.id === id)).concat(
    lineup?.starters.filter(p => !current.includes(p.id)).map(p => p.id) ?? []);
  const discard = () => { setIds(undefined); room.cancel(); reset(n => n + 1); };
  const save = async () => {
    const reading = state.reading;
    if (!reading || !room.canChange() || !changed.length || reading.lineup.week !== lineup?.week) return;
    await room.review({ command: 'commish.lineup.set', mode: 'preview', arguments: {
      league, command: 'commish.lineup.set', args: { lineup: { season: reading.lineup.season,
        franchise: mine, week: reading.lineup.week, starters: current } },
      reason: '', expected: reading.lineup.revision, request: newKey(), effective: 'now',
      visibility: 'private' } }, `Set your week ${reading.lineup.week} lineup`);
  };
  const reserve = async (player: string) => {
    const reading = state.reading;
    if (!reading || !room.canChange()) return;
    await room.review({ command: 'roster.reserve', mode: 'preview', arguments: {
      league, command: 'roster.reserve', args: { waiver: { player, franchise: mine, bucket: 'ir' } },
      reason: '', expected: reading.roster.revision, request: newKey(), effective: 'now',
      visibility: 'league' } }, 'Move player to injured reserve');
  };
  const message = state.error || state.message || (changed.length ? `${changed.length} players changed from saved`
    : 'No changes. Swaps stay a plan until you save.');
  return { room, state, edited, changed, assign: setIds, version, discard, save, reserve, message,
    canEdit: room.canChange() && !state.review, dirty: ids !== undefined };
}

export function NativeLineupReview({ editor, refresh }: {
  editor: ReturnType<typeof useNativeLineup>; refresh: () => void;
}) {
  const { review, pending } = editor.state;
  if (!review) return null;
  return <div className="bdrawer open" role="dialog" aria-label="Confirm league change">
    <div className="bd-head"><h2>{review.summary}</h2><HQAct onAction={editor.room.cancel}>CLOSE</HQAct></div>
    <div className="sd-body"><p>This changes your league server, not MFL or a local plan.</p>
      {review.preview?.fields.map((f, i) => <p key={i}>{f.label}: {String(f.before)} → {String(f.after)}</p>)}
      {review.preview?.blockers.map((b, i) => <p key={i} role="alert">{b}</p>)}
      {!review.preview?.blockers.length && <HoldToFire appearance="hq" label="Confirm league change"
        body={JSON.stringify(review.request)} disabled={pending || !editor.room.canChange()}
        onConfirm={() => { void editor.room.confirm().then(() => {
          if (!editor.room.peek().error) { editor.discard(); refresh(); }
        }); }} />}
    </div></div>;
}
