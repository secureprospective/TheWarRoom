import { useEffect, useMemo, useState } from 'react';
import { FailureAlert } from '../data/FailureAlert';
import { createReadQuery } from '../data/query';
import { hostedReads, hostedSnapshot, onHostedChange } from '../data/hostedReads';
import { normalized, type ReadTags } from '../data/session';
import type { NativeRequest } from '../commands/native';
import { HQAct } from './HQAct';
import { money, stillReading, unavailable } from './hqModel';
import { HQWhatIfDrawer } from './HQWhatIfDrawer';
import {
  parseNativeWhatIfStack, type NativeWhatIfMove, type NativeWhatIfStack,
} from '../commissioner/hqReading';
import { hostedWorkflow, type HostedResponse } from '../commissioner/api';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { Snapshot } from '../data/contract';

type StackArgs = [league: string, franchise: string, moves: NativeWhatIfMove[], viewerZone: string];
type StackState = { reading?: NativeWhatIfStack; scope?: string };
const actions = [
  'cut', 'tag', 'extension', 'restructure', 'buyout', 'rfa_tender', 'resign', 'retire', 'death',
  'cap_relief',
];
const tags = (value: HostedResponse['tags']): ReadTags => {
  if (!value) throw new Error('The private quote freshness could not be read.');
  return value;
};

export function HQNativeWhatIf({ snapshot, league, mine, chosen, select, port = hostedWorkflow, close }: {
  snapshot: Snapshot; league: string; mine: string; chosen?: string; select: (id: string) => void;
  port?: HostedPort; close: () => void;
}) {
  const roster = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players ?? [];
  const [moves, setMoves] = useState<NativeWhatIfMove[]>([]);
  const [player, setPlayer] = useState(chosen ?? roster[0]?.id ?? '');
  const [action, setAction] = useState('cut');
  const [years, setYears] = useState(0);
  const [level, setLevel] = useState(0);
  const [moveCents, setMoveCents] = useState(0);
  const [amountCents, setAmountCents] = useState(0);
  const [reason, setReason] = useState('');
  const [identity, setIdentity] = useState(hostedSnapshot);
  useEffect(() => onHostedChange(() => setIdentity(hostedSnapshot())), []);
  useEffect(() => { if (chosen) setPlayer(chosen); }, [chosen]);
  const viewerZone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  const query = useMemo(() => createReadQuery<NativeWhatIfStack, StackState, StackArgs>('whatif.stack.read', {
    initial: () => ({}),
    hosted: { session: hostedReads, schemaVersion: '1', dependencies: [
      { kind: 'league' }, { kind: 'team', franchise: mine },
    ], policy: 'memory', public: false },
    read: async ([scopeLeague, _franchise, stack, zone]) => {
      const request: NativeRequest = { command: 'whatif.stack.read', mode: 'read', arguments: {
        league: scopeLeague, viewerZone: zone, moves: stack,
      } };
      const response = await port(request);
      return { data: parseNativeWhatIfStack(response.data, scopeLeague, stack), tags: tags(response.tags) };
    },
    accept: (reading, args) => ({ reading, scope: JSON.stringify(args) }),
    failure: () => 'The stacked quote could not be read. Refresh the league before trying again.',
  }), [league, mine, port]);
  const scopeArgs: StackArgs = [league, mine, moves, viewerZone];
  const scope = JSON.stringify(normalized(scopeArgs));
  useEffect(() => {
    if (!league || !mine) { query.reset(); return; }
    let key: string;
    try { key = hostedReads.key('whatif.stack.read', '1', scopeArgs); }
    catch { query.reset(); return; }
    hostedReads.invalidate(key);
    void query.load(scopeArgs, { overlap: 'join' });
    return () => query.fence();
    // The server scope and full ordered stack define this read; no move is saved or sent.
  }, [query, scope, identity.key]);
  useEffect(() => () => query.dispose(), [query]);
  const state = query.use();
  const reading = state.scope === scope ? state.reading : undefined;
  const add = () => {
    if (!roster.some(entry => entry.id === player)) return;
    setMoves(current => [...current, { player, action, years, level, moveCents, amountCents, reason }]);
  };
  const summary = reading?.after[0];
  const selectedName = (id: string) => snapshot.players.value.find(p => p.id === id)?.name ?? id;
  const input = <>
    <div className="wi-roster" aria-label="Players for what-if moves">
      {roster.map(entry => <HQAct key={entry.id} className={`wi ${player === entry.id ? 'on' : ''}`}
        label={`Select ${selectedName(entry.id)}`}
        onAction={() => { setPlayer(entry.id); select(entry.id); }}>
        <span className="box" /><span className="k">MOVE</span><span>{selectedName(entry.id)}</span>
      </HQAct>)}
    </div>
    <label>Move type <select aria-label="Move type" value={action}
      onChange={event => setAction(event.target.value)}>
      {actions.map(item => <option key={item} value={item}>
        {item.replaceAll('_', ' ').toUpperCase()}
      </option>)}
    </select></label>
    <label>Years <input aria-label="Years" type="number" min="0" max="99" value={years}
      onChange={event => setYears(Number(event.target.value))} /></label>
    <label>Level <input aria-label="Level" type="number" min="0" max="99" value={level}
      onChange={event => setLevel(Number(event.target.value))} /></label>
    <label>Move cents <input aria-label="Move cents" type="number" min="0" value={moveCents}
      onChange={event => setMoveCents(Number(event.target.value))} /></label>
    <label>Amount cents <input aria-label="Amount cents" type="number" min="0" value={amountCents}
      onChange={event => setAmountCents(Number(event.target.value))} /></label>
    <label>Reason <input aria-label="Reason" maxLength={200} value={reason}
      onChange={event => setReason(event.target.value)} /></label>
    <HQAct className="btn" disabled={!player || state.pending} onAction={add}>ADD MOVE</HQAct>
    <div className="wi-stack" aria-label="Stacked moves">
      {moves.map((move, index) => <div className="wi" key={`${move.player}:${index}`}>
        <span className="k">{move.action.toUpperCase()}</span><span>{selectedName(move.player)}</span>
        <HQAct className="btn" onAction={() => setMoves(current => current.filter((_, i) => i !== index))}
          >Remove {selectedName(move.player)} {move.action}</HQAct>
      </div>)}
      {moves.length === 0 && <p>No moves in this what-if stack.</p>}
      <HQAct className="btn" disabled={moves.length === 0 && !state.pending}
        onAction={() => setMoves([])}>RESET MOVES</HQAct>
    </div>
  </>;
  const preview = <>
    <div className="sum"><div><b>{money(summary?.roomCents)}</b><span>CAP ROOM AFTER</span></div>
      <div><b>{money(summary?.deadCents)}</b><span>DEAD MONEY AFTER</span></div>
      <div><b>{money(reading?.before[0]?.roomCents)}</b><span>CAP ROOM BEFORE</span></div>
      <div><b>{reading?.season ?? unavailable}</b><span>SEASON</span></div></div>
    <p className="note">Nothing is saved or sent. Every change is quoted by the league server.</p>
    {state.pending && <p role="status">{stillReading}</p>}
    {state.error && <FailureAlert
      message={state.error}
      code={state.errorCode}
      league={league}
      screen="Franchise HQ · What-If"
      route="hq/what-if"
      port={port}
    />}
    {reading && <div className="wi-runway" aria-label="Combined cap runway">
      <h3>Combined runway</h3><table><thead><tr><th>Season</th><th>Committed</th>
        <th>Dead</th><th>Room</th></tr></thead>
        <tbody>{reading.after.map(row => <tr key={row.season}><th>{row.season}</th>
          <td>{money(row.committedCents)}</td><td>{money(row.deadCents)}</td>
          <td>{money(row.roomCents)}</td></tr>)}</tbody>
      </table>
      {reading.moves.map((quote, index) => <p className="note" key={`${quote.player}:${index}`}>
        {quote.name} · {quote.actionName}: {quote.allowed ? 'Quoted' : quote.reason || unavailable}
      </p>)}
    </div>}
  </>;
  return <HQWhatIfDrawer close={close} detail="Stack moves · server quote · nothing is saved or sent."
    port={{ input, preview }} />;
}
