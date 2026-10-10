package lab

import (
	"encoding/json"
	"time"
)

type RawTradeColumns struct {
	Ts          []int      `json:"ts"`
	Year        []int      `json:"year"`
	A           []string   `json:"a"`
	B           []string   `json:"b"`
	ARank       []int      `json:"aRank"`
	BRank       []int      `json:"bRank"`
	RankBasis   []string   `json:"rankBasis"`
	ARecord     []*string  `json:"aRecord"`
	BRecord     []*string  `json:"bRecord"`
	APrev       []*int     `json:"aPrev"`
	BPrev       []*int     `json:"bPrev"`
	APlayers    []int      `json:"aPlayers"`
	APicks      []int      `json:"aPicks"`
	BPlayers    []int      `json:"bPlayers"`
	BPicks      []int      `json:"bPicks"`
	AOther      []int      `json:"aOther"`
	BOther      []int      `json:"bOther"`
	OneSided    []bool     `json:"oneSided"`
	Phase       []string   `json:"phase"`
	Week        []int      `json:"week"`
	ToDeadline  []int      `json:"toDeadline"`
	ToDraft     []*int     `json:"toDraft"`
	ACap        []*float64 `json:"aCap"`
	BCap        []*float64 `json:"bCap"`
	AName       []string   `json:"aName"`
	BName       []string   `json:"bName"`
	ANet        []*float64 `json:"aNet"`
	ARealNet    []*float64 `json:"aRealNet"`
	RealSeasons []*int     `json:"realSeasons"`
	Window      []*string  `json:"window"`
	Season      []*int     `json:"season"`
}
type Trade struct {
	Ts          int       `json:"ts"`
	Year        int       `json:"year"`
	A           string    `json:"a"`
	B           string    `json:"b"`
	ARank       int       `json:"aRank"`
	BRank       int       `json:"bRank"`
	RankBasis   string    `json:"rankBasis"`
	ARecord     *string   `json:"aRecord"`
	BRecord     *string   `json:"bRecord"`
	APrev       *int      `json:"aPrev"`
	BPrev       *int      `json:"bPrev"`
	APlayers    int       `json:"aPlayers"`
	APicks      int       `json:"aPicks"`
	BPlayers    int       `json:"bPlayers"`
	BPicks      int       `json:"bPicks"`
	AOther      int       `json:"aOther"`
	BOther      int       `json:"bOther"`
	OneSided    bool      `json:"oneSided"`
	Phase       string    `json:"phase"`
	Week        int       `json:"week"`
	ToDeadline  int       `json:"toDeadline"`
	ToDraft     *int      `json:"toDraft"`
	ACap        *float64  `json:"aCap"`
	BCap        *float64  `json:"bCap"`
	AName       string    `json:"aName"`
	BName       string    `json:"bName"`
	ANet        *float64  `json:"aNet"`
	ARealNet    *float64  `json:"aRealNet"`
	RealSeasons *int      `json:"realSeasons"`
	Window      *string   `json:"window"`
	Season      *int      `json:"season"`
	ID          int       `json:"id"`
	Legs        []Leg     `json:"-"`
	Date        time.Time `json:"-"`
}
type RawLegColumns struct {
	XValue       []*float64 `json:"xValue"`
	XClass       []*string  `json:"xClass"`
	XPrice       []*float64 `json:"xPrice"`
	XReal        []*float64 `json:"xReal"`
	XRealSeasons []*int     `json:"xRealSeasons"`
	Trade        []int      `json:"trade"`
	From         []string   `json:"from"`
	To           []string   `json:"to"`
	FromRank     []int      `json:"fromRank"`
	ToRank       []int      `json:"toRank"`
	FromPrev     []*int     `json:"fromPrev"`
	ToPrev       []*int     `json:"toPrev"`
	Kind         []string   `json:"kind"`
	ContractFrom []*string  `json:"contractFrom"`
	Pid          []*string  `json:"pid"`
	Pos          []*string  `json:"pos"`
	Age          []*float64 `json:"age"`
	Salary       []*float64 `json:"salary"`
	YearsLeft    []*int     `json:"yearsLeft"`
	Contract     []*string  `json:"contract"`
	Tier         []*string  `json:"tier"`
	AfterStarts  []*int     `json:"afterStarts"`
	AfterPts     []*float64 `json:"afterPts"`
	AfterWeeks   []*int     `json:"afterWeeks"`
	PickYear     []*int     `json:"pickYear"`
	Round        []*int     `json:"round"`
	YearsOut     []*int     `json:"yearsOut"`
	Slot         []*int     `json:"slot"`
	Third        []*string  `json:"third"`
	Orig         []*string  `json:"orig"`
	OrigRank     []*int     `json:"origRank"`
	PickPid      []*string  `json:"pickPid"`
}
type Leg struct {
	XValue       *float64 `json:"xValue"`
	XClass       *string  `json:"xClass"`
	XPrice       *float64 `json:"xPrice"`
	XReal        *float64 `json:"xReal"`
	XRealSeasons *int     `json:"xRealSeasons"`
	Trade        int      `json:"trade"`
	From         string   `json:"from"`
	To           string   `json:"to"`
	FromRank     int      `json:"fromRank"`
	ToRank       int      `json:"toRank"`
	FromPrev     *int     `json:"fromPrev"`
	ToPrev       *int     `json:"toPrev"`
	Kind         string   `json:"kind"`
	ContractFrom *string  `json:"contractFrom"`
	Pid          *string  `json:"pid"`
	Pos          *string  `json:"pos"`
	Age          *float64 `json:"age"`
	Salary       *float64 `json:"salary"`
	YearsLeft    *int     `json:"yearsLeft"`
	Contract     *string  `json:"contract"`
	Tier         *string  `json:"tier"`
	AfterStarts  *int     `json:"afterStarts"`
	AfterPts     *float64 `json:"afterPts"`
	AfterWeeks   *int     `json:"afterWeeks"`
	PickYear     *int     `json:"pickYear"`
	Round        *int     `json:"round"`
	YearsOut     *int     `json:"yearsOut"`
	Slot         *int     `json:"slot"`
	Third        *string  `json:"third"`
	Orig         *string  `json:"orig"`
	OrigRank     *int     `json:"origRank"`
	PickPid      *string  `json:"pickPid"`
	Side         int      `json:"side"`
	Player       *Player  `json:"-"`
	PickPlayer   *Player  `json:"-"`
}
type RawDeliveryColumns struct {
	Trade     []int     `json:"trade"`
	Class     []string  `json:"class"`
	Expected  []float64 `json:"expected"`
	Delivered []float64 `json:"delivered"`
	Seasons   []int     `json:"seasons"`
}
type Delivery struct {
	Trade     int     `json:"trade"`
	Class     string  `json:"class"`
	Expected  float64 `json:"expected"`
	Delivered float64 `json:"delivered"`
	Seasons   int     `json:"seasons"`
}
type RawPlayerColumns struct {
	ID       []string  `json:"id"`
	Name     []string  `json:"name"`
	Pos      []*string `json:"pos"`
	Birth    []*int    `json:"birth"`
	NflYear  []*int    `json:"nflYear"`
	NflRound []*int    `json:"nflRound"`
}
type Player struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Pos      *string `json:"pos"`
	Birth    *int    `json:"birth"`
	NflYear  *int    `json:"nflYear"`
	NflRound *int    `json:"nflRound"`
}
type RawDraftPickColumns struct {
	Year           []int        `json:"year"`
	Round          []int        `json:"round"`
	Slot           []int        `json:"slot"`
	Overall        []int        `json:"overall"`
	Fid            []string     `json:"fid"`
	Pid            []string     `json:"pid"`
	Pos            []*string    `json:"pos"`
	Ts             []int        `json:"ts"`
	Group          []*string    `json:"group"`
	StarterSeasons []int        `json:"starterSeasons"`
	Best           []*string    `json:"best"`
	SeasonsSeen    []int        `json:"seasonsSeen"`
	Pts3           []float64    `json:"pts3"`
	NflRound       []*int       `json:"nflRound"`
	NflPick        []*int       `json:"nflPick"`
	Age            []*float64   `json:"age"`
	Rank           []int        `json:"rank"`
	Prev           []int        `json:"prev"`
	KeptYear2      []*bool      `json:"keptYear2"`
	Wins           [][]*float64 `json:"wins"`
	SlotWins       [][]float64  `json:"slotWins"`
}
type DraftPick struct {
	Year           int        `json:"year"`
	Round          int        `json:"round"`
	Slot           int        `json:"slot"`
	Overall        int        `json:"overall"`
	Fid            string     `json:"fid"`
	Pid            string     `json:"pid"`
	Pos            *string    `json:"pos"`
	Ts             int        `json:"ts"`
	Group          *string    `json:"group"`
	StarterSeasons int        `json:"starterSeasons"`
	Best           *string    `json:"best"`
	SeasonsSeen    int        `json:"seasonsSeen"`
	Pts3           float64    `json:"pts3"`
	NflRound       *int       `json:"nflRound"`
	NflPick        *int       `json:"nflPick"`
	Age            *float64   `json:"age"`
	Rank           int        `json:"rank"`
	Prev           int        `json:"prev"`
	KeptYear2      *bool      `json:"keptYear2"`
	Wins           []*float64 `json:"wins"`
	SlotWins       []float64  `json:"slotWins"`
	ID             int        `json:"id"`
	Band           string     `json:"band"`
	Player         *Player    `json:"-"`
}
type RawTeamSeasonColumns struct {
	Year         []int      `json:"year"`
	Fid          []string   `json:"fid"`
	Rank         []*int     `json:"rank"`
	LiveRank     []int      `json:"liveRank"`
	W            []int      `json:"w"`
	L            []int      `json:"l"`
	T            []int      `json:"t"`
	Pf           []float64  `json:"pf"`
	Playoffs     []*bool    `json:"playoffs"`
	Champion     []bool     `json:"champion"`
	Finalist     []bool     `json:"finalist"`
	CapUsed      []float64  `json:"capUsed"`
	Extended     []int      `json:"extended"`
	Tagged       []int      `json:"tagged"`
	StarterAge   []float64  `json:"starterAge"`
	Name         []string   `json:"name"`
	AllPlay      []*float64 `json:"allPlay"`
	ExpectedWins []*float64 `json:"expectedWins"`
	Luck         []*float64 `json:"luck"`
}
type TeamSeason struct {
	Year         int      `json:"year"`
	Fid          string   `json:"fid"`
	Rank         *int     `json:"rank"`
	LiveRank     int      `json:"liveRank"`
	W            int      `json:"w"`
	L            int      `json:"l"`
	T            int      `json:"t"`
	Pf           float64  `json:"pf"`
	Playoffs     *bool    `json:"playoffs"`
	Champion     bool     `json:"champion"`
	Finalist     bool     `json:"finalist"`
	CapUsed      float64  `json:"capUsed"`
	Extended     int      `json:"extended"`
	Tagged       int      `json:"tagged"`
	StarterAge   float64  `json:"starterAge"`
	Name         string   `json:"name"`
	AllPlay      *float64 `json:"allPlay"`
	ExpectedWins *float64 `json:"expectedWins"`
	Luck         *float64 `json:"luck"`
}
type RawOfferColumns struct {
	Ts      []int    `json:"ts"`
	Year    []int    `json:"year"`
	Season  []int    `json:"season"`
	Phase   []string `json:"phase"`
	By      []string `json:"by"`
	To      []string `json:"to"`
	Outcome []string `json:"outcome"`
	Decided []*int   `json:"decided"`
}
type Offer struct {
	Ts      int    `json:"ts"`
	Year    int    `json:"year"`
	Season  int    `json:"season"`
	Phase   string `json:"phase"`
	By      string `json:"by"`
	To      string `json:"to"`
	Outcome string `json:"outcome"`
	Decided *int   `json:"decided"`
}
type RawSeason struct {
	Year              int             `json:"year"`
	RegularWeeks      int             `json:"regularWeeks"`
	WeeksDone         []int           `json:"weeksDone"`
	Cap               json.RawMessage `json:"cap"`
	Champion          json.RawMessage `json:"champion"`
	ContractsRecorded json.RawMessage `json:"contractsRecorded"`
	Era               json.RawMessage `json:"era"`
	Finalist          json.RawMessage `json:"finalist"`
	LastFinishedWeek  json.RawMessage `json:"lastFinishedWeek"`
	Marks             json.RawMessage `json:"marks"`
	PlayoffTeams      json.RawMessage `json:"playoffTeams"`
	RosterSize        json.RawMessage `json:"rosterSize"`
	StarterLine       json.RawMessage `json:"starterLine"`
	Taxi              json.RawMessage `json:"taxi"`
	WeekStart         json.RawMessage `json:"weekStart"`
}
type RawFranchise struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Short       string          `json:"short"`
	Division    json.RawMessage `json:"division"`
	FormerNames json.RawMessage `json:"formerNames"`
}
type CompilerPrice struct {
	ID           string          `json:"id"`
	Price        float64         `json:"price"`
	AvgValue     json.RawMessage `json:"avgValue"`
	Delivered    json.RawMessage `json:"delivered"`
	Early        json.RawMessage `json:"early"`
	Even         json.RawMessage `json:"even"`
	Hi           json.RawMessage `json:"hi"`
	Label        json.RawMessage `json:"label"`
	Late         json.RawMessage `json:"late"`
	Lo           json.RawMessage `json:"lo"`
	N            json.RawMessage `json:"n"`
	Odd          json.RawMessage `json:"odd"`
	PerDelivered json.RawMessage `json:"perDelivered"`
	Season       json.RawMessage `json:"season"`
	Spring       json.RawMessage `json:"spring"`
	Summer       json.RawMessage `json:"summer"`
}
type RawData struct {
	Schema         string         `json:"schema"`
	Built          string         `json:"built"`
	ArchiveThrough string         `json:"archiveThrough"`
	Seasons        []RawSeason    `json:"seasons"`
	Franchises     []RawFranchise `json:"franchises"`
	Market         struct {
		Classes         []CompilerPrice `json:"classes"`
		Anchor          json.RawMessage `json:"anchor"`
		Discount        json.RawMessage `json:"discount"`
		PerAsset        json.RawMessage `json:"perAsset"`
		PerAssetSe      json.RawMessage `json:"perAssetSe"`
		Trades          json.RawMessage `json:"trades"`
		PerCapMillion   json.RawMessage `json:"perCapMillion"`
		PerCapMillionSe json.RawMessage `json:"perCapMillionSe"`
		CapTrades       json.RawMessage `json:"capTrades"`
	} `json:"market"`
	Trades      RawTradeColumns      `json:"trades"`
	Legs        RawLegColumns        `json:"legs"`
	Delivered   RawDeliveryColumns   `json:"delivered"`
	Players     RawPlayerColumns     `json:"players"`
	Draft       RawDraftPickColumns  `json:"draft"`
	TeamSeasons RawTeamSeasonColumns `json:"teamSeasons"`
	Offers      RawOfferColumns      `json:"offers"`
	Eras        json.RawMessage      `json:"eras"`
	Checks      json.RawMessage      `json:"checks"`
	Checks2     json.RawMessage      `json:"checks2"`
	Ledger      json.RawMessage      `json:"ledger"`
	Source      json.RawMessage      `json:"source"`
}
