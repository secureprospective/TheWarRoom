import type { ReactNode } from 'react';

interface WorkspaceProps {
  children: ReactNode;
  title?: string;
}

// Center zone (Session A): a thin mono chrome bar over the mounted module. The
// module scrolls inside its own container; the shell itself never scrolls.
export function Workspace({ children, title }: WorkspaceProps) {
  return (
    <section
      style={{
        minWidth: 0, // grid overflow guard
        position: 'relative',
        background: 'var(--surface-canvas)',
        display: 'flex',
        flexDirection: 'column',
        overflow: 'hidden',
      }}
    >
      <header
        style={{
          height: '32px',
          borderBottom: '1px solid var(--hairline)',
          display: 'flex',
          alignItems: 'center',
          padding: '0 16px',
          background: 'var(--surface-raised)',
          flexShrink: 0,
        }}
      >
        {title ? (
          <span
            style={{
              fontFamily: 'var(--mono)',
              color: 'var(--text-tertiary)',
              fontSize: '12px',
              letterSpacing: '0.05em',
            }}
          >
            {title}
          </span>
        ) : null}
      </header>
      <div style={{ flex: 1, overflow: 'auto', position: 'relative' }}>{children}</div>
    </section>
  );
}
