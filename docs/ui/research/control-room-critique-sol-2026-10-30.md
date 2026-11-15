# Verdict: D overall

The design promises a trustworthy control surface, but the clickable mock teaches false success, unsafe delegation and incomplete workflows.

**Scope:** Read-only source review; no files changed or commits made. Backend/frontend comparisons use `origin/main` **364379ba**. Browser rendering, runtime enforcement and tests were **not exercised**.

Citations below: **Mock** = `docs/ui/mocks/control-room-mock-v1.html`; **Tree** = `docs/ui/control-room/Control_Room_Tree_v1.md`; **Terra** = `docs/ui/research/commish-permission-hierarchy-terra-2026-10-30.md`. Code citations refer to `origin/main`.

## 1. Grades

| Area | Grade | Justification |
|---|---|---|
| Organization/findability | C | Useful task labels, but nine duplicated navigation branches and partial routing undermine “no digging.” [Mock:535, 613–615, 903–906] |
| Sandbox depth | C | Broad command inventory; complex repairs remain disconnected actions with generic previews. [Mock:571–577, 782–785] |
| Owner-overlap discipline | C | Clear written principle; mock recreates trade cards and owner-facing summaries locally. [Tree §4; Mock:666–676, 732–736] |
| Honesty versus code | F | Missing job producers appear live; unbuilt actions can report themselves recorded. [Tree:299; Mock:544–555, 800–804, 925] |
| Permission model | D | Additive sets are sensible; the implementation contradicts crucial restrictions in its own research. [Terra §3; Mock:847–889] |
| Normie usability | D | Raw command checkboxes, generic forms and nonfunctional affordances require technical interpretation. [Mock:870, 922–925] |
| Fit to built commissioner UI | D | Replaces existing schema-driven forms and backend previews with canned inputs and confirmations. [Mock:720–725; `RuleChangeForm.tsx`:59–95] |
| Overall | D | Directionally useful organization, not yet a reliable behavioral specification. [Findings below] |

## 2. Top 10 problems, worst first

### 1. “Recorded” does not mean anything changed

`record()` adds ledger prose, closes the interface and announces success. Rule saving ignores the entered value and reason; fixes ignore their inputs. Planned pick commands can also “APPLY.”

**Evidence:** Mock:627, 783–785, 800–804, 911, 914, 929.  
**Hurts:** Everyone, especially the commissioner verifying a repair.  
**Fix:** Either simulate coherent state transitions or explicitly report “Demonstration only—nothing changed.” Disable execution for unbuilt commands.

### 2. The permission editor offers prohibited delegation

“Trade processor” grants `trade.appeal.rule` to a DOT. Existing appeal judges are eligible commissioners/deputies, not arbitrary delegates. `commish.standings.record.adjust` is absent from `LOCKED`, despite belonging to commissioner-only CM-033. `commish.rules.set` has no enforceable field restriction in the set data/editor.

**Evidence:** Mock:847, 850, 853, 870; Terra:18, 23; `internal/governance/policy.go`:88–97; census CM-033.  
**Hurts:** DOT, co-commissioner, owners.  
**Fix:** Build the delegation catalog from canonical commands **and validated context**, preserving protected controls and special appeal routing.

### 3. Safety is copy, not demonstrated behavior

Required reasons are not checked. Trade overrides have no reason input. Generic actions choose hold-versus-click through a command-name regex, not backend safeguards. Static “touches 1 team” previews can describe a transaction involving several teams.

**Evidence:** Mock:671, 783–785, 924–929; `internal/commissioner/safeguards.go`:20–55.  
**Hurts:** Commissioner and affected owners.  
**Fix:** Reuse backend-generated safeguards and real before/after previews. Block confirmation until required information is present.

### 4. Permission changes take effect before review

Checking a box immediately edits the shared set and claims a new version. There is no review of added authority, affected holders, scope or pending work.

**Evidence:** Mock:889; Terra:20 requires preview and confirmation.  
**Hurts:** Commissioner, delegates, owners.  
**Fix:** Stage changes, show a permission diff and affected holders, then explicitly confirm a versioned revision.

### 5. Jobs invent operational truth

Vacancies, invites, missed-lineup flags and feed health appear as ordinary operational jobs despite acknowledged missing producers. Setup is marked blocked although the backend emits it live while rulings remain unset.

**Evidence:** Mock:544–555; Tree:287, 299; `internal/commissioner/jobs.go`:107–140; `jobs_league.go`:13–44.  
**Hurts:** Commissioner relying on the dashboard as an attention queue.  
**Fix:** Separate sample proposed jobs from supported jobs. Preserve backend state semantics; label missing producers at each row.

### 6. “Who can do this?” gives false answers

Unknown text still returns the commissioner and usually the co-commissioner. Results do not evaluate expiry, scope, field restrictions or conflicts. Role switching does not disable permission administration or most protected actions.

**Evidence:** Mock:857–883, 926; only rollover is locked in the shared desk renderer at 741.  
**Hurts:** Commissioner, co-commissioner, DOT.  
**Fix:** Resolve an exact supported action and context before listing eligibility. Explain exclusions. Render each persona from an authoritative capability response.

### 7. Findability claims exceed the navigation

Search omits permission sets and many administrative actions. Situation steps frequently route to a whole branch rather than a prefilled tool; trade-related steps all open the same review job. The command line implements only three task commands plus help/find.

**Evidence:** Mock:635–645, 892–906; Tree M1.  
**Hurts:** Nontechnical commissioners during an incident.  
**Fix:** One complete action index, shared by search, situations, launchers and commands, with context-preserving destinations.

### 8. Complex rule editing regresses existing functionality

Maps, lists and structured rules become a single text input unless they have enumerated choices. The existing UI already converts schema-driven drafts, validates natural values and requests backend previews.

**Evidence:** Mock:720–725; `internal/leaguerules/catalogue.go`:18–25; `RuleChangeForm.tsx`:59–95.  
**Hurts:** Commissioner configuring scoring, alignment, brackets or salary tables.  
**Fix:** Reorganize the existing editors rather than replacing them with generic mock forms.

### 9. Owner reuse is asserted, not actually modeled

The trade card says it is the same owner card, but it is separately hardcoded inside Control Room. Season Desk also reproduces auction, draft-order, schedule and seed summaries. No actual owner-screen handoff carries context or a return route.

**Evidence:** Mock:666–676, 732–736; Tree:254–273, 312.  
**Hurts:** Commissioner and owners facing inconsistent views.  
**Fix:** Demonstrate shared cards or explicit contextual links; retain only operational controls locally.

### 10. Accessibility and overflow are unresolved

Job rows and rule rows are clickable non-button elements without keyboard handlers. Holds are pointer-only and last 900ms, versus the binding roughly 600ms. Multiple containers hide overflow; only Rules/Picks receive fit calculations.

**Evidence:** Mock:348–354, 649, 705, 708–710, 929–936; `UI_Direction_v2_2026-10-30.md`:23.  
**Hurts:** Keyboard users and commissioners using zoom or smaller windows.  
**Fix:** Semantic controls, accessible confirmation alternatives and bounded internal scrolling. **Actual clipping remains unverified without browser testing.**

## 3. Directional bias

- **Inventory mistaken for capability.** Naming commands and displaying forms substitutes for showing a successful, verifiable repair. [Mock:743–785]
- **Situations used as a routing crutch.** The stories are useful, but “Nothing” under remaining work conceals dependencies—for example restoring a cut player still requires checking existing waiver claims. [Mock:572]
- **Breadth mistaken for simultaneous exposure.** Nine branches appear in both rail and tabs; permissions expose four columns of raw verbs. Breadth does not require making commissioners understand the command catalog. [Mock:535, 613–615, 870]
- **Earlier criticism filed rather than fully incorporated.** The triage says statuses were split, missing jobs relabeled and undo narrowed; the body still contains blanket B labels, unsupported jobs and “undo anything.” [Tree:99–106, 143, 151, 230, 299–304]

## 4. Missing commissioner workflows

I checked the complete **92-command fixture**, not just visible buttons. These are missing **workflows**, not claims that correction primitives are absent:

1. **One coordinated mistaken-release repair:** roster, contract, dead cap, intervening claims and subsequent awards reviewed together. The situation supplies three separate commands; nothing coordinates their consequences. [Mock:572; Tree:127; fixture: `correct.roster`, `commish.contract.edit`, `commish.cap.adjust`, `waivers.award.override`]
2. **A disputed RFA matching case:** show tender, offer, matching deadline, decision and compensation together. Rule settings exist; the tree itself acknowledges no match-dispute command. [Tree:180–181, 311; command fixture]
3. **Expansion/contraction asset reconciliation:** allocated players, contracts, cap liabilities and picks need a coherent migration plan, not just team-count edits. Guided expansion is explicitly a gap; shrinking also requires vacant seats and revoked invitations. [Mock:590; Tree:288; `internal/governance/people.go`:32–75]

## 5. Over-built

- Remove either the duplicate branch tabs or duplicate rail leaves. [Mock:613–615]
- Merge the second feed-health surface; League and Server both display feed controls. [Mock:828, 840]
- Replace raw permission-command selection with job templates and advanced disclosure. [Mock:849–855, 870]
- Remove unsupported success paths, not the owner-approved future capabilities. [Mock:800–804, 925]

## 6. Permission model attack

- **Escalation:** delegate broad rule edits or commissioner-only record adjustments through the holes above. **Unanswered.** [Mock:847–870]
- **Conflict of interest:** DOT appeal judging contradicts existing judge selection; no context-sensitive exclusion appears in eligibility results. **Unanswered.** [Mock:850, 880–883; policy:88–97]
- **Lockout:** backend protects the last commissioner, but the mock never demonstrates rejection or recovery. **Backend safeguard exists; UI behavior unverified.** [`people.go`:145–146; Mock:925]
- **Co-commish confusion:** explanatory text promises restrictions while the switched view still offers protected controls. **Unanswered.** [Mock:633, 926]
- **Private-grant leakage:** grants use the generic public-looking ledger recorder; permission administration has no role gate. Terra explicitly requires private events and leak testing. **Backend leakage not established.** [Mock:627, 872–877; Terra:31–32, 62]
- **Mid-season shared-set edit:** checkbox changes immediately alter every holder without confirmation, concurrency handling or pending-action policy. **Unanswered.** [Mock:889]

## 7. Three things to preserve

1. **Jobs-first entry**, rather than settings-first administration. [Tree M3; Mock:647–654]
2. **Situation → tools → remaining judgment → verification** structure. [Mock:692–694]
3. **Additive task sets separate from roles and succession**, provided Terra’s restrictions survive implementation. [Terra:19–24]

## 8. One more day: five changes

1. Stop false success; label and disable unsupported execution.
2. Correct delegation boundaries and require reviewed set revisions.
3. Reuse existing schema editors, previews and backend safeguards.
4. Complete the shared action index and contextual routing.
5. Exercise five full tasks across commissioner/co-commissioner/DOT views, including denied actions, keyboard operation and zoom.
