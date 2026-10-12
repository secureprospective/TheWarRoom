package facts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type ResolvedInput struct {
	File   string
	SHA256 string
	Bytes  int64
	Body   []byte
}

type compiler struct {
	inputs map[string]ResolvedInput
	facts  factRows
	years  []int64
}

func Compile(ctx context.Context, inputs []ResolvedInput, notes []RawNoteReading) (Facts, error) {
	c, err := newCompiler(ctx, inputs)
	if err != nil {
		return Facts{}, err
	}
	for _, year := range c.years {
		if err := ctx.Err(); err != nil {
			return Facts{}, fmt.Errorf("compile facts: %w", err)
		}
		if err := c.league(year); err != nil {
			return Facts{}, err
		}
		if err := c.standings(year); err != nil {
			return Facts{}, err
		}
		if err := c.rules(year); err != nil {
			return Facts{}, err
		}
		if err := c.calendar(year); err != nil {
			return Facts{}, err
		}
		if err := c.season(year); err != nil {
			return Facts{}, err
		}
	}
	if err := c.ruleCodes(); err != nil {
		return Facts{}, err
	}
	if err := c.players(ctx); err != nil {
		return Facts{}, err
	}
	if err := c.notes(ctx, notes); err != nil {
		return Facts{}, err
	}
	if err := ctx.Err(); err != nil {
		return Facts{}, fmt.Errorf("publish facts: %w", err)
	}
	sortFacts(&c.facts)
	return toColumns(c.facts), nil
}

func newCompiler(ctx context.Context, inputs []ResolvedInput) (*compiler, error) {
	c := &compiler{inputs: make(map[string]ResolvedInput, len(inputs))}
	years := map[int64]bool{}
	for _, input := range inputs {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("compile inputs: %w", err)
		}
		if err := input.Validate(); err != nil {
			return nil, err
		}
		if _, exists := c.inputs[input.File]; exists {
			return nil, fmt.Errorf("duplicate input %s", input.File)
		}
		c.inputs[input.File] = input
		c.facts.SourceFile = append(c.facts.SourceFile, SourceFile{Src: input.File, Bytes: input.Bytes, Sha256: input.SHA256})
		if part, _, ok := strings.Cut(input.File, "/"); ok {
			year, err := strconv.ParseInt(part, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("input year %s: %w", input.File, err)
			}
			years[year] = true
		}
	}
	for year := range years {
		c.years = append(c.years, year)
	}
	slices.Sort(c.years)
	return c, nil
}

func (r ResolvedInput) Validate() error {
	sha, err := hex.DecodeString(r.SHA256)
	if err != nil || len(sha) != sha256.Size || r.Bytes < 0 || r.File == "" || strings.Contains(r.File, "..") || strings.HasPrefix(r.File, "/") {
		return fmt.Errorf("invalid resolved input %q", r.File)
	}
	if r.Body != nil {
		sum := sha256.Sum256(r.Body)
		if hex.EncodeToString(sum[:]) != r.SHA256 || int64(len(r.Body)) != r.Bytes {
			return fmt.Errorf("resolved bytes differ from pin: %s", r.File)
		}
	}
	return nil
}

func (c *compiler) read(file string) (RawValue, error) {
	input, exists := c.inputs[file]
	if !exists || input.Body == nil {
		return RawValue{}, nil
	}
	r, err := decodeRaw(input.Body)
	if err != nil {
		return RawValue{}, fmt.Errorf("%s: %w", file, err)
	}
	if err := r.Validate(); err != nil {
		return RawValue{}, fmt.Errorf("%s: %w", file, err)
	}
	if _, exists := r.Object["error"]; exists {
		return RawValue{}, nil
	}
	return r, nil
}

func seasonFile(year int64, name string) string { return fmt.Sprintf("%d/%s.json", year, name) }

func requiredID(r RawValue, key, src string) (string, error) {
	field := r.get(key)
	id := field.str()
	if id == nil || *id == "" || len(field.JSON) == 0 || field.JSON[0] != '"' {
		return "", fmt.Errorf("%s: missing %s", src, key)
	}
	return *id, nil
}

// keyID reads an id the Lab indexes as row[key]: it must be present as a JSON string, but MFL's
// weekly files carry blank ids (projectedScores lists {"id": ""}) and the Lab keeps those rows.
func keyID(r RawValue, key, src string) (string, error) {
	field := r.get(key)
	if field.Scalar == nil || len(field.JSON) == 0 || field.JSON[0] != '"' {
		return "", fmt.Errorf("%s: missing %s", src, key)
	}
	return *field.Scalar, nil
}
