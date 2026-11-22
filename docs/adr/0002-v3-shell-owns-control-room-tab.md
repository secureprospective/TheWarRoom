# Desktop v3 shell owns Control Room tab state

Status: Retired 2026-11-22 (a screen-wiring detail, not a lasting decision; the code and its tests carry it)
Date: 2026-11-20

In the desktop v3 shell (on the v3 UI branch: `V3Shell.tsx`), the outer frame owns the active Control Room tab (`crTab`) and passes `tab` and `setTab` down into `ControlRoomV3`, which syncs its underlying `useHostedControlRoom` hook (`frontend/src/app/commissioner/HostedControlRoom.tsx`) to follow the shell on mount and on every shell click while propagating room-initiated jumps (such as a job or situation card opening a tool on another tab) back up to `setTab` after mount. We do this because the v3 frame renders Control Room tab controls in two shell-owned regions outside the room body—the top workspace tab strip (`.ws-tabs`) and the left navigation rail leaves (`.rail .leaf`)—while progressively mounting individual v3 tab bodies or falling back to `HostedControlRoomBody` for unported tabs.

## Considered Options

- **Let `useHostedControlRoom` own tab state internally (as legacy `HostedControlRoom.tsx` did)**: Rejected because `HostedControlRoom` drew its own `.ws-tabs` inside the work area and had no left-rail leaf buttons; in v3, the rail and workspace header live in the shell above `<Suspense>` and must read and switch `crTab` directly.
- **Make `ControlRoomV3` purely controlled and remove `c.tab` from `useHostedControlRoom`**: Rejected because internal room actions (`openTool`, situation routing) switch tabs when opening a tool tray, and existing `HostedControlRoom` tests and fallback bodies rely on `useHostedControlRoom`'s `c.tab` and `c.navigateTo`.

## Consequences

- Tab state stays synchronized between the outer shell navigation and internal Control Room tool jumps during incremental v3 migration.
- Unported Control Room tabs can continue using `HostedControlRoomBody` without rewriting `useHostedControlRoom`.
