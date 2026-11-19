import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { HoldToFire } from '../commissioner/HoldToFire';
import { Review } from '../commissioner/HostedReview';
import { createHostedLeagueRoom } from '../commissioner/hostedRoom';
import { reviewHostedWaiverClaim } from '../commissioner/HostedWorkflows';
import { previewChange } from '../commissioner/preview';
import type { Position } from '../data/contract';
import { HQAct } from '../shell/HQAct';
import { WarRoomInspector } from '../shell/WarRoomInspector';
import { confirmWarRoomRosterAdd, reviewWarRoomRosterAdd,
  type WarRoomRosterAddReview } from '../shell/warroomMarketReads';
import type { WarRoomContext } from '../shell/warroomReads';
import { executeWarRoomOffer, readWarRoomCatalog, readWarRoomPartner, readWarRoomPreview,
  type WarRoomCatalog, type WarRoomDeal } from '../shell/warroomTradeReads';

export type MovesPlayer = { id: string; name: string; position: string; window?: string; waiver?: string };
export type MovesRoster = { revision: number; writable: boolean; football: string;
  mine: { id: string; name: string }[] };
const message = (cause: unknown) => cause instanceof Error ? cause.message : String(cause);

/** MoveSheet is the phone frame's native-dialog sheet: Escape, backdrop and Close all dismiss it. */
export function MoveSheet({ title, onClose, children }: { title: string; onClose: () => void;
  children: ReactNode }) {
  const dialog = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const element = dialog.current;
    if (element && !element.open) element.showModal?.();
    return () => element?.close?.();
  }, []);
  return <dialog ref={dialog} className="phone-app__sheet mv4-sheet" role="dialog" aria-modal="true"
    aria-label={title} onCancel={event => { event.preventDefault(); onClose(); }}
    onPointerUp={event => { if (event.target === event.currentTarget) onClose(); }}>
    <div className="mv4-sheet__head"><h3>{title}</h3>
      <HQAct className="mv4-close" label="Close" onAction={onClose}>Close</HQAct></div>
    {children}
  </dialog>;
}

export function PlayerSheet({ context, player, onMove, onClose }: { context: WarRoomContext;
  player: MovesPlayer; onMove: () => void; onClose: () => void }) {
  return <MoveSheet title={player.name} onClose={onClose}>
    <WarRoomInspector context={context} asset={{ token: player.id, name: player.name,
      position: (player.position || undefined) as Position | undefined }} add={onMove} />
    <HQAct className="mv4-key" onAction={onMove}>{player.window ? 'Claim' : 'Add'}</HQAct>
  </MoveSheet>;
}

/** AddSheet wraps roster.add: choose who goes, give a reason, review the server preview, hold to confirm. */
export function AddSheet({ context, player, roster, onDone, onClose }: { context: WarRoomContext;
  player: MovesPlayer; roster: MovesRoster; onDone: (words: string) => void; onClose: () => void }) {
  const [drop, setDrop] = useState('');
  const [reason, setReason] = useState('');
  const [review, setReview] = useState<WarRoomRosterAddReview>();
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  const run = async (step: () => Promise<void>) => {
    setError(''); setPending(true);
    try { await step(); } catch (cause) { setError(message(cause)); } finally { setPending(false); }
  };
  return <MoveSheet title={`Add ${player.name}`} onClose={onClose}>
    <label className="mv4-field">Who goes for this player
      <select value={drop} onChange={event => { setDrop(event.target.value); setReview(undefined); }}>
        <option value="">Nobody: use an open bench spot</option>
        {roster.mine.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
      </select></label>
    <label className="mv4-field">Reason
      <input value={reason} onChange={event => { setReason(event.target.value); setReview(undefined); }} />
    </label>
    {!review && <HQAct className="mv4-key" disabled={pending || !reason.trim()}
      onAction={() => void run(async () => setReview(await reviewWarRoomRosterAdd(context, {
        player: player.id, drop: drop || undefined, reason, revision: roster.revision,
        writable: roster.writable, football: roster.football })))}>Review add</HQAct>}
    {review && <section aria-label="Add preview">
      {review.preview.fields.map((field, i) => <p key={i}><b>{field.label}</b>: {previewChange(field)}</p>)}
      {review.preview.blockers.map(b => <p role="alert" key={b}>{b}</p>)}
      {!review.preview.blockers.length && <HoldToFire label="Confirm add"
        body={JSON.stringify(review.request)}
        disabled={pending} onConfirm={() => void run(async () => {
          await confirmWarRoomRosterAdd(context, review);
          onDone(`${player.name} added`);
        })} />}
    </section>}
    {error && <p role="alert">{error}</p>}
  </MoveSheet>;
}

/** ClaimSheet wraps the owner's private waivers.claim through the hosted review room. */
export function ClaimSheet({ context, player, revision, blind, onDone, onClose }: { context: WarRoomContext;
  player: MovesPlayer; revision: number; blind: boolean; onDone: (words: string) => void;
  onClose: () => void }) {
  const native = context.native!;
  const room = useMemo(() => createHostedLeagueRoom(native.league, native.port),
    [native.league, native.port]);
  const state = room.use();
  const [error, setError] = useState('');
  useEffect(() => { void room.refresh(); return room.dispose; }, [room]);
  return <MoveSheet title={`Claim ${player.name}`} onClose={onClose}>
    <p className="mv4-note">{player.waiver}</p>
    <p className="mv4-note">Your claim stays private until waivers run.</p>
    {blind && <p className="mv4-note">Bid amount: Not available yet</p>}
    {!state.review && <HQAct className="mv4-key" disabled={!room.canChange()} onAction={() => {
      setError('');
      void reviewHostedWaiverClaim(room, native.league, context.mine, player.window!, revision)
        .catch(cause => setError(message(cause)));
    }}>Review claim</HQAct>}
    {state.review && <Review room={room} onConfirmed={() => onDone(`Claim placed for ${player.name}`)} />}
    {(error || state.error) && <p role="alert">{error || state.error}</p>}
  </MoveSheet>;
}

/** TradeSheet: pick a partner, pick assets on both sides, review trade.offer, hold to send, reread. */
export function TradeSheet({ context, start, onDone, onClose }: { context: WarRoomContext;
  start?: WarRoomDeal; onDone: (words: string) => void; onClose: () => void }) {
  const [catalog, setCatalog] = useState<WarRoomCatalog>();
  const [deal, setDeal] = useState<WarRoomDeal>(start ?? { partner: '', give: [], get: [] });
  const [record, setRecord] = useState<string>();
  const [blockers, setBlockers] = useState<string[]>();
  const [error, setError] = useState('');
  const [pending, setPending] = useState(false);
  useEffect(() => {
    let active = true;
    void readWarRoomCatalog(context).then(c => { if (active) setCatalog(c); })
      .catch(cause => { if (active) setError(message(cause)); });
    return () => { active = false; };
  }, [context]);
  useEffect(() => {
    let active = true;
    setRecord(undefined);
    void readWarRoomPartner(context, deal.partner).then(p => { if (active) setRecord(p?.record); })
      .catch(() => { if (active) setRecord(undefined); });
    return () => { active = false; };
  }, [context, deal.partner]);
  const change = (next: WarRoomDeal) => { setDeal(next); setBlockers(undefined); };
  const team = (id: string) => catalog?.teams.find(t => t.id === id);
  if (!catalog) return <MoveSheet title="Start a trade" onClose={onClose}>
    {error ? <p role="alert">Unavailable: {error}</p> : <p role="status">Reading rosters…</p>}</MoveSheet>;
  if (!deal.partner) return <MoveSheet title="Trade with" onClose={onClose}>
    {catalog.teams.filter(t => t.id !== context.mine).map(t => <HQAct key={t.id} className="mv4-row"
      onAction={() => change({ partner: t.id, give: [], get: [] })}>
      <b>{t.name}</b><em>{t.assets.length} assets</em></HQAct>)}
    {catalog.teams.length < 2 && <p className="mv4-empty">No other team to trade with.</p>}
  </MoveSheet>;
  const side = (key: 'give' | 'get', owner: string, label: string) => {
    const chosen = deal[key];
    return <section className="mv4-side" aria-label={label}>
      <h4>{label} <em>{chosen.length} selected</em></h4>
      {(team(owner)?.assets ?? []).map(a => <HQAct key={a.token} className="mv4-row"
        selected={chosen.includes(a.token)} onAction={() => change({ ...deal, [key]: chosen.includes(a.token)
          ? chosen.filter(t => t !== a.token) : [...chosen, a.token] })}>
        <span>{chosen.includes(a.token) ? '✓ ' : ''}<b>{a.name}</b></span>
        <em>{[a.position, a.contract].filter(Boolean).join(' · ')}</em></HQAct>)}
    </section>;
  };
  const ready = deal.give.length > 0 && deal.get.length > 0;
  const run = async (step: () => Promise<void>) => {
    setError(''); setPending(true);
    try { await step(); } catch (cause) { setError(message(cause)); } finally { setPending(false); }
  };
  return <MoveSheet title={`Trade with ${team(deal.partner)?.name ?? deal.partner}`} onClose={onClose}>
    {record && <p className="mv4-note">Record {record}</p>}
    <HQAct className="mv4-link" onAction={() => change({ partner: '', give: [], get: [] })}>
      Change partner</HQAct>
    {side('get', deal.partner, 'You get')}
    {side('give', context.mine, 'You give')}
    {blockers === undefined && <HQAct className="mv4-key" disabled={!ready || pending}
      onAction={() => void run(async () =>
        setBlockers((await readWarRoomPreview(context, catalog, deal)).reasons))}>Review offer</HQAct>}
    {blockers?.map(b => <p role="alert" key={b}>{b}</p>)}
    {blockers && !blockers.length && <HoldToFire label="Send offer" body={JSON.stringify(deal)}
      disabled={pending} onConfirm={() => void run(async () => {
        await executeWarRoomOffer(context, catalog, deal);
        onDone(`Offer sent to ${team(deal.partner)?.name ?? deal.partner}`);
      })} />}
    {error && <p role="alert">{error}</p>}
  </MoveSheet>;
}
