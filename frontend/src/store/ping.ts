import { create } from 'zustand';
import { Ping } from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

// WF5 rule: the IPC call lives in the store, never in the component. The
// component reads this slice and dispatches ping(); it never touches
// window.go directly.
interface PingState {
  result: main.PingResult | null;
  loading: boolean;
  ping: () => Promise<void>;
}

export const usePingStore = create<PingState>((set) => ({
  result: null,
  loading: false,
  ping: async () => {
    set({ loading: true });
    const result = await Ping();
    set({ result, loading: false });
  },
}));
