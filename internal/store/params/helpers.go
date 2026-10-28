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

// scanDef reads one param_defaults row.
func scanDef(rows *sql.Rows, shipped map[string]bool) (ParamDef, bool, error) {
	var (
		d                   ParamDef
		vtype, defv, mn, mx string
		calibrated          int
	)
	if err := rows.Scan(&d.Key, &d.Position, &vtype, &defv, &mn, &mx,
		&calibrated, &d.Description); err != nil {
		return ParamDef{}, false, fmt.Errorf("params: scan default: %w", err)
	}
	if !shipped[defKey(d.Key, d.Position)] {
		return ParamDef{}, false, nil
	}
	var err error
	if d.Default, err = atof(defv, "default"); err != nil {
		return ParamDef{}, false, err
	}
	if d.Min, err = atof(mn, "min"); err != nil {
		return ParamDef{}, false, err
	}
	if d.Max, err = atof(mx, "max"); err != nil {
		return ParamDef{}, false, err
	}
	d.Type = ValueType(vtype)
	d.IsCalibrated = calibrated != 0
	return d, true, nil
}

// scanOverride reads one param_overrides row into a typed Override.
func scanOverride(rows *sql.Rows, defs map[string]ParamDef) (Override, bool, error) {
	var (
		o   Override
		val string
	)
	if err := rows.Scan(&o.Key, &o.Position, &val, &o.Note, &o.UpdatedAt); err != nil {
		return Override{}, false, fmt.Errorf("params: scan override: %w", err)
	}
	if _, ok := defs[defKey(o.Key, o.Position)]; !ok {
		return Override{}, false, nil
	}
	v, err := atof(val, "override value")
	if err != nil {
		return Override{}, false, err
	}
	o.Value = v
	return o, true, nil
}
