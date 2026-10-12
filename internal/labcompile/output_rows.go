package labcompile

type tradeRow struct {
	Ts          int64    `json:"ts"`
	Year        int64    `json:"year"`
	A           string   `json:"a"`
	B           string   `json:"b"`
	ARank       *int64   `json:"aRank"`
	BRank       *int64   `json:"bRank"`
	RankBasis   *string  `json:"rankBasis"`
	ARecord     *string  `json:"aRecord"`
	BRecord     *string  `json:"bRecord"`
	APrev       *int64   `json:"aPrev"`
	BPrev       *int64   `json:"bPrev"`
	APlayers    int64    `json:"aPlayers"`
	APicks      int64    `json:"aPicks"`
	BPlayers    int64    `json:"bPlayers"`
	BPicks      int64    `json:"bPicks"`
	AOther      int64    `json:"aOther"`
	BOther      int64    `json:"bOther"`
	OneSided    bool     `json:"oneSided"`
	Phase       string   `json:"phase"`
	Week        int64    `json:"week"`
	ToDeadline  *int64   `json:"toDeadline"`
	ToDraft     *int64   `json:"toDraft"`
	ACap        *float64 `json:"aCap"`
	BCap        *float64 `json:"bCap"`
	AName       *string  `json:"aName"`
	BName       *string  `json:"bName"`
	ANet        *float64 `json:"aNet"`
	ARealNet    *float64 `json:"aRealNet"`
	RealSeasons *int64   `json:"realSeasons"`
	Window      *string  `json:"window"`
	Season      *int64   `json:"season"`
}

type legRow struct {
	XValue       *float64 `json:"xValue"`
	XClass       *string  `json:"xClass"`
	XPrice       *float64 `json:"xPrice"`
	XReal        *float64 `json:"xReal"`
	XRealSeasons *int64   `json:"xRealSeasons"`
	Trade        int64    `json:"trade"`
	From         string   `json:"from"`
	To           string   `json:"to"`
	FromRank     *int64   `json:"fromRank"`
	ToRank       *int64   `json:"toRank"`
	FromPrev     *int64   `json:"fromPrev"`
	ToPrev       *int64   `json:"toPrev"`
	Kind         string   `json:"kind"`
	ContractFrom *string  `json:"contractFrom"`
	Pid          *string  `json:"pid"`
	Pos          *string  `json:"pos"`
	Age          *float64 `json:"age"`
	Salary       *float64 `json:"salary"`
	YearsLeft    *int64   `json:"yearsLeft"`
	Contract     *string  `json:"contract"`
	Tier         *string  `json:"tier"`
	AfterStarts  *int64   `json:"afterStarts"`
	AfterPts     *float64 `json:"afterPts"`
	AfterWeeks   *int64   `json:"afterWeeks"`
	PickYear     *int64   `json:"pickYear"`
	Round        *int64   `json:"round"`
	YearsOut     *int64   `json:"yearsOut"`
	Slot         *int64   `json:"slot"`
	Third        *string  `json:"third"`
	Orig         *string  `json:"orig"`
	OrigRank     *int64   `json:"origRank"`
	PickPid      *string  `json:"pickPid"`
}

type draftRow struct {
	Year           int64      `json:"year"`
	Round          int64      `json:"round"`
	Slot           int64      `json:"slot"`
	Overall        int64      `json:"overall"`
	Fid            string     `json:"fid"`
	Pid            string     `json:"pid"`
	Pos            *string    `json:"pos"`
	Ts             int64      `json:"ts"`
	Group          *string    `json:"group"`
	StarterSeasons int64      `json:"starterSeasons"`
	Best           *string    `json:"best"`
	SeasonsSeen    int64      `json:"seasonsSeen"`
	Pts3           float64    `json:"pts3"`
	NflRound       *int64     `json:"nflRound"`
	NflPick        *int64     `json:"nflPick"`
	Age            *float64   `json:"age"`
	Rank           *int64     `json:"rank"`
	Prev           *int64     `json:"prev"`
	KeptYear2      *bool      `json:"keptYear2"`
	Wins           []*float64 `json:"wins"`
	SlotWins       []*float64 `json:"slotWins"`
}

type teamRow struct {
	Year         int64    `json:"year"`
	Fid          string   `json:"fid"`
	Rank         *int64   `json:"rank"`
	LiveRank     *int64   `json:"liveRank"`
	W            *int64   `json:"w"`
	L            *int64   `json:"l"`
	T            *int64   `json:"t"`
	Pf           *float64 `json:"pf"`
	Playoffs     *bool    `json:"playoffs"`
	Champion     bool     `json:"champion"`
	Finalist     bool     `json:"finalist"`
	CapUsed      *float64 `json:"capUsed"`
	Extended     *int64   `json:"extended"`
	Tagged       *int64   `json:"tagged"`
	StarterAge   *float64 `json:"starterAge"`
	Name         *string  `json:"name"`
	AllPlay      *float64 `json:"allPlay"`
	ExpectedWins *float64 `json:"expectedWins"`
	Luck         *float64 `json:"luck"`
}

type playerRow struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Pos      *string `json:"pos"`
	Birth    *int64  `json:"birth"`
	NflYear  *int64  `json:"nflYear"`
	NflRound *int64  `json:"nflRound"`
}

type deliveryRow struct {
	Trade     int64   `json:"trade"`
	Class     string  `json:"class"`
	Expected  float64 `json:"expected"`
	Delivered float64 `json:"delivered"`
	Seasons   int64   `json:"seasons"`
}
