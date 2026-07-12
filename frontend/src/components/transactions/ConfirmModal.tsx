import { main } from '../../../wailsjs/go/models';

// ConfirmModal is the shared staged-confirm step for every priced/unpriced op (design D3).
// It renders one of three states for a pending action:
//   - plain    : a costless op (ROSTER_STATUS) — no preview, just "confirm this move"
//   - quote    : a preview came back OK — show it will commit + players affected (D5)
//   - rejected : the preview returned the AUTHORITATIVE rejection reason (D5) — no confirm
// Confirming re-sends the SAME intent to ExecuteTransaction (never the preview result — D4);
// the parent owns that call so the number is always recomputed server-side (invariant 1).

export type Pending = {
  kind: 'ROSTER_STATUS' | 'WAIVER' | 'SIGN';
  title: string;
  subject: string; // player name for the header
  meta: string; // "WR · Free agent" etc.
  note: string; // human sentence describing the effect
  destructive: boolean;
  previewing: boolean; // true while PreviewTransaction is in flight
  previewOK: boolean | null; // null = not previewed yet (plain confirm), true/false after
  detail: string; // rejection reason (previewOK === false)
  playersAffected: number;
  request: main.TransactionRequest;
};

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
  const labelColor = pending.destructive ? 'text-[#f87171]' : 'text-[#5b9dff]';

  return (
    <div
      className="fixed inset-0 z-20 flex items-center justify-center bg-[rgba(6,10,18,0.66)]"
      onClick={(e) => {
        // Non-dismissable while a commit is in flight: ExecuteTransaction runs an atomic WriteTx to
        // completion with no abort, so a backdrop-click "cancel" would close the modal while the
        // destructive op still commits — a false cancel (GLM L2). Only the terminal states dismiss.
        if (e.target === e.currentTarget && !busy) onCancel();
      }}
    >
      <div className="w-[420px] max-w-[92vw] border border-[#29344a] bg-[#161d2b] shadow-[0_24px_60px_rgba(0,0,0,0.5)]">
        <div className="border-b border-[#202a3d] px-5 py-4">
          <div className={`text-[10.5px] font-semibold uppercase tracking-[0.09em] ${labelColor}`}>
            {pending.title}
          </div>
          <h2 className="mt-1 text-[17px] font-bold">{pending.subject}</h2>
          <div className="mt-0.5 text-[11.5px] text-[#93a1b8]">{pending.meta}</div>
        </div>

        <div className="px-5 py-[18px]">
          {pending.previewing ? (
            <p className="text-[13px] text-[#93a1b8]">Checking the move with the engine…</p>
          ) : rejected ? (
            <div className="border border-[rgba(248,113,113,0.3)] bg-[rgba(248,113,113,0.12)] px-3 py-2.5">
              <div className="text-[10.5px] font-semibold uppercase tracking-[0.06em] text-[#f87171]">
                The engine rejected this move
              </div>
              <p className="mt-1.5 text-[13px] text-[#e2e8f0]">{pending.detail}</p>
            </div>
          ) : (
            <>
              <p className="mb-3.5 text-[12px] text-[#93a1b8]">{pending.note}</p>
              {pending.previewOK === true && (
                <div className="flex items-start gap-1.5 text-[11px] text-[#64748b]">
                  <span className="mt-px text-[#34d399]">✓</span>
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

        <div className="flex justify-end gap-2.5 border-t border-[#202a3d] px-5 py-3.5">
          <button
            type="button"
            onClick={onCancel}
            disabled={busy}
            className="h-9 border border-[#29344a] px-4 text-[13px] font-semibold text-[#93a1b8] hover:border-[#64748b] hover:text-[#e2e8f0] disabled:opacity-40"
          >
            {rejected ? 'Close' : 'Cancel'}
          </button>
          {!rejected && (
            <button
              type="button"
              disabled={!canConfirm}
              onClick={onConfirm}
              className={`h-9 px-4 text-[13px] font-semibold disabled:opacity-40 ${
                pending.destructive
                  ? 'bg-[#f87171] text-[#2b0808] hover:bg-[#fca5a5]'
                  : 'bg-[#34d399] text-[#06251a] hover:bg-[#4ade9f]'
              }`}
            >
              {busy ? 'Committing…' : pending.kind === 'WAIVER' ? 'Confirm cut' : 'Confirm move'}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
