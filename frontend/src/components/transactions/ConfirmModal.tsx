import { main } from '../../../wailsjs/go/models';

// ConfirmModal is the shared staged-confirm step for every priced/unpriced op (design D3).
// It renders one of three states for a pending action:
//   - plain    : a costless op (ROSTER_STATUS) — no preview, just "confirm this move"
//   - quote    : a preview came back OK — show it will commit + players affected (D5)
//   - rejected : the preview returned the AUTHORITATIVE rejection reason (D5) — no confirm
// Confirming re-sends the SAME intent to ExecuteTransaction (never the preview result — D4);
// the parent owns that call so the number is always recomputed server-side (invariant 1).
//
// B-2 restyle: Session-C token contract + control classes, off the hardcoded hex. The
// commit button is the one solid, terminal affordance (.twr-btn--commit); everything else
// is ghost/tokened per the restraint doctrine.

export type Pending = {
  kind:
    | 'ROSTER_STATUS'
    | 'WAIVER'
    | 'SIGN'
    | 'TAG'
    | 'EXTENSION'
    | 'BUYOUT'
    | 'RESTRUCTURE'
    | 'TRADE'
    // D6 commissioner surfaces (calendar + segregated destructive ops).
    | 'ADVANCE_PHASE'
    | 'ROLLOVER_SEASON'
    | 'SET_SIGNING_WINDOW'
    | 'RETIREMENT'
    | 'DEATH'
    | 'CAP_RELIEF'
    // Commissioner calendar CRUD-by-append (schedule / drag-reschedule / cancel a blob).
    | 'SCHEDULE_EVENT'
    | 'RESCHEDULE_EVENT'
    | 'CANCEL_EVENT';
  title: string;
  subject: string; // player name (or, for a TRADE, a summary like "3-leg trade") for the header
  meta: string; // "WR · Free agent" etc.
  note: string; // human sentence describing the effect
  destructive: boolean;
  previewing: boolean; // true while PreviewTransaction is in flight
  previewOK: boolean | null; // null = not previewed yet (plain confirm), true/false after
  detail: string; // rejection reason (previewOK === false)
  playersAffected: number;
  // capDeltas is the preview's pre-commit dollar breakdown (a §8/§12/§13 dead-cap charge, a §13
  // relief credit). Empty for an op not yet wired for a breakdown; the quote then shows only the
  // will-commit line and the dollar lands on the post-commit roster/cap refresh. Signed via `cents`
  // (positive = a charge that raises cap used, negative = a credit that lowers it).
  capDeltas: main.CapDeltaDTO[];
  request: main.TransactionRequest;
};

// confirmLabel is the confirm button's verb per op kind, so the terminal action reads
// naturally (a cut vs. a buyout vs. a season rollover) instead of a generic "Confirm".
function confirmLabel(kind: Pending['kind']): string {
  switch (kind) {
    case 'WAIVER':
      return 'Confirm cut';
    case 'BUYOUT':
      return 'Confirm buyout';
    case 'TRADE':
      return 'Confirm trade';
    case 'ADVANCE_PHASE':
      return 'Advance phase';
    case 'ROLLOVER_SEASON':
      return 'Roll season over';
    case 'SET_SIGNING_WINDOW':
      return 'Apply window';
    case 'RETIREMENT':
      return 'Confirm retirement';
    case 'DEATH':
      return 'Confirm';
    case 'CAP_RELIEF':
      return 'Grant relief';
    case 'SCHEDULE_EVENT':
      return 'Schedule';
    case 'RESCHEDULE_EVENT':
      return 'Reschedule';
    case 'CANCEL_EVENT':
      return 'Cancel event';
    default:
      return 'Confirm move';
  }
}

const labelStyle = (color: string): React.CSSProperties => ({
  fontSize: 10.5,
  fontWeight: 600,
  textTransform: 'uppercase',
  letterSpacing: '0.09em',
  color,
});

export function ConfirmModal({
  pending,
  busy,
  onConfirm,
  onCancel,
}: {
  pending: Pending | null;
  busy: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  if (!pending) return null;

  const rejected = pending.previewOK === false;
  const canConfirm = !pending.previewing && !rejected && !busy;
  const accent = pending.destructive ? 'var(--red-base)' : 'var(--blue-base)';

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 20,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: 'color-mix(in srgb, var(--surface-sunken) 72%, transparent)',
      }}
      onClick={(e) => {
        // Non-dismissable while a commit is in flight: ExecuteTransaction runs an atomic WriteTx to
        // completion with no abort, so a backdrop-click "cancel" would close the modal while the
        // destructive op still commits — a false cancel (GLM L2). Only the terminal states dismiss.
        if (e.target === e.currentTarget && !busy) onCancel();
      }}
    >
      <div
        style={{
          width: 420,
          maxWidth: '92vw',
          border: '1px solid var(--hairline)',
          background: 'var(--surface-overlay)',
        }}
      >
        <div style={{ borderBottom: '1px solid var(--hairline)', padding: '16px 20px' }}>
          <div style={labelStyle(accent)}>{pending.title}</div>
          <h2 style={{ margin: '4px 0 0', fontSize: 17, fontWeight: 700, color: 'var(--text-primary)' }}>
            {pending.subject}
          </h2>
          <div style={{ marginTop: 2, fontSize: 11.5, color: 'var(--text-secondary)' }}>{pending.meta}</div>
        </div>

        <div style={{ padding: '18px 20px' }}>
          {pending.previewing ? (
            <p style={{ margin: 0, fontSize: 13, color: 'var(--text-secondary)' }}>
              Checking the move with the engine…
            </p>
          ) : rejected ? (
            <div
              style={{
                border: '1px solid var(--red-muted)',
                borderLeft: '2px solid var(--red-base)',
                background: 'color-mix(in srgb, var(--red-base) 12%, var(--surface-tile))',
                padding: '10px 12px',
              }}
            >
              <div style={labelStyle('var(--red-base)')}>The engine rejected this move</div>
              <p style={{ margin: '6px 0 0', fontSize: 13, color: 'var(--text-primary)' }}>{pending.detail}</p>
            </div>
          ) : (
            <>
              <p style={{ margin: '0 0 14px', fontSize: 12, color: 'var(--text-secondary)' }}>{pending.note}</p>
              {pending.previewOK === true && (pending.capDeltas ?? []).length > 0 && (
                <div
                  style={{
                    marginBottom: 14,
                    border: '1px solid var(--hairline)',
                    background: 'var(--surface-sunken)',
                    padding: '10px 12px',
                  }}
                >
                  <div style={labelStyle('var(--text-secondary)')}>Cap impact (pre-commit)</div>
                  <ul style={{ margin: '6px 0 0', padding: 0, listStyle: 'none' }}>
                    {(pending.capDeltas ?? []).map((d, i) => (
                      <li
                        key={i}
                        style={{
                          display: 'flex',
                          alignItems: 'baseline',
                          justifyContent: 'space-between',
                          gap: 12,
                          fontSize: 12,
                          marginTop: i === 0 ? 0 : 4,
                        }}
                      >
                        <span style={{ color: 'var(--text-primary)' }}>
                          {d.franchiseName || d.franchiseID}
                          <span style={{ marginLeft: 6, fontSize: 10.5, color: 'var(--text-tertiary)' }}>
                            {d.reason}
                          </span>
                        </span>
                        <span
                          style={{
                            fontWeight: 600,
                            fontFamily: 'var(--mono)',
                            fontVariantNumeric: 'tabular-nums',
                            color: d.cents < 0 ? 'var(--green-base)' : 'var(--red-base)',
                          }}
                        >
                          {d.amount}
                        </span>
                      </li>
                    ))}
                  </ul>
                </div>
              )}
              {pending.previewOK === true && (
                <div style={{ display: 'flex', alignItems: 'flex-start', gap: 6, fontSize: 11, color: 'var(--text-tertiary)' }}>
                  <span style={{ marginTop: 1, color: 'var(--green-base)' }}>✓</span>
                  <span>
                    The engine confirmed this commits ({pending.playersAffected} player
                    {pending.playersAffected === 1 ? '' : 's'} affected). Confirming re-sends the
                    move — the engine recomputes the cap authoritatively, and the new figure lands on
                    the roster.
                  </span>
                </div>
              )}
            </>
          )}
        </div>

        <div
          style={{
            display: 'flex',
            justifyContent: 'flex-end',
            gap: 10,
            borderTop: '1px solid var(--hairline)',
            padding: '14px 20px',
          }}
        >
          <button type="button" className="twr-btn" onClick={onCancel} disabled={busy}>
            {rejected ? 'Close' : 'Cancel'}
          </button>
          {!rejected && (
            <button
              type="button"
              className={`twr-btn twr-btn--commit${pending.destructive ? ' is-danger' : ''}`}
              disabled={!canConfirm}
              onClick={onConfirm}
            >
              {busy ? 'Committing…' : confirmLabel(pending.kind)}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
