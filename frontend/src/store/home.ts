import { create } from 'zustand';
import { GetCurrentPhase, GetCalendarEvents } from '../../wailsjs/go/main/App';
import { main } from '../../wailsjs/go/models';

// Home store — the league-landing gateway (WF5: IPC lives in the store, never in the card).
//
// B-4b scope decision (Christopher, 2026-07-25): Home reads LOCAL state only — the season phase
// and the commissioner calendar, both served off the concrete SQLite store with no network. It
// deliberately does NOT call GetPowerRankings: that fetch is live MFL and treats a standings
// failure as FATAL (m2_app.go:79), which would put a hard-failure network path on the calmest
// screen in the product. A standings card waits for a cached/soft-fail read.
//
// The M1 pulse card reads the harness store's already-persisted rankings rather than refetching;
// its only network touch is the cached display-name directory, which degrades rather than fails
// (m1_app.go GetRankings) — so Home still cannot be blanked by an MFL outage.

interface HomeState {
  phase: string;
  events: main.CalendarEventDTO[];
  loading: boolean;
  // phaseError / calendarError are tracked SEPARATELY so one failing read degrades only its own
  // card. A shared error string would blank both surfaces on either failure.
  phaseError: string;
  calendarError: string;
  load: () => Promise<void>;
}

export const useHomeStore = create<HomeState>((set) => ({
  phase: '',
  events: [],
  loading: false,
  phaseError: '',
  calendarError: '',

  load: async () => {
    set({ loading: true });
    // Settled, not all-or-nothing: the calendar must still render if the phase read fails.
    const [phaseRes, calRes] = await Promise.allSettled([GetCurrentPhase(), GetCalendarEvents()]);

    const next: Partial<HomeState> = { loading: false };

    if (phaseRes.status === 'fulfilled') {
      next.phase = phaseRes.value.ok ? phaseRes.value.phase : '';
      next.phaseError = phaseRes.value.ok ? '' : phaseRes.value.detail || 'Season phase unavailable.';
    } else {
      next.phase = '';
      next.phaseError = `The engine was unreachable (${reason(phaseRes.reason)}).`;
    }

    if (calRes.status === 'fulfilled') {
      next.events = calRes.value.ok ? (calRes.value.events ?? []) : [];
      next.calendarError = calRes.value.ok ? '' : calRes.value.detail || 'Calendar unavailable.';
    } else {
      next.events = [];
      next.calendarError = `The engine was unreachable (${reason(calRes.reason)}).`;
    }

    set(next);
  },
}));

function reason(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}
