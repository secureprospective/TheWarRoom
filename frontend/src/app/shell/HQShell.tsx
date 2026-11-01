import { HQAct } from './HQAct';
import type { ReactNode, RefObject } from 'react';
import { commands } from '../commands/registry';
import { unavailable } from './hqModel';
import type { Node } from './nodes';
export function lockLabel(at: string): string {
  const date = new Date(at);
  const day = date.toLocaleDateString(undefined, { weekday: 'short' });
  const time = date.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', hour12: false });
  return `Locks ${day} ${time}`;
}
export type HQTab = 'lineup-and-roster' | 'contracts-and-cap' | 'my-moves';
export function HQShell({ root, week, lock, tab, navigate, navigateNode, children, inspector, queue,
  leagueName, fullscreen, liveCount }: {
  root?: RefObject<HTMLDivElement>; week?: number; lock?: { at?: string }; tab: HQTab;
  navigate: (tab: HQTab) => void; navigateNode: (node: Node) => void;
  children: ReactNode; inspector: ReactNode; queue?: ReactNode; leagueName?: string; fullscreen?: boolean;
    liveCount?: number;
}) {
  return <div className="hq-v10 shell" ref={root} style={fullscreen ? { position: 'fixed', inset: 0,
    zIndex: 10 } : undefined}>
    <nav className="rail" aria-label="Main">
      <div className="rail-head"><div className="lg">THE WAR ROOM</div><div className="ph">{leagueName ||
        unavailable} · <i>{week ? `Wk ${week}` : unavailable} · {liveCount === undefined ? unavailable :
          `${liveCount} live`}</i></div></div>
      <div className="rail-body"><div className="gl">MY SHORTCUTS</div>
        <HQAct className="leaf" onAction={() => navigate('lineup-and-roster')}>Lineup and roster</HQAct>
        <HQAct className="leaf" onAction={() => { void navigateNode('war'); }}>Free-agent pool</HQAct>
        <HQAct className="leaf" disabled title={unavailable}>Offers</HQAct>
        <div className="gl">LEAGUE</div>
        {(['Home', 'War Room', 'Franchise HQ', 'Trade Floor', 'League Pulse', 'Control Room'] as
          const).map((name, index) => <div className={`node ${index === 2 ? 'open' : ''}`} key={name}>
          <HQAct className="nb" onAction={() => { void navigateNode((['home', 'war', 'hq', 'trade',
            'pulse', 'control'] as const)[index]); }}>
            <span className="k">{index + 1}</span>{name}<span className="car">›</span></HQAct>
          {index === 2 && <><HQAct className="leaf now" disabled><span className="tag">NOW</span>{lock?.at
            ? lockLabel(lock.at) : unavailable}</HQAct>
            {(['lineup-and-roster', 'contracts-and-cap', 'my-moves'] as const).map((ws, i) => <HQAct
              key={ws} className={`leaf ${tab === ws ? 'sel' : ''}`} onAction={() => navigate(ws)}>
              <span className="tr">{i === 2 ? '└' : '├'}</span>{['Lineup and roster', 'Contracts and cap',
                'My moves'][i]}</HQAct>)}</>}
        </div>)}
      </div><div className="rail-foot"><HQAct className="btn" disabled
        title={unavailable}>CUSTOMIZE</HQAct><span>/ command</span></div>
    </nav>
    <main className="work">
      <div className="ws-tabs" role="tablist">
        <HQAct role="tab" selected={tab === 'lineup-and-roster'} onAction={() =>
          navigate('lineup-and-roster')}>LINEUP AND ROSTER</HQAct>
        <HQAct role="tab" selected={tab === 'contracts-and-cap'} onAction={() =>
          navigate('contracts-and-cap')}>CONTRACTS AND CAP</HQAct>
        <HQAct className={`queue ${tab === 'my-moves' ? 'on' : ''}`} onAction={() => navigate('my-moves')}
          label="My moves queue"><span
          className="ql">MY MOVES</span>{queue}</HQAct>
        <div className="cmd"><kbd>/</kbd> lineup.swap out=… in=…</div>
      </div>
      {children}
    </main>
    {inspector}
  </div>;
}
