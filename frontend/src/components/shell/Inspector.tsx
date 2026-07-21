import type { ReactNode } from 'react';

interface InspectorProps {
  open: boolean;
  onClose: () => void;
  children?: ReactNode;
}

// Contextual inspector (Session A): a 320px TRANSFORM overlay (translateX), not
// a width-trade — opening slides it over the workspace's right edge without
// reflow, closing frees the space. It stays mounted when closed (translated
// off), so re-opening is instant. Full per-entity anatomy is B-4; B-1 ships the
// shell + empty state.
export function Inspector({ open, onClose, children }: InspectorProps) {
  return (
    <aside
      aria-hidden={!open}
      style={{
        position: 'absolute',
        top: 0,
        right: 'var(--comms-w)', // sit just left of the comms strip
        bottom: 0,
        width: 'var(--insp-w)',
        background: 'var(--surface-overlay)',
        borderLeft: '1px solid var(--hairline)',
        boxShadow: 'inset 1px 1px 0 var(--bevel-hi), inset -1px -1px 0 var(--bevel-lo)',
        transform: open ? 'translateX(0)' : 'translateX(calc(100% + var(--comms-w)))',
        transition: 'transform calc(150ms * var(--motion-mult)) ease-out',
        display: 'flex',
        flexDirection: 'column',
        zIndex: 20,
      }}
    >
      <div
        style={{
          height: '32px',
          borderBottom: '1px solid var(--hairline)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '0 12px 0 16px',
          flexShrink: 0,
        }}
      >
        <span
          style={{
            fontFamily: 'var(--mono)',
            color: 'var(--text-secondary)',
            fontSize: '11px',
            letterSpacing: '0.1em',
          }}
        >
          INSPECTOR
        </span>
        <button
          type="button"
          aria-label="Close inspector"
          onClick={onClose}
          tabIndex={open ? 0 : -1}
          className="focus:outline-none focus-visible:outline focus-visible:outline-2 focus-visible:outline-[color:var(--edge-focus)]"
          style={{
            background: 'transparent',
            border: 'none',
            color: 'var(--text-secondary)',
            cursor: 'pointer',
            padding: '4px 8px',
            fontFamily: 'var(--mono)',
            fontSize: '14px',
          }}
        >
          ✕
        </button>
      </div>

      <div style={{ flex: 1, overflow: 'auto', padding: '16px' }}>
        {children ?? (
          <div
            style={{
              height: '100%',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              color: 'var(--text-tertiary)',
              fontFamily: 'var(--mono)',
              fontSize: '12px',
              textAlign: 'center',
            }}
          >
            Select an entity
          </div>
        )}
      </div>
    </aside>
  );
}
