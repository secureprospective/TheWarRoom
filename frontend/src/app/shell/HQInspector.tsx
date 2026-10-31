import { useState, type ReactNode } from 'react';
import type { Snapshot } from '../data/contract';
import { commands } from '../commands/registry';
import { readAgeArc } from '../market/api';
import { readPlayerUsage } from '../research/api';
import { readPersonalNotes, personalWire } from '../offers/api';
import { marketAPI } from '../market/api';
import { parseAssetBoard } from '../assets/contract';
import { contractsProvider } from '../contracts/provider';
import { useContractRead } from '../contracts/Contracts';
import { useHQRead, hqEvents } from './hqReads';
import { parseArc, parseUsage, parseNotes, researchFact, type Arc, type Usage, type Note } from './hqPlayerData';
import type { RosterFacts } from './hqRosterFacts';
import { money, unavailable } from './hqModel';

const arcRead = async (id: string) => parseArc(await readAgeArc(id));
const usageRead = async (id: string) => parseUsage(await readPlayerUsage(id));
const notesRead = async (mine: string, id: string) => parseNotes(await readPersonalNotes(mine, id));
const assetsRead = async () => parseAssetBoard(await marketAPI.assetBoard());
function Drawer({ title, detail, mine, children }: { title: string; detail?: ReactNode; mine?: boolean; children: ReactNode }) {
  const [open, setOpen] = useState(true);
  return <div className={`idr ${open ? 'open' : ''}`}><button aria-expanded={open} onClick={() => setOpen(!open)}>
    {title}{mine && <span className="mine">MINE</span>}<span className="c">{detail}</span></button>
    <div className="body" hidden={!open}>{children}</div></div>;
}
export function AgeArc({ arc, season }: { arc?: Arc; season: number }) {
  const max = Math.max(1, ...arc?.points.map(p => p.level) ?? []);
  return <>{arc?.points.length ? <div className="arc" title={arc.reason || 'Model age-value arc; past bars are backcasts'}>
    {arc.points.map(p => <i key={p.season} className={p.season === season ? 'now' : ''}
      style={{ height: `${Math.max(0, p.level / max * 100)}%` }} title={`${p.season} · age ${p.age.toFixed(1)} · ${p.level.toFixed(1)}`} />)}
  </div> : <p>{unavailable}</p>}
    <KV label="Peak window" value={arc ? `${arc.peakStart}–${arc.peakEnd}` : undefined} />
  </>;
}
export function KV({ label, value, title }: { label: string; value?: ReactNode; title?: string }) {
  return <div className="kv"><span>{label}</span><span title={title}>{value ?? unavailable}</span></div>;
}
export function UsageRows({ usage, projection, position }: { usage?: Usage; projection?: number; position?: string }) {
  const share = position === 'RB' ? usage?.carryShare : ['QB', 'DT', 'DE', 'LB', 'CB', 'S'].includes(position ?? '') ? usage?.snapShare : usage?.targetShare;
  return <><KV label="Projection" value={projection?.toFixed(1)} />
    <KV label={`${position === 'RB' ? 'Carry' : ['QB', 'DT', 'DE', 'LB', 'CB', 'S'].includes(position ?? '') ? 'Snap' : 'Target'} share, last 4`}
      value={share === undefined ? undefined : `${(share * 100).toFixed(0)}%`} title={usage ? `${usage.weeksUsed} weeks used · ${usage.reason}` : undefined} />
    <KV label={`Matchup vs ${position ?? '—'}`} value={usage?.label && usage.allowedRank !== undefined ? `${usage.label} · ${usage.allowedRank}` : undefined} title={usage?.reason} /></>;
}
export function NotesThread({ notes, count, pending, error, add, remove }: { notes?: Note[]; count?: number;
  pending?: boolean; error?: string; add?: (text: string) => Promise<void>; remove?: (note: Note) => Promise<void> }) {
  const [text, setText] = useState('');
  return <Drawer title="MY NOTES" mine detail={count ?? unavailable}>
    {notes ? notes.length ? <ol>{notes.map(note => <li key={note.at}><time dateTime={note.at}>{new Date(note.at).toLocaleDateString()}</time>
      <p>{note.text}</p><button className="btn" disabled={pending || !remove} onClick={() => { void remove?.(note); }}>REMOVE</button></li>)}</ol> : <p>No notes</p> : <p>{unavailable}</p>}
    <form onSubmit={e => { e.preventDefault(); if (add && text.trim()) void add(text).then(() => setText('')).catch(() => {}); }}>
      <textarea aria-label="Add personal note" value={text} maxLength={8000} disabled={!add || pending} onChange={e => setText(e.target.value)} />
      <button className="btn" type="submit" disabled={!add || pending || !text.trim()}>ADD NOTE</button>
    </form>{error && <p role="alert">{error}</p>}
  </Drawer>;
}
export function HQInspector({ snapshot, mine, id, facts, projection, native = false, bench }: {
  snapshot: Snapshot; mine: string; id?: string; facts?: RosterFacts; projection?: number; native?: boolean;
  bench?: (id: string) => void;
}) {
  return id ? <LoadedInspector key={`${native}:${mine}:${id}`} {...{ snapshot, mine, id, facts, projection, native, bench }} />
    : <aside className="insp" aria-label="Inspector"><div className="ihead"><div className="nm">{unavailable}</div></div><div className="nums"><div /><div /></div><div className="idrawers" /><div className="iact"><span className="h">ACT ON THIS PLAYER</span></div><div className="ifoot">{unavailable}</div></aside>;
}
function LoadedInspector({ snapshot, mine, id, facts, projection, native, bench }: {
  snapshot: Snapshot; mine: string; id: string; facts?: RosterFacts; projection?: number; native: boolean; bench?: (id: string) => void;
}) {
  const player = snapshot.players.value.find(p => p.id === id);
  const contract = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players.find(p => p.id === id);
  const arc = useHQRead('market.TargetAgeArc', [id], arcRead, !native);
  const usage = useHQRead('research.TargetPlayerUsage', [id], usageRead, !native);
  const [revision, setRevision] = useState(0);
  const readNotes = useContractRead(JSON.stringify([mine, id, revision]), () => native ? Promise.reject(new Error(unavailable)) : notesRead(mine, id), hqEvents);
  const assets = useHQRead('market.TargetAssetBoard', [], assetsRead, !native);
  const years = useContractRead(`${native}:${id}`, () => native ? Promise.reject(new Error(unavailable)) : contractsProvider().player(id), hqEvents);
  const [busy, setBusy] = useState(false); const [error, setError] = useState('');
  const change = async (action: 'add' | 'remove', text = '', at = '') => {
    if (busy || native) return;
    setBusy(true); setError('');
    try { await personalWire.change(mine, 'note', { id, entity: 'player', action, text, at }); setRevision(n => n + 1); }
    catch (e) { setError(String(e)); throw e; }
    finally { setBusy(false); }
  };
  const noteData = !native && readNotes.reading?.player === id ? readNotes.reading : undefined;
  const research = facts?.get(id)?.research;
  const ppg = researchFact(research, 'Points per game')?.split(' · ')[0];
  const dynasty = assets.value?.values.find(v => v.id === id)?.dynasty;
  const status = snapshot.eligibility.players[id]?.ir;
  const subject = { kind: 'player' as const, id, franchiseId: mine };
  return <aside className="insp" aria-label="Inspector">
    <div className="ihead"><div className="top"><span className="pg">{player?.position ?? '—'}</span><span className="on">{unavailable}</span></div>
      <div className="nm">{player?.name ?? id}</div><div className="sub">{player?.team ?? unavailable} · {money(contract?.salary)} · {contract?.contractStatus ?? unavailable}</div></div>
    <div className="nums"><div><b>{ppg ?? unavailable}</b><span>pts / game</span></div><div><b title={dynasty?.detail}>{dynasty?.held ? dynasty.text : unavailable}</b><span>dynasty value</span></div></div>
    <div className="idrawers">
      <Drawer title="THIS WEEK" detail={usage.value?.opponent || unavailable}><UsageRows usage={usage.value} projection={projection} position={player?.position} /></Drawer>
      <Drawer title="CONTRACT" detail={contract?.yearsRemaining === undefined ? unavailable : `${contract.yearsRemaining} yrs`}>
        {years.reading?.value?.years.map(y => <KV key={y.season} label={String(y.season)} value={money(y.cost.cents)} title={y.reason} />) ?? <p>{unavailable}</p>}
        <KV label="Dead cap if cut today" value={researchFact(research, 'Dead cap if cut today')} />
      </Drawer>
      <Drawer title="AGE ARC" mine detail={arc.value?.age.toFixed(1) ?? unavailable}><AgeArc arc={arc.value} season={snapshot.league.value.season} /></Drawer>
      <NotesThread notes={noteData?.notes} count={noteData?.count} pending={busy} error={error || (!native ? readNotes.error : '')}
        add={!native && noteData ? text => change('add', text) : undefined} remove={!native && noteData ? note => change('remove', '', note.at).catch(() => {}) : undefined} />
    </div>
    <div className="iact"><span className="h">ACT ON THIS PLAYER</span><div className="ia">
      <button className="btn" disabled={!bench} onClick={() => bench?.(id)}>BENCH</button>
      <button className="btn" disabled={native || status?.verdict !== 'eligible'} title={status?.reason || unavailable} onClick={() => { void commands.dispatch('roster.ir', { subject }); }}>TO IR</button>
      <button className="btn" disabled title={unavailable}>ADD TO TRADE</button>
      <button className="btn" disabled={native} title={native ? unavailable : undefined} onClick={() => { void commands.dispatch('watch.add', { id, entity: 'player', action: 'add' }); }}>WATCH</button>
    </div></div><div className="ifoot"><span>{native ? 'League ledger' : snapshot.rosters.provenance.source}</span><button disabled title={unavailable}>+ ADD DRAWER</button></div>
  </aside>;
}
