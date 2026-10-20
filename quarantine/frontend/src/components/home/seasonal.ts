import { main } from '../../../wailsjs/go/models';

// Seasonal-card selection — PURE (no IPC, no clock reads beyond the `now` parameter), so the
// mapping from league state to the seasonal slot is inspectable in one place and testable if a
// frontend runner ever lands (M12: behaviour lives in data, not in conditionals scattered across
// the render).
//
// Spec reconciliation (UI_Direction_Document §11): the locked trigger map lists NINE seasonal cards
// keyed off calendar DATES (contract options, RFA tender, UFA bidding, re-signing, rookie draft,
// UDFA, cut day, in-season). The Phase-1 backend does not carry those dates — it knows THREE phases
// (domain.Phase) plus the append-only commissioner calendar. Hardcoding nine date windows against
// data the engine cannot confirm would be inventing state, so the alpha set is driven by what the
// backend actually knows:
//
//   1. OFFSEASON      — the transaction season: the queue of PLANNED commissioner ops
//   2. REGULAR_SEASON — in-season: the league asset pulse off the persisted M1 board
//   3. PLAYOFFS       — the season boundary: rollover readiness
//   4. (overlay) DEADLINE — promoted above the phase card whenever a PLANNED op is imminent
//
// Card 4 is the honest generalisation of the spec's date-triggered family: the commissioner
// schedules the real deadline as a calendar blob, and the card counts down to it — rather than the
// app asserting a date it was never told.

export type SeasonalKind = 'deadline' | 'offseason' | 'inseason' | 'playoffs' | 'unknown';

export interface SeasonalCard {
  kind: SeasonalKind;
  title: string;
  // `note` states what the card is showing, in the operator's language. Never a marketing line.
  note: string;
  // The PLANNED events this card is about, soonest first. Empty is a valid, honest state.
  planned: main.CalendarEventDTO[];
  // Set only on the deadline card: the imminent event and its remaining time.
  imminent?: main.CalendarEventDTO;
}

// A PLANNED op inside this window promotes the deadline card over the phase card. Seven days is
// the tightest window in the locked trigger map (RFA tender: May 1 → May 3) rounded up to a week,
// so a real deadline is on screen for at least as long as the spec's shortest window.
const DEADLINE_WINDOW_MS = 7 * 24 * 60 * 60 * 1000;

// selectSeasonal picks the single active seasonal card. Only one is ever active (§11).
export function selectSeasonal(
  phase: string,
  events: main.CalendarEventDTO[],
  now: number,
): SeasonalCard {
  const planned = plannedAhead(events, now);
  const imminent = planned.find((e) => {
    const t = parseAt(e.scheduledAt);
    return t !== null && t - now <= DEADLINE_WINDOW_MS;
  });

  if (imminent) {
    return {
      kind: 'deadline',
      title: 'DEADLINE WINDOW',
      note: 'A commissioner op is scheduled inside the next 7 days.',
      planned,
      imminent,
    };
  }

  switch (phase) {
    case 'OFFSEASON':
      return {
        kind: 'offseason',
        title: 'OFFSEASON',
        note: planned.length
          ? 'Scheduled commissioner ops, soonest first.'
          : 'No commissioner ops scheduled. Schedule one from the calendar.',
        planned,
      };
    case 'REGULAR_SEASON':
      return {
        kind: 'inseason',
        title: 'IN SEASON',
        note: 'League asset pulse from the persisted board. Weekly matchups and the playoff picture need league standings, which Home does not read yet.',
        planned,
      };
    case 'PLAYOFFS':
      return {
        kind: 'playoffs',
        title: 'PLAYOFFS',
        note: 'Season boundary. Rollover to the next league year runs from Control once the last result is in.',
        planned,
      };
    default:
      return {
        kind: 'unknown',
        title: 'SEASONAL',
        note: 'Season phase unavailable — the card cannot say where the league year stands.',
        planned,
      };
  }
}

// plannedAhead returns PLANNED events still in the future, soonest first. FIRED and CANCELLED rows
// are history (the calendar board renders those); an unparseable or past timestamp is dropped
// rather than rendered as an expired countdown.
function plannedAhead(events: main.CalendarEventDTO[], now: number): main.CalendarEventDTO[] {
  return events
    .filter((e) => {
      if (e.status !== 'PLANNED') return false;
      const t = parseAt(e.scheduledAt);
      return t !== null && t >= now;
    })
    .sort((a, b) => (parseAt(a.scheduledAt) ?? 0) - (parseAt(b.scheduledAt) ?? 0));
}

// parseAt returns epoch ms, or null when the backend's ISO-8601 string does not parse — the caller
// drops the row instead of rendering NaN as a time.
export function parseAt(iso: string): number | null {
  const t = Date.parse(iso);
  return Number.isNaN(t) ? null : t;
}

// countdown renders remaining time at the coarsest useful unit. Under a minute reads "due now"
// rather than counting seconds the operator cannot act on.
export function countdown(target: number, now: number): string {
  const ms = target - now;
  if (ms <= 0) return 'due now';
  const mins = Math.floor(ms / 60000);
  if (mins < 1) return 'due now';
  if (mins < 60) return `${mins}m`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ${mins % 60}m`;
  const days = Math.floor(hours / 24);
  return `${days}d ${hours % 24}h`;
}

// KIND_LABELS maps the transaction Kind vocabulary (internal/transactions/request.go) onto the
// operator-facing card label. An unmapped kind falls back to its raw value — never a silent blank.
const KIND_LABELS: Record<string, string> = {
  BUYOUT: 'Buyout',
  SIGN: 'Signing',
  WAIVER: 'Waiver',
  TAG: 'Franchise tag',
  EXTENSION: 'Extension',
  RESTRUCTURE: 'Restructure',
  TRADE: 'Trade',
  ROSTER_STATUS: 'Roster status',
  RETIREMENT: 'Retirement',
  DEATH: 'Player death',
  CAP_RELIEF: 'Cap relief',
  ADVANCE_PHASE: 'Advance phase',
  ROLLOVER_SEASON: 'Season rollover',
  SET_SIGNING_WINDOW: 'Signing window',
};

export function kindLabel(kind: string): string {
  return KIND_LABELS[kind] ?? kind;
}
