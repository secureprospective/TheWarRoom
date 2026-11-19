import type { ReactNode } from 'react';
import { RailNodes } from './RailNodes';
import { ThinPlayerSearch, THIN_LEAVES, type PlayerCandidate, type ThinScreenRoute }
  from './ThinPlayerSearch';
import './shell.css';
import './ThinLeagueShell.css';

const ORDER = ['home', 'hq', 'war', 'pulse', 'control'] as const;

export function ThinLeagueShell({ leagueName, route, commissioner, players, navigate, openPlayer,
  onHome, children }: {
    leagueName: string; route: ThinScreenRoute; commissioner: boolean;
    players: readonly PlayerCandidate[]; navigate: (route: ThinScreenRoute) => void;
    openPlayer: (player: PlayerCandidate) => void; onHome?: () => void; children: ReactNode }) {
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
      <div className="rail-foot"><span>THIN MODE</span></div>
    </aside>
    <main className="thin-league-main">{children}</main>
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
