import { create } from 'zustand';
import { GetPlayerScore } from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

// Inspector store — the shared SELECTION + per-player breakdown gateway for the B-4a Contextual
// Inspector. Selection is genuinely cross-component state (an M1 board row sets it, the Inspector
// panel reads it), so it lives in a store, not local component state; per WF5 the IPC call
// (GetPlayerScore) lives here too, never in the panel. The board writes only the mflID; the store
// fetches the full DTO (name/pos included), so the click site needs no player data.

// selectSeq monotonically tags each select() so a slow earlier GetPlayerScore cannot overwrite a
// newer selection's result (fast row-clicking: pick A, then B, B dispatched second but A resolves
// last → A must not win). Only the latest select commits — the powerRankings guard, applied here.
let selectSeq = 0;

interface InspectorState {
  selectedMflID: string | null;
  // openNonce increments on EVERY select() (even re-selecting the same id), so the shell can open the
  // inspector on each click — keying an open-effect on selectedMflID alone would no-op when the user
  // closes the panel and clicks the SAME row again (GLM L1).
  openNonce: number;
  player: main.PlayerScoreDTO | null;
  label: string; // BasePoints-proxy honesty string (rendered verbatim)
  warning: string; // names-offline degradation (breakdown still complete)
  loading: boolean;
  notFound: boolean; // resolved OK but no score row for this id (rescore / off-board)
  error: string;
  select: (mflID: string) => Promise<void>;
  clear: () => void;
}

export const useInspectorStore = create<InspectorState>((set) => ({
  selectedMflID: null,
  openNonce: 0,
  player: null,
  label: '',
  warning: '',
  loading: false,
  notFound: false,
  error: '',

  select: async (mflID) => {
    const seq = ++selectSeq;
    set((s) => ({
      selectedMflID: mflID,
      openNonce: s.openNonce + 1,
      loading: true,
      error: '',
      notFound: false,
      label: '', // drop the prior player's honesty string until this fetch resolves (GLM L9)
      warning: '',
    }));
    try {
      const res = await GetPlayerScore(mflID);
      if (seq !== selectSeq) return; // a newer selection superseded this one
      if (!res.ok) {
        set({ loading: false, error: res.error || 'Could not load this player.', player: null });
        return;
      }
      set({
        loading: false,
        notFound: !res.found,
        player: res.found ? res.player : null,
        label: res.label,
        warning: res.warning,
        error: '',
      });
    } catch (e) {
      if (seq !== selectSeq) return;
      set({
        loading: false,
        error: `The engine was unreachable (${e instanceof Error ? e.message : String(e)}).`,
        player: null,
      });
    }
  },

  // clear deselects (closing the inspector keeps the selection so re-opening is instant; clear is for
  // an explicit deselect). Bumps the seq so any in-flight fetch is discarded.
  clear: () => {
    selectSeq++;
    set({ selectedMflID: null, player: null, notFound: false, error: '', warning: '' });
  },
}));
