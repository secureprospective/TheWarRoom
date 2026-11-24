# Catalogue report: which league commands lack a preview or an undo rule

**134 commands change something. 97 have both a preview and an undo rule. 11 have neither,
12 have a preview but no undo rule, 14 have an undo rule but no preview.**
Read commands are left out. Source: every entry of `commandcatalog.GenerationRegistry()`
(`internal/commandcatalog/generation.go:18`), the same registry the CLI, MCP and browser tables are generated from,
dumped 2026-11-24 on main `5097adb50`. The registry still holds desktop-only doors that desktop removal (T710) deletes.

**What it means for a league.** A command with no preview runs the moment it is sent: the commissioner or owner
never sees "this is what will change" first. A command with no undo rule has no declared way back: a mistake stays
until someone builds a correction by hand. The "neither" group below is where a slip costs the most. Gravity runs
G0 (harmless) to G3 (league-wide or irreversible); start with the highest.

**Before the Dec 1 rehearsal drafts.** Seven commissioner draft-room commands are G3 with a preview but no undo
rule: `draft.assign`, `draft.change`, `draft.insert`, `draft.revert`, `draft.skip`, `draft.timer`,
`draft.startup.open`. The commissioner sees each change before it happens, but a confirmed mistake has no declared
way back except another draft command. `draft.pick` (an owner's own pick) has neither a preview nor an undo rule.

## How it was read
- Preview: the command's `Modes` include `preview` (`internal/commandcatalog/command.go:17-29`).
- Undo rule: the command's `Undo` is set and is not `none`. Most native league commands declare `checked-reversal`
  (`internal/commandcatalog/native_mutations.go:171-199`). This is declared metadata, not a tested reversal.

## Neither a preview nor an undo rule (11)
| Command | Kind | Permission | Gravity | Undo rule |
|---|---|---|---|---|
| `draft.pick` | domain | command:draft.pick | G1 | none |
| `trade.draft.delete` | domain | franchise-owner | G1 | none |
| `feedback.report.create` | domain | league-member | G0 | none |
| `mflmoves.plan.done` | domain | franchise-owner | G0 | none |
| `mflmoves.plan.save` | domain | franchise-owner | G0 | none |
| `mflmoves.plan.withdraw` | domain | franchise-owner | G0 | none |
| `mflmoves.refresh` | domain | franchise-owner | G0 | none |
| `mflmoves.team.set` | domain | franchise-owner | G0 | none |
| `player.note.set` | domain | franchise-owner | G0 | none |
| `player.watch.order` | domain | franchise-owner | G0 | none |
| `player.watch.set` | domain | franchise-owner | G0 | none |

## Preview, but no undo rule (12)
| Command | Kind | Permission | Gravity | Undo rule |
|---|---|---|---|---|
| `commish.league.create` | domain | command:commish.league.create | G3 | none |
| `commish.league.seed_held` | domain | command:commish.league.seed_held | G3 | none |
| `draft.assign` | domain | command:draft.assign | G3 | none |
| `draft.change` | domain | command:draft.change | G3 | none |
| `draft.insert` | domain | command:draft.insert | G3 | none |
| `draft.revert` | domain | command:draft.revert | G3 | none |
| `draft.skip` | domain | command:draft.skip | G3 | none |
| `draft.startup.open` | domain | command:draft.startup.open | G3 | none |
| `draft.timer` | domain | command:draft.timer | G3 | none |
| `draft.archive` | domain | command:draft.archive | G2 | none |
| `oline.auction.close` | commish | command:oline.auction.close | G2 | none |
| `backup.now` | domain | commissioner-or-co-commissioner | G1 | none |

## Undo rule, but no preview (14)
| Command | Kind | Permission | Gravity | Undo rule |
|---|---|---|---|---|
| `league.stand_up` | operator | offline-operator | G3 | fresh-folder-only |
| `mfl.bid.handoff` | gm,commish,admin | local-desktop-owner | G2 | withdraw-local-plan |
| `mfl.bid.posted` | gm,commish,admin | local-desktop-owner | G2 | withdraw-local-plan |
| `mfl.bid.withdraw` | gm,commish,admin | local-desktop-owner | G2 | draft-new-local-plan |
| `mfl.trade.accept` | gm,commish | local-desktop-owner | G2 | withdraw-local-plan |
| `draft.queue.set` | domain | franchise-owner | G1 | replace-own-private-queue |
| `mfl.bid.revise` | gm,commish,admin | local-desktop-owner | G1 | revise-local-plan |
| `mfl.bid.submit` | gm,commish,admin | local-desktop-owner | G1 | withdraw-local-plan |
| `mfl.lineup.set` | gm,commish | local-desktop-owner | G1 | withdraw-local-plan |
| `mfl.pool.refresh` | gm,commish,admin | local-desktop-owner | G1 | refresh-again |
| `trade.draft.save` | domain | franchise-owner | G1 | delete-own-private-proposal |
| `home.layout.set` | domain | account-owner | G0 | replace-own-account-preference |
| `job.snooze.set` | domain | account-owner | G0 | replace-own-account-snooze |
| `job.snooze.wake` | domain | account-owner | G0 | clear-own-account-snooze |

## Both (97)
`bid.place`, `bid.raise`, `bids.award.override`, `bids.process`, `bracket.advance`, `bracket.build`, `bracket.edit`, `bracket.preset.save`, `bracket.seed`, `calendar.window.move`, `calendar.window.remove`, `calendar.window.set`, `commish.abilities.set`, `commish.calendar.exception`, `commish.cap.adjust`, `commish.cap.relief`, `commish.contract.edit`, `commish.correction.revert`, `commish.deputy.order`, `commish.deputy.permissions.set`, `commish.franchise.direction.set`, `commish.franchise.edit`, `commish.lineup.correct`, `commish.lineup.set`, `commish.notice.publish`, `commish.player.position`, `commish.player.position.clear`, `commish.reserve.exception`, `commish.role.grant`, `commish.role.revoke`, `commish.rules.set`, `commish.score.franchise`, `commish.score.player`, `commish.seat.caretaker`, `commish.seat.invite`, `commish.seat.invite.redeem`, `commish.seat.invite.revoke`, `commish.seat.remove`, `commish.seat.replace`, `commish.standings.override`, `commish.standings.override.clear`, `commish.standings.record.adjust`, `commish.stat.correct`, `commish.stat.queue.approve`, `commish.stat.queue.reject`, `commish.succession.accept`, `commish.succession.activity`, `commish.succession.decline`, `commish.succession.election.open`, `commish.succession.inactivity.set`, `commish.succession.periods.set`, `commish.succession.stepdown`, `commish.succession.takeover`, `commish.succession.voteout.open`, `commish.visibility.set`, `commish.vote.cast`, `commish.vote.close`, `commish.vote.tiebreak`, `commish.whatsnew.post`, `contract.action.set`, `contract.decision.set`, `correct.pick`, `correct.provenance`, `correct.roster`, `correct.transaction`, `draft.order.auto`, `draft.order.pattern`, `draft.order.set`, `draft.order.tiebreak`, `draft.start`, `history.link`, `roster.add`, `roster.cut`, `roster.reserve`, `rule.exception`, `schedule.edit`, `schedule.generate`, `schedule.publish`, `season.rollover`, `title.set`, `trade.accept`, `trade.appeal`, `trade.appeal.rule`, `trade.decline`, `trade.hold`, `trade.offer`, `trade.override`, `trade.recuse`, `trade.review`, `trade.withdraw`, `waivers.award.override`, `waivers.claim`, `waivers.claim.cancel`, `waivers.claim.order.set`, `waivers.order.auto`, `waivers.order.set`, `waivers.process`
