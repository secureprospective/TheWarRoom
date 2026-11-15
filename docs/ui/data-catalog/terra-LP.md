# League Pulse data catalog

Scope: `docs/ui/mocks/league-pulse-mock-v14.html` only. Each repeated table row describes one field per repeated item, not one row per sample value. UI selections write local state only; no mock control writes league data. Roles are not distinguished by the mock unless noted. Source NEW means no serving path verified, not proof none exists.

| ID | Screen › Region › Element | Shows / does | Data shape | Source on origin/main (path:symbol) or NEW | Read/Write | Refresh | Role/state variants |
|---|---|---|---|---|---|---|---|
| LP-001 | LP › Shell › League label | League name | `name:string` | NEW | Read; local UI write where interactive | per-load | all roles; sample Legacy NFL |
| LP-001 | LP › Shell › League week | Current week | `week:int` | NEW | Read; local UI write where interactive | on-event | all roles |
| LP-001 | LP › Shell › Live count | Live NFL game count | `count:int` | NEW | Read; local UI write where interactive | live (stat feed) | all roles |
| LP-001 | LP › Shell › Leading shortcut | Leading/trailing and score, opens live | `{mine:number,theirs:number,week:int}` | NEW | Read; local UI write where interactive | live (stat feed) | prior week shows current week live |
| LP-001 | LP › Shell › Navigation tabs | Live, Race, Archive workspace selection | `enum{live,race,arch}` | NEW | Read; local UI write where interactive | on-event | local selection |
| LP-001 | LP › Shell › Command hint | Workspace-specific sample command | `{workspace:string,command:string}` | NEW | Read; local UI write where interactive | on-event | hint only; no command handler |
| LP-001 | LP › Strip › Week label | Selected week | `week:int` | NEW | Read; local UI write where interactive | on-event | week 1–9 in mock |
| LP-001 | LP › Strip › Previous week | Select previous week; title Previous week | `week:int` | NEW | Read; local UI write where interactive | on-event | minimum week 1 |
| LP-001 | LP › Strip › Next week | Select next week; title Next week | `week:int` | NEW | Read; local UI write where interactive | on-event | disabled at week 9 |
| LP-001 | LP › Strip › Slate timestamp | Current slate day and time | `timestamp:datetime` | NEW | Read; local UI write where interactive | live (stat feed) | current week only; hardcoded mock |
| LP-001 | LP › Strip › Slate live | Number of NFL live games | `count:int` | NEW | Read; local UI write where interactive | live (stat feed) | current week |
| LP-001 | LP › Strip › Slate final | Number of NFL final games | `count:int` | NEW | Read; local UI write where interactive | live (stat feed) | current week |
| LP-001 | LP › Strip › Slate pending | Number of NFL games to play | `count:int` | NEW | Read; local UI write where interactive | live (stat feed) | current week |
| LP-001 | LP › Strip › Historical lock | Final and 16 results locked | `{week:int,final:bool,locked_count:int}` | NEW | Read; local UI write where interactive | per-load | previous weeks |
| LP-001 | LP › Strip › Freshness | ESPN live source and age; official nflverse Tuesday | `{source:string,age_seconds:int,official_at:datetime}` | NEW | Read; local UI write where interactive | live (stat feed) | historical says checked facts |
| LP-001 | LP › Strip › My score | Own matchup points | `points:number` | NEW | Read; local UI write where interactive | live (stat feed) | all roles |
| LP-001 | LP › Strip › Opponent score | Opponent abbreviation and points | `{team_id:string,abbr:string,points:number}` | NEW | Read; local UI write where interactive | live (stat feed) | all roles |
| LP-001 | LP › Strip › Win probability bar | Own win probability; title win probability | `probability:number` | NEW | Read; local UI write where interactive | live (stat feed) | model estimate |
| LP-001 | LP › Strip › My matchup control | Title Open my matchup; selects mine and live matchup list | `{matchup_id:string,workspace:string}` | NEW | Read; local UI write where interactive | on-event | local selection |
| LP-001 | LP › Matchups › Week header | Week and game count | `{week:int,count:int}` | NEW | Read; local UI write where interactive | on-event | hidden workspace exists in DOM; no visible tab |
| LP-001 | LP › Matchups › All filter | All matchups and count | `{selected:bool,count:int}` | NEW | Read; local UI write where interactive | on-event | local selection |
| LP-001 | LP › Matchups › Mine filter | Own matchup and count | `{selected:bool,count:int}` | NEW | Read; local UI write where interactive | on-event | local selection |
| LP-001 | LP › Matchups › Live filter | Live matchups and count | `{selected:bool,count:int}` | NEW | Read; local UI write where interactive | live (stat feed) | omitted prior weeks |
| LP-001 | LP › Matchups › Close filter | Matchups within 10 points and count | `{selected:bool,count:int}` | NEW | Read; local UI write where interactive | live (stat feed) | local selection |
| LP-001 | LP › Matchups › Section headers | My matchup and conference counts | `{conference:string,count:int}` | NEW | Read; local UI write where interactive | on-event | omit empty groups |
| LP-001 | LP › Matchups › Row teams | Both franchise names and selection | `{matchup_id:string,team_ids:string[],names:string[]}` | NEW | Read; local UI write where interactive | on-event | own matchup marked |
| LP-001 | LP › Matchups › Row scores | Both fantasy totals; leading side emphasized | `points:number[2]` | NEW | Read; local UI write where interactive | live (stat feed) | ties favor first side |
| LP-001 | LP › Matchups › Row status | LIVE, WAITING, ALL PLAYED or FINAL | `enum{live,waiting,all_played,final}` | NEW | Read; local UI write where interactive | live (stat feed) | past week final |
| LP-001 | LP › Matchups › Row win probability | Probability text and bar | `probability:number` | NEW | Read; local UI write where interactive | live (stat feed) | text hidden past week |
| LP-001 | LP › Matchups › Empty filter | No games match this filter | `empty:bool` | NEW | Read; local UI write where interactive | on-event | no matching games |
| LP-001 | LP › Matchups › Scoreboard names | Both franchise names and YOU/THEM labels | `{team_ids:string[],names:string[],mine:bool}` | NEW | Read; local UI write where interactive | on-event | own matchup only labels |
| LP-001 | LP › Matchups › Scoreboard records | Both records | `{wins:int,losses:int}[2]` | NEW | Read; local UI write where interactive | per-load | all roles |
| LP-001 | LP › Matchups › Scoreboard divisions | Both conference/division labels | `{conference:string,division:string}[2]` | NEW | Read; local UI write where interactive | per-load | mock hardcodes NFC |
| LP-001 | LP › Matchups › Scoreboard projections | Projected final score each side | `number[2]` | NEW | Read; local UI write where interactive | live (stat feed) | model estimate |
| LP-001 | LP › Matchups › Scoreboard scores | Fantasy totals for each side | `number[2]` | NEW | Read; local UI write where interactive | live (stat feed) | all roles |
| LP-001 | LP › Matchups › Scoreboard player counts | Playing, to play and done counts each side | `{playing:int,pending:int,done:int}[2]` | NEW | Read; local UI write where interactive | live (stat feed) | all roles |
| LP-001 | LP › Matchups › Scoreboard win chances | Two probabilities and bar | `number[2]` | NEW | Read; local UI write where interactive | live (stat feed) | model estimate |
| LP-001 | LP › Matchups › Scoreboard status | Matchup current/final status | `enum{live,waiting,all_played,final}` | NEW | Read; local UI write where interactive | live (stat feed) | past week final |
| LP-001 | LP › Matchups › Layout switch | Across the line or classic | `enum{across,classic}` | NEW | Read; local UI write where interactive | on-event | local selection |
| LP-001 | LP › Matchups › Source note | Live ESPN or historical checked facts | `{source:string,week:int}` | NEW | Read; local UI write where interactive | on-event | past versus current |
| LP-001 | LP › Matchups › Slot | Starting lineup position per side | `slot:string` | NEW | Read; local UI write where interactive | on-event | across and classic |
| LP-001 | LP › Matchups › Player name | Starter name; click/drag to inspector | `{player_id:string,name:string}` | NEW | Read; local UI write where interactive | on-event | each side |
| LP-001 | LP › Matchups › Player NFL team | Starter NFL abbreviation | `nfl_team:string` | NEW | Read; local UI write where interactive | on-event | each side |
| LP-001 | LP › Matchups › Player game state | Status dot and clock/kickoff/done | `{state:string,clock:string}` | NEW | Read; local UI write where interactive | live (stat feed) | pending/live/final |
| LP-045 | LP › Matchups › Player points | Starter fantasy points and comparison | `points:number` | null | Read; local UI write where interactive | NEW | live (stat feed) |
| LP-045 | LP › Matchups › Player projection | Starter projected points | `points:number` | NEW | Read; local UI write where interactive | live (stat feed) | across only |
| LP-045 | LP › Live › Sidebar mode | Matchups or NFL games | `enum{mu,nfl}` | NEW | Read; local UI write where interactive | on-event | local selection |
| LP-045 | LP › Live › Matchup filters | All/live/close buttons and counts | `{filter:string,count:int,selected:bool}` | NEW | Read; local UI write where interactive | live (stat feed) | mine filter reset |
| LP-045 | LP › Live › Matchup list | Matchup rows, status, scores, win percentages and selection | `list<{id:string,teams:string[],scores:number[],status:string,probabilities:number[]}>` | NEW | Read; local UI write where interactive | live (stat feed) | probability hidden prior week |
| LP-045 | LP › Live › Matchup empty | No matchups match this filter | `empty:bool` | NEW | Read; local UI write where interactive | on-event | empty filter |
| LP-045 | LP › Live › NFL filters | All/my matchup/live buttons and counts | `{filter:string,count:int,selected:bool}` | NEW | Read; local UI write where interactive | live (stat feed) | my matchup counts games with exposure |
| LP-045 | LP › Live › NFL groups | Live/to play/final and group counts | `{state:string,count:int}` | NEW | Read; local UI write where interactive | live (stat feed) | empty groups omitted |
| LP-045 | LP › Live › NFL row teams | Away/home abbreviations; click selects game | `{game_id:string,away:string,home:string}` | NEW | Read; local UI write where interactive | on-event | selected/dim final |
| LP-045 | LP › Live › NFL row scores | Away/home scores | `number[2]` | NEW | Read; local UI write where interactive | live (stat feed) | blank pending |
| LP-045 | LP › Live › NFL row possession | Possession indicator per team | `team_id:string` | null | Read; local UI write where interactive | NEW | live (stat feed) |
| LP-045 | LP › Live › NFL row clock | Quarter and time or kickoff/final | `{state:string,clock:string}` | NEW | Read; local UI write where interactive | live (stat feed) | live/pending/final |
| LP-045 | LP › Live › NFL exposure yours | Count and tooltip your starters | `count:int` | NEW | Read; local UI write where interactive | live (stat feed) | hidden when zero |
| LP-045 | LP › Live › NFL exposure theirs | Count and tooltip their starters | `count:int` | NEW | Read; local UI write where interactive | live (stat feed) | hidden when zero |
| LP-045 | LP › Live › Following chips | Four followed games; select game | `list<{game_id:string,teams:string[],scores:number[],status:string}>` | NEW | Read; local UI write where interactive | live (stat feed) | mock hardcoded; pending score dash |
| LP-045 | LP › Live › Game header away | Away team and score | `{team:string,score:int` | null} | Read; local UI write where interactive | NEW | live (stat feed) |
| LP-045 | LP › Live › Game header home | Home team and score | `{team:string,score:int` | null} | Read; local UI write where interactive | NEW | live (stat feed) |
| LP-045 | LP › Live › Game header clock | Game time/status and matchup | `{game_id:string,clock:string,status:string}` | NEW | Read; local UI write where interactive | live (stat feed) | pending/live/final |
| LP-045 | LP › Live › Field end zones | Away/home team abbreviations | `{away:string,home:string}` | NEW | Read; local UI write where interactive | live (stat feed) | live game with field data |
| LP-045 | LP › Live › Field drive | Drive start and current line of scrimmage visualization | `{start_yard:number,los_yard:number}` | NEW | Read; local UI write where interactive | live (stat feed) | live field |
| LP-045 | LP › Live › Field ball | Ball and line of scrimmage positions | `yard:number` | NEW | Read; local UI write where interactive | live (stat feed) | animates on scripted play |
| LP-045 | LP › Live › Field first down | First-down marker position | `yard:number` | NEW | Read; local UI write where interactive | live (stat feed) | live field |
| LP-045 | LP › Live › Field possession | Possessing NFL team | `team_id:string` | NEW | Read; local UI write where interactive | live (stat feed) | live field |
| LP-045 | LP › Live › Field down/distance | Down/distance and field location | `{down_distance:string,location:string}` | NEW | Read; local UI write where interactive | live (stat feed) | live field |
| LP-045 | LP › Live › Field last play | Last play description | `description:string` | NEW | Read; local UI write where interactive | live (stat feed) | live field |
| LP-045 | LP › Live › Field empty | Kickoff pending or final score with box reference | `{state:string,kickoff:datetime,scores:int[]}` | NEW | Read; local UI write where interactive | live (stat feed) | pending/final |
| LP-045 | LP › Live › Game rostered count | Count of listed players | `count:int` | NEW | Read; local UI write where interactive | live (stat feed) | sample counts free agents too |
| LP-045 | LP › Live › Game score contribution yours | Sum of your starters in selected game | `points:number` | NEW | Read; local UI write where interactive | live (stat feed) | all roles |
| LP-045 | LP › Live › Game score contribution theirs | Opponent starter points in selected game | `points:number` | NEW | Read; local UI write where interactive | live (stat feed) | all roles |
| LP-045 | LP › Live › Game players name | Player name; click/drag to inspector | `{player_id:string,name:string}` | NEW | Read; local UI write where interactive | on-event | yours/theirs/other/free agent |
| LP-045 | LP › Live › Game players ownership | Owner and NFL team | `{owner_team_id:string` | null,nfl_team:string} | Read; local UI write where interactive | NEW | on-event |
| LP-045 | LP › Live › Game players points | Fantasy points for player | `points:number` | NEW | Read; local UI write where interactive | live (stat feed) | all roles |
| LP-045 | LP › Live › Game players empty | No rostered players in this game | `empty:bool` | NEW | Read; local UI write where interactive | on-event | zero listed |
| LP-045 | LP › Live › Starters heading | My matchup or selected matchup | `{matchup_id:string,mine:bool}` | NEW | Read; local UI write where interactive | on-event | other matchup has back button |
| LP-045 | LP › Live › My matchup back | Return selected matchup to own | `matchup_id:string` | NEW | Read; local UI write where interactive | on-event | only other matchup |
| LP-045 | LP › Live › Starter projected total | Both matchup projected totals | `number[2]` | NEW | Read; local UI write where interactive | live (stat feed) | model |
| LP-045 | LP › Live › Starter team header | Both franchise names and YOU/THEM | `{teams:string[],mine:bool}` | NEW | Read; local UI write where interactive | on-event | selected matchup |
| LP-045 | LP › Live › Starter total | Both fantasy scores | `number[2]` | NEW | Read; local UI write where interactive | live (stat feed) | selected matchup |
| LP-045 | LP › Live › Starter slot | Offense/defense grouping and position | `{group:string,slot:string}` | NEW | Read; local UI write where interactive | on-event | each matchup row |
| LP-045 | LP › Live › Starter name | Player name and NFL team; pin inspector | `{player_id:string,name:string,nfl_team:string}` | NEW | Read; local UI write where interactive | on-event | both sides |
| LP-045 | LP › Live › Starter compact stats | Position-specific passing/rushing/receiving/defense line | `{player_id:string,stats:map<string,number>}` | NEW | Read; local UI write where interactive | live (stat feed) | pending shows projection; mock synthetic |
| LP-045 | LP › Live › Starter projected points | Player projection before game | `points:number` | NEW | Read; local UI write where interactive | live (stat feed) | pending only |
| LP-045 | LP › Live › Starter game chip | Quarter/clock, kickoff or FINAL | `{state:string,clock:string}` | NEW | Read; local UI write where interactive | live (stat feed) | pending/live/final |
| LP-088 | LP › Live › Starter on field | On-field indicator and tooltip | `{on_field:bool,team_possession:string,position:string}` | NEW | Read; local UI write where interactive | live (stat feed) | mock inferred from possession |
| LP-088 | LP › Live › Starter fantasy points | Player score and comparison | `points:number` | null | Read; local UI write where interactive | NEW | live (stat feed) |
| LP-088 | LP › Live › Play-by-play header | Play count newest first or kickoff time | `{count:int,kickoff:datetime}` | NEW | Read; local UI write where interactive | live (stat feed) | pending versus started |
| LP-088 | LP › Live › Play-by-play clock | Play quarter and clock | `clock:string` | NEW | Read; local UI write where interactive | live (stat feed) | each play |
| LP-088 | LP › Live › Play-by-play description | Play text | `description:string` | NEW | Read; local UI write where interactive | live (stat feed) | each play |
| LP-088 | LP › Live › Play-by-play owner | Your/their starter marker | `enum{mine,theirs,none}` | NEW | Read; local UI write where interactive | live (stat feed) | when involved |
| LP-088 | LP › Live › Play-by-play points | Fantasy points delta | `delta:number` | null | Read; local UI write where interactive | NEW | live (stat feed) |
| LP-088 | LP › Live › Play-by-play empty | Starts at kickoff or no plays held | `{state:string,empty:bool}` | NEW | Read; local UI write where interactive | live (stat feed) | pending/started |
| LP-088 | LP › Live › Tick notification | Player, play, points delta and side | `{player_id:string,text:string,delta:number,side:string}` | NEW | Read; local UI write where interactive | live (stat feed) | scripted feed event |
| LP-097 | LP › Race › Conference switch | AFC or NFC selection | `enum{AFC,NFC}` | NEW | Read; local UI write where interactive | on-event | local selection |
| LP-098 | LP › Race › Standings view | Seeds or divisions | `enum{seed,division}` | NEW | Read; local UI write where interactive | on-event | local selection |
| LP-099 | LP › Race › Seed | Current playoff seed and cutoff | `seed:int` | PARTIAL: internal/app/season/playoffs_reading.go:View.Seeding lacks this mock standings view | Read; local UI write where interactive | per-load | seed/division modes |
| LP-100 | LP › Race › Franchise | Franchise name; click opens inspector | `{team_id:string,name:string}` | NEW | Read; local UI write where interactive | on-event | own/selected highlighted |
| LP-101 | LP › Race › Division | Division label and division winner star | `{division:string,winner:bool}` | PARTIAL: internal/app/season/playoffs_reading.go:View.Fields lacks mock display | Read; local UI write where interactive | per-load | seed mode |
| LP-102 | LP › Race › Record | Wins and losses | `{wins:int,losses:int}` | PARTIAL: internal/app/season/playoffs_reading.go:View.Seeding lacks this mock standings view | Read; local UI write where interactive | per-load | all roles |
| LP-103 | LP › Race › Points for | Season fantasy points for | `points:number` | NEW | Read; local UI write where interactive | per-load | seed view |
| LP-104 | LP › Race › Games back | Games behind division leader | `games:number` | NEW | Read; local UI write where interactive | per-load | division view; dash for leader |
| LP-105 | LP › Race › Power rank | Season power rank | `rank:int` | PARTIAL: frontend/src/app/race/state.ts:createRaceState lacks mock component metrics | Read; local UI write where interactive | per-load | highlight high/low |
| LP-106 | LP › Race › Playoff odds | Probability of making playoffs | `percent:number` | NEW | Read; local UI write where interactive | per-load | estimate |
| LP-107 | LP › Race › Division header | Conference/division name | `{conference:string,division:string}` | NEW | Read; local UI write where interactive | per-load | division view |
| LP-108 | LP › Race › Division leader | Leader franchise name and record | `{team_id:string,name:string,wins:int,losses:int}` | NEW | Read; local UI write where interactive | per-load | division view |
| LP-109 | LP › Race › My road week | Remaining schedule week | `week:int` | NEW | Read; local UI write where interactive | per-load | weeks 9–14 |
| LP-110 | LP › Race › My road opponent | Opponent name and division leader marker | `{team_id:string,name:string,division_leader:bool}` | NEW | Read; local UI write where interactive | per-load | current game NOW |
| LP-111 | LP › Race › My road record | Opponent W-L | `{wins:int,losses:int}` | NEW | Read; local UI write where interactive | per-load | all roles |
| LP-112 | LP › Race › My road chance | Win probability versus opponent | `percent:number` | NEW | Read; local UI write where interactive | per-load | mock hardcodes current game probability |
| LP-113 | LP › Race › Race mode | Power rankings or playoff picture | `enum{power,path}` | NEW | Read; local UI write where interactive | on-event | local selection |
| LP-114 | LP › Race › Power horizon | This season, dynasty 3-year, blend | `enum{now,dyn,blend}` | PARTIAL: frontend/src/app/race/state.ts:createRaceState lacks mock horizons | Read; local UI write where interactive | on-event | power mode |
| LP-115 | LP › Race › Race context | Conference, week and in-season status | `{conference:string,week:int,season_phase:string}` | NEW | Read; local UI write where interactive | on-event | mock hardcodes in season |
| LP-116 | LP › Race › Power explanation | Description of selected model/horizon | `{horizon:string,description:string}` | NEW | Read; local UI write where interactive | on-event | season/dynasty/blend |
| LP-117 | LP › Race › Power weight | Each model factor name and percentage | `{factor:string,weight_percent:number}` | NEW | Read; local UI write where interactive | on-event | blend 60/40 in mock |
| LP-118 | LP › Race › Power weight tooltip | Description of each component | `description:string` | NEW | Read; local UI write where interactive | on-event | component-specific; blank in blend mock |
| LP-119 | LP › Race › Power rank row | Rank per team | `{team_id:string,rank:int}` | PARTIAL: frontend/src/app/race/state.ts:createRaceState lacks mock horizons | Read; local UI write where interactive | per-load | season/dynasty/blend |
| LP-120 | LP › Race › Power movement | Week-over-week movement arrow and magnitude | `delta:int` | NEW | Read; local UI write where interactive | per-load | mock zero outside season |
| LP-121 | LP › Race › Power team | Franchise name and inspector selection | `{team_id:string,name:string}` | NEW | Read; local UI write where interactive | on-event | mine highlighted |
| LP-122 | LP › Race › Power window badge | Built to last/win now/on the rise/rebuilding/middle | `window:string` | NEW | Read; local UI write where interactive | per-load | calculated from ranks |
| LP-123 | LP › Race › Power score | Composite numeric score and bar | `score:number` | NEW | Read; local UI write where interactive | per-load | selected horizon |
| LP-124 | LP › Race › Power component ranks | Rank for each season/dynasty component | `map<string,int>` | NEW | Read; local UI write where interactive | per-load | not blend |
| LP-125 | LP › Race › Power season rank | Season rank in blend | `rank:int` | NEW | Read; local UI write where interactive | per-load | blend only |
| LP-126 | LP › Race › Power dynasty rank | Dynasty rank in blend | `rank:int` | NEW | Read; local UI write where interactive | per-load | blend only |
| LP-127 | LP › Race › Power season score | Season score in blend | `score:number` | NEW | Read; local UI write where interactive | per-load | blend only |
| LP-128 | LP › Race › Power dynasty score | Dynasty score in blend | `score:number` | NEW | Read; local UI write where interactive | per-load | blend only |
| LP-129 | LP › Race › Power shift | Difference and tooltip dynasty rank vs season rank | `delta:int` | NEW | Read; local UI write where interactive | per-load | blend only |
| LP-130 | LP › Race › Power peak year | Estimated peak roster year | `year:int` | NEW | Read; local UI write where interactive | per-load | dynasty/blend only; synthetic |
| LP-131 | LP › Race › Power vs record | Difference between seed and power rank | `delta:int` | NEW | Read; local UI write where interactive | per-load | all horizons |
| LP-132 | LP › Race › Movers period | This week, this season, last 12 months | `enum{week,season,year}` | NEW | Read; local UI write where interactive | on-event | local selection |
| LP-133 | LP › Race › Movers franchise | Franchise name and inspector selection | `{team_id:string,name:string}` | NEW | Read; local UI write where interactive | on-event | each listed mover |
| LP-134 | LP › Race › Movers delta | Rank movement arrow and magnitude | `delta:int` | NEW | Read; local UI write where interactive | per-load | selected period |
| LP-135 | LP › Race › Weekly mover reason | Narrative reason for change | `reason:string` | NEW | Read; local UI write where interactive | per-load | week view |
| LP-136 | LP › Race › Season mover chart | Weekly power rank trajectory and endpoints | `list<{week:int,rank:int}>` | NEW | Read; local UI write where interactive | per-load | season view; mock generated |
| LP-137 | LP › Race › Season mover tooltip | Week and rank for each chart point | `{week:int,rank:int}` | NEW | Read; local UI write where interactive | per-load | hover each chart point |
| LP-138 | LP › Race › Year mover reason | Trade/draft/free-agent reason | `reason:string` | NEW | Read; local UI write where interactive | per-load | year view |
| LP-139 | LP › Race › Year mover dates | Starting and current blended ranks and date | `{from_date:date,from_rank:int,to_rank:int}` | NEW | Read; local UI write where interactive | per-load | year view |
| LP-140 | LP › Race › My seed summary | Current and projected seed | `{current:int` | null,projected:int} | Read; local UI write where interactive | NEW | per-load |
| LP-141 | LP › Race › My playoff odds | Simulation probability to qualify | `percent:number` | NEW | Read; local UI write where interactive | per-load | mock synthetic not simulation |
| LP-142 | LP › Race › My bye odds | Simulation probability of first-round bye | `percent:number` | NEW | Read; local UI write where interactive | per-load | mock hardcoded |
| LP-143 | LP › Race › Magic number | My wins plus eighth-place losses needed | `count:int` | NEW | Read; local UI write where interactive | per-load | mock hardcoded |
| LP-144 | LP › Race › Run-in current seed | Current seed per team | `seed:int` | PARTIAL: internal/app/season/playoffs_reading.go:View.Seeding lacks future projection | Read; local UI write where interactive | per-load | top 11 only |
| LP-145 | LP › Race › Run-in projected seed | Projected seed and direction | `{seed:int,change:int}` | NEW | Read; local UI write where interactive | per-load | model |
| LP-146 | LP › Race › Run-in team | Name and inspector selection | `{team_id:string,name:string}` | NEW | Read; local UI write where interactive | on-event | mine and selected highlighting |
| LP-147 | LP › Race › Run-in record | Current wins and losses | `{wins:int,losses:int}` | NEW | Read; local UI write where interactive | per-load | all teams |
| LP-148 | LP › Race › Run-in projected wins | Projected season wins | `wins:number` | NEW | Read; local UI write where interactive | per-load | model |
| LP-149 | LP › Race › Run-in opponent | Opponent abbreviation per future week | `{week:int,team_id:string,abbr:string,division_leader:bool}` | NEW | Read; local UI write where interactive | per-load | weeks 10–14 |
| LP-150 | LP › Race › Run-in opponent chance | Win probability per opponent | `percent:number` | NEW | Read; local UI write where interactive | per-load | color indicates chance |
| LP-151 | LP › Race › Run-in tooltip | Opponent full name and win probability | `{name:string,percent:number}` | NEW | Read; local UI write where interactive | per-load | hover opponent |
| LP-152 | LP › Race › Run-in odds | Playoff probability | `percent:number` | NEW | Read; local UI write where interactive | per-load | model |
| LP-153 | LP › Race › Run-in bubble | Current qualifying team projected out | `bubble:bool` | NEW | Read; local UI write where interactive | per-load | conditional |
| LP-154 | LP › Race › Run-in difficulty | Consecutive leaders, count of leaders or soft finish | `{leader_count:int,longest_run:int}` | NEW | Read; local UI write where interactive | per-load | five remaining games |
| LP-155 | LP › Race › Projected bracket header | Conference, weeks and opposite-side final | `{conference:string,weeks:int[],opposing_conference:string}` | NEW | Read; local UI write where interactive | per-load | selected conference |
| LP-156 | LP › Race › Projected bracket slots | Seeds and teams for wildcard and bye | `list<{round:string,seed:int,team_id:string}>` | PARTIAL: internal/app/season/playoffs_reading.go:View.Seeding lacks projected future seeds | Read; local UI write where interactive | per-load | future winners TBD |
| LP-157 | LP › Race › Projected bracket opponent | Other conference projected top seed | `{conference:string,team_id:string,seed:int}` | NEW | Read; local UI write where interactive | per-load | final |
| LP-158 | LP › Race › Scenario own win | Playoff odds if own team wins | `percent:number` | NEW | Read; local UI write where interactive | per-load | mock hardcoded |
| LP-159 | LP › Race › Scenario own loss | Playoff odds if own team loses | `percent:number` | NEW | Read; local UI write where interactive | per-load | mock hardcoded |
| LP-160 | LP › Race › Scenario rival loss bye | Bye odds if named rival loses | `{team_id:string,percent:number}` | NEW | Read; local UI write where interactive | per-load | mock hardcoded |
| LP-161 | LP › Race › Scenario rival loss change | Odds change if named rival loses | `{team_id:string,delta_percent:number}` | NEW | Read; local UI write where interactive | per-load | mock hardcoded |
| LP-162 | LP › Archive › Mode | Franchise or league | `enum{franchise,league}` | NEW | Read; local UI write where interactive | on-event | local selection |
| LP-163 | LP › Archive › Franchise selector | All franchise names, own team first and marked mine | `list<{team_id:string,name:string,mine:bool}>` | NEW | Read; local UI write where interactive | on-event | franchise mode only |
| LP-164 | LP › Archive › All eras | All eras record, owner count and years | `{start_year:int,end_year:int,owners:int,wins:int,losses:int}` | PARTIAL: internal/app/archive/read.go:Read lacks this mock era aggregation | Read; local UI write where interactive | per-load | franchise mode |
| LP-165 | LP › Archive › Owner era | Owner, years, number of seasons and W-L | `{owner_id:string,name:string,start_year:int,end_year:int,wins:int,losses:int}` | NEW | Read; local UI write where interactive | per-load | click selects era; own marked |
| LP-166 | LP › Archive › Era mode empty | League mode prompt to pick franchise | `mode:string` | NEW | Read; local UI write where interactive | on-event | league mode |
| LP-167 | LP › Archive › Totals record | Era W-L | `{wins:int,losses:int}` | PARTIAL: internal/app/archive/read.go:Read lacks era aggregate view | Read; local UI write where interactive | per-load | selected era |
| LP-168 | LP › Archive › Totals win percentage | Era wins divided by games | `percent:number` | null | Read; local UI write where interactive | NEW | per-load |
| LP-169 | LP › Archive › Totals playoff appearances | Playoff seasons and total seasons | `{playoff_seasons:int,seasons:int}` | NEW | Read; local UI write where interactive | per-load | selected era |
| LP-170 | LP › Archive › Totals division titles | Division titles count | `count:int` | NEW | Read; local UI write where interactive | per-load | selected era |
| LP-171 | LP › Archive › Totals conference titles | Conference titles count | `count:int` | NEW | Read; local UI write where interactive | per-load | selected era; mock labels NFC |
| LP-172 | LP › Archive › Totals championships | Titles and finals counts | `{titles:int,finals:int}` | NEW | Read; local UI write where interactive | per-load | selected era |
| LP-173 | LP › Archive › Last title | Most recent title year or never | `year:int` | null | Read; local UI write where interactive | NEW | per-load |
| LP-174 | LP › Archive › Last final | Most recent final year or never | `year:int` | null | Read; local UI write where interactive | NEW | per-load |
| LP-175 | LP › Archive › Playoff streak | Consecutive playoff seasons | `count:int` | NEW | Read; local UI write where interactive | per-load | mock ignores era selection |
| LP-176 | LP › Archive › Most titles | Top seven franchise names and title stars | `list<{team_id:string,name:string,titles:int}>` | NEW | Read; local UI write where interactive | per-load | league mode |
| LP-177 | LP › Archive › Shelf trophies | Trophy type and year | `list<{type:string,year:int}>` | NEW | Read; local UI write where interactive | per-load | first 12; empty says no trophies yet |
| LP-178 | LP › Archive › Shelf action | Open trophy case or record book; tooltip | `{mode:string,drawer:string}` | NEW | Read; local UI write where interactive | on-event | franchise/league |
| LP-179 | LP › Archive › Season chart heading | Franchise name, chart meaning | `{team_id:string,name:string}` | NEW | Read; local UI write where interactive | on-event | franchise mode |
| LP-180 | LP › Archive › Season bars | Record and win percentage per season; select season | `{year:int,wins:int,losses:int,win_percent:number}` | NEW | Read; local UI write where interactive | per-load | title marker; selected/other era dim |
| LP-181 | LP › Archive › Season labels | Year and regular-season rank of 32 | `{year:int,rank:int,league_size:int}` | NEW | Read; local UI write where interactive | per-load | per season |
| LP-182 | LP › Archive › Season stamps | Division title, champion, conference champ, playoff win/playoffs | `list<{year:int,type:string}>` | NEW | Read; local UI write where interactive | per-load | per season; mock labels NFC |
| LP-183 | LP › Archive › Era chart bands | Owner or YOUR ERA and year range | `{owner_id:string,name:string,start_year:int,end_year:int,mine:bool}` | NEW | Read; local UI write where interactive | per-load | per owner |
| LP-184 | LP › Archive › Season tooltip record | Year and W-L | `{year:int,wins:int,losses:int}` | NEW | Read; local UI write where interactive | per-load | hover bar |
| LP-185 | LP › Archive › Season tooltip rank | Regular season rank and league size | `{rank:int,league_size:int}` | NEW | Read; local UI write where interactive | per-load | hover bar |
| LP-186 | LP › Archive › Season tooltip stage | Playoff stage | `stage:string` | NEW | Read; local UI write where interactive | per-load | hover bar |
| LP-187 | LP › Archive › Season tooltip division | Division won or dash | `won:bool` | NEW | Read; local UI write where interactive | per-load | hover bar |
| LP-188 | LP › Archive › Season tooltip points | Season points for | `points:number` | NEW | Read; local UI write where interactive | per-load | hover bar |
| LP-189 | LP › Archive › Season tooltip owner | Owner name for season | `{owner_id:string,name:string}` | NEW | Read; local UI write where interactive | per-load | hover bar |
| LP-190 | LP › Archive › Season story header | Year, playoff stage, record and rank | `{year:int,stage:string,wins:int,losses:int,rank:int}` | NEW | Read; local UI write where interactive | per-load | selected season |
| LP-191 | LP › Archive › Season story owner | Owner and own-era styling | `{owner_id:string,name:string,mine:bool}` | NEW | Read; local UI write where interactive | per-load | selected season |
| LP-192 | LP › Archive › Season checks | Checked count and open count | `{checked:int,total:int,open:int}` | PARTIAL: internal/app/archive/read.go:Read lacks mock season check counts | Read; local UI write where interactive | per-load | old years 7/9; newer 9/9 |
| LP-193 | LP › Archive › Season story title | Season narrative headline | `headline:string` | NEW | Read; local UI write where interactive | per-load | fallback team and stage |
| LP-194 | LP › Archive › Season story text | Season narrative and commissioner notes | `text:string` | NEW | Read; local UI write where interactive | per-load | fallback no commissioner notes |
| LP-195 | LP › Archive › Season story stamps | Titles/playoff stamps or no titles message | `list<string>` | NEW | Read; local UI write where interactive | per-load | empty/no playoffs |
| LP-196 | LP › Archive › Playoff run count | Number of playoff games | `count:int` | NEW | Read; local UI write where interactive | per-load | none if missed |
| LP-197 | LP › Archive › Playoff run round | Round name for each game | `round:string` | NEW | Read; local UI write where interactive | per-load | selected season |
| LP-198 | LP › Archive › Playoff run opponent | Opponent name; opens inspector | `{team_id:string,name:string}` | NEW | Read; local UI write where interactive | on-event | each game |
| LP-199 | LP › Archive › Playoff run score | Team/opponent points | `number[2]` | NEW | Read; local UI write where interactive | per-load | each game |
| LP-200 | LP › Archive › Playoff run result | Win or loss | `enum{W,L}` | NEW | Read; local UI write where interactive | per-load | each game |
| LP-201 | LP › Archive › Playoff run empty | Missed playoffs with record and rank | `{wins:int,losses:int,rank:int}` | NEW | Read; local UI write where interactive | per-load | no games |
| LP-202 | LP › Archive › Award top scorer | Top scorer and label | `{player_id:string` | null,name:string | Read; local UI write where interactive | null} | NEW |
| LP-203 | LP › Archive › Award best week | Best weekly score and week | `{points:number,week:int}` | NEW | Read; local UI write where interactive | per-load | mock synthetic |
| LP-204 | LP › Archive › Award points for | Season points for and rank | `{points:number,rank:int}` | NEW | Read; local UI write where interactive | per-load | selected season |
| LP-205 | LP › Archive › Lineage/moves tabs | Who was here versus how we got here | `enum{lineage,moves}` | NEW | Read; local UI write where interactive | on-event | local selection |
| LP-206 | LP › Archive › Lineage slot | Player position | `position:string` | NEW | Read; local UI write where interactive | per-load | selected season |
| LP-207 | LP › Archive › Lineage player | Player name; pin inspector | `{player_id:string,name:string}` | NEW | Read; local UI write where interactive | on-event | selected season |
| LP-208 | LP › Archive › Lineage acquisition | Acquisition method and year | `{method:string,year:int}` | NEW | Read; local UI write where interactive | per-load | per player |
| LP-209 | LP › Archive › Lineage owner | Owner who acquired and optional note | `{owner_id:string,name:string,note:string` | null} | Read; local UI write where interactive | NEW | per-load |
| LP-210 | LP › Archive › Lineage empty | Transactions not yet imported | `imported:bool` | NEW | Read; local UI write where interactive | per-load | other franchises in mock |
| LP-211 | LP › Archive › Key move year | Move year | `year:int` | NEW | Read; local UI write where interactive | per-load | era filter |
| LP-212 | LP › Archive › Key move description | Narrative of key move | `description:string` | NEW | Read; local UI write where interactive | per-load | era filter |
| LP-213 | LP › Archive › Key move owner | Owner credited for move | `{owner_id:string,name:string}` | NEW | Read; local UI write where interactive | per-load | own says you |
| LP-214 | LP › Archive › Key moves empty | No key moves recorded | `empty:bool` | NEW | Read; local UI write where interactive | per-load | other franchises in mock |
| LP-215 | LP › Archive › League season | Season year | `year:int` | PARTIAL: internal/app/archive/read.go:Read lacks mock champions table | Read; local UI write where interactive | per-load | league mode |
| LP-216 | LP › Archive › League champion | Champion name; row opens champion history | `{team_id:string,name:string}` | NEW | Read; local UI write where interactive | on-event | league mode |
| LP-217 | LP › Archive › League runner-up | Runner-up franchise name | `{team_id:string,name:string}` | NEW | Read; local UI write where interactive | per-load | league mode |
| LP-218 | LP › Archive › League champion record | Champion regular-season W-L | `{wins:int,losses:int}` | NEW | Read; local UI write where interactive | per-load | league mode |
| LP-219 | LP › Archive › League my finish | Own regular rank and playoff stage | `{rank:int,stage:string}` | NEW | Read; local UI write where interactive | per-load | league mode |
| LP-220 | LP › Archive › League checks | Checked/total count | `{checked:int,total:int}` | NEW | Read; local UI write where interactive | per-load | 7/9 old years; 9/9 newer |
| LP-221 | LP › Drawer › Pull controls | Open/close workspace-specific drawers | `{workspace:string,drawer:string,open:bool}` | NEW | Read; local UI write where interactive | on-event | mock adds case and book to archive pulls |
| LP-222 | LP › Drawer › Stat changes badge | Number of stat changes | `count:int` | NEW | Read; local UI write where interactive | on-event | mock fixed at 2 |
| LP-223 | LP › Drawer › Close | Close button and Escape | `open:bool` | NEW | Read; local UI write where interactive | on-event | local UI only |
| LP-224 | LP › Box score › Team tab | Select first or second matchup team | `{team_id:string,side:string}` | NEW | Read; local UI write where interactive | on-event | YOU/THEM when mine |
| LP-225 | LP › Box score › Context | Team names and selected week | `{teams:string[],week:int}` | NEW | Read; local UI write where interactive | on-event | selected matchup |
| LP-226 | LP › Box score › Slot | Starting slot | `slot:string` | NEW | Read; local UI write where interactive | on-event | each starter |
| LP-227 | LP › Box score › Player | Starter name; pin inspector | `{player_id:string,name:string}` | NEW | Read; local UI write where interactive | on-event | each starter |
| LP-228 | LP › Box score › Stat line one | Position-specific primary stats | `map<string,number>` | NEW | Read; local UI write where interactive | live (stat feed) | mock synthetic; pending dashes |
| LP-229 | LP › Box score › Stat line two | Position-specific secondary stats | `map<string,number>` | NEW | Read; local UI write where interactive | live (stat feed) | mock synthetic; pending dashes |
| LP-230 | LP › Box score › Touchdowns/turnovers | Position-specific TD/TO stats | `map<string,number>` | NEW | Read; local UI write where interactive | live (stat feed) | mock synthetic; pending dashes |
| LP-231 | LP › Box score › State | Done, playing or to play | `state:string` | NEW | Read; local UI write where interactive | live (stat feed) | each starter |
| LP-232 | LP › Box score › Points | Fantasy points | `points:number` | null | Read; local UI write where interactive | NEW | live (stat feed) |
| LP-233 | LP › Week story › Tabs | Summary, preview this game, recaps | `enum{summary,preview,recaps}` | NEW | Read; local UI write where interactive | on-event | local selection |
| LP-234 | LP › Week story › Summary time | So-far timestamp or final | `{timestamp:datetime` | null,final:bool} | Read; local UI write where interactive | NEW | on-event |
| LP-235 | LP › Week story › Summary headline | Weekly story headline | `headline:string` | NEW | Read; local UI write where interactive | on-event | current/past week |
| LP-236 | LP › Week story › Summary narrative | Current week developments or results checked | `text:string` | NEW | Read; local UI write where interactive | live (stat feed) | mock hardcoded |
| LP-237 | LP › Week story › High score | Highest team name and points | `{team_id:string,name:string,points:number}` | NEW | Read; local UI write where interactive | live (stat feed) | all matchups |
| LP-238 | LP › Week story › Top QB | Player, points and fantasy owner | `{player_id:string,name:string,points:number,team_id:string}` | NEW | Read; local UI write where interactive | live (stat feed) | mock hardcoded |
| LP-239 | LP › Week story › Top RB | Player, points and fantasy owner | `{player_id:string,name:string,points:number,team_id:string}` | NEW | Read; local UI write where interactive | live (stat feed) | mock hardcoded |
| LP-240 | LP › Week story › Top IDP | Player, points and fantasy owner | `{player_id:string,name:string,points:number,team_id:string}` | NEW | Read; local UI write where interactive | live (stat feed) | mock hardcoded |
| LP-241 | LP › Week story › Preview timestamp | Preview written Thursday | `written_at:datetime` | NEW | Read; local UI write where interactive | per-load | preview tab |
| LP-242 | LP › Week story › Preview matchup | Both team names | `{team_ids:string[],names:string[]}` | NEW | Read; local UI write where interactive | on-event | selected matchup |
| LP-243 | LP › Week story › Preview projections | Pre-kickoff projected score | `number[2]` | NEW | Read; local UI write where interactive | per-load | preview does not update per copy |
| LP-244 | LP › Week story › Preview narrative | Generated matchup preview/edge | `text:string` | NEW | Read; local UI write where interactive | per-load | own or other matchup |
| LP-245 | LP › Week story › Recap | Week, team names, final scores and recap text | `{week:int,teams:string[],scores:number[],text:string}` | NEW | Read; local UI write where interactive | per-load | past week only; mock one hardcoded |
| LP-246 | LP › Week story › Recaps pending | Recaps post after Monday final; week navigation hint | `{final:bool,previous_week:int}` | NEW | Read; local UI write where interactive | on-event | current week |
| LP-247 | LP › Schedule › Tabs | League, my team, compact | `enum{league,franchise,compact}` | NEW | Read; local UI write where interactive | on-event | local selection |
| LP-248 | LP › Schedule › Team schedule week | Regular-season week | `week:int` | NEW | Read; local UI write where interactive | per-load | my team |
| LP-249 | LP › Schedule › Team schedule opponent | Opponent name; pin inspector | `{team_id:string,name:string}` | NEW | Read; local UI write where interactive | on-event | my team |
| LP-250 | LP › Schedule › Team schedule result | W, L, LIVE or dash | `result:string` | null | Read; local UI write where interactive | NEW | live (stat feed) |
| LP-251 | LP › Schedule › Team schedule score | Both fantasy scores | `number[2]` | NEW | Read; local UI write where interactive | live (stat feed) | blank future |
| LP-252 | LP › Schedule › League grid franchise | Franchise name or abbreviation | `{team_id:string,name:string,abbr:string}` | NEW | Read; local UI write where interactive | per-load | league/compact, selected conference |
| LP-253 | LP › Schedule › League grid week | Week column | `week:int` | NEW | Read; local UI write where interactive | per-load | mock shows weeks 8–14 |
| LP-254 | LP › Schedule › League grid cell | Opponent abbreviation and prior result | `{opponent_id:string,abbr:string,result:string` | null} | Read; local UI write where interactive | NEW | live (stat feed) |
| LP-255 | LP › Stat changes › Week | Week of correction | `week:int` | NEW | Read; local UI write where interactive | per-load | mock weeks 7–8 |
| LP-256 | LP › Stat changes › Player | Player name; pin inspector | `{player_id:string,name:string}` | NEW | Read; local UI write where interactive | on-event | each correction |
| LP-257 | LP › Stat changes › Stat change | Stat name, original and updated value | `{stat:string,was:number,now:number}` | NEW | Read; local UI write where interactive | per-load | first held versus latest |
| LP-258 | LP › Stat changes › Fantasy score change | Original and updated player points | `{was:number,now:number}` | NEW | Read; local UI write where interactive | per-load | each correction |
| LP-259 | LP › Stat changes › Delta and owner | Point delta and affected team | `{delta:number,team_id:string}` | NEW | Read; local UI write where interactive | per-load | each correction |
| LP-260 | LP › Stat changes › Result impact | Matchup result changed or stood | `{changed:bool,result:string}` | NEW | Read; local UI write where interactive | per-load | mock no change |
| LP-261 | LP › NFL box › Header | Selected NFL teams, scores and clock | `{teams:string[],scores:int[],clock:string}` | NEW | Read; local UI write where interactive | live (stat feed) | mock box rows fixed regardless selected game |
| LP-262 | LP › NFL box › Category | Passing, rushing, receiving, defence | `enum{pass,rush,rec,def}` | NEW | Read; local UI write where interactive | on-event | local tab |
| LP-263 | LP › NFL box › Player | Player name; pin inspector | `{player_id:string,name:string}` | NEW | Read; local UI write where interactive | on-event | each stat row |
| LP-264 | LP › NFL box › NFL team | Player NFL team | `team:string` | NEW | Read; local UI write where interactive | per-load | all categories |
| LP-265 | LP › NFL box › Attempts/completions | Passing completions/attempts | `{completions:int,attempts:int}` | NEW | Read; local UI write where interactive | live (stat feed) | passing |
| LP-266 | LP › NFL box › Carries | Rushing attempts | `carries:int` | NEW | Read; local UI write where interactive | live (stat feed) | rushing |
| LP-267 | LP › NFL box › Receptions | Receiving catches | `receptions:int` | NEW | Read; local UI write where interactive | live (stat feed) | receiving |
| LP-268 | LP › NFL box › Yards | Passing/rushing/receiving yards | `yards:int` | NEW | Read; local UI write where interactive | live (stat feed) | offense |
| LP-269 | LP › NFL box › Touchdowns | Passing/rushing/receiving touchdowns | `touchdowns:int` | NEW | Read; local UI write where interactive | live (stat feed) | offense |
| LP-270 | LP › NFL box › Interceptions | Passing or defense interceptions | `interceptions:int` | NEW | Read; local UI write where interactive | live (stat feed) | passing/defense |
| LP-271 | LP › NFL box › Targets | Receiving targets | `targets:int` | NEW | Read; local UI write where interactive | live (stat feed) | receiving |
| LP-272 | LP › NFL box › Tackles | Defensive tackles | `tackles:int` | NEW | Read; local UI write where interactive | live (stat feed) | defense |
| LP-273 | LP › NFL box › Sacks | Defensive sacks | `sacks:number` | NEW | Read; local UI write where interactive | live (stat feed) | defense |
| LP-274 | LP › NFL box › Rostered | Fantasy owner or unrostered dash | `team_id:string` | null | Read; local UI write where interactive | NEW | on-event |
| LP-275 | LP › Tiebreakers › Group rank | Tie resolution group number | `index:int` | NEW | Read; local UI write where interactive | per-load | selected conference |
| LP-276 | LP › Tiebreakers › Tied teams | Franchise names and tied record | `{team_ids:string[],names:string[],record:string}` | NEW | Read; local UI write where interactive | per-load | mock fixed data |
| LP-277 | LP › Tiebreakers › Step used | Applied tiebreak rule | `rule:string` | NEW | Read; local UI write where interactive | per-load | division winner/head-to-head/points |
| LP-278 | LP › Tiebreakers › Detail | Evidence for tiebreak decision | `text:string` | NEW | Read; local UI write where interactive | per-load | per group |
| LP-279 | LP › Tiebreakers › Order | Resolved order and seeds | `list<{team_id:string,seed:int}>` | NEW | Read; local UI write where interactive | per-load | per group |
| LP-280 | LP › Tiebreakers › Rule order | Division winners, head-to-head, points for, all-play | `list<string>` | NEW | Read; local UI write where interactive | per-load | mock hardcoded league setting |
| LP-281 | LP › Bracket › Seeded teams | Current conference seeds and wildcard opponents | `list<{conference:string,seed:int,team_id:string}>` | PARTIAL: internal/app/season/playoffs_reading.go:View.Seeding lacks this drawer rendering | Read; local UI write where interactive | per-load | 7 teams and 1 bye in mock |
| LP-282 | LP › Bracket › Later round placeholders | Lowest seed, winners and conference champions | `{round:string,slot:string}` | NEW | Read; local UI write where interactive | per-load | future round TBD |
| LP-283 | LP › Production › Tabs | All-play, starters by position, starters by player, franchise view | `enum{allplay,position,player,franchise}` | NEW | Read; local UI write where interactive | on-event | local selection |
| LP-284 | LP › Production › All-play rank | Rank among franchises | `rank:int` | NEW | Read; local UI write where interactive | per-load | all-play tab |
| LP-285 | LP › Production › All-play franchise | Franchise name | `{team_id:string,name:string}` | NEW | Read; local UI write where interactive | per-load | all-play tab |
| LP-286 | LP › Production › All-play record | All-play W-L | `{wins:int,losses:int}` | NEW | Read; local UI write where interactive | per-load | mock synthetic |
| LP-287 | LP › Production › All-play percentage | All-play win percentage | `percent:number` | NEW | Read; local UI write where interactive | per-load | all-play tab |
| LP-288 | LP › Production › Actual record | Actual W-L | `{wins:int,losses:int}` | NEW | Read; local UI write where interactive | per-load | all-play tab |
| LP-289 | LP › Production › Luck | Actual wins minus expected wins | `number:number` | NEW | Read; local UI write where interactive | per-load | mock synthetic |
| LP-290 | LP › Production › Position franchise | Franchise name | `{team_id:string,name:string}` | NEW | Read; local UI write where interactive | per-load | position tab |
| LP-291 | LP › Production › Position production | Starter points at QB/RB/WR/TE/OL/DL/LB/DB | `map<string,number>` | NEW | Read; local UI write where interactive | per-load | one cell per position; mock synthetic |
| LP-292 | LP › Production › Player rank | Production rank | `rank:int` | NEW | Read; local UI write where interactive | per-load | player tab |
| LP-293 | LP › Production › Player name | Player name; pin inspector | `{player_id:string,name:string}` | NEW | Read; local UI write where interactive | on-event | player tab |
| LP-294 | LP › Production › Player owner | Fantasy franchise | `team_id:string` | NEW | Read; local UI write where interactive | per-load | player tab |
| LP-295 | LP › Production › Player position | NFL/fantasy position | `position:string` | NEW | Read; local UI write where interactive | per-load | player tab |
| LP-296 | LP › Production › Player points | Season starter points | `points:number` | NEW | Read; local UI write where interactive | per-load | player tab |
| LP-297 | LP › Production › Player starts | Games started | `count:int` | NEW | Read; local UI write where interactive | per-load | player tab |
| LP-298 | LP › Production › Franchise name | Franchise name | `{team_id:string,name:string}` | NEW | Read; local UI write where interactive | per-load | franchise tab |
| LP-299 | LP › Production › Franchise seasons | Number of seasons | `count:int` | NEW | Read; local UI write where interactive | per-load | franchise tab |
| LP-300 | LP › Production › Franchise titles | Championships count | `count:int` | NEW | Read; local UI write where interactive | per-load | franchise tab |
| LP-301 | LP › Production › Franchise playoffs | Playoff appearances count | `count:int` | NEW | Read; local UI write where interactive | per-load | franchise tab |
| LP-302 | LP › Production › Franchise all-time record | All-time wins and losses | `{wins:int,losses:int}` | NEW | Read; local UI write where interactive | per-load | franchise tab |
| LP-303 | LP › Production › Franchise dynasty rank | Dynasty power rank | `rank:int` | NEW | Read; local UI write where interactive | per-load | franchise tab |
| LP-304 | LP › Archive drawer › Week context | Selected franchise and season | `{team_id:string,season:int}` | NEW | Read; local UI write where interactive | on-event | selected archive franchise |
| LP-305 | LP › Archive drawer › Week number | Regular-season week | `week:int` | NEW | Read; local UI write where interactive | per-load | selected season |
| LP-306 | LP › Archive drawer › Week opponent | Opponent franchise name | `{team_id:string,name:string}` | NEW | Read; local UI write where interactive | per-load | per week |
| LP-307 | LP › Archive drawer › Week result | Win/loss or missing | `result:string` | null | Read; local UI write where interactive | NEW | per-load |
| LP-308 | LP › Archive drawer › Week points for | Own fantasy score | `points:number` | null | Read; local UI write where interactive | NEW | per-load |
| LP-309 | LP › Archive drawer › Week points against | Opponent fantasy score | `points:number` | null | Read; local UI write where interactive | NEW | per-load |
| LP-310 | LP › Archive drawer › Week standing | Standing after week | `rank:int` | null | Read; local UI write where interactive | NEW | per-load |
| LP-311 | LP › Trophy case › Group | League, conference, division titles or league records and counts | `{type:string,count:int}` | NEW | Read; local UI write where interactive | per-load | empty groups omitted |
| LP-312 | LP › Trophy case › Trophy year | Award year and glyph | `{year:int,type:string}` | NEW | Read; local UI write where interactive | per-load | each trophy |
| LP-313 | LP › Trophy case › Trophy label | Champion, conference champion, division or record | `label:string` | NEW | Read; local UI write where interactive | per-load | mock hardcodes NFC West |
| LP-314 | LP › Trophy case › Trophy owner | Owner credited; won by you/name | `{owner_id:string,name:string,mine:bool}` | NEW | Read; local UI write where interactive | per-load | per trophy |
| LP-315 | LP › Trophy case › Record value | Record description and numeric value | `{record:string,value:number}` | NEW | Read; local UI write where interactive | per-load | mock only own team |
| LP-316 | LP › Record book › Record name | League record category | `record:string` | NEW | Read; local UI write where interactive | per-load | each record |
| LP-317 | LP › Record book › Holder | Franchise holding record | `{team_id:string,name:string}` | NEW | Read; local UI write where interactive | per-load | each record |
| LP-318 | LP › Record book › Value | Record value | `value:string` | NEW | Read; local UI write where interactive | per-load | different units by record |
| LP-319 | LP › Record book › Season | Season/week or range of record | `{season:int,week:int` | null,end_season:int | Read; local UI write where interactive | null} | NEW |
| LP-320 | LP › Inspector › Pin action | Click or drag team/player into inspector | `{kind:string,id:string}` | NEW | Read; local UI write where interactive | on-event | local selection |
| LP-321 | LP › Inspector › My team button | Reset inspector to own franchise | `team_id:string` | NEW | Read; local UI write where interactive | on-event | local selection |
| LP-322 | LP › Inspector › Accordion controls | Expand/collapse inspector sections | `map<string,bool>` | NEW | Read; local UI write where interactive | on-event | local state |
| LP-323 | LP › Inspector › Franchise name | Pinned franchise and YOU badge | `{team_id:string,name:string,mine:bool}` | NEW | Read; local UI write where interactive | on-event | team inspector |
| LP-324 | LP › Inspector › Franchise metadata | Conference, division, record, seed | `{conference:string,division:string,wins:int,losses:int,seed:int}` | NEW | Read; local UI write where interactive | on-event | team inspector |
| LP-325 | LP › Inspector › Franchise power | Rank and movement | `{rank:int,delta:int}` | PARTIAL: frontend/src/app/race/state.ts:createRaceState lacks mock inspector | Read; local UI write where interactive | per-load | team inspector |
| LP-326 | LP › Inspector › Franchise odds | Playoff probability | `percent:number` | NEW | Read; local UI write where interactive | per-load | team inspector |
| LP-327 | LP › Inspector › This week opponent | Opposing franchise name | `{team_id:string,name:string}` | NEW | Read; local UI write where interactive | on-event | team inspector |
| LP-328 | LP › Inspector › This week scores | Both fantasy totals | `number[2]` | NEW | Read; local UI write where interactive | live (stat feed) | team inspector |
| LP-329 | LP › Inspector › This week win probability | Pinned team win chance | `percent:number` | NEW | Read; local UI write where interactive | live (stat feed) | team inspector |
| LP-330 | LP › Inspector › Season games | Season game count | `count:int` | NEW | Read; local UI write where interactive | per-load | team inspector; mock fixed 8 |
| LP-331 | LP › Inspector › Season points for | Fantasy points scored | `points:number` | NEW | Read; local UI write where interactive | per-load | team inspector |
| LP-332 | LP › Inspector › Season points against | Fantasy points allowed | `points:number` | NEW | Read; local UI write where interactive | per-load | team inspector |
| LP-333 | LP › Inspector › Season all-play | All-play percentage | `percent:number` | NEW | Read; local UI write where interactive | per-load | team inspector |
| LP-334 | LP › Inspector › Season streak | Winning/losing streak | `streak:string` | NEW | Read; local UI write where interactive | per-load | team inspector |
| LP-335 | LP › Inspector › Season chart | Points by week bar chart | `list<{week:int,points:number}>` | NEW | Read; local UI write where interactive | per-load | mock bars synthetic |
| LP-336 | LP › Inspector › Head-to-head record | All-time W-L vs you | `{wins:int,losses:int}` | NEW | Read; local UI write where interactive | per-load | other teams only; synthetic |
| LP-337 | LP › Inspector › Head-to-head last meeting | Week, result and scores | `{week:int,result:string,scores:number[]}` | NEW | Read; local UI write where interactive | per-load | other teams only; hardcoded |
| LP-338 | LP › Inspector › Last five | Week-by-week W/L badges | `list<{week:int,result:string}>` | NEW | Read; local UI write where interactive | per-load | mock synthetic |
| LP-339 | LP › Inspector › History titles | Championship count | `count:int` | NEW | Read; local UI write where interactive | per-load | team inspector |
| LP-340 | LP › Inspector › History finals | Final appearances count | `count:int` | NEW | Read; local UI write where interactive | per-load | team inspector |
| LP-341 | LP › Inspector › History archive link | Open in archive label | `{team_id:string,route:string}` | NEW | Read; local UI write where interactive | on-event | mock label has no handler |
| LP-342 | LP › Inspector › Team action links | Roster and trade desk buttons | `{team_id:string,route:string}` | NEW | Read; local UI write where interactive | on-event | mock buttons do nothing |
| LP-343 | LP › Inspector › Player identity | Player name and YOURS/THEIRS badge | `{player_id:string,name:string,owner_team_id:string` | null} | Read; local UI write where interactive | NEW | on-event |
| LP-344 | LP › Inspector › Player ownership | NFL team and owner or free agent | `{nfl_team:string,owner_team_id:string` | null,owner_name:string | Read; local UI write where interactive | null} | NEW |
| LP-345 | LP › Inspector › Player this week | Current fantasy points and projection | `{points:number,projection:number}` | NEW | Read; local UI write where interactive | live (stat feed) | player inspector |
| LP-346 | LP › Inspector › Player season PPG | Season fantasy points per game | `ppg:number` | NEW | Read; local UI write where interactive | per-load | mock projection times 1.04 |
| LP-347 | LP › Inspector › Player today game | NFL matchup and game clock | `{game_id:string` | null,clock:string | Read; local UI write where interactive | null} | NEW |
| LP-348 | LP › Inspector › Player today state | Done, playing or to play | `state:string` | NEW | Read; local UI write where interactive | live (stat feed) | unknown defaults done |
| LP-349 | LP › Inspector › Player recent plays | Up to three play descriptions and point deltas | `list<{text:string,delta:number}>` | NEW | Read; local UI write where interactive | live (stat feed) | may be empty |
| LP-350 | LP › Inspector › Player last six | Fantasy points bar chart by week | `list<{week:int,points:number}>` | NEW | Read; local UI write where interactive | per-load | mock synthetic heights |
| LP-351 | LP › Inspector › Player contract accordion | Contract/Franchise HQ placeholder | `player_id:string` | NEW | Read; local UI write where interactive | on-event | no body in mock |
| LP-352 | LP › Inspector › Player value accordion | Value/War Room placeholder | `player_id:string` | NEW | Read; local UI write where interactive | on-event | no body in mock |
| LP-353 | LP › Inspector › Player action links | My roster or trade desk; NFL box | `{player_id:string,route:string}` | NEW | Read; local UI write where interactive | on-event | mock buttons do nothing |
| LP-354 | LP › Shell › Lineup and roster shortcut | Shortcut button shown with no handler | `route:string` | NEW | Read; local UI write where interactive | static | mock nonfunctional |
| LP-355 | LP › Shell › Home navigation | Home navigation button with no handler | `route:string` | NEW | Read; local UI write where interactive | static | mock nonfunctional |
| LP-356 | LP › Shell › War Room navigation | War Room navigation button with no handler | `route:string` | NEW | Read; local UI write where interactive | static | mock nonfunctional |
| LP-357 | LP › Shell › Franchise HQ navigation | Franchise HQ navigation button with no handler | `route:string` | NEW | Read; local UI write where interactive | static | mock nonfunctional |
| LP-358 | LP › Shell › Control Room navigation | Control Room navigation button with no handler | `route:string` | NEW | Read; local UI write where interactive | static | mock nonfunctional |
| LP-359 | LP › Shell › Customize | Customize shortcuts button with no handler | `settings:object` | NEW | Read; local UI write where interactive | static | mock nonfunctional |
| LP-360 | LP › Race › Season form component | Last four weeks points versus league; weight 30 | `{rank:int,weight_percent:number,description:string}` | NEW | Read; local UI write where interactive | per-load | season power horizon |
| LP-361 | LP › Race › Season lineup component | Best healthy lineup projection; weight 30 | `{rank:int,weight_percent:number,description:string}` | NEW | Read; local UI write where interactive | per-load | season power horizon |
| LP-362 | LP › Race › Season health component | Starter value on IR/out; weight 15 | `{rank:int,weight_percent:number,description:string}` | NEW | Read; local UI write where interactive | per-load | season power horizon |
| LP-363 | LP › Race › Season all-play component | Record versus every team every week; weight 15 | `{rank:int,weight_percent:number,description:string}` | NEW | Read; local UI write where interactive | per-load | season power horizon |
| LP-364 | LP › Race › Season run-in component | Remaining schedule strength; weight 10 | `{rank:int,weight_percent:number,description:string}` | NEW | Read; local UI write where interactive | per-load | season power horizon |
| LP-365 | LP › Race › Dynasty value component | Best 21 roster value; weight 35 | `{rank:int,weight_percent:number,description:string}` | NEW | Read; local UI write where interactive | per-load | dynasty horizon |
| LP-366 | LP › Race › Dynasty age component | Value-weighted age curve; weight 20 | `{rank:int,weight_percent:number,description:string}` | NEW | Read; local UI write where interactive | per-load | dynasty horizon |
| LP-367 | LP › Race › Dynasty picks component | Next three drafts slot-valued; weight 20 | `{rank:int,weight_percent:number,description:string}` | NEW | Read; local UI write where interactive | per-load | dynasty horizon |
| LP-368 | LP › Race › Dynasty cap component | Cap room versus next year; weight 15 | `{rank:int,weight_percent:number,description:string}` | NEW | Read; local UI write where interactive | per-load | dynasty horizon |
| LP-369 | LP › Race › Dynasty depth component | Value beyond top 21; weight 10 | `{rank:int,weight_percent:number,description:string}` | NEW | Read; local UI write where interactive | per-load | dynasty horizon |

## Shared entities

- **Franchise/team:** LP-001, LP-001, LP-001, LP-001, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-088, LP-088, LP-100, LP-108, LP-110, LP-111, LP-112, LP-119, LP-121, LP-133, LP-144, LP-146, LP-147, LP-149, LP-150, LP-151, LP-153, LP-156, LP-157, LP-158, LP-159, LP-160, LP-161, LP-162, LP-163, LP-164, LP-165, LP-166, LP-172, LP-176, LP-178, LP-179, LP-182, LP-183, LP-189, LP-191, LP-193, LP-198, LP-199, LP-209, LP-210, LP-213, LP-214, LP-215, LP-216, LP-217, LP-218, LP-224, LP-225, LP-237, LP-238, LP-239, LP-240, LP-242, LP-245, LP-247, LP-248, LP-249, LP-250, LP-251, LP-252, LP-254, LP-259, LP-261, LP-264, LP-274, LP-276, LP-279, LP-281, LP-282, LP-283, LP-284, LP-285, LP-290, LP-294, LP-298, LP-299, LP-300, LP-301, LP-302, LP-303, LP-304, LP-306, LP-309, LP-313, LP-314, LP-315, LP-317, LP-320, LP-321, LP-323, LP-324, LP-325, LP-326, LP-327, LP-328, LP-329, LP-330, LP-331, LP-332, LP-333, LP-334, LP-336, LP-337, LP-339, LP-340, LP-341, LP-342, LP-343, LP-344, LP-351, LP-357, LP-363.
- **Player:** LP-001, LP-001, LP-001, LP-001, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-088, LP-088, LP-088, LP-088, LP-202, LP-205, LP-206, LP-207, LP-208, LP-209, LP-210, LP-226, LP-227, LP-231, LP-238, LP-239, LP-240, LP-256, LP-258, LP-263, LP-264, LP-283, LP-291, LP-292, LP-293, LP-294, LP-295, LP-296, LP-297, LP-320, LP-343, LP-344, LP-345, LP-346, LP-347, LP-348, LP-349, LP-350, LP-351, LP-352, LP-353, LP-362.
- **NFL game:** LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-088, LP-088, LP-088, LP-088, LP-088, LP-088, LP-088, LP-101, LP-104, LP-110, LP-112, LP-154, LP-168, LP-196, LP-197, LP-198, LP-199, LP-200, LP-201, LP-233, LP-243, LP-261, LP-262, LP-263, LP-264, LP-265, LP-266, LP-267, LP-268, LP-269, LP-270, LP-271, LP-272, LP-273, LP-274, LP-295, LP-297, LP-330, LP-344, LP-346, LP-347, LP-353.
- **Fantasy matchup:** LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-112, LP-150, LP-151, LP-224, LP-225, LP-237, LP-242, LP-244, LP-260, LP-329, LP-347.
- **Season and week:** LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-045, LP-099, LP-101, LP-102, LP-103, LP-105, LP-109, LP-114, LP-115, LP-116, LP-119, LP-120, LP-124, LP-125, LP-127, LP-129, LP-132, LP-135, LP-136, LP-137, LP-144, LP-148, LP-149, LP-155, LP-156, LP-162, LP-163, LP-164, LP-165, LP-166, LP-167, LP-168, LP-169, LP-170, LP-171, LP-172, LP-173, LP-174, LP-175, LP-176, LP-177, LP-178, LP-179, LP-180, LP-181, LP-182, LP-183, LP-184, LP-185, LP-186, LP-187, LP-188, LP-189, LP-190, LP-191, LP-192, LP-193, LP-194, LP-195, LP-196, LP-197, LP-198, LP-199, LP-200, LP-201, LP-202, LP-203, LP-204, LP-205, LP-206, LP-207, LP-208, LP-209, LP-210, LP-211, LP-212, LP-213, LP-214, LP-215, LP-216, LP-217, LP-218, LP-219, LP-220, LP-221, LP-225, LP-233, LP-234, LP-235, LP-236, LP-237, LP-238, LP-239, LP-240, LP-241, LP-242, LP-243, LP-244, LP-245, LP-246, LP-248, LP-253, LP-255, LP-281, LP-296, LP-299, LP-304, LP-305, LP-306, LP-307, LP-308, LP-309, LP-310, LP-316, LP-317, LP-318, LP-319, LP-327, LP-328, LP-329, LP-330, LP-331, LP-332, LP-333, LP-334, LP-335, LP-337, LP-338, LP-341, LP-345, LP-346, LP-350, LP-360, LP-361, LP-362, LP-363, LP-364.
- **Owner era:** LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-088, LP-088, LP-088, LP-088, LP-088, LP-088, LP-088, LP-088, LP-088, LP-097, LP-098, LP-099, LP-100, LP-101, LP-102, LP-103, LP-104, LP-105, LP-106, LP-107, LP-108, LP-109, LP-110, LP-111, LP-112, LP-113, LP-114, LP-115, LP-116, LP-117, LP-118, LP-119, LP-120, LP-121, LP-122, LP-123, LP-124, LP-125, LP-126, LP-127, LP-128, LP-129, LP-130, LP-131, LP-132, LP-133, LP-134, LP-135, LP-136, LP-137, LP-138, LP-139, LP-140, LP-141, LP-142, LP-143, LP-144, LP-145, LP-146, LP-147, LP-148, LP-149, LP-150, LP-151, LP-152, LP-153, LP-154, LP-155, LP-156, LP-157, LP-158, LP-159, LP-160, LP-161, LP-162, LP-163, LP-164, LP-165, LP-166, LP-167, LP-168, LP-169, LP-170, LP-171, LP-172, LP-173, LP-174, LP-175, LP-176, LP-177, LP-178, LP-179, LP-180, LP-181, LP-182, LP-183, LP-184, LP-185, LP-186, LP-187, LP-188, LP-189, LP-190, LP-191, LP-192, LP-193, LP-194, LP-195, LP-196, LP-197, LP-198, LP-199, LP-200, LP-201, LP-202, LP-203, LP-204, LP-205, LP-206, LP-207, LP-208, LP-209, LP-210, LP-211, LP-212, LP-213, LP-214, LP-215, LP-216, LP-217, LP-218, LP-219, LP-220, LP-221, LP-222, LP-223, LP-224, LP-225, LP-226, LP-227, LP-228, LP-229, LP-230, LP-231, LP-232, LP-233, LP-234, LP-235, LP-236, LP-237, LP-238, LP-239, LP-240, LP-241, LP-242, LP-243, LP-244, LP-245, LP-246, LP-247, LP-248, LP-249, LP-250, LP-251, LP-252, LP-253, LP-254, LP-255, LP-256, LP-257, LP-258, LP-259, LP-260, LP-261, LP-262, LP-263, LP-264, LP-265, LP-266, LP-267, LP-268, LP-269, LP-270, LP-271, LP-272, LP-273, LP-274, LP-275, LP-276, LP-277, LP-278, LP-279, LP-280, LP-281, LP-282, LP-283, LP-284, LP-285, LP-286, LP-287, LP-288, LP-289, LP-290, LP-291, LP-292, LP-293, LP-294, LP-295, LP-296, LP-297, LP-298, LP-299, LP-300, LP-301, LP-302, LP-303, LP-304, LP-305, LP-306, LP-307, LP-308, LP-309, LP-310, LP-311, LP-312, LP-313, LP-314, LP-315, LP-316, LP-317, LP-318, LP-319, LP-320, LP-321, LP-322, LP-323, LP-324, LP-325, LP-326, LP-327, LP-328, LP-329, LP-330, LP-331, LP-332, LP-333, LP-334, LP-335, LP-336, LP-337, LP-338, LP-339, LP-340, LP-341, LP-342, LP-343, LP-344, LP-345, LP-346, LP-347, LP-348, LP-349, LP-350, LP-351, LP-352, LP-353, LP-354, LP-355, LP-356, LP-357, LP-358, LP-359, LP-360, LP-361, LP-362, LP-363, LP-364, LP-365, LP-366, LP-367, LP-368, LP-369.
- **Playoff seed/bracket:** LP-098, LP-099, LP-101, LP-102, LP-103, LP-106, LP-113, LP-131, LP-140, LP-141, LP-144, LP-145, LP-152, LP-155, LP-156, LP-157, LP-158, LP-159, LP-169, LP-175, LP-182, LP-186, LP-190, LP-195, LP-196, LP-197, LP-198, LP-199, LP-200, LP-201, LP-219, LP-279, LP-281, LP-282, LP-301, LP-324, LP-326.
- **Power model:** LP-105, LP-113, LP-114, LP-116, LP-117, LP-118, LP-119, LP-120, LP-121, LP-122, LP-123, LP-124, LP-125, LP-126, LP-127, LP-128, LP-129, LP-130, LP-131, LP-132, LP-133, LP-134, LP-135, LP-136, LP-137, LP-138, LP-139, LP-144, LP-145, LP-146, LP-147, LP-148, LP-149, LP-150, LP-151, LP-152, LP-153, LP-154, LP-303, LP-325, LP-360, LP-361, LP-362, LP-363, LP-364, LP-365, LP-366, LP-367, LP-368, LP-369.
- **Stat correction:** LP-222, LP-255, LP-256, LP-257, LP-258, LP-259, LP-260.
- **Trophy/record:** LP-177, LP-178, LP-202, LP-203, LP-204, LP-311, LP-312, LP-313, LP-314, LP-315, LP-316, LP-317, LP-318, LP-319.
- **Roster/lineup:** LP-001, LP-001, LP-001, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-088, LP-088, LP-088, LP-130, LP-226, LP-227, LP-231, LP-274, LP-283, LP-291, LP-296, LP-342, LP-353, LP-354, LP-361, LP-362, LP-365.
- **Draft pick:** LP-208, LP-211, LP-212, LP-213, LP-214, LP-367.
- **League rule:** LP-275, LP-276, LP-277, LP-278, LP-279, LP-280.

## New data paths

- **Shell:** LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-354, LP-355, LP-356, LP-357, LP-358, LP-359. All rows NEW except those explicitly marked PARTIAL in the table.
- **Strip:** LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001. All rows NEW except those explicitly marked PARTIAL in the table.
- **Matchups:** LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-001, LP-045, LP-045. All rows NEW except those explicitly marked PARTIAL in the table.
- **Live:** LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-045, LP-088, LP-088, LP-088, LP-088, LP-088, LP-088, LP-088, LP-088, LP-088. All rows NEW except those explicitly marked PARTIAL in the table.
- **Race:** LP-097, LP-098, LP-099, LP-100, LP-101, LP-102, LP-103, LP-104, LP-105, LP-106, LP-107, LP-108, LP-109, LP-110, LP-111, LP-112, LP-113, LP-114, LP-115, LP-116, LP-117, LP-118, LP-119, LP-120, LP-121, LP-122, LP-123, LP-124, LP-125, LP-126, LP-127, LP-128, LP-129, LP-130, LP-131, LP-132, LP-133, LP-134, LP-135, LP-136, LP-137, LP-138, LP-139, LP-140, LP-141, LP-142, LP-143, LP-144, LP-145, LP-146, LP-147, LP-148, LP-149, LP-150, LP-151, LP-152, LP-153, LP-154, LP-155, LP-156, LP-157, LP-158, LP-159, LP-160, LP-161, LP-360, LP-361, LP-362, LP-363, LP-364, LP-365, LP-366, LP-367, LP-368, LP-369. All rows NEW except those explicitly marked PARTIAL in the table.
- **Archive:** LP-162, LP-163, LP-164, LP-165, LP-166, LP-167, LP-168, LP-169, LP-170, LP-171, LP-172, LP-173, LP-174, LP-175, LP-176, LP-177, LP-178, LP-179, LP-180, LP-181, LP-182, LP-183, LP-184, LP-185, LP-186, LP-187, LP-188, LP-189, LP-190, LP-191, LP-192, LP-193, LP-194, LP-195, LP-196, LP-197, LP-198, LP-199, LP-200, LP-201, LP-202, LP-203, LP-204, LP-205, LP-206, LP-207, LP-208, LP-209, LP-210, LP-211, LP-212, LP-213, LP-214, LP-215, LP-216, LP-217, LP-218, LP-219, LP-220. All rows NEW except those explicitly marked PARTIAL in the table.
- **Drawer:** LP-221, LP-222, LP-223. All rows NEW except those explicitly marked PARTIAL in the table.
- **Box score:** LP-224, LP-225, LP-226, LP-227, LP-228, LP-229, LP-230, LP-231, LP-232. All rows NEW except those explicitly marked PARTIAL in the table.
- **Week story:** LP-233, LP-234, LP-235, LP-236, LP-237, LP-238, LP-239, LP-240, LP-241, LP-242, LP-243, LP-244, LP-245, LP-246. All rows NEW except those explicitly marked PARTIAL in the table.
- **Schedule:** LP-247, LP-248, LP-249, LP-250, LP-251, LP-252, LP-253, LP-254. All rows NEW except those explicitly marked PARTIAL in the table.
- **Stat changes:** LP-255, LP-256, LP-257, LP-258, LP-259, LP-260. All rows NEW except those explicitly marked PARTIAL in the table.
- **NFL box:** LP-261, LP-262, LP-263, LP-264, LP-265, LP-266, LP-267, LP-268, LP-269, LP-270, LP-271, LP-272, LP-273, LP-274. All rows NEW except those explicitly marked PARTIAL in the table.
- **Tiebreakers:** LP-275, LP-276, LP-277, LP-278, LP-279, LP-280. All rows NEW except those explicitly marked PARTIAL in the table.
- **Bracket:** LP-281, LP-282. All rows NEW except those explicitly marked PARTIAL in the table.
- **Production:** LP-283, LP-284, LP-285, LP-286, LP-287, LP-288, LP-289, LP-290, LP-291, LP-292, LP-293, LP-294, LP-295, LP-296, LP-297, LP-298, LP-299, LP-300, LP-301, LP-302, LP-303. All rows NEW except those explicitly marked PARTIAL in the table.
- **Archive drawer:** LP-304, LP-305, LP-306, LP-307, LP-308, LP-309, LP-310. All rows NEW except those explicitly marked PARTIAL in the table.
- **Trophy case:** LP-311, LP-312, LP-313, LP-314, LP-315. All rows NEW except those explicitly marked PARTIAL in the table.
- **Record book:** LP-316, LP-317, LP-318, LP-319. All rows NEW except those explicitly marked PARTIAL in the table.
- **Inspector:** LP-320, LP-321, LP-322, LP-323, LP-324, LP-325, LP-326, LP-327, LP-328, LP-329, LP-330, LP-331, LP-332, LP-333, LP-334, LP-335, LP-336, LP-337, LP-338, LP-339, LP-340, LP-341, LP-342, LP-343, LP-344, LP-345, LP-346, LP-347, LP-348, LP-349, LP-350, LP-351, LP-352, LP-353. All rows NEW except those explicitly marked PARTIAL in the table.

## Doubts

1. NEW denotes no serving path verified in the limited origin/main files opened; it is not a repository-wide proof of absence. PARTIAL citations establish related code, not end-to-end coverage of the mock.
2. The mock does not distinguish owner, commissioner, co-commissioner, or DOT visibility, nor define authorization or offseason behavior. Actual permissions and unavailable-state handling need specification.
3. Previous `renderArch`, `renderStand`, and `renderRace` declarations are overwritten by later declarations; the older archive timeline tooltip and earlier race layouts are not rendered. The standalone matchups workspace has no visible tab, but its renderer and drawer paths are cataloged.
4. Many values are synthetic or hardcoded: projections, probabilities advertised as simulations, season histories, lineups, statistical lines, record-book values, checks and schedules. Their production algorithms, provenance, and refresh guarantees are unknown.
5. `renderSpine` has no invocation and its target is absent from the current live DOM; its historical play feed is therefore not treated as an on-screen element. The NFL box rows are hardcoded irrespective of selected game.
6. The schedule, archive, and other views have inconsistent mock constants, including 16 results, NFC labels, historical checks and missing import states; league-configurable values and authoritative sources need confirmation.
7. Inspector navigation buttons and shell navigation/customize buttons have no handlers in this mock. Their intended destinations, settings and access controls are unknown.
