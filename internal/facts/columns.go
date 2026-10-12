// Package facts compiles pinned archive bodies into deterministic in-memory league columns.
package facts

type factRows struct {
	Season          []Season
	FranchiseSeason []FranchiseSeason
	Standings       []Standings
	ScoringRule     []ScoringRule
	RuleCode        []RuleCode
	CalendarEvent   []CalendarEvent
	Player          []Player
	PlayerConflict  []PlayerConflict
	PlayerSeason    []PlayerSeason
	SourceFile      []SourceFile
	LoadNote        []LoadNote
	NoteReading     []NoteReading
	Game            []Game
	Lineup          []Lineup
	PlayerWeek      []PlayerWeek
	InjuryWeek      []InjuryWeek
	RosterWeek      []RosterWeek
	NflGame         []NflGame
	PointsAllowed   []PointsAllowed
	PlayoffBracket  []PlayoffBracket
	PlayoffGame     []PlayoffGame
}

type Season struct {
	Year            int64    `json:"year"`
	LeagueID        string   `json:"league_id"`
	Name            *string  `json:"name"`
	StartWeek       *int64   `json:"start_week"`
	EndWeek         *int64   `json:"end_week"`
	LastRegularWeek *int64   `json:"last_regular_week"`
	SalaryCap       *float64 `json:"salary_cap"`
	RosterSize      *int64   `json:"roster_size"`
	TaxiSquad       *string  `json:"taxi_squad"`
	InjuredReserve  *string  `json:"injured_reserve"`
	Starters        *string  `json:"starters"`
	Raw             string   `json:"raw"`
	Src             string   `json:"src"`
}

type FranchiseSeason struct {
	Year           int64   `json:"year"`
	Fid            string  `json:"fid"`
	Name           *string `json:"name"`
	Abbrev         *string `json:"abbrev"`
	Division       *string `json:"division"`
	DivisionName   *string `json:"division_name"`
	Conference     *string `json:"conference"`
	ConferenceName *string `json:"conference_name"`
	Src            string  `json:"src"`
}

type Standings struct {
	Year     int64    `json:"year"`
	Fid      string   `json:"fid"`
	H2hW     *int64   `json:"h2h_w"`
	H2hL     *int64   `json:"h2h_l"`
	H2hT     *int64   `json:"h2h_t"`
	Pf       *float64 `json:"pf"`
	Pa       *float64 `json:"pa"`
	Pp       *float64 `json:"pp"`
	AllPlayW *int64   `json:"all_play_w"`
	AllPlayL *int64   `json:"all_play_l"`
	AllPlayT *int64   `json:"all_play_t"`
	Eff      *float64 `json:"eff"`
	Raw      string   `json:"raw"`
	Src      string   `json:"src"`
}

type ScoringRule struct {
	Year      int64   `json:"year"`
	Positions string  `json:"positions"`
	Event     string  `json:"event"`
	Range     *string `json:"range"`
	Points    *string `json:"points"`
	Src       string  `json:"src"`
}

type RuleCode struct {
	Code        string  `json:"code"`
	Short       *string `json:"short"`
	Description *string `json:"description"`
	Src         string  `json:"src"`
}

type CalendarEvent struct {
	Year    int64   `json:"year"`
	EventID string  `json:"event_id"`
	Type    *string `json:"type"`
	Title   *string `json:"title"`
	StartTs *int64  `json:"start_ts"`
	EndTs   *int64  `json:"end_ts"`
	Src     string  `json:"src"`
}

type Player struct {
	Pid         string  `json:"pid"`
	Name        string  `json:"name"`
	Position    *string `json:"position"`
	Birthdate   *string `json:"birthdate"`
	BirthdateTs *int64  `json:"birthdate_ts"`
	DraftYear   *int64  `json:"draft_year"`
	DraftTeam   *string `json:"draft_team"`
	DraftRound  *int64  `json:"draft_round"`
	DraftPick   *int64  `json:"draft_pick"`
	College     *string `json:"college"`
	Height      *string `json:"height"`
	Weight      *string `json:"weight"`
	FirstYear   *int64  `json:"first_year"`
	LastYear    *int64  `json:"last_year"`
	Src         string  `json:"src"`
}

type PlayerConflict struct {
	Pid   string  `json:"pid"`
	Field string  `json:"field"`
	Year  int64   `json:"year"`
	Value *string `json:"value"`
	Kept  *string `json:"kept"`
}

type PlayerSeason struct {
	Year     int64   `json:"year"`
	Pid      string  `json:"pid"`
	Name     *string `json:"name"`
	Position *string `json:"position"`
	NflTeam  *string `json:"nfl_team"`
	Src      string  `json:"src"`
}

type SourceFile struct {
	Src    string `json:"src"`
	Bytes  int64  `json:"bytes"`
	Sha256 string `json:"sha256"`
}

type LoadNote struct {
	Step string  `json:"step"`
	Src  *string `json:"src"`
	Note string  `json:"note"`
	N    int64   `json:"n"`
}

type NoteReading struct {
	TxnID      int64    `json:"txn_id"`
	FromFid    *string  `json:"from_fid"`
	ToFid      *string  `json:"to_fid"`
	Kind       string   `json:"kind"`
	PickYear   *int64   `json:"pick_year"`
	PickRound  *int64   `json:"pick_round"`
	PickSlot   *int64   `json:"pick_slot"`
	PickOrig   *string  `json:"pick_orig"`
	CapM       *float64 `json:"cap_m"`
	CapSeasons *string  `json:"cap_seasons"`
	Player     *string  `json:"player"`
	Clarity    string   `json:"clarity"`
	Reading    string   `json:"reading"`
	DraftCheck *string  `json:"draft_check"`
	Confirmed  int64    `json:"confirmed"`
}

type Facts struct {
	Season          SeasonColumns
	FranchiseSeason FranchiseSeasonColumns
	Standings       StandingsColumns
	ScoringRule     ScoringRuleColumns
	RuleCode        RuleCodeColumns
	CalendarEvent   CalendarEventColumns
	Player          PlayerColumns
	PlayerConflict  PlayerConflictColumns
	PlayerSeason    PlayerSeasonColumns
	SourceFile      SourceFileColumns
	LoadNote        LoadNoteColumns
	NoteReading     NoteReadingColumns
	Game            GameColumns
	Lineup          LineupColumns
	PlayerWeek      PlayerWeekColumns
	InjuryWeek      InjuryWeekColumns
	RosterWeek      RosterWeekColumns
	NflGame         NflGameColumns
	PointsAllowed   PointsAllowedColumns
	PlayoffBracket  PlayoffBracketColumns
	PlayoffGame     PlayoffGameColumns
}

type SeasonColumns struct {
	Year            []int64    `json:"year"`
	LeagueID        []string   `json:"league_id"`
	Name            []*string  `json:"name"`
	StartWeek       []*int64   `json:"start_week"`
	EndWeek         []*int64   `json:"end_week"`
	LastRegularWeek []*int64   `json:"last_regular_week"`
	SalaryCap       []*float64 `json:"salary_cap"`
	RosterSize      []*int64   `json:"roster_size"`
	TaxiSquad       []*string  `json:"taxi_squad"`
	InjuredReserve  []*string  `json:"injured_reserve"`
	Starters        []*string  `json:"starters"`
	Raw             []string   `json:"raw"`
	Src             []string   `json:"src"`
}

type FranchiseSeasonColumns struct {
	Year           []int64   `json:"year"`
	Fid            []string  `json:"fid"`
	Name           []*string `json:"name"`
	Abbrev         []*string `json:"abbrev"`
	Division       []*string `json:"division"`
	DivisionName   []*string `json:"division_name"`
	Conference     []*string `json:"conference"`
	ConferenceName []*string `json:"conference_name"`
	Src            []string  `json:"src"`
}

type StandingsColumns struct {
	Year     []int64    `json:"year"`
	Fid      []string   `json:"fid"`
	H2hW     []*int64   `json:"h2h_w"`
	H2hL     []*int64   `json:"h2h_l"`
	H2hT     []*int64   `json:"h2h_t"`
	Pf       []*float64 `json:"pf"`
	Pa       []*float64 `json:"pa"`
	Pp       []*float64 `json:"pp"`
	AllPlayW []*int64   `json:"all_play_w"`
	AllPlayL []*int64   `json:"all_play_l"`
	AllPlayT []*int64   `json:"all_play_t"`
	Eff      []*float64 `json:"eff"`
	Raw      []string   `json:"raw"`
	Src      []string   `json:"src"`
}

type ScoringRuleColumns struct {
	Year      []int64   `json:"year"`
	Positions []string  `json:"positions"`
	Event     []string  `json:"event"`
	Range     []*string `json:"range"`
	Points    []*string `json:"points"`
	Src       []string  `json:"src"`
}

type RuleCodeColumns struct {
	Code        []string  `json:"code"`
	Short       []*string `json:"short"`
	Description []*string `json:"description"`
	Src         []string  `json:"src"`
}

type CalendarEventColumns struct {
	Year    []int64   `json:"year"`
	EventID []string  `json:"event_id"`
	Type    []*string `json:"type"`
	Title   []*string `json:"title"`
	StartTs []*int64  `json:"start_ts"`
	EndTs   []*int64  `json:"end_ts"`
	Src     []string  `json:"src"`
}

type PlayerColumns struct {
	Pid         []string  `json:"pid"`
	Name        []string  `json:"name"`
	Position    []*string `json:"position"`
	Birthdate   []*string `json:"birthdate"`
	BirthdateTs []*int64  `json:"birthdate_ts"`
	DraftYear   []*int64  `json:"draft_year"`
	DraftTeam   []*string `json:"draft_team"`
	DraftRound  []*int64  `json:"draft_round"`
	DraftPick   []*int64  `json:"draft_pick"`
	College     []*string `json:"college"`
	Height      []*string `json:"height"`
	Weight      []*string `json:"weight"`
	FirstYear   []*int64  `json:"first_year"`
	LastYear    []*int64  `json:"last_year"`
	Src         []string  `json:"src"`
}

type PlayerConflictColumns struct {
	Pid   []string  `json:"pid"`
	Field []string  `json:"field"`
	Year  []int64   `json:"year"`
	Value []*string `json:"value"`
	Kept  []*string `json:"kept"`
}

type PlayerSeasonColumns struct {
	Year     []int64   `json:"year"`
	Pid      []string  `json:"pid"`
	Name     []*string `json:"name"`
	Position []*string `json:"position"`
	NflTeam  []*string `json:"nfl_team"`
	Src      []string  `json:"src"`
}

type SourceFileColumns struct {
	Src    []string `json:"src"`
	Bytes  []int64  `json:"bytes"`
	Sha256 []string `json:"sha256"`
}

type LoadNoteColumns struct {
	Step []string  `json:"step"`
	Src  []*string `json:"src"`
	Note []string  `json:"note"`
	N    []int64   `json:"n"`
}

type NoteReadingColumns struct {
	TxnID      []int64    `json:"txn_id"`
	FromFid    []*string  `json:"from_fid"`
	ToFid      []*string  `json:"to_fid"`
	Kind       []string   `json:"kind"`
	PickYear   []*int64   `json:"pick_year"`
	PickRound  []*int64   `json:"pick_round"`
	PickSlot   []*int64   `json:"pick_slot"`
	PickOrig   []*string  `json:"pick_orig"`
	CapM       []*float64 `json:"cap_m"`
	CapSeasons []*string  `json:"cap_seasons"`
	Player     []*string  `json:"player"`
	Clarity    []string   `json:"clarity"`
	Reading    []string   `json:"reading"`
	DraftCheck []*string  `json:"draft_check"`
	Confirmed  []int64    `json:"confirmed"`
}

type Game struct {
	Year   int64    `json:"year"`
	Week   int64    `json:"week"`
	Fid    string   `json:"fid"`
	Opp    *string  `json:"opp"`
	IsHome *int64   `json:"is_home"`
	Score  *float64 `json:"score"`
	Result *string  `json:"result"`
	OptPts *float64 `json:"opt_pts"`
	Src    string   `json:"src"`
}

type Lineup struct {
	Year        int64    `json:"year"`
	Week        int64    `json:"week"`
	Fid         string   `json:"fid"`
	Pid         string   `json:"pid"`
	Started     int64    `json:"started"`
	ShouldStart *int64   `json:"should_start"`
	Score       *float64 `json:"score"`
	Src         string   `json:"src"`
}

type PlayerWeek struct {
	Year      int64    `json:"year"`
	Week      int64    `json:"week"`
	Pid       string   `json:"pid"`
	Score     *float64 `json:"score"`
	Projected *float64 `json:"projected"`
}

type InjuryWeek struct {
	Year      int64   `json:"year"`
	Week      int64   `json:"week"`
	Pid       string  `json:"pid"`
	Status    *string `json:"status"`
	Details   *string `json:"details"`
	ExpReturn *string `json:"exp_return"`
	Src       string  `json:"src"`
}

type RosterWeek struct {
	Year           int64    `json:"year"`
	Week           int64    `json:"week"`
	Fid            string   `json:"fid"`
	Pid            string   `json:"pid"`
	Status         *string  `json:"status"`
	Salary         *float64 `json:"salary"`
	ContractYear   *string  `json:"contract_year"`
	ContractStatus *string  `json:"contract_status"`
	ContractInfo   *string  `json:"contract_info"`
	Src            string   `json:"src"`
}

type NflGame struct {
	Year    int64    `json:"year"`
	Week    int64    `json:"week"`
	Team    string   `json:"team"`
	Opp     *string  `json:"opp"`
	IsHome  *int64   `json:"is_home"`
	Score   *int64   `json:"score"`
	Spread  *float64 `json:"spread"`
	Kickoff *int64   `json:"kickoff"`
	Src     string   `json:"src"`
}

type PointsAllowed struct {
	Year     int64    `json:"year"`
	NflTeam  string   `json:"nfl_team"`
	Position string   `json:"position"`
	Points   *float64 `json:"points"`
	Src      string   `json:"src"`
}

type PlayoffBracket struct {
	Year        int64   `json:"year"`
	BracketID   string  `json:"bracket_id"`
	Name        *string `json:"name"`
	WinnerTitle *string `json:"winner_title"`
	StartWeek   *int64  `json:"start_week"`
	Teams       *int64  `json:"teams"`
	Src         string  `json:"src"`
}

type PlayoffGame struct {
	Year      int64    `json:"year"`
	BracketID string   `json:"bracket_id"`
	Week      *int64   `json:"week"`
	GameID    *string  `json:"game_id"`
	Fid       *string  `json:"fid"`
	Side      *string  `json:"side"`
	Points    *float64 `json:"points"`
	Seed      *string  `json:"seed"`
	Src       string   `json:"src"`
}

type GameColumns struct {
	Year   []int64    `json:"year"`
	Week   []int64    `json:"week"`
	Fid    []string   `json:"fid"`
	Opp    []*string  `json:"opp"`
	IsHome []*int64   `json:"is_home"`
	Score  []*float64 `json:"score"`
	Result []*string  `json:"result"`
	OptPts []*float64 `json:"opt_pts"`
	Src    []string   `json:"src"`
}

type LineupColumns struct {
	Year        []int64    `json:"year"`
	Week        []int64    `json:"week"`
	Fid         []string   `json:"fid"`
	Pid         []string   `json:"pid"`
	Started     []int64    `json:"started"`
	ShouldStart []*int64   `json:"should_start"`
	Score       []*float64 `json:"score"`
	Src         []string   `json:"src"`
}

type PlayerWeekColumns struct {
	Year      []int64    `json:"year"`
	Week      []int64    `json:"week"`
	Pid       []string   `json:"pid"`
	Score     []*float64 `json:"score"`
	Projected []*float64 `json:"projected"`
}

type InjuryWeekColumns struct {
	Year      []int64   `json:"year"`
	Week      []int64   `json:"week"`
	Pid       []string  `json:"pid"`
	Status    []*string `json:"status"`
	Details   []*string `json:"details"`
	ExpReturn []*string `json:"exp_return"`
	Src       []string  `json:"src"`
}

type RosterWeekColumns struct {
	Year           []int64    `json:"year"`
	Week           []int64    `json:"week"`
	Fid            []string   `json:"fid"`
	Pid            []string   `json:"pid"`
	Status         []*string  `json:"status"`
	Salary         []*float64 `json:"salary"`
	ContractYear   []*string  `json:"contract_year"`
	ContractStatus []*string  `json:"contract_status"`
	ContractInfo   []*string  `json:"contract_info"`
	Src            []string   `json:"src"`
}

type NflGameColumns struct {
	Year    []int64    `json:"year"`
	Week    []int64    `json:"week"`
	Team    []string   `json:"team"`
	Opp     []*string  `json:"opp"`
	IsHome  []*int64   `json:"is_home"`
	Score   []*int64   `json:"score"`
	Spread  []*float64 `json:"spread"`
	Kickoff []*int64   `json:"kickoff"`
	Src     []string   `json:"src"`
}

type PointsAllowedColumns struct {
	Year     []int64    `json:"year"`
	NflTeam  []string   `json:"nfl_team"`
	Position []string   `json:"position"`
	Points   []*float64 `json:"points"`
	Src      []string   `json:"src"`
}

type PlayoffBracketColumns struct {
	Year        []int64   `json:"year"`
	BracketID   []string  `json:"bracket_id"`
	Name        []*string `json:"name"`
	WinnerTitle []*string `json:"winner_title"`
	StartWeek   []*int64  `json:"start_week"`
	Teams       []*int64  `json:"teams"`
	Src         []string  `json:"src"`
}

type PlayoffGameColumns struct {
	Year      []int64    `json:"year"`
	BracketID []string   `json:"bracket_id"`
	Week      []*int64   `json:"week"`
	GameID    []*string  `json:"game_id"`
	Fid       []*string  `json:"fid"`
	Side      []*string  `json:"side"`
	Points    []*float64 `json:"points"`
	Seed      []*string  `json:"seed"`
	Src       []string   `json:"src"`
}
