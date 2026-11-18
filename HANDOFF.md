# Handoff

- Baton: Bee, 2026-11-17.
- State: B5 deputy-permission work stopped at D65 discovery; no source changes.
- Findings: `AuthorizeRecords` is the current server-side authorization entry point, with special trade, succession, owner, and rules paths plus generic `GovernanceAuthorizer.Allow`; no single `mayRun` gate exists. `commish.abilities.set` stores existing owner ability flags (13 named abilities) in governance ledger state, not deputy grants. D65 specifies a single gate, private grants, and locked commands, but not the delegation catalog. The Terra permission-hierarchy document proposes command-level sets and six starter examples, explicitly as proposals needing approval.
- Next: get an approved command-level permission/delegability ruling, then implement and test the gate and effective-abilities read.
- Blocked: permission definitions and delegation boundaries are not approved. No tests run; no code changed.
