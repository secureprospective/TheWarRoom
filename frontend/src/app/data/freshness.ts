// The frontend half of the B-5 degradation contract. The Go side is freshness.go: these
// string values cross the IPC boundary, so the union below and the Go consts must agree.
//
// A surface degrades honestly rather than failing or lying. Data the app still holds stays on
// screen and stays legible; what changes is that the surface states its age. Freshness is an
// edge treatment only (Session-C ruling 4): never a blur, an opacity fade, a hide, or a
// spinner laid over real numbers.
export type FreshnessState = 'live' | 'stale' | 'fail';
