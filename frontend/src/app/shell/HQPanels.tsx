import { Act } from '../commands/Act';
import { HQAct } from './HQAct';
import { useMemo, useState } from 'react';
import type { AssetBoard } from '../assets/contract';
import type { Snapshot } from '../data/contract';
import type { HQCap, HQSettings } from './hqModel';
import { capFromReads, capacity, money, unavailable } from './hqModel';
import type { Injury } from './hqPlayerData';
import type { Readings } from '../contracts/contract';
import type { RosterFacts } from './hqRosterFacts';
import type { LeagueValues } from '../data/nativeValues';
import { researchFact } from './hqPlayerData';
import { commands } from '../commands/registry';
import { useContractRead } from '../contracts/Contracts';
import { whatIfProvider } from '../contracts/whatif';
import { contractsProvider } from '../contracts/provider';
import { hqEvents } from './hqReads';
import { HQWhatIfDrawer } from './HQWhatIfDrawer';
import type { ContractDecisionChoice, ContractDecisionOption, ContractDecisionPlayer,
  ContractDecisionSubmit, ContractDecisions } from './hqContractDecisions';

function decisionLabel(action: ContractDecisionOption['action']): string {
  switch (action) {
    case 'extension': return 'Extension';
    case 'tag': return 'Franchise tag';
    case 'rfa_tender': return 'RFA tender';
    case 'resign': return 'Re-sign';
  }
}
function decisionTerms(choice: ContractDecisionChoice): string {
  const term = [choice.years ? `${choice.years} ${choice.years === 1 ? 'year' : 'years'}` : '',
    choice.level ? `Level ${choice.level}` : ''].filter(Boolean).join(' · ');
  const schedule = choice.quote.salaries.map(year => `${year.season} ${money(year.cents)}`).join(' · ');
  return [term, schedule].filter(Boolean).join(' · ');
}
function decisionDate(value: string): string {
  return new Intl.DateTimeFormat(undefined, { year: 'numeric', month: 'short', day: 'numeric',
    hour: 'numeric', minute: '2-digit', timeZoneName: 'short' }).format(new Date(value));
}
export function ContractDecisionList({ reading, canDecide, decide }: { reading?: ContractDecisions;
  canDecide: boolean; decide?: (selection: ContractDecisionSubmit) => void }) {
  const [selected, setSelected] = useState<Record<string, string>>({});
  const [reasons, setReasons] = useState<Record<string, string>>({});
  if (!reading) return <p className="empty">{unavailable}</p>;
  if (!reading.players.length) return <p className="empty">{reading.words}</p>;
  return <div className="contract-decision-list">
    {reading.players.map(player => <ContractDecisionCard key={player.player} player={player}
      canDecide={canDecide} choose={(selection) => decide?.(selection)}
      selected={selected} onSelect={(key, value) => setSelected(current => ({ ...current, [key]: value }))}
      reason={reasons[player.player] ?? ''} onReason={value => setReasons(current => ({ ...current,
        [player.player]: value }))} />)}
  </div>;
}
function ContractDecisionCard({ player, canDecide, choose, selected, onSelect, reason, onReason }: {
  player: ContractDecisionPlayer; canDecide: boolean; choose: (selection: ContractDecisionSubmit) => void;
  selected: Record<string, string>; onSelect: (key: string, value: string) => void; reason: string;
  onReason: (value: string) => void }) {
  return <article className="contract-decision">
    <div className="decision-player"><b>{player.name}</b><span>Expires {player.expiration}</span>
      {player.decision && <span className="decision-saved">Recorded: {decisionLabel(player.decision.action)}
        {player.decision.years ? ` · ${player.decision.years}y` : ''}
        {player.decision.level ? ` · Level ${player.decision.level}` : ''}</span>}</div>
    {player.options.some(option => option.open && !option.pastDeadline) && <label className="decision-reason">
      Reason for this choice <input required aria-label={`Reason for ${player.name}'s contract choice`}
        value={reason} onChange={event => onReason(event.target.value)} maxLength={200} />
      <small>Required before preview.</small>
    </label>}
    {player.options.map(option => {
      const key = `${player.player}:${option.action}`;
      const defaultChoice = option.choices.find(choice => player.decision?.action === option.action &&
        choice.years === player.decision.years && choice.level === player.decision.level)
        ?? option.choices[0];
      const chosenKey = selected[key] ?? `${defaultChoice.years}:${defaultChoice.level}`;
      const choice = option.choices.find(candidate => `${candidate.years}:${candidate.level}` === chosenKey)
        ?? defaultChoice;
      const current = player.decision?.action === option.action && player.decision.years === choice.years &&
        player.decision.level === choice.level;
      const closed = !option.open || option.pastDeadline;
      return <div className="contract-decision-option" key={option.action} role="group"
        aria-label={`${decisionLabel(option.action)} for ${player.name}`}>
        <div className="decision-option-head"><b>{decisionLabel(option.action)}</b>
          <span className={closed ? 'clock cold' : 'clock'}>{option.pastDeadline ? 'PAST DEADLINE' :
            option.open ? 'OPEN' : 'NOT OPEN'}</span></div>
        <time dateTime={option.deadline}>Deadline · {decisionDate(option.deadline)}</time>
        <label className="decision-choice">Terms
          <select aria-label={`${player.name} ${decisionLabel(option.action)} terms`}
            value={chosenKey} disabled={closed || !canDecide} onChange={event =>
              onSelect(key, event.target.value)}>
            {option.choices.map(candidate => <option key={`${candidate.years}:${candidate.level}`}
              value={`${candidate.years}:${candidate.level}`}>{decisionTerms(candidate)}</option>)}
          </select>
        </label>
        {choice.quote.cap && <small title={choice.quote.cap.words}>{choice.quote.cap.words}</small>}
        <HQAct className="btn" label={`${current ? 'Current choice' : player.decision ? 'Change choice' :
          'Review choice'} · ${decisionLabel(option.action)} · ${player.name}`}
          disabled={closed || !canDecide || current || !reason.trim()}
          onAction={() => choose({ player: player.player, action: option.action, years: choice.years,
            level: choice.level, reason: reason.trim() })}>
          {current ? 'CURRENT CHOICE' : player.decision ? 'CHANGE CHOICE' : 'REVIEW CHOICE'}</HQAct>
      </div>;
    })}
  </article>;
}

export function CapRunway({ years }: { years?: HQCap[] }) {
  return <div className="run">{years?.length ? years.map(year => <div className={`y ${(year.room ?? 0) < 0
    ? 'tight' : ''}`} key={year.season}>
    <span>{year.season}</span><div className="bar"
      title={['Committed ' + money(year.committed), 'Dead ' + money(year.dead),
        'Room ' + money(year.room) + ' of ' + money(year.max)].join(' · ')}>
      {([['seg-c', year.committed], ['seg-d', year.dead]] as const).map(([css, value]) => <span key={css}
        className={css}
        style={{ width: year.max && value !== undefined ? `${Math.max(0, Math.min(100, value / year.max *
          100))}%` : '0%' }} />)}
    </div><b>{money(year.room)}</b></div>) : <p className="empty">{unavailable}</p>}</div>;
}
function ContractRow({ row, season, mine, years, facts, select, assets }: { assets?: AssetBoard; row: { id:
  string; name: string; position: string; salary: number | null; status: string };
  season: number; mine: string; years: number[]; facts?: RosterFacts; select: (id: string) => void }) {
  const reading = useContractRead(row.id, () => contractsProvider().player(row.id), hqEvents);
  const asset = assets?.values.find(v => v.id === row.id);
  const dynasty = asset?.dynasty.held ? asset.dynastyValue : undefined;
  const ratio = row.salary !== null && dynasty && dynasty > 0 ? row.salary / 1_000_000 / dynasty : undefined;
  return <tr>
    <td><span className="pg">{row.position}</span></td><td><HQAct className="contract-row-act" onAction={() =>
       select(row.id)}>{row.name}</HQAct></td>
    {years.map(year => { const value = reading.reading?.value?.years.find(y => y.season === year);
      return <td key={year} className={`yr ${value && reading.reading?.value?.years.at(-1)?.season === year
        ? 'last' : ''}`}
        title={value?.reason}>{value ? money(value.cost.cents) : year
        === season ? money(row.salary) : unavailable}
        {value && reading.reading?.value?.years.at(-1)?.season === year
          && ['RFA', 'UFA'].includes(row.status) && <i className={`cap-end ${row.status.toLowerCase()}`}
            title={row.status} />}</td>; })}
    <td>{researchFact(facts?.get(row.id)?.research, 'Dead cap if cut today') ?? unavailable}</td><td
      title={asset?.dynasty.detail}>{asset?.dynasty.held ? asset.dynasty.text :
      unavailable}</td><td className={ratio !== undefined && ratio < 2.2 ? 'ok' : ratio !== undefined &&
        ratio > 5.5 ? 'warn' : ''}>
      {ratio?.toFixed(1) ?? unavailable}</td>
  </tr>;
}
export function HQContracts({ snapshot, mine, contracts, runway, facts, select, native = false, assets,
  positions, cutCosts, annual, values, decisions, contractDecisions, decisionReady, decide }: {
  snapshot: Snapshot; mine: string; contracts?: Readings; runway?: HQCap[]; facts?: RosterFacts; select:
    (id: string) => void; native?: boolean; assets?: AssetBoard; positions?: Map<string, string>;
      cutCosts?: Map<string, number>; annual?: Map<string, { season: number; cents: number }[]>;
      values?: LeagueValues; decisions?: { label: string; at?: string; live: boolean }[];
  contractDecisions?: ContractDecisions; decisionReady?: boolean;
  decide?: (selection: ContractDecisionSubmit) => void;
}) {
  const season = snapshot.league.value.season;
  const roster = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players ?? [];
  const capYears: HQCap[] = runway ?? [capFromReads(snapshot, mine, contracts),
    ...contracts?.cap.value?.future.map(y => ({ season: y.season, committed: y.cost.cents })) ?? []];
  const years = runway?.length ? runway.map(y => y.season) : Array.from(new Set([season,
    ...contracts?.cap.value?.future.map(y => y.season) ?? []]));
  const rows = contracts?.table.value?.rows.map(r => ({ id: r.id, name: r.name, position: r.position,
    salary: r.salary.cents, status: r.status })) ?? roster.map(r => ({ id: r.id,
      name: snapshot.players.value.find(p => p.id === r.id)?.name ?? r.id,
      position: positions?.get(r.id) ?? snapshot.players.value.find(p => p.id === r.id)?.position ?? '—',
        salary: r.salary, status:
        r.contractStatus }));
  return <section className="ws capws on" id="ws-cap"><div className="ctable">
    <div className="ch"><b>CONTRACTS · {rows.length}</b><span>final year ends in<i className="key-cap rfa"
      />RFA · team can match<i className="key-cap ufa" />UFA · walks free</span></div>
    <div className="ct-wrap"><table className="ct"><thead><tr><th>POS</th><th>PLAYER</th>{years.map(y =>
      <th key={y}>{y}</th>)}<th>DEAD IF CUT</th><th>VALUE</th><th>$ / VALUE</th></tr></thead>
      <tbody>{rows.map(row => {
        if (!native) return <ContractRow key={row.id} {...{ row, season, mine, years, facts,
          select, assets }} />;
        const value = values?.players.find(v => v.player === row.id);
        const valued = value?.available ? value.dynastyPPG : undefined;
        const ratio = row.salary !== null && valued !== undefined && valued > 0
          ? row.salary / 100_000_000 / valued : undefined;
        return <tr key={row.id}><td><span className="pg">{row.position}</span></td>
          <td><HQAct className="contract-row-act" onAction={() => select(row.id)}>{row.name}</HQAct></td>
          {years.map(y => <td className="yr" key={y}>{money(annual?.get(row.id)?.find(v =>
            v.season === y)?.cents ?? (y === season ? row.salary : undefined))}</td>)}
          <td>{money(cutCosts?.get(row.id))}</td><td title={value?.words}>
            {valued?.toFixed(1) ?? unavailable}</td>
          <td className={ratio !== undefined && ratio < 2.2 ? 'ok' : ratio !== undefined && ratio > 5.5 ?
            'warn' : ''}>{ratio?.toFixed(1) ?? unavailable}</td></tr>;
      })}</tbody></table>
      {!rows.length && <p className="empty">{unavailable}</p>}</div>
  </div><div className="cside"><div
    className="ch"><b>CAP RUNWAY</b><span>committed · dead · room</span></div><CapRunway years={capYears} />
    {native && <><div className="ch"><b>DECISIONS DUE</b><span>priced by league rules · season {
      contractDecisions?.season ?? season}</span></div><ContractDecisionList reading={contractDecisions}
        canDecide={decisionReady ?? false} decide={decide} /></>}
    <div className="ch"><b>DEADLINES</b>
      <span>from the league calendar</span></div>
    <div className="dec">{decisions?.length ? decisions.map((decision, i) =>
      <p className="empty" key={`${decision.label}-${i}`} title={decision.at}>{decision.label}
        {decision.live ? ' · open' : ''}
        {decision.at ? ` · ${new Date(decision.at).toLocaleDateString()}` : ''}</p>) :
      <p className="empty">{unavailable}</p>}</div>
    <div className="ch" style={{ borderTop: '2px solid var(--black)'
      }}><b>DEAD CAP TOTAL · {money(capYears[0]?.dead)}</b><span>dead money included in runway</span></div>
  </div></section>;
}
export function HQReserves({ kind, snapshot, mine, settings, injuries, close, native = false, week,
  countedIRPay, assets, values, reserveNotes, reserveExperience, activate }: {
  kind: 'ir' | 'taxi'; snapshot: Snapshot; mine: string; settings: HQSettings; injuries?: Injury[]; close:
    () => void; native?: boolean; week?: number; countedIRPay?: number; assets?: AssetBoard;
      values?: LeagueValues; reserveNotes?: Map<string, string>;
      reserveExperience?: Map<string, { experienceYears?: number; taxiYearsLeft?: number }>;
      activate?: (id: string) => void;
}) {
  const roster = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players ?? [];
  const players = roster.filter(p => p.rosterStatus === (kind === 'ir' ? 'IR' : 'TAXI_SQUAD'));
  const active = roster.filter(p => p.rosterStatus === 'ROSTER');
  const limit = kind === 'ir' ? settings.ir : settings.taxi;
  const knownEligibility = active.every(p => snapshot.eligibility.players[p.id]
    && snapshot.eligibility.players[p.id][kind].verdict !== 'unknown');
  const candidates = active.filter(p => snapshot.eligibility.players[p.id]?.[kind].verdict === 'eligible');
  const returns = players.flatMap(p => {
    const injury = injuries?.find(i => i.id === p.id);
    return injury?.returnWeek && injury.returnSeason === snapshot.league.value.season ? [{ ...injury, name:
      snapshot.players.value.find(v => v.id === p.id)?.name ?? p.id }] : [];
  }).sort((a, b) => a.returnWeek! - b.returnWeek!);
  const hasTaxiExperience = native && kind === 'taxi' && players.some(p => {
    const held = reserveExperience?.get(p.id);
    return held?.experienceYears !== undefined || held?.taxiYearsLeft !== undefined;
  });

  return <section className="bdrawer open" aria-label="Roster drawer"><div className="bd-head"><h2>{kind
    === 'ir' ? 'Injured reserve' : 'Taxi squad'}</h2>
    <span className="sub">{capacity(players.length, limit)}</span><HQAct className="x"
      onAction={close}>ESC</HQAct></div>
    <div className="bd-body"><div className="bd-main"><table className="ct bt"><thead><tr>{(kind === 'ir'
      ? ['POS', 'PLAYER', 'INJURY', 'BACK', 'IN', 'SALARY', 'NOTE', ''] : ['POS', 'PLAYER',
        ...(hasTaxiExperience ? ['NFL YR', 'TAXI LEFT'] : []), 'SALARY', 'VALUE', 'NOTE', ''])
        .map((label, i) => <th key={i}>{label}</th>)}</tr></thead>
      <tbody>{players.map(p => { const player = snapshot.players.value.find(v => v.id === p.id); const
        injury = injuries?.find(i => i.id === p.id);
        const hostedValue = values?.players.find(v => v.player === p.id);
        const value = native ? { held: !!hostedValue?.available, text: hostedValue?.dynastyPPG.toFixed(1) } :
          assets?.values.find(v => v.id === p.id)?.dynasty;
        return <tr key={p.id}><td><span className="pg">{player?.position ??
          '—'}</span></td><td>{player?.name ?? p.id}<small>{player?.team}</small></td>
          {kind === 'ir' ? <><td>{injury?.details || unavailable}</td><td
            title={injury?.returnReason}>{injury?.returnWeek ?
            `${injury.returnSeason} · Wk ${injury.returnWeek}` :
            unavailable}</td><td>{injury?.returnWeek && injury.returnSeason === snapshot.league.value.season
            && week ? `${Math.max(0, injury.returnWeek - week)}w` :
              unavailable}</td><td>{money(p.salary)}</td><td>{injury?.returnReason ||
                reserveNotes?.get(p.id) ||
            unavailable}</td></>
            : <>{hasTaxiExperience && <><td>{reserveExperience?.get(p.id)?.experienceYears ??
                unavailable}</td><td>{reserveExperience?.get(p.id)?.taxiYearsLeft ?? unavailable}</td></>}
              <td>{money(p.salary)}</td><td>{value?.held ? value.text : unavailable}</td>
              <td>{reserveNotes?.get(p.id) ??
                snapshot.eligibility.players[p.id]?.taxi.reason ?? unavailable}</td></>}
          <td>{native ? <HQAct className="btn" disabled={!activate}
            onAction={() => activate?.(p.id)}>{kind === 'ir' ? 'ACTIVATE' : 'PROMOTE'}</HQAct> :
            <Act variant="button" className="btn" verb={kind === 'ir' ? 'roster.ir' : 'roster.taxi'}
            args={{ subject: { kind: 'player', id: p.id, franchiseId: mine } }}
            disabled={snapshot.eligibility.players[p.id]?.[kind].verdict === 'ineligible'}>{kind
              === 'ir' ? 'ACTIVATE' : 'PROMOTE'}</Act>}</td></tr>;
      })}</tbody></table>{!players.length && <p className="empty">{unavailable}</p>}
      {kind === 'ir' && <><div className="ch" style={{ borderTop: '1px solid var(--edge)'
        }}><b>RETURNS</b><span>plan the active spot before they come back</span></div>
        {returns.length ? <div className="tl" style={{ gridTemplateColumns: `repeat(${returns.length},1fr)`
          }}>{returns.map(r => <div key={r.id}>WK {r.returnWeek}<b>{r.name}</b></div>)}</div> : <p
          className="empty">{unavailable}</p>}</>}
    </div><div className="bd-side"><div className="sum"><div><b>{capacity(players.length,
      limit)}</b><span>{kind.toUpperCase()} SLOTS</span></div><div><b>{capacity(active.length,
      settings.active)}</b><span>ACTIVE</span></div>
      <div><b>{kind === 'taxi' ? money(players.some(p => p.salary === null) ? null : players.reduce((n, p) =>
        n + (p.salary ?? 0), 0)) :
        money(countedIRPay)}</b><span>{kind === 'ir' ? 'IR PAY · ON CAP' : 'TAXI PAY'}</span></div>
      <div><b>{kind === 'ir' ? returns[0] ? `Wk ${returns[0].returnWeek}` : unavailable : limit ===
        undefined ? unavailable : limit - players.length}</b><span>{kind === 'ir' ? 'NEXT RETURN' :
        'OPEN TAXI SLOT'}</span></div></div>
      <div className="ch"><b>{kind === 'ir' ? 'IR CANDIDATES' : 'COULD STASH'}
        · {knownEligibility ? candidates.length : unavailable}</b><span>{kind === 'ir' ? 'on your roster' :
        'active, still eligible'}</span></div>
      {candidates.map(p => <div className="drow" key={p.id}><span
        className="pg">{snapshot.players.value.find(v => v.id === p.id)?.position ??
        '—'}</span><span>{snapshot.players.value.find(v => v.id === p.id)?.name ?? p.id}</span>
        <Act variant="button" className="btn" verb={kind === 'ir' ? 'roster.ir' : 'roster.taxi'}
          args={{ subject: { kind: 'player', id: p.id, franchiseId: mine } }} disabled={native}>
          TO {kind.toUpperCase()}</Act><span
          className="why">{snapshot.eligibility.players[p.id]?.[kind].reason}</span></div>)}
    </div></div></section>;
}
export function HQWhatIf({ mine, close }: { mine: string; close: () => void }) {
  const provider = useMemo(whatIfProvider, []);
  const { reading, error } = useContractRead(mine, () => provider.query(mine), provider);
  const value = reading?.value;
  const input = value?.choices.map(choice => {
    const move = value.moves.find(m => m.player === choice.player);
    return <HQAct key={choice.player} className={`wi ${move ? 'on' : ''}`} disabled={value.stale}
      onAction={() => void commands.dispatch('contracts.set', { field: 'whatif', value: move
        ? { mine, action: 'undo', move: move.id } : { mine, action: 'cut', player: choice.player } })}>
      <span className="box" /><span className="k">CUT</span><span>{choice.cutLabel}</span>
      <span className="d">{move?.charge.text ?? ''}</span></HQAct>;
  });
  const preview = <><div className="sum">{value?.figures.map(figure => <div key={figure.label}>
    <b>{figure.after}</b><span>{figure.label}</span>
    <span className="delta">{`BEFORE ${figure.before} ┆ AFTER ${figure.after}`}</span></div>)}</div>
    <div className="tray" style={{ background: 'none', marginTop: 'auto' }}>
      <span className="msg">{value?.notice || `${value?.moves.length ?? 0} stacked`}</span>
      <HQAct className="btn sp" disabled={!value?.moves.length}
        onAction={() => void commands.dispatch('contracts.set', {
          field: 'whatif', value: { mine, action: 'reset' } })}>CLEAR</HQAct>
      <HQAct className="btn g2" disabled={!value?.moves.length || value.stale}
        onAction={() => value?.moves.filter(m => !m.plan).forEach(move => { void commands.dispatch(
          'contracts.set', { field: 'release', value: { mine, action: 'draft', move: move.id } });
        })}>SEND TO MY MOVES</HQAct>
    </div></>;
  return <HQWhatIfDrawer close={close} detail="Tick moves to stack them. Nothing is saved or sent."
    port={{ input: <>{input}{(!value || !value.choices.length) &&
      <p className="empty" title={error}>{unavailable}</p>}</>, preview }} />;
}
