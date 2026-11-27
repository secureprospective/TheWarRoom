# Line budget reason for `session/f6-phone-mypicks-1127`

- Restrict `myPicks` used-slot tracking to scheduled picks (`(p.extra ?? 0) === 0`) so inserted extra picks do not hide unused or owed picks at the same round and slot.
- Refresh `TeamTab`'s `NativeHQ` reading and open bench slot count after a successful phone `draft.pick` without unmounting the open draft room sheet, with Vitest regressions covering both fixes.
