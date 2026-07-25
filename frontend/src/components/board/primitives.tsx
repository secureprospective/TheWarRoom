// Session-B board presentational primitives (B-2). Thin helpers over the CSS
// grammar in style.css (.twr-board*) — deliberately NOT a generic table
// component: the wireframe's contract IS the CSS class set, and M1/M2/roster
// share it by using the classes directly. These encode only the bits that are
// awkward inline: the typographic sort affordance and the honest empty/loading
// states currently wired into the boards.
//
// B-5 completed the set: the two remaining §8 states now have their backend signals and
// land here as FreshnessBar (cache-only) and PhaseBar (offseason), and §1 delta-in-weight
// lands as DeltaRank. The two bars are deliberately SEPARATE components — see PhaseBar.
// Source: docs/ui/wireframes/session-b/session-b-wireframe.html §1 + §2·3 + §8.

import { Freshness, ageLabel, freshnessState, isFinalPhase } from './freshness';

export type SortDir = 'asc' | 'desc';

// SortHeader — a sort button for a sub-header cell. Active column shows ▼/▲
// bright; every other sortable column shows a near-invisible ¦ (wireframe §1
// ADOPTED: no icon chrome). Right-alignment is the CALLER's job: numeric headers
// wrap this in the grid-child `<span className="twr-r">` (SortHeader is not a
// grid child itself, so it can't align the cell).
export function SortHeader<K extends string>({
  label,
  sortKey,
  activeKey,
  dir,
  onSort,
}: {
  label: string;
  sortKey: K;
  activeKey: K;
  dir: SortDir;
  onSort: (key: K) => void;
}) {
  const active = sortKey === activeKey;
  const glyph = active ? (dir === 'desc' ? '▼' : '▲') : '¦';
  return (
    <button
      type="button"
      className="twr-sortbtn"
      onClick={() => onSort(sortKey)}
      aria-label={`Sort by ${label} ${active ? (dir === 'desc' ? 'descending' : 'ascending') : ''}`.trim()}
    >
      {label}
      <span className={active ? 'twr-sort' : 'twr-isort'} aria-hidden>
        {glyph}
      </span>
    </button>
  );
}

// DeltaRank — §1 DELTA-IN-WEIGHT (Session-B, locked). Movement is carried by FONT WEIGHT,
// not by colour: +Δ renders at 600, −Δ at 400. The rule is GREYSCALE-HONEST — the meaning
// must survive with all colour removed, so hue may only reinforce what weight already
// says. That is what makes it readable for a colour-blind user and in a screenshot.
//
// ok=false means there is NO previous board to compare against (a first-ever scoring run),
// and it renders as an em dash — absent, not zero. "Held position" and "we don't know yet"
// are different claims, and a 0 would assert the first one on no evidence.
export function DeltaRank({ delta, ok }: { delta: number; ok: boolean }) {
  // Go sends RankDelta as a plain int, so it cannot currently arrive null — but the TS
  // type would not stop it if the DTO changed, and the failure mode is silent: ok=true
  // with a missing delta falls through the comparisons and renders "−NaN". Treating an
  // absent number as "unknown" keeps the component total (GLM review, B-5).
  if (!ok || delta == null || Number.isNaN(delta)) {
    return <span className="twr-delta twr-delta--none">—</span>;
  }
  if (delta === 0) return <span className="twr-delta twr-delta--flat">0</span>;
  const up = delta > 0;
  return (
    <span
      className={`twr-delta ${up ? 'twr-delta--up' : 'twr-delta--down'}`}
      title={`${up ? 'Up' : 'Down'} ${Math.abs(delta)} since the previous scoring run`}
    >
      {up ? '+' : '−'}
      {Math.abs(delta)}
    </span>
  );
}

// EngraveState — the "no data" honest empty state (wireframe §8). Empty is never
// "broken": a fresh install engraves where data will live so a first run feels
// oriented rather than blank.
export function EngraveState({ lines }: { lines: string[] }) {
  return (
    <div className="twr-state">
      <div className="twr-engrave">
        {lines.map((l, i) => (
          <span key={l}>
            {l}
            {i < lines.length - 1 ? <br /> : null}
          </span>
        ))}
      </div>
    </div>
  );
}

// FreshnessBar — the §8 cache-only / freshness treatment (B-5), and the ONLY thing that
// should ever indicate staleness on a board. It renders ABOVE the data as a thin labelled
// edge; the rows below it are untouched and fully legible. See freshness.ts for why
// dimming the data instead would be exactly backwards.
//
// It renders nothing at all when data is live, so a healthy board carries no chrome — a
// permanent "everything is fine" banner is noise that trains the user to stop reading the
// strip, costing exactly the moments it matters.
// `board` names the surface this bar belongs to. role="status" announces on change, and a
// bare "CACHED, last updated 3h ago" tells a screen-reader user nothing about WHICH board
// went stale when several can (GLM review, B-5) — so the accessible name carries it.
//
// On the 'fail' path the rows below are empty and the board's own engrave state renders;
// this bar is not redundant with it — the engrave says "nothing here", this says WHY.
export function FreshnessBar({
  freshness,
  board,
}: {
  freshness: Freshness | undefined | null;
  board?: string;
}) {
  const state = freshnessState(freshness);
  if (state === 'live') return null;

  const stale = state === 'stale';
  const detail = stale
    ? `last updated ${ageLabel(freshness?.fetchedAt ?? '')}`
    : 'no data available';
  const label = stale ? 'CACHED' : 'UNAVAILABLE';
  return (
    <div
      className={`twr-fresh twr-fresh--${state}`}
      role="status"
      aria-label={`${board ? `${board}: ` : ''}${label} — ${detail}`}
    >
      <span className="twr-fresh__label">{label}</span>
      <span className="twr-fresh__note">
        {detail}
        {freshness?.note ? ` — ${freshness.note}` : ''}
      </span>
    </div>
  );
}

// PhaseBar — the §8 OFFSEASON state (B-5). Deliberately a SEPARATE component from
// FreshnessBar and a separate visual treatment: an offseason board is final, not degraded.
// It reads as a neutral statement of fact, not a warning, because nothing is wrong.
export function PhaseBar({ phase }: { phase: string | undefined }) {
  if (!isFinalPhase(phase)) return null;
  return (
    <div className="twr-fresh twr-fresh--final" role="status">
      <span className="twr-fresh__label">FINAL</span>
      <span className="twr-fresh__note">season complete — standings will not change again</span>
    </div>
  );
}

// SkeletonState — a fetch in flight shows data-shaped pulses, never a spinner.
export function SkeletonState() {
  return (
    <div className="twr-state" style={{ alignItems: 'stretch', gap: 8 }}>
      <div className="twr-skel is-hero" />
      <div className="twr-skel" style={{ width: '88%' }} />
      <div className="twr-skel" style={{ width: '70%' }} />
      <div className="twr-skel" style={{ width: '45%' }} />
    </div>
  );
}
