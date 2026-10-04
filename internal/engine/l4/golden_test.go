package l4_test

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash/fnv"
	"math"
	"math/rand/v2"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4"
)

func present(has bool, v float64) float64 {
	if !has {
		return l4.NeutralNorm
	}
	return v
}

// goldenCases is how many inputs each position's golden hash covers.
const goldenCases = 20000

// golden holds, per position, the sha256 of every output bit over grid(pos). The hashes were
// taken from the ten per-position rubrics this package replaced (Stage 5), so a match proves the
// one routine reproduces them bit for bit.
func golden() map[domain.Position]string {
	return map[domain.Position]string{
		domain.PosQB: "0e0cd91b144cd89eb298dca05be5c79d8e12c9f54ed1e014db84e99d14bd8611",
		domain.PosRB: "adc43a15605a751b4f07973f1ea35497938b3ff3fdf71db716aba6e9d13b254c",
		domain.PosWR: "a6bf3dbd97a31862fa253fb27527842965c259e13ca53572384e03e973668863",
		domain.PosTE: "125ea01d74e807b37b066be0a4b212e342678c9b634c88d7ff7ff883d6ecf538",
		domain.PosDT: "f5edce5b2f932cb8188c5390f21d7c5494d512d62a9591e300c4948024b9887b",
		domain.PosDE: "49e28ac1b4b23d72ce34afb490e3b3619366daf03d489ee7a444040400ca0923",
		domain.PosLB: "e37af7d99d5a50ec7fa4f1d1e7632fdac7eed1ff19f6afe738e2304b0225bb0b",
		domain.PosCB: "2889c379027cdb4590f17456a180b650a989e044e138f0db689a0498ada043ac",
		domain.PosS:  "07e4aea0333012f6eeac2373a93a168834576450ae612ef03c58c5a32661aa4a",
		domain.PosK:  "97ac6dbf2a6cce5df3049863915dd1bcb5a5b1d1a1d7659fad801f01122dadcc",
	}
}

// grid returns deterministic inputs that cover each sub-signal present and absent, values inside
// and outside every curve, and the non-finite values the rubric must treat as unknown. Only DT
// carries a cushion, as composition hands it out. K's film draws are the 0.60/0.40 blend of the
// two kicking components the old K rubric read, so the hash taken from it still applies.
func grid(pos domain.Position) []engine.Layer4Input {
	h := fnv.New64a()
	_, _ = h.Write([]byte(pos))
	r := rand.New(rand.NewPCG(h.Sum64(), 2026)) //nolint:gosec // a fixed test grid, not a secret
	odd := []float64{math.NaN(), math.Inf(1), math.Inf(-1)}
	pick := func(lo, hi float64, edges ...float64) float64 {
		switch r.IntN(8) {
		case 0:
			return odd[r.IntN(len(odd))]
		case 1, 2:
			return edges[r.IntN(len(edges))]
		default:
			return lo + r.Float64()*(hi-lo)
		}
	}
	out := make([]engine.Layer4Input, goldenCases)
	for i := range out {
		in := engine.Layer4Input{Player: engine.PlayerInput{
			Position: pos,
			Age:      pick(18, 40, 19.5, 20, 21, 22, 24, 25, 26, 28, 29, 30, 32, 33, 34, 37),
			RAS:      pick(-1, 11, 0, 5, 7.99, 8, 8.01, 10),
			HasRAS:   r.IntN(2) == 0,
		}}
		sc := &in.Scouting
		sc.FilmComposite, sc.HasFilm = pick(-0.2, 1.2, 0, 0.5, 1), r.IntN(3) > 0
		sc.BreakoutAge, sc.HasBreakoutAge = pick(17, 25, 19, 19.5, 20, 20.5, 21, 22, 23), r.IntN(3) > 0
		sc.SchoolTierNorm, sc.HasSchoolTier = pick(0, 1, 0.1, 0.15, 0.4, 0.45, 0.7, 0.75, 1), r.IntN(3) > 0
		sc.CollegeShare, sc.HasCollegeShare = pick(0, 0.8, 0.08, 0.15, 0.22, 0.35, 0.5, 0.65), r.IntN(3) > 0
		if pos == domain.PosK {
			madden, hasMadden := pick(0, 1, 0, 0.5, 1), r.IntN(2) == 0
			production, hasProduction := pick(0, 1, 0, 0.5, 1), r.IntN(2) == 0
			sc.HasFilm = hasMadden || hasProduction
			sc.FilmComposite = 0.60*present(hasMadden, madden) + 0.40*present(hasProduction, production)
		}
		if pos == domain.PosDT && r.IntN(4) > 0 {
			in.Cushion = engine.CushionGuard{RASThreshold: pick(0, 10, 8), DeclineFactor: pick(0, 1, 0.9, 1.5, -0.1)}
		}
		out[i] = in
	}
	return out
}

// digest hashes every output field's bits, in grid order.
func digest(rubric engine.Layer4, inputs []engine.Layer4Input) string {
	h := sha256.New()
	var buf [8]byte
	for _, in := range inputs {
		o := rubric.Apply(in)
		for _, v := range []float64{o.FilmEffective, o.FilmRaw, o.RASEffective, o.BreakoutEffective, o.Combined} {
			binary.LittleEndian.PutUint64(buf[:], math.Float64bits(v))
			_, _ = h.Write(buf[:])
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

func TestRubricsMatchGolden(t *testing.T) {
	want := golden()
	for pos, rubric := range l4.Rubrics(l4.Defaults()) {
		inputs := grid(pos)
		got := digest(rubric, inputs)
		if want[pos] != got {
			t.Errorf("%s: outputs hash %s, want %s", pos, got, want[pos])
		}
	}
}
