# Franchise HQ v10 wiring

The locked `docs/ui/mocks/franchise-hq-mock-v10.html` owns layout and styling.
`FranchiseHQ` replaces only the HQ route's outer shell; other workspaces keep their shell.
Feature API adapters call existing held reads; parsers reject malformed replies before mapping.
`useHQRead` uses the shared keyed query, fences late replies, and re-reads on season/move events.
The existing contract reader retains its own request fencing. A failed or pending read displays
`Not available yet`, never mock fixtures or a previous franchise's facts.

The snapshot supplies franchise/roster identity and cap limit/room. The settings report supplies
roster, IR, taxi and starter counts and position limits. The lineup read supplies saved starters
and legality; existing command-registry drafts remain the sole owner of lineup edits.
Signed MFL adjustments are not necessarily dead money: no sum is relabelled as dead cap.

This is a staged local build; the report records the final wired regions and remaining gaps.
