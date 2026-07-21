import { MODULES, type ModuleId } from './types';

interface NavRailProps {
  active: ModuleId;
  onSelect: (m: ModuleId) => void;
}

// Left instrument rail (Session A): a PORTFOLIO sector (app crest + active
// league — single league in v1, Phase-3 cross-league seam marked) over a MODULE
// sector. Achromatic per the restraint doctrine; the active item takes the
// NEUTRAL selection axis, never a semantic hue.
export function NavRail({ active, onSelect }: NavRailProps) {
  return (
    <nav
      style={{
        width: 'var(--nav-w)',
        background: 'var(--surface-sunken)',
        borderRight: '1px solid var(--hairline)',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
      }}
    >
      <div
        style={{
          height: '160px',
          borderBottom: '1px solid var(--hairline)',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'center',
          padding: '0 16px',
          gap: '12px',
          flexShrink: 0,
        }}
      >
        <div
          style={{
            fontFamily: 'var(--mono)',
            color: 'var(--text-secondary)',
            letterSpacing: '0.1em',
            fontSize: '14px',
            fontWeight: 700,
          }}
        >
          THE WAR ROOM
        </div>
        <div
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            alignSelf: 'flex-start',
            padding: '2px 8px',
            border: '1px solid var(--hairline)',
            background: 'var(--surface-tile)',
            color: 'var(--text-primary)',
            fontFamily: 'var(--mono)',
            fontSize: '10px',
            letterSpacing: '0.05em',
          }}
        >
          LEGACY
        </div>
        {/* Phase-3 cross-league injection seam */}
      </div>

      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        {(MODULES ?? []).map((m) => {
          const isActive = m.id === active;
          return (
            <button
              key={m.id}
              type="button"
              onClick={() => onSelect(m.id)}
              className="focus:outline-none focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-[-2px] focus-visible:outline-[color:var(--edge-focus)]"
              style={{
                position: 'relative',
                width: '100%',
                textAlign: 'left',
                padding: '12px 16px',
                background: 'transparent',
                border: 'none',
                cursor: 'pointer',
                color: isActive ? 'var(--text-primary)' : 'var(--text-secondary)',
                display: 'flex',
                flexDirection: 'column',
                gap: '2px',
                transition: 'background-color 0.12s ease-out',
              }}
              onMouseEnter={(e) => {
                if (!isActive) e.currentTarget.style.backgroundColor = 'var(--surface-tile)';
              }}
              onMouseLeave={(e) => {
                if (!isActive) e.currentTarget.style.backgroundColor = 'transparent';
              }}
            >
              {isActive ? (
                <span
                  aria-hidden
                  style={{
                    position: 'absolute',
                    left: 0,
                    top: 0,
                    bottom: 0,
                    width: '2px',
                    background: 'var(--edge-selection)',
                  }}
                />
              ) : null}
              <span
                style={{
                  fontFamily: 'var(--mono)',
                  fontSize: '11px',
                  letterSpacing: '0.06em',
                  textTransform: 'uppercase',
                }}
              >
                {m.label}
              </span>
              <span style={{ fontFamily: 'var(--sans)', fontSize: '11px', color: 'var(--text-tertiary)' }}>
                {m.hint}
              </span>
            </button>
          );
        })}
      </div>
    </nav>
  );
}
