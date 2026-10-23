package history

import "testing"

// Counts supplied from Claude's MFL archive census, not inferred from native events.
type leagueVolume struct {
	year, transactions, trades, addRows, ir, taxi, loads, lifecycle, picks int
	adds, drops, assets                                                    int
	proposals, accepts, rejects, revokes, expired                          int
}

func legacyVolumes() []leagueVolume {
	return []leagueVolume{
		{2013, 1483, 5, 1451, 24, 0, 0, 0, 1696, 1522, 574, 14, 0, 0, 0, 0, 0},
		{2014, 1059, 88, 904, 39, 0, 8, 0, 170, 908, 910, 306, 0, 0, 0, 0, 0},
		{2015, 1192, 209, 949, 34, 0, 0, 0, 170, 817, 686, 649, 0, 0, 0, 0, 0},
		{2016, 1177, 204, 891, 36, 0, 4, 42, 170, 698, 616, 583, 22, 16, 4, 0, 0},
		{2017, 1461, 255, 997, 34, 0, 140, 35, 176, 422, 593, 656, 19, 10, 3, 3, 0},
		{2018, 1145, 151, 864, 26, 0, 84, 20, 200, 374, 521, 658, 11, 2, 3, 4, 0},
		{2019, 1240, 137, 943, 77, 0, 67, 16, 164, 574, 413, 591, 8, 2, 3, 3, 0},
		{2020, 1303, 226, 842, 90, 0, 115, 30, 178, 559, 341, 922, 15, 4, 3, 8, 0},
		{2021, 1276, 206, 633, 109, 94, 160, 74, 166, 374, 334, 973, 37, 13, 12, 9, 3},
		{2022, 1483, 222, 735, 107, 136, 174, 109, 163, 468, 333, 980, 53, 16, 20, 18, 2},
		{2023, 1680, 316, 703, 87, 133, 212, 229, 163, 419, 300, 1323, 112, 43, 41, 30, 3},
		{2024, 1534, 258, 683, 91, 105, 157, 240, 162, 394, 346, 1101, 120, 30, 44, 46, 0},
		{2025, 1378, 248, 535, 117, 115, 261, 102, 160, 316, 240, 1107, 51, 10, 15, 26, 0},
		{2026, 809, 109, 458, 45, 42, 112, 43, 160, 287, 224, 453, 22, 4, 8, 9, 0},
	}
}

// thickVolumes is the startup (seed only, no moves) followed by the three busiest real seasons,
// 2022-2024 (4,697 transactions), replayed as 2014-2016 so rollover, picks and contracts line up.
func thickVolumes() []leagueVolume {
	legacy := legacyVolumes()
	out := []leagueVolume{{year: 2021, picks: legacy[0].picks}}
	return append(out, legacy[9:12]...)
}

func TestBigLeagueVolumeModel(t *testing.T) {
	got := thickVolumes()
	if len(got) != 4 || got[0].picks != 1696 || got[0].adds != 0 || got[0].year != 2021 || got[3].year != 2024 {
		t.Fatal("thick tier must be one seed-only startup and three seasons")
	}
	var transactions int
	for _, v := range got {
		transactions += v.transactions
	}
	if transactions != 1483+1680+1534 {
		t.Fatalf("thick tier transactions=%d, want the 2022-2024 total", transactions)
	}
}
