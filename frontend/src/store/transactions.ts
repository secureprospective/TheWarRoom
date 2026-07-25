import { create } from 'zustand';
import {
  GetFranchises,
  GetRoster,
  GetFreeAgentPool,
  GetCurrentPhase,
  GetLegalOps,
} from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

// Transactions store — Session 44 (SLIM_MAP §6.2). Owns the M4 READ-MODEL that the
// four transaction surfaces (TransactionWorkspace, TradeBuilder, LeagueControls,
// ConfirmModal is action-only) all pull from: franchises, phase, legalOps,
// rosters/FA pool. Per WF5 (harness.ts) the IPC call lives here, never in a
// component. ACTIONS (PreviewTransaction/ExecuteTransaction) and ephemeral per-op
// UI state (cart, form inputs, modal/stageGen) stay LOCAL to each component —
// Christopher explicitly rejected a "full store migration" (handoff 44).
//
// Only one of TransactionWorkspace/TradeBuilder/LeagueControls is ever mounted at
// a time (App.tsx's ModuleView/ControlModule tab switches are mutually exclusive),
// so a single shared `roster`/`pool` slot is safe — no two surfaces browse a
// roster concurrently.
//
// legalOpsPhase is kept separate from `phase`: TransactionWorkspace's phase comes
// from GetCurrentPhase, TradeBuilder's comes from GetLegalOps' phase field — two
// different pre-existing call sites, preserved as-is (mechanical move, no behavior
// change).
interface TransactionsState {
  franchises: main.M4Franchise[];
  phase: string;
  legalOps: string[];
  legalOpsPhase: string;
  roster: main.RosterResult | null;
  pool: main.FreeAgentPoolResult | null;
  loadFranchises: () => Promise<void>;
  loadPhase: () => Promise<void>;
  loadLegalOps: () => Promise<void>;
  loadRoster: (franchiseID: string) => Promise<void>;
  clearRoster: () => void;
  loadPool: () => Promise<void>;
  clearPool: () => void;
}

export const useTransactionsStore = create<TransactionsState>((set) => ({
  franchises: [],
  phase: '…',
  legalOps: [],
  legalOpsPhase: '…',
  roster: null,
  pool: null,

  loadFranchises: async () => {
    const r = await GetFranchises();
    set({ franchises: r.ok ? (r.franchises ?? []) : [] });
  },

  loadPhase: async () => {
    const r = await GetCurrentPhase();
    set({ phase: r.ok ? r.phase : `? (${r.detail})` });
  },

  loadLegalOps: async () => {
    const r = await GetLegalOps();
    set({
      legalOps: r.ok ? (r.kinds ?? []) : [],
      legalOpsPhase: r.ok ? r.phase : `? (${r.detail})`,
    });
  },

  loadRoster: async (franchiseID) => {
    const r = await GetRoster(franchiseID);
    set({ roster: r });
  },

  clearRoster: () => set({ roster: null }),

  loadPool: async () => {
    const r = await GetFreeAgentPool();
    set({ pool: r });
  },

  clearPool: () => set({ pool: null }),
}));
