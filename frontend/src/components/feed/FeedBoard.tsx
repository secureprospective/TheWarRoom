// FeedBoard — the backward-looking Activity / Transaction Feed, mounted as a summon-overlay
// (the "Feed/Pulse" facet from the Session-E facet map). It reads the append-only ledger
// tables through GetFeed and renders the Session-D event grammar — one time-ordered stream,
// 2px achromatic spine (semantic hue by Kind), mono timestamp, subject/predicate weight split.
//
// This is NOT a new visual language: it is the EXISTING Session-D grammar applied to
// backward-looking historical data instead of forward-looking live data. The live Feed (the
// forward-looking variant) is a separate future surface; this one consumes the read-model that
// projects every row of the existing append-only ledger tables into one chronological river.
//
// Filter scope decision (documented in the session-1 commit message): historical data benefits
// from a kind filter the live Feed does not need (live shows the latest N regardless; historical
// asks "show me all the trades" or "what got cut"). v1 ships a kind chip rail — cheap, useful,
// client-side (no IPC param). Date-range is deferred (the list is already chronological desc;
// scroll + the day separators get the operator most of the way there).

import { useCallback, useEffect, useMemo, useState } from 'react';
import { GetFeed } from '../../../wailsjs/go/main/App';
import { main } from '../../../wailsjs/go/models';
import { EngraveState, SkeletonState } from '../board/primitives';

// KindFilter is the set of coarse filter buckets the chip rail exposes. Each is a set of feed
// Kinds; the ALL bucket is special-cased (no filter). The buckets are chosen so every Kind the
// backend can emit falls into exactly one — if a new Kind appears, defaultKindBucket sends it
// to ALL (visible by default; never silently hidden).
type KindFilter = 'all' | 'trades' | 'releases' | 'cap' | 'contract';

const FILTERS: { id: KindFilter; label: string }[] = [
  { id: 'all', label: 'All' },
  { id: 'trades', label: 'Trades' },
  { id: 'releases', label: 'Releases' },
  { id: 'cap', label: 'Cap' },
  { id: 'contract', label: 'Contracts' },
];

// KIND_BUCKET maps a feed Kind onto its filter bucket. Unclassified kinds land in 'all' (the
// no-op filter) — a future Kind appears by default rather than being dropped silently.
const KIND_BUCKET: Record<string, KindFilter> = {
  TRADE: 'trades',
  SIGN: 'contract',
  EXTENSION: 'contract',
  RESTRUCTURE: 'contract',
  TAG: 'contract',
  WAIVER_VOID: 'contract',
  CONTRACT_CHANGE: 'contract',
  RELEASE: 'releases',
  RETIREMENT: 'releases',
  DEATH: 'releases',
  DEAD_CAP: 'cap',
  CAP_RELIEF: 'cap',
};

// SPINE_COLOR maps a Kind onto the semantic hue its 2px left-axis spine adopts (Session-D
// "Severity Spine"). The mapping is conservative: only Kinds whose meaning is unambiguous get
// a hue; CONTRACT_CHANGE (the uncategorized bucket) gets the default achromatic. Color is never
// the only signal — the predicate text always carries the meaning too.
const SPINE_COLOR: Record<string, string> = {
  TRADE: 'var(--blue-base)',
  SIGN: 'var(--green-base)',
  EXTENSION: 'var(--blue-base)',
  RESTRUCTURE: 'var(--blue-base)',
  TAG: 'var(--blue-base)',
  RELEASE: 'var(--amber-base)',
  WAIVER_VOID: 'var(--amber-base)',
  DEAD_CAP: 'var(--amber-base)',
  RETIREMENT: 'var(--red-base)',
  DEATH: 'var(--red-base)',
  CAP_RELIEF: 'var(--green-base)',
};

// PREDICATE is the human-readable action line for each Kind — the second line of the
// subject/predicate weight split. The subject (player name or franchises) is rendered by the
// row component above this; the predicate is what HAPPENED.
const PREDICATE: Record<string, string> = {
  TRADE: 'Trade executed',
  SIGN: 'Signed (free agency)',
  EXTENSION: 'Contract extended',
  RESTRUCTURE: 'Contract restructured',
  TAG: 'Franchise tag applied',
  WAIVER_VOID: 'Contract voided (waiver)',
  CONTRACT_CHANGE: 'Contract changed',
  RELEASE: 'Released',
  RETIREMENT: 'Retired',
  DEATH: 'Removed (Gaines-Adams)',
  DEAD_CAP: 'Dead cap charged',
  CAP_RELIEF: 'Cap relief credited',
};

// PROVENANCE_LABEL is the inline chip text shown next to the predicate when an event carries
// acquisition provenance (MFL's "Player Acquired Info" vocabulary). Empty provenance renders
// nothing — most events are not acquisitions.
const PROVENANCE_LABEL: Record<string, string> = {
  trade: 'via trade',
  waiver: 'via waiver',
  'free-agent-signing': 'via FA signing',
  draft: 'via draft',
};

function spineColor(kind: string): string {
  return SPINE_COLOR[kind] ?? 'var(--bevel-hi)';
}

function predicate(kind: string): string {
  return PREDICATE[kind] ?? kind;
}

// dayKey/dayLabel bucket events into sticky day separators (the calendar agenda's idiom). A
// missing/invalid timestamp falls into "—" so a malformed row never crashes the render.
function dayKey(rfc: string): string {
  const d = new Date(rfc);
  return Number.isNaN(d.getTime()) ? '—' : d.toDateString();
}

function dayLabel(key: string): string {
  if (key === '—') return 'Unknown time';
  const today = new Date().toDateString();
  const yest = new Date(Date.now() - 86_400_000).toDateString();
  if (key === today) return 'Today';
  if (key === yest) return 'Yesterday';
  return key;
}

function timeLabel(rfc: string): string {
  const d = new Date(rfc);
  return Number.isNaN(d.getTime())
    ? '—'
    : d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
}

// subjectLine builds the row's subject — the heaviest line. It prefers resolved player name;
// falls back to the player id when the name did not resolve (OQ-013: stale created-id); uses
// the franchise names for trade rows; uses the franchise name for cap-relief (no player).
function subjectLine(ev: main.FeedEventDTO): string {
  if (ev.kind === 'TRADE') {
    if (ev.franchiseNames && ev.franchiseNames.length > 0) {
      return ev.franchiseNames.join(' ↔ ');
    }
    return 'Trade';
  }
  if (ev.kind === 'CAP_RELIEF') {
    const f = ev.franchiseNames && ev.franchiseNames[0];
    return f ?? 'Cap relief';
  }
  if (ev.playerName) {
    return ev.playerPosition ? `${ev.playerName} · ${ev.playerPosition}` : ev.playerName;
  }
  if (ev.mflID) {
    // OQ-013 read-side reconciliation seam: id displayed verbatim when no name resolves. The
    // historical ledger row references the id that was live at the time of the event.
    return ev.mflID;
  }
  // Fall back to franchises (e.g. a future kind that carries only franchises).
  if (ev.franchiseNames && ev.franchiseNames.length > 0) {
    return ev.franchiseNames.join(' ↔ ');
  }
  return '—';
}

// subjectNote is the optional second subject-side line — the franchise context for events
// whose subject is a player but which still touch a specific franchise (dead cap, releases
// recorded with a known franchise, etc.).
function subjectNote(ev: main.FeedEventDTO): string {
  if (ev.kind === 'TRADE' || ev.kind === 'CAP_RELIEF') return '';
  if (ev.franchiseNames && ev.franchiseNames.length > 0) {
    return ev.franchiseNames.join(', ');
  }
  return '';
}

export function FeedBoard({ onClose }: { onClose: () => void }) {
  const [filter, setFilter] = useState<KindFilter>('all');
  const [events, setEvents] = useState<main.FeedEventDTO[]>([]);
  const [loadErr, setLoadErr] = useState('');
  const [dirWarning, setDirWarning] = useState('');
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const r = await GetFeed();
      if (r.ok) {
        setEvents(r.events ?? []);
        setDirWarning(r.directoryWarning ?? '');
        setLoadErr('');
      } else {
        setLoadErr(r.detail || 'Could not load the activity feed.');
      }
    } catch (e) {
      setLoadErr(`The engine was unreachable (${e instanceof Error ? e.message : String(e)}).`);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  // Filter is applied CLIENT-SIDE: the IPC returns the most-recent N (the store default) once,
  // and the chip rail narrows within that slice. Trivial at v1 row counts; a server-side filter
  // param is the natural follow-up if the cap ever hides rows the operator wants.
  const filtered = useMemo(() => {
    if (filter === 'all') return events;
    return events.filter((e) => KIND_BUCKET[e.kind] === filter);
  }, [events, filter]);

  // Group by calendar day, preserving the chronological order the IPC returned.
  const grouped = useMemo(() => {
    const m = new Map<string, main.FeedEventDTO[]>();
    for (const e of filtered) {
      const k = dayKey(e.timestamp);
      const arr = m.get(k) ?? [];
      arr.push(e);
      m.set(k, arr);
    }
    return [...m.entries()];
  }, [filtered]);

  return (
    <div
      className="twr-cal"
      style={{
        position: 'absolute',
        inset: '0 0 0 auto',
        zIndex: 30,
        width: 440,
        maxWidth: '92vw',
        display: 'flex',
        flexDirection: 'column',
        borderLeft: '1px solid var(--hairline)',
        background: 'var(--surface-overlay)',
        boxShadow: 'inset 1px 1px 0 var(--bevel-hi), inset -1px -1px 0 var(--bevel-lo)',
        transition: 'transform calc(150ms * var(--motion-mult)) ease-out',
      }}
    >
      {/* Header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          gap: 8,
          borderBottom: '1px solid var(--hairline)',
          padding: '10px 12px',
        }}
      >
        <span
          style={{
            fontFamily: 'var(--mono)',
            fontSize: 11,
            fontWeight: 600,
            textTransform: 'uppercase',
            letterSpacing: '0.06em',
            color: 'var(--text-secondary)',
          }}
        >
          Activity Feed
        </span>
        <button type="button" className="twr-iconbtn" aria-label="Close feed" onClick={onClose} style={{ fontSize: 14 }}>
          ✕
        </button>
      </div>

      {/* Filter chip rail — historical-data affordance the live Feed will not need. */}
      <div
        style={{
          display: 'flex',
          gap: 6,
          padding: '8px 12px',
          borderBottom: '1px solid var(--hairline)',
          flexWrap: 'wrap',
        }}
      >
        {FILTERS.map((f) => (
          <button
            key={f.id}
            type="button"
            className={`twr-chip${filter === f.id ? ' is-on' : ''}`}
            onClick={() => setFilter(f.id)}
            aria-pressed={filter === f.id}
          >
            {f.label}
          </button>
        ))}
      </div>

      {/* Body */}
      <div style={{ flex: 1, overflowY: 'auto' }}>
        {loading ? (
          <div style={{ padding: 12 }}>
            <SkeletonState />
          </div>
        ) : loadErr ? (
          <div
            className="twr-banner twr-banner--warn"
            style={{ margin: 12, display: 'block', fontWeight: 500 }}
          >
            {loadErr}
          </div>
        ) : filtered.length === 0 ? (
          <div style={{ padding: 12 }}>
            {events.length === 0 ? (
              <EngraveState
                lines={[
                  'No activity recorded yet.',
                  'Trades, signings, cuts, and cap moves will appear here in chronological order.',
                ]}
              />
            ) : (
              <EngraveState
                lines={['No events in this filter.', 'Switch the chip rail above to All to see everything.']}
              />
            )}
          </div>
        ) : (
          <>
            {dirWarning && (
              <div
                style={{
                  padding: '6px 12px',
                  background: 'var(--surface-sunken)',
                  borderBottom: '1px solid var(--hairline)',
                  fontFamily: 'var(--mono)',
                  fontSize: 10.5,
                  color: 'var(--text-tertiary)',
                }}
              >
                {dirWarning}
              </div>
            )}
            {grouped.map(([day, evs]) => (
              <div key={day}>
                <div
                  style={{
                    position: 'sticky',
                    top: 0,
                    zIndex: 1,
                    padding: '6px 12px',
                    background: 'var(--surface-sunken)',
                    borderBottom: '1px solid var(--hairline)',
                    fontFamily: 'var(--mono)',
                    fontSize: 10.5,
                    fontWeight: 600,
                    textTransform: 'uppercase',
                    letterSpacing: '0.07em',
                    color: 'var(--text-tertiary)',
                  }}
                >
                  {dayLabel(day)}
                </div>
                {evs.map((ev) => (
                  <FeedRow key={ev.stableKey} ev={ev} />
                ))}
              </div>
            ))}
          </>
        )}
      </div>
    </div>
  );
}

// FeedRow renders one event as the Session-D substrate: 2px left-axis spine (semantic hue by
// Kind), mono timestamp on the right, subject/predicate weight split in the middle. The spine
// is the only place color appears — the body text is always primary/secondary text tones, so
// the row stays legible at any density and survives a screenshot in greyscale.
function FeedRow({ ev }: { ev: main.FeedEventDTO }) {
  const prov = ev.provenance ? PROVENANCE_LABEL[ev.provenance] : '';
  const note = subjectNote(ev);
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'stretch',
        gap: 0,
        borderBottom: '1px solid var(--hairline)',
      }}
    >
      {/* 2px severity spine (Session-D grammar) */}
      <div aria-hidden style={{ width: 2, flexShrink: 0, background: spineColor(ev.kind) }} />
      <div style={{ flex: 1, minWidth: 0, padding: '8px 12px' }}>
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 8 }}>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div
              style={{
                fontSize: 13,
                fontWeight: 600,
                color: 'var(--text-primary)',
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              {subjectLine(ev)}
              {ev.playerUnknown && ev.playerName === '' && ev.mflID && (
                <span
                  style={{
                    marginLeft: 6,
                    fontFamily: 'var(--mono)',
                    fontSize: 10,
                    color: 'var(--text-tertiary)',
                    fontWeight: 400,
                  }}
                  title="This id did not resolve in the current players DB — it may be a stale commissioner-created id (OQ-013)."
                >
                  id?
                </span>
              )}
            </div>
            <div
              style={{
                marginTop: 2,
                fontSize: 11.5,
                color: 'var(--text-secondary)',
                display: 'flex',
                gap: 6,
                alignItems: 'baseline',
                flexWrap: 'wrap',
              }}
            >
              <span>{predicate(ev.kind)}</span>
              {prov && (
                <span style={{ color: 'var(--text-tertiary)' }}>{prov}</span>
              )}
              {note && <span style={{ color: 'var(--text-tertiary)' }}>· {note}</span>}
            </div>
            {(ev.tradeRationale || ev.tradePicksNote) && (
              <div
                style={{
                  marginTop: 4,
                  padding: 6,
                  background: 'var(--surface-sunken)',
                  borderRadius: 2,
                  borderLeft: '2px solid var(--bevel-hi)',
                  fontSize: 11.5,
                  color: 'var(--text-secondary)',
                }}
              >
                {ev.tradeRationale && <div>{ev.tradeRationale}</div>}
                {ev.tradePicksNote && (
                  <div style={{ marginTop: ev.tradeRationale ? 2 : 0, color: 'var(--text-tertiary)' }}>
                    picks: {ev.tradePicksNote}
                  </div>
                )}
              </div>
            )}
            {ev.reason && !ev.tradeRationale && (
              <div style={{ marginTop: 2, fontSize: 10.5, color: 'var(--text-tertiary)', fontFamily: 'var(--mono)' }}>
                {ev.reason}
              </div>
            )}
          </div>
          <span
            style={{
              fontFamily: 'var(--mono)',
              fontSize: 11,
              fontVariantNumeric: 'tabular-nums',
              color: 'var(--text-secondary)',
              flexShrink: 0,
            }}
          >
            {timeLabel(ev.timestamp)}
          </span>
        </div>
      </div>
    </div>
  );
}
