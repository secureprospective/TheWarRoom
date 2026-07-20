// Shared presentation helpers for the M4 transaction surfaces. TransactionWorkspace and
// TradeBuilder both render the same franchise/roster tables, so the money/initials
// formatters and the Th/Empty table primitives live here once (M17: extract on 2nd use).

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

// Th is the sticky, uppercase table header cell shared by the transaction tables.
export function Th({ children, right }: { children: React.ReactNode; right?: boolean }) {
  return (
    <th
      className={`sticky top-0 z-[1] border-b border-[#29344a] bg-[#0e1420] px-3.5 py-[9px] text-[10.5px] font-semibold uppercase tracking-[0.07em] ${
        right ? 'text-right' : 'text-left'
      }`}
    >
      {children}
    </th>
  );
}

// Empty is the centered placeholder shown when a table or panel has no rows.
export function Empty({ text }: { text: string }) {
  return <div className="m-auto max-w-[260px] p-8 text-center text-[13px] text-[#64748b]">{text}</div>;
}
