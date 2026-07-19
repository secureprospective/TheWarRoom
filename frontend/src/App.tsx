import { useEffect, useState, type ReactNode } from 'react';
import { useHarnessStore } from './store/harness';
import { RookieTable } from './components/RookieTable';
import { ValidationBoard } from './components/ValidationBoard';
import { AdminPanel } from './components/AdminPanel';
import { RankingsBoard } from './components/RankingsBoard';
import { PowerRankingsBoard } from './components/PowerRankingsBoard';
import { TransactionsPanel } from './components/TransactionsPanel';
import { TransactionWorkspace } from './components/transactions/TransactionWorkspace';
import { TradeBuilder } from './components/transactions/TradeBuilder';
import { LeagueControls } from './components/transactions/LeagueControls';

type Tab =
  | 'rankings'
  | 'power'
  | 'rookies'
  | 'validation'
  | 'workspace'
  | 'trade'
  | 'league'
  | 'transactions';

// D8 phased shell cutover: the real M4 operator UI ("Transactions" workspace + "League Controls")
// ships alongside the OLD raw-mflID dev panel, which stays reachable behind this flag until all 14
// ops port into the workspace — then it is deleted. Flip to hide the dev tab.
const SHOW_DEV_PANEL = true;

// App shell: M1 (the real 32-team asset rankings, the first live module) plus the
// testing-harness tabs (rookie sandbox, architectural validation), with the live
// admin panel as a persistent right sidebar. Debuggability over polish — minimal
// chrome, every output on screen.
function App() {
  const loadAll = useHarnessStore((s) => s.loadAll);
  const loading = useHarnessStore((s) => s.loading);
  const [tab, setTab] = useState<Tab>('workspace');

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  return (
    <div className="min-h-screen bg-slate-900 text-slate-100">
      <header className="border-b border-slate-700 px-6 py-3">
        <h1 className="text-xl font-bold">The War Room — Testing Harness</h1>
        <p className="text-xs text-slate-400">
          Engine sandbox · identity Layer 4 · validates each B5b rubric as it lands
        </p>
      </header>

      <div className="flex">
        <main className="flex-1 p-6">
          <nav className="mb-4 flex gap-2">
            <TabButton active={tab === 'workspace'} onClick={() => setTab('workspace')}>
              Transactions
            </TabButton>
            <TabButton active={tab === 'trade'} onClick={() => setTab('trade')}>
              Trade
            </TabButton>
            <TabButton active={tab === 'league'} onClick={() => setTab('league')}>
              League Controls
            </TabButton>
            <TabButton active={tab === 'rankings'} onClick={() => setTab('rankings')}>
              M1: Asset Rankings
            </TabButton>
            <TabButton active={tab === 'power'} onClick={() => setTab('power')}>
              M2: Power Rankings
            </TabButton>
            <TabButton active={tab === 'rookies'} onClick={() => setTab('rookies')}>
              Sandbox: Rookie Rankings
            </TabButton>
            <TabButton active={tab === 'validation'} onClick={() => setTab('validation')}>
              Module 3: Architectural Tests
            </TabButton>
            {SHOW_DEV_PANEL && (
              <TabButton active={tab === 'transactions'} onClick={() => setTab('transactions')}>
                B7a: Transactions (dev)
              </TabButton>
            )}
            <button
              type="button"
              onClick={() => void loadAll()}
              className="ml-auto rounded bg-slate-700 px-3 py-1 text-sm hover:bg-slate-600"
            >
              {loading ? 'Loading…' : 'Reload'}
            </button>
          </nav>

          {tab === 'workspace' ? (
            <TransactionWorkspace />
          ) : tab === 'trade' ? (
            <TradeBuilder />
          ) : tab === 'league' ? (
            <LeagueControls />
          ) : tab === 'rankings' ? (
            <RankingsBoard />
          ) : tab === 'power' ? (
            <PowerRankingsBoard />
          ) : tab === 'rookies' ? (
            <RookieTable />
          ) : tab === 'validation' ? (
            <ValidationBoard />
          ) : (
            <TransactionsPanel />
          )}
        </main>

        <aside className="w-80 border-l border-slate-700 p-4">
          <AdminPanel />
        </aside>
      </div>
    </div>
  );
}

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded px-4 py-1.5 text-sm font-medium ${
        active ? 'bg-emerald-700 text-white' : 'bg-slate-800 text-slate-300 hover:bg-slate-700'
      }`}
    >
      {children}
    </button>
  );
}

export default App;
