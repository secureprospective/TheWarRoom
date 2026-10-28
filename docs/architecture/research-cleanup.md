# Research screen cleanup

Removed screens do not justify removing shared operational code or stored history.

- Counterparties, Offers history, Draft Lab and Market Lab APIs and their frontend views were already
  absent at the starting revision (`e3bc9d41`). Their retired Wails methods stay unregistered.
- Archive's board and IPC were removed by `0ba00108`. Its checked snapshot reader stays: Comms consumes
  roster weeks and NFL games, and startup compilation warms that cache.
- The orphan DataSources and CacheTools views and their view-only tests/styles are deleted. The
  authoritative endpoint CSV retires the removed workspaces and merged children; the browser no longer
  needs its parallel retirement overlay. No table, raw archive or stored personal offer is deleted.
- `internal/app/offers` stays for live pool, bids and personal-list callers; `draft` for live draft
  reports/NFL selections; `market` for the approved asset board and held market reports/pool facts.
- `internal/app/sources` stays for background Signals ingestion, crosswalk linking, checked-fact
  compilation, status-strip health, settings and the named refresh/cache/parameter commands. The
  command state still rereads Sources/crosswalk/parameters after actions; those reads are live even
  without the removed view. Their API and wire tests stay. Commissioner catalogue rows remain kept.

Source trace: composition command bindings, `sources_runtime.go`, `offers_reading.go`, browser
`commands/registry.ts`, `datasources/state.ts`, `pool/provider.ts` and `personal/state.ts`.
`TestResearchBindingsStayRetired` rejects removed IPCs; frontend registry tests reject their routes.
No GUI acceptance is claimed.
