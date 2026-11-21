// Desktop v3 War Room, Valuations and pool, after TheWarRoom-mock-v3-FINAL.html (screens.js war.pool):
// the free-agent pool, my bids and the budget with recent winning bids. Reads and every action are the
// War Room market's own (useWarRoomMarket): bids are reviewed by the server and held to confirm.
import { useState, type ReactNode } from 'react';
import { HQAct } from '../shell/HQAct';
import { HoldToFire } from '../commissioner/HoldToFire';
import { previewChange } from '../commissioner/preview';
import { useWarRoomMarket } from '../shell/WarRoomMarket';
import type { WarRoomContext } from '../shell/warroomReads';
import type { WarRoomMarketPlayer } from '../shell/warroomMarketReads';
import type { LensNeed } from '../shell/warroomLens';
import { money, unavailable } from '../shell/hqModel';

/** A figure the league has not supplied shows as a dash; the reason is in its tooltip. */
const fig = (v?: string) => !v || v === unavailable ? '—' : v;

const IDP = ['DL', 'DE', 'DT', 'LB', 'DB', 'CB', 'S'];
const POS = ['All', 'QB', 'RB', 'WR', 'TE', 'IDP'];
/** The mock's position chips over the pool. */
export const posMatch = (chip: string, position: string) =>
  chip === 'All' || (chip === 'IDP' ? IDP.includes(position) : position === chip);

const St = ({ children }: { children: ReactNode }) =>
  <div className="st"><div className="eng">{children}</div></div>;
function Panel({ title, badge, children, foot, style, tag }: { title: string; badge: ReactNode;
  children: ReactNode; foot?: ReactNode; style?: React.CSSProperties; tag?: ReactNode }) {
  return <section className="pc panel sz-L" style={style}>
    <div className="ph">{badge}<b className="pt">{title}</b><span className="sp" />{tag}</div>
    <div className="pb">{children}</div>
    {foot && <div className="pf"><span className="ell">{foot}</span></div>}</section>;
}

export function WarPoolV3({ context, room, needs, select, badge }: { context: WarRoomContext;
  room?: number; needs?: LensNeed[]; select?: (p: WarRoomMarketPlayer) => void; badge: ReactNode }) {
  const m = useWarRoomMarket(context, room, needs);
  const [pos, setPos] = useState('All');
  const { value, reading, pending, editor, review, addEditor, addReview } = m;
  const rows = (m.players ?? []).filter(p => posMatch(pos, p.position));
  const owner = context.native?.owner !== false;
  const placed = m.bids ?? [];
  return <div className="wbody scr scr-war"><div className="lay" style={{
    gridTemplateColumns: 'minmax(0,1.6fr) minmax(0,1fr)',
    gridTemplateRows: 'minmax(0,1.2fr) minmax(0,1fr)' }}>
    <Panel title="Free-agent pool" badge={badge} style={{ gridRow: '1/3' }}
      foot={value?.waiverPriority?.place ? `waiver order: you are ${value.waiverPriority.place} of ${
        value.waiverPriority.total} · ${value.openBenchSlots ?? '—'} bench slots open`
        : 'a placed bid is reviewed by the league, then held to confirm'}>
      <div className="fbar">{POS.map(c => <HQAct key={c} className="chip" selected={pos === c}
        onAction={() => setPos(c)}>{c}</HQAct>)}
        <span className="sp" /><input className="srch" aria-label="Search free agents" value={m.search}
          placeholder="Search players or teams" onChange={e => m.setSearch(e.target.value)} /></div>
      {addEditor && <div className="st"><div className="eng" aria-label="Add free agent">
        <b>Add {addEditor.player.name} to your roster</b>
        <label>Release if the roster is full <select value={addEditor.drop} disabled={pending}
          onChange={e => m.setAddEditor({ ...addEditor, drop: e.target.value })}>
          <option value="">Keep current roster</option>
          {(value?.rosterPlayers ?? []).map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
        </select></label>
        <label>Reason <input value={addEditor.reason} disabled={pending}
          onChange={e => m.setAddEditor({ ...addEditor, reason: e.target.value })} /></label>
        <div className="tbar"><HQAct className="btn sm" disabled={pending || !addEditor.reason.trim()}
          onAction={() => { void m.reviewRosterAdd(); }}>Review add</HQAct>
          <HQAct className="btn sm" onAction={() => m.setAddEditor(undefined)}>Cancel</HQAct>
          </div></div></div>}
      {addReview && <div className="st"><div className="eng" aria-label="Review roster add">
        <b>{addReview.preview.question}</b>
        {addReview.preview.fields.map((f, i) => <p key={i}>{f.label}: {previewChange(f)}</p>)}
        {addReview.preview.blockers.map((b, i) => <p key={i} role="alert">{b}</p>)}
        <HoldToFire label="Confirm roster add" body={JSON.stringify(addReview.request)}
          disabled={pending || addReview.preview.blockers.length > 0}
          onConfirm={() => { void m.confirmRosterAdd(); }} />
        <HQAct className="btn sm" onAction={() => m.setAddReview(undefined)}>Cancel</HQAct></div></div>}
      {m.error && <St><span role="alert">{m.error}</span></St>}
      {m.addNotice && <St><span role="status">{m.addNotice}</span></St>}
      {!value ? <St>{reading.error ?? 'Reading the pool…'}</St>
        : !rows.length ? <St>{value.players === undefined
          ? value.errors.join(' · ') || 'The pool could not be read'
          : 'No free agents match this view.'}</St>
          : <table className="t"><tbody><tr><th>Player</th><th>Pos</th><th>NFL string</th>
            <th className="n">Season pts</th><th className="n">Value</th><th className="n">Min bid</th>
            <th /></tr>
            {rows.map(p => {
              const mine = placed.some(b => b.player === p.id);
              const add = m.addUnavailable(p);
              return <tr key={p.id}>
                <td>{select ? <HQAct className="lk" onAction={() => select(p)}>{p.name}</HQAct> : p.name}</td>
                <td><span className="pos">{p.position}</span></td><td className="d">{fig(p.string)}</td>
                <td className="n">{fig(p.points)}</td><td className="n">{fig(p.value)}</td>
                <td className="n">{fig(p.minimum)}</td>
                <td className="acts">{context.native && <HQAct className="btn sm"
                  disabled={pending || !owner || mine} title={!owner ? 'Only this franchise\'s owner bids'
                    : mine ? 'You already bid on this player' : undefined}
                  onAction={() => { void m.bid(p); }}>{mine ? 'Bid placed' : 'Bid'}</HQAct>}
                  {p.freeAgent && !p.waiverStatus && <HQAct className="btn sm"
                    disabled={pending || Boolean(add)}
                    title={add || undefined} onAction={() => {
                      m.setError(''); m.setAddNotice(''); m.setAddReview(undefined);
                      m.setAddEditor({ player: p, drop: '', reason: '' }); }}>Add</HQAct>}</td></tr>;
            })}</tbody></table>}
    </Panel>
    <Panel title="My bids" badge={badge}
      tag={placed.length ? <span className="tag o">{placed.length}</span> : null}
      foot={m.capRoom === undefined ? 'cap room unavailable' : `${money(m.capRoom)} cap room left`}>
      {editor && <div className="st"><div className="eng" aria-label="Bid">
        <b>{editor.type === 'place' ? `Bid on ${editor.player.name}` : `Raise ${editor.bid.name}`}</b>
        <label>Annual salary ($M) <input type="number" step="any" value={editor.salary} disabled={pending}
          onChange={e => m.setEditor({ ...editor, salary: e.target.value })} /></label>
        <label>Years <select value={editor.years} disabled={pending || !value?.maxYears}
          onChange={e => m.setEditor({ ...editor, years: Number(e.target.value) })}>
          <option value={0}>Choose</option>
          {Array.from({ length: value?.maxYears ?? 0 }, (_, i) => i + 1).map(y =>
            <option key={y} value={y}>{y}</option>)}</select></label>
        <div className="tbar"><HQAct className="btn sm" disabled={pending || !editor.salary || !editor.years}
          onAction={() => { void m.reviewBid(); }}>Review bid</HQAct>
          <HQAct className="btn sm" onAction={() => m.setEditor(undefined)}>Cancel</HQAct></div></div></div>}
      {review && <div className="st"><div className="eng" aria-label="Review bid">
        <b>{review.preview.question}</b>
        {review.preview.fields.map((f, i) => <p key={i}>{f.label}: {previewChange(f)}</p>)}
        {review.preview.blockers.map((b, i) => <p key={i} role="alert">{b}</p>)}
        <HoldToFire label="Confirm private bid" body={JSON.stringify(review.request)}
          disabled={pending || review.preview.blockers.length > 0}
          onConfirm={() => { void m.confirmBid(); }} />
        <HQAct className="btn sm" onAction={() => m.setReview(undefined)}>Cancel</HQAct></div></div>}
      {m.notice && <St><span role="status">{m.notice}</span></St>}
      {!value ? <St>Reading your bids…</St> : !placed.length ? <St>{owner
        ? 'No bids yet. Pick Bid on a player'
        + ' in the pool.' : 'Bids are visible only to this franchise\'s owner.'}</St>
        : placed.map(b => <div key={b.player} className="row"
          style={{ gridTemplateColumns: 'minmax(0,1fr) auto auto auto' }}>
          <span className="ell">{b.name}</span><span className="v">{b.contract}</span>
          <span className="tag">{b.state}</span>
          <span className="acts">{b.bidId && owner && <HQAct className="btn sm" disabled={pending}
            onAction={() => { void m.revise(b, {}); }}>Raise</HQAct>}</span></div>)}
    </Panel>
    <Panel title="Budget and recent winning bids" badge={badge}
      foot="winning bids are public once the auction closes">
      <div className="row" style={{ gridTemplateColumns: 'minmax(0,1fr) auto' }}>
        <span className="k">CAP ROOM</span><span className="v">{money(m.capRoom)}</span></div>
      {!value?.wins ? <St>Recent winning bids are not available.</St>
        : !value.wins.length ? <St>No completed auction wins yet.</St>
          : value.wins.map(w => <div key={`${w.player}-${w.decidedAt}`} className="row"
            style={{ gridTemplateColumns: 'minmax(0,1fr) auto minmax(0,1fr) auto' }}>
            <span className="ell">{w.name}</span><span className="v">{money(w.salaryCents)}</span>
            <span className="ell d">{w.team}</span>
            <span className="m">{new Date(w.decidedAt).toLocaleDateString('en-US', {
              month: 'short', day: 'numeric',
              timeZone: 'America/Chicago' })}</span></div>)}
    </Panel>
  </div></div>;
}
