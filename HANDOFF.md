# Roles catalog handoff

## Baton
ClaudeBox/reviewer — 2026-10-22. Bee completed the roles-catalog brief on `session/roles-catalog`. Do not push without permission.

## Where it stands
- All 133 command rows match the accepted Role/Gravity table; IDs, versions and availability are unchanged. Metadata only, no role enforcement.
- `internal/composition/selfprotect_test.go` rejects unset role or gravity. Deliberately unsetting TargetArchive's role failed with the intended diagnostic; restored.
- `internal/app/sources/api.go` checks Ready before SetPhotos reads/writes settings or publishes the switch. Existing test covers refusal with unchanged state and success after startup; it failed before the fix.

## Next move
Review the two commits; perform independent review and throwaway-profile live checks before merge.

## Blocked on
No implementation blocker. Independent review and live GUI verification remain unknown.

## Tried and rejected
- Shared commissioner metadata cannot express the ruling; each method now declares its own role and gravity.
- Shared lint cache returned stale deleted-worktree diagnostics; a worktree-local cache avoids them.
