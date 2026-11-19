import { lazy, Suspense, useState, type ReactNode } from 'react';
import { RailNodes } from './RailNodes';
import { ThinPlayerSearch, THIN_LEAVES, type PlayerCandidate, type ThinScreenRoute }
  from './ThinPlayerSearch';
import { PhoneShell, type PhoneTab } from './PhoneShell';
import { HQAct } from './HQAct';
import './shell.css';
import './ThinLeagueShell.css';
import '../control/problemReport.css';

const ORDER = ['home', 'hq', 'war', 'pulse', 'control'] as const;
const ProblemReport = lazy(() => import('../control/ProblemReport')
  .then(module => ({ default: module.ProblemReport })));

export function ThinLeagueShell({ league, leagueName, route, commissioner, players, navigate, openPlayer,
  onHome, phoneShell = false, phoneTab = 'today', phoneLeagueName, onPhoneTab, onPhoneControlRoom,
  children }: {
    league?: string; leagueName: string; route: ThinScreenRoute; commissioner: boolean;
    players: readonly PlayerCandidate[]; navigate: (route: ThinScreenRoute) => void;
    openPlayer: (player: PlayerCandidate) => void; onHome?: () => void; phoneShell?: boolean;
    phoneTab?: PhoneTab; phoneLeagueName?: string; onPhoneTab?: (tab: PhoneTab) => void;
    onPhoneControlRoom?: () => void;
    children: ReactNode }) {
  const [reportOpen, setReportOpen] = useState(false);
  const order = commissioner ? ORDER : ORDER.filter(node => node !== 'control');
  const open = (node: ThinScreenRoute['node'], workspace?: string) => {
    if (node === 'home' && onHome) { onHome(); return; }
    navigate({ node, workspace: workspace ?? defaultWorkspace(node) });
  };
  return <div className="twr-app target-frame thin-league-shell" data-native-workspace="true">
    <aside className="rail mock">
      <div className="rail-head">
        <div className="lg">THE WAR ROOM</div>
        <div className="ph">{leagueName || 'Your league'} · hosted</div>
        <ThinPlayerSearch players={players} onOpen={openPlayer} />
      </div>
      <RailNodes node={route.node} workspace={route.workspace} order={order} leaves={THIN_LEAVES}
        open={open} />
      <div className="rail-foot">
        {league && !phoneShell && <HQAct className="problem-report-entry"
          onAction={() => setReportOpen(true)}>Report a problem</HQAct>}
        <span>THIN MODE</span>
      </div>
    </aside>
    {reportOpen && league && !phoneShell && <Suspense fallback={null}>
      <ProblemReport league={league} route={route} onClose={() => setReportOpen(false)} />
    </Suspense>}
    <main className="thin-league-main"><PhoneShell enabled={phoneShell}
      leagueName={phoneLeagueName ?? leagueName}
      activeTab={phoneTab} commissioner={commissioner} onSelect={tab => onPhoneTab?.(tab)}
      onControlRoom={() => onPhoneControlRoom?.()}>{children}</PhoneShell></main>
  </div>;
}

function defaultWorkspace(node: ThinScreenRoute['node']): string {
  switch (node) {
    case 'home': return 'overview';
    case 'hq': return 'lineup-and-roster';
    case 'war': return 'trade-desk';
    case 'pulse': return 'now';
    case 'control': return 'league';
  }
}
