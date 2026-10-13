package leaguerules

import "time"

type ContractsRules struct {
	MaximumYears            Value[int]
	FlatSalaries            Value[bool]
	MinimumSalaries         Value[[]ExperienceSalary]
	DeadCap                 Value[Decimal]
	RestructuredDeadCap     Value[Decimal]
	ClaimClearsDeadCap      Value[bool]
	ReleaseReSignSameSeason Value[bool]
	Tag                     Value[TagRules]
	Extension               Value[ExtensionRules]
	Restructure             Value[RestructureRules]
	Buyout                  Value[BuyoutRules]
	Tender                  Value[TenderRules]
	ReSign                  Value[ReSignRules]
	Retirement              Value[SpecialRules]
	Death                   Value[SpecialRules]
	CapRelief               Value[ReliefRules]
}

func legacyContracts() ContractsRules {
	return ContractsRules{
		MaximumYears:            cited[int](4, "Rulebook §6"),
		FlatSalaries:            cited[bool](true, "Math §1"),
		MinimumSalaries:         cited[[]ExperienceSalary](legacyMinimums(), "Rulebook §6; Math §14 ruling 2"),
		DeadCap:                 cited[Decimal](3500, "Math §4"),
		RestructuredDeadCap:     cited[Decimal](5000, "Math §4"),
		ClaimClearsDeadCap:      cited[bool](true, "Math §4"),
		ReleaseReSignSameSeason: cited[bool](false, "Rulebook §8"),
		Tag:                     cited[TagRules](legacyTag(), "Rulebook §9; Math §5"),
		Extension:               cited[ExtensionRules](legacyExtension(), "Rulebook §10; Math §6"),
		Restructure:             cited[RestructureRules](legacyRestructure(), "Rulebook §11; Math §7"),
		Buyout:                  cited[BuyoutRules](legacyBuyout(), "Rulebook §12; Math §14 ruling 3"),
		Tender:                  cited[TenderRules](legacyTender(), "Rulebook §7"),
		ReSign:                  cited[ReSignRules](legacyReSign(), "Rulebook §7"),
		Retirement:              cited[SpecialRules](special(3000, "Rulebook §13"), "Rulebook §13; Math §9"),
		Death:                   cited[SpecialRules](special(0, "Rulebook §13"), "Rulebook §13"),
		CapRelief:               cited[ReliefRules](legacyRelief(), "Rulebook §13"),
	}
}

func contractsFields() []ruleField {
	return []ruleField{
		field[int]{path: "contracts.maximumYears", label: "Maximum new contract years", access: func(r *RuleSet) *Value[int] { return &r.Contracts.MaximumYears }},
		field[bool]{path: "contracts.flatSalaries", label: "Flat annual salaries", access: func(r *RuleSet) *Value[bool] { return &r.Contracts.FlatSalaries }},
		field[[]ExperienceSalary]{path: "contracts.minimumSalaries", label: "Minimum bidding salaries", access: func(r *RuleSet) *Value[[]ExperienceSalary] { return &r.Contracts.MinimumSalaries }},
		field[Decimal]{path: "contracts.deadCap", label: "Release dead cap rate", access: func(r *RuleSet) *Value[Decimal] { return &r.Contracts.DeadCap }},
		field[Decimal]{path: "contracts.restructuredDeadCap", label: "Restructured release dead cap rate", access: func(r *RuleSet) *Value[Decimal] { return &r.Contracts.RestructuredDeadCap }},
		field[bool]{path: "contracts.claimClearsDeadCap", label: "Waiver claim clears dead cap", access: func(r *RuleSet) *Value[bool] { return &r.Contracts.ClaimClearsDeadCap }},
		field[bool]{path: "contracts.releaseReSignSameSeason", label: "Re-sign a released player this season", access: func(r *RuleSet) *Value[bool] { return &r.Contracts.ReleaseReSignSameSeason }},
		field[TagRules]{path: "contracts.tag", label: "Franchise tag rules", access: func(r *RuleSet) *Value[TagRules] { return &r.Contracts.Tag }},
		field[ExtensionRules]{path: "contracts.extension", label: "Contract extension rules", access: func(r *RuleSet) *Value[ExtensionRules] { return &r.Contracts.Extension }},
		field[RestructureRules]{path: "contracts.restructure", label: "Contract restructure rules", access: func(r *RuleSet) *Value[RestructureRules] { return &r.Contracts.Restructure }},
		field[BuyoutRules]{path: "contracts.buyout", label: "Contract buyout rules", access: func(r *RuleSet) *Value[BuyoutRules] { return &r.Contracts.Buyout }},
		field[TenderRules]{path: "contracts.tender", label: "Restricted free agent tender rules", access: func(r *RuleSet) *Value[TenderRules] { return &r.Contracts.Tender }},
		field[ReSignRules]{path: "contracts.reSign", label: "Restricted free agent re-sign rules", access: func(r *RuleSet) *Value[ReSignRules] { return &r.Contracts.ReSign }},
		field[SpecialRules]{path: "contracts.retirement", label: "Retirement rules", access: func(r *RuleSet) *Value[SpecialRules] { return &r.Contracts.Retirement }},
		field[SpecialRules]{path: "contracts.death", label: "Player death rules", access: func(r *RuleSet) *Value[SpecialRules] { return &r.Contracts.Death }},
		field[ReliefRules]{path: "contracts.capRelief", label: "Cap relief appeals", access: func(r *RuleSet) *Value[ReliefRules] { return &r.Contracts.CapRelief }},
	}
}

type CapRules struct {
	Amount             Value[Money]
	Enforcement        Value[CapMode]
	IRShare            Value[Decimal]
	TaxiShare          Value[Decimal]
	Quantum            Value[Money]
	ComplianceDeadline Value[time.Time]
}

func legacyCap() CapRules {
	return CapRules{
		Amount:             cited[Money](12500000000, "Rulebook §1"),
		Enforcement:        cited[CapMode](HardCap, "Census CM-008 Q-A3"),
		IRShare:            cited[Decimal](10000, "Rulebook §1; Math §0"),
		TaxiShare:          cited[Decimal](10000, "Rulebook §1; Math §0"),
		Quantum:            cited[Money](1000000, "Math §0"),
		ComplianceDeadline: Needs[time.Time]("Census CM-008 Q-A3"),
	}
}

func capFields() []ruleField {
	return []ruleField{
		field[Money]{path: "cap.amount", label: "Salary cap", access: func(r *RuleSet) *Value[Money] { return &r.Cap.Amount }},
		field[CapMode]{path: "cap.enforcement", label: "Cap enforcement", access: func(r *RuleSet) *Value[CapMode] { return &r.Cap.Enforcement }},
		field[Decimal]{path: "cap.irShare", label: "IR salary share", access: func(r *RuleSet) *Value[Decimal] { return &r.Cap.IRShare }},
		field[Decimal]{path: "cap.taxiShare", label: "Taxi salary share", access: func(r *RuleSet) *Value[Decimal] { return &r.Cap.TaxiShare }},
		field[Money]{path: "cap.quantum", label: "Money rounding increment", access: func(r *RuleSet) *Value[Money] { return &r.Cap.Quantum }},
		field[time.Time]{path: "cap.complianceDeadline", label: "Soft cap compliance deadline", access: func(r *RuleSet) *Value[time.Time] { return &r.Cap.ComplianceDeadline }},
	}
}
