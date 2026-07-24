// Session-B board presentational primitives (B-2). Thin helpers over the CSS
// grammar in style.css (.twr-board*) — deliberately NOT a generic table
// component: the wireframe's contract IS the CSS class set, and M1/M2/roster
// share it by using the classes directly. These encode only the bits that are
// awkward inline: the typographic sort affordance and the honest empty/loading
// states currently wired into the boards. The remaining §8 states (cache-only,
// offseason) and §1 delta-in-weight land when their backend signals do
// (freshness / season phase), in B-5 — not stubbed speculatively here.
// Source: docs/ui/wireframes/session-b/session-b-wireframe.html §1 + §2·3 + §8.

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
