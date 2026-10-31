import type { ReactNode, RefObject } from 'react';
import { commands } from '../commands/registry';
import { unavailable } from './hqModel';
import type { Node } from './nodes';
export type HQTab = 'lineup-and-roster' | 'contracts-and-cap' | 'my-moves';
export function HQShell({ root, week, lock, tab, navigate, navigateNode, children, inspector, queue, leagueName, fullscreen }: {
  root?: RefObject<HTMLDivElement>; week?: number; lock?: { at?: string }; tab: HQTab;
  navigate: (tab: HQTab) => void; navigateNode: (node: Node) => void;
  children: ReactNode; inspector: ReactNode; queue?: ReactNode; leagueName?: string; fullscreen?: boolean;
}) {
  return <div className="hq-v10 shell" ref={root} style={fullscreen ? { position: 'fixed', inset: 0, zIndex: 10 } : undefined}>
    <nav className="rail" aria-label="Main">
      <div className="rail-head"><div className="lg">THE WAR ROOM</div><div className="ph">{leagueName || unavailable} · <i>{week ? `Wk ${week}` : unavailable}</i></div></div>
      <div className="rail-body"><div className="gl">MY SHORTCUTS</div>
        <button className="leaf" onClick={() => navigate('lineup-and-roster')}>Lineup and roster</button>
        <button className="leaf" onClick={() => { void navigateNode('war'); }}>Free-agent pool</button>
        <button className="leaf" disabled title={unavailable}>Offers</button>
        <div className="gl">LEAGUE</div>
        {(['Home', 'War Room', 'Franchise HQ', 'Trade Floor', 'League Pulse', 'Control Room'] as const).map((name, index) => <div className={`node ${index === 2 ? 'open' : ''}`} key={name}>
          <button className="nb" onClick={() => { void navigateNode((['home', 'war', 'hq', 'trade', 'pulse', 'control'] as const)[index]); }}>
            <span className="k">{index + 1}</span>{name}<span className="car">›</span></button>
          {index === 2 && <><button className="leaf now" disabled><span className="tag">NOW</span>{lock?.at ? `Locks ${new Date(lock.at).toLocaleString()}` : unavailable}</button>
            {(['lineup-and-roster', 'contracts-and-cap', 'my-moves'] as const).map((ws, i) => <button key={ws} className={`leaf ${tab === ws ? 'sel' : ''}`} onClick={() => navigate(ws)}>
              <span className="tr">{i === 2 ? '└' : '├'}</span>{['Lineup and roster', 'Contracts and cap', 'My moves'][i]}</button>)}</>}
        </div>)}
      </div><div className="rail-foot"><button className="btn" disabled title={unavailable}>CUSTOMIZE</button><span>/ command</span></div>
    </nav>
    <main className="work">
      <div className="ws-tabs" role="tablist">
        <button role="tab" aria-selected={tab === 'lineup-and-roster'} onClick={() => navigate('lineup-and-roster')}>LINEUP AND ROSTER</button>
        <button role="tab" aria-selected={tab === 'contracts-and-cap'} onClick={() => navigate('contracts-and-cap')}>CONTRACTS AND CAP</button>
        <button className="queue" onClick={() => navigate('my-moves')} aria-label="My moves queue"><span className="ql">MY MOVES</span>{queue}</button>
        <button className="cmd" onClick={() => { void commands.dispatch('commandbar.open', {}); }}><kbd>/</kbd> lineup.swap out=… in=…</button>
      </div>
      {children}
    </main>
    {inspector}
  </div>;
}
