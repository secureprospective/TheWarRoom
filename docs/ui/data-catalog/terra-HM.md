# HM — Home v5 data catalog

Mock-only demo selectors are identified as such. `NEW` means no serving path established in the files inspected, not proof that no implementation exists. Actions described as inert have no matching handler in the mock.

| ID | Screen › Region › Element | Shows / does | Data shape | Source on origin/main (path:symbol) or NEW | Read/Write | Refresh | Role/state variants |
|---|---|---|---|---|---|---|---|
| HM-001 | HM › Shell › League identity | League name and current week or offseason | {league_id,name,phase,week} | NEW | Read | per-load | league scoped; offseason |
| HM-002 | HM › Shell › Live game count | Six live games beside week | {week,live_game_count} | PARTIAL: frontend/src/app/now/liveContract.ts:LiveView lacks league header count | Read | live (stat feed) | in-season only |
| HM-003 | HM › Shell › NOW rail | Lineup lock or tag-window date | {phase,event_label,deadline_at} | PARTIAL: frontend/src/app/clock/useLeagueClock.ts:useLeagueClock lacks selected rail summary | Read | on-event | in-season / offseason |
| HM-004 | HM › Shell › Brief tab | Selects brief view | {tab:brief} | NEW | Write | on-event | all roles; UI state |
| HM-005 | HM › Shell › Depth charts tab | Selects depth view | {tab:depth} | NEW | Write | on-event | all roles; UI state |
| HM-006 | HM › Shell › NFL stats tab | Selects stats view | {tab:stats} | NEW | Write | on-event | all roles; UI state |
| HM-007 | HM › Shell › Rail destinations | Home, War Room, Franchise HQ, League Pulse, Control Room navigation; mock buttons inert | {route,allowed} | NEW | Read | static | role permissions unknown |
| HM-008 | HM › Shell › Rail shortcut | Lineup and roster navigation; mock inert | {route,label} | NEW | Read | per-load | owner; pin behavior unimplemented |
| HM-009 | HM › Shell › Command | Command-bar affordance; mock inert | {query,route} | NEW | Read | on-event | permissions unknown |
| HM-010 | HM › Shell › Palette | Handler supports slate, graphite, grey but no palette buttons exist in this mock | {palette} | NEW | Write | on-event | mock UI preference |
| HM-011 | HM › Shell › Mock phase | In-season/offseason selector changes demo data and default layout | {phase} | NEW | Write | on-event | demo only; custom layout stays |
| HM-012 | HM › Shell › Mock alerts | 4, 0, FAIL selectors set sample alert state | {state:some/none/fail} | NEW | Write | on-event | demo only |
| HM-013 | HM › Brief › NOW phase badge | LIVE or OFFSEASON | {phase,status} | NEW | Read | on-event | in-season / offseason |
| HM-014 | HM › Brief › NOW summary | Week, opponent, lead and score, lineup lock; or tag, draft, FA deadlines | {week,opponent_id,score,opponent_score,deadlines[]} | PARTIAL: frontend/src/app/clock/useLeagueClock.ts:useLeagueClock lacks matchup summary | Read | live (stat feed) | owner; phase dependent |
| HM-015 | HM › Brief › NOW record | Team win-loss record | {wins,losses} | PARTIAL: frontend/src/app/race/contract.ts:Race lacks explicit NOW summary | Read | on-event | in-season |
| HM-016 | HM › Brief › NOW seed | Current playoff seed | {seed:int} | PARTIAL: frontend/src/app/race/contract.ts:Race lacks explicit seed field | Read | on-event | in-season |
| HM-017 | HM › Brief › NOW power | Seasonal or dynasty power rank | {rank:int,horizon} | NEW | Read | on-event | phase changes horizon |
| HM-018 | HM › Brief › NOW cap | Current or next-season cap room | {season,room_cents} | PARTIAL: frontend/src/app/contracts/contract.ts:Cap lacks home summary | Read | on-event | owner; phase dependent |
| HM-019 | HM › Brief › NOW first pick | Next first-round draft slot | {season,round,slot} | NEW | Read | on-event | offseason |
| HM-020 | HM › Brief › NOW roster | Roster used and limit | {used:int,limit:int} | NEW | Read | on-event | offseason |
| HM-021 | HM › Brief › Alert band tooltip | Open all alerts | {title:string} | NEW | Read | static | hidden when no active alerts |
| HM-022 | HM › Brief › Alert band category | Most urgent active alert kind and urgency | {kind,urgency} | PARTIAL: frontend/src/app/data/parseAlerts.ts:parseAlerts lacks snooze state | Read | on-event | critical styling; active only |
| HM-023 | HM › Brief › Alert band title | Most urgent alert summary | {alert_id,title} | PARTIAL: frontend/src/app/data/parseAlerts.ts:parseAlerts lacks stable alert ID | Read | on-event | owner; active only |
| HM-024 | HM › Brief › Alert band countdown | Time to deadline | {deadline_at,remaining} | PARTIAL: frontend/src/app/data/parseAlerts.ts:parseAlerts lacks countdown deadline | Read | on-event | active only |
| HM-025 | HM › Brief › Alert band action | Open offer, set lineup, roster or IR; mock band click opens alerts drawer | {alert_id,route,label} | PARTIAL: frontend/src/app/data/parseAlerts.ts:parseAlerts lacks action label | Read | on-event | depends on alert kind |
| HM-026 | HM › Brief › Alert band more | Other active alert count and drawer toggle | {active_count} | PARTIAL: frontend/src/app/data/parseAlerts.ts:parseAlerts lacks snooze-aware count | Read/Write | on-event | hidden when none |
| HM-027 | HM › Brief › Alert failure band | Last failure timestamp, retry interval, not-all-clear warning, details | {failed_at,retry_seconds,error} | PARTIAL: frontend/src/app/data/parseAlerts.ts:parseAlerts lacks retry metadata | Read | on-event | fail only |
| HM-028 | HM › Brief › Ticker tooltip | Hover to pause | {paused:bool} | NEW | Read | static | hover; CSS behavior |
| HM-029 | HM › Brief › Ticker week | Top five week and live/final state | {week,status} | NEW | Read | live (stat feed) | week 9 live or week 18 final |
| HM-030 | HM › Brief › Ticker position | Position group QB RB WR TE DL LB DB K OL | {position} | NEW | Read | live (stat feed) | position specific |
| HM-031 | HM › Brief › Ticker rank | Top-five ordinal and leader highlight | {rank:int} | NEW | Read | live (stat feed) | top five per position |
| HM-032 | HM › Brief › Ticker player | Player name, NFL team; click opens player card | {player_id,name,nfl_team,position} | PARTIAL: frontend/src/app/research/contract.ts:Player lacks ticker ranking | Read | live (stat feed) | player or OL unit |
| HM-033 | HM › Brief › Ticker performance | Stat summary string and fantasy points | {stat_summary,points:decimal} | NEW | Read | live (stat feed) | league scoring; position-specific |
| HM-034 | HM › Brief › Layout indicator | Default in-season/offseason or custom mine | {mode,phase} | NEW | Read | on-event | per owner per league as claimed in mock |
| HM-035 | HM › Brief › Customize | Enter/exit edit mode; close drawers on exit | {editing:bool} | NEW | Write | on-event | brief only |
| HM-036 | HM › Brief › Tile title | Registered tile name | {tile_id,title} | NEW | Read | static | selected/edit states |
| HM-037 | HM › Brief › Tile source | Source label for each tile | {tile_id,source_label} | NEW | Read | static | per tile |
| HM-038 | HM › Brief › Tile link | HQ, LIVE, RACE, PULSE, LEDGER, POWER, PLANS, DRAFT, CAP, MARKET links; mock inert | {tile_id,route} | NEW | Read | static | no link for news and shortcuts |
| HM-039 | HM › Brief › Tile size buttons | 1×1, 2×1, 1×2; supported sizes enabled, current pressed | {tile_id,size,allowed_sizes[]} | NEW | Read/Write | on-event | edit only; grid may be full |
| HM-040 | HM › Brief › Tile drag tooltip | Drag to move; reorder by swapping if grid fits | {tile_id,grid_position,size} | NEW | Read/Write | on-event | edit only; invalid fit message |
| HM-041 | HM › Brief › Tile pin tooltip | Pin this screen to MY SHORTCUTS; star toggles | {owner_id,league_id,tile_id,pinned} | NEW | Read/Write | on-event | edit only; no persistence implemented |
| HM-042 | HM › Brief › Tile overflow tooltip | Move tile to overflow | {tile_id,grid,overflow[]} | NEW | Read/Write | on-event | edit only |
| HM-043 | HM › Brief › Tile footer freshness | As-of time and live tag | {tile_id,as_of,status} | NEW | Read | on-event | matchup live; mock timestamps fixed |
| HM-044 | HM › Brief › Tile footer clipped count | More rows available in source screen | {hidden_count,source_route} | NEW | Read | on-event | only if rows clipped |
| HM-045 | HM › Brief › Tile selection | Select tile to populate inspector | {selected_tile_id} | NEW | Write | on-event | read mode; edit may also select |
| HM-046 | HM › Decisions due › Category | Lineup, trades, bye, IR or clock badge | {kind} | PARTIAL: frontend/src/app/data/parseAlerts.ts:parseAlerts lacks clock events | Read | on-event | in-season/offseason |
| HM-047 | HM › Decisions due › Description | Specific pending action or league milestone | {id,label} | PARTIAL: frontend/src/app/clock/useLeagueClock.ts:useLeagueClock lacks alert aggregation | Read | on-event | phase-dependent |
| HM-048 | HM › Decisions due › Due date | Event date and time | {deadline_at} | PARTIAL: frontend/src/app/clock/useLeagueClock.ts:useLeagueClock lacks aggregated alerts | Read | on-event | event dependent |
| HM-049 | HM › Decisions due › Countdown | Remaining time, cold styling after 3 days | {deadline_at,remaining_seconds} | PARTIAL: frontend/src/app/clock/useLeagueClock.ts:useLeagueClock lacks alert aggregation | Read | on-event | long deadlines cold |
| HM-050 | HM › This week › Team names | Owner and opponent | {team_id,name,opponent_id,opponent_name} | NEW | Read | per-load | in-season matchup |
| HM-051 | HM › This week › Team records | Both records | {wins,losses} | PARTIAL: frontend/src/app/race/contract.ts:Race lacks matchup team pairing | Read | on-event | each side |
| HM-052 | HM › This week › Projected team totals | Both projected scores | {team_id,projected_points:decimal} | NEW | Read | live (stat feed) | each side |
| HM-053 | HM › This week › Actual team scores | Both current scores | {team_id,points:decimal} | NEW | Read | live (stat feed) | each side |
| HM-054 | HM › This week › Win probability | Both percentages and bar | {team_probability:decimal,opponent_probability:decimal} | NEW | Read | live (stat feed) | pre/live/final unknown |
| HM-055 | HM › This week › Player progress | Playing, yet to play and done counts | {playing:int,pending:int,done:int} | NEW | Read | live (stat feed) | roster state |
| HM-056 | HM › This week › Head-to-head player | Owner player name and position, click card | {player_id,name,position} | PARTIAL: frontend/src/app/research/contract.ts:Player lacks matchup pairing | Read | live (stat feed) | per roster slot |
| HM-057 | HM › This week › Owner player points | Per-player score | {player_id,points:decimal} | NEW | Read | live (stat feed) | per roster slot |
| HM-058 | HM › This week › Opponent player points | Opponent per-player score | {player_id,points:decimal} | NEW | Read | live (stat feed) | per roster slot |
| HM-059 | HM › This week › Opponent player name | Opponent player per roster slot | {player_id,name} | NEW | Read | live (stat feed) | mock name not clickable |
| HM-060 | HM › Race › Seed now | Current seed | {seed:int} | PARTIAL: frontend/src/app/race/contract.ts:Race lacks explicit seed | Read | on-event | playoff race |
| HM-061 | HM › Race › Projected seed | Forecast seed and bubble badge | {projected_seed:int,bubble:bool} | NEW | Read | on-event | bubble state |
| HM-062 | HM › Race › Playoff odds | Probability percentage | {probability:decimal} | NEW | Read | on-event | playoff eligibility |
| HM-063 | HM › Race › Power | Power rank | {rank:int} | NEW | Read | on-event | current owner |
| HM-064 | HM › Race › Run-in week | Future week per scheduled opponent | {week:int} | NEW | Read | on-event | remaining season |
| HM-065 | HM › Race › Run-in opponent | Opponent and home/away indicator | {opponent_id,name,away:bool} | NEW | Read | on-event | division leader highlighted |
| HM-066 | HM › Race › Run-in chance | Matchup win percentage colored by threshold | {win_probability:decimal} | NEW | Read | on-event | favorable/unfavorable |
| HM-067 | HM › Race › Division warning | Count of upcoming games against division leaders | {count:int,window:int} | NEW | Read | on-event | leader schedule |
| HM-068 | HM › Race › Mini standings rank | Rank per team | {team_id,rank:int} | PARTIAL: frontend/src/app/race/contract.ts:Race lacks explicit mini standings | Read | on-event | owner highlighted |
| HM-069 | HM › Race › Mini standings team | Team name | {team_id,name} | PARTIAL: frontend/src/app/race/contract.ts:Race lacks explicit mini standings | Read | on-event | owner highlighted |
| HM-070 | HM › Race › Mini standings record | Wins and losses | {wins:int,losses:int} | PARTIAL: frontend/src/app/race/contract.ts:Race lacks explicit mini standings | Read | on-event | per team |
| HM-071 | HM › Race › Playoff cut line | CUT marker | {cut_after_rank:int} | NEW | Read | on-event | cutoff row |
| HM-072 | HM › Headlines › Category | High, upset, trade, streak, record, clinch, low, injury, division | {category} | NEW | Read | on-event | category specific |
| HM-073 | HM › Headlines › Fact | Headline with team/player/score/streak/event | {id,text,entity_ids[]} | NEW | Read | on-event | per fact |
| HM-074 | HM › Headlines › Week | Week label beside each fact | {week:int} | NEW | Read | on-event | mock hardcodes week 8 |
| HM-075 | HM › News › All filter | Show all items | {filter:all} | PARTIAL: frontend/src/app/research/contract.ts:Research lacks Home feed filters | Read/Write | on-event | selected state |
| HM-076 | HM › News › My players filter | Filter to owner players | {filter:mine,owner_id} | PARTIAL: frontend/src/app/research/contract.ts:Research lacks Home feed filters | Read/Write | on-event | owner specific |
| HM-077 | HM › News › Injuries filter | Filter INJ, IR, SUSP | {filter:inj,categories[]} | PARTIAL: frontend/src/app/research/contract.ts:Research lacks Home feed filters | Read/Write | on-event | category subset |
| HM-078 | HM › News › On the wire filter | Filter unrostered players | {filter:wire} | PARTIAL: frontend/src/app/research/contract.ts:Research lacks Home feed filters | Read/Write | on-event | league ownership |
| HM-079 | HM › News › Filter result count | Number of matching items | {count:int} | NEW | Read | on-event | zero matches possible |
| HM-080 | HM › News › Item time | Publication time | {published_at} | PARTIAL: frontend/src/app/research/contract.ts:Article lacks publication timestamp | Read | live (stat feed) | per article |
| HM-081 | HM › News › Item category | INJ, DEPTH, TRADE, SIGN, IR, SUSP | {category} | PARTIAL: frontend/src/app/research/contract.ts:Article lacks category | Read | live (stat feed) | category colors |
| HM-082 | HM › News › Item headline | Headline opens news card | {article_id,headline} | PARTIAL: frontend/src/app/research/contract.ts:Article lacks Home feed card mapping | Read | live (stat feed) | owner/opponent/wire/other |
| HM-083 | HM › News › Item attribution | Reporter and source | {source,author} | PARTIAL: frontend/src/app/research/contract.ts:Article lacks source attribution | Read | live (stat feed) | per article |
| HM-084 | HM › News › Item ownership | Your FLEX, your position, opponent position, wire, or other franchise | {ownership,team_id,roster_slot} | PARTIAL: frontend/src/app/research/contract.ts:Player lacks matchup/slot classification | Read | on-event | owner/opponent/wire/other |
| HM-085 | HM › Transactions › Time | Transaction time or day | {occurred_at} | NEW | Read | on-event | league-scoped |
| HM-086 | HM › Transactions › Type | Trade, bid, wire, IR, taxi, FA, cut | {type} | NEW | Read | on-event | per kind |
| HM-087 | HM › Transactions › Headline | Teams and player opens transaction card | {transaction_id,summary} | NEW | Read | on-event | league-scoped |
| HM-088 | HM › Transactions › Detail | Assets, bid amount, roster movement, deadline or dead cap | {detail} | NEW | Read | on-event | varies by type |
| HM-089 | HM › Shortcuts › Lineup | Set lineup button and L key label; inert | {route,key} | NEW | Read | static | owner |
| HM-090 | HM › Shortcuts › Roster | Roster button and O key label; inert | {route,key} | NEW | Read | static | owner |
| HM-091 | HM › Shortcuts › Cap | Cap button and C key label; inert | {route,key} | NEW | Read | static | owner |
| HM-092 | HM › Shortcuts › Taxi IR | Taxi/IR button and I key label; inert | {route,key} | NEW | Read | static | owner |
| HM-093 | HM › Shortcuts › Trade Desk | Trade Desk button and T key label; inert | {route,key} | NEW | Read | static | owner |
| HM-094 | HM › Shortcuts › Market | Market button and M key label; inert | {route,key} | NEW | Read | static | owner |
| HM-095 | HM › Shortcuts › Draft Room | Draft Room button and D key label; inert | {route,key} | NEW | Read | static | owner |
| HM-096 | HM › Shortcuts › Plans | Plans button and P key label; inert | {route,key} | NEW | Read | static | owner |
| HM-097 | HM › Shortcuts › Live | Live button and G key label; inert | {route,key} | NEW | Read | static | role unknown |
| HM-098 | HM › Shortcuts › Race | Race button and R key label; inert | {route,key} | NEW | Read | static | role unknown |
| HM-099 | HM › Shortcuts › Archive | Archive button and A key label; inert | {route,key} | NEW | Read | static | role unknown |
| HM-100 | HM › Shortcuts › Rules | Rules button and U key label; inert | {route,key} | NEW | Read | static | role unknown |
| HM-101 | HM › Power › Model | Dynasty 3-year horizon and value/age/picks/cap/depth weights | {horizon_years,weights:{value,age,picks,cap,depth}} | NEW | Read | per-load | mock static model |
| HM-102 | HM › Power › Rank | Team ordinal | {team_id,rank:int} | NEW | Read | on-event | owner highlighted |
| HM-103 | HM › Power › Team | Ranked franchise name | {team_id,name} | NEW | Read | on-event | owner highlighted |
| HM-104 | HM › Power › Score | Power score and proportional bar | {score:decimal} | NEW | Read | on-event | team-specific |
| HM-105 | HM › Plans › Active plan | Plan name and checked count | {plan_id,name,completed_count,total_count} | NEW | Read | on-event | owner; mock count conflicts with list |
| HM-106 | HM › Plans › Checkbox | Per-item completion status; mock not interactive | {item_id,completed:bool} | NEW | Read | on-event | completed/unchecked |
| HM-107 | HM › Plans › Item | Plan task text | {item_id,text} | NEW | Read | on-event | owner |
| HM-108 | HM › Plans › Due | Plan task date | {item_id,due_at} | NEW | Read | on-event | owner |
| HM-109 | HM › Picks › Pick slot | Year, round, pick number | {pick_id,season,round,slot} | NEW | Read | on-event | owned/traded |
| HM-110 | HM › Picks › Pick provenance | Own/from team/to team and projection | {original_owner_id,from_team_id,to_team_id,projected_slot} | NEW | Read | on-event | owned/traded |
| HM-111 | HM › Picks › Pick value | Salary slot amount or traded/unknown | {slot_cents,status} | NEW | Read | on-event | future slot unavailable |
| HM-112 | HM › Cap › Room | Current season room and cap maximum | {season,room_cents,cap_cents} | PARTIAL: frontend/src/app/contracts/contract.ts:Cap lacks Home tile | Read | on-event | owner |
| HM-113 | HM › Cap › Bar committed | Committed percentage | {committed_cents,cap_cents} | PARTIAL: frontend/src/app/contracts/contract.ts:Cap lacks explicit committed breakdown | Read | on-event | current year |
| HM-114 | HM › Cap › Bar dead | Dead-cap amount and percentage | {dead_cents,cap_cents} | PARTIAL: frontend/src/app/contracts/contract.ts:Cap lacks dead-cap breakdown | Read | on-event | current year |
| HM-115 | HM › Cap › Bar room | Remaining percentage | {room_cents,cap_cents} | PARTIAL: frontend/src/app/contracts/contract.ts:Cap lacks Home bar | Read | on-event | current year |
| HM-116 | HM › Cap › Runway season | Four years | {season:int} | PARTIAL: frontend/src/app/contracts/contract.ts:Cap lacks room by future year | Read | on-event | future projections |
| HM-117 | HM › Cap › Runway room | Per-year room | {season,room_cents} | PARTIAL: frontend/src/app/contracts/contract.ts:Cap lacks future room | Read | on-event | projections |
| HM-118 | HM › Cap › Expirations | Number expiring or named RFA/UFA | {season,count,player_id,free_agency_type} | PARTIAL: frontend/src/app/contracts/contract.ts:Row lacks free agency type | Read | on-event | future seasons |
| HM-119 | HM › Bids › Player | Player name and position | {player_id,name,position} | PARTIAL: frontend/src/app/pool/contract.ts:Plan lacks Home bid list | Read | on-event | owner; private bids |
| HM-120 | HM › Bids › Amount | Bid salary | {amount_cents} | PARTIAL: frontend/src/app/pool/contract.ts:Plan lacks Home bid list | Read | on-event | owner |
| HM-121 | HM › Bids › Status | Placed high, outbid, draft not placed | {state,is_high} | PARTIAL: frontend/src/app/pool/contract.ts:Plan lacks high/outbid field | Read | on-event | privacy and sealed-bid behavior unknown |
| HM-122 | HM › Form › Abbreviation | Player initials derived from name | {player_id,abbreviation} | PARTIAL: frontend/src/app/research/contract.ts:Player lacks form | Read | on-event | starters |
| HM-123 | HM › Form › Player | Starter name | {player_id,name} | PARTIAL: frontend/src/app/research/contract.ts:Player lacks starter flag | Read | on-event | starters |
| HM-124 | HM › Form › Average | Five-game form number | {average:decimal} | NEW | Read | on-event | per player |
| HM-125 | HM › Form › Bars | Last five weekly points sparkline | {weekly_points:decimal[]} | NEW | Read | on-event | up to five games |
| HM-126 | HM › Alerts drawer › Pull count | Active alert count or ? if failed | {active_count,failed} | PARTIAL: frontend/src/app/data/parseAlerts.ts:parseAlerts lacks snooze-aware count | Read | on-event | none/fail |
| HM-127 | HM › Alerts drawer › Summary | Active and snoozed counts and urgency ordering | {active_count,snoozed_count} | PARTIAL: frontend/src/app/data/parseAlerts.ts:parseAlerts lacks snooze | Read | on-event | fail shows check failed |
| HM-128 | HM › Alerts drawer › Failure | Failure time and incomplete lineup/trade/IR/bye checks | {failed_at,failed_kinds[]} | PARTIAL: frontend/src/app/data/parseAlerts.ts:parseAlerts lacks check timestamp | Read | on-event | failed only |
| HM-129 | HM › Alerts drawer › Retry now | Retry check button; mock inert | {retry_requested:bool} | NEW | Write | on-event | failed only |
| HM-130 | HM › Alerts drawer › Alert kind | Kind and urgency color | {kind,urgency} | frontend/src/app/data/parseAlerts.ts:parseAlerts | Read | on-event | lineup/trade/bye/IR |
| HM-131 | HM › Alerts drawer › Alert countdown | Time remaining | {deadline_at,remaining_seconds} | PARTIAL: frontend/src/app/data/parseAlerts.ts:parseAlerts lacks deadline | Read | on-event | active/snoozed |
| HM-132 | HM › Alerts drawer › Alert title | Offer or lineup/bye/IR issue | {title} | frontend/src/app/data/parseAlerts.ts:parseAlerts | Read | on-event | per kind |
| HM-133 | HM › Alerts drawer › Alert detail | Expiration, recommendation and projection or eligibility | {detail} | frontend/src/app/data/parseAlerts.ts:parseAlerts | Read | on-event | per kind |
| HM-134 | HM › Alerts drawer › Action | Open offer/set lineup/roster/IR route; mock inert | {route,label} | PARTIAL: frontend/src/app/data/parseAlerts.ts:parseAlerts lacks action label | Read | on-event | active only |
| HM-135 | HM › Alerts drawer › Snooze 1h | Temporarily hide alert | {alert_id,snoozed_until} | NEW | Write | on-event | active only; mock fixed time |
| HM-136 | HM › Alerts drawer › Snooze until 8am | Temporarily hide alert | {alert_id,snoozed_until} | NEW | Write | on-event | active only |
| HM-137 | HM › Alerts drawer › Snoozed reason | Until time and early return if more urgent | {snoozed_until,urgency} | NEW | Read | on-event | snoozed only |
| HM-138 | HM › Alerts drawer › Unsnooze | Remove snooze | {alert_id,snoozed_until:null} | NEW | Write | on-event | snoozed only |
| HM-139 | HM › Alerts drawer › Snooze policy | Layout independent; cannot disable alert type | {policy} | NEW | Read | static | all roles unknown |
| HM-140 | HM › Layout drawer › Add tile pull | Opens tile registry | {editing,drawer_open} | NEW | Read/Write | on-event | edit only |
| HM-141 | HM › Layout drawer › Overflow pull | Opens overflow and displays count | {overflow_ids[]} | NEW | Read/Write | on-event | empty/nonempty |
| HM-142 | HM › Layout drawer › Add tile option | Tile title, supported sizes, source, on-grid/in-overflow badges; click adds | {tile_id,title,sizes[],source,location} | NEW | Read/Write | on-event | on-grid disabled |
| HM-143 | HM › Layout drawer › Overflow option | Title, source and RESTORE action | {tile_id,title,source} | NEW | Read/Write | on-event | overflow only |
| HM-144 | HM › Layout drawer › Overflow empty | Empty message explains bump behavior | {overflow_count:int} | NEW | Read | on-event | empty only |
| HM-145 | HM › Layout tray › Grid-full prompt | Requested tile and size; select victim | {tile_id,size,victim_id} | NEW | Read/Write | on-event | only when full |
| HM-146 | HM › Layout tray › Still no room | Rejected victim and retry prompt | {victim_id,error} | NEW | Read | on-event | failed fit |
| HM-147 | HM › Layout tray › Cancel | Cancels grid-full selection | {pending_pick:null} | NEW | Write | on-event | grid full |
| HM-148 | HM › Layout tray › Use phase default | Return to default phase layout; preserves overflow and pins | {mode:default,phase} | NEW | Write | on-event | edit only |
| HM-149 | HM › Layout tray › Reset default | Reset layout, overflow and pins | {mode:default,overflow:[],pins:{}} | NEW | Write | on-event | edit only |
| HM-150 | HM › Layout tray › Done | Leave customize mode | {editing:false} | NEW | Write | on-event | edit only |
| HM-151 | HM › Layout tray › Edit guidance | Drag, resize, overflow and no league commit message | {editing:bool} | NEW | Read | on-event | customize mode |
| HM-152 | HM › Layout tray › Depth provenance | Source TBD and ownership coloring explanation | {source_label} | NEW | Read | static | depth tab |
| HM-153 | HM › Layout tray › Stats provenance | nflverse, ESPN/Sleeper and league rules claim | {official_source,live_source,scoring_rule_id} | NEW | Read | static | stats tab; claim not verified |
| HM-154 | HM › Inspector › Franchise division | AFC North | {division_id,name} | NEW | Read | per-load | selected franchise |
| HM-155 | HM › Inspector › Franchise name | Owner's team name | {team_id,name} | PARTIAL: frontend/src/app/race/contract.ts:Race lacks inspector identity | Read | per-load | owner |
| HM-156 | HM › Inspector › Owner and tenure | Owner you, since year | {owner_id,joined_year} | NEW | Read | per-load | owner |
| HM-157 | HM › Inspector › Window | WIN NOW label | {team_id,window} | NEW | Read | on-event | owner |
| HM-158 | HM › Inspector › Record and seed | Wins-losses and seed | {wins,losses,seed} | PARTIAL: frontend/src/app/race/contract.ts:Race lacks inspector summary | Read | on-event | owner |
| HM-159 | HM › Inspector › Power | Season power rank | {rank,horizon} | NEW | Read | on-event | owner |
| HM-160 | HM › Inspector › Selected tile empty | Prompt to select tile | {selected_tile_id:null} | NEW | Read | on-event | no tile selected |
| HM-161 | HM › Inspector › Selected tile title | Current selected tile | {tile_id,title} | NEW | Read | on-event | selected only |
| HM-162 | HM › Inspector › ID | Registered tile identifier | {tile_id} | NEW | Read | static | selected only |
| HM-163 | HM › Inspector › Source | Human-readable source | {source_label} | NEW | Read | static | selected only |
| HM-164 | HM › Inspector › Provider | Claimed provider string | {provider_label} | NEW | Read | static | mock registry claims, not code citations |
| HM-165 | HM › Inspector › Refresh | Refresh policy | {refresh_policy} | NEW | Read | static | per tile |
| HM-166 | HM › Inspector › As of | Timestamp and live/fresh status | {as_of,state} | NEW | Read | on-event | mock fixed timestamp |
| HM-167 | HM › Inspector › Opens | Destination or dash | {route_or_null} | NEW | Read | static | per tile |
| HM-168 | HM › Inspector › Sizes | Supported tile spans | {sizes[]} | NEW | Read | static | per tile |
| HM-169 | HM › Inspector › States | Loading, empty, stale, unavailable, error labels | {supported_states[]} | NEW | Read | static | declared only; mock does not render these states |
| HM-170 | HM › Inspector › Layout scope | Saved per owner and league claim | {owner_id,league_id} | NEW | Read | static | mock has no persistence |
| HM-171 | HM › Inspector › Layout mode | Default follows phase/custom never moves automatically | {mode,phase} | NEW | Read | on-event | default/custom |
| HM-172 | HM › Inspector › Layout counts | Grid and overflow counts | {grid_count,overflow_count} | NEW | Read | on-event | owner |
| HM-173 | HM › Inspector › Alerts policy | Alerts always on top, outside layout | {layout_exempt:bool} | NEW | Read | static | all roles unknown |
| HM-174 | HM › Inspector › Customize status | CUSTOMIZE ON or read only | {editing:bool} | NEW | Read | on-event | edit/read |
| HM-175 | HM › Inspector › My drawer | Add personal drawer button; mock inert | {drawer_id} | NEW | Write | on-event | owner; behavior unspecified |
| HM-176 | HM › News card › Category | Article type badge | {category} | PARTIAL: frontend/src/app/research/contract.ts:Article lacks category | Read | live (stat feed) | article dependent |
| HM-177 | HM › News card › Time and source | Timestamp and attribution | {published_at,source} | PARTIAL: frontend/src/app/research/contract.ts:Article lacks timestamp and source | Read | live (stat feed) | article dependent |
| HM-178 | HM › News card › Headline | Article heading | {headline} | frontend/src/app/research/contract.ts:Article | Read | live (stat feed) | article dependent |
| HM-179 | HM › News card › Summary | Paragraphs of reporting and league context | {paragraphs:string[]} | PARTIAL: frontend/src/app/research/contract.ts:Article lacks summary paragraphs | Read | live (stat feed) | mock fallback for most articles |
| HM-180 | HM › News card › Player name | Related player | {player_id,name} | PARTIAL: frontend/src/app/research/contract.ts:Article lacks explicit player card mapping | Read | on-event | fallback may show dash |
| HM-181 | HM › News card › Player position team age | Player position, NFL team, age | {position,nfl_team,age} | PARTIAL: frontend/src/app/research/contract.ts:Player lacks age | Read | on-event | unknown values dash |
| HM-182 | HM › News card › League ownership | Franchise or unrostered | {franchise_id,ownership_label} | frontend/src/app/research/contract.ts:Player | Read | on-event | mine/opponent/wire |
| HM-183 | HM › News card › Player status | Injury/health state | {status,injury} | NEW | Read | on-event | unknown dash |
| HM-184 | HM › News card › For you | Roster slot or wire closing time | {roster_slot,wire_closes_at} | NEW | Read | on-event | owner/wire/unknown |
| HM-185 | HM › News card › Player action | Open player career card where headline regex matches; otherwise inert | {player_id} | PARTIAL: frontend/src/app/research/contract.ts:Player lacks Home navigation | Read | on-event | parsing fails for many headlines |
| HM-186 | HM › News card › Open source | Link full story to source; mock inert | {url} | PARTIAL: frontend/src/app/research/contract.ts:Article lacks source URL | Read | on-event | licensing/source unknown |
| HM-187 | HM › Trade card › Type/time | TRADE badge, time, league ledger | {transaction_id,type,occurred_at} | NEW | Read | on-event | trade only |
| HM-188 | HM › Trade card › Teams | Trade counterparties and send direction | {from_team_id,to_team_id,names[]} | NEW | Read | on-event | two-sided mock |
| HM-189 | HM › Trade card › Asset name | Player or draft pick | {asset_id,asset_type,name} | NEW | Read | on-event | pick/player |
| HM-190 | HM › Trade card › Asset metadata | Position, NFL team, age | {position,nfl_team,age} | NEW | Read | on-event | player; picks blank |
| HM-191 | HM › Trade card › Asset contract or slot | Salary and years, or pick projection | {salary_cents,contract_years,pick_slot} | NEW | Read | on-event | player/pick |
| HM-192 | HM › Trade card › Asset count | Number of assets sent per side | {count:int} | NEW | Read | on-event | each side |
| HM-193 | HM › Trade card › Cap out | Outgoing cap per side | {team_id,cap_out_cents} | NEW | Read | on-event | each side |
| HM-194 | HM › Trade card › Processing | Processed time, commissioner approval and no veto window | {processed_at,approval_status,veto_window} | NEW | Read | on-event | role/league rules |
| HM-195 | HM › Trade card › Both rosters | Open both rosters; mock inert | {team_ids[]} | NEW | Read | on-event | trade |
| HM-196 | HM › Trade card › Ledger | Open transaction ledger; mock inert | {transaction_id} | NEW | Read | on-event | trade |
| HM-197 | HM › Transaction card › Type and time | Badge and ledger time | {type,occurred_at} | NEW | Read | on-event | nontrade |
| HM-198 | HM › Transaction card › Heading | Transaction description | {summary} | NEW | Read | on-event | nontrade |
| HM-199 | HM › Transaction card › What | Meaning of wire, blind bid, FA, IR, taxi or release | {type,description} | NEW | Read | on-event | nontrade subtype |
| HM-200 | HM › Transaction card › Detail | Move-specific detail | {detail} | NEW | Read | on-event | nontrade |
| HM-201 | HM › Transaction card › Claimable by | Eligibility and bid close | {eligible_teams,closes_at} | NEW | Read | on-event | wire only |
| HM-202 | HM › Transaction card › Dead cap | Dead cap to dropping team | {dead_cap_cents} | NEW | Read | on-event | wire only |
| HM-203 | HM › Transaction card › Runner-up | Sealed runner-up until window ends | {runner_up_bid,visibility} | NEW | Read | on-event | bid only; sealed |
| HM-204 | HM › Transaction card › When | Transaction timestamp | {occurred_at} | NEW | Read | on-event | nontrade |
| HM-205 | HM › Transaction card › Market action | Open market; mock inert | {player_id,route} | NEW | Read | on-event | wire only |
| HM-206 | HM › Transaction card › Ledger action | Open ledger; mock inert | {transaction_id,route} | NEW | Read | on-event | nontrade |
| HM-207 | HM › Depth › Division group | Eight AFC/NFC divisions | {division_id,name} | NEW | Read | per-load | all NFL teams |
| HM-208 | HM › Depth › Team selector | Abbreviation and name; click changes chart | {team_id,abbreviation,name} | NEW | Read/Write | on-event | 32 teams; selected state |
| HM-209 | HM › Depth › Team heading | Selected NFL team name | {team_id,name} | NEW | Read | on-event | selected team |
| HM-210 | HM › Depth › Updated timestamp | Depth source and last updated time | {source,updated_at} | NEW | Read | on-event | source TBD |
| HM-211 | HM › Depth › Ownership legend | Yours, rostered, on wire color key | {owner_class} | PARTIAL: frontend/src/app/research/contract.ts:Player lacks owner class | Read | on-event | league owner relative |
| HM-212 | HM › Depth › Injury legend | Q injury badge | {injury_designation} | NEW | Read | on-event | questionable |
| HM-213 | HM › Depth › Rookie legend | R rookie badge | {is_rookie:bool} | PARTIAL: frontend/src/app/cards/playerModel.ts:playerCardModel lacks depth chart | Read | on-event | rookie |
| HM-214 | HM › Depth › Draft code legend | Draft year/round, U UDFA, SF/CF signed | {draft_year,round,acquisition_code} | NEW | Read | per-load | player specific |
| HM-215 | HM › Depth › Offense defense ST tab | Select combined sections | {section:od} | NEW | Write | on-event | selected state |
| HM-216 | HM › Depth › Special teams tab | Select special teams | {section:st} | NEW | Write | on-event | selected state |
| HM-217 | HM › Depth › Practice squad tab | Select practice squad | {section:ps} | NEW | Write | on-event | selected state |
| HM-218 | HM › Depth › Reserves tab | Select reserves and coaching staff | {section:res} | NEW | Write | on-event | selected state |
| HM-219 | HM › Depth › Offense scheme | Personnel formation and usage percentage | {formation,usage_percent} | NEW | Read | on-event | offense section |
| HM-220 | HM › Depth › Defense scheme | Base formation | {formation} | NEW | Read | on-event | defense section |
| HM-221 | HM › Depth › Reserve category | IR, NFI, designated for return | {reserve_type} | NEW | Read | on-event | reserves |
| HM-222 | HM › Depth › Coaching staff | Staff names and roles text | {staff:list<{name,role}>} | NEW | Read | on-event | reserves |
| HM-223 | HM › Depth › Position | Position or depth role per row | {position} | NEW | Read | on-event | section-specific |
| HM-224 | HM › Depth › Jersey number | Number at each depth slot | {player_id,jersey_number} | NEW | Read | on-event | blank slots |
| HM-225 | HM › Depth › Player name | Player name clickable to card, ordered by depth | {player_id,name,depth_order} | PARTIAL: frontend/src/app/research/contract.ts:Player lacks ordered depth | Read | on-event | rostered/wire, blank slots |
| HM-226 | HM › Depth › Owner tooltip | Fantasy owner or Wire shown on hover | {franchise_id,owner_label} | frontend/src/app/research/contract.ts:Player | Read | on-event | owner/opponent/wire |
| HM-227 | HM › Depth › Draft/signed code | Draft year and round or signing code beside player | {draft_year,round,signing_code} | NEW | Read | on-event | per player |
| HM-228 | HM › Depth › Injury badge | Q indicator | {player_id,injury_designation} | NEW | Read | on-event | questionable |
| HM-229 | HM › Depth › Rookie badge | R indicator | {player_id,is_rookie:bool} | PARTIAL: frontend/src/app/cards/playerModel.ts:playerCardModel lacks depth order | Read | on-event | rookie |
| HM-230 | HM › Stats › Saved view Wire RBs | RB, wire, weeks 5–8 | {view_id,position,owner_filter,week_start,week_end} | NEW | Read/Write | on-event | saved preset |
| HM-231 | HM › Stats › Saved view My starters | QB, mine, weeks 1–8 | {view_id,position,owner_filter,week_start,week_end} | NEW | Read/Write | on-event | mock label says starters but filter only mine |
| HM-232 | HM › Stats › Saved view IDP sacks | DL, all, weeks 1–8 | {view_id,position,owner_filter,week_start,week_end} | NEW | Read/Write | on-event | mock resets sort to fantasy points |
| HM-233 | HM › Stats › Saved view Target hogs | WR, all, weeks 1–8 | {view_id,position,owner_filter,week_start,week_end} | NEW | Read/Write | on-event | mock resets sort to fantasy points |
| HM-234 | HM › Stats › Saved view Rostered TEs | TE, rostered, weeks 6–8 | {view_id,position,owner_filter,week_start,week_end} | NEW | Read/Write | on-event | mock rostered excludes mine |
| HM-235 | HM › Stats › Save current | Save current as view; mock inert | {name,filters,sort} | NEW | Write | on-event | owner |
| HM-236 | HM › Stats › Position selector | QB RB WR TE DL LB DB K | {position} | NEW | Read/Write | on-event | changes columns; resets sort |
| HM-237 | HM › Stats › Week start selector | Weeks 1–9 | {week_start:int} | NEW | Read/Write | on-event | mock select has no handler |
| HM-238 | HM › Stats › Week end selector | Weeks 1–9 | {week_end:int} | NEW | Read/Write | on-event | mock select has no handler |
| HM-239 | HM › Stats › Season | 2026 label | {season:int} | NEW | Read | per-load | mock fixed |
| HM-240 | HM › Stats › League ownership all | All players filter | {owner_filter:all} | NEW | Read/Write | on-event | active state |
| HM-241 | HM › Stats › League ownership mine | My players filter | {owner_filter:me} | NEW | Read/Write | on-event | owner scoped |
| HM-242 | HM › Stats › League ownership rostered | Other teams' rostered filter | {owner_filter:op} | NEW | Read/Write | on-event | excludes mine in mock |
| HM-243 | HM › Stats › League ownership wire | Unrostered filter | {owner_filter:fa} | NEW | Read/Write | on-event | league ownership |
| HM-244 | HM › Stats › NFL team selector | All 32 teams and individual team options | {team_id_or_all} | NEW | Read/Write | on-event | mock select has no handler |
| HM-245 | HM › Stats › Minimum GP | GP ≥ 3 chip | {min_games:3} | NEW | Read/Write | on-event | mock inert |
| HM-246 | HM › Stats › Minimum FFB | FFB ≥ 10/game chip | {min_ffb_per_game:10} | NEW | Read/Write | on-event | mock inert |
| HM-247 | HM › Stats › Minimum snaps | Snaps ≥ 50% chip | {min_snap_percent:50} | NEW | Read/Write | on-event | mock inert |
| HM-248 | HM › Stats › Heading | Position and week range | {position,week_start,week_end} | NEW | Read | on-event | filter state |
| HM-249 | HM › Stats › Summary | Player count, sort key, as-of timestamp | {count,sort_key,as_of} | NEW | Read | on-event | filter state; mock timestamp fixed |
| HM-250 | HM › Stats › Columns action | Choose visible columns; mock inert | {visible_columns[]} | NEW | Read/Write | on-event | position-specific |
| HM-251 | HM › Stats › Export CSV | Export filtered table; mock inert | {filters,columns,rows[]} | NEW | Read | on-event | permissions unknown |
| HM-252 | HM › Stats › Rank | Sorted row number | {rank:int} | NEW | Read | on-event | filtered result |
| HM-253 | HM › Stats › Player | Player name and NFL team; click opens card | {player_id,name,nfl_team,position} | PARTIAL: frontend/src/app/research/contract.ts:Player lacks stats-table routing | Read | on-event | per result |
| HM-254 | HM › Stats › In league | YOU, franchise name, or WIRE | {franchise_id,owner_label} | frontend/src/app/research/contract.ts:Player | Read | on-event | mine/other/wire |
| HM-255 | HM › Stats › Column sorting | Click stat header to sort descending | {sort_key,direction:desc} | NEW | Read/Write | on-event | mock no ascending toggle |
| HM-256 | HM › Stats › Fantasy points | League-scored total | {points:decimal} | NEW | Read | live (stat feed) | position and league scoring |
| HM-257 | HM › Stats › Fantasy points per game | Total divided by games | {points_per_game:decimal} | NEW | Read | live (stat feed) | position and league scoring |
| HM-258 | HM › Player card › Position | Position badge | {position} | frontend/src/app/research/contract.ts:Player | Read | on-event | mock maps OL to RB |
| HM-259 | HM › Player card › NFL team | Team abbreviation/name | {nfl_team_id,name} | frontend/src/app/research/contract.ts:Player | Read | on-event | may be missing |
| HM-260 | HM › Player card › Jersey | Player jersey number in header and avatar | {jersey_number} | NEW | Read | on-event | mock generated |
| HM-261 | HM › Player card › Full name | Full uppercase name | {player_id,name} | frontend/src/app/research/contract.ts:Player | Read | on-event | player selected |
| HM-262 | HM › Player card › Height | Player height | {height} | NEW | Read | per-load | mock generated except two players |
| HM-263 | HM › Player card › Weight | Weight in pounds | {weight_lbs:int} | NEW | Read | per-load | mock generated |
| HM-264 | HM › Player card › Born | Date of birth and age | {birthdate,age} | PARTIAL: frontend/src/app/cards/playerModel.ts:playerCardModel lacks displayed birthdate | Read | per-load | player selected |
| HM-265 | HM › Player card › College | College name | {college} | NEW | Read | per-load | player selected |
| HM-266 | HM › Player card › NFL draft | Year, round/pick, drafting team, experience | {year,round,pick,team,experience_years} | NEW | Read | per-load | not all fields populated in mock |
| HM-267 | HM › Player card › Fantasy team | Franchise or wire ownership | {franchise_id,owner_label} | frontend/src/app/research/contract.ts:Player | Read | on-event | mine/other/wire |
| HM-268 | HM › Player card › Salary | Current salary | {salary_cents} | PARTIAL: frontend/src/app/contracts/contract.ts:Row lacks card composition | Read | on-event | rostered/wire ambiguity |
| HM-269 | HM › Player card › Contract year | Contract expiry year | {contract_end_year} | PARTIAL: frontend/src/app/contracts/contract.ts:Row lacks card composition | Read | on-event | owner/other/wire |
| HM-270 | HM › Player card › Contract notes | Extension, years, restructure details | {contract_notes} | PARTIAL: frontend/src/app/contracts/contract.ts:Row lacks full notes semantics | Read | on-event | owner/other/wire |
| HM-271 | HM › Player card › Acquired | Acquisition method, date and prior team | {method,at,from_team_id} | NEW | Read | on-event | owner/other/wire |
| HM-272 | HM › Player card › Player news tab | Select news | {tab:news} | NEW | Write | on-event | card |
| HM-273 | HM › Player card › 2026 stats tab | Select season game log | {tab:stats,season} | NEW | Write | on-event | default |
| HM-274 | HM › Player card › Projections tab | Select projections | {tab:proj} | NEW | Write | on-event | card |
| HM-275 | HM › Player card › Career stats tab | Select career | {tab:career} | NEW | Write | on-event | card |
| HM-276 | HM › Player card › Full profile | Link full profile; mock inert | {player_id,route} | NEW | Read | on-event | player |
| HM-277 | HM › Player card › Propose trade | Link trade proposal; mock inert | {player_id,franchise_id,route} | NEW | Read | on-event | shown when not mine, including wire in mock |
| HM-278 | HM › Player card › Watchlist | Add watchlist; mock inert | {owner_id,player_id} | NEW | Write | on-event | all ownership classes |
| HM-279 | HM › Player card › Transaction history | Link player transactions; mock inert | {player_id,route} | NEW | Read | on-event | player |
| HM-280 | HM › Player card news › Headline | News item headline | {article_id,headline} | frontend/src/app/research/contract.ts:Article | Read | on-event | player |
| HM-281 | HM › Player card news › Age | Relative publication time | {published_at} | PARTIAL: frontend/src/app/research/contract.ts:Article lacks timestamp | Read | on-event | player |
| HM-282 | HM › Player card news › Summary | News summary text | {summary} | PARTIAL: frontend/src/app/research/contract.ts:Article lacks summary | Read | on-event | player |
| HM-283 | HM › Player card stats › Week | Game-log week | {week:int} | NEW | Read | live (stat feed) | bye row |
| HM-284 | HM › Player card stats › Opponent | NFL opponent and home/away | {opponent_id,away:bool} | NEW | Read | live (stat feed) | bye has no opponent |
| HM-285 | HM › Player card stats › Result | Win/loss and NFL score | {result,team_score,opponent_score} | NEW | Read | live (stat feed) | completed game |
| HM-286 | HM › Player card stats › Stat cells | Per-position game stat columns defined in stats table | {week,position,stats:map<string,number>} | NEW | Read | live (stat feed) | bye or missing |
| HM-287 | HM › Player card stats › FFB points | Per-week fantasy points | {week,points:decimal} | NEW | Read | live (stat feed) | league scoring |
| HM-288 | HM › Player card stats › Bye | BYE row | {week,is_bye:bool} | NEW | Read | on-event | bye only |
| HM-289 | HM › Player card stats › Season total | Game count and total fantasy points | {games:int,total_points:decimal} | NEW | Read | live (stat feed) | season |
| HM-290 | HM › Player card projections › Week | Future week | {week:int} | NEW | Read | on-event | future weeks 10–17 in mock |
| HM-291 | HM › Player card projections › Opponent | Future opponent and home/away | {opponent_id,away:bool} | NEW | Read | on-event | future |
| HM-292 | HM › Player card projections › Rank vs position | Opponent defensive rank | {opponent_rank:int,position} | NEW | Read | on-event | position |
| HM-293 | HM › Player card projections › Projected points | Fantasy projection | {points:decimal} | NEW | Read | on-event | league scoring |
| HM-294 | HM › Player card projections › Range | Low and high projected points | {low:decimal,high:decimal} | NEW | Read | on-event | uncertainty model unspecified |
| HM-295 | HM › Player card career › Year | Season | {season:int} | NEW | Read | per-load | since draft |
| HM-296 | HM › Player card career › NFL team | Team for season | {team_id} | NEW | Read | per-load | season |
| HM-297 | HM › Player card career › Fantasy team | Franchise for season | {franchise_id} | NEW | Read | per-load | historical ownership |
| HM-298 | HM › Player card career › Stat cells | Position-specific season statistics | {season,stats:map<string,number>} | NEW | Read | on-event | position-specific |
| HM-299 | HM › Player card career › Fantasy points | Season league points | {season,points:decimal} | NEW | Read | on-event | historical league rules unknown |
| HM-300 | HM › Player card career › Fantasy points/game | Season per-game average | {season,points_per_game:decimal} | NEW | Read | on-event | historical league rules unknown |
| HM-301 | HM › Player card career › Career total | Seasons and career fantasy total | {seasons:int,points:decimal} | NEW | Read | on-event | player |
| HM-302 | HM › Stats metric › Games played | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | QB RB WR TE DL LB DB K; missing displays dash |
| HM-303 | HM › Stats metric › Completions and attempts | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | QB; missing displays dash |
| HM-304 | HM › Stats metric › Passing yards | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | QB; missing displays dash |
| HM-305 | HM › Stats metric › Passing touchdowns | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | QB; missing displays dash |
| HM-306 | HM › Stats metric › Interceptions thrown | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | QB; missing displays dash |
| HM-307 | HM › Stats metric › Rushing yards | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | QB RB WR; missing displays dash |
| HM-308 | HM › Stats metric › Rushing touchdowns | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | QB RB; missing displays dash |
| HM-309 | HM › Stats metric › Rushing attempts | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | RB; missing displays dash |
| HM-310 | HM › Stats metric › Yards per carry | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | RB; missing displays dash |
| HM-311 | HM › Stats metric › Targets | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | RB WR TE; missing displays dash |
| HM-312 | HM › Stats metric › Receptions | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | RB WR TE; missing displays dash |
| HM-313 | HM › Stats metric › Receiving yards | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | RB WR TE; missing displays dash |
| HM-314 | HM › Stats metric › Yards per reception | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | WR TE; missing displays dash |
| HM-315 | HM › Stats metric › Receiving touchdowns | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | RB WR TE; missing displays dash |
| HM-316 | HM › Stats metric › Tackles | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | DL LB DB; missing displays dash |
| HM-317 | HM › Stats metric › Assists | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | DL LB DB; missing displays dash |
| HM-318 | HM › Stats metric › Sacks | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | DL LB; missing displays dash |
| HM-319 | HM › Stats metric › Tackles for loss | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | DL LB; missing displays dash |
| HM-320 | HM › Stats metric › Quarterback hits | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | DL; missing displays dash |
| HM-321 | HM › Stats metric › Forced fumbles | Position-specific sortable statistical cell; mock binds FF to fantasy-point field | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | DL DB; missing displays dash |
| HM-322 | HM › Stats metric › Interceptions defensive | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | LB DB; missing displays dash |
| HM-323 | HM › Stats metric › Passes defended | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | LB DB; missing displays dash |
| HM-324 | HM › Stats metric › Field goals made and attempted | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | K; missing displays dash |
| HM-325 | HM › Stats metric › Longest field goal | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | K; missing displays dash |
| HM-326 | HM › Stats metric › Extra points | Position-specific sortable statistical cell | {player_id,week_range,value:decimal} | NEW | Read | live (stat feed) | K; missing displays dash |
| HM-327 | HM › Shell › Rail customize | Rail CUSTOMIZE button; no matching handler in mock | {editing:bool} | NEW | Write | on-event | inert; distinct from working workspace customize |
| HM-328 | HM › Brief › Read-only guidance | Home reads and links only; nothing here changes league | {screen_mode:read_only} | NEW | Read | static | brief outside customize |
| HM-329 | HM › Layout tray › Swap rejection | Sizes do not fit swapped; resize one first | {tile_ids[],error} | NEW | Read | on-event | invalid drag/drop |
| HM-330 | HM › Shared overlay › Close card | ESC, close button or scrim dismisses news, transaction or player card | {card_open:bool} | NEW | Write | on-event | card open |
| HM-331 | HM › Shared drawer › Close drawer | ESC or drawer close button closes alert/layout drawers | {drawer_open:bool} | NEW | Write | on-event | drawer open; ESC cancels pending tile choice first |
| HM-332 | HM › Depth › Depth slot header | Player 1–4 with NO columns; reserves up to 5 and special teams up to 3 | {section,slot_index,jersey_number} | NEW | Read | on-event | varies by section |

| HM-333 | HM › Stats › NFL team option BUF | BUF · Bills | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-334 | HM › Stats › NFL team option MIA | MIA · Dolphins | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-335 | HM › Stats › NFL team option NE | NE · Patriots | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-336 | HM › Stats › NFL team option NYJ | NYJ · Jets | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-337 | HM › Stats › NFL team option BAL | BAL · Ravens | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-338 | HM › Stats › NFL team option CIN | CIN · Bengals | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-339 | HM › Stats › NFL team option CLE | CLE · Browns | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-340 | HM › Stats › NFL team option PIT | PIT · Steelers | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-341 | HM › Stats › NFL team option HOU | HOU · Texans | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-342 | HM › Stats › NFL team option IND | IND · Colts | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-343 | HM › Stats › NFL team option JAX | JAX · Jaguars | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-344 | HM › Stats › NFL team option TEN | TEN · Titans | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-345 | HM › Stats › NFL team option DEN | DEN · Broncos | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-346 | HM › Stats › NFL team option KC | KC · Chiefs | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-347 | HM › Stats › NFL team option LV | LV · Raiders | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-348 | HM › Stats › NFL team option LAC | LAC · Chargers | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-349 | HM › Stats › NFL team option DAL | DAL · Cowboys | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-350 | HM › Stats › NFL team option NYG | NYG · Giants | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-351 | HM › Stats › NFL team option PHI | PHI · Eagles | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-352 | HM › Stats › NFL team option WAS | WAS · Commanders | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-353 | HM › Stats › NFL team option CHI | CHI · Bears | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-354 | HM › Stats › NFL team option DET | DET · Lions | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-355 | HM › Stats › NFL team option GB | GB · Packers | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-356 | HM › Stats › NFL team option MIN | MIN · Vikings | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-357 | HM › Stats › NFL team option ATL | ATL · Falcons | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-358 | HM › Stats › NFL team option CAR | CAR · Panthers | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-359 | HM › Stats › NFL team option NO | NO · Saints | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-360 | HM › Stats › NFL team option TB | TB · Buccaneers | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-361 | HM › Stats › NFL team option ARI | ARI · Cardinals | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-362 | HM › Stats › NFL team option LAR | LAR · Rams | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-363 | HM › Stats › NFL team option SF | SF · 49ers | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-364 | HM › Stats › NFL team option SEA | SEA · Seahawks | {team_id,abbreviation,name} | NEW | Read/Write | on-event | select has no handler in mock; same team selectable in depth |
| HM-365 | HM › Stats › NFL team option all | All 32 teams | {team_filter:all} | NEW | Read/Write | on-event | select has no handler in mock |
| HM-366 | HM › Stats › Week option 1 | Week 1 in both start and end selectors | {week:1} | NEW | Read/Write | on-event | select has no handler in mock |
| HM-367 | HM › Stats › Week option 2 | Week 2 in both start and end selectors | {week:2} | NEW | Read/Write | on-event | select has no handler in mock |
| HM-368 | HM › Stats › Week option 3 | Week 3 in both start and end selectors | {week:3} | NEW | Read/Write | on-event | select has no handler in mock |
| HM-369 | HM › Stats › Week option 4 | Week 4 in both start and end selectors | {week:4} | NEW | Read/Write | on-event | select has no handler in mock |
| HM-370 | HM › Stats › Week option 5 | Week 5 in both start and end selectors | {week:5} | NEW | Read/Write | on-event | select has no handler in mock |
| HM-371 | HM › Stats › Week option 6 | Week 6 in both start and end selectors | {week:6} | NEW | Read/Write | on-event | select has no handler in mock |
| HM-372 | HM › Stats › Week option 7 | Week 7 in both start and end selectors | {week:7} | NEW | Read/Write | on-event | select has no handler in mock |
| HM-373 | HM › Stats › Week option 8 | Week 8 in both start and end selectors | {week:8} | NEW | Read/Write | on-event | select has no handler in mock |
| HM-374 | HM › Stats › Week option 9 | Week 9 in both start and end selectors | {week:9} | NEW | Read/Write | on-event | select has no handler in mock |
| HM-375 | HM › Stats › Position option QB | Choose QB and reset sort | {position:QB} | NEW | Read/Write | on-event | selected or unselected |
| HM-376 | HM › Stats › Position option RB | Choose RB and reset sort | {position:RB} | NEW | Read/Write | on-event | selected or unselected |
| HM-377 | HM › Stats › Position option WR | Choose WR and reset sort | {position:WR} | NEW | Read/Write | on-event | selected or unselected |
| HM-378 | HM › Stats › Position option TE | Choose TE and reset sort | {position:TE} | NEW | Read/Write | on-event | selected or unselected |
| HM-379 | HM › Stats › Position option DL | Choose DL and reset sort | {position:DL} | NEW | Read/Write | on-event | selected or unselected |
| HM-380 | HM › Stats › Position option LB | Choose LB and reset sort | {position:LB} | NEW | Read/Write | on-event | selected or unselected |
| HM-381 | HM › Stats › Position option DB | Choose DB and reset sort | {position:DB} | NEW | Read/Write | on-event | selected or unselected |
| HM-382 | HM › Stats › Position option K | Choose K and reset sort | {position:K} | NEW | Read/Write | on-event | selected or unselected |

## Shared entities

IDs reflect the mock elements that reference each entity, including related controls.

- **League**: HM-001, HM-003, HM-007, HM-011, HM-013, HM-034, HM-041, HM-047, HM-148, HM-151, HM-153, HM-170, HM-171, HM-179, HM-182, HM-187, HM-240, HM-241, HM-242, HM-243, HM-254, HM-256, HM-299, HM-328
- **Fantasy team**: HM-007, HM-014, HM-015, HM-032, HM-041, HM-050, HM-051, HM-052, HM-053, HM-054, HM-056, HM-057, HM-058, HM-059, HM-064, HM-065, HM-068, HM-069, HM-073, HM-076, HM-084, HM-087, HM-102, HM-103, HM-110, HM-152, HM-154, HM-155, HM-156, HM-157, HM-170, HM-181, HM-182, HM-188, HM-190, HM-193, HM-195, HM-201, HM-202, HM-208, HM-209, HM-211, HM-216, HM-226, HM-230, HM-231, HM-232, HM-233, HM-234, HM-240, HM-241, HM-242, HM-243, HM-244, HM-253, HM-254, HM-259, HM-266, HM-267, HM-271, HM-277, HM-278, HM-284, HM-285, HM-291, HM-292, HM-296, HM-297, HM-332
- **NFL team**: HM-333, HM-334, HM-335, HM-336, HM-337, HM-338, HM-339, HM-340, HM-341, HM-342, HM-343, HM-344, HM-345, HM-346, HM-347, HM-348, HM-349, HM-350, HM-351, HM-352, HM-353, HM-354, HM-355, HM-356, HM-357, HM-358, HM-359, HM-360, HM-361, HM-362, HM-363, HM-364, HM-014, HM-032, HM-050, HM-054, HM-058, HM-059, HM-064, HM-065, HM-084, HM-181, HM-190, HM-208, HM-209, HM-244, HM-253, HM-259, HM-284, HM-285, HM-291, HM-292, HM-296
- **Player**: HM-028, HM-029, HM-030, HM-031, HM-032, HM-033, HM-055, HM-056, HM-057, HM-058, HM-059, HM-073, HM-076, HM-078, HM-087, HM-118, HM-119, HM-122, HM-123, HM-124, HM-125, HM-180, HM-181, HM-183, HM-185, HM-189, HM-205, HM-219, HM-220, HM-224, HM-225, HM-227, HM-228, HM-229, HM-240, HM-241, HM-249, HM-253, HM-258, HM-259, HM-260, HM-261, HM-262, HM-263, HM-264, HM-265, HM-266, HM-267, HM-268, HM-269, HM-270, HM-271, HM-272, HM-273, HM-274, HM-275, HM-276, HM-277, HM-278, HM-279, HM-280, HM-281, HM-282, HM-283, HM-284, HM-285, HM-286, HM-287, HM-288, HM-289, HM-290, HM-291, HM-292, HM-293, HM-294, HM-295, HM-296, HM-297, HM-298, HM-299, HM-300, HM-301, HM-302, HM-303, HM-304, HM-305, HM-306, HM-307, HM-308, HM-309, HM-310, HM-311, HM-312, HM-313, HM-314, HM-315, HM-316, HM-317, HM-318, HM-319, HM-320, HM-321, HM-322, HM-323, HM-324, HM-325, HM-326, HM-330, HM-332
- **Contract and cap**: HM-018, HM-038, HM-088, HM-091, HM-101, HM-111, HM-112, HM-113, HM-114, HM-115, HM-116, HM-117, HM-118, HM-120, HM-191, HM-193, HM-202, HM-268, HM-269, HM-270
- **Draft pick**: HM-014, HM-019, HM-038, HM-095, HM-101, HM-109, HM-110, HM-111, HM-121, HM-147, HM-189, HM-191, HM-214, HM-227, HM-266
- **Bid**: HM-038, HM-086, HM-088, HM-094, HM-119, HM-120, HM-121, HM-199, HM-201, HM-203, HM-205
- **Transaction and trade**: HM-046, HM-072, HM-078, HM-081, HM-084, HM-085, HM-086, HM-087, HM-088, HM-093, HM-111, HM-128, HM-184, HM-187, HM-188, HM-189, HM-190, HM-191, HM-192, HM-193, HM-194, HM-195, HM-196, HM-197, HM-198, HM-199, HM-200, HM-201, HM-202, HM-203, HM-204, HM-205, HM-206, HM-211, HM-226, HM-230, HM-243, HM-254, HM-267, HM-277, HM-279, HM-330
- **Alert**: HM-012, HM-021, HM-022, HM-023, HM-024, HM-025, HM-026, HM-027, HM-126, HM-127, HM-128, HM-129, HM-130, HM-131, HM-132, HM-133, HM-134, HM-135, HM-136, HM-137, HM-138, HM-139, HM-173, HM-331
- **Clock event**: HM-003, HM-014, HM-024, HM-046, HM-047, HM-048, HM-049, HM-067, HM-088, HM-108, HM-131, HM-157, HM-194, HM-203
- **News article**: HM-072, HM-073, HM-074, HM-075, HM-076, HM-077, HM-078, HM-079, HM-080, HM-081, HM-082, HM-083, HM-084, HM-087, HM-176, HM-177, HM-178, HM-179, HM-180, HM-181, HM-182, HM-183, HM-184, HM-185, HM-186, HM-272, HM-280, HM-281, HM-282, HM-330
- **League rule / scoring**: HM-100, HM-153
- **Matchup**: HM-038, HM-050, HM-051, HM-052, HM-053, HM-054, HM-055, HM-056, HM-057, HM-058, HM-059, HM-060, HM-061, HM-062, HM-063, HM-064, HM-065, HM-066, HM-067, HM-068, HM-069, HM-070, HM-071, HM-098
- **Standing / power**: HM-016, HM-017, HM-038, HM-060, HM-061, HM-062, HM-063, HM-064, HM-065, HM-066, HM-067, HM-068, HM-069, HM-070, HM-071, HM-098, HM-101, HM-102, HM-103, HM-104, HM-158, HM-159
- **Plan / task**: HM-038, HM-096, HM-105, HM-106, HM-107, HM-108
- **Home layout / tile / shortcut**: HM-008, HM-011, HM-034, HM-036, HM-037, HM-038, HM-039, HM-040, HM-041, HM-042, HM-043, HM-044, HM-045, HM-089, HM-090, HM-091, HM-092, HM-093, HM-094, HM-095, HM-096, HM-097, HM-098, HM-099, HM-100, HM-139, HM-140, HM-141, HM-142, HM-143, HM-144, HM-145, HM-146, HM-147, HM-148, HM-149, HM-150, HM-151, HM-152, HM-153, HM-154, HM-155, HM-156, HM-157, HM-158, HM-159, HM-160, HM-161, HM-162, HM-163, HM-164, HM-165, HM-166, HM-167, HM-168, HM-169, HM-170, HM-171, HM-172, HM-173, HM-174, HM-175, HM-329, HM-331
- **NFL statistical observation**: HM-006, HM-028, HM-029, HM-030, HM-031, HM-032, HM-033, HM-110, HM-122, HM-123, HM-124, HM-125, HM-133, HM-153, HM-185, HM-191, HM-219, HM-220, HM-230, HM-231, HM-232, HM-233, HM-234, HM-235, HM-236, HM-237, HM-238, HM-239, HM-240, HM-241, HM-242, HM-243, HM-244, HM-245, HM-246, HM-247, HM-248, HM-249, HM-250, HM-251, HM-252, HM-253, HM-254, HM-255, HM-256, HM-257, HM-273, HM-274, HM-275, HM-283, HM-284, HM-285, HM-286, HM-287, HM-288, HM-289, HM-290, HM-291, HM-292, HM-293, HM-294, HM-295, HM-296, HM-297, HM-298, HM-299, HM-300, HM-301, HM-302, HM-303, HM-304, HM-305, HM-306, HM-307, HM-308, HM-309, HM-310, HM-311, HM-312, HM-313, HM-314, HM-315, HM-316, HM-317, HM-318, HM-319, HM-320, HM-321, HM-322, HM-323, HM-324, HM-325, HM-326
- **Depth chart**: HM-005, HM-081, HM-101, HM-152, HM-207, HM-208, HM-209, HM-210, HM-211, HM-212, HM-213, HM-214, HM-215, HM-216, HM-217, HM-218, HM-219, HM-220, HM-221, HM-222, HM-223, HM-224, HM-225, HM-226, HM-227, HM-228, HM-229, HM-332
- **Saved stats view**: HM-230, HM-231, HM-232, HM-233, HM-234, HM-235

## New data paths

- **Individual NFL team, week and position options**: HM-333, HM-334, HM-335, HM-336, HM-337, HM-338, HM-339, HM-340, HM-341, HM-342, HM-343, HM-344, HM-345, HM-346, HM-347, HM-348, HM-349, HM-350, HM-351, HM-352, HM-353, HM-354, HM-355, HM-356, HM-357, HM-358, HM-359, HM-360, HM-361, HM-362, HM-363, HM-364, HM-365, HM-366, HM-367, HM-368, HM-369, HM-370, HM-371, HM-372, HM-373, HM-374, HM-375, HM-376, HM-377, HM-378, HM-379, HM-380, HM-381, HM-382

Every NEW/PARTIAL row is grouped below; see its row for the exact gap. NEW indicates not established in inspected paths, not an exhaustive repository search.

- **Alerts and league clock**: HM-012, HM-021, HM-022, HM-023, HM-024, HM-025, HM-026, HM-027, HM-046, HM-047, HM-048, HM-049, HM-126, HM-127, HM-128, HM-129, HM-131, HM-134, HM-135, HM-136, HM-137, HM-138, HM-139, HM-173
- **Contracts and cap**: HM-018, HM-091, HM-112, HM-113, HM-114, HM-115, HM-116, HM-117, HM-118, HM-269, HM-270
- **Draft picks**: HM-109, HM-110, HM-111
- **Home shell, preferences and navigation**: HM-001, HM-002, HM-004, HM-007, HM-008, HM-009, HM-010, HM-011, HM-034, HM-035, HM-036, HM-037, HM-038, HM-039, HM-040, HM-041, HM-042, HM-043, HM-044, HM-045, HM-089, HM-090, HM-092, HM-094, HM-095, HM-097, HM-099, HM-100, HM-140, HM-141, HM-142, HM-143, HM-144, HM-145, HM-146, HM-147, HM-148, HM-149, HM-150, HM-151, HM-154, HM-155, HM-156, HM-157, HM-158, HM-160, HM-161, HM-162, HM-163, HM-164, HM-165, HM-166, HM-167, HM-168, HM-169, HM-170, HM-171, HM-172, HM-174, HM-175, HM-327, HM-328, HM-329, HM-330, HM-331
- **Matchups, standings and rankings**: HM-003, HM-013, HM-014, HM-015, HM-016, HM-017, HM-019, HM-020, HM-050, HM-051, HM-052, HM-053, HM-054, HM-055, HM-056, HM-057, HM-058, HM-059, HM-060, HM-061, HM-062, HM-063, HM-064, HM-065, HM-066, HM-067, HM-068, HM-069, HM-070, HM-071, HM-098, HM-101, HM-102, HM-103, HM-104, HM-159
- **NFL depth and directory**: HM-005, HM-152, HM-207, HM-208, HM-209, HM-210, HM-211, HM-212, HM-213, HM-214, HM-215, HM-216, HM-217, HM-218, HM-219, HM-220, HM-221, HM-222, HM-223, HM-224, HM-225, HM-227, HM-228, HM-229, HM-244, HM-332
- **NFL statistics, scoring and projections**: HM-006, HM-028, HM-029, HM-030, HM-031, HM-032, HM-033, HM-122, HM-123, HM-124, HM-125, HM-153, HM-230, HM-231, HM-232, HM-233, HM-234, HM-235, HM-236, HM-237, HM-238, HM-239, HM-240, HM-241, HM-242, HM-243, HM-245, HM-246, HM-247, HM-248, HM-249, HM-250, HM-251, HM-252, HM-253, HM-255, HM-256, HM-257, HM-273, HM-274, HM-275, HM-283, HM-284, HM-285, HM-286, HM-287, HM-288, HM-289, HM-290, HM-291, HM-292, HM-293, HM-294, HM-295, HM-296, HM-297, HM-298, HM-299, HM-300, HM-301, HM-302, HM-303, HM-304, HM-305, HM-306, HM-307, HM-308, HM-309, HM-310, HM-311, HM-312, HM-313, HM-314, HM-315, HM-316, HM-317, HM-318, HM-319, HM-320, HM-321, HM-322, HM-323, HM-324, HM-325, HM-326
- **News and league headlines**: HM-072, HM-073, HM-074, HM-075, HM-076, HM-077, HM-078, HM-079, HM-080, HM-081, HM-082, HM-083, HM-084, HM-176, HM-177, HM-179, HM-180, HM-181, HM-183, HM-184, HM-185, HM-186, HM-272, HM-281, HM-282
- **Plans**: HM-096, HM-105, HM-106, HM-107, HM-108
- **Player profile and ownership**: HM-260, HM-262, HM-263, HM-264, HM-265, HM-266, HM-268, HM-271, HM-276, HM-278
- **Transactions, trades and market**: HM-085, HM-086, HM-087, HM-088, HM-093, HM-119, HM-120, HM-121, HM-187, HM-188, HM-189, HM-190, HM-191, HM-192, HM-193, HM-194, HM-195, HM-196, HM-197, HM-198, HM-199, HM-200, HM-201, HM-202, HM-203, HM-204, HM-205, HM-206, HM-277, HM-279

## Doubts

1. No role-specific behavior is implemented in this mock; owner, commissioner, co-commissioner and DOT visibility/permissions need product confirmation, particularly sealed bids and actions.
2. Mock provider names are illustrative; only the origin/main files explicitly cited in the table were inspected. NEW does not establish absence elsewhere in the repository.
3. Mock data are fabricated, with generated ownership, player bios, depth charts and statistics; timestamps and snooze deadlines are fixed. No real refresh or persistence was verified.
4. Saved-view labels do not always match their implemented filters/sort, and week/team/minimum selectors, many links and buttons have no handlers.
5. Depth source is explicitly TBD; the mock's stats-feed and scoring-source claims were not validated.
6. Power, playoff odds, projections, future cap room and historical scoring methodologies are unspecified.
7. The DL/DB FF column uses the same `ff` key as fantasy points in the mock, rather than its generated forced-fumble value.
8. The plan header says 2 of 8 checked while the mock renders 13 tasks.
9. Slate/graphite/grey click handling exists, but no palette controls are present in the HTML.
10. Empty/loading/stale/unavailable/error are listed in the tile inspector; actual per-tile handling and many empty states are not implemented.
11. No source was established for fantasy transaction ledger cards; the inspected commissioner ledger is explicitly app-setting changes, not proof of transaction coverage.
12. The news-to-player button uses a restrictive headline regex; many headlines cannot open the player card.
