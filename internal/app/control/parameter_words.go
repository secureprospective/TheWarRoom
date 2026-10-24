package control

import (
	"fmt"
	"strings"
)

func ParameterWords(key, description string) (label, group string) {
	fixed := map[string][2]string{
		"captier.cold_ceiling_pct":    {"Cold salary tier ceiling (% of cap)", "Salary tiers"},
		"captier.hot_floor_pct":       {"Hot salary tier floor (% of cap)", "Salary tiers"},
		"layer3.decay_rate":           {"Annual age decline", "Board age adjustment"},
		"cushion_guard.ras_threshold": {"Athletic rating threshold", "Defensive tackle decline guard"},
		"cushion_guard.reduction":     {"Decline reduction", "Defensive tackle decline guard"},
		"dynasty.seasons":             {"Seasons counted, starting this season", "Dynasty horizon"},
		"dynasty.discount":            {"Weight of each later season", "Dynasty horizon"},
		"model.k_now":                 {"Games until production and scouting weigh equally", "Production blend · this season"},
		"model.k_dynasty":             {"Games until production and scouting weigh equally", "Production blend · dynasty"},
		"model.z_exponential":         {"Use the exponential production blend", "Production blend"},
		"model.board_kept":            {"Value this position with the fitted ratings (off: use the board rule)", "Production blend"},
	}
	if words, ok := fixed[key]; ok {
		return words[0], words[1]
	}
	if strings.HasPrefix(key, "l4.") {
		area, detail, _ := strings.Cut(description, ": ")
		area = strings.ReplaceAll(area, "RAS", "Athletic rating")
		detail = strings.ReplaceAll(detail, "RAS", "athletic rating")
		return detail, "Scouting adjustment · " + area
	}
	if strings.HasPrefix(key, "model.recency.") {
		return fmt.Sprintf("Weight of season %s years back", strings.TrimPrefix(key, "model.recency.")), "Production history"
	}
	parts := strings.Split(key, ".")
	if len(parts) < 3 || parts[0] != "model" {
		return description, "Other settings"
	}
	groups := map[string]string{"prior": "Scouting prior", "arc": "Talent and age arc",
		"debut": "Rookie becoming a regular", "survival": "Staying in the NFL"}
	terms := map[string]string{
		"intercept": "Starting level", "level": "Starting level", "age": "Age effect",
		"age_squared": "Age curve", "second_year": "Second-year lift", "draft": "Draft capital",
		"bench": "Bench press", "broad_jump": "Broad jump", "college": "College production",
		"college_best": "Best college season", "college_interceptions": "College interceptions",
		"college_passes_defended": "College passes defended", "college_qb_hurries": "College quarterback hurries",
		"college_sacks": "College sacks", "college_seasons": "College seasons played",
		"college_tackles_for_loss": "College tackles for loss", "college_total_tackles": "College tackles",
		"entry_age": "NFL entry age", "forty": "40-yard dash", "height": "Height", "shuttle": "Short shuttle",
		"three_cone": "Three-cone drill", "vertical": "Vertical jump", "weight": "Body weight",
		"contract_cap": "Contract share of salary cap", "contract_guaranteed": "Guaranteed contract share",
		"contract_unknown": "Contract not held", "contract_years_left": "Contract years left",
		"games": "Games played", "percentile": "Production percentile",
	}
	label, ok := terms[parts[2]]
	if !ok {
		return description, "Other settings"
	}
	if len(parts) == 4 {
		suffix := map[string]string{"weight": "weight", "missing": "adjustment when not held"}
		label += " · " + suffix[parts[3]]
	}
	return label, groups[parts[1]]
}
