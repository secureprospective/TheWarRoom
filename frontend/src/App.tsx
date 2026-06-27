import { useEffect, useState, type ReactNode } from 'react';
import { useHarnessStore } from './store/harness';
import { RookieTable } from './components/RookieTable';
import { ValidationBoard } from './components/ValidationBoard';
import { AdminPanel } from './components/AdminPanel';

type Tab = 'rookies' | 'validation';

// App is the testing-harness shell: a tab between Module 1 (rookie rankings) and Module 3
// (architectural validation), with the live admin panel as a persistent right sidebar.
// Debuggability over polish — minimal chrome, every output on screen.
function App() {
  const loadAll = useHarnessStore((s) => s.loadAll);
  const loading = useHarnessStore((s) => s.loading);
  const [tab, setTab] = useState<Tab>('rookies');

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
            <TabButton active={tab === 'rookies'} onClick={() => setTab('rookies')}>
              Module 1: Rookie Rankings
            </TabButton>
            <TabButton active={tab === 'validation'} onClick={() => setTab('validation')}>
              Module 3: Architectural Tests
            </TabButton>
            <button
              type="button"
              onClick={() => void loadAll()}
              className="ml-auto rounded bg-slate-700 px-3 py-1 text-sm hover:bg-slate-600"
            >
              {loading ? 'Loading…' : 'Reload'}
            </button>
          </nav>

          {tab === 'rookies' ? <RookieTable /> : <ValidationBoard />}
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
