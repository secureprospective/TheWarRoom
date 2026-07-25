import { useCallback, useEffect, useRef, useState } from 'react';
import {
  ExecuteTransaction,
  GetCalendarEvents,
  PreviewTransaction,
} from '../../../wailsjs/go/main/App';
import { main } from '../../../wailsjs/go/models';
import { ConfirmModal, type Pending } from '../transactions/ConfirmModal';

// CalendarBoard is the B-3 COMMISSIONER-CALENDAR surface — the agenda/list render of the
// append-only calendar_events log (backend merged in c22be1e). It reads the head view
// (GetCalendarEvents: the latest row per logical event id) and drives the three CRUD-by-append
// ops — SCHEDULE_EVENT, RESCHEDULE_EVENT (the "drag" to a new time), CANCEL_EVENT — through the
// same D5 preview → ConfirmModal → D4 re-send-intent path every other transaction surface uses.
//
// FAITHFUL TO THE BACKEND CONTRACT:
//   - A blob is never edited: reschedule/cancel APPEND a new row carrying the SAME event_id, so
//     they just re-send the head row's kind + payload verbatim with a new scheduledAt / status.
//     Each row is self-contained — the frontend holds the head payload and echoes it back.
//   - status is PLANNED / FIRED / CANCELLED. Only a PLANNED blob is actionable; FIRED/CANCELLED
//     heads render inert (history), matching the store returning them so the UI can style state.
//   - AUTO-FIRE IS OUT OF SCOPE (backend §12 gate, DuePlannedEvents is a later phase) and there is
//     no fire-now IPC yet, so this v1 is render + schedule/reschedule/cancel only.
//   - The "schedule new" form is limited to the SEASON-CLOCK trio (advance phase / roll season /
//     signing window) whose payloads are trivial. The §13 destructive acts (retirement / death /
//     cap relief) are schedulable server-side but are minted from their own LeagueControls surface;
//     they still render, reschedule, and cancel here because those re-send the payload verbatim.

// Month time-grid + drag-to-reschedule is the later render; v1 is the agenda list (Christopher,
// 2026-07-25) — it fits the summon drawer and reuses the whole token contract with no new dep.

type Phase = 'OFFSEASON' | 'REGULAR_SEASON' | 'PLAYOFFS';
type ClockKind = 'ADVANCE_PHASE' | 'ROLLOVER_SEASON' | 'SET_SIGNING_WINDOW';

// KIND_LABEL is the short human name for a blob's eventual op, shown on the agenda row.
const KIND_LABEL: Record<string, string> = {
  ADVANCE_PHASE: 'Advance phase',
  ROLLOVER_SEASON: 'Roll season',
  SET_SIGNING_WINDOW: 'Signing window',
  RETIREMENT: 'Retirement',
  DEATH: 'Death',
  CAP_RELIEF: 'Cap relief',
};

// destructiveKind flags the §13 commissioner acts so the confirm modal + the row accent read red.
const destructiveKind = (k: string) => k === 'RETIREMENT' || k === 'DEATH';

// toLocalInput renders an RFC3339 instant as the `datetime-local` value (local wall-clock, no tz).
function toLocalInput(rfc: string): string {
  const d = new Date(rfc);
  if (Number.isNaN(d.getTime())) return '';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

// fromLocalInput converts a `datetime-local` value back to an RFC3339 UTC instant for the store.
function fromLocalInput(local: string): string {
  const d = new Date(local);
  return Number.isNaN(d.getTime()) ? '' : d.toISOString();
}

// dayKey / dayLabel bucket blobs into calendar-day groups (the agenda's day separators).
function dayKey(rfc: string): string {
  const d = new Date(rfc);
  return Number.isNaN(d.getTime()) ? '—' : d.toDateString();
}
function timeLabel(rfc: string): string {
  const d = new Date(rfc);
  return Number.isNaN(d.getTime())
    ? '—'
    : d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
}

export function CalendarBoard({ onClose }: { onClose: () => void }) {
  const [events, setEvents] = useState<main.CalendarEventDTO[]>([]);
  const [loadErr, setLoadErr] = useState('');
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  // Inline reschedule: the event id whose time picker is open, and its draft datetime-local value.
  const [editing, setEditing] = useState<string | null>(null);
  const [editAt, setEditAt] = useState('');

  const [pending, setPending] = useState<Pending | null>(null);
  const [busy, setBusy] = useState(false);
  // A ref token stamped by each stage() and bumped by cancel/confirm, so a PreviewTransaction that
  // resolves AFTER the operator dismissed or re-staged the modal is discarded instead of resurrecting
  // it with stale data (the LeagueControls stageGen guard — GLM L1). Cancel IS clickable during a
  // preview (busy is still false then), so this race is reachable without the token.
  const stageGen = useRef(0);

  // cancelPending bumps the token so an in-flight preview is discarded, then clears the modal.
  const cancelPending = useCallback(() => {
    stageGen.current++;
    setPending(null);
  }, []);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const r = await GetCalendarEvents();
      if (r.ok) {
        setEvents(r.events ?? []);
        setLoadErr('');
      } else {
        setLoadErr(r.detail || 'Could not load the calendar.');
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

  // stage dry-runs the request through the engine (D5) and folds the authoritative result into the
  // modal; confirm re-sends the SAME intent (D4). Mirrors the LeagueControls stage/confirm pattern.
  function stage(
    base: Omit<Pending, 'previewing' | 'previewOK' | 'detail' | 'playersAffected' | 'capDeltas'>,
  ) {
    const gen = ++stageGen.current;
    const p: Pending = {
      ...base,
      previewing: true,
      previewOK: null,
      detail: '',
      playersAffected: 0,
      capDeltas: [],
    };
    setPending(p);
    void (async () => {
      try {
        const res = await PreviewTransaction(base.request);
        if (gen !== stageGen.current) return; // cancelled/re-staged mid-flight (GLM L1)
        setPending({
          ...p,
          previewing: false,
          previewOK: res.ok,
          detail: res.detail,
          playersAffected: res.playersAffected,
          capDeltas: res.capDeltas ?? [],
        });
      } catch (e) {
        if (gen !== stageGen.current) return;
        setPending({
          ...p,
          previewing: false,
          previewOK: false,
          detail: `Couldn't preview this action — the engine was unreachable (${e instanceof Error ? e.message : String(e)}).`,
        });
      }
    })();
  }

  async function confirm() {
    if (!pending) return;
    stageGen.current++; // this commit owns the modal; discard any older in-flight preview
    setBusy(true);
    try {
      const res = await ExecuteTransaction(pending.request);
      if (!res.ok) {
        setPending({ ...pending, previewing: false, previewOK: false, detail: res.detail });
        return;
      }
      setPending(null);
      setEditing(null);
      await refresh();
    } catch (e) {
      // A thrown IPC call (bridge down, panic) must surface — not vanish as an unhandled rejection
      // while the modal sits open (GLM L2). Fall into the rejected terminal state so it can be read.
      setPending({
        ...pending,
        previewing: false,
        previewOK: false,
        detail: `Couldn't commit this action — the engine was unreachable (${e instanceof Error ? e.message : String(e)}).`,
      });
    } finally {
      setBusy(false);
    }
  }

  // stageReschedule re-sends the head row's kind + payload verbatim with a new time (a new PLANNED
  // row sharing the event_id) — never an edit.
  function stageReschedule(ev: main.CalendarEventDTO) {
    const at = fromLocalInput(editAt);
    if (!at) return;
    stage({
      kind: 'RESCHEDULE_EVENT',
      title: 'Calendar · reschedule',
      subject: `Move "${KIND_LABEL[ev.kind] ?? ev.kind}" to ${new Date(at).toLocaleString()}`,
      meta: `Was ${new Date(ev.scheduledAt).toLocaleString()}`,
      note: 'Appends a new planned time for this event, preserving its full drag history. The eventual op is not run now — the calendar only records intent.',
      destructive: false,
      request: main.TransactionRequest.createFrom({
        kind: 'RESCHEDULE_EVENT',
        eventID: ev.eventID,
        eventKind: ev.kind,
        scheduledAt: at,
        payload: ev.payload,
      }),
    });
  }

  // stageCancel withdraws a blob (appends a CANCELLED row sharing the event_id). Re-sends verbatim.
  function stageCancel(ev: main.CalendarEventDTO) {
    stage({
      kind: 'CANCEL_EVENT',
      title: 'Calendar · cancel',
      subject: `Cancel "${KIND_LABEL[ev.kind] ?? ev.kind}"`,
      meta: `Scheduled ${new Date(ev.scheduledAt).toLocaleString()}`,
      note: 'Withdraws this planned event. Its schedule history is preserved; a cancelled event never fires. You can schedule it again later.',
      destructive: destructiveKind(ev.kind),
      request: main.TransactionRequest.createFrom({
        kind: 'CANCEL_EVENT',
        eventID: ev.eventID,
        eventKind: ev.kind,
        scheduledAt: ev.scheduledAt,
        payload: ev.payload,
      }),
    });
  }

  const grouped = groupByDay(events);

  return (
    <div
      className="twr-cal"
      style={{
        position: 'absolute',
        inset: '0 0 0 auto',
        zIndex: 30,
        width: 420,
        maxWidth: '92vw',
        display: 'flex',
        flexDirection: 'column',
        borderLeft: '1px solid var(--hairline)',
        background: 'var(--surface-overlay)',
        // Inset-bevel elevation only — the design allows no drop shadows (Session-A/C restraint).
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
          Commissioner Calendar
        </span>
        <div style={{ display: 'flex', gap: 6 }}>
          <button
            type="button"
            className={`twr-chip${showForm ? ' is-on' : ''}`}
            onClick={() => setShowForm((v) => !v)}
          >
            ＋ Schedule
          </button>
          <button type="button" className="twr-iconbtn" aria-label="Close calendar" onClick={onClose}>
            ✕
          </button>
        </div>
      </div>

      {showForm && (
        <ScheduleForm
          onCancel={() => setShowForm(false)}
          onStage={(base) => {
            setShowForm(false);
            stage(base);
          }}
        />
      )}

      {/* Agenda body */}
      <div style={{ flex: 1, overflowY: 'auto' }}>
        {loading ? (
          <Note text="Loading the calendar…" />
        ) : loadErr ? (
          <div
            className="twr-banner twr-banner--warn"
            style={{ margin: 12, display: 'block', fontWeight: 500 }}
          >
            {loadErr}
          </div>
        ) : events.length === 0 ? (
          <Note text="No scheduled events. Use ＋ Schedule to plan a season-clock op." />
        ) : (
          grouped.map(([day, evs]) => (
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
                {day}
              </div>
              {evs.map((ev) => (
                <EventRow
                  key={ev.eventID}
                  ev={ev}
                  editing={editing === ev.eventID}
                  editAt={editAt}
                  setEditAt={setEditAt}
                  onOpenEdit={() => {
                    setEditing(ev.eventID);
                    setEditAt(toLocalInput(ev.scheduledAt));
                  }}
                  onCloseEdit={() => setEditing(null)}
                  onApplyEdit={() => stageReschedule(ev)}
                  onCancelEvent={() => stageCancel(ev)}
                />
              ))}
            </div>
          ))
        )}
      </div>

      <ConfirmModal pending={pending} busy={busy} onConfirm={() => void confirm()} onCancel={cancelPending} />
    </div>
  );
}

// EventRow renders one head blob. PLANNED rows expose reschedule (⇄) + cancel (✕); FIRED/CANCELLED
// heads are inert history (dimmed, no actions), matching the append-only "latest row wins" model.
function EventRow({
  ev,
  editing,
  editAt,
  setEditAt,
  onOpenEdit,
  onCloseEdit,
  onApplyEdit,
  onCancelEvent,
}: {
  ev: main.CalendarEventDTO;
  editing: boolean;
  editAt: string;
  setEditAt: (v: string) => void;
  onOpenEdit: () => void;
  onCloseEdit: () => void;
  onApplyEdit: () => void;
  onCancelEvent: () => void;
}) {
  const planned = ev.status === 'PLANNED';
  const dot = statusDot(ev.status, ev.kind);
  return (
    <div style={{ borderBottom: '1px solid var(--hairline)', padding: '8px 12px' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <span aria-hidden style={{ fontSize: 11, color: dot.color }}>
          {dot.glyph}
        </span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div
            style={{
              fontSize: 13,
              fontWeight: 600,
              color: planned ? 'var(--text-primary)' : 'var(--text-disabled)',
              textDecoration: ev.status === 'CANCELLED' ? 'line-through' : 'none',
            }}
          >
            {KIND_LABEL[ev.kind] ?? ev.kind}
          </div>
          {ev.note && (
            <div style={{ fontSize: 11, color: 'var(--text-tertiary)', marginTop: 1 }}>{ev.note}</div>
          )}
        </div>
        <span
          style={{
            fontFamily: 'var(--mono)',
            fontSize: 11.5,
            fontVariantNumeric: 'tabular-nums',
            color: 'var(--text-secondary)',
          }}
        >
          {timeLabel(ev.scheduledAt)}
        </span>
        {planned && !editing && (
          <div style={{ display: 'flex', gap: 2 }}>
            <button type="button" className="twr-iconbtn" aria-label="Reschedule event" onClick={onOpenEdit}>
              ⇄
            </button>
            <button type="button" className="twr-iconbtn" aria-label="Cancel event" onClick={onCancelEvent}>
              ✕
            </button>
          </div>
        )}
        {!planned && (
          <span
            style={{
              fontFamily: 'var(--mono)',
              fontSize: 9.5,
              textTransform: 'uppercase',
              letterSpacing: '0.06em',
              color: 'var(--text-disabled)',
            }}
          >
            {ev.status}
          </span>
        )}
      </div>
      {editing && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 8 }}>
          <input
            className="twr-input"
            type="datetime-local"
            style={{ flex: 1 }}
            value={editAt}
            onChange={(e) => setEditAt(e.target.value)}
          />
          <button type="button" className="twr-btn" disabled={!editAt} onClick={onApplyEdit}>
            Reschedule
          </button>
          <button type="button" className="twr-iconbtn" aria-label="Discard reschedule" onClick={onCloseEdit}>
            ✕
          </button>
        </div>
      )}
    </div>
  );
}

// ScheduleForm mints a new PLANNED season-clock blob. It builds the eventual op's request as the
// verbatim payload the store keeps (executed only when the event later fires), and mints a fresh
// event id. Scope is the trivial-payload trio; the §13 acts schedule from LeagueControls.
function ScheduleForm({
  onCancel,
  onStage,
}: {
  onCancel: () => void;
  onStage: (
    base: Omit<Pending, 'previewing' | 'previewOK' | 'detail' | 'playersAffected' | 'capDeltas'>,
  ) => void;
}) {
  const [kind, setKind] = useState<ClockKind>('ADVANCE_PHASE');
  const [at, setAt] = useState('');
  const [note, setNote] = useState('');
  const [toPhase, setToPhase] = useState<Phase>('REGULAR_SEASON');
  const [windowOpen, setWindowOpen] = useState(true);

  function submit() {
    const scheduledAt = fromLocalInput(at);
    if (!scheduledAt) return;
    // The payload is the eventual op's own request, stored verbatim and run only when it fires.
    const eventual: Record<string, unknown> = { kind, note };
    if (kind === 'ADVANCE_PHASE') eventual.toPhase = toPhase;
    if (kind === 'SET_SIGNING_WINDOW') eventual.windowOpen = windowOpen;
    onStage({
      kind: 'SCHEDULE_EVENT',
      title: 'Calendar · schedule',
      subject: `Schedule "${KIND_LABEL[kind]}" for ${new Date(scheduledAt).toLocaleString()}`,
      meta: 'New planned event',
      note: 'Adds a planned event to the commissioner calendar. It records intent only — the op runs when the event is fired, not now.',
      destructive: false,
      request: main.TransactionRequest.createFrom({
        kind: 'SCHEDULE_EVENT',
        eventID: crypto.randomUUID(),
        eventKind: kind,
        scheduledAt,
        payload: JSON.stringify(eventual),
      }),
    });
  }

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
        padding: 12,
        borderBottom: '1px solid var(--hairline)',
        background: 'var(--surface-sunken)',
      }}
    >
      <select
        className="twr-select"
        value={kind}
        onChange={(e) => setKind(e.target.value as ClockKind)}
      >
        <option value="ADVANCE_PHASE">Advance phase</option>
        <option value="ROLLOVER_SEASON">Roll season (§14)</option>
        <option value="SET_SIGNING_WINDOW">Signing window (§6)</option>
      </select>
      {kind === 'ADVANCE_PHASE' && (
        <select
          className="twr-select"
          value={toPhase}
          onChange={(e) => setToPhase(e.target.value as Phase)}
        >
          <option value="OFFSEASON">OFFSEASON</option>
          <option value="REGULAR_SEASON">REGULAR_SEASON</option>
          <option value="PLAYOFFS">PLAYOFFS</option>
        </select>
      )}
      {kind === 'SET_SIGNING_WINDOW' && (
        <select
          className="twr-select"
          value={windowOpen ? 'open' : 'closed'}
          onChange={(e) => setWindowOpen(e.target.value === 'open')}
        >
          <option value="open">Open the window</option>
          <option value="closed">Close the window</option>
        </select>
      )}
      <input
        className="twr-input"
        type="datetime-local"
        value={at}
        onChange={(e) => setAt(e.target.value)}
      />
      <input
        className="twr-input"
        placeholder="Note (optional)"
        value={note}
        onChange={(e) => setNote(e.target.value)}
      />
      <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
        <button type="button" className="twr-btn" onClick={onCancel}>
          Cancel
        </button>
        <button type="button" className="twr-btn twr-btn--commit" disabled={!at} onClick={submit}>
          Schedule…
        </button>
      </div>
    </div>
  );
}

function Note({ text }: { text: string }) {
  return (
    <div
      style={{
        margin: 'auto',
        maxWidth: 260,
        padding: 32,
        textAlign: 'center',
        fontSize: 13,
        color: 'var(--text-tertiary)',
      }}
    >
      {text}
    </div>
  );
}

// statusDot maps a head status to its agenda glyph + token color: a PLANNED season-clock blob is a
// neutral filled dot, a PLANNED §13 act is amber (it carries weight), FIRED is a green check, and a
// CANCELLED head is a hollow disabled ring.
function statusDot(status: string, kind: string): { glyph: string; color: string } {
  if (status === 'FIRED') return { glyph: '✓', color: 'var(--green-base)' };
  if (status === 'CANCELLED') return { glyph: '○', color: 'var(--text-disabled)' };
  if (destructiveKind(kind) || kind === 'CAP_RELIEF')
    return { glyph: '●', color: 'var(--amber-base)' };
  return { glyph: '●', color: 'var(--text-primary)' };
}

// groupByDay buckets the already-scheduled-ordered head events into day groups for the agenda's day
// separators. The backend returns them ordered by scheduled_at, so insertion order is preserved.
function groupByDay(events: main.CalendarEventDTO[]): [string, main.CalendarEventDTO[]][] {
  const out: [string, main.CalendarEventDTO[]][] = [];
  for (const ev of events) {
    const key = dayKey(ev.scheduledAt);
    const last = out[out.length - 1];
    if (last && last[0] === key) last[1].push(ev);
    else out.push([key, [ev]]);
  }
  return out;
}
