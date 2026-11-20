import { HQAct as WarRoomAct } from './HQAct';
import { useEffect, useState } from 'react';
import type { PlayerCandidate } from './ThinPlayerSearch';
import { HoldToFire } from '../commissioner/HoldToFire';
import { previewChange } from '../commissioner/preview';
import type { WarRoomContext } from './warroomReads';
import type { WarRoomOwnBid, WarRoomMarketActionResult } from './warroomMarketReads';
import { useWarRoomRead } from './warroomReads';
import { readWarRoomMarket, changeWarRoomTarget, warRoomMarketAction, warRoomTargetLanes,
  confirmWarRoomRosterAdd, reviewWarRoomRosterAdd, type WarRoomMarketReading,
  type WarRoomMarketPlayer, type WarRoomRosterAddReview } from './warroomMarketReads';
import { WarRoomEmpty } from './WarRoomHeader';
import { money, unavailable } from './hqModel';
import { copyText, planForm } from '../pool/state';
import type { Plan } from '../pool/contract';
import type { LensNeed } from './warroomLens';
import { useWarRoomDrag } from './warroomDrag';
export const ownBid = (p: Plan): WarRoomOwnBid => ({ player: p.playerId, name: p.player,
  contract: p.contract, state: p.status || p.state, kind: 'plan', id: p.id, post: p.post, plan: p });
const STEP = 0.1;
export function WarRoomOwnBids({ reading, kind, revise }: { reading?: WarRoomMarketReading;
  kind?: 'plan' | 'placed';
  revise?: (b: WarRoomOwnBid, change: { salary?: string; years?: number }) => void }) {
  const bids = reading?.bids?.filter(b => !kind || b.kind === kind);
  const [copied, setCopied] = useState('');
  const copy = async (b: WarRoomOwnBid) => {
    setCopied((await copyText(b.post ?? '')) ? b.player : '');
  };
  return <>{bids ? bids.length ? bids.map((b, k) => <div className="bidb"
    key={b.player}><span className="nm">{kind === 'plan' ? `${k + 1}. ` :
      ''}{b.name}</span><b>{b.contract}</b>
      {revise && b.plan && <span className="step2"><WarRoomAct label={`Lower ${b.name} bid`}
        onAction={() => revise(b, { salary:
          String(b.plan!.salary.cents / 1e8 - STEP) })}>−</WarRoomAct><input key={b.plan.salary.cents}
          aria-label={`${b.name} bid salary in millions`}
          className="bid-salary" type="number" step="any" defaultValue={b.plan.salary.cents / 1e8}
          onBlur={e => { if (e.currentTarget.value !== String(b.plan!.salary.cents / 1e8))
            revise(b, { salary: e.currentTarget.value }); }}
          onKeyDown={e => { if (e.key === 'Enter') e.currentTarget.blur(); }} /><WarRoomAct
          label={`Raise ${b.name} bid`} onAction={() => revise(b, { salary:
            String(b.plan!.salary.cents / 1e8 + STEP) })}>+</WarRoomAct></span>}
      {revise && b.plan && <label>Years <select aria-label={`${b.name} bid years`}
        value={b.plan.years} disabled={!reading?.maxYears}
        onChange={e => revise(b, { years: Number(e.target.value) })}>
        {Array.from({ length: reading?.maxYears ?? 0 }, (_, i) => i + 1).map(year =>
          <option key={year} value={year}>{year}</option>)}
      </select>{!reading?.maxYears && <small>Contract length setting unavailable</small>}</label>}
      {revise && b.bidId && <WarRoomAct label={`Revise ${b.name} bid`}
        onAction={() => revise(b, {})}>REVISE</WarRoomAct>}
      <span className="chip2">{b.state}</span>{b.post && <WarRoomAct className="chip"
        label={`Copy ProBoards post for ${b.name}`} onAction={() => { void copy(b); }}>
        {copied === b.player ? 'COPIED' : 'COPY PROBOARDS POST'}</WarRoomAct>}</div>) :
    kind === 'plan' ? null : <WarRoomEmpty reason="No own bids recorded" /> : <WarRoomEmpty />}
    {kind === 'plan' && <div className="add">Drag a player here to bid. Order = priority.</div>}</>;
}
type BidEditor = { type: 'place'; player: WarRoomMarketPlayer; salary: string; years: number } |
  { type: 'raise'; bid: WarRoomOwnBid; salary: string; years: number };
type BidReview = Extract<WarRoomMarketActionResult, { kind: 'review' }>;
type RefreshedBids = { bids?: WarRoomOwnBid[]; revision?: number; capRoom?: number; error?: string };
type RosterAddEditor = { player: WarRoomMarketPlayer; drop: string; reason: string };
export function WarRoomMarket({ context, room, needs, select, onPlayers }: { context: WarRoomContext;
  room?: number; needs?: LensNeed[]; select: (p: WarRoomMarketPlayer) => void;
  onPlayers?: (players: PlayerCandidate[]) => void }) {
  const [refresh, setRefresh] = useState(0);
  const [filter, setFilter] = useState('All');
  const [search, setSearch] = useState('');
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [editor, setEditor] = useState<BidEditor>();
  const [review, setReview] = useState<BidReview>();
  const [addEditor, setAddEditor] = useState<RosterAddEditor>();
  const [addReview, setAddReview] = useState<WarRoomRosterAddReview>();
  const [addNotice, setAddNotice] = useState('');
  const [fresh, setFresh] = useState<RefreshedBids>();
  const [own, setOwn] = useState<Record<string, WarRoomOwnBid>>({});
  const reading = useWarRoomRead(JSON.stringify([context.native?.league, context.mine, refresh]), () =>
    readWarRoomMarket(context));
  const value = reading.reading;
  useEffect(() => {
    if (value?.players) onPlayers?.(value.players.map(player => ({ id: player.id,
      name: player.name, position: player.position })));
  }, [onPlayers, value]);
  const targets = new Set(value?.targets?.map(t => t.id));
  const capRoom = fresh ? fresh.capRoom : value?.capRoom ?? room;
  const bidRevision = fresh?.revision ?? value?.bidRevision;
  const sourceBids = fresh?.bids ?? value?.bids;
  const bids = value && [...(sourceBids ?? []).filter(b => !own[b.player]), ...Object.values(own)];
  const shown = value && { ...value, bids };
  const change = async (id: string, withPlayer?: string) => {
    if (pending) return;
    setPending(true); setError(''); setNotice(''); setFresh(undefined);
    try { await changeWarRoomTarget(context, id, targets.has(id), withPlayer); setRefresh(n => n + 1); }
    catch { setError(unavailable); } finally { setPending(false); }
  };
  const bid = async (p: WarRoomMarketPlayer) => {
    if (pending || fresh?.error || bids?.some(b => b.player === p.id)) return;
    if (context.native) {
      if (!value?.writable || !value.football) {
        setError('This league is read-only or football evidence is unavailable; refresh before bidding');
        return;
      }
      if (bidRevision === undefined) {
        setError('Your bid version is unavailable; refresh before continuing');
        return;
      }
      setEditor({ type: 'place', player: p, salary: '', years: 0 }); return;
    }
    setPending(true); setError('');
    try {
      const result = await warRoomMarketAction(context, { type: 'draft', form: {
        planId: '', playerId: p.id, playerName: p.name, salary: String((p.minimumCents ?? 0) / 1e8),
        years: 1, snipe: false, beat: '', evidence: '', closesOpen: false, closesDate: '',
        closesTime: '', conditionalDrop: '', conditionalOpen: false } });
      if (result.kind === 'plan') setOwn(o => ({ ...o, [result.plan.playerId]: ownBid(result.plan) }));
    } catch (cause) { setError(cause instanceof Error ? cause.message : unavailable); }
    finally { setPending(false); }
  };
  const revise = async (b: WarRoomOwnBid, change: { salary?: string; years?: number }) => {
    if (pending || fresh?.error) return;
    if (context.native) {
      if (!value?.writable || !value.football) {
        setError('This league is read-only or football evidence is unavailable; refresh before bidding');
        return;
      }
      setEditor({ type: 'raise', bid: b, salary: '', years: 0 }); return;
    }
    if (!b.plan) return;
    if (change.years !== undefined && (!value?.maxYears || !Number.isSafeInteger(change.years) ||
      change.years < 1 || change.years > value.maxYears)) {
        setError('Choose a contract length within the league limit'); return; }
    if (change.salary !== undefined) {
      const salary = Number(change.salary);
      if (!Number.isFinite(salary) || salary <= 0) { setError('Enter a bid greater than zero'); return; }
      if (capRoom === undefined) { setError('Cap room is unavailable; bid was not changed'); return; }
      if (Math.round(salary * 1e8) > capRoom) { setError('Bid exceeds your cap room'); return; }
    }
    setPending(true); setError('');
    try {
      const result = await warRoomMarketAction(context, { type: 'revise', bid: b,
        form: { ...planForm(b.plan), ...change } });
      if (result.kind === 'plan') setOwn(o => ({ ...o, [result.plan.playerId]: ownBid(result.plan) }));
    } catch (cause) { setError(cause instanceof Error ? cause.message : unavailable); }
    finally { setPending(false); }
  };
  const reviewBid = async () => {
    if (!editor || pending || fresh?.error) return;
    if (bidRevision === undefined) {
      setError('Your bid version is unavailable; refresh before continuing');
      return;
    }
    setPending(true); setError(''); setNotice('');
    try {
      const salary = Number(editor.salary);
      const details = { salaryCents: Math.round(salary * 1e8), years: editor.years,
        revision: bidRevision, maxYears: value?.maxYears ?? 0, capRoom,
        writable: value?.writable === true, football: value?.football ?? '' };
      const result = editor.type === 'place' ? await warRoomMarketAction(context, {
        type: 'place', player: editor.player, ...details }) : await warRoomMarketAction(context, {
        type: 'raise', bid: editor.bid, ...details });
      if (result.kind === 'review') { setReview(result); setEditor(undefined); }
    } catch (cause) { setError(cause instanceof Error ? cause.message : unavailable); }
    finally { setPending(false); }
  };
  const reviewRosterAdd = async () => {
    if (!addEditor || pending || value?.rosterRevision === undefined) return;
    setPending(true); setError(''); setAddNotice('');
    try {
      const next = await reviewWarRoomRosterAdd(context, { player: addEditor.player.id,
        ...(addEditor.drop ? { drop: addEditor.drop } : {}), reason: addEditor.reason,
        revision: value.rosterRevision, writable: value.writable === true, football: value.football ?? '' });
      setAddReview(next); setAddEditor(undefined);
    } catch (cause) { setError(cause instanceof Error ? cause.message : unavailable); }
    finally { setPending(false); }
  };
  const confirmRosterAdd = async () => {
    if (!addReview || pending) return;
    setPending(true); setError(''); setAddNotice('');
    try {
      const result = await confirmWarRoomRosterAdd(context, addReview);
      if (result.kind === 'recorded') {
        setAddNotice(result.receipt.message); setAddReview(undefined); setAddEditor(undefined);
        setRefresh(n => n + 1);
      }
    } catch (cause) { setError(cause instanceof Error ? cause.message : unavailable); }
    finally { setPending(false); }
  };
  const confirmBid = async () => {
    if (!review || pending) return;
    setPending(true); setError(''); setNotice('');
    try {
      const result = await warRoomMarketAction(context, { type: 'confirm', request: review.request,
        football: review.football });
      if (result.kind === 'recorded') {
        setFresh({ bids: result.bids?.bids, revision: result.bids?.revision,
          capRoom: result.capRoom, error: result.refreshError });
        setNotice(result.receipt.message);
        setError(result.refreshError ?? '');
        setReview(undefined);
      }
    } catch (cause) { setError(cause instanceof Error ? cause.message : unavailable); }
    finally { setPending(false); }
  };
  const drag = useWarRoomDrag((id, over) => {
    const p = value?.players?.find(x => x.id === id);
    if (!p) return;
    if (over.closest('.bids')) { void bid(p); return; }
    const lane = over.closest<HTMLElement>('.lane2');
    if (lane && lane.dataset.lane === p.position && !targets.has(id)) void change(id);
  });
  const needPositions = new Set((needs ?? []).filter(n => n.state !== 'ok').map(n =>
    n.position === 'PK' ? 'K' : n.position));
  const players = value?.players?.filter(p => (filter !== 'Targets' || targets.has(p.id)) &&
    (filter !== 'My needs' || needPositions.has(p.position)) && (filter !== 'Affordable' ||
      capRoom !== undefined && p.minimumCents !== undefined && p.minimumCents <= capRoom) &&
      (!search.trim() || `${p.name} ${p.position} ${p.string ?? ''}`.toLowerCase()
        .includes(search.trim().toLowerCase())));
  const filterUnavailable = (name: string) => name === 'My needs' ? needs === undefined :
    name === 'Affordable' ? !context.native || capRoom === undefined : false;
  const canManageBids = context.native && context.native.owner !== false && !fresh?.error;
  const canManageTargets = !context.native || context.native.owner !== false;
  const addUnavailable = (player: WarRoomMarketPlayer) => !context.native || context.native.owner !== true
    ? 'Only the seated franchise owner can add players'
    : player.waiverStatus ? `${player.waiverStatus}; use the Waivers workflow instead`
      : player.freeAgent !== true ? 'This player is not confirmed in the free-agent pool'
        : value?.waiverPriority === undefined
          ? 'Waiver status is unavailable; direct add is unavailable'
          : value.writable !== true || !value.football || value.rosterRevision === undefined
            ? 'Add unavailable until the server reading is ready' : '';
  return <section className="mk3 ws on" aria-label="Market"><div className="tg"><div className="ch">
    <b>MY TARGETS</b><span>ranked per need</span></div><div className="tg-l">
    {value?.targets?.length ? [...warRoomTargetLanes(context, value)].map(([position, lane]) => <div
      className="lane2" key={position} data-lane={position}><div
        className="lh"><b>{position}</b><span>{unavailable}</span></div>
      {lane.map(({ entry, player }, i) => <div className="tgt" key={entry.id}>
        <span className="n">{i + 1}</span><WarRoomAct className="nm" onAction={() => select(player)}>
          {player.name}</WarRoomAct><small>{value.bids?.find(b => b.player === entry.id)?.contract ??
            unavailable}</small><div><WarRoomAct disabled={pending || !i} label={`Move ${player.name} up`}
              onAction={() => { void change(entry.id, lane[i - 1].entry.id); }}>↑</WarRoomAct>
          <WarRoomAct disabled={pending || i === lane.length - 1} label={`Move ${player.name} down`}
            onAction={() => { void change(entry.id, lane[i + 1].entry.id); }}>↓</WarRoomAct></div></div>)}
    </div>) : <WarRoomEmpty pending={!value && !reading.error} reason={context.native &&
      value?.targets === undefined ? 'Private targets are not available for this league' :
      'Star a player to add a target'} />}</div></div>
    <div><div className="ch"><b>FREE AGENT POOL</b><span>in-season · drag a row to a target lane or to BIDS
    </span></div><div className="phone-market-tools">
      <label className="fa-search">Search players or teams
        <input aria-label="Search free agents by player or team" value={search}
          onChange={e => setSearch(e.target.value)} placeholder="Search players or teams" />
      </label>
      <div className="waiver-priority" aria-label="Waiver priority"><b>WAIVER ORDER</b>
        <span>{value?.waiverPriority?.place
          ? `You are ${value.waiverPriority.place} of ${value.waiverPriority.total}`
          : 'Your place is unavailable'}</span>
        <small>{value?.waiverPriority?.source || unavailable}</small></div>
      <div className="bench-capacity" aria-label="Open bench slots"><b>OPEN BENCH</b>
        <span>{value?.openBenchSlots === undefined ? unavailable
          : `${value.openBenchSlots} slots open`}</span>
      </div>
    </div><div className="bench"><div className="filters">{['All', 'My needs', 'Targets',
      'Affordable'].map(f => <WarRoomAct key={f} className="chip" active={filter === f}
        disabled={filterUnavailable(f)} title={filterUnavailable(f) ? unavailable : undefined}
        onAction={() => setFilter(f)}>{f}</WarRoomAct>)}</div></div>
      <div className="fa-h">{['☆', 'POS', 'PLAYER', 'AGE', 'STRING', 'PTS', 'VALUE',
        'MIN BID'].map((h, i) =>
        <span key={i}>{h}</span>)}</div><div className="fa-list">
      {players?.length ? players.map(p =>
        <div className={`fa-r${bids?.some(b => b.player === p.id) ? ' bid' : ''}`}
          key={p.id} onPointerDown={e => { if ((e.target as HTMLElement).closest('button')) return;
          drag(e, p.id); }}><WarRoomAct className="st"
          label={`${targets.has(p.id) ? 'Unwatch' : 'Watch'} ${p.name}`} active={targets.has(p.id)}
          disabled={pending || !canManageTargets || !value?.targets} onAction={() => { void change(p.id); }}>
          {targets.has(p.id) ? '★' : '☆'}</WarRoomAct><span className="ps">{p.position}</span>
        <WarRoomAct className="nm" onAction={() => select(p)}>{p.name}</WarRoomAct>
      <span>{p.age ?? "—"}</span>
        <span>{p.string ?? "—"}</span><span>{p.points ?? "—"}</span>
        <span className="fa-value">{p.value ?? "—"}</span><span>{p.minimum ?? "—"}</span>
        <WarRoomAct className="add-player" disabled={pending || Boolean(addUnavailable(p))}
          title={addUnavailable(p) || undefined}
          label={`${p.freeAgent && !p.waiverStatus && value?.waiverPriority
            ? 'Add' : 'Unavailable'} ${p.name}`}
          onAction={() => { setError(''); setAddNotice(''); setAddReview(undefined);
            setAddEditor({ player: p, drop: '', reason: '' }); }}>
          {p.waiverStatus ? 'WAIVER'
            : p.freeAgent && value?.waiverPriority ? 'ADD' : 'UNAVAILABLE'}</WarRoomAct></div>) :
        <WarRoomEmpty pending={!value && !reading.error} reason={reading.error ?? (value?.players ===
          undefined ? value?.errors.join(' · ') ||
          'Pool could not be read' : 'No free agents match this view')} />}</div>
        {addEditor && <div className="bidb roster-add-editor" aria-label="Add free agent">
          <b>Add {addEditor.player.name} to your roster</b>
          <label>Release a player if the roster is full<select value={addEditor.drop} disabled={pending}
            onChange={e => setAddEditor({ ...addEditor, drop: e.target.value })}>
            <option value="">Keep current roster</option>
            {(value?.rosterPlayers ?? []).map(player => <option key={player.id} value={player.id}>
              {player.name}</option>)}
          </select></label>
          <label>Reason<textarea value={addEditor.reason} disabled={pending}
            onChange={e => setAddEditor({ ...addEditor, reason: e.target.value })} /></label>
          <WarRoomAct disabled={pending || !addEditor.reason.trim() || value?.rosterRevision === undefined}
            onAction={() => { void reviewRosterAdd(); }}>REVIEW ADD</WarRoomAct>
          <WarRoomAct disabled={pending} onAction={() => setAddEditor(undefined)}>CANCEL</WarRoomAct>
        </div>}
        {addReview && <div className="bidb roster-add-review" aria-label="Review roster add">
          <b>{addReview.preview.question}</b>
          {addReview.preview.fields.map((field, i) => <p key={i}><strong>{field.label}</strong>:
            {' '}{previewChange(field)}</p>)}
          {addReview.preview.blockers.map((blocker, i) => <p key={i} role="alert">{blocker}</p>)}
          <HoldToFire label="Confirm roster add" body={JSON.stringify(addReview.request)}
            disabled={pending || addReview.preview.blockers.length > 0}
            onConfirm={() => { void confirmRosterAdd(); }} />
          <WarRoomAct disabled={pending} onAction={() => setAddReview(undefined)}>CANCEL</WarRoomAct>
        </div>}
        {error && <p className="empty" role="alert">{error}</p>}
        {addNotice && <p role="status">{addNotice}</p>}</div>
    <div className="bcol"><div className="budget"><div className="t"><span>CAP ROOM</span><b>{money(capRoom)}
      </b><span className="lim">HARD LIMIT</span></div><div className="bar" /><div className="t">
        <span>committed {unavailable}</span><span>free {unavailable}</span></div></div>
      <div className="ch"><b>MY BIDS</b><span>{unavailable}</span></div><div className="bids">
        <div className="ch"><b>READY</b></div><WarRoomOwnBids reading={shown} kind="plan"
          revise={(b, change) => { void revise(b, change); }} />
        <div className="ch"><b>PLACED</b></div>
        <WarRoomOwnBids reading={shown} kind="placed"
          revise={canManageBids ? revise : undefined} />
        {editor && <div className="bidb" aria-label={editor.type === 'place' ? 'New bid' :
          `Revise ${editor.bid.name} bid`}>
          <b>{editor.type === 'place' ? `Bid on ${editor.player.name}` : `Revise ${editor.bid.name}`}</b>
          <label>Annual salary ($M) <input type="number" step="any" value={editor.salary}
            disabled={pending} onChange={e => setEditor({ ...editor, salary: e.target.value })} /></label>
          <label>Years <select value={editor.years} disabled={pending || !value?.maxYears}
            onChange={e => setEditor({ ...editor, years: Number(e.target.value) })}>
            <option value={0}>Choose years</option>
            {Array.from({ length: value?.maxYears ?? 0 }, (_, i) => i + 1).map(year =>
              <option key={year} value={year}>{year}</option>)}
          </select>{!value?.maxYears && <small>Contract length setting unavailable</small>}</label>
          <WarRoomAct disabled={pending || !value?.maxYears || !editor.salary || !editor.years ||
            fresh?.error !== undefined} onAction={() => { void reviewBid(); }}>REVIEW BID</WarRoomAct>
          <WarRoomAct disabled={pending} onAction={() => setEditor(undefined)}>CANCEL</WarRoomAct>
        </div>}
        {review && <div className="bidb" aria-label="Review bid">
          <b>{review.preview.question}</b>
          {review.preview.fields.map((field, i) => <p key={i}><strong>{field.label}</strong>:
            {' '}{previewChange(field)}</p>)}
          {review.preview.blockers.map((blocker, i) => <p key={i} role="alert">{blocker}</p>)}
          <HoldToFire label="Confirm private bid" body={JSON.stringify(review.request)}
            disabled={pending || review.preview.blockers.length > 0}
            onConfirm={() => { void confirmBid(); }} />
          <WarRoomAct disabled={pending} onAction={() => setReview(undefined)}>CANCEL</WarRoomAct>
        </div>}
        {fresh?.error && <WarRoomAct disabled={pending} onAction={() => {
          setFresh(undefined); setRefresh(n => n + 1);
        }}>REFRESH BIDS AND CAP</WarRoomAct>}
        {notice && <p role="status">{notice}</p>}
      </div><div className="nested" />
      <div className="recent"><div className="ch"><b>RECENT WINS</b></div>
        {value?.wins?.length ? value.wins.map(win => <div className="r win"
          key={`${win.player}-${win.decidedAt}`}><span className="player"><b>{win.name}</b>
            <small>{win.team}</small></span><span className="terms"><b>{money(win.salaryCents)}</b>
            <small>{win.years} {win.years === 1 ? 'year' : 'years'} · {
              new Date(win.decidedAt).toLocaleDateString()}</small></span></div>) :
          <WarRoomEmpty pending={!value && !reading.error}
            reason={value?.wins ? 'No completed auction wins yet' : undefined} />}</div></div>
  </section>;
}
