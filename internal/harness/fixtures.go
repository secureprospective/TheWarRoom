package harness

import (
	"github.com/secureprospective/TheWarRoom/internal/composition"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
)

// SampleRookies is the versioned hardcoded scaffold for Module 1: a synthetic, clearly-
// labelled set spanning several position groups with varied ages, RAS, and base points so
// the rankings table and every engine intermediate are visible end to end. It is NOT real
// 2026 draft data — names are deliberately generic ("WR Alpha") so nothing here is mistaken
// for a scouting claim. When the MFL pull and real fixtures land they REPLACE this; until
// then it is the deterministic input that exercises the composition boundary + pipeline.
//
// Positions with a REGISTERED B5b rubric (QB today) now differentiate on their L4
// scouting sub-signals; positions still on identity L4 ignore the scouting fields (which
// are zero/absent here) and rank on BasePoints × AgePull × CapMultiplier — the scouting
// baseline. The two QBs carry contrasting breakout profiles so the QB rubric visibly
// separates them once registered. Film is absent (HasFilm false — no source until the film
// calibration pass), so the QB rubric forces a neutral film component (Data-Parity Rule).
func SampleRookies() []composition.PlayerSpec {
	return []composition.PlayerSpec{
		// Elite breakout: true-freshman starter, Power Four, dominant college share.
		{MFLID: "0101", Name: "QB Alpha", Position: domain.PosQB, BasePoints: 280, Age: 22, RAS: 8.1, HasRAS: true, Salary: 12,
			BreakoutAge: 20, HasBreakoutAge: true, SchoolTier: scouting.SchoolPowerFour, CollegeShare: 0.70, HasCollegeShare: true},
		// Weaker breakout: later starter, Group of Five, moderate share.
		{MFLID: "0102", Name: "QB Bravo", Position: domain.PosQB, BasePoints: 240, Age: 23, RAS: 6.0, HasRAS: true, Salary: 6,
			BreakoutAge: 22, HasBreakoutAge: true, SchoolTier: scouting.SchoolGroupOfFive, CollegeShare: 0.50, HasCollegeShare: true},
		// Two RBs with contrasting profiles so the B5b-RB rubric visibly separates them (active
		// Medium-tier RAS + breakout; film Data-Parity neutral). RB Alpha: elite athlete, early
		// breakout, Power Four, dominant college workload. RB Bravo: thin profile — later
		// breakout, Group of Five, low workload, older — pulls Layer 4 below 1.000 (the §7
		// Herbert pattern, breakout-driven while film is neutral).
		{MFLID: "0201", Name: "RB Alpha", Position: domain.PosRB, BasePoints: 210, Age: 21, RAS: 9.4, HasRAS: true, Salary: 9,
			BreakoutAge: 19, HasBreakoutAge: true, SchoolTier: scouting.SchoolPowerFour, CollegeShare: 0.45, HasCollegeShare: true},
		{MFLID: "0202", Name: "RB Bravo", Position: domain.PosRB, BasePoints: 195, Age: 28, RAS: 6.0, HasRAS: true, Salary: 4,
			BreakoutAge: 22, HasBreakoutAge: true, SchoolTier: scouting.SchoolGroupOfFive, CollegeShare: 0.18, HasCollegeShare: true},
		// Three WRs with contrasting profiles so the B5b-WR rubric visibly separates them (active
		// High-tier RAS + breakout; film Data-Parity neutral — offense film weights unset). WR
		// Alpha: elite athlete, true-freshman breakout, Power Four, dominant target share. WR
		// Bravo: RAS ABSENT (Data-Parity neutral RAS) but a solid draft profile. WR Charlie: thin
		// profile — late breakout, smaller school, low usage, older — but the static signals keep
		// his breakout from collapsing the way RB Bravo's does (the §5 Lockett structural point).
		{MFLID: "0301", Name: "WR Alpha", Position: domain.PosWR, BasePoints: 250, Age: 21, RAS: 9.8, HasRAS: true, Salary: 14,
			BreakoutAge: 19, HasBreakoutAge: true, SchoolTier: scouting.SchoolPowerFour, CollegeShare: 0.40, HasCollegeShare: true},
		{MFLID: "0302", Name: "WR Bravo", Position: domain.PosWR, BasePoints: 220, Age: 22, RAS: 0, HasRAS: false, Salary: 7,
			BreakoutAge: 21, HasBreakoutAge: true, SchoolTier: scouting.SchoolGroupOfFive, CollegeShare: 0.22, HasCollegeShare: true},
		{MFLID: "0303", Name: "WR Charlie", Position: domain.PosWR, BasePoints: 180, Age: 23, RAS: 5.5, HasRAS: true, Salary: 2,
			BreakoutAge: 22, HasBreakoutAge: true, SchoolTier: scouting.SchoolFCS, CollegeShare: 0.16, HasCollegeShare: true},
		// Two aging TEs with contrasting athletic-and-draft profiles so the B5b-TE rubric's
		// SL-019 RAS-modulator visibly separates them (the §5 Higbee/Henry structural finding):
		// past-peak age, but the athletic vet's high RAS + strong static signals keep his breakout
		// up while the non-athletic vet's low RAS + weak static signals do not. TE Alpha: athletic
		// vet — high RAS, early breakout, Power Four, strong target share (Henry-like).
		{MFLID: "0401", Name: "TE Alpha", Position: domain.PosTE, BasePoints: 160, Age: 31, RAS: 9.2, HasRAS: true, Salary: 5,
			BreakoutAge: 20, HasBreakoutAge: true, SchoolTier: scouting.SchoolPowerFour, CollegeShare: 0.22, HasCollegeShare: true},
		// TE Bravo: non-athletic vet — low RAS, late breakout, Group of Five, thin target share
		// (Higbee-like). SL-019 gives him little modulator help; his breakout stays below Alpha's.
		{MFLID: "0402", Name: "TE Bravo", Position: domain.PosTE, BasePoints: 150, Age: 31, RAS: 6.0, HasRAS: true, Salary: 3,
			BreakoutAge: 22, HasBreakoutAge: true, SchoolTier: scouting.SchoolGroupOfFive, CollegeShare: 0.15, HasCollegeShare: true},
		// Two DEs with contrasting athletic-and-draft profiles so the B5b-DE rubric's SL-019
		// RAS-modulator visibly separates them (the §5 Garrett/Lawrence finding — the SECOND
		// SL-019 instance, reusing curve.SL019). DE Alpha: athletic profile — high RAS, early
		// breakout, Power Four, dominant Sack+TFL market share (Garrett-like).
		{MFLID: "0501", Name: "DE Alpha", Position: domain.PosDE, BasePoints: 175, Age: 22, RAS: 9.1, HasRAS: true, Salary: 8,
			BreakoutAge: 19, HasBreakoutAge: true, SchoolTier: scouting.SchoolPowerFour, CollegeShare: 0.30, HasCollegeShare: true},
		// DE Bravo: non-athletic profile — low RAS, late breakout, Group of Five, thin share
		// (Lawrence-like). SL-019 gives him little modulator help; his breakout stays below Alpha's.
		{MFLID: "0502", Name: "DE Bravo", Position: domain.PosDE, BasePoints: 165, Age: 23, RAS: 6.0, HasRAS: true, Salary: 4,
			BreakoutAge: 21, HasBreakoutAge: true, SchoolTier: scouting.SchoolGroupOfFive, CollegeShare: 0.15, HasCollegeShare: true},
		{MFLID: "0601", Name: "CB Alpha", Position: domain.PosCB, BasePoints: 165, Age: 21, RAS: 8.9, HasRAS: true, Salary: 6},
		// Two DTs with contrasting RAS + breakout so the B5b-DT rubric visibly separates them
		// (active RAS component + breakout; film is Data-Parity neutral — IDP weights unset).
		// DT Alpha: elite athlete, early breakout, dominant TFL+Sack share, Power Four.
		{MFLID: "0701", Name: "DT Alpha", Position: domain.PosDT, BasePoints: 185, Age: 22, RAS: 9.2, HasRAS: true, Salary: 9,
			BreakoutAge: 20, HasBreakoutAge: true, SchoolTier: scouting.SchoolPowerFour, CollegeShare: 0.24, HasCollegeShare: true},
		// DT Bravo: weaker athlete, later breakout, moderate share, Group of Five.
		{MFLID: "0702", Name: "DT Bravo", Position: domain.PosDT, BasePoints: 170, Age: 23, RAS: 6.4, HasRAS: true, Salary: 4,
			BreakoutAge: 22, HasBreakoutAge: true, SchoolTier: scouting.SchoolGroupOfFive, CollegeShare: 0.15, HasCollegeShare: true},
	}
}
