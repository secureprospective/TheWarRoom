import { create } from 'zustand';
import { AppInfo } from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

// Build-stamp store. One job: hold the link-time AppInfo (version / commit /
// buildDate) the backend reports, so any surface can display which binary is
// running. The binary is the single source of truth (D-V2) — this store just
// caches the one IPC read; it never derives or duplicates a version string.
interface AppInfoState {
  info: main.AppInfo | null;
  load: () => Promise<void>;
}

export const useAppInfoStore = create<AppInfoState>((set) => ({
  info: null,
  load: async () => {
    try {
      const info = await AppInfo();
      set({ info });
    } catch {
      // A failed AppInfo read is non-fatal — the stamp surface simply shows
      // nothing rather than blocking the shell.
    }
  },
}));
