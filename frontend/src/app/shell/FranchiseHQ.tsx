import { useEffect, useMemo, useRef, useState, useSyncExternalStore } from 'react';
import type { Snapshot } from '../data/contract';
import { commands } from '../commands/registry';
import { FranchisePicker } from './FranchisePicker';
import { franchiseRoster } from './roster';
import { LineupRoster } from './LineupRoster';
import { HQBoard } from './HQBoard';
import { readHQMarket, readRosterFacts } from './hqRosterFacts';
import { byeWeeks, projections, standingFromReport } from './hqPlayerData';
import { hqEvents } from './hqReads';
import { useLeagueClock } from '../clock/useLeagueClock';
import { countdown } from '../clock/urgency';
import { HQShell, type HQTab } from './HQShell';
import { HQRelease } from './HQRelease';
import { marketAPI } from '../market/api';
import { parseAssetBoard } from '../assets/contract';
import { HQInspector } from './HQInspector';
import { HQContracts, HQReserves, HQWhatIf } from './HQPanels';
import MyMoves from './MyMoves';
import { HQQueue } from './HQMoves';
import { readHQInjuries } from './hqRosterFacts';
import { HQHeader } from './HQHeader';
import { capFromReads, lineupCheck, settingsFromReport, unavailable } from './hqModel';
import { readHQRace, readHQSettings, useHQRead } from './hqReads';
import { contractsProvider } from '../contracts/provider';
import { useContractRead } from '../contracts/Contracts';
import { readRosterReports } from '../personal/RosterReports';
import { targetNow } from '../now/api';
import { parseNow } from '../now/contract';
import './hq.css';

const readHQAssets = async () => parseAssetBoard(await marketAPI.assetBoard());
const readHQNow = async (week: number, reportWeek: number, mine: string) => parseNow(await targetNow(week, reportWeek, mine));

export function FranchiseHQ({ snapshot }: { snapshot: Snapshot }) {
  const s = commands.use();
  if (!s.franchiseId) return <FranchisePicker snapshot={snapshot} />;
  return <HQFranchise key={s.franchiseId} snapshot={snapshot} mine={s.franchiseId} />;
}
function HQFranchise({ snapshot, mine }: { snapshot: Snapshot; mine: string }) {
  const s = commands.use();
  const heldLineup = useSyncExternalStore(commands.lineups.subscribe, () => commands.lineups.peek(mine));
  const selectedId = s.subject?.franchiseId === mine ? s.subject.id : heldLineup.reading?.starters[0]?.id;
  const [releaseId, setRelease] = useState<string>();
  const [drawer, setDrawer] = useState<'taxi' | 'ir' | 'whatif'>();
  useEffect(() => { const key = (e: KeyboardEvent) => { if (e.key === 'Escape') { setDrawer(undefined); setRelease(undefined); } };
    document.addEventListener('keydown', key); return () => document.removeEventListener('keydown', key); }, []);
  const root = useRef<HTMLDivElement>(null);
  const clock = useLeagueClock(root);
  const week = clock?.value.week ?? commands.lineups.peek(mine).reading?.week ?? 0;
  const settings = useHQRead('control.TargetLeagueSettings', [], readHQSettings);
  const provider = useMemo(contractsProvider, []);
  const contracts = useContractRead(mine, () => provider.contracts({ mine, team: mine, position: '', sort: 'salary', desc: true }), provider);
  const race = useHQRead('race.TargetRace', [week, mine, mine, ''], readHQRace);
  const now = useHQRead('now.TargetNow', [week, 0, mine], readHQNow);
  const reports = useContractRead(mine, () => readRosterReports(mine), hqEvents);
  const injuries = useContractRead(mine, readHQInjuries, hqEvents);
  const moves = commands.useMoves(s => s.moves.filter(r => r.spec.franchiseId === mine));
  const movesReady = commands.useMoves(s => s.movesLoading[mine] === false && !s.movesErrors[mine]);
  const assets = useHQRead('market.TargetAssetBoard', [], readHQAssets);
  const market = useHQRead('market.TargetMarketReports', [], readHQMarket);
  const ids = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players.map(p => p.id) ?? [];
  const facts = useContractRead(JSON.stringify([mine, ids]), () => readRosterFacts(ids), hqEvents);
  const rules = settingsFromReport(settings.value);
  const roster = franchiseRoster(snapshot, mine);
  const franchise = snapshot.franchises.value.find(f => f.id === mine)!;
  const lock = clock?.value.deadlines.find(d => /lineup|lock/i.test(d.id + d.label));
  const tab = s.workspace.hq;
  const navigate = (workspace: 'lineup-and-roster' | 'contracts-and-cap' | 'my-moves') => { void commands.dispatch('nav.open', { node: 'hq', workspace }); };
  return <HQShell root={root} week={week} lock={lock} tab={tab as HQTab} navigate={navigate}
    navigateNode={node => { void commands.dispatch('nav.open', { node }); }}
    queue={<HQQueue available={movesReady} moves={moves.map(r => ({ id: r.correlationId, state: r.state, intent: r.spec.intent, players: r.spec.subject.players, note: r.audit.at(-1)?.note ?? '', at: r.audit.at(-1)?.at ?? '' }))} />}
    inspector={<HQInspector snapshot={snapshot} mine={mine} id={selectedId}
      facts={facts.reading} projection={projections(snapshot, market.value, week).get(selectedId ?? '')}
      bench={selectedId && commands.lineups.peek(mine).reading?.starters.some(p => p.id === selectedId) ? id => {
        if (!commands.read().lineupDraft) commands.dispatch('lineup.edit', { franchiseId: mine });
        commands.dispatch('lineup.toggle', { playerId: id });
      } : undefined} />}>

      <LineupRoster snapshot={snapshot} roster={roster} franchiseId={mine} selected={s.subject} cap={null}
        render={editor => <>
          <HQHeader name={franchise.name ?? mine} active={roster.groups[0].players.length}
            settings={rules} cap={capFromReads(snapshot, mine, contracts.reading)}
            check={editor.checkError ? unavailable : lineupCheck(editor.reading, editor.error)}
            lock={lock && { ...lock, text: countdown(lock.at, Date.now()) }}
            {...standingFromReport(race.value?.standings)} />
          {tab === 'lineup-and-roster' ? <HQBoard snapshot={snapshot} mine={mine} reading={editor.error ? undefined : editor.reading}
            settings={rules} values={projections(snapshot, market.value, editor.reading?.week)} facts={facts.reading}
            byes={byeWeeks(snapshot, reports.reading)} now={now.value} changed={editor.changes} canEdit={!editor.reason}
            select={id => { void commands.dispatch('inspector.open', { subject: { kind: 'player', id, franchiseId: mine } }); }}
            place={(id, out) => { if (editor.reading?.starters.some(p => p.id === id)) return;
              if (!editor.draft) commands.dispatch('lineup.edit', { franchiseId: mine });
              if (out && out !== id) commands.dispatch('lineup.toggle', { playerId: out });
              if (out !== id) commands.dispatch('lineup.toggle', { playerId: id }); }}
            bench={id => { if (!editor.draft) commands.dispatch('lineup.edit', { franchiseId: mine });
              commands.dispatch('lineup.toggle', { playerId: id }); }} />
            : tab === 'contracts-and-cap' ? <HQContracts snapshot={snapshot} mine={mine} contracts={contracts.reading} assets={assets.value} facts={facts.reading} select={id => { setRelease(id); void commands.dispatch('inspector.open', { subject: { kind: 'player', id, franchiseId: mine } }); }} />
            : <MyMoves snapshot={snapshot} appearance="hq" />}
          <div className="tray"><div className="pulls" style={{ visibility: tab === 'lineup-and-roster' ? 'visible' : 'hidden' }}><button className="pull" aria-expanded={drawer === 'taxi'} onClick={() => setDrawer(drawer === 'taxi' ? undefined : 'taxi')}>TAXI <b>{roster.groups[2].players.length} / {rules.taxi ?? unavailable}</b></button><button className="pull" aria-expanded={drawer === 'ir'} onClick={() => setDrawer(drawer === 'ir' ? undefined : 'ir')}>IR <b>{roster.groups[1].players.length} / {rules.ir ?? unavailable}</b></button><button className="pull" aria-expanded={drawer === 'whatif'} onClick={() => setDrawer(drawer === 'whatif' ? undefined : 'whatif')}>WHAT-IF</button></div>
            <span className="msg" role="status">{editor.checkError || s.notice || (editor.changes.length ? `${editor.changes.length} players changed from saved` : 'No changes. Swaps stay a plan until you save.')}</span>
            <button className="btn sp" disabled={!editor.draft} onClick={() => { void commands.dispatch('lineup.cancel', {}); }}>DISCARD</button>
            <button className="btn g2" disabled={Boolean(editor.saveReason)} title={editor.saveReason} onClick={() => { void commands.dispatch('lineup.draft', {}); }}>CHECK AND SAVE PLAN</button>
            <button className="btn solid" disabled={!moves.find(r => r.correlationId === commands.read().lineupReceiptId && r.state === 'ready')}
              title="Open the saved lineup plan on MFL; nothing is submitted by the app" onClick={() => { const id = commands.read().lineupReceiptId; if (id) void commands.dispatch('move.handoff', { correlationId: id }); }}>OPEN MFL ↗</button>
          </div>
          {(drawer === 'ir' || drawer === 'taxi') && <HQReserves kind={drawer} snapshot={snapshot} mine={mine} settings={rules} injuries={injuries.reading} close={() => setDrawer(undefined)} />}
          {drawer === 'whatif' && <HQWhatIf mine={mine} close={() => setDrawer(undefined)} />}
          {releaseId && <HQRelease key={releaseId} id={releaseId} mine={mine} snapshot={snapshot} facts={facts.reading} activeLimit={rules.active} cap={capFromReads(snapshot, mine, contracts.reading)} close={() => setRelease(undefined)} />}
        </>} />
  </HQShell>;
}
