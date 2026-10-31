import { useHQLive } from './hqLive';
import { useCachedLineup } from './lineups';
import { assertShortList } from './shortList';
import { useHQMovePlans, mergeHQMoves } from './hqMovePlans';
import { Act } from '../commands/Act';
import { HQAct } from './HQAct';
import { useEffect, useMemo, useRef, useState } from 'react';
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
const readHQNow = async (week: number, reportWeek: number, mine: string) => parseNow(await targetNow(week,
  reportWeek, mine));

export function FranchiseHQ({ snapshot }: { snapshot: Snapshot }) {
  const s = commands.use();
  const roster = useMemo(() => s.franchiseId ? franchiseRoster(snapshot, s.franchiseId) : undefined,
    [snapshot, s.franchiseId]);
  if (import.meta.env.DEV) {
    assertShortList(snapshot.franchises.value.length);
    if (roster) assertShortList(roster.groups.reduce((n, g) => n + g.players.length, 0));
  }
  if (!s.franchiseId || !roster) return <FranchisePicker snapshot={snapshot} />;
  return <HQFranchise key={s.franchiseId} snapshot={snapshot} mine={s.franchiseId} roster={roster} />;
}
function HQFranchise({ snapshot, mine, roster }: { snapshot: Snapshot; mine: string; roster:
  ReturnType<typeof franchiseRoster> }) {
  const s = commands.use();
  const live = useHQLive(mine);
  const heldLineup = useCachedLineup(commands.lineups, mine);
  const edit = commands.useLineupEdit();
  const teamReason = commands.useTeamActionReason('lineup.edit', { franchiseId: mine });
  const selectedId = s.subject?.franchiseId === mine ? s.subject.id : heldLineup.reading?.starters[0]?.id;
  const [boardVersion, setBoardVersion] = useState(0);
  const [releaseId, setRelease] = useState<string>();
  const [drawer, setDrawer] = useState<'taxi' | 'ir' | 'whatif'>();
  useEffect(() => { const key = (e: KeyboardEvent) => { if (e.key === 'Escape') { setDrawer(undefined);
    setRelease(undefined); } };
    document.addEventListener('keydown', key); return () => document.removeEventListener('keydown', key);
      }, []);
  const root = useRef<HTMLDivElement>(null);
  const clock = useLeagueClock(root);
  const week = clock?.value.week ?? heldLineup.reading?.week ?? 0;
  const settings = useHQRead('control.TargetLeagueSettings', [], readHQSettings);
  const provider = useMemo(contractsProvider, []);
  const contracts = useContractRead(mine, () => provider.contracts({ mine, team: mine, position: '', sort:
    'salary', desc: true }), provider);
  const race = useHQRead('race.TargetRace', [week, mine, mine, ''], readHQRace);
  const now = useHQRead('now.TargetNow', [week, 0, mine], readHQNow);
  const reports = useContractRead(mine, () => readRosterReports(mine), hqEvents);
  const injuries = useContractRead(mine, readHQInjuries, hqEvents);
  const moves = commands.useMoves(s => s.moves.filter(r => r.spec.franchiseId === mine));
  const lineupError = commands.useMoves(state => state.draftErrors[`lineup:${mine}`]);
  const plans = useHQMovePlans(mine, commands.providerKind !== 'fixture');
  const movesReady = commands.useMoves(s => s.movesLoading[mine] === false && !s.movesErrors[mine]);
  const assets = useHQRead('market.TargetAssetBoard', [], readHQAssets);
  const market = useHQRead('market.TargetMarketReports', [], readHQMarket);
  const ids = snapshot.rosters.value.find(r => r.franchiseId === mine)?.players.map(p => p.id) ?? [];
  const facts = useContractRead(JSON.stringify([mine, ids]), () => readRosterFacts(ids), hqEvents);
  const rules = settingsFromReport(settings.value);
  const franchise = snapshot.franchises.value.find(f => f.id === mine)!;
  const lock = clock?.value.deadlines.find(d => /lineup|lock/i.test(d.id + d.label));
  const tab = s.workspace.hq;
  const navigate = (workspace: 'lineup-and-roster' | 'contracts-and-cap' | 'my-moves') => { void
    commands.dispatch('nav.open', { node: 'hq', workspace }); };
  return <HQShell root={root} leagueName={settings.value?.summary.find(r => r.cells[0] === 'League name')?.cells[1]} liveCount={live?.games.filter(g => g.status === 'in').length} week={week} lock={lock} tab={tab as HQTab} navigate={navigate}
    navigateNode={node => { void commands.dispatch('nav.open', { node }); }}
    queue={<HQQueue available={movesReady && plans.reading?.errors.length === 0}
      moves={mergeHQMoves(moves.map(r => ({ id: r.correlationId, state: r.state,
      intent: r.spec.intent, players: r.spec.subject.players, note: r.audit.at(-1)?.note ?? '', at:
      r.audit.at(-1)?.at ?? '' })), plans.reading?.moves ?? [])} />}
    inspector={<HQInspector snapshot={snapshot} mine={mine} id={selectedId}
      facts={facts.reading} projection={projections(snapshot, market.value, week).get(selectedId ?? '')}
      bench={!teamReason && selectedId && heldLineup.reading?.starters.some(p => p.id === selectedId) ?
        id => {
        if (!edit.draft) commands.dispatch('lineup.edit', { franchiseId: mine });
        commands.dispatch('lineup.toggle', { playerId: id });
      } : undefined} />}>

      <LineupRoster snapshot={snapshot} roster={roster} franchiseId={mine} selected={s.subject} cap={null}
        render={editor => <>
          <HQHeader name={franchise.name ?? mine} active={roster.groups[0].players.length}
            settings={rules} cap={capFromReads(snapshot, mine, contracts.reading)}
            check={editor.checkError || editor.checkPending ? unavailable : lineupCheck(editor.reading,
              editor.error)}
            lock={lock && { ...lock, text: countdown(lock.at, Date.now()) }}
            {...standingFromReport(race.value?.standings)} />
          {tab === 'lineup-and-roster' ? <HQBoard key={boardVersion} snapshot={snapshot} mine={mine} reading={editor.error ?
            undefined : editor.reading}
            settings={rules} values={projections(snapshot, market.value, editor.reading?.week)}
              facts={facts.reading}
            injuries={injuries.reading} byes={byeWeeks(snapshot, reports.reading)} now={now.value} live={live} changed={editor.changes}
              canEdit={!editor.reason && !teamReason}
            select={id => { void commands.dispatch('inspector.open', { subject: { kind: 'player', id,
              franchiseId: mine } }); }}
            assign={ids => {
              if (!editor.draft) commands.dispatch('lineup.edit', { franchiseId: mine });
              const current = editor.reading?.starters.map(p => p.id) ?? [];
              for (const playerId of current.filter(id => !ids.includes(id)).concat(
                ids.filter(id => !current.includes(id))))
                commands.dispatch('lineup.toggle', { playerId });
            }}
            bench={id => { if (!editor.draft) commands.dispatch('lineup.edit', { franchiseId: mine });
              commands.dispatch('lineup.toggle', { playerId: id }); }} />
            : tab === 'contracts-and-cap' ? <HQContracts snapshot={snapshot} mine={mine}
              contracts={contracts.reading} assets={assets.value} facts={facts.reading} select={id => {
              setRelease(id); void commands.dispatch('inspector.open', { subject: { kind: 'player', id,
              franchiseId: mine } }); }} />
            : <MyMoves snapshot={snapshot} appearance="hq" />}
          <div className="tray"><div className="pulls" style={{ visibility: tab === 'lineup-and-roster' ?
            'visible' : 'hidden' }}><HQAct className="pull" expanded={drawer === 'taxi'} onAction={() =>
            setDrawer(drawer === 'taxi' ? undefined : 'taxi')}>TAXI <b>{roster.groups[2].players.length}
            / {rules.taxi ?? unavailable}</b></HQAct><HQAct className="pull" expanded={drawer === 'ir'}
            onAction={() => setDrawer(drawer === 'ir' ? undefined :
            'ir')}>IR <b>{roster.groups[1].players.length} / {rules.ir ?? unavailable}</b></HQAct><HQAct
            className="pull" expanded={drawer === 'whatif'} onAction={() => setDrawer(drawer === 'whatif' ?
            undefined : 'whatif')}>WHAT-IF</HQAct></div>
            <span className="msg" role="status">{editor.checkError || lineupError || s.notice ||
              moves.find(r => r.correlationId === edit.receiptId)?.audit.at(-1)?.note ||
              (editor.changes.length ?
              `${editor.changes.length} players changed from saved` :
              'No changes. Swaps stay a plan until you save.')}</span>
            <HQAct className="btn sp" disabled={!editor.draft} onAction={() => { void
              commands.dispatch('lineup.cancel', {}); setBoardVersion(n => n + 1); }}>DISCARD</HQAct>
            <Act variant="button" className="btn g2" verb="lineup.draft" args={{}}
              disabled={Boolean(editor.saveReason || teamReason)} label={teamReason ||
                editor.saveReason}>CHECK AND SAVE PLAN</Act>
            <Act variant="button" className="btn solid" verb="move.handoff"
              args={{ correlationId: edit.receiptId ?? '' }}
              disabled={!moves.find(r => r.correlationId === edit.receiptId && r.state === 'ready')}
              label="Open the saved lineup plan on MFL; nothing is submitted by the app">OPEN MFL ↗</Act>
          </div>
          {(drawer === 'ir' || drawer === 'taxi') && <HQReserves kind={drawer} snapshot={snapshot}
            mine={mine} settings={rules} assets={assets.value} injuries={injuries.reading} week={week}
              close={() =>
              setDrawer(undefined)} />}
          {drawer === 'whatif' && <HQWhatIf mine={mine} close={() => setDrawer(undefined)} />}
          {releaseId && <HQRelease key={releaseId} id={releaseId} mine={mine} snapshot={snapshot}
            facts={facts.reading} activeLimit={rules.active} close={() => setRelease(undefined)} />}
        </>} />
  </HQShell>;
}
