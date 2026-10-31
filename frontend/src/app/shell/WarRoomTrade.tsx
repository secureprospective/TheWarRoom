import { HQAct as WarRoomAct } from './HQAct';
import { useState } from 'react';
import type { WarRoomContext } from './warroomReads';
import { useWarRoomRead } from './warroomReads';
import { readWarRoomDelta, readWarRoomPreview, readWarRoomPartner, type WarRoomAsset, type WarRoomCatalog,
  type WarRoomDeal, type WarRoomDeltaSide } from './warroomTradeReads';
import { WarRoomEmpty } from './WarRoomHeader';
import { money, unavailable } from './hqModel';
export function WarRoomAssetBlock({ asset, used, select, toggle, from, inTray }: { asset: WarRoomAsset;
  used?: boolean; inTray?: boolean; select: (a: WarRoomAsset) => void; toggle: () => void; from: string }) {
  return <div className={`blk ${asset.pick || asset.token.startsWith('pick:') ? 'pick' : ''} ${
    used ? 'sel' : ''} ${inTray ? 'trow' : ''}`}
    draggable onDragStart={e => e.dataTransfer.setData('application/warroom-asset', JSON.stringify({
      from, token: asset.token }))}>
    <WarRoomAct className="ps" label={`${used ? 'Remove' : 'Add'} ${asset.name}`} onAction={toggle}>
      {asset.position ?? (asset.pick ? 'PK' : '—')}</WarRoomAct>
    <WarRoomAct className="nm" onAction={() => select(asset)} title={asset.contract}>{asset.name}
      <small>{asset.contract ?? unavailable}</small>
      </WarRoomAct>
      <span className="ag">{asset.age ?? unavailable}</span>
    <span className="vl" title={asset.now?.detail}>{asset.now?.held ? asset.now.text : unavailable}</span>
    {inTray && <span className="v3">{unavailable}</span>}
  </div>;
}
export function WarRoomDeltaSidePanel({ side, name, names }: { side?: WarRoomDeltaSide; name: string;
  names: Map<string, string> }) {
  const player = (id: string) => names.get(id) ?? id;
  return <div className="wide"><span className="h">{name} · IN YOUR LINEUP</span>
    {side ? <><b>{side.unprojected.length ? unavailable : `${side.perWeek >= 0 ? '+' : ''}${
      side.perWeek.toFixed(1)}/wk`}</b><small>This week: {side.unprojected.length ? unavailable :
        side.thisWeek.toFixed(1)}</small>
      {side.changes.map(c => <div className="swap" key={`${c.kind}:${c.player}`}>
        <span className="sl">{c.position}</span><span>{c.kind === 'benched' ?
          `${name === 'YOU' ? 'Your' : 'Their'} best lineup would bench ${player(c.player)}` :
          `${player(c.player)} ${c.kind === 'enters' ? 'enters the best lineup' : c.kind === 'leaves' ?
            'leaves the best lineup' : 'is promoted to the best lineup'}`}</span></div>)}
      {side.unprojected.map(id => <p key={id}>{player(id)} · projection {unavailable}</p>)}
      {(['before', 'after'] as const).map(stage => <details key={stage}><summary>{stage === 'before' ?
        'Before starters' : 'After starters'}</summary>{side[stage].map(p => <div className="kv"
          key={p.player}><span>{p.position} · {player(p.player)}</span><span>{side.unprojected.includes(
            p.player) ? unavailable : p.ppg.toFixed(1)}</span></div>)}</details>)}
    </> : <WarRoomEmpty />}</div>;
}
export function WarRoomTrade({ context, catalog, deal, change, select }: { context: WarRoomContext;
  catalog?: WarRoomCatalog; deal: WarRoomDeal; change: (d: WarRoomDeal) => void;
  select: (a: WarRoomAsset) => void }) {
  const [filter, setFilter] = useState('ALL');
  const mine = catalog?.teams.find(t => t.id === context.mine);
  const theirs = catalog?.teams.find(t => t.id === deal.partner);
  const key = JSON.stringify([context.native?.league, context.mine, context.week, catalog?.revision, deal]);
  const partner = useWarRoomRead(JSON.stringify([context.native?.league, deal.partner, context.week]),
    () => readWarRoomPartner(context, deal.partner));
  const preview = useWarRoomRead(key, async () => catalog && deal.partner &&
    deal.give.length + deal.get.length ? readWarRoomPreview(context, catalog, deal) : undefined);
  const delta = useWarRoomRead(key, async () => catalog ? readWarRoomDelta(context, catalog, deal) :
    undefined);
  const toggle = (side: 'give' | 'get', token: string) => change({ ...deal, [side]: deal[side].includes(
    token) ? deal[side].filter(t => t !== token) : [...deal[side], token] });
  const block = (a: WarRoomAsset, side: 'give' | 'get', inTray = false) => <WarRoomAssetBlock
    key={a.token} asset={a}
    inTray={inTray} from={side === 'give' ? context.mine : deal.partner} select={select}
    used={deal[side].includes(a.token)} toggle={() => toggle(side, a.token)} />;
  const names = new Map(catalog?.teams.flatMap(t => t.assets.map(a => [a.token, a.name] as const)));
  return <section className="desk ws on" aria-label="Trade desk"><div className="shelf">
    <div className="ch"><b>MY SHELF</b><span>drag to SEND</span></div><div className="bench">
      <div className="filters">{['ALL', ...new Set(mine?.assets.map(a => a.position ?? (a.pick ? 'PICKS' :
        '—')))].map(p => <WarRoomAct key={p} className="chip" active={p === filter}
          onAction={() => setFilter(p)}>{p}</WarRoomAct>)}</div></div><div className="list">
      {mine?.assets.length ? mine.assets.filter(a => filter === 'ALL' || (a.position ?? (a.pick ? 'PICKS' :
        '—')) === filter).map(a => block(a, 'give')) : <WarRoomEmpty />}</div></div>
    <div className="build"><div className="vers" role="tablist"><WarRoomAct role="tab"
      selected> CURRENT DRAFT</WarRoomAct><WarRoomAct className="add" disabled title={unavailable}>+ VERSION
      </WarRoomAct><span className="t">unsent · only you see this</span></div>
      <div className="trays">{(['give', 'get'] as const).map(side => {
        const team = side === 'give' ? mine : theirs;
        return <div className="tray2" data-side={side === 'give' ? 'me' : 'them'} key={side}
          onDragOver={e => e.preventDefault()} onDrop={e => {
          e.preventDefault();
          try { const raw: unknown = JSON.parse(e.dataTransfer.getData('application/warroom-asset'));
            if (!raw || typeof raw !== 'object') return;
            const r = raw as { from?: unknown; token?: unknown };
            if (r.from === team?.id && typeof r.token === 'string' && team?.assets.some(a =>
              a.token === r.token) && !deal[side].includes(r.token)) toggle(side, r.token);
          } catch { /* An external drag is not a trade asset. */ }
        }}><div className="ch"><b>{side === 'give' ? 'YOU SEND' : 'YOU GET'}</b></div>
          <div className="tcols"><span /><span /><span>AGE</span><span className="y">YOU</span>
            <span className="t">THEM</span></div><div className="zone">{deal[side].length ? team?.assets
              .filter(a => deal[side].includes(a.token)).map(a => block(a, side, true)) :
              <div className="hint">Drag players or picks here</div>}</div>
          <div className="tot"><span>{side === 'give' ? 'total sent' : 'total received'}</span>
            <b>{unavailable}</b></div></div>;
      })}</div>
      <div className="lenses">{['YOUR LENS', 'THEIR LENS'].map(label => <div className="lens" key={label}>
        <span className="lb">{label}<small>{unavailable}</small></span><div className="scale" />
        <span className="n">{unavailable}</span></div>)}<div className="verdict" title={preview.error}>
        <span className="vd">{preview.reading ? preview.reading.reasons.join(' · ') || 'Preview checked' :
          unavailable}</span></div></div>
      <div className="effect" title={delta.error}>{preview.reading?.teams.flatMap(t => t.figures.map(f =>
        <div key={`${t.team}:${f.label}`}><span className="h">{t.team} · {f.label}</span><b>{f.after}</b>
          <small>{f.before} → {f.after} · {f.change}</small></div>))}
        <WarRoomDeltaSidePanel side={delta.reading?.you} name="YOU" names={names} />
        <WarRoomDeltaSidePanel side={delta.reading?.them} name="THEM" names={names} />
        {delta.reading?.notes.map(n => <div className="wide" key={n}><small>{n}</small></div>)}
      </div><div className="sugg"><div className="ch"><b>TUNE IT</b>
        <span>smallest changes that keep it acceptable</span></div><WarRoomEmpty /></div></div>
    <div className="shelf"><div className="read"><div className="partner"><select
      aria-label="Trade partner" value={deal.partner} onChange={e => change({ ...deal,
        partner: e.target.value, get: [] })}><option value="">Choose partner</option>
      {catalog?.teams.filter(t => t.id !== context.mine).map(t => <option key={t.id} value={t.id}>
        {t.name}</option>)}</select><span className="dir">{unavailable}</span></div>
      {['Record · cap room', 'Needs', 'Spare'].map(label => <div className="kv" key={label}>
        <span>{label}</span><span>{label === 'Record · cap room' ?
          `${partner.reading?.record ?? unavailable} · ${money(partner.reading?.room)}` : unavailable}
        </span></div>)}<div className="lean">{unavailable}</div>
    </div><div className="ch"><b>THEIR SHELF</b><span>drag to GET</span></div><div className="list">
      {theirs?.assets.length ? theirs.assets.map(a => block(a, 'get')) : <WarRoomEmpty />}</div></div>
  </section>;
}
