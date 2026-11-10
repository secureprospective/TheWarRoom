import { HQAct as WarRoomAct } from './HQAct';
import { useState } from 'react';
import type { WarRoomContext } from './warroomReads';
import { useWarRoomRead } from './warroomReads';
import { readWarRoomDelta, readWarRoomPreview, readWarRoomPartner, type WarRoomAsset, type WarRoomCatalog,
  type WarRoomDeal, type WarRoomDeltaSide } from './warroomTradeReads';
import { directionLabel, leanWords, lensValue, lensWords, signed, verdictWords, type LensAsset,
  type LensSide, type WarRoomLens } from './warroomLens';
import { CompactAbsent, WarRoomEmpty } from './WarRoomHeader';
import { absent, money, unavailable } from './hqModel';
const offence = ['QB', 'RB', 'WR', 'TE'];
const needWords = {
  hole: 'Fewer starter-grade players signed for next season than the lineup must start',
  thin: 'Exactly as many starter-grade players as the lineup must start, where it can start more',
};
const filters = ['ALL', 'OFF', 'DEF', 'PICKS'] as const;
const isPick = (a: WarRoomAsset) => Boolean(a.pick) || a.token.startsWith('pick:');
const passes = (a: WarRoomAsset, filter: typeof filters[number]) => filter === 'ALL' ||
  (filter === 'PICKS' ? isPick(a) : !isPick(a) && offence.includes(a.position ?? '') === (filter === 'OFF'));
// The scale is full at a swing of ten points a game in a side's best N: more than any one player moves it.
const fullScale = 10;
export function WarRoomAssetBlock({ asset, lens, note, used, select, toggle, from, inTray }: {
  asset: WarRoomAsset; lens?: LensAsset; note?: string; used?: boolean; inTray?: boolean;
  select: (a: WarRoomAsset) => void; toggle: () => void; from: string }) {
  const age = asset.age ?? lens?.age;
  const why = lens && [lens.words, lens.valued ? `this season ${lens.now.toFixed(1)} · long term ${
    lens.dynasty.toFixed(1)}` : ''].filter(Boolean).join(' · ');
  return <div className={`blk ${isPick(asset) ? 'pick' : ''} ${used ? 'sel' : ''} ${inTray ? 'trow' : ''}`}
    draggable onDragStart={e => e.dataTransfer.setData('application/warroom-asset', JSON.stringify({
      from, token: asset.token }))}>
    <WarRoomAct className="ps" label={`${used ? 'Remove' : 'Add'} ${asset.name}`} onAction={toggle}>
      {asset.position ?? (isPick(asset) ? 'PK' : '—')}</WarRoomAct>
    <WarRoomAct className="nm" onAction={() => select(asset)} title={asset.contract ?? note}>{asset.name}
      <small>{isPick(asset) ? note : asset.contract ?? unavailable}</small></WarRoomAct>
    <span className="ag">{age === undefined ? '' : Math.floor(age)}</span>
    <span className={inTray ? 'v2' : 'vl'} title={why}>{lensValue(lens, 'you')}</span>
    {inTray && <span className="v3" title={why}>{lensValue(lens, 'them')}</span>}
  </div>;
}
export function WarRoomDeltaSidePanel({ side, name, names }: { side?: WarRoomDeltaSide; name: string;
  names: Map<string, string> }) {
  const player = (id: string) => names.get(id) ?? id;
  if (!side) return null;
  const known = !side.unprojected.length;
  return <div className="wide half"><span className="h">{name} LINEUP · {known ?
    `${signed(side.perWeek)} / WEEK` : 'PROJECTION MISSING'}</span>
    {side.changes.length ? side.changes.map(c => <div className="swap" key={`${c.kind}:${c.player}`}>
      <span className="sl">{c.position}</span><span>{player(c.player)}</span><span>{c.kind === 'enters' ?
        'starts' : c.kind === 'promoted' ? 'moves up to start' : c.kind === 'benched' ? 'goes to the bench' :
          'leaves the lineup'}</span><span /></div>) :
      <div className="swap"><span /><span>No starter changes</span></div>}
    {side.unprojected.map(id => <div className="swap" key={id}><span /><span>{player(id)}</span>
      <span>no projection yet</span></div>)}</div>;
}
function Lens({ label, side, ends }: { label: string; side?: LensSide; ends: [string, string] }) {
  const width = side ? Math.min(Math.abs(side.net), fullScale) / fullScale * 50 : 0;
  return <div className="lens"><span className="lb">{label}<small>{side ? lensWords(side) :
    'choose a partner'}</small></span><div className="scale">{side && <i className={`f ${side.net < 0 ?
      'neg' : ''}`} style={{ left: `${side.net >= 0 ? 50 : 50 - width}%`, width: `${width}%` }} />}
    <span className="ends"><span>{ends[0]}</span><span>{ends[1]}</span></span></div>
    <span className="n">{side ? signed(side.net) : <CompactAbsent />}</span></div>;
}
export function WarRoomTrade({ context, catalog, deal, change, select, lens, lensPending }: {
  context: WarRoomContext; catalog?: WarRoomCatalog; deal: WarRoomDeal; change: (d: WarRoomDeal) => void;
  select: (a: WarRoomAsset) => void; lens?: WarRoomLens; lensPending?: boolean }) {
  const [filter, setFilter] = useState<typeof filters[number]>('ALL');
  const mine = catalog?.teams.find(t => t.id === context.mine);
  const theirs = catalog?.teams.find(t => t.id === deal.partner);
  const key = JSON.stringify([context.native?.league, context.mine, context.week, catalog?.revision, deal]);
  const partner = useWarRoomRead(JSON.stringify([context.native?.league, deal.partner, context.week]),
    () => readWarRoomPartner(context, deal.partner));
  const preview = useWarRoomRead(key, async () => catalog && deal.partner &&
    deal.give.length + deal.get.length ? readWarRoomPreview(context, catalog, deal) : undefined);
  const delta = useWarRoomRead(key, async () => catalog ? readWarRoomDelta(context, catalog, deal) :
    undefined);
  const held = lens && !lens.preparing ? lens : undefined;
  const value = (a: WarRoomAsset) => held?.assets.get(a.token);
  const ranked = (assets: WarRoomAsset[]) => held ? [...assets].sort((a, b) =>
    (value(b)?.you ?? -1) - (value(a)?.you ?? -1) || (value(b)?.dynasty ?? -1) - (value(a)?.dynasty ?? -1)) :
    assets;
  const teamName = (id: string) => catalog?.teams.find(t => t.id === id)?.name ?? id;
  const toggle = (side: 'give' | 'get', token: string) => change({ ...deal, [side]: deal[side].includes(
    token) ? deal[side].filter(t => t !== token) : [...deal[side], token] });
  const block = (a: WarRoomAsset, side: 'give' | 'get', inTray = false) => {
    const owner = side === 'give' ? context.mine : deal.partner;
    return <WarRoomAssetBlock key={a.token} asset={a} lens={value(a)} inTray={inTray} from={owner}
      note={a.pick?.original && a.pick.original !== owner ? `from ${teamName(a.pick.original)}` : ''}
      select={select} used={deal[side].includes(a.token)} toggle={() => toggle(side, a.token)} />;
  };
  const names = new Map(catalog?.teams.flatMap(t => t.assets.map(a => [a.token, a.name] as const)));
  const verdict = held && verdictWords(held);
  const total = (side: 'give' | 'get') => {
    if (!held?.them || !deal[side].length) return undefined;
    return side === 'give' ? [held.you.out, held.them.in] : [held.you.in, held.them.out];
  };
  const waiting = lens?.preparing ? 'Values are being prepared' : absent(lensPending);
  const n = held?.starterN;
  return <section className="desk ws on" aria-label="Trade desk"><div className="shelf">
    <div className="ch"><b>MY SHELF</b><span>drag to SEND</span></div><div className="bench">
      <div className="filters">{filters.map(p => <WarRoomAct key={p} className="chip" active={p === filter}
        onAction={() => setFilter(p)}>{p}</WarRoomAct>)}</div></div><div className="list">
      {mine?.assets.length ? ranked(mine.assets).filter(a => passes(a, filter)).map(a => block(a, 'give')) :
        <WarRoomEmpty />}</div></div>
    <div className="build"><div className="vers" role="tablist"><WarRoomAct role="tab"
      selected> CURRENT DRAFT</WarRoomAct><WarRoomAct className="add" disabled title={unavailable}>+ VERSION
      </WarRoomAct><span className="t">unsent · only you see this</span></div>
      <div className="trays">{(['give', 'get'] as const).map(side => {
        const team = side === 'give' ? mine : theirs;
        const sums = total(side);
        return <div className="tray2" data-side={side === 'give' ? 'me' : 'them'} key={side}
          onDragOver={e => e.preventDefault()} onDrop={e => {
          e.preventDefault();
          try { const raw: unknown = JSON.parse(e.dataTransfer.getData('application/warroom-asset'));
            if (!raw || typeof raw !== 'object') return;
            const r = raw as { from?: unknown; token?: unknown };
            if (r.from === team?.id && typeof r.token === 'string' && team?.assets.some(a =>
              a.token === r.token) && !deal[side].includes(r.token)) toggle(side, r.token);
          } catch { /* An external drag is not a trade asset. */ }
        }}><div className="ch"><b>{side === 'give' ? 'YOU SEND' : 'YOU GET'}</b><span>{deal[side].length ?
          `${deal[side].length} object${deal[side].length === 1 ? '' : 's'}` : ''}</span></div>
          <div className="tcols"><span /><span /><span>AGE</span><span className="y">YOU</span>
            <span className="t">THEM</span></div><div className="zone">{deal[side].length ? team?.assets
              .filter(a => deal[side].includes(a.token)).map(a => block(a, side, true)) :
              <div className="hint">Drag {side === 'give' ? 'your' : 'their'} players or picks here</div>}
          </div><div className="tot" title={n ? `What these take from or add to each side's best ${n}, ` +
            'counted together. One row alone can read differently.' : undefined}>
            <span>{side === 'give' ? 'total sent' : 'total received'}</span>
            {sums ? <b><span className="y">{sums[0].toFixed(1)}</span><span className="t">
              {sums[1].toFixed(1)}</span></b> : <b><CompactAbsent /></b>}</div></div>;
      })}</div>
      <div className="lenses"><Lens label="YOUR LENS" side={held?.you} ends={['you lose', 'you win']} />
        <Lens label="THEIR LENS" side={held?.them} ends={['they lose', 'they win']} />
        <div className="verdict">{verdict ? <><span className={`vd ${verdict.className}`}>{verdict.tag}
        </span><span>{verdict.words}</span></> : <span>{waiting}</span>}</div></div>
      <div className="effect" title={delta.error || preview.error ? unavailable : undefined}>
        {preview.reading?.reasons.map(r => <div className="wide blocked" key={r}><span className="h">
          BLOCKED</span><small>{r}</small></div>)}
        {preview.reading?.teams.flatMap(t => t.figures.map(f =>
        <div key={`${t.team}:${f.label}`}><span className="h">{f.label.toUpperCase()}</span><b>{f.after}</b>
          <small>was {f.before}</small></div>))}
        {held && held.verdict !== 'empty' && <div><span className="h">YOUR BEST {n}</span>
          <b>{held.you.after.toFixed(1)}</b><small>was {held.you.before.toFixed(1)} · {signed(held.you.net)}
          </small></div>}
        {delta.reading && <div><span className="h">PROJ / WEEK</span>
          <b>{delta.reading.you.unprojected.length ? '—' :
            signed(delta.reading.you.perWeek)}</b><small>your best lineup</small></div>}
        <WarRoomDeltaSidePanel side={delta.reading?.you} name="YOUR" names={names} />
        <WarRoomDeltaSidePanel side={delta.reading?.them} name="THEIR" names={names} />
        {[...delta.reading?.notes ?? [], ...held?.notes ?? []].map(note => <div className="wide" key={note}>
          <small>{note}</small></div>)}
      </div><div className="sugg"><div className="ch"><b>TUNE IT</b>
        <span>smallest changes that keep it acceptable</span></div>
        {held?.tune.length ? held.tune.map(s => <div className="sg" key={`${s.kind}:${s.token}`}>
          <span className={`k ${s.kind === 'trim' ? 'trim' : ''}`}>{s.kind.toUpperCase()}</span>
          <span>{s.kind === 'trim' ? `Take ${names.get(s.token) ?? s.token} out` : s.kind === 'ask' ?
            `Also ask for ${names.get(s.token) ?? s.token}` : `Add ${names.get(s.token) ?? s.token}`}</span>
          <span className="r">you {signed(s.you)} · them {signed(s.them)}</span>
          <WarRoomAct className="btn" onAction={() => toggle(s.kind === 'ask' ? 'get' : 'give', s.token)}>
            APPLY</WarRoomAct></div>) :
          <div className="sg"><span /><span className="quiet">{!held ? waiting : held.verdict === 'empty' ?
            'Build a trade and the smallest fixes show here.' :
            'Nothing smaller keeps both sides whole.'}</span></div>}</div></div>
    <div className="shelf"><div className="read"><div className="partner"><select
      aria-label="Trade partner" value={deal.partner} onChange={e => change({ ...deal,
        partner: e.target.value, get: [] })}><option value="">Choose partner</option>
      {catalog?.teams.filter(t => t.id !== context.mine).map(t => <option key={t.id} value={t.id}>
        {t.name}</option>)}</select>{held?.them && <span className="dir" title={held.them.set ?
          'Their owner set this direction' :
          'Our read from their record and roster age. They have not set one.'}>
        {directionLabel[held.them.direction].toUpperCase()}</span>}</div>
      {deal.partner ? <><div className="kv"><span>Record · cap room</span><span>
        {partner.reading?.record ?? '—'} · {partner.reading?.room == null ? '—' : money(partner.reading.room)}
      </span></div>{(['hole', 'thin'] as const).map(state => <div className="kv" key={state}
        title={needWords[state]}>
        <span>{state === 'hole' ? 'Holes' : 'Thin'}</span><span>{held?.them ? held.them.needs.filter(x =>
          x.state === state).map(x => x.position).join(' · ') || 'none' : '—'}</span></div>)}
      <div className="kv"><span>Spare</span><span>{held?.them ? held.them.spare.join(' · ') || 'none' : '—'}
      </span></div><div className="lean">{held?.them ? <>{held.them.set ? '' : <b>Our read · </b>}
        {leanWords(held.them)}</> : waiting}</div></> :
        <div className="lean">Choose a partner to see how they would read a deal.</div>}
    </div><div className="ch"><b>THEIR SHELF</b><span>drag to GET</span></div><div className="list">
      {theirs?.assets.length ? ranked(theirs.assets).map(a => block(a, 'get')) :
        <p className="empty">{deal.partner ? absent(!catalog) : 'No partner chosen'}</p>}</div></div>
  </section>;
}
