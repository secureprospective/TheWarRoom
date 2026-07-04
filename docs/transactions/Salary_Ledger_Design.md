# Salary Ledger — Design (Foundation Pass)

**Status:** DRAFT — pending Christopher review → expert-panel decision gate → build
**Written:** 2026-07-04 (session `session/salary-ledger`)
**Supersedes in priority:** B7c §10 Extension (parked until the ledger foundation is built)
**Decision class:** schema / architecture invariant → **expert-panel gate REQUIRED before lock** (`feedback_expert_panel_decision_gate`)

---

## 0. Governing principles (non-negotiable — bind every op and every re-fit)

1. **The ledger is king.** The `contract_years` cells are the single source of truth for money. When any other value (a stored `annual_salary`, a UI figure, a cached total, a commissioner's mental note) disagrees with the ledger, **the ledger wins** and the other value is wrong. Every derived number — cap usage, dead cap, "years remaining," "highest-paid remaining year" — is computed *from* the cells, never stored as a competing truth.

2. **When in doubt, take all math out of the buttons.** No calculation and no rule judgment lives in the React layer. The button gathers owner intent (an amount, which years, a confirm) and hands it to the engine; the engine computes the numbers, enforces the rules, writes the cells, and logs the reason. The interface is the automation, not the math. If a button is doing arithmetic, that is a bug to move into the engine.

These two are the tie-breakers for every ambiguity below and every Pass-2 correction of shipped code.

---

## 1. Root problem

The contracts table stores **one annual salary per player** (`contracts.annual_salary_cents`, one row per `(league, season, mfl_id)`). Every multi-year contract rule — "highest-paid remaining year," "6 total contract years," per-year extension pricing, per-year restructure moves — needs to know the salary **for each year of the deal**. That data does not exist, so the rules were being inferred, and the confusion around §9/§10/§11 traces directly to it.

Worse, the commissioners have been doing the math by hand — restructures, extensions, franchise-tag figures entered manually. **The goal is to take the math off the table entirely: the ledger is the source of truth, the engine computes and rule-checks, the interface only triggers and displays.**

> **Mindset (locked with Christopher):** flat math, all of it. The interface is the automation, never the math. The button reacts — it does not calculate and it does not judge.

---

## 2. The model — a per-year salary ledger

A player's contract is a **column of cells**, one per contract year — a spreadsheet with player on one axis and league year on the other. Each cell is the salary owed that year.

Proposed table (mirrors the existing store idioms — TEXT ids, int64 cents, per-league):

```sql
CREATE TABLE IF NOT EXISTS contract_years (
    league_id    TEXT    NOT NULL,
    mfl_id       TEXT    NOT NULL,
    league_year  INTEGER NOT NULL,          -- absolute year, e.g. 2026
    salary_cents INTEGER NOT NULL CHECK (salary_cents >= 0),
    year_status  TEXT    NOT NULL,          -- PAID | UFA (post-expiration marker)
    source       TEXT    NOT NULL,          -- SEED | BID | EXTENSION | RESTRUCTURE | TAG | OVERRIDE
    last_updated TEXT    NOT NULL,
    PRIMARY KEY (league_id, mfl_id, league_year)
);
```

- **One cell = one (player, year).** A player with a 3-year deal has 3 `PAID` rows.
- `salary_cents` is `int64` cents (`$2M = 200000000`), consistent with `domain.Money`.
- `source` tags **which mechanic last wrote the cell** — the quick provenance of the *current* value (full dated history lives in the change log below).
- `year_status` = `PAID` for a contract year; the `UFA` boundary is derivable (first year past the last `PAID` cell) but stored explicitly for clean querying and to hold the override case.

### 2a. Per-cell change history (dated, reasoned — human-verifiable)

Every time a cell deviates from its **original contracted (seeded) value** — extension, restructure, tag, or commissioner override — we append a row recording what changed, why, and when. A commissioner (or we, debugging) can read a player's full transaction history straight from this log and reconcile it against the current cells.

```sql
CREATE TABLE IF NOT EXISTS contract_year_changes (
    id           TEXT    PRIMARY KEY,
    league_id    TEXT    NOT NULL,
    mfl_id       TEXT    NOT NULL,
    league_year  INTEGER NOT NULL,          -- which cell (year) changed
    old_cents    INTEGER,                   -- prior value; NULL if the cell was newly created
    new_cents    INTEGER NOT NULL CHECK (new_cents >= 0),
    reason       TEXT    NOT NULL,          -- "§10 extension" | "§11 restructure" | "§9 tag" | "commissioner: <free text>"
    source       TEXT    NOT NULL,          -- EXTENSION | RESTRUCTURE | TAG | OVERRIDE
    changed_at   TEXT    NOT NULL           -- ISO-8601 local+offset, per the timestamp convention
);
-- append-only: BEFORE UPDATE / BEFORE DELETE triggers RAISE(ABORT), the dead_cap_ledger idiom
```

- **Append-only + immutable** (triggers), like `dead_cap_ledger` — history can never be silently rewritten.
- The **seed does not log** (it *is* the original baseline); the change log records only deviations from it, which is precisely "the number is different than the original per-year contract."
- Every engine op and every override writes a change row **in the same transaction** as the cell write — a cell can never move without a dated reason attached.

**Relationship to the current `contracts` table:** the ledger becomes the **money source of truth**; the single `annual_salary_cents` column is retired from being truth (kept transitional for rollback, dropped a release on, exactly like the REAL→cents migration did). `contract_years`/`expiration_year` become **derived** views over the ledger, not stored authorities.

---

## 3. Seed construction (the flat-fill rule)

MFL gives us a single salary + an expiration year (e.g. **`$2M, UFA 2028`**). We **construct** the per-year cells on load:

- **Fencepost (locked with Christopher 2026-07-04):** the stated year is the **last PAID year**. `$2M, UFA 2028` →

  | year | 2026 | 2027 | 2028 | 2029 |
  |------|------|------|------|------|
  | cell | $2M `PAID` | $2M `PAID` | $2M `PAID` | `UFA` |

  UFA attaches to the **offseason after the 2028 season = 2029.**
- Flat-fill: every year from the current league season through `expiration_year` gets a `PAID` cell at the current MFL salary, `source = SEED`.
- Seed runs **once** on a fresh load (idempotent; never overwrites cells that a mechanic or override has since changed — same discipline as B3c "seed once, never reseed").

**Assumption made explicit:** MFL carries no per-year schedule, so seeded contracts are flat by construction. Real per-year variation only appears once extensions/restructures/overrides start writing cells. This is acceptable because the pre-existing manual variation was itself never captured anywhere machine-readable — the ledger becomes the first real record.

---

## 4. Cap math — a column sum

Cap usage for a franchise in a given league year =

```
Σ salary_cents  over that franchise's rostered players' cells WHERE league_year = <season>
  + Σ dead_cap_cents from dead_cap_ledger for that (franchise, league_year)
```

This is flat and exact (int64), and it finally matches how a cap actually works — **annual**. It replaces today's "sum the single `annual_salary` per player" derivation. The dead-cap half is unchanged (B7b already keys dead cap to an absolute `league_year`).

---

## 5. Op write-paths (each op just sets cells)

Conceptual — detailed re-fit is Pass 2 (§7). Each mechanic becomes a set of cell writes computed by the engine:

| Op | What it writes to the ledger |
|----|------------------------------|
| **Bid / signing** | Writes N `PAID` cells at the bid salary for the contracted years. |
| **Extension §10** | Appends ≤3 new `PAID` cells at `max(1.5 × highest remaining PAID cell, position floor)`; original cells untouched; total PAID cells ≤ 6. |
| **Restructure §11** | **Owner-directed money movement between the player's existing contract-year cells.** The owner enters an amount and *which years* to move it from → to; the source cell(s) go down, the destination cell(s) go up by the same total (money is conserved — restructure changes *when* salary hits the cap, never the total). Bounded by the §11 tier max on the reduction and by the source cell's balance (no cell can go negative). See §5a. |
| **Tag §9** | Writes a single `PAID` cell at the tag price for the tag year (tag = a one-year deal). |
| **Waive §8 dead cap** | Reads remaining PAID cells to compute the 35%/50% charge, then clears the player's future cells. |

The **rule checks live at the engine boundary** (eligibility, floors, term caps, per-season op limits via `transaction_counts`), exactly as tag/restructure already do — the ledger just gives them real per-year numbers to read.

### 5a. Restructure as fluid money movement (owner choice)

Restructure stays an **owner discretionary** action — the owner decides how much and which years. Because money now lives in per-year cells, this is just **moving dollars from one year to another**:

- **Owner input:** an amount + a source year + a destination year (or years). The intended UI is a fluid money-mover — a field/stepper that pulls a dollar value out of one year and drops it into another; the owner watches the cells rebalance live.
- **Conservation:** the sum of the deltas across the affected cells is **zero** — total contract money is unchanged; only its per-year distribution (and therefore each year's cap hit) moves.
- **Engine validation (not the button):** the reduction on the source year ≤ the §11 tier max for that year's salary; no cell goes negative; destinations are existing contract years of this same player; one restructure per team per season (`transaction_counts`, `op_kind = "RESTRUCTURE"`), plus the extension-unlock exception (§11 L248).
- **Audit:** every cell touched appends a `contract_year_changes` row (§2a, `source = RESTRUCTURE`) in the same tx.
- **Dead-cap link:** the moved money still lives in future cells, so a later §8 waive charges 35%/50% over the remaining cells as they now stand — the 50% penalty still keys off the restructure marker.

**This resolves the face-vs-adjusted question:** no separate "adjusted" amount is needed inside a cell — lowering a year's cap hit *is* moving dollars out of that year's cell. One `salary_cents` per cell.

---

## 6. Commissioner override (first-class — the human error-correction path)

A commissioner can write **any cell directly**, bypassing the rule engine. This is the escape hatch for the bugs we know we'll find.

- **Authoritative direct write** to `(mfl_id, league_year) → salary_cents / year_status`, `source = OVERRIDE`.
- **Gated:** commissioner-only, on the admin path (AD-05 — never through the ordinary B7 transaction request flow), consistent with B4 params' admin `validateOverride`.
- **Sanity-checked, not rule-checked:** range/non-negative/finite guards only — the whole point is to override the rules.
- **Logged with a reason (§2a):** the commissioner **must** supply a free-text reason; the override appends a `contract_year_changes` row (`source = OVERRIDE`, `reason = "commissioner: <text>"`, `changed_at` dated) in the same transaction as the cell write. An override is therefore always distinguishable from an engine write and fully traceable. Mirrors the rulebook's layered-override-record pattern.

---

## 7. Re-fit of shipped work (Pass 2 — after the foundation is panel-approved + built)

These are merged and functionally verified on the single-`Salary` model; they change shape, they are not discarded:

| Shipped | Re-fit |
|---------|--------|
| **§8 dead cap (B7b)** | "remaining years × salary" reads real per-year cells instead of `annual_salary × contract_years`. |
| **§11 restructure** | Move applies to specific year cell(s); tier max still off the base cell. |
| **§9 tag** | Tag price already league-wide top-5; now writes a single-year cell + resets flags. |
| **Cents migration** | Ledger cells are already int64 cents; retire the single-salary column as truth. |
| **M1 rankings / B6 output** | Read cap/salary from the ledger column-sum instead of `EffectiveSalary(ps)`. |

**§10 Extension** then falls out almost for free — it is "append cells at 150% of the max remaining cell, floored," which is trivial once the ledger exists.

---

## 8. Open questions for the expert panel

1. **Ledger vs. contracts table** — new `contract_years` table alongside `contracts` (contracts keeps identity/status, ledger owns money), or fold everything into the ledger? Rec: separate table, contracts keeps roster/status linkage.
2. **Restructure money mechanics** — RESOLVED with Christopher (2026-07-04): restructure is **owner-directed redistribution of money between the player's contract-year cells** (conserved total, bounded by the §11 tier max on the reduction). See §5a. Panel input welcome on the guards (tier-max attribution when moving from multiple years, rounding), not the model.
3. **`adjusted_salary` vs `salary`** — RESOLVED with Christopher (2026-07-04): **one `salary_cents` per cell.** The per-year ledger makes a separate adjusted figure unnecessary — restructure moves dollars *between* cells rather than splitting face vs adjusted *within* a cell (§5a).
4. **Seed idempotency + year rollover** — when the league advances a season, do past-year cells stay as historical record (append-only) or get pruned? Rec: keep them (cheap, and enables real "total years" + audit).
5. **Override audit shape** — RESOLVED with Christopher (2026-07-04): both — a `source` column on the cell for the *current* value's provenance, plus an append-only, immutable `contract_year_changes` log (§2a) recording every deviation from the seeded original with old→new, reason, and date. Every engine op and override writes a change row in the same tx as the cell write. Panel input welcome on the shape, not the requirement.

---

## 9. Scope of this pass

**Foundation-first** (Christopher's lean): build and panel-approve **the ledger table + seed construction + cap-math column-sum + commissioner override** as a verifiable unit *before* disturbing the four merged ops. The op re-fit (Pass 2) and §10 Extension (Pass 3) follow once the foundation is green and functionally gated on the Beelink.

**Gate before any code:** this doc → Christopher review → expert-panel decision gate → then build.
