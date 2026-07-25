import { useEffect, useMemo, useRef, useState } from 'react';
import { useHarnessStore } from '../store/harness';
import { main } from '../../wailsjs/go/models';
import {
  SortHeader,
  EngraveState,
  SkeletonState,
  FreshnessBar,
  PhaseBar,
} from './board/primitives';

// DEFAULT_SCOUTING_WEIGHT mirrors Go's powerrankings.DefaultScoutingWeight (0.60).
// Kept in sync by hand — the Go const is the source of truth; if it moves, move this.
const DEFAULT_SCOUTING_WEIGHT = 0.6;

// PowerRankingsBoard is the M2 module view: the 32 franchises ranked by the blended
// PowerScore, in MFL's report column layout. The headline metric standardizes our
// forward-looking scouting engine (the 60) and MFL's luck-adjusted all-play result
// (the 40) to z-scores, weights them w / (1−w), and min-max's the result to [0,1];
// w is a free 0–100% slider, default 0.60. Scouting talent aggregates as the full
// roster sum OR the top-N starters (toggle). The raw MFL columns (pwr, altpwr,
// all-play, PF/PA/PP) come free in the same standings call and are shown as sortable
// context. Read-only: no writes, the live MFL fetch is the only network touch.
//
// B-2 restyle: shares the M1 Session-B board grammar (docs/ui/wireframes/session-b)
// — PowerScore is the dominant hero column, the raw-MFL context columns recede and
// drop in Matrix (pure scan = Rank·Team·Power·Scout z·AllPlay%). The slider/agg
// release-fetch logic is unchanged.

type SortKey =
  | 'rank'
  | 'scoutingZ'
  | 'allPlayWinPct'
  | 'pf'
  | 'pa'
  | 'pp'
  | 'pwr'
  | 'altPwr';

// Tactical carries the full MFL report; Matrix collapses to the blend essentials.
const COLS = '34px 1fr 88px 66px 74px 66px 66px 58px 58px 58px 66px 60px';
const COLS_MTX = '24px 1fr 76px 62px 70px';

export function PowerRankingsBoard() {
  const powerRankings = useHarnessStore((s) => s.powerRankings);
  const powerWeight = useHarnessStore((s) => s.powerWeight);
  const powerAgg = useHarnessStore((s) => s.powerAgg);
  const powerLoading = useHarnessStore((s) => s.powerLoading);
  const error = useHarnessStore((s) => s.error);
  const loadPowerRankings = useHarnessStore((s) => s.loadPowerRankings);

  // Local slider value for instant display; the network fetch fires only on release
  // (onPointerUp/onKeyUp), never on every drag tick.
  const [slider, setSlider] = useState<number>(powerWeight);
  const [sortKey, setSortKey] = useState<SortKey>('rank');
  const [asc, setAsc] = useState<boolean>(true); // rank ascending = best first

  // interacting suppresses the powerWeight→slider echo while the user is dragging,
  // so a resolving fetch can't snap the thumb out from under an in-progress drag.
  const interacting = useRef(false);
  // lastApplied is the weight the current rows were fetched for — a release that
  // didn't change the value skips a redundant live fetch.
  const lastApplied = useRef(powerWeight);

  useEffect(() => {
    void loadPowerRankings(powerWeight, powerAgg);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loadPowerRankings]);

  // Sync the slider to the backend-echoed (clamped) weight — but never mid-drag.
  useEffect(() => {
    if (!interacting.current) {
      setSlider(powerWeight);
      lastApplied.current = powerWeight;
    }
  }, [powerWeight]);

  const rows = useMemo(() => powerRankings?.rows ?? [], [powerRankings]);

  const sorted = useMemo(() => {
    const dir = asc ? 1 : -1;
    return rows.slice().sort((a, b) => (getSortVal(a, sortKey) - getSortVal(b, sortKey)) * dir);
  }, [rows, sortKey, asc]);

  // applyWeight commits the current slider on release. It skips a redundant fetch
  // when the value hasn't changed (a no-op click on the track), and clears the
  // interacting flag so the echo-sync can resume.
  const applyWeight = () => {
    interacting.current = false;
    if (slider === lastApplied.current) return;
    lastApplied.current = slider;
    void loadPowerRankings(slider, powerAgg);
  };

  // setAgg re-fetches at the CURRENTLY-APPLIED weight with the new aggregation.
  const setAgg = (mode: string) => {
    if (mode === powerAgg) return;
    void loadPowerRankings(lastApplied.current, mode);
  };

  const onSort = (key: SortKey) => {
    if (key === sortKey) {
      setAsc(!asc);
    } else {
      setSortKey(key);
      // Rank and PA are "lower is better" → ascending; every other metric is
      // "higher is better" → descending.
      setAsc(key === 'rank' || key === 'pa');
    }
  };
  const dir = asc ? 'asc' : 'desc';

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 12 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        {powerRankings?.ok && (
          <span style={{ fontFamily: 'var(--mono)', fontSize: 11, color: 'var(--text-tertiary)' }}>
            season {powerRankings.season} · {rows.length} franchises
            {powerLoading && ' · refreshing…'}
          </span>
        )}
      </div>

      {error && <div className="twr-banner twr-banner--warn">{error}</div>}

      {/* B-5 degradation contract: an MFL standings outage now serves the last-known-good
          board with a CACHED edge rather than blanking M2. The bar states the age; the
          rows below are untouched and fully readable. PhaseBar is separate and neutral —
          an offseason board is final, not degraded. */}
      <FreshnessBar freshness={powerRankings?.freshness} board="Power Rankings" />
      <PhaseBar phase={powerRankings?.phase} />

      {/* Scouting rides the BasePoints proxy — carry the same honest label M1 shows. */}
      <div className="twr-banner twr-banner--caution">
        {powerRankings?.label ?? 'BasePoints proxy — L2 pending'} · scouting rides the same
        proxy; MFL all-play is live-season actual. Scouting is robust-standardized (median +
        MAD, outlier-resistant) and all-play z-standardized before the weighted blend, then
        scaled 0–1 for display. Scout z of 0 = a typical team.
      </div>

      {/* Weight control (Ledger B6): free 0–100% scouting weight, fires on release. */}
      <div className="twr-panel" style={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 12 }}>
        <span style={{ fontWeight: 500, color: 'var(--text-primary)' }}>Blend weight</span>
        <span style={{ fontFamily: 'var(--mono)', fontSize: 11, color: 'var(--text-tertiary)' }}>
          scouting {(slider * 100).toFixed(0)}% / all-play {((1 - slider) * 100).toFixed(0)}%
        </span>
        <input
          type="range"
          min={0}
          max={1}
          step={0.01}
          value={slider}
          onChange={(e) => {
            interacting.current = true;
            setSlider(Number(e.target.value));
          }}
          onPointerUp={applyWeight}
          onKeyUp={applyWeight}
          className="twr-slider"
          style={{ width: 220 }}
          aria-label="scouting weight"
        />
        <button
          type="button"
          className="twr-chip"
          onClick={() => {
            interacting.current = false;
            setSlider(DEFAULT_SCOUTING_WEIGHT);
            lastApplied.current = DEFAULT_SCOUTING_WEIGHT;
            void loadPowerRankings(DEFAULT_SCOUTING_WEIGHT, powerAgg);
          }}
        >
          Reset 60/40
        </button>

        <span style={{ marginLeft: 12, fontWeight: 500, color: 'var(--text-primary)' }}>Scouting</span>
        <button type="button" className={`twr-chip${powerAgg === 'sum' ? ' is-on' : ''}`} aria-pressed={powerAgg === 'sum'} onClick={() => setAgg('sum')}>
          Roster sum
        </button>
        <button type="button" className={`twr-chip${powerAgg === 'topn' ? ' is-on' : ''}`} aria-pressed={powerAgg === 'topn'} onClick={() => setAgg('topn')}>
          Top-{powerRankings?.starterN || 'N'} starters
        </button>
      </div>

      {rows.length === 0 ? (
        powerLoading ? (
          <SkeletonState />
        ) : (
          <EngraveState
            lines={
              error
                ? ['M2 Power Rankings', '— could not load standings —', 'see the error above']
                : ['M2 Power Rankings', '— no blend yet —', 'score the M1 board first, then reload']
            }
          />
        )
      ) : (
        <div
          className="twr-board"
          style={{ ['--twr-cols' as string]: COLS, ['--twr-cols-mtx' as string]: COLS_MTX }}
        >
          <div className="twr-board__sub">
            <SortHeader label="#" sortKey="rank" activeKey={sortKey} dir={dir} onSort={onSort} />
            <span>Team</span>
            <span className="twr-r">Power</span>
            <span className="twr-r"><SortHeader label="Scout z" sortKey="scoutingZ" activeKey={sortKey} dir={dir} onSort={onSort} /></span>
            <span className="twr-r"><SortHeader label="AllPlay%" sortKey="allPlayWinPct" activeKey={sortKey} dir={dir} onSort={onSort} /></span>
            <span className="twr-r twr-hide-mtx">Record</span>
            <span className="twr-r twr-hide-mtx">AllPlay</span>
            <span className="twr-r twr-hide-mtx"><SortHeader label="PF" sortKey="pf" activeKey={sortKey} dir={dir} onSort={onSort} /></span>
            <span className="twr-r twr-hide-mtx"><SortHeader label="PA" sortKey="pa" activeKey={sortKey} dir={dir} onSort={onSort} /></span>
            <span className="twr-r twr-hide-mtx"><SortHeader label="PP" sortKey="pp" activeKey={sortKey} dir={dir} onSort={onSort} /></span>
            <span className="twr-r twr-hide-mtx"><SortHeader label="MFL Pwr" sortKey="pwr" activeKey={sortKey} dir={dir} onSort={onSort} /></span>
            <span className="twr-r twr-hide-mtx"><SortHeader label="AltPwr" sortKey="altPwr" activeKey={sortKey} dir={dir} onSort={onSort} /></span>
          </div>
          {sorted.map((r) => (
            <div key={r.franchiseID} className="twr-board__row">
              <span className="twr-c-rank">{r.rank}</span>
              <span className="twr-c-name">{r.name}</span>
              <span className="twr-c-adj twr-r">{r.powerScore.toFixed(3)}</span>
              <span className="twr-c-num twr-r">{r.scoutingZ.toFixed(2)}</span>
              <span className="twr-c-num twr-r">{(r.allPlayWinPct * 100).toFixed(1)}%</span>
              <span className="twr-c-num twr-r twr-hide-mtx">
                {r.h2hW}-{r.h2hL}{r.h2hT > 0 ? `-${r.h2hT}` : ''}
              </span>
              <span className="twr-c-num twr-r twr-hide-mtx">
                {r.allPlayW}-{r.allPlayL}{r.allPlayT > 0 ? `-${r.allPlayT}` : ''}
              </span>
              <span className="twr-c-num twr-r twr-hide-mtx">{r.pf.toFixed(1)}</span>
              <span className="twr-c-num twr-r twr-hide-mtx">{r.pa.toFixed(1)}</span>
              <span className="twr-c-num twr-r twr-hide-mtx">{r.pp.toFixed(1)}</span>
              <span className="twr-c-diag twr-r twr-hide-mtx">{r.pwr.toFixed(2)}</span>
              <span className="twr-c-diag twr-r twr-hide-mtx">{r.altPwr.toFixed(1)}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// getSortVal maps a row + key to a comparable number. Rank ascends (1 = best); every
// other column is a magnitude where higher is better, handled by the sort direction.
function getSortVal(r: main.PowerRow, key: SortKey): number {
  switch (key) {
    case 'rank':
      return r.rank;
    case 'scoutingZ':
      return r.scoutingZ;
    case 'allPlayWinPct':
      return r.allPlayWinPct;
    case 'pf':
      return r.pf;
    case 'pa':
      return r.pa;
    case 'pp':
      return r.pp;
    case 'pwr':
      return r.pwr;
    case 'altPwr':
      return r.altPwr;
    default:
      return r.rank;
  }
}
