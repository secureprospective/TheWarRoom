// Shared presentation helpers for the M4 transaction surfaces. TransactionWorkspace and
// TradeBuilder both render the same franchise/roster boards, so the money/initials
// formatters and the Empty placeholder live here once (M17: extract on 2nd use).
//
// B-2: the roster/FA tables migrated onto the .twr-board* grammar, so the old <table>
// header helper (Th) is gone — .twr-board__sub carries headers now. Empty stays for the
// non-engrave placeholders (loading, "pick a franchise").

// money formats a millions value as the league's `$X.XM` cap notation.
export const money = (m: number) => `$${m.toFixed(1)}M`;

// initials renders a player's up-to-two-letter avatar monogram.
export const initials = (name: string) =>
  name
    .split(' ')
    .map((s) => s[0])
    .slice(0, 2)
    .join('')
    .toUpperCase();

// Empty is the centered placeholder shown when a table or panel has no rows.
export function Empty({ text }: { text: string }) {
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
