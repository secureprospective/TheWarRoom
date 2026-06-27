package engine

// Cleaned is L1's output: the player values downstream layers read, after the floor
// and imputation rules are applied. RASImputed marks that the RAS is a fallback, not a
// measured value (an internal flag — not surfaced to the UI).
type Cleaned struct {
	RAS        float64
	Salary     float64
	RASImputed bool
}

// ApplyHygiene is Layer 1: a pure per-record clean. It imputes a fallback RAS when one
// is absent and raises a sub-floor salary to the floor (Backend_Architecture:235). The
// position-group RAS mean is a BATCH concern that belongs to the composition boundary;
// L1 here takes the already-chosen fallback as a parameter and stays pure.
func ApplyHygiene(p PlayerInput, c Calibration) Cleaned {
	ras := p.RAS
	imputed := false
	if !p.HasRAS {
		ras = c.RASFallback
		imputed = true
	}
	salary := p.Salary
	if salary < c.SalaryFloor {
		salary = c.SalaryFloor
	}
	return Cleaned{RAS: ras, Salary: salary, RASImputed: imputed}
}
