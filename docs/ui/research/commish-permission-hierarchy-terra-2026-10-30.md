1. **Recommendation: commissioner-built permission sets, added to the existing roles.** A DOT can receive extra powers without becoming a co-commissioner. The commissioner chooses exact actions, groups them under a plain-English name, and assigns the set to named people. The existing commissioner, co-commissioner and DOT roles retain their current meaning.

   **Decision needing Christopher’s approval:** the signed-off census contains commissioner-only controls and special trade and succession rules. This proposal preserves them; it does not silently make them delegable.

2. **Prior art**

   - **MFL:** Commissioner setup and owner “abilities” are separate concepts; abilities control things such as owner chat, not necessarily granular co-commissioner powers. Familiar labels help, but the distinction is easy to miss. [MFL assistant](https://www.myfantasyleague.com/fflnet2001/coach.html), [owner abilities](https://myfantasyleague.wordpress.com/2009/02/20/owner-chat-abilities/).
   - **Sleeper:** Multiple commissioners provide straightforward shared control, but this is a broad promotion rather than a verified task-specific delegation model. [Sleeper](https://support.sleeper.com/en/articles/3971182-what-can-a-commissioner-do).
   - **ESPN:** League Manager powers can be transferred; a second *team manager* is a different, team-level arrangement. Current granular league-admin delegation was not verified. [League Manager powers](https://support.espn.com/hc/en-us/articles/360012530592-League-Manager-Powers), [second team manager](https://support.espn.com/hc/en-us/articles/52638184359700-Adding-a-Second-Team-Manager-LM-Leagues).
   - **Yahoo:** Supports commissioner and co-commissioner assignment. Simple, but the cited help does not establish fine-grained custom powers. [Yahoo](https://help.yahoo.com/kb/fantasy-football/transfer-commissioner-powers-manager-sln6090.html).
   - **Fleaflicker:** Offers co-commissioners with broad powers and documented exceptions. Easy to assign; users must understand what the exceptions mean. [Fleaflicker](https://www.fleaflicker.com/help/commissioner-options).
   - **Fantrax:** Its feature listing reports multiple commissioners; exact current delegated permissions were not independently verified because the page is JavaScript-rendered. [Fantrax](https://www.fantrax.com/premium-league-features).
   - **Discord:** Roles combine permissions, with hierarchy and channel overrides. Powerful, but role order and overrides make “why can this person do this?” difficult. The official hierarchy page returned 403; its search result was available. [Discord](https://support.discord.com/hc/en-us/articles/206141927-How-is-the-permission-hierarchy-structured).
   - **GitHub and Slack:** Custom roles package selected powers; GitHub adds permissions to a base role, while Slack has task-oriented admin roles. Useful precedent for templates, although base-role inheritance and plan restrictions add complexity. [GitHub](https://docs.github.com/en/enterprise-cloud%40latest/organizations/managing-user-access-to-your-organizations-repositories/managing-repository-roles/managing-custom-repository-roles-for-an-organization), [Slack](https://slack.com/help/articles/29800891169939-Create-custom-roles).

3. **The model**

   - **Permission unit:** a canonical **command**, not a whole census control. CM-037, for example, contains review *and* commissioner override; granting the whole control would be unsafe. For `commish.rules.set`, the controlled field and effective timing must also be part of the permission.
   - **Sets are additive:** built-in roles remain unchanged; a named person may hold several sets. Owner abilities remain franchise-specific and separate. A set is not a deputy appointment and cannot change succession priority.
   - **Administration:** only a commissioner creates, edits, assigns or revokes sets. Consequently no delegate can grant powers they do not hold, alter role grants, or modify commissioner authority. Editing a shared set previews everyone affected and requires confirmation; version it so old grants and audit records remain intelligible.
   - **Scope:** support league-wide or specified franchises, season and optional expiry where the command can enforce them. Reject unsupported combinations rather than pretending a scope is enforced. Check expiry at execution against a trusted recorded time, not the client clock.
   - **Conflicts:** automatically refuse a conflicted holder’s action where the action has identifiable affected parties; do not let a set bypass existing trade recusal, appeal routing or other preflight rules. Where affected parties cannot be established reliably, fail closed or send to an eligible commissioner. Commissioner full-control overrides and their existing safeguards remain intact; do not invent a blanket ban on the commissioner.
   - **Never delegable through sets:** census `COnly` controls—CM-001, 002, 005, 021, 027, 033 and 055—plus separately protected commands such as `trade.override`, `contract.action.set`, and `oline.auction.close`. Preserve specific succession, appeal and caretaker rules. Server administration and support remain separate.
   - **Trade caveat:** today reviewers are selected from DOT members. Giving a non-DOT person `trade.review` alone must **not** silently make them a DOT reviewer or change the three-vote rule. Expanding reviewer eligibility needs a separate ruling and implementation.

4. **Fit to the code**

   - `internal/governance/policy.go`: extend `CommandRef` with canonical command identity and validated context. Keep existing built-in checks, then check an active matching set for eligible commands; retain owner ability checks. The commissioner remains unrestricted by sets.
   - `internal/governance/census.go`: retain the existing control and `COnly` data for compatibility; add or connect to a **command-level**, default-deny delegation catalog with field, gravity and scope metadata. Do not infer permission from the UI’s confirmation label: a command’s gravity can increase with circumstances.
   - `internal/commissioner/governance.go`: the authorizer currently maps commands to census IDs, while trade and succession have separate authorization paths. Pass the actual command and trusted arguments through **all** paths, preserve their special checks, and enforce again at execution for UI, CLI and AI. AI acts for a named authorized human and records “via AI.”
   - Keep `RoleGranted` / `RoleRevoked` and deputy ordering unchanged. Proposed private events: `PermissionSetCreated`, `PermissionSetRevised`, `PermissionSetGranted`, `PermissionSetRevoked`; corresponding commands: `commish.permissions.create`, `.revise`, `.grant`, `.revoke`. Store versions, scope and expiry explicitly. Expiration is evaluated from recorded time, not a replay-time wall clock.
   - Extend `State` cloning and `RawCheckpoint` validation/resume with backward-compatible empty defaults. Old records and checkpoints must fold identically; do not retroactively authorize or reclassify historical events. Test full replay versus checkpoint resume, old-ledger fixtures, revocation, expiry, concurrent edits and every execution door. Public league-change entries continue to show who/what/why; set definitions and grants remain private.
   - **Rough effort:** catalog and policy design 2–4 days; event/checkpoint and authorization wiring 4–7; UI and templates 3–5; integration, security and replay testing 3–5. Roughly **12–21 working days**, excluding review and unresolved rulings. Implementation status of each catalog command was not independently established.

5. **Starter sets** — exact identifiers from the 92 distinct commands in `commands.wire.testdata.json`. These are proposals, not claims that every command is implemented. Effective gravity is checked at runtime; the ceiling blocks an action if its circumstances exceed it.

   | Set | Commands | Ceiling |
   |---|---|---|
   | Trade reviewer | `trade.review`, `trade.recuse` | G3 |
   | Stat desk | `commish.stat.queue.approve`, `commish.stat.queue.reject`, `commish.stat.correct` | G2 |
   | Draft assistant | `draft.skip`, `draft.assign`, `draft.timer` | G3 |
   | Rules clerk | `commish.rules.set` **only for field `house.texts`**, subject to validated effective timing | G1 |
   | New-owner helper | `commish.seat.invite`, `commish.seat.invite.revoke` | G2 |
   | League notices | `commish.notice.publish`, `commish.whatsnew.post` | G1 |

   Trade reviewer is useful for packaging but is **not extra authority for an existing DOT**: they already have those review powers. A DOT who needs help with another task receives an additional set, such as Stat desk. A distinct trade-*processing* power is not established by the supplied command list; do not imply review includes override.

6. **People screen**

   Show **People and permissions** with each person’s built-in role, assigned sets and expiry. Beside it, show sets with plain-language actions and holders.

   - **Add person:** choose a set, person, supported scope and optional expiry; review the exact powers before granting.
   - **Edit set:** list actions grouped by job, with consequences and all affected holders; commissioner-only actions are visible but unavailable.
   - **What can Maya do?** Show effective actions and whether each comes from her role or a set, including restrictions.
   - **Who can do this?** Search an action and show eligible people, explaining exclusions such as expiry or trade-party recusal.
   - Permission administration is visible only to authorized people; ordinary owners still see public league changes, not private grants. No Discord-style drag hierarchy is needed.

7. **Five risks and fixes**

   1. **Privilege escalation:** default-deny command catalog, protected-action denylist, validated scopes, server-side checks and tests across every entry point.
   2. **League lockout:** preserve last-commissioner protection and existing deputy/election succession; custom sets never confer or replace succession authority.
   3. **Private-grant leaks:** keep set events and permission views private; test ledger, checkpoints, APIs, search and notifications for leakage.
   4. **Ledger noise:** record actual set changes and revocations once; calculate expiry without repeated entries. Publicly record league actions, not private permission churn.
   5. **Co-commissioner confusion:** label co-commissioner as near-parity and a successor in priority order; label sets as **extra tasks, not a promotion**. Show why each power is available.

**Repository evidence opened on `origin/main`:** `internal/governance/{policy,census,people,seats,succession,state,checkpoint}.go`, `internal/commissioner/governance.go`, `internal/commandcatalog/registry.go`, `docs/commissioner/{Commissioner_Control_Census,README}.md`, and `frontend/src/app/commissioner/commands.wire.testdata.json`. The requested `docs/ui/control-room/Control_Room_Tree_v1.md` **does not exist on `origin/main`**, so its People-branch details were not verified. No files were edited or committed.
