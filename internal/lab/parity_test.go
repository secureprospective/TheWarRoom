package lab

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

type golden struct {
	Header struct {
		DataSHA   string `json:"dataSHA"`
		EngineSHA string `json:"engineSHA"`
		Node      string `json:"node"`
		Scope     Scope  `json:"scope"`
		Name      string `json:"name"`
	} `json:"header"`
	CompilerPrices []CompilerPrice            `json:"compilerPrices"`
	Values         map[string]json.RawMessage `json:"values"`
}

func loadData(t *testing.T) *Data {
	t.Helper()
	d, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	return d
}
func readGoldenBytes(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	decoded, readErr := io.ReadAll(r)
	closeErr := r.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	return decoded
}
func readGolden(t *testing.T, path string) golden {
	t.Helper()
	b := readGoldenBytes(t, path)
	var g golden
	if err := json.Unmarshal(b, &g); err != nil {
		t.Fatal(err)
	}
	if g.Header.DataSHA != DataSHA || g.Header.EngineSHA != "d7a914bcf5f12c22b004a513d0c1a443356fad2bffc3ef6080f788283d296cfd" || g.Header.Node != "v24.21.0" {
		t.Fatal("invalid golden provenance")
	}
	return g
}

func computed(d *Data, s Scope, name string) map[string]any {
	keep := InScope(s)
	vt := ValuedTrades(d, func(t Trade) bool {
		if !keep(t) {
			return false
		}
		if name == "pick-only" {
			for _, l := range t.Legs {
				if l.XClass == nil || !strings.HasPrefix(*l.XClass, "pick") {
					return false
				}
			}
		}
		return true
	})
	f := FitPrices(vt)
	stretches := StretchPrices(d, s, []string{"winter", "opening", "between", "rookieDraft", "summer", "early", "midseason", "deadline"})
	kinds := []string{"LB", "QB", "pick1:next", "pick1:later", "missing"}
	if s.ByAge {
		kinds = []string{"LB:young", "LB:prime", "LB:older", "pick1:next", "pick1:later", "missing"}
	}
	profiles := make(map[string]*Profile)
	for _, k := range kinds {
		profiles[k] = AssetProfile(d, s, k, stretches)
	}
	deliveries := make(map[string]*Delivered)
	ranges := make(map[string]*Range)
	grades := make(map[string]string)
	groups := make(map[string]string)
	ages := make(map[string]string)
	assets := make(map[string]string)
	for _, k := range f.Kinds {
		deliveries[k] = DeliveredRatio(recordsFor(d, keep, AgeOf, k))
		p := f.Prices[k]
		ranges[k] = PerDelivered(&p, deliveries[k])
		grades[k] = evidence(ranges[k])
		groups[k] = GroupOf(k)
		ages[k] = AgeOf(k)
		assets[k] = AssetOf(k)
	}
	ids := []int{}
	for _, t := range d.Trades {
		if keep(t) {
			ids = append(ids, t.ID)
		}
	}
	values := []float64{}
	for _, v := range vt {
		for _, side := range v.Sides {
			for _, a := range side {
				values = append(values, a.Value)
			}
		}
	}
	pc, err := TradesByPhase(d, s)
	if err != nil {
		panic(err)
	}
	return map[string]any{"valuedTrades": vt, "fitPrices": f, "deliveredRatio": deliveries, "perDelivered": ranges, "evidence": grades, "priceList": PriceList(d, s), "meanVar": MeanVar(values), "groupOf": groups, "ageOf": ages, "assetOf": assets, "inScope": ids, "leagueMarket": LeagueMarket(d, s), "tradesByPhase": pc, "pickClock": PickClock(d, s), "contenderTrades": ContenderTrades(d, s), "stretchPrices": stretches, "assetProfile": profiles}
}
func parity(want, got any, path string, maxError *float64) error {
	if reflect.TypeOf(want) != reflect.TypeOf(got) {
		return fmt.Errorf("%s: type/null mismatch want %T got %T", path, want, got)
	}
	switch w := want.(type) {
	case map[string]any:
		g := got.(map[string]any)
		if len(w) != len(g) {
			return fmt.Errorf("%s: keys differ want %v got %v", path, keys(w), keys(g))
		}
		for _, k := range keys(w) {
			v := w[k]
			if _, ok := g[k]; !ok {
				return fmt.Errorf("%s: missing %s", path, k)
			}
			if err := parity(v, g[k], path+"."+k, maxError); err != nil {
				return err
			}
		}
	case []any:
		g := got.([]any)
		if len(w) != len(g) {
			return fmt.Errorf("%s: length %d != %d", path, len(w), len(g))
		}
		for i, v := range w {
			if err := parity(v, g[i], fmt.Sprintf("%s[%d]", path, i), maxError); err != nil {
				return err
			}
		}
	case float64:
		g := got.(float64)
		e := math.Abs(g - w)
		*maxError = math.Max(*maxError, e)
		if exactNumber(path) {
			if g != w {
				return fmt.Errorf("%s: exact %g != %g", path, w, g)
			}
		} else if e > 1e-9+1e-9*math.Abs(w) {
			return fmt.Errorf("%s: %g != %g (error %g)", path, w, g, e)
		}
	default:
		if !reflect.DeepEqual(want, got) {
			return fmt.Errorf("%s: %v != %v", path, want, got)
		}
	}
	return nil
}
func keys(m map[string]any) []string {
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
func exactNumber(path string) bool {
	for _, name := range []string{".n", ".trades", ".from", ".to", ".mid", ".records", ".id", ".moved", ".realN", ".total", ".at", ".like", ".overall", ".season"} {
		if strings.HasSuffix(path, name) {
			return true
		}
	}
	return strings.Contains(path, ".n.") || strings.Contains(path, "inScope[") || strings.Contains(path, ".months[") || strings.Contains(path, ".span[") || strings.Contains(path, ".seasons[") || strings.HasSuffix(path, ".seasons")
}
func jsonValue(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}
func TestParity(t *testing.T) {
	d := loadData(t)
	paths, err := filepath.Glob("testdata/goldens/*.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 13 {
		t.Fatalf("expected 13 fixture files; got %d", len(paths))
	}
	maxima := make(map[string]float64)
	for _, path := range paths {
		if strings.HasSuffix(path, "boundaries.json.gz") || strings.HasSuffix(path, "draft.json.gz") || strings.HasSuffix(path, "teams.json.gz") {
			continue
		}
		g := readGolden(t, path)
		t.Run(g.Header.Name, func(t *testing.T) {
			actual := computed(d, g.Header.Scope, g.Header.Name)
			for _, name := range keysRaw(g.Values) {
				var want any
				if err := json.Unmarshal(g.Values[name], &want); err != nil {
					t.Fatal(err)
				}
				maximum := 0.0
				if err := parity(want, jsonValue(t, actual[name]), name, &maximum); err != nil {
					t.Error(err)
				}
				maxima[name] = math.Max(maxima[name], maximum)
			}
		})
	}
	for _, name := range keysFloat(maxima) {
		t.Logf("%-18s max absolute error %.17g", name, maxima[name])
	}
}
func keysRaw(m map[string]json.RawMessage) []string {
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
func keysFloat(m map[string]float64) []string {
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
func TestCompilerAnchor(t *testing.T) {
	d := loadData(t)
	f := FitPrices(ValuedTrades(d, nil))
	g := readGolden(t, "testdata/goldens/all.json.gz")
	for _, c := range g.CompilerPrices {
		p, ok := f.Prices[c.ID]
		if !ok || math.Abs(p.Price-c.Price) >= 0.01 {
			t.Errorf("compiler price %s: %.12g != %.12g", c.ID, p.Price, c.Price)
		}
	}
	if f.Prices[Anchor] != (Price{1, 0}) {
		t.Fatal("anchor moved")
	}
}
func TestMutants(t *testing.T) {
	d := loadData(t)
	g := readGolden(t, "testdata/goldens/measured.json.gz")
	s := g.Header.Scope
	var want any
	if err := json.Unmarshal(g.Values["fitPrices"], &want); err != nil {
		t.Fatal(err)
	}
	m := fitPrices(ValuedTrades(d, InScope(s)), true)
	maxError := 0.0
	if err := parity(want, jsonValue(t, m), "fitPrices.dropRidge", &maxError); err == nil {
		t.Fatal("ridge mutant survived")
	} else {
		t.Logf("ridge mutant killed: %v", err)
	}
	b := readGoldenBytes(t, "testdata/goldens/boundaries.json.gz")
	var boundary struct {
		Records        []Delivery `json:"records"`
		DeliveredRatio *Delivered `json:"deliveredRatio"`
	}
	if err := json.Unmarshal(b, &boundary); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(DeliveredRatio(boundary.Records), boundary.DeliveredRatio) {
		t.Fatal("normal boundary parity failed")
	}
	maxError = 0
	if err := parity(jsonValue(t, boundary.DeliveredRatio), jsonValue(t, deliveredRatio(boundary.Records, 7)), "deliveredRatio.threshold7", &maxError); err == nil {
		t.Fatal("threshold mutant survived")
	} else {
		t.Logf("threshold mutant killed: %v", err)
	}
}
