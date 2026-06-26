package params

import (
	"database/sql"
	"fmt"
	"strconv"
)

// ftoa encodes a float64 for TEXT storage with full round-trip precision.
func ftoa(f float64) string { return strconv.FormatFloat(f, 'g', -1, 64) }

// atof parses a stored TEXT float, wrapping a corrupt value with its field name.
func atof(s, field string) (float64, error) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("params: parse %s %q: %w", field, s, err)
	}
	return f, nil
}

// boolToInt maps a bool to the 0/1 SQLite integer encoding.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// scanDef reads one param_defaults row into a typed ParamDef, parsing its float
// fields from their TEXT storage.
func scanDef(rows *sql.Rows) (ParamDef, error) {
	var (
		d                   ParamDef
		vtype, defv, mn, mx string
		calibrated          int
	)
	if err := rows.Scan(&d.Key, &d.Position, &vtype, &defv, &mn, &mx,
		&calibrated, &d.Description); err != nil {
		return ParamDef{}, fmt.Errorf("params: scan default: %w", err)
	}
	var err error
	if d.Default, err = atof(defv, "default"); err != nil {
		return ParamDef{}, err
	}
	if d.Min, err = atof(mn, "min"); err != nil {
		return ParamDef{}, err
	}
	if d.Max, err = atof(mx, "max"); err != nil {
		return ParamDef{}, err
	}
	d.Type = ValueType(vtype)
	d.IsCalibrated = calibrated != 0
	return d, nil
}

// scanOverride reads one param_overrides row into a typed Override.
func scanOverride(rows *sql.Rows) (Override, error) {
	var (
		o   Override
		val string
	)
	if err := rows.Scan(&o.Key, &o.Position, &val, &o.Note, &o.CreatedAt); err != nil {
		return Override{}, fmt.Errorf("params: scan override: %w", err)
	}
	v, err := atof(val, "override value")
	if err != nil {
		return Override{}, err
	}
	o.Value = v
	return o, nil
}
