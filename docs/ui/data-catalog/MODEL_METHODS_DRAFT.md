# MODEL METHODS DRAFT (planning only — no code)
TheWarRoom L2 forecasting models: one method per domain, each with a replay proof before it ships.
Drafted 2026-10-30, NL/PRD worktree. Read DATA_FLOW_MAP.md sections 1/3/4/10 for isolation classes and feed chains.

## Rules (fixed)
1. Correctness first: no model ships until it proves itself by replaying past seasons; each section states its replay test and pass bar.
2. "Value is value": Best 21 = the 21 most valuable players on a roster, ANY position — it is not a lineup.
3. Player ID is the player: value = f(one player record — draft capital, production, reputation, age), keyed by permanent player ID (`internal/playerrecord`).
4. League-public models read ONLY public inputs (never an owner's notes/watchlist/plans, never sealed bids); enforced per DATA_FLOW_MAP isolation class PUB.
5. Plain language everywhere: owners are not technical; constants get honest names and fitted-where notes.

## Dependency order (build order)
D26 → D25 → D27 → D28 → D32 → D33 · D11 → D05 → D06 · D35 · D42
(Trees: C2 value chain D26→D25→D27→D28→D32; C3 contend chain D11→D05→D06; C5 market chain D35; C8 draft chain D42.)

## Assumption marks
Anything not confirmed in code or the map is marked ASSUMPTION. Honest deltas from existing code are called out in the model's Inputs section.

---

## 1. D26 — Age curve (value by age, per position)

### What the owner sees
A small chart per position: "Hooked at 28, sliding after 30" — how a player's value travels with his age, and where each position peaks. It answers "am I buying him before or after the fall?"

### Inputs
- D12/T4 production history: per-player league PPG per finished season (keyed by permanent player ID, via `playerrecord.Book` / `model.Obs` production keyed by our ID).
- `playerrecord.Record.Birth` and `Record.Rookie` (internal/playerrecord/record.go:50-57) for age; `Record.Position` for the curve's position.NFL season = `Rookie` season + years of service; use actual birth date when held, fall back to Rookie year + 21 ASSUMPTION for missing birth dates.
- Prior native-value model runs (`history.Run`, `history.ModelScore` — the same feed `market.LoadNative` reads, internal/app/market/native.go:126-130) for the seasons-replayed basis.
- REUSE: the native value model's expected-PPG output (`Native.Player` → `Dynasty.Discounted`) is the scale to fold the curve back into; the curve is a smooth corrective multiplier on it, not a second value.

### Method
1. Define football age = league season − `Rookie` season (equivalently age from Birth at season midpoint; the curve is fitted on service years so undrafted-age gaps don't distort it).
2. For every finished past season t and every player p with ≥ 6 of a possible series of seasons around t ASSUMPTION (else bin too thin): record the pair (footballAge at t, PPG[p,t] in this league's scoring, Position[p]). This is the same production feed D12 already holds — no new data.
3. Normalize each player's series by his own peak: r = PPG(t) / max over t of PPG(p). This removes the player's absolute level and leaves the shape — "how much of himself is he at age X".
4. Bucket by (Position, footballAge) over 19–37; inside a bucket take the median of r across players, with winsorizing at the 10th/90th percentile so one late bloomer can't set the curve. Constants: `MIN_PLAYERS_PER_BUCKET` = 8 (below that, the bucket borrows from neighbouring ages — linear interpolation — before borrowing from next position player group is ever considered), `WINSOR_P` = 0.10.
5. Peak window = the contiguous age range where the smoothed curve stays ≥ 97% of its maximum. Constant `PEAK_LEVEL` = 0.97.
6. Smooth per position with a small monotone-tail constraint: value may rise mid-career but past 30 the curve may not rise year-over-year for RB/WR/TE (median-reality rule, verify in replay; drop the constraint for QB if replay says QBs don't decline that way).
7. Publish `multiplier(Position, footballAge)`: curve value relative to the position's peak, applied to the native value model's expected PPG for a player of that age. The published number a screen sees is still "dynasty PPG"; the curve is upstream of D25 and D24's display only in the D25 chain (per C2: D24 value → D26 → D25).

### Output shape
- `AgeCurve{ Season int; Points map[string][]CurvePoint }` where `CurvePoint{ Position string; FootballAge int; Multiplier float64; PeakWindowLow, PeakWindowHigh int; BucketN int }`
- `Multiplier(pos string, age int) float64` — interpolated, clamped [0, 1.0/1.15…], legacy NaN-safe.

### Refresh trigger
Nightly with the value model run (same batch as D24); never intra-week.

### Replay proof
Rewind to each of the last 4 seasons at each week the app would have run the model. For each player aged ≤ 30 at rewind: predict his next-season production percentile within position using (a) full curve, (b) flat-age baseline (this season's percentile only). Pass bar: the curve-augmented prediction outranks the flat baseline (Spearman, per position, pooled 4 seasons) by a margin that is positive in ≥ 3 of the 4 seasons and never negative in 2 of them; and the fitted peak windows land within 1 year of the windows documented from the last fit (docs/modules/M2_Power_Ranking_Factors.md assumptions).
Secondary proof: D25 Best-21 order built with the curve must not churn > 2 players per roster per week vs itself run night-to-night on the same data (stability, not accuracy).

### Failure / thin-data behaviour
- Rookie year data (age 21) is under-represented in finished seasons only a few weeks in: the curve point exists but is marked `Thin` when `BucketN < MIN_PLAYERS_PER_BUCKET`; D25 still applies it (interpolated), D27 flagged "young-player value is provisional".
- Position with < 15 players of history (kicker, IDP novel positions): publish no curve; multiplier(x) = 1.0 with reason "no curve at this position", so value falls through to plain native value (never a wrong number).
- Week 1 of a new season: curve is last season's; each lattice point carries its Season so stale curves are labelled "fitted through 2025".

### Open questions
- Does the commissioner want the curve visible as a standalone tile on the War Room, or only felt through D25? (map says feeds-to WR/HQ/D25 — confirming the standalone view.)
- Keeper leagues (not the default) may want age curves per league-type; out of scope now, flag when Phase 2 lands.

---

## 2. D25 — Best 21 team score + league rank

### What the owner sees
"Your best 21 are worth 142.3 dynasty PPG — 4th of 32." The 21 MOST VALUABLE players on the roster, any position, in a plain league-wide rank. Depth beyond 21 does not inflate the number, and a bench RB doesn't flatter a shy QB.

### Inputs
- D17 rosters (store/state, PUB): every franchise's players, any slot (taxi/IR count — value is value, not availability).
- D24/D26 value chain: `market.Model.Player(pid, at) → Dynasty.Discounted` (internal/app/market/assets.go:60-63, native.go:86-96) folded through D26's `AgeCurve.Multiplier(pos, footballAge)`.
- NOT a lineup: `powerrankings.Lineup` (internal/powerrankings/lineup.go:43) is slot-bounded and is NOT reused here — its job (D33, C3) is different. Best 21 is a plain top-N by value with no slot constraints.
- HONESTY NOTE on an assumption I nearly made: DefaultAgg counts "the starting lineup by default" for franchise value today (season/power.go:90-97). D25 deliberately diverges: top 21 by value is the locked ruling; the lineup view stays as-is for D05's LINEUP component (step 8).

### Method
1. Per roster: fetch every rostered player's adjusted value = native dynasty PPG × D26 multiplier at his football age. A player the model can't value (no run, mislinked) contributes 0 and is listed with his reason — never silently dropped.
2. Sort desc; take the top 21 (`BEST_N` = 21, a named constant so a shallower league could change it). Ties broken by player ID for determinism; equal values shown as equal.
3. Team score = sum of the 21 adjusted values. Divide nothing, discount nothing — the discount already lives in `Dynasty.Discounted` (later seasons discounted over `Dynasty.Seasons`).
4. Contracts: the DEFAULT league has contracts + cap, but contract PRICE is not a value input (ruling: value = f(player record)). Price burden surfaces in D20 runway and D27, not here.
5. League rank: rank all 32 franchises by score, high→low, franchise-ID tiebreak; write one row per franchise nightly with the model run ID so history and "why my rank moved" are answerable.
6. "Why it moved" is left to D05's narrative layer; D25 only publishes the delta of the score and which of the 21 changed (in/out of the 21 is a cheap pass — record the membership change, one line).

### Output shape
- `Best21{ FranchiseID string; Score float64; Rank int; Season int; Members []struct{ PlayerID string; Value float64; WasIn bool /* change vs previous run */ }; ReplacementN int }`
- League snapshot: `[]Best21` sorted by Rank, plus `AsOf`, `ModelRunID int64`.

### Refresh trigger
Nightly with the value model run; also on any roster move (a trade dealining should not leave the board stale all day) — map says nightly, I am flagging "on move" as the cheap second trigger ASSUMPTION pending commish appetite for compute.

### Replay proof
Rewind each of the last 4 seasons to (a) the preseason and (b) week 8. Best 21 score per franchise at rewind must predict that team's all-play win percentage over the NEXT 8 weeks and the NEXT season's power rank. Pass bar: Spearman ≥ 0.55 against next-8-week all-play (it is a dynasty value so it should be at least a fair predictor), and Best 21 with the age curve outranks plain sum-of-all-rostered-value as a predictor in ≥ 3 of the 4 seasons. Also a monotonicity spot-check: the franchise whose rewind Best 21 ranks 1st ends the following season inside the top half of power in ≥ 3 of 4 seasons.

### Failure / thin-data behaviour
- No model run yet: the tile renders skeleton with "Not valued yet — no model run read yet" (same wording as DynastyCells, assets.go:156) and no rank.
- Fewer than 21 rostered players (early dynasty churn): score = sum of what exists, rank computed anyway, tile notes "roster below 21 players · score reflects {n}".
- Week 1: values come from the offseason run; the tile is labelled with that run's season. Never parallel-compute over a partial week's scores.

### Open questions
- Is a 4th metric "salary-adjusted Best 21" wanted for cap-strapped owners, or does D27 surplus cover it? (Real: the default league is a cap league and the ruling says value ∝ player record, so price never enters here — confirming the commissioner agrees price burden stays only in D20/D27.)
- Should the tile show the 21 names always (long scroll) or the count + "who's new"? (Mock says a strip; verifying with WR v6 mock.)

---

## 3. D27 — Team needs / surplus / "worth about a late pick" equivalence

### What the owner sees
Three plain lines per roster: "Thin at WR — your Best 21 has 1, league teams carry 3", "Surplus: 2 RBs beyond your 21, combined worth ~ a mid 1st", and on any player strip "worth about a late 2nd". The equivalence line is the one owners quote back.

### Inputs
- D17 rosters (PUB); D25 Best 21 membership (in/out of the 21); D26 curve.
- Pick values: `Native.Pick(year, round, original, at) → Dynasty` (internal/app/market/native.go:99-122), which is Pick Value v2 fitted on the records' draft capital + run values — this is the pick-to-PPG language. Its units are the SAME dynasty-PPG units as players, which is what makes an equivalence line lawful rather than a vibes match.
- D20 cap runway for the "can I afford to add" half-line (C1); cap shapes the ADVICE wording, never the arithmetic of needs.

### Method
1. League template: for each position p, `template[p]` = median across all 32 franchises of (count of that position among the team's Best 21). Single source: the Best 21 membership, not raw roster counts — depth beyond the 21 is surplus by definition.
2. Need: `need[p] = template[p] − count_mine[p]`, clamped at ≥ 0; a shortfall of 2+ needs gets graded "critical", 1 "thin".
3. Surplus: rostered players NOT in the Best 21 with adjusted value > 0, grouped by position; `surplus[p]` = their combined value. Players valued 0 or not-valued are listed separately as "the model can't value him" with reasons (mislinked, no draft capital).
4. Pick equivalence for any single asset with value V (player or surplus group): scan next two drafts' pick values via `Native.Pick`; find the pick whose value comes closest to V; output its ROUND-LEVEL label ("late 2nd", "early 3rd") and the band [adjacent-pick-low, adjacent-pick-high]. Constants: `EQUIV_SCAN_DRAFTS` = 2; rounding to nearest fifth of a round (1st, early 2nd, late 2nd…) — fifth-round granularity is what owners actually say.
5. Equivalence wording is rounded DOWN on the owner's surplus and DOWN on his needs' value (a need worth 1.3 "early 2nd" of value is worded "worth about a late 2nd"): never flattering on either side, so advice hedges against overpaying.
6. Cap: if `D20 runwway` says there is no room, the need line adds "; you're capped — this need needs a swap, not a signing." Wording lane only.

### Output shape
- `TeamNeeds{ FranchiseID string; Template map[string]int; Needs []struct{ Position string; Short int; Grade string }; Surplus []struct{ Position string; Players []string; CombinedValue float64; EquivPickBand string }; EquityNotes []string }`
- `EquivPick(playerID|assetRef string) struct{ Label string; Low, High float64; WhichPick struct{ Year, Round, Slot int } }`

### Refresh trigger
Nightly with the value model run (it is downstream of D25/D26); re-equivalences on any pick trade (picks change original team, wildcard valuation shifts) and on a model run — map says nightly; the pick-trade recompute is the case the nightly misses.

### Replay proof
Use the T3 ledger's completed trades from the last 3+ seasons. For each trade: at its date, compute the model's equivalence label for every asset that moved (with inputs as of that date); compare against the picks actually paid. Pass bar: ≥ 60% of assets' equivalence labels land within one round-band of the pick actually received in value terms (sum both sides, compare SUM not per-asset); and in ≥ 70% of trades the side judged "surplus by position" was the side RECEIVING picks rather than the side paying them. The second bar is the real test — surplus identification has got to reproduce actual behaviour, since owners traded before our model existed.
Second proof: needs at mid-season predict acquisition — of the positions our model grades critical at week 8, ≥ 50% were later addressed (trade or significant FA signing) by that franchise in the same season window.

### Failure / thin-data behaviour
- No pick model fitted (no draft history — `pickvalue` returns why, native.go:66-69): equivalences are suppressed, cells say "picks not valued yet: {why}"; needs/surplus still show.
- Expansion league / shallow league: template median is over what exists; if all teams lack a position, template[p] = 0 and "need" is never manufactured there.
- Week 1: identical to nightly — needs are offseason-dated equivalents of that (roster snapshot, no in-season stat deltas yet).

### Open questions
- Does the commissioner want needs computed against the Best 21 (ruling-consistent) or against the whole roster (deeper truth, muddier story)? Map's chain reads D17+D25+D26 → the ruling-faithful default stands; flagging as a real wording choice.
- Should "worth about a late pick" carry the pick's ORIGINAL team (early-Parity picks are worth less per slot than late-Parity)? Pick Value v2 already prices by slot (n.s.lots in NativePickHistory); the wording lane needs to decide whether we surface slot or round only.

---

## 4. D28 — Franchise posture (contend / retool / rebuild) + the trade lens

### What the owner sees
One word above the shelf — "You're set up to CONTEND" — and, inside a trade draft, the same assets valuated THROUGH that word: "you're rebuilding; this trade leans future value, which suits you." The lens is honest about the tilt, it never hides it.

### Inputs
- D05 power BOTH views (`_ViewSeason` and `ViewFranchise`, internal/app/season/power.go:90-97): this-season strength and dynasty strength, already z-blended with age (`AgeZ`, powerrankings/blend.go:63).
- D25 Best 21 league rank; D27 needs/surplus shape; D20 cap runway (max cap is a pressure input); D26 curve for the roster's age arc.
- Draft capital owned: pick values of next 2 drafts' picks in hand via `Native.Pick`.
- Class note: the posture LABEL is PUB (map: "PUB label"); the LENS applied to MY trade draft is DERIV-PRIV (compute per request, never cached league-wide — DATA_FLOW_MAP §1).

### Method
1. Posture score, two axes: `SEASON`: the franchise's percentile on the season-view power board. `DYNASTY`: percentile on the franchise-view board, i.e. Best-21-based value after the age penalty the blend already applies (`FranchiseAgeWeight`).
2. Classify: `CONTEND` = season pct ≥ 0.65 AND dynasty pct ≥ 0.65. `REBUILD` = season pct ≤ 0.35 OR dynasty pct ≤ 0.35 AND (ageZ ≥ +0.3 OR running out of contracts years — cap pressure high). `RETOOL` = everything between. Bands are FIXED constants (not percentiles-of-this-year's-field, which would force one team into rebuild unnecessarily): `CONTEND_BAND` = .65, `REBUILD_BAND` = .35 ASSUMPTION — replay feedback calibrates the two bands, not owner vibes.
3. Reason line, one sentence, built from the inputs that decided the band ("Both boards top third and roster in its window"). LEAGUE-PUBLIC because it is composition of PUB inputs.
4. The lens, for a trade's mirrored sides: `value_lensed(asset) = λ(posture) · on_field_now(asset) + (1 − λ) · dynasty_value(asset)`, where `on_field_now` is the player's current PPG valued to the trade window's remainder of season, `dynasty_value` the D24/D26-folded value as in D25. Constants: λ CONTEND = 0.65, RETOOL = 0.50, REBUILD = 0.30. λ is named and shown on screen as a tilt dial, not hidden in math ("this lens leans 65% on this season").
5. For picks, the lens applies the mirror: `pick_lensed = (1−λ) · pick_value(port) + λ · pick_value(near)`, where near = this/next draft, future = later. A rebuilder's 2028 first is NOT discounted to nothing, and a contender's roster-flagged downgrade is honest.
6. Output surfaces both the raw (λ=0.5) and the lensed deltas of every trade draft column, so the owner always sees what the tilt did. The verdict words "favors you / favors them / fair" are computed on the LENSED values only, with the lens worded on the page.

### Output shape
- `Posture{ FranchiseID string; Label string; SeasonPct, DynastyPct float64; Reason string; Lambda float64; AsOf }` (PUB).
- `LensResult{ AmberDelta正版, LensedDelta []struct{ PlayerID string; Raw, Lensed float64 }; Verdict string; LambdaShown float64 }` (DERIV-PRIV, per request).

### Refresh trigger
Nightly with the model run (it reads D05's nightly boards); lens runs only on trade-draft requests, never cached.

### Replay proof
Rewind each of past 4 seasons to the each midseason point (week 8). Classify every franchise's posture with inputs as of that date. Pass bar 1 (does the word mean anything): teams labelled CONTEND at week 8 made the playoff field that season ≥ 65% of the time; teams labelled REBUILD earned a top-quarter pick the next season ≥ 50% of the time. Pass bar 2 (lens history): of completed past trades flagged in the ledger, the model's lensed verdict (as of tate-of-trade) must agree with the direction each side actually moved value in ≥ 55% of cases, and disagree nowhere by more than one round-band equivalent. Pass bar 3: posture flips season-over-season at most once per franchise per 2 seasons on average (a model that says REBUILD on Tuesday and CONTEND on Thursday is noise).

### Failure / thin-data behaviour
- Early season / no power run: posture shows "the boards aren't scored yet" and the lens runs at λ = 0.5 (neutral tilt, i.e. raw value), never refuses.
- Expansion team mid-dynasty build (everything young, picks-heavy): the age-based rebuild trigger fires gently, and the reason line says "everything young and cap-tight" rather than "you're bad"; a wrong-feeling word on a 3-team expansion league would be worse than none — pass bar 3's replay protects against this.
- λ constants are the thing most likely wrong for one owner's taste: this is why the lens shows both deltas, so the owner can decide against the tilt (the tilt is advice, not a verdict).

### Open questions
- When the season board and dynasty board disagree wildly (old contender in his final push): which band wins? Proposed: REBUILD_band OR-fires (season ≤ .35) floods, while a season-strong/dynasty-weak OLD roster reads RETOOL-not-rebuild ("selling his last window"). Needs the commissioner's ear, not a silent default — flagging as a REAL ruling.
- Should POSTURE itself be public (PUB, per the map) even though "this team will trade picks for a QB" leaks some intent? Map ruled the label PUB already; upholding that, flagging the sharpness of the reason line.

---

## 5. D32 — Tune-it: TRIM / ASK / SWEETEN for a trade draft

### What the owner sees
Under a trade draft, three (at most) suggestions, one word each: "TRIM: take out Darnell Mooney — still fair without him", "ASK: ask for a 2027 2nd — you're a round light", "SWEETEN: add your 2027 3rd — cap-safe, and it's your surplus". One tap each to apply, each with the verdict it would produce shown before applying.

### Inputs
- D29 the trade draft itself (store/personal version/pocket) — PRIV inputs, so the whole model is DERIV-PRIV: computed on request, never league-cached (DATA_FLOW_MAP §1 row DERIV-PRIV).
- D28 lens for both parties (each side gets HIS OWN lens λ on his own assets — the mirror of what a rival's posture says, computed from that rival's PUB posture, never his private data).
- D27 needs/surplus for both franchises (PUB); D33 lineup delta so a suggestion can see who it benches; D20 cap runway for both (a sweetener that busts my cap is filtered out at source — C1's check belongs to the suggestion's legality, per map chain C1).
- Existing reuse: `powerrankings.Lineup` for the "does this asset start?" check inside TRIM (internal/powerrankings/lineup.go:43), `Native.Pick` for pick-valued sweeteners.

### Method
1. Compute the gap: `gap = Σ lensed(mine out) − Σ lensed(theirs out)` under each side's own λ (D28 step 4). Fair band: `|gap| ≤ FAIR_BAND` where `FAIR_BAND = 0.5 × one early-2nd-pick's value` ASSUMPTION — the band IS the definition of "fair" in this model, so it is named and calibrated by replay, not vibes.
2. If fair: no suggestion, article worded "already fair" — the model stays quiet rather than inventing a reason to meddle.
3. If I am OVERPAYING (gap > +FAIR_BAND): propose TRIM or ASK, in this order.
   - TRIM: sort my assets by lensed value ascending; for each, compute the gap WITHOUT it; the first removal that lands the gap inside the band is "TRIM: out {asset} — still fair". Prefer assets that (i) are NOT in my Best 21 (D25 out-members), and (ii) are NOT position-fillers of an active D27 need. If no single removal lands in-band, TRIM is not offered silently — fall through to ASK.
   - ASK: find the smallest asset NOT in the draft on their roster whose value ≈ the gap − FAIR_BAND; prefer their D27 surplus at a position of MY need (a need too — the ask fixes two problems). One candidate, with its pick-equivalence label from D27's `EquivPick`.
4. If I am UNDERPAYING (gap < −FAIR_BAND): SWEETEN — the smallest addition from my roster outside the draft that closes the gap, strongly preferring (i) D27 surplus (not Best 21), (ii) a position that fills ITS need so the trade is likelier accepted, (iii) cap-legal both sides. Pick sweeteners land as "your {round}"; players as "add {name}".
5. Every suggestion shows, before apply: the new gap, the new verdict word, and the D33 lineup delta it will cause ("adds 2.1 PPG to your starting lineup" or "you'd be starting him" — a TRIM that benches someone gets that flag first, because "still fair" is worthless if your lineup breaks).
6. At most THREE suggestions total, ranked by the one the model most endorses: the surplus-first / need-first suggestion outranks the raw-value-closest one. The page never stacks ten nudges; one clear line wins.

### Output shape
- `TuneSuggestions{ Gap float64; Fair float64; Items []struct{ Verb string /* TRIM|ASK|SWEETEN */; Asset PlayerID-or-PickRef; NewGap float64; NewVerdict string; NewBest21Change string; CapIllegal bool; Endorsed bool } }` (DERIV-PRIV, per draft request).

### Refresh trigger
Only on request (opening/touching a trade draft, or after a suggestion is applied); never nightly — this is conversation-local math on the draft in hand.

### Replay proof
For each completed trade in 3+ seasons of ledger history: rebuild BOTH franchises' postures, needs, and lenses as of the trade date; run D32 against the ACTUAL trade as it stood at accept. Pass bars (all three hold):
(i) Legality — every SWEETEN/TRIM/ASK the model offers at that date is cap-legal-at-that-date per D20's replayed runway (checked, never assumed).
(ii) Fairness — applying the model's single endorsed suggestion moves the actual trade's gap inside the fair band in ≥ 70% of the trades it offered suggestions on.
(iii) No-harm — in trades the model called fair already, it offers zero suggestions ≥ 95% of the time (measured: silence is the failure mode worth replaying; a tuning engine that can't shut up is worse than none).
(iv) Posture honesty — suggested ASK/SWEETEN assets never exceed the trade's gap by more than one round-band equivalent (checked per trade, so the model never "asks" for more than it knows).

### Failure / thin-data behaviour
- An asset in the draft can't be valued: no suggestions at all, one line — "the board can't value {player} yet (no model run / mislinked); tune-it waits for a fair number for him" — never a suggestion built around a named-but-unvalued hole.
- Both sides capped hard: SWEETEN falls back to non-cap salary shapes only; if nothing cap-legal closes the band, the page says so plainly ("nothing affordable closes this — trim or ask instead").
- Week 1 / offseason: works identically, but on-now vs dynasty tilt shows the offseason wording ("leaned on future value: league between season windows").

### Open questions
- Should TRIM/ASK/SWEETEN send anything to the OTHER owner's draft when accepted (a counter-draft clock), or is D32's authority strictly advisory-verbal? Map routes D32 → WR shelf/verdict, but D30 threads are the PAIR store — who carries a round-trip counter is threading design the model isn't in; leaving it as a real open thread question for build.
- The tolerable gap asymmetry (owners generally hate underpaying more than overpaying): should FAIR_BAND tighten when the model says I'M underpaying (willing acceptors historically accept wider overpays)? Marked ASSUMPTION, replay bar (ii) is the measurement that answers it.

---

## 6. D33 — Lineup delta of a trade (who it replaces, points per week)

### What the owner sees
"Who does he start FOR? Your top-2 lineup gains 4.2 PPG; he replaces {starter} in your lineup, {other starter} goes to your bench." Plain, minute-detail: who enters/leaves the starters, who gets benched, points per week.

### Inputs
- D17 lineups +/- roster snapshot of BOTH parties and D29 the trade draft; D11 projections (per player PPG forecast, step 7's native feed).
- `powerrankings.Lineup(cands, rules)` (internal/powerrankings/lineup.go:43) — REUSED AS IS: we hand it (roster ± traded assets, projections as Candidate.Value) against the league LineupRules from leaguerules; it computes the best legal lineup greedy-fill and returns indices. DO NOT re-implement an optimizer; the greedy fill (minimums first, remaining slots to highest remaining value) is the existing, tested math.
- `powerrankings.ExpectedPoints(lineup, pointsFor, games)` (lineup.go:77) to blend early-season actual PPG with projections where weeks played > 0 (adds season results to a raw projection at the `RosterCredibilityWeeks` rate — the model carries no separate credibility fit of its own).
- `powerrankings.WinProbability(a, b, GameSpread)` — only if the page shows "wins added" in the week race; it does NOT uphold the whole playoff thing (D06 does that).

### Method
1. Build the before/after candidate lists for ME: every rostered player plus the draft's incoming assets, projected PPG − projected injury discount (D15 return week assumptions; "out weeks" are named on the tile). Value = `ExpectedPoints` when games > 0 else raw projection.
2. `Lineup()` against the league's actual LineupRules → before-total, after-total; `delta = after − before`.
3. The replacement map: for each STARTING slot in before, find its player in after. Outcomes = (i) same player still starts (unchanged), (ii) replaced by an incoming asset — replaced, (iii) starter now benched — demoted, (iv) incoming starts where nobody did — new starter (a lineup hole filled). Each slot change gets named by the incoming/outgoing asset.
4. Per-week shape: a delta for THIS coming week (players on bye — D16 — and "out" injuries excluded from the after-candidates, so a trade "adding" a player who doesn't play this week does NOT fatten the number against that week); a rest-of-season delta (all weeks remaining including the week of the trade); a next-week/this-week distinction worded because owners quote ONE number, and that number is the rest-of-season PPG/week.
5. Same pass for the OTHER party — both deltas shown ("what you get / what they get"), so each page answers the legitimacy of both sides at once.
6. The delta is DERIV-PRIV because the draft is private. Nothing here feeds into a league-wide cache; each screen fetch serves its request only, mirroring map's C6.

### Output shape
- `LineupDelta{ FranchiseID string; BeforeStarters, AfterStarters []struct{ PlayerID, Position string; PPG float64 }; PerWeek { Week, RestOfSeason float64 }; Changes []struct{ Kind string /* enters|replaces|leaves|demoted|new-starter */; Incoming, Outgoing PlayerID; OldSlot, NewSlot string } ; Verdict string }` (DERIV-PRIV, both parties' results in one response).

### Refresh trigger
On trade-draft request (with D29); also re-computed on any projection refresh while a draft stays open — a person leaves a draft open overnight.

### Replay proof
Take mid-season trades from 3+ seasons of ledger history. At each trade date, compute the model's per-week PPG delta for the acquiring team with all inputs as of T. Pass bars:
(i) Direction: teams the model said gain ≥ +2.0 PPG/week actually improve their points-forward rate in the weeks AFTER the trade in a majority of these historically-labeled trades (sign test, α ≥ 0.05 across seasons pooled).
(ii) Magnitude calibration: actual post-trade PPG change vs predicted — Pearson |r| ≥ 0.4 with a slope between 0.7 and 1.3 (calibration beats correlation-only passes).
(iii) Replacement identity: players the model named as "BENCHED" actually lose starter starts in ≥ 70% of cases in the following week's lineup (checked from the league's actual posted lineups, not from inference — this is the strongest human-truth test in the family).
(iv) One-by-one: the model's rest-of-season delta (with all weeks live) and per-week delta must AGREE in sign 100% of the time; if they ever disagree, the week-excluding logic is wrong, not the trade.

### Failure / thin-data behaviour
- Rookies / players with no projection: they enter the candidate list at the native fallback (their historical rookie production band, D42 tiers for a judgement), and the tile says which numbers are deriver from a projection vs a tier fall-back. Never silently 0 a traded-for rookie.
- A trade involving an IR-resident player: the "who starts" list treats him as a candidate from his return week; the tile states "he joins the lineup from week {n}" rather than puffing this week's delta.
- Empty lineups at a franchise (usage weirdness, a roster blesssed mid-season): the tile's before-lineup locks into whatever MFL's actual STARTERS were rather than the model's parsed optimal best lineup — the delta story is about the team's true inefficiency, not the model's imagined perfection; worded "you're currently starting X, so the real effect of this trade differs from the optimal".

### Open questions
- Should the delta include IDP slot-level replacements in the same pass, or is offense-only wording enough for the first ship? (Real build-scope question: IDP positions in LineupRules are supported in the existing function, but the verbal clarity falls fast with 6 IDP slots + 2 flexes described in text.)
- Bye-week-EXCLUDED per-week delta: owners feel that number differently ("I lose 5 PPG this week but the window's after"), but the WR mock shows ONE number — which week's number wins the headline when they disagree? Real wording ruling needed from Christopher's run through the mocks.

---

## 7. D11 — Native projections: weekly floor / median / ceiling, position rank, rest of season

### What the owner sees
Per player: a range, not a single number — "floor 6.1 · median 12.4 · ceiling 21.8 (WR#14) — rest of season 138". Owners weigh trades on the RANGE: "his ceiling is the reason I'm holding".

### Inputs (and one honesty note up front)
- HONESTY: today D11 is "weekly projections via marketfeeds (MFL-sourced)" only (map §10, row D11); the NATIVE part — our own floor/median/ceiling — has 0 hits. This section is the native method.
- This league's historical scoring: fantasy points per week per player from nflverse via fantasyscore (D54 feed, keyed by our player ID through `playerrecord.Book`).
- Prior seasons' same-scoring production (the same source, seasons ≥ 2 back).
- D14 injuries (hourly) and D15 return week: cap or zero-out a projection when applied.
- D16 schedule (byes, opponents).
- OPPONENT DEFENSE: nflverse team-defense vs position — REQUIRES D13 usage/def-vs-pos (map: NEW, L4). Until D13 lands, native projections run WITHOUT a matchup factor and the tile says "no matchup adjustment yet". Named honestly here so the build order doesn't pretend.
- NOT MFL's projections as an input to the math — they stay as a separate displayed source ("MFL says") so the native number can disagree with them visibly, which is the point of building native.

### Method
1. Distribution: per player, weekly PPG scores as the distribution, recency-weighted (weight w = `RECENCY_HALF_LIFE_WEEKS` = 8 by half-life over finished weeks; prior seasons enter at a flat prior weight = 1/3 of this season's so far-week ASSUMPTION — replay picks the prior weight). `MIN_SCORES` = 8 before we trust his own distribution.
2. Quantiles: floor = p10, median = p50, ceiling = p90 of the weighted replay of his historical weeks, all in THIS league's scoring.
3. Shrinkage toward position mean: blend = `MIN_SCORES/(MIN_SCORES+games)·player_distribution + rest·position_week_distribution` (position weekly p10/p50/p90 across all players at the position, same source). New-to-league players land near the position distribution, weighted by draft capital via D24's value tier (not by ADP — value is value).
4. Matchup (once D13 exists): multiplier per opponent-vs-position rank band from the last 8 NFL weeks — a MEAN-REVERTING band multiplier (±10% at the extremes, fractions between), `MATCHUP_CAP` = 0.10. It shifts median and ceiling, never widens the floor range by much (floor barely moves; matchup is about how good a good day gets).
5. Injury modifiers (D14/D15, applied joint-last, monotone downwards): OUT → projection is the backup's; DOUBTFUL → floor halves, ceiling −45%; QUESTIONABLE → ceiling −20%, median −8% (constants named `INJURY_*`, calibrated in replay); unknown return week (D15 "") → ceiling −30% with "return unknown" wording.
6. Rest of season: Σ remaining weeks' medians minus one bye week (D16), injury caps held for the season. Expressed as "rest of season ~138 points, median lane".

### Output shape
- `NativeProjection{ PlayerID string; Week int; Floor, Median, Ceiling float64; PosRank int; PosN int; MatchupMult float64; InjuryNote string; Season int }`
- Rest-of-season: `ROS{ PlayerID string; Points float64; PosRank int; WeeksRemaining int }`
- One row per player per week for the remainder + ROS, published dail (daily refresh per map).

### Refresh trigger
Daily (map: "daily") — combined with an hourly injury pass that publishes ONLY the injury modifiers to avoid a full recompute.

### Replay proof
Rewind week-by-week across the last 2–3 seasons and score three things (the SAME test an operator would keep running weekly in production):
(i) Coverage: the fraction of actual weekly scores inside [floor, ceiling] must land 78–90% of the time pool-wide (if coverage is 60%, our ranges are lies; if 99%, our ranges are useless). Pass bar: ≥ 75% AND ≤ 92%.
(ii) Median truth: the median's week rank vs position correlates with the actual finish, Spearman ≥ 0.45 at QB/RB/WR/TE (and reports honestly where it fails).
(iii) Shrinkage wins: for players with < 8 games of history, the shrunk projection must beat the player's raw history alone (lower RMSE on the next week) — the shrinkage is not cargo.
And one league-level bar every season: the ROS pre-season projection (sum of medians before week 1) ranks teams' actual final points-forward within reason — Spearman ≥ 0.5 vs actual final PF rank, because this ROS anchor feeds D05 and D06 downstream.

### Failure / thin-data behaviour
- Rookie week 1: he IS the position distribution, dampened by draft capital; the tile words "projected off the position shape — no NFL history yet" rather than pretending we know his ceiling.
- 2020-style resume gaps or team-change mid-season: the distribution splits by team; the CURRENT team's weeks get the recency weight.
- A player out long-term (D15 return unknown): the projection carries a ceiling haircut and "return unknown"; ROS applies the haircut to every week until D15 resolves it — wrong until told otherwise, but visibly.
- A new league with no scoring history (expansion): projections fall to the LEAGUE-WIDE position distributions with a "no scoring yet in this league" note; until week 4ish there will be nothing league-specific to learn, and the tile says so.

### Open questions
- Does this league want floor/median/ceiling from NATIVE HISTORY or industry ADP bands too? (Real: with MFL-sourced projections held separately, our native number can be visibly compared — worth a screen placement ruling from the WR mock.)
- Is the ±10% MATCHUP_CAP the right knob for the commissioner to own as a league setting, or should it be hardbaked and invisible? (Setting-owning leagues customize too much; the commissioner asked for plain language but sets the default.)

---

## 8. D05 — Power ranking components FORM / LINEUP / HEALTH / ALL-PLAY / RUN-IN + "why it moved"

### What the owner sees
One board, five named columns. A team at power 4 shows why: "FORM +1.4, LINEUP +0.9, HEALTH −0.6…". And below any rank change: "moved up 3: your form and your easy run-in did it" — one sentence, the component that moved him named.

### Inputs
- D12 finished-week points and results (form), already keyed per franchise via MFL results.
- D17 rosters + D11 projections → `powerrankings.Lineup` (starters' projected PPG, the LINEUP component).
- D14/D15 injuries → HEALTH's haircut (how much of the starting lineup is expected to play).
- D04 all-play records via `engine/race` Table (already built; `race.Table.addScores`, race.go:131-147, counts all-play for every scored week including playoffs, matching MFL's standings).
- Remaining schedule from D16/MFL fixtures → RUN-IN via `powerrankings.ExpectedWins(games, expected)` (lineup.go:97), which already computes expected wins per franchise from `WinProbability(a, b)` (lineup.go:86, `GameSpread` = 50 league points, existing constant).
- D25 Best 21 for the dynasty-horizon board (DYNASTY view stays as Blend has it now — 3 components, roster + performance − age; the FIVE components are the SEASON view's upgrade).
- EXISTING MATH REUSE: `powerrankings.Blend` z-scores components then weights then min-maxes (blend.go:73-134). HONESTY: Blend currently takes exactly 3 inputs (Roster, Performance, Age); D05's five-component season view needs Blend generalized to N components of the same z-then-weight shape — an extension of the existing function and its z conventions (median/MAD for heavy-tailed value components, mean/std for bounded ones), NOT a new blending engine.

### Method
1. Five components, computed per franchise:
   - FORM: z of (points per game last 3 weeks / points per game so far), or the all-play record over those weeks (equivalent; results language reads better): last 3 weeks all-play win% z-scored (3 is `FORM_WEEKS`; short samples fold into the season number).
   - LINEUP: z of `ExpectedPoints(Lineup(roster, projections), pointsForSoFar, gamesPlayed)` — this week's expected lineup total.
   - HEALTH: z of (1 − projected-points haircut): starters' combined D11 projections at full health vs with injuries applied; the fraction weights all things a busted roster loses.
   - ALL-PLAY: z of all-play win% (from race Table), season to date.
   - RUN-IN: z of ExpectedWins(remaining schedule)/remaining games — "how hard the road to the end is".
2. Power = weighted sum of the five z's; ONE weight vector of five, `W = (FORM .20, LINEUP .25, HEALTH .10, ALL-PLAY .25, RUN-IN .20)` ASSUMPTION, fitted by replay (step "Replay proof" below), NOT hand-picked every year. The existing `DefaultRosterWeight`-style philosophy carries: the weights are fitted once and stored, not owned by a screen.
3. "Why it moved": at each weekly run, store the component z's (they are already stored per Row in blend.go — extend to five). RankDelta decomposes as: for each component, Δz since last run × weight; the moved-by narrative = the component with the largest |contribution|, phrased in plain words ("your run-in eased — three home dates"), the second named if it matters. Never narrate a move into existence: if |Δrank| = 0, no "why" renders.
4. The DYNASTY view keeps its 3-component Blend exactly as built (roster + results − age), fed by D25. The SEASON view is where the five-component board lands, so a lineage/copy change in one never distorts the other.
5. Injury-hourly refresh: an injury event re-runs ONLY the HEALTH/LINEUP components (cheap: they depend on rosters + projections) and reweighs the board without touching FORM/ALL-PLAY/RUN-IN, so the board can breathe mid-week without redoing everything. Map already says "weekly + injury".

### Output shape
- `PowerBoard{ Season, View, Week, Rows []struct{ FranchiseID; Rank, RankDelta int; PowerScore float64; ComponentZs struct{ Form, Lineup, Health, AllPlay, RunIn float64 }; MovementNarrative string; Weights []float64; ModelRunID int64 } }` (PUB, league-wide lineage-friendly).

### Refresh trigger
Weekly (game-final each week) + hourly-on-injury (only HEALTH/LINEUP recomputed); map's "weekly + injury" exactly.

### Replay proof
Backtest the five-component board over the completed seasons the ledger already has, run-as-of-each-week:
(i) Predictive bar: the season-view PowerScore at week 8 must out-predict final all-play win% than ALL-PLAY alone (or no board exists to defend it); incremental-v-marginal check: board's Spearman vs final all-play ≥ 0.70 pool-wide, each of the last 4 seasons.
(ii) Ablation (the weights-fitted proof): each component dropped in turn; NO single component's removal may improve Spearman (if it does, the weight was wrong, refit, rerun the ablation). The five-notch shape comes from data, not taste.
(iii) Spread sanity: the PowerScore spread across 32 franchises must stay in [0.05, 0.6] (else one whale crushes the field and the board reads as two leagues; the existing z-scale approach targets exactly this).
(iv) "Why it moved" checks: in ≥ 80% of rank moves of |Δ| ≥ 2, the component named as the driver did in fact move by more than its residual noise band that week (measured from the z-history); if it fails, our narrative is a hallucination we can measure, kill it.
The dozen-or-so constants weight the board once in this pass; a change after a season is a refit, and the fit is versioned in the commit that makes it (nolint-exemption precedents already established in the repo style).

### Failure / thin-data behaviour
- Weeks 1–2: FORM z's are degenerate and RUN-IN has little left to measure; blend weights shift to LINEUP/HEALTH/DYNASTY ((existing `AutoRosterWeight(weeks)`-style ramp (powerrankings/blend.go:36) generalized to all five), and tiles show which weights are carrying weight, labeled "early-season weights" on the page.
- A bye week's blind slate: LINEUP is computed anyway on who's NOT on bye; HEALTH accounts for the byes as known downtime and not as health.
- A franchise with no scores submitted (a broken byes edge): he z's toward the neutral rather than the zero, PORTS from last week's z's (aged decay) — n "he wrote a 0" row, and the board writes "no scores yet" instead.
- A shortened season: run-in surpassed whatever the calendar holds; the board shows fewer weeks without pretending the season is longer.

### Open questions
- Do owners want the five z's VISIBLE as a table, or one sentence per row? The existing PowerRow already exposes z's (season/power.go:22-25) so the data's there — the WR mock's density ruling decides.
- When a commissioner weights ALL-PLAY above ALL others (results-eisethics league) — is the weight vector a league SETTING (params) or is W fitted once app-wide? Real ruling: the params system exists (params.Set), the fit says one-canonical works — flagging that the commissioner may want W exposed but the fit disagrees.

---

## 9. D06 — Playoff simulation: playoff odds, bye %, magic number, "this week moves it"

### What the owner sees
"Playoff odds 78% · first-round bye 23% · magic number 4". And tonight's constant companion: "If you win this week, odds go 78→91; lose, 78→62." The number that tells the race's truth, before the race calls it.

### Inputs
- D02 standings at the current week via `engine/race.Build` from MFL weekly results (race.go:78) — the same Table the app already holds (app/season/playoffs_reading.go, View.Table).
- Remaining schedule: D16/MFL fixtures, i.e. `race.Table.Line(id).Remaining` (race.go:64) plus each game's pairing.
- Stage-strength feed: each franchise's expected score from D05's LINEUP component and Points/For results — the SAME expected-score inputs `powerrankings.ExpectedWins` reads, so odds and the power board can never disagree about who's favored in a given game.
- The league's tiebreak keys: `race.ParseSort(StandingsSort)` (playoffs_reading.go:245) and division/conference shape.
- LOCKED BUILD RULE (map C3): the sim MUST call `race.Table.Seeding` (seeding.go:37) and `race.FirstRound` (seeding.go:68) to resolve each sim's bracket; it NEVER re-implements seeding or byes. `race.Table.Bubble` (seeding.go:90) feeds the "on the bubble" framing around the odds.

### Method
1. Per-sim run: for every remaining regular-season game, draw each team's score… NO — draw from a simpler truth: for each game, winner by `WinProbability(expectedHome, expectedAway)` plus a score-noise draw around each expected score of σ = `GameSpread` (existing, 50). Noise on the score not just the win/loss bit: because tiebreakers (points-for!) live inside `race`'s tiebreak chain, the sim must score games to let them decide.
   ASSUMPTION (real, flagged): normal score noise is convenient but real FFC margins are heavier-tailed; the replay proof (i) calibrates this — if coverage is off, switch the noise distribution (e.g. resampled historical margin residuals) BEFORE shipping anything else.
2. For each of `SIM_N` = 20,000 iterations (named constant; timing budget check at build): run all remaining games week-by-week, then reconstruct a `race.Table`-equivalent standing AT THE FULL SEASON RECORD by adding the simulated games to the CURRENT table, then call `race`-equivalent `Seeding` per conference.
   BUILD NOTE: race.Table is built from `leaguefeed.WeekResult`; the sim fabricates the remaining weeks' WeekResults and re-runs the SAME code path (race.Build + race.Seeding) — one engine, one truth; per DATA_FLOW_MAP, "the sim must call engine/race Seeding, never re-implement it."
3. Per franchise counters across sims: `playoffOdds` = share of sims reaching the field, `byeOdds` = share landing a seed ≤ byes (from FirstRound), `divisionOdds` = share winning the division, `wildcardOdds` as the residual, recorded as one row per franchise.
4. Magic number (exact arithmetic, no sim): the minimal k such that, with MY next-k wins and the LAST SEED'S next-k losses forced, my seed is guaranteed EVEN ON ALL TIEBREAKS the race knows. Computed by the forced-trace: run the seeding once with forced outcomes (this is cheap — one sim) and increment k until guaranteed; replay bar (iv) checks it against bread-and-butter hand-computed cases. Worded "magic number for the last seed" per franchise, which is the honest claim.
5. "This week moves it": run the sim twice more with this week's outcomes forced (my win / my loss) → the two deltas, as in the what-owner-sees line. No third scenario, no tree of futures; owners cope with one fork at a time.
6. DIVISION/WILD separation in the response so the WR/HQ tiles can show "leading the North" vs "chasing a wild card" correctly.

### Output shape
- `PlayoffSim{ Season, Week, SimN int; Rows []struct{ FranchiseID string; PlayoffOdds, ByeOdds, DivisionOdds, WildOdds float64; SeedNow int; MagicNumber *int /* nil when already seeded or out */ }; ThisWeek struct{ IfWin, IfLoss map[string]float64 /* per franchise playoff odds */ } }` (PUB).

### Refresh trigger
Game final (map) — each finished game shifts odds; and nightly as a cheap anchor (injury-driven moves don't wait for game finals), on the same run as D05/D11 feeds.

### Replay proof
The strongest proof in the family, because odds so easily lie:
(i) Calibration (the headline test): back-cast every week of the last 4 seasons from week 3 onward against what ACTUALLY happened. Bucket predicted playoff-odds into 10 bands; actual in-band rate must land within ±7pp of the band's 10-band diagonal line shape (e.g. claimed-70%s must happen between 63% and 77%). Same for bye odds. Below passing → do not ship, recalibrate the noise model per step 1's flag.
(ii) Brier score ≤ 0.22 for playoff odds {made it} at week 8 across seasons, ≤ 0.30 at week 4 (earlier weeks are harder; both bars must pass, no cheating by week-claim).
(iii) No-degenerate-league check: the sim at a fully-determined week's end (post-season, race finished) must return 0%/100% exactly, never 2%/98% — a pure code-truth test of the whole sim block.
(iv) Magic number: exhaustive unit-replay across the smallest historical division races (hand-computed table for those seasons) — every hand-computed magic number must equal the model's; plus a property test that "magic # accomplished" ⇒ seeding guaranteed in the NEXT forced-trace, and that magic k−1 is correctly NOT guaranteed.
(v) ThisWeek honesty: the DELTAS the model shows (IfWin/IfLoss) must bracket the true post-week outcome in the historical replay; measured as: the fraction of weeks where the actual result's implied odds fall between ifWin and ifLoss bands ≥ 85% (when the week's actual result is at one of the two forced ends, the number shown BEFORE the game must land on that side in ≥ ¾ of cases, else our per-week narrative is a lie).

### Failure / thin-data behaviour
- Week 1: RUN-IN uses no completed weeks' truths yet; the sim runs on D05's offseason expected scores and publishes plainly "early-season: injury and lineup change will move this often."
- DIVISION/CONFERENCE-less leagues: map's conference/division shape changes which branch runs; the tiles show what the league structurally HAS (all-wildcard race), never defaults-by-good-luck to a 12-team-style field.
- Ties in the sim's fabrications: include ties (moments when the score noise leaves them level) at their real频率 — a league never seeing tie simulations means the noise model is miscalibrated downward, reported via (i) quietly.
- Tiebreak data missing (MFL power rating absent etc.): `race`'s seeding already SKIPS that tiebreak (playoffs_reading.go:169-171 wording "Skipped: …"); the sim's replay inherits the skip — correct, and worded "decided level; the tiebreak order carries it".

### Open questions
- What does "magic number" mean when the racing positions include the BYE (teams racing for a bye, not the last seed)? Proposed: one per goal — "magic for a bye", "magic for the field" — two numbers, one tile. Confirm that's what the WR tiles have space for.
- Do we show opposing fans the SAME odds (one PUB table) or personalized "my path" wording? PUB is per map; personalization is only wording lanes via D11-style per-franchise lenses (already the map's pattern for D52), no numbers may differ.

---

## 10. D35 — Bid market: rivals, estimated winning price, trend, win odds (revealed history only)

### What the owner sees
On a free agent: "Gone for ~ $12 lately · 3 rivals can pay · your $8 wins about 1 time in 5". The four truths of the auction, computed ONLY from what the ledger already shows everyone: past revealed winning bids.

### Inputs (isolation rule first)
- The T3 ledger's REVEALED bid history (bids are SEALED until window close — Q1/Q4 rulings 10-30; everything here is a function of the ledger's closed windows, never a peek at an open one). This is the model's clearest isolation constraint and its reason to exist: "rivals" is computed from public needs + cap, NOT from sealed bids (map C5, verbatim).
- D20 cap runway per franchise (who has room — PUB).
- D27 team needs (who is short at a position — PUB).
- D34 the FA pool being bid on (PUB pool; the SEALED half of D34 is REFUSED as an input by construction — build-time test 2 covers the refusal).

### Method
1. Training set: every completed FA window in ledger history → for each won auction: (asset, winning price, player's value then, position, week-of-season, rivals' cap states at the time). No bid counts as data until the window closed and landed in the ledger.
2. Estimated winning price: regress revealed winning bids against (player's D24 value at the time, position, week of season) — starting shape is a per-position fraction `price ≈ RATIO(pos, week) × player_dynasty_value`, fitted with recency weighting (`PRICE_HALF_LIFE_WINDOWS` = 6 windows), then refined by the residual distribution. Output: median + a "Gone for ~" band (p25–p75). Round to the league's bid increments.
3. Trend: the ratio's slope over the last N windows (won prices rising vs flat vs falling) phrased once ("prices this year run ~15% over last") and shown as the history sparkline; never a separate per-player claim without at least 3 previous windows of the same position.
4. Win odds for MY bid b: empirical P(my price ≥ winning price | asset class) from history, with the tail stretched by the number of revealed RIVALS this window (step 5): `P(win | b) = base_curve(b / est_price) adjusted × rival_load` where `rival_load` grows the expected top price without pretending we can see anyone's actual bid. Blend as a small monotone adjustment, not a re-estimate (the bid the owner places stays his decision).
5. Rivals: franchises with (D20 cap room ≥ my estimated winning price) AND (D27 need ≥ 1 at the asset's position), excluded my own franchise. Count + the nearest three named plainly by cost capacity — their need count is public (D27), never their intent. This is the ONLY place "rival" appears, and it is built from two PUB numbers, both of which the owner can verify by hand on the same page.
6. The tile also renders "what the last 3 like him went for" straight from the ledger, so the estimate never floats free of its evidence.

### Output shape
- `BidMarket{ PlayerID string; EstPrice { Median, P25, P75 float64 }; Trend struct{ SlopeLabel string; LastN int }; WinOddsAt map[bid increment]float64; Rivals []struct{ FranchiseID string; CapRoom float64; NeedPosition string } ; LastRevealed []struct{ WindowClosed int; Price float64 } }` (PUB — every input is, so the output is too).

### Refresh trigger
Window close (each closed window is a new data point); nightly otherwise (cap and needs drift). It never refreshes hidden data mid-window — a mid-window re-compute could leak nothing (inputs are already public), so it is safe, but it wastes nothing either; nightly stays.

### Replay proof
Replay completed windows with the model built only on data before them:
(i) Price calibration: the model's "Gone for ~" P25–P75 band must contain the ACTUAL winning price in ≥ 50% of past windows, tightening the median absolute error to ≤ one bid increment for the price worded tables; measured per position pool ≥ 30 windows.
(ii) Trend honesty: when the model says "prices rising", realized prices in the following windows must exceed the trailing median in ≥ 55% of cases (a real trend phrase, not decorative).
(iii) Win-odds calibration: group MY historical (revealed) bids by the odds the model would have claimed; won-fraction inside the odds band's ±10pp (e.g. bids where we claimed 1-in-5 must have won in [10%, 30%] on average).
(iv) Rivals: among franchises the model named as rivals, ≥ 40% actually outbid the eventual winner's runner-up OR finished top-3 in that window's revealed ladder; if our rival list never overlaps who actually paid, the rival composite (room × need) is wrong, not the framing (measured across all windows).
This model is the one where "correct first" bites hardest — bad estimates here cost owners real leagues PPG capital, and there is no second source to lean on once the public ruling stands.

### Failure / thin-data behaviour
- New league, no revealed history: the tile refuses to pretend: "no closed windows yet — the market learns, we don't guess" — the four numbers render as "—" with the source line explaining what永远 fills them.
- Fewer than `MIN_WINDOWS_FOR_TREND` = 3 similar windows: trend says "not enough history"; the estimate still renders from whatever exists with a wider band.
- A player nobody has bid on before but similar assets have: nearest-neighbor across the position's class (same D24 value bucket, same position), always labeled "from nearest neighbors" — never a silent proxy.
- Mid-window freezes: D35 does NOT re-estimate within an open window (estimates are anchored at window start); a re-anchor mid-window would move the goalposts the owner already read.

### Open questions
- Do we surface RIVAL NAMES or only a count? (The composite (room, need) reveals who's loaded at QB, which is true public info any owner can hand-build; but naming may be read as surveillance. Map's C5 wording says "rivals = owners with room + need" — names are public, the count-first tile pages decide.) Real ruling wanted from Christopher's read of the WR mock.
- Is the tick "your $8 wins 1-in-5" honest when MFL's bids are blind within a window? Our curve says yes only because revealed history is blind-bid distribution data; still, the wording lane ("about 1 time in 5") is set deliberately vague — confirming owners don't read it as a promise.

---

## 11. D42 — Prospect tiers + "likely still there at pick #N"

### What the owner sees
The board as tiers, not a 260-row list: "Tier 1 · 4 backs, then a cliff" — and per player, the honest offer: "still there at pick 8: 40%." How the board breathes when your pick comes up.

### Inputs
- `ingestion/marketfeeds` ADP, ranks, AAV feeds (map §10, D42 PARTIAL — feeds exist; tiers and "likely still there" are 0 hits, this section is that build).
- The app's OWN rookie draft history (draft bodies / `commissioner.PickValueDrafts`, pick_history.go:10) — real drafts this league (and held peer leagues, wherever pick history has them) actually ran.
- D41 draft order/positions (PUB), plus each pick's known slot when slots are explicit.
- D80 college context: teaser only in the tile ("his college usage says…"), ALWAYS optional, never a value input (it's L4 and thin).
- `pickvalue` fits AAV space (value language) — reused only to link a tier's AAV band to a dynasty-PPG range when the WR shelf wants it; the TIER boundaries themselves come from ADP, which is what owners see reaching draft day.

### Method
1. Board assembly: every prospect (a player with `Record.Rookie` = the draft year in play, valued or not) ranked by ADP ascending; standard-deviation of his draft position σ from the sources' spread where the feeds release it, else his |rank−ADP| spread across the peer-feed sources.
2. Tiers: walk the ordered board; cut a new tier where the ADP gap between adjacent players exceeds `GAP_K × σ(class-gap)`, with `GAP_K` = 1.2 and `MIN_TIER_SIZE` = 2 (a lone player on an island stays tier-less and reads "own tier"). Boundaries also require the gap to be asymmetric (large after, small before), a cheap guard against boundary-flicker between refreshes.
3. Position-tagged tiers: tiers cut within position too (RB tier 1, WR tier 1…), because owners run position cliffs next to their risk, and the position cliffs don't line up with the board cliff.
4. "Likely still there at pick N": per player, an availability curve `P(N) = P(draft_position > N)` with draft position distributed ~ Normal(ADP, σ_pos) where σ_pos is fitted per position and per pick-number-distance from his ADP, calibrated from the app's own past drafts (step (iii) of the proof). The tile shows P(N) for the owner's next pick ONLY (a scalar question), and the curve shape stays internal.
5. The "ADP-still-there" number is worded as its own risk lane: "ADP says pick {x}. You're pick {N}: still there, 40%." No invented urgency language beyond that.
6. Tier movement day-to-day is a real thing owners follow ("he jumped two tiers on the depth-chart news"); the refresh trigger ensures the tiers are recomputed AFTER (not during) a feed flush, so the tile never renders a half-flushed ADP table.

### Output shape
- `ProspectBoard{ Season int; Tiers []struct{ N int; Scope string /* board | position */; Players []string; ADPBands { Lo, Hi float64 } }; Availability []struct{ PlayerID string; ADP float64; PAtPick map[int]float64 } }` (PUB).

### Refresh trigger
Daily with marketfeeds (map); forced recompute when D41 order changes (a traded pick with a new original team changes who waits where).

### Replay proof
Replay the app's own past rookie drafts (the strongest possible test, since it is real behavior by real owners, not a feed's opinion):
(i) Availability calibration: for pick N in each past draft, the set of players the model said "≥ 60% likely still there at N" must actually be there in ≥ ~55–65% of true curriculum weeks, with the same diagonal-bucket calibration as D06's odds — bands must hit their claimed probabilities within ±7pp.
(ii) Tier-truth test on real drafts: within-tier players must NOT be separated in the actual draft more often than cross-tier pairs are together — measure as: P(actual draft separation | same tier) < P(actual draft separation | different tier) by a margin the replay prints and pass-flags per tier boundary; a boundary that never separates anyone (or always does) is a wrong boundary.
(iii) σ_pos calibration: the fitted draft-position deviations from history must have empirical sd within ±25% of what σ_pos assumed across 2+ seasons of real drafts (the availability curve leans on σ; a wrong σ silently ruins everything the tile promises).
(iv) No-feed-information test: the model's tiers should mostly agree with the feed's stated tiers when the source publishes any (agreement ≥ 80% of shared boundaries). Where they DISAGREE and the source's tier was right, we log why (feed knows things we don't yet — that's D80's lane).

### Failure / thin-data behaviour
- Players with no ADP (haven't crossed the feeds yet): they render BELOW the last tier with "ADP not out" — never mid-board silently; the tiers stay readable without data services that update hourly.
- Perfectly flat ADP stretch (feed gives everyone a rank and no spread): the availability capability shows "ADP is flat here — spread unknown" and PAtPick shows the flat-line truth (roughly 50%) across the adjacent picks rather than pretending a cliff.
- Expansion draft / league draft with altered rounds: σ_pos calibration replays on that league's OWN drafts where history exists; where it doesn't (expansion), the tile says "the model has never seen a draft this size — the odds below are the stock curve, re-fitted weekly as this draft unfolds."
- Day-of-draft: live in-draft board moves are D41's lane; D42's tiers do NOT re-cut mid-draft (a mid-draft re-cut of tiers could mislead a pick-clock scenario; the board freezes at draft start).

### Open questions
- Should D42 tiers incorporate OUR CommissgreSQL's slot pricing (rookie slot salary D77) as a fourth dimension ("tier 3 slot-value at round 3")? Real: D77 exists and works; the tier boundary is draft-value-only, though; worth a ruling whether the slot salary's effect on draft-day urgency moves the methodology or stays a tile beside it.
- The "likely still there" number lanes into narrative contradiction with the actual pick clock: when the owner is ON the clock, does the tile switch to the pick-independent "who's gone" truth instead (a different question, D41's data)? Proposed: yes; flag as a persuasion trade, not a math ruling.

---

## 12. Summary — one line per model

| Model | Inputs (domains) | Output | Method in one line | Replay proof | Open questions |
|---|---|---|---|---|---|
| D26 Age curve | T4 + D12 history + prior D24 runs | value multiplier by (pos, football age) + peak windows | median of each player's self-normalized production r = PPG/own-peak, bucketed by age and position, winsorized, smooth monotone past 30 | curve-augmented next-season prediction beats flat-age baseline in ≥3 of 4 seasons (Spearman, pooled); D25 order stable ≤2 players/roster/week | standalone tile or D25-only? [commish confirmation] · keeper leagues curve? out of scope |
| D25 Best 21 | D17 + D24 + D26 | best 21 by value (any pos) + score + league rank | sum of the top-21 native dynasty values × age-curve multiplier; no slot constraints, contracts NOT inputs | Spearman ≥ .55 vs next-8-week all-play; with-curve beats plain sum in ≥3 of 4 | salary-adjusted 21? · show 21 names always or count + newcomers? |
| D27 Needs/surplus | D17 + D25 + D26 + D20 (wording) | needs vs league template + surplus groups + "worth about a late pick" | need = median league template − own Best 21 count; surplus = value outside the Best 21; equivalence via Pick Value v2's pick table, rounded to fifths of a round | ≥60% of past trades' labels land within one round-band of actual pick value; ≥50% of critical needs later addressed | template from Best 21 (ruled) vs whole roster? · equivalence picks by slot value? |
| D28 Posture + lens | D05 (both views) + D25 + D27 + D20 + D26 | CONTEND / RETOOL / REBUILD + λ-tilt trade lens from PUB data | fixed bands on season/dynasty power percentiles; lens = convex λ mix of now-value and dynasty-value per side | CONTEND label → playoffs ≥65%; REBUILD → top-quart pick ≥50%; posture flip ≤ once per 2 seasons | season-strong/dynasty-weak OLD roster → RETOOL or REBUILD? [ruling] · reason line sharpness |
| D32 Tune-it | D29 (PRIV) + D28 (both sides) + D27 + D33 + D20 | TRIM/ASK/SWEETEN suggestions, ≤3, with previews | value gap vs a FAIR_BAND under both lenses' mirrored values; suggestions prefer surplus-first, need-first candidates, cap-checked by D20 | endorsed suggestion moves past trades inside the fair band ≥70%; stays silent ≥95% on already-fair trades; always cap-legal | who carries the counter-draft? D30 threading · tighter band when I'm the one underpaying? [assumption, measured by replay] |
| D33 Lineup delta | D17 + D11 + D29 (PRIV) | who starts/benches before vs after + points per week | `powerrankings.Lineup` (existing, greedy-optimal) on roster ± trade, with `ExpectedPoints` blending results and projections | gain ≥+2.0 PPG claims hold in direction on realized trades over 3+ seasons [sign test α ≥.05]; slope 0.7–1.3; named-benchings agree with actual starters ≥70% | IDP clarity in wording?ilihan? · which number wins when this-week and rest-of-season disagree? |
| D11 Native projections | D54/T4 league scoring history + D14 + D15 + D16 + (D13 later) | floor/median/ceiling per week, position rank, rest of season | recency-weighted quantiles (p10/p50/p90) of this scoring's weekly PPG, shrunk to the position distribution by draft-capital value; health and matchup on top | coverage 75–92%; median per-week Spearman ≥ .45 per position; shrinkage beats raw history for <8-game players (RMSE) | native weights on WR tile next to MFL-sourced? · CUSTOM matchup multiplier for the commissioner? |
| D05 Power components | D12 + D17 + D11 + D14/D15 + D04 via race + D16/D25 | power rank, five named components, rank movement narrative | five z-components (FORM, LINEUP, HEALTH, ALL-PLAY, RUN-IN), weights fitted once (W = .20/.25/.10/.25/.20), why-moved = largest Δz×w driver | board predicts final all-play ≥ .70 Spearman per season ×4; ablation: NO component's removal improves it | z-table visible or one sentence? · W as a params setting, or fitted once app-wide? |
| D06 Playoff sim | D02 + D05 (expected scores) + D16 + race Seeding/FirstRound/Bubble | playoff odds, bye/div odds, magic number, this-week shift | 20k sims of remaining games via WinProbability + score noise (σ = existing GameSpread), re-seeded through race's OWN Seeding each draw | band calibration ≤ ±7pp; Brier ≤ .22 week 8; SIM = exactly 0/100% post-season; magic # matches hand-computed races | magic for bye and magic for field both? (tile space) · same PUB odds plus only wording lanes |
| D35 Bid market | T3 revealed bids + D20 + D27 + D34 pool | estimated winning price + trend + win odds + rivals, no sealed input | winning price = recency-weighted per-position ratio of dynasty value; rivals = cap room AND need, from PUB data only | P25–P75 band catches ACTUAL winning bids ≥50%; named rivals hit the actual bids' top-3 in ≥40%; win-odds bands calibrate ±10pp | rival names public or count-first? [real ruling] · blind-bid wording lanes ("about 1-in-5") |
| D42 Prospect tiers | D42 feeds (ADP/AAV) + draft history + D41 + D80 (eventuel) | tier cuts per position + "still there at pick N" per player | ADP-ordered board, σ-gap-cut tiers (GAP_K 1.2, min 2); availability curve from Normal(ADP, σ_pos) calibrated on the app's own past drafts | availability bands hit claimed probabilities ±7pp; tier boundaries separate players MORE within than across (per contract); σ_pos's within-25% bar holds | D77 slot salary as a dimension? (beside tiers vs inside) · switch the tone when on the clock? |

Auto-check note: the draft's open questions are the commissioner's door — flagged INSIDE each section as real placeholders for Christopher's rulings, never as un-noted assumptions.

---

## 13. Claude review (10-30) — fixes before this becomes the method of record
Code citations checked on origin/main: `GameSpread = 50.0` and `FranchiseAgeWeight = 0.5` (powerrankings/blend.go:29–32) are real; `playerrecord` carries Birth (assemble.go:88); `season/power.go` PowerRow holds 3 z's (RosterZ, MFLPerfZ, AgeZ), so D05 extends it to 5.

**Must fix**
1. **D26 uses experience, not age.** Step 1 defines "football age" as seasons since rookie year. The record has a real birth date; fit the curve on true age, with experience as a secondary input.
2. **D26 leaks the future in replay.** Step 3 normalizes each player by his career peak, which includes seasons after the rewind date. In replay, normalize only on data before the rewind.
3. **Build order is wrong.** D33 needs D11 projections and D15 return weeks, and D32 needs D33. Corrected order: D26 → D25 → D27 → D28 · D11 → D33 → D32 · D11 → D05 → D06 · D35 · D42. D15 is L4 content, so D33 ships with "return week unknown" handling until D15 lands.
4. **D28 rule is ambiguous.** "season ≤ .35 OR dynasty ≤ .35 AND (…)" needs brackets; write it as an explicit truth table.
5. **D06 can't run per request.** 20,000 sims × 32 teams is far beyond the 200 ms screen budget. Run it at game final and nightly, store the results, and have screens read the stored row. The two forced "this week" runs are part of the same nightly job.
6. **D11 injury constants** (DOUBTFUL −45% etc.) are invented. Mark them ASSUMPTION and fit them in replay like the rest.
7. **D27 needs** counts positions in Best 21 only, so it misses lineup requirements (a team can be thin at a required starting slot that its Best 21 hides). Needs = max(Best-21 template gap, starting-slot shortfall).
8. Drop D05's "nolint-exemption precedents" line, which has nothing to do with the method.

**Fine as drafted:** D25, D32 (bounded suggestions, no-harm bar), D35 (revealed history only, which stays right after Q4: the commish seeing open bids never feeds the model), D42, the D06 one-engine rule (reuse race.Build + Seeding).

**Questions for Christopher (consolidated from GLM's open questions, real ones only)**
- Q-M1 D35: name the rival owners, or show a count only?
- Q-M2 D06: two magic numbers (for a bye, for the field)?
- Q-M3 D05: power weights fitted once for every league, or a commish setting?
- Q-M4 D33: when this week's and rest-of-season deltas differ, which one is the headline?
- Q-M5 D28: is an old contender in its last push labelled RETOOL or REBUILD?
