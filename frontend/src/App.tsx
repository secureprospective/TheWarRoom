import { useEffect, useState, useCallback } from 'react';
import { useHarnessStore } from './store/harness';
import { useAppInfoStore } from './store/appInfo';
import { AppShell } from './components/shell/AppShell';
import { useDensity } from './components/shell/useDensity';
import { MODULES, type ModuleId } from './components/shell/types';
import { RankingsBoard } from './components/RankingsBoard';
import { PowerRankingsBoard } from './components/PowerRankingsBoard';
import { TransactionWorkspace } from './components/transactions/TransactionWorkspace';
import { TradeBuilder } from './components/transactions/TradeBuilder';
import { LeagueControls } from './components/transactions/LeagueControls';
import { AdminPanel } from './components/AdminPanel';
import { RookieTable } from './components/RookieTable';
import { ValidationBoard } from './components/ValidationBoard';
import { CalendarBoard } from './components/calendar/CalendarBoard';
import { FeedBoard } from './components/feed/FeedBoard';
import { InspectorContent } from './components/inspector/InspectorContent';
import { HomeBoard } from './components/home/HomeBoard';
import { useInspectorStore } from './store/inspector';
import { isTypingTarget } from './components/board/keys';

// B-1 shell: the confirmed 4-column instrument console (Session A grid + Session C
// tokens) replaces the flat testing-harness tab bar. The shipped modules are
// re-homed into the workspace AS-IS this session — restyle to the Session-B
// component language is B-2. Modules self-fetch through the single Zustand
// gateway (store/harness.ts, WF5), so they mount here with no prop wiring.

const MODULE_TITLES: Record<ModuleId, string> = {
  home: 'HOME',
  assets: 'M1 · ASSET RANKINGS',
  pulse: 'M2 · POWER RANKINGS',
  txn: 'TRANSACT · ROSTER OPS',
  trade: 'TRADE · MULTI-LEG BUILDER',
  control: 'CONTROL · COMMISSIONER + DEV',
};

function App() {
  const loadAll = useHarnessStore((s) => s.loadAll);
  const loadAppInfo = useAppInfoStore((s) => s.load);
  const { density, setDensity } = useDensity();
  const [module, setModule] = useState<ModuleId>('assets');
  const [inspectorOpen, setInspectorOpen] = useState(false);
  // 'feed' is the Session-1 backward-looking Activity Feed (Session-D grammar applied to
  // historical ledger data). 'comms' is the still-future live thread; 'calendar' is the
  // commissioner-calendar board (B-3).
  const [summoned, setSummoned] = useState<'comms' | 'calendar' | 'feed' | null>(null);
  const selectedMflID = useInspectorStore((s) => s.selectedMflID);
  const openNonce = useInspectorStore((s) => s.openNonce);

  useEffect(() => {
    void loadAll();
    void loadAppInfo();
  }, [loadAll, loadAppInfo]);

  // Every player select (an M1 row click → the inspector store bumps openNonce) auto-opens the
  // inspector — including re-clicking the SAME row after a close (keying on selectedMflID alone would
  // no-op then). Closing keeps the selection so the 'i' toggle re-opens on the same player.
  useEffect(() => {
    if (openNonce > 0) setInspectorOpen(true);
  }, [openNonce]);

  // Global keyboard: density 1/2/3, inspector toggle (I), escape closes overlays.
  // Ignored while typing in an input/textarea (per the Session-B keyboard map).
  // Board-local J/K/Enter live in useBoardKeys and share the SAME typing guard — B-5
  // extracted isTypingTarget so the two handlers can never drift apart.
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (isTypingTarget(e.target)) return;
      if (e.key === '1') setDensity('narrative');
      else if (e.key === '2') setDensity('tactical');
      else if (e.key === '3') setDensity('matrix');
      else if (e.key === 'i' || e.key === 'I') setInspectorOpen((v) => !v);
      else if (e.key === 'Escape') {
        setSummoned(null);
        setInspectorOpen(false);
      }
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [setDensity]);

  const onSummon = useCallback((target: 'comms' | 'calendar' | 'feed') => {
    setSummoned((cur) => (cur === target ? null : target));
  }, []);

  return (
    <AppShell
      module={module}
      onModule={setModule}
      density={density}
      inspectorOpen={inspectorOpen}
      onInspectorClose={() => setInspectorOpen(false)}
      onSummon={onSummon}
      workspaceTitle={MODULE_TITLES[module]}
      inspector={selectedMflID ? <InspectorContent /> : undefined}
    >
      <ModuleView module={module} />
      {summoned === 'calendar' ? (
        <CalendarBoard onClose={() => setSummoned(null)} />
      ) : summoned === 'feed' ? (
        <FeedBoard onClose={() => setSummoned(null)} />
      ) : summoned === 'comms' ? (
        <SummonPlaceholder target="comms" onClose={() => setSummoned(null)} />
      ) : null}
    </AppShell>
  );
}

// Maps the 6 locked nav modules onto the surfaces that exist today. Restyle is
// B-2; wiring is B-1. Sandbox (rookie) + architectural tests live under CONTROL
// as dev-only surfaces — they are harness tools, not league-facing modules.
function ModuleView({ module }: { module: ModuleId }) {
  switch (module) {
    case 'assets':
      return <RankingsBoard />;
    case 'pulse':
      return <PowerRankingsBoard />;
    case 'txn':
      return <TransactionWorkspace />;
    case 'trade':
      return <TradeBuilder />;
    case 'control':
      return <ControlModule />;
    case 'home':
    default:
      return <HomeBoard />;
  }
}

type ControlTab = 'league' | 'admin' | 'sandbox' | 'tests';

function ControlModule() {
  const [tab, setTab] = useState<ControlTab>('league');
  const tabs: { id: ControlTab; label: string }[] = [
    { id: 'league', label: 'League Controls' },
    { id: 'admin', label: 'Engine Admin' },
    { id: 'sandbox', label: 'Rookie Sandbox (dev)' },
    { id: 'tests', label: 'Architectural Tests (dev)' },
  ];
  return (
    <div className="p-4">
      <div className="mb-4 flex gap-2 border-b border-hairline pb-2">
        {(tabs ?? []).map((t) => (
          <button
            key={t.id}
            type="button"
            onClick={() => setTab(t.id)}
            className={`rounded px-3 py-1 font-mono text-[11px] uppercase tracking-wide ${
              tab === t.id
                ? 'bg-surface-raised text-text-primary'
                : 'text-text-secondary hover:bg-surface-tile'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>
      {tab === 'league' ? (
        <LeagueControls />
      ) : tab === 'admin' ? (
        <AdminPanel />
      ) : tab === 'sandbox' ? (
        <RookieTable />
      ) : (
        <ValidationBoard />
      )}
    </div>
  );
}

// Placeholder for the summon-over COMMS quick-dash (Ledger A10/A11). The calendar summon is now the
// live CalendarBoard (B-3); comms stays a placeholder until its Session-D terminal-log grammar lands.
function SummonPlaceholder({ target, onClose }: { target: 'comms'; onClose: () => void }) {
  return (
    <div
      className="absolute inset-y-0 right-0 z-30 flex w-80 flex-col border-l border-hairline bg-surface-overlay shadow-bevel"
      style={{ transition: 'transform calc(150ms * var(--motion-mult)) ease-out' }}
    >
      <div className="flex items-center justify-between border-b border-hairline px-3 py-2">
        <span className="font-mono text-[11px] uppercase tracking-[0.06em] text-text-secondary">
          COMMS
        </span>
        <button
          type="button"
          aria-label={`Close ${target}`}
          onClick={onClose}
          className="font-mono text-text-tertiary hover:text-text-primary"
        >
          ✕
        </button>
      </div>
      <div className="flex flex-1 items-center justify-center p-4 text-center text-sm text-text-tertiary">
        Terminal-log comms thread lands in a later session (Session D grammar).
      </div>
    </div>
  );
}

export default App;
