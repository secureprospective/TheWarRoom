import type { LeaguePlayerValue } from '../data/nativeValues';
import { Act } from '../commands/Act';
import type { NativeCut } from '../commissioner/hqReading';
import { HQAct } from './HQAct';
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
import { useHQRead, hqEvents, addHQTrade } from './hqReads';
import { parseArc, parseUsage, parseNotes, researchFact, type Arc, type Usage, type Note } from
  './hqPlayerData';
import type { RosterFacts } from './hqRosterFacts';
import { money, pickPlayer, unavailable } from './hqModel';

const arcRead = async (id: string) => parseArc(await readAgeArc(id));
const usageRead = async (id: string) => parseUsage(await readPlayerUsage(id));
const notesRead = async (mine: string, id: string) => parseNotes(await readPersonalNotes(mine, id));
const assetsRead = async () => parseAssetBoard(await marketAPI.assetBoard());
function Drawer({ title, detail, mine, children }: { title: string; detail?: ReactNode; mine?: boolean;
  children: ReactNode }) {
  const [open, setOpen] = useState(true);
  return <div className={`idr ${open ? 'open' : ''}`}><HQAct expanded={open} onAction={() => setOpen(!open)}>
    {title}{mine && <span className="mine">MINE</span>}<span className="c">{detail}</span></HQAct>
    <div className="body" hidden={!open}>{children}</div></div>;
}
export function AgeArc({ arc, season }: { arc?: Arc; season: number }) {
  const max = Math.max(1, ...arc?.points.map(p => p.level) ?? []);
  return <>{arc?.points.length ? <div className="arc" title={arc.reason ||
    'Model age-value arc; past bars are backcasts'}>
    {arc.points.map(p => <i key={p.season} className={p.season === season ? 'now' : ''}
      style={{ height: `${Math.max(0, p.level / max * 100)}%` }}
        title={`${p.season} · age ${p.age.toFixed(1)} · ${p.level.toFixed(1)}`} />)}
  </div> : <p>{unavailable}</p>}
    <KV label="Peak window" value={arc ? `${arc.peakStart}–${arc.peakEnd}` : undefined} />
  </>;
}
export function KV({ label, value, title }: { label: string; value?: ReactNode; title?: string }) {
  return <div className="kv"><span>{label}</span><span title={title}>{value ?? unavailable}</span></div>;
}
export function UsageRows({ usage, projection, position }: { usage?: Usage; projection?: number; position?:
  string }) {
  const share = position === 'RB' ? usage?.carryShare : ['QB', 'DT', 'DE', 'LB', 'CB',
    'S'].includes(position ?? '') ? usage?.snapShare : usage?.targetShare;
  return <><KV label="Projection" value={projection?.toFixed(1)} />
    <KV label={`${position === 'RB' ? 'Carry' : ['QB', 'DT', 'DE', 'LB', 'CB', 'S'].includes(position ??
      '') ? 'Snap' : 'Target'} share, last 4`}
      value={share === undefined ? undefined : `${(share * 100).toFixed(0)}%`} title={usage ?
        `${usage.weeksUsed} weeks used · ${usage.reason}` : undefined} />
    <KV label={`Matchup vs ${position ?? '—'}`} value={usage?.label && usage.allowedRank !== undefined ?
      `${usage.label} · ${usage.allowedRank}` : undefined} title={usage?.reason} /></>;
}
export function NotesThread({ notes, count, pending, error, add, remove }: { notes?: Note[]; count?: number;
  pending?: boolean; error?: string; add?: (text: string) => Promise<void>; remove?: (note: Note) =>
    Promise<void> }) {
  const [text, setText] = useState('');
  return <Drawer title="MY NOTES" mine detail={count ?? unavailable}>
    {notes ? notes.length ? <ol>{notes.map(note => <li key={note.at}><time dateTime={note.at}>{new
      Date(note.at).toLocaleDateString()}</time>
      <p>{note.text}</p><HQAct className="btn" disabled={pending || !remove} onAction={() => { void
        remove?.(note); }}>REMOVE</HQAct></li>)}</ol> : <p>No notes</p> : <p>{unavailable}</p>}
    <form onSubmit={e => { e.preventDefault(); if (add && text.trim()) void add(text).then(() =>
      setText('')).catch(() => {}); }}>
      <textarea aria-label="Add personal note" value={text} maxLength={8000} disabled={!add || pending}
        onChange={e => setText(e.target.value)} />
      <HQAct className="btn" submit disabled={!add || pending || !text.trim()}>ADD NOTE</HQAct>
    </form>{error && <p role="alert">{error}</p>}
  </Drawer>;
}
export function HQInspector({ snapshot, mine, id, facts, projection, native = false, bench, position, cut,
  toIR, annual, leagueValue, valuesPreparing }: {
  valuesPreparing?: boolean; leagueValue?: LeaguePlayerValue;
  snapshot: Snapshot; mine: string; id?: string; facts?: RosterFacts;
    projection?: number; native?: boolean;
  bench?: (id: string) => void; toIR?: (id: string) => void; position?: string; cut?: NativeCut; annual?: {
    season: number; cents: number }[];
}) {
  return id ? <LoadedInspector key={`${native}:${mine}:${id}`} {...{ snapshot, mine, id, facts, projection,
    native, bench, position, cut, toIR, annual, leagueValue, valuesPreparing }} />
    : <aside className="insp" aria-label="Inspector"><p className="empty"
      role="status">{pickPlayer}</p></aside>;
}
function LoadedInspector({ snapshot, mine, id, facts, projection, native, bench, position:
  selectedPosition, cut, toIR, annual, leagueValue, valuesPreparing
  }: {
  valuesPreparing?: boolean; leagueValue?: LeaguePlayerValue;
  snapshot: Snapshot; mine: string; id: string; facts?: RosterFacts;
    projection?: number; native: boolean;
    bench?: (id: string) => void; toIR?: (id: string) => void; position?: string; cut?: NativeCut; annual?:
      { season: number; cents: number }[];
}) {
  const teamReason = commands.useTeamActionReason('lineup.edit', { franchiseId: mine });
  const player = snapshot.players.value.find(p => p.id === id);
  const position = selectedPosition ?? player?.position;
  const contract = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players.find(p => p.id === id);
  const arc = useHQRead('market.TargetAgeArc', [id], arcRead);
  const usage = useHQRead('research.TargetPlayerUsage', [id], usageRead, !native);
  const [revision, setRevision] = useState(0);
  const readNotes = useContractRead(JSON.stringify([mine, id, revision]), () => native ? Promise.reject(new
    Error(unavailable)) : notesRead(mine, id), hqEvents);
  const assets = useHQRead('market.TargetAssetBoard', [], assetsRead, !native);
  const years = useContractRead(`${native}:${id}`, () => native ? Promise.reject(new Error(unavailable)) :
    contractsProvider().player(id), hqEvents);
  const [busy, setBusy] = useState(false); const [error, setError] = useState('');
  const change = async (action: 'add' | 'remove', text = '', at = '') => {
    if (busy || native) return;
    setBusy(true); setError('');
    try { await personalWire.change(mine, 'note', { id, entity: 'player', action, text, at });
      setRevision(n => n + 1); }
    catch (e) { setError(String(e)); throw e; }
    finally { setBusy(false); }
  };
  const noteData = !native && !readNotes.error && readNotes.reading?.player === id ? readNotes.reading :
    undefined;
  const research = facts?.get(id)?.research;
  const ppg = leagueValue ? (leagueValue.available ? leagueValue.nowPPG.toFixed(1) : undefined) :
    researchFact(research, 'Points per game')?.split(' · ')[0];
  const dynasty = leagueValue ? { held: leagueValue.available,
    text: leagueValue.dynastyPPG.toFixed(1), detail: leagueValue.words } :
    assets.value?.values.find(v => v.id === id)?.dynasty;
  const status = snapshot.eligibility.players[id]?.ir;
  const subject = { kind: 'player' as const, id, franchiseId: mine };
  return <aside className="insp" aria-label="Inspector">
    <div className="ihead"><div className="top"><span className="pg">{position ?? '—'}</span><span
      className="on">{unavailable}</span></div>
      <div className="nm">{player?.name ?? id}</div><div className="sub">{player?.team ?? unavailable}
        · {money(contract?.salary)} · {contract?.contractStatus ?? unavailable}</div></div>
    {valuesPreparing ? <p role="status">Values are being prepared</p> :
    <div className="nums"><div><b data-empty={!ppg}>{ppg ??
      unavailable}</b><span>pts / game</span></div><div><b data-empty={!dynasty?.held}
      title={dynasty?.detail}>{dynasty?.held ? dynasty.text :
      unavailable}</b><span>dynasty value</span></div></div>}
    <div className="idrawers">
      <Drawer title="THIS WEEK" detail={usage.value?.opponent || unavailable}><UsageRows
        usage={usage.value} projection={projection} position={position} /></Drawer>
      <Drawer title="CONTRACT" detail={contract?.yearsRemaining === undefined ? unavailable :
        `${contract.yearsRemaining} yrs`}>
        {native ? annual?.length ? annual.map(y => <KV key={y.season} label={String(y.season)}
          value={money(y.cents)} />) : <p>{unavailable}</p> : years.reading?.value?.years.map(y => <KV
            key={y.season} label={String(y.season)}
          value={money(y.cost.cents)} title={y.reason} />) ?? <p>{unavailable}</p>}
        <KV label="Dead cap if cut today" value={native ? money(cut?.deadNow) : researchFact(research,
          'Dead cap if cut today')} />
      </Drawer>
      <Drawer title="AGE ARC" mine detail={arc.value?.age.toFixed(1) ?? unavailable}><AgeArc
        arc={arc.value} season={snapshot.league.value.season} /></Drawer>
      <NotesThread notes={noteData?.notes} count={noteData?.count} pending={busy} error={error}
        add={!native && noteData ? text => change('add', text) : undefined} remove={!native && noteData ?
          note => change('remove', '', note.at).catch(() => {}) : undefined} />
    </div>
    <div className="iact"><span className="h">ACT ON THIS PLAYER</span><div className="ia">
      <HQAct className="btn" disabled={!bench} onAction={() => bench?.(id)}>BENCH</HQAct>
      {native ? <HQAct className="btn" disabled={!toIR} onAction={() => toIR?.(id)}>TO IR</HQAct> :
        <Act variant="button" className="btn" verb="roster.ir" args={{ subject }}
          disabled={status?.verdict !== 'eligible'} label={status?.reason}>TO IR</Act>}
      <HQAct className="btn" disabled={native || Boolean(teamReason)} title={teamReason}
        onAction={() => addHQTrade(mine, id)}>ADD TO TRADE</HQAct>
      <Act variant="button" className="btn" verb="watch.add"
        args={{ id, entity: 'player', action: 'add' }} disabled={native}>WATCH</Act>
    </div></div><div className="ifoot"><span>{native ? 'League ledger' :
      snapshot.rosters.provenance.source}</span><HQAct disabled title={unavailable}>+ ADD DRAWER</HQAct></div>
  </aside>;
}
