// Desktop v3 War Room, Trade desk, after TheWarRoom-mock-v3-FINAL.html (screens.js war.trade): the
// franchise strip, my shelf, the desk (send and get ledgers, both lenses, the effect, tune it) and their
// shelf. Reads, deal edits and the send are the War Room desk's own (useWarRoomTrade); a send is checked
// by the server's preview first and confirmed in a second step.
import { useState, type ReactNode } from 'react';
import { HQAct } from '../shell/HQAct';
import { useWarRoomTrade } from '../shell/WarRoomTrade';
import { executeWarRoomOffer, type WarRoomAsset, type WarRoomCatalog, type WarRoomDeal }
  from '../shell/warroomTradeReads';
import { directionLabel, directions, leanWords, lensValue, signed, verdictWords, type Direction,
  type LensSide, type WarRoomLens } from '../shell/warroomLens';
import type { WarRoomContext, WarRoomHeaderReading } from '../shell/warroomReads';
import { newKey } from '../commissioner/forms';
import { money } from '../shell/hqModel';

const OFF = ['QB', 'RB', 'WR', 'TE'];
const FILTERS = ['ALL', 'OFF', 'DEF', 'PICKS'] as const;
const isPick = (a: WarRoomAsset) => Boolean(a.pick) || a.token.startsWith('pick:');
/** The shelf filters: offence, defence (anything else that is not a pick), or picks. */
export const shelfPasses = (a: WarRoomAsset, f: typeof FILTERS[number]) => f === 'ALL'
  || (f === 'PICKS' ? isPick(a) : !isPick(a) && OFF.includes(a.position ?? '') === (f === 'OFF'));
const FULL = 10;

function LensBar({ who, side, ends }: { who: 'you' | 'they'; side?: LensSide; ends: [string, string] }) {
  const width = side ? Math.min(Math.abs(side.net), FULL) / FULL * 50 : 0;
  return <div className={`lens ${who}`}><div><b>{who === 'you' ? 'YOUR LENS' : 'THEIR LENS'}</b>
    <small>{side ? `${directionLabel[side.direction].toLowerCase()} · now ${Math.round(side.nowWeight * 100)
    } / later ${100 - Math.round(side.nowWeight * 100)}` : 'choose a partner'}</small></div>
    <div className="lbar"><span>{ends[0]}</span><span>{ends[1]}</span>
      <i className={`p ${who}`} style={{ width: `${width}%`,
        marginLeft: side && side.net < 0 ? `${50 - width}%` : '50%' }} /></div>
    <b className="mono">{side ? signed(side.net) : '—'}</b></div>;
}

export function WarTradeV3({ context, catalog, deal, change, lens, lensPending, previewNonce, header, name,
  choose, directing, tray }: { context: WarRoomContext; catalog?: WarRoomCatalog; deal: WarRoomDeal;
  change: (d: WarRoomDeal) => void; lens?: WarRoomLens; lensPending?: boolean; previewNonce?: number;
  header?: WarRoomHeaderReading; name?: string; choose?: (d: Direction) => void;
  directing: { pending?: boolean; error?: string }; tray: ReactNode }) {
  const t = useWarRoomTrade({ context, catalog, deal, change, select: () => {}, lens, lensPending,
    previewNonce });
  const { mine, theirs, held, preview, delta, partner, send, setSend, fingerprint, ranked, teamName, toggle,
    names, waiting } = t;
  const [filter, setFilter] = useState<typeof FILTERS[number]>('ALL');
  const owner = context.native?.owner !== false;
  const you = held?.you;
  const verdict = held && verdictWords(held);
  const value = (a: WarRoomAsset, who: 'you' | 'them') => lensValue(held?.assets.get(a.token), who);
  const row = (a: WarRoomAsset, side: 'give' | 'get') => {
    const from = a.pick?.original && a.pick.original !== (side === 'give' ? context.mine : deal.partner)
      ? `from ${teamName(a.pick.original)}` : '';
    return <HQAct key={a.token} className={`trow ${isPick(a) ? 'pick' : 'pl'}${deal[side].includes(a.token)
      ? ' on' : ''}`} label={`${deal[side].includes(a.token) ? 'Remove' : 'Add'} ${a.name}`}
      onAction={() => toggle(side, a.token)}>
      <span className="pos">{a.position ?? (isPick(a) ? 'PK' : '—')}</span>
      <span className="nm ell">{a.name} <small>{isPick(a) ? from : a.contract ?? ''}</small></span>
      <span className="ag">{a.age === undefined ? '' : Math.floor(a.age)}</span>
      <b>{value(a, 'you')}</b><span /></HQAct>;
  };
  const ledger = (side: 'give' | 'get') => {
    const team = side === 'give' ? mine : theirs;
    const sums = held?.them && deal[side].length
      ? side === 'give' ? [held.you.out, held.them.in] : [held.you.in, held.them.out] : undefined;
    return <div className={`led ${side}`}><div className="lh"><b>{side === 'give' ? 'YOU SEND' :
      'YOU GET'}</b>
      <span>{deal[side].length} object{deal[side].length === 1 ? '' : 's'}</span></div>
      <div className="lcol"><span /><span /><span>AGE</span><span>YOU</span><span>THEM</span></div>
      <div className="lb">{!deal[side].length
        ? <div className="le">{side === 'give' ? 'Nothing sent.' : 'Nothing asked for.'}</div>
        : deal[side].map(token => {
          const a = team?.assets.find(x => x.token === token);
          return <HQAct key={token} className="trow pl" label={`Remove ${a?.name ?? token}`}
            style={{ width: '100%' }}
            onAction={() => toggle(side, token)}><span className="pos">{a?.position ?? '!'}</span>
            <span className="nm ell">{a?.name ?? `${token} · no longer available`}</span>
            <span className="ag">{a?.age === undefined ? '' : Math.floor(a.age)}</span>
            <b>{a ? value(a, 'you') : '—'}</b><b className="tm">{a ? value(a, 'them') : '—'}</b></HQAct>;
        })}</div>
      <div className="lt"><span>{side === 'give' ? 'total sent' : 'total received'}</span>
        <b className="b">{sums ? sums[0].toFixed(1) : '—'}</b><b className="tm">{sums ? sums[1].toFixed(1) :
          '—'}</b>
      </div></div>;
  };
  const blocked = Boolean(preview.error) || !preview.reading || Boolean(preview.reading.reasons.length);
  const empty = !deal.give.length && !deal.get.length;
  const sendNow = () => {
    if (!catalog || !send.request) return;
    setSend({ ...send, pending: true });
    void executeWarRoomOffer(context, catalog, deal, send.request).then(receipt => setSend({ pending: false,
      confirm: false, done: `Offer ${receipt.entry} recorded and read back.` }),
    cause => setSend(s => ({ ...s, pending: false, confirm: true, error: String(cause) })));
  };
  const ages = header?.age ?? [];
  const peak = Math.max(1, ...ages.map(b => b.value));
  return <div className="wbody scr scr-war"><div className="trade">
    <div className="tstrip">
      <div className="who"><h2>{name ?? '—'}</h2><span className="mono d">
        {[header?.record, header?.standing].filter(Boolean).join(' · ') || '—'}</span></div>
      <div><span className="tk">DIRECTION{directing.error ? ` · not recorded: ${directing.error}` : ''}</span>
        <div className="tseg">{directions.map(d => <HQAct key={d} className={you?.set && you.direction === d
          ? 'on' : ''} disabled={!choose || !you || directing.pending}
          title="Your direction is shown to the whole league. Your lens follows it."
          onAction={() => choose?.(d)}>{directionLabel[d].toUpperCase()}</HQAct>)}</div></div>
      <div><span className="tk">{header?.starterN ? `BEST ${header.starterN}` : 'BEST'}</span>
        <div className="big mono">{header?.valuesPreparing ? '…' : header?.best ?? '—'}
          {' '}<small>{header?.rank ?? ''}</small></div></div>
      <div><span className="tk">VALUE BY AGE</span>{ages.some(b => b.players)
        ? <div className="ages">{ages.map((b, i) => <span key={b.from}><i
          style={{ height: `${Math.max(1, b.value / peak * 24)}px` }} />{b.open ? `${b.from}+`
          : i === 0 && ages.length > 1 ? `≤${ages[1].from - 1}` : b.from}</span>)}</div>
        : <div className="d mono">not available yet</div>}</div>
      <div><span className="tk">NEEDS · NEXT SEASON</span>{you?.needs.length
        ? <div className="needs">{you.needs.map(n => <span key={n.position} className={n.state}
          title={`${n.have} starter-grade ${n.position} signed · the lineup starts ${n.starts}`}>
          <b>{n.position}</b>{n.state}</span>)}</div>
        : <div className="d mono">{lens?.preparing ? 'values being prepared' : waiting}</div>}</div>
      <div><span className="tk">CAP ROOM</span><div className="runway">{(header?.runway ?? []).slice(0, 3)
        .map(y => <span key={y.season} style={{ display: 'contents' }}><span>{y.season}</span>
          <i><u style={{ width: `${y.max && y.room !== undefined ? Math.max(0, Math.min(100, y.room / y.max
            * 100)) : 0}%` }} /></i><b>{money(y.room)}</b></span>)}</div></div>
    </div>
    <div className="tcol"><div className="thd"><b>MY SHELF</b><span>click to SEND</span></div>
      <div className="tflt">{FILTERS.map(f => <HQAct key={f} className={filter === f ? 'on' : ''}
        onAction={() => setFilter(f)}>{f}</HQAct>)}</div>
      <div className="tcolh"><span /><span /><span>AGE</span><span>YOU</span><span /></div>
      <div className="tlist scrollkeep">{mine?.assets.length
        ? ranked(mine.assets).filter(a => shelfPasses(a, filter)).map(a => row(a, 'give'))
        : <div className="le">{catalog ? 'Nothing on this roster to trade.' : 'Reading the shelves…'}</div>}
      </div></div>
    <div className="tmid">
      <div className="vtabs"><HQAct className="on">CURRENT DRAFT</HQAct>
        <span>unsent · only you see this</span></div>
      <div className="leds">{ledger('give')}{ledger('get')}</div>
      <div className="lensw"><LensBar who="you" side={held?.you} ends={['you lose', 'you win']} />
        <LensBar who="they" side={held?.them} ends={['they lose', 'they win']} />
        <div className="verd">{verdict ? <><span className={`tag ${verdict.className === 'ok' ? '' : 'o'}`}>
          {verdict.tag}</span><span>{verdict.words}</span></> : <span>{waiting}</span>}</div></div>
      <div className="aft">{preview.reading?.teams.flatMap(tm => tm.figures.map(f => <div
        key={`${tm.team}:${f.label}`}><span className="tk">{f.label.toUpperCase()}</span>
        <b className="mono">{f.after}</b><small>was {f.before}</small></div>))}
        {held && held.verdict !== 'empty' && <div><span className="tk">YOUR EDGE</span>
          <b className="mono">{held.you.after.toFixed(1)}</b><small>was
            {held.you.before.toFixed(1)}</small></div>}
        {delta.reading && <div><span className="tk">PROJ / WEEK</span><b className="mono">
          {delta.reading.you.unprojected.length ? '—' : signed(delta.reading.you.perWeek)}</b>
          <small>your best lineup</small></div>}</div>
      {preview.reading?.reasons.map(r => <div key={r} className="tsec"><div className="thd"><b>BLOCKED</b>
        <span>{r}</span></div></div>)}
      <div className="tsec"><div className="thd"><b>LINEUP CHANGES</b><span>{delta.reading?.you.changes.length
        ? `${delta.reading.you.changes.length} change${delta.reading.you.changes.length === 1 ? '' : 's'}`
        : 'your starters do not change'}</span></div>
        {delta.reading?.you.changes.map(c => <div key={`${c.kind}:${c.player}`} className="row"
          style={{ gridTemplateColumns: '34px minmax(0,1fr) auto' }}><span className="pos">{c.position}</span>
          <span className="ell">{names.get(c.player) ?? c.player}</span><span className="m">{c.kind ===
            'enters'
            ? 'starts' : c.kind === 'promoted' ? 'moves up' : c.kind === 'benched' ? 'to the bench'
              : 'leaves the lineup'}</span></div>)}</div>
      <div className="tsec grow"><div className="thd"><b>TUNE IT</b>
        <span>single changes that would make it acceptable</span></div>
        <div className="tunes scrollkeep">{held?.tune.length ? held.tune.map(s => <div
          key={`${s.kind}:${s.token}`}
          className="row" style={{ gridTemplateColumns: '54px minmax(0,1fr) auto auto' }}>
          <span className="tag">{s.kind.toUpperCase()}</span><span className="ell">{s.kind === 'trim'
            ? `Take ${names.get(s.token) ?? s.token} out` : s.kind === 'ask'
              ? `Also ask for ${names.get(s.token) ?? s.token}` :
                `Add ${names.get(s.token) ?? s.token}`}</span>
          <span className="m">you {signed(s.you)} · them {signed(s.them)}</span>
          <HQAct className="btn sm" onAction={() => toggle(s.kind === 'ask' ? 'get' : 'give', s.token)}>
            Apply</HQAct></div>)
          : <div className="le">{!held ? waiting : held.verdict === 'empty' ? 'Build a draft first.'
            : 'Nothing smaller keeps both sides whole.'}</div>}</div></div>
    </div>
    <div className="tcol rt"><div className="tpart"><div className="tsel"><select aria-label="Trade partner"
      value={deal.partner} onChange={e => change({ ...deal, partner: e.target.value, get: [] })}>
      <option value="">Choose a partner</option>
      {catalog?.teams.filter(x => x.id !== context.mine).map(x => <option key={x.id} value={x.id}>{x.name}
      </option>)}</select>{held?.them && <span className="tag">
        {directionLabel[held.them.direction].toUpperCase()}</span>}</div>
      {deal.partner ? <><div className="kv"><span>Record · cap room</span><span>{partner.reading?.record ??
        '—'}
        {' · '}{partner.reading?.room == null ? '—' : money(partner.reading.room)}</span></div>
        <div className="kv"><span>Needs</span><span>{held?.them ? held.them.needs.filter(x => x.state !==
          'ok')
          .map(x => x.position).join(' · ') || 'none' : '—'}</span></div>
        <div className="kv"><span>Spare</span><span>{held?.them ? held.them.spare.join(' · ') || 'none' : '—'}
        </span></div><p>{held?.them ? `${held.them.set ? '' : 'Our read: '}${leanWords(held.them)}` :
          waiting}</p></>
        : <p>Choose a partner to see how they would read a deal.</p>}</div>
      <div className="thd"><b>THEIR SHELF</b><span>click to GET</span></div>
      <div className="tcolh"><span /><span /><span>AGE</span><span>YOU</span><span /></div>
      <div className="tlist scrollkeep">{theirs?.assets.length ? ranked(theirs.assets).map(a => row(a, 'get'))
        : <div className="le">{deal.partner ? 'Reading their shelf…' :
          'No partner chosen.'}</div>}</div></div>
    <div className="tray ttray">{tray}<span className="msg ell" aria-live="polite">{send.error ? send.error
      : send.done ? send.done : send.confirm ? send.fingerprint === fingerprint
        ? `Confirm offer to ${teamName(deal.partner)}? The league's preview has no blockers.`
        : 'The deal changed; cancel and review the new preview.' : empty ? 'The desk is empty.'
        : preview.reading?.reasons.length ? 'Blocked: see the desk.' : ''}</span><span className="sp" />
      {context.native && (!send.confirm
        ? <HQAct className="btn go" disabled={!owner || send.pending || preview.pending || !catalog
          || !deal.partner || empty || blocked} title={!owner ? 'Only this franchise\'s owner sends offers'
            : undefined} onAction={() => setSend({ pending: false, confirm: true, request: newKey(),
            fingerprint })}>Send to {deal.partner ? teamName(deal.partner) : '…'}</HQAct>
        : <><HQAct className="btn" disabled={send.pending}
          onAction={() => setSend({ pending: false, confirm: false })}>Cancel</HQAct>
          <HQAct className="btn go" disabled={send.pending || send.fingerprint !== fingerprint}
            onAction={sendNow}>{send.pending ? 'Sending…' : 'Confirm send'}</HQAct></>)}
    </div>
  </div></div>;
}
