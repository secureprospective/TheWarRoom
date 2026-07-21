import type { CSSProperties } from 'react';

interface CommsStripProps {
  onSummon: (target: 'comms' | 'calendar') => void;
}

// Right-edge 48px quick-dash strip (Command Ledger A10/A11). B-1 ships the
// summon/collapse affordances only; the terminal-log comms thread and the
// fully-functional calendar land in B-3 (Session D grammar).
export function CommsStrip({ onSummon }: CommsStripProps) {
  const btn: CSSProperties = {
    width: '32px',
    height: '32px',
    background: 'var(--surface-tile)',
    border: '1px solid var(--hairline)',
    boxShadow: 'inset 1px 1px 0 var(--bevel-hi), inset -1px -1px 0 var(--bevel-lo)',
    color: 'var(--text-secondary)',
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    fontFamily: 'var(--mono)',
    fontSize: '16px',
  };

  return (
    <aside
      style={{
        width: 'var(--comms-w)',
        height: '100%',
        background: 'var(--surface-sunken)',
        borderLeft: '1px solid var(--hairline)',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        padding: '16px 0',
        gap: '16px',
        flexShrink: 0,
      }}
    >
      <button
        type="button"
        aria-label="Summon comms"
        onClick={() => onSummon('comms')}
        className="focus:outline-none focus-visible:outline focus-visible:outline-2 focus-visible:outline-[color:var(--edge-focus)]"
        style={btn}
      >
        ▟
      </button>
      <button
        type="button"
        aria-label="Summon calendar"
        onClick={() => onSummon('calendar')}
        className="focus:outline-none focus-visible:outline focus-visible:outline-2 focus-visible:outline-[color:var(--edge-focus)]"
        style={btn}
      >
        ▤
      </button>
    </aside>
  );
}
