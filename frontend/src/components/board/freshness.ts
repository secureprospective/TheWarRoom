// The frontend half of the B-5 degradation contract. The Go side is freshness.go — these
// string values cross the IPC boundary, so the union below and the Go consts must agree.
//
// The rule, restated so it survives a refactor: a surface DEGRADES HONESTLY rather than
// failing or lying. Data the app still holds stays on screen and stays LEGIBLE; what
// changes is that the surface states its age. Freshness is therefore an EDGE treatment
// only (Session-C ruling 4) — never a blur, an opacity fade, a hide, or a spinner laid
// over real numbers. If you find yourself dimming the data to signal staleness, that is
// the ruling being violated: the user is reading those numbers precisely BECAUSE the live
// path is down, so that is the worst possible moment to make them hard to read.

export type FreshnessState = 'live' | 'stale' | 'fail';

export type Freshness = {
  state: string;
  fetchedAt: string;
  note: string;
};

// A Wails result may arrive before its store has populated, and Go marshals a nil struct
// as null — so every consumer needs a total function, not an optional-chain at each call
// site. An unrecognized or absent state resolves to 'live' rather than 'fail': the boards
// that predate this contract send nothing, and defaulting them to a red failure edge would
// paint the whole app broken on day one. 'fail' is reserved for a backend that explicitly
// says so.
export function freshnessState(f: Freshness | undefined | null): FreshnessState {
  switch (f?.state) {
    case 'stale':
      return 'stale';
    case 'fail':
      return 'fail';
    default:
      return 'live';
  }
}

// ageLabel renders how old a timestamp is, in the coarsest unit that is still honest.
// Precision is deliberately dropped as the age grows — "3h ago" is more readable than
// "3h 14m ago" and no less useful for judging whether a board can be trusted.
// An unparseable or missing timestamp yields "unknown age", never a fabricated "just now".
export function ageLabel(iso: string, now: number = Date.now()): string {
  if (!iso) return 'unknown age';
  const t = Date.parse(iso);
  if (Number.isNaN(t)) return 'unknown age';

  const mins = Math.floor((now - t) / 60000);
  if (mins < 0) return 'just now'; // clock skew — do not render a negative age
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

// Season phases that mean "this season's competitive data is final". An offseason board is
// NOT stale — its numbers are perfectly fresh, they simply describe a season that has
// ended. Conflating the two would label a correct board as degraded for months at a time,
// which trains the user to ignore the staleness signal entirely.
export function isFinalPhase(phase: string | undefined): boolean {
  return phase === 'OFFSEASON';
}
