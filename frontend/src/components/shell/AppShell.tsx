import { useRef, useState, useCallback, type ReactNode, type CSSProperties, type PointerEvent } from 'react';
import type { ModuleId, Density } from './types';
import { NavRail } from './NavRail';
import { Workspace } from './Workspace';
import { Inspector } from './Inspector';
import { CommsStrip } from './CommsStrip';

interface AppShellProps {
  module: ModuleId;
  onModule: (m: ModuleId) => void;
  density: Density;
  inspectorOpen: boolean;
  onInspectorClose: () => void;
  onSummon: (t: 'comms' | 'calendar' | 'feed') => void;
  children: ReactNode; // the active module element
  workspaceTitle?: string;
  inspector?: ReactNode; // inspector body (optional; empty state otherwise)
}

// Clamps for the edge-resize (Ledger A12). Nav and inspector each have a sane
// operator range; presets (Draft Mode, etc.) are a later nicety.
const NAV_MIN = 160;
const NAV_MAX = 320;
const INSP_MIN = 280;
const INSP_MAX = 480;
const COMMS_W = 48;

type Ghost = { x: number } | null;

// The confirmed 4-zone instrument console (Session A grid + Session C tokens).
// Grid is 3 columns — nav | workspace | comms — and the Inspector floats as a
// translateX OVERLAY over the workspace's right edge (NOT a reserved column;
// the rejected width-trade would reflow the workspace on every open).
//
// Edge-resize (A12): dragging a divider shows a 1px ghost guide and commits the
// new width ONCE on pointer-UP by writing the layout CSS var onto the shell
// root — no live per-frame reflow, honoring the WebKitGTK cheap-paint floor.
export function AppShell({
  module,
  onModule,
  density,
  inspectorOpen,
  onInspectorClose,
  onSummon,
  children,
  workspaceTitle,
  inspector,
}: AppShellProps) {
  const shellRef = useRef<HTMLDivElement>(null);
  const [ghost, setGhost] = useState<Ghost>(null);

  const startResize = useCallback(
    (edge: 'nav' | 'inspector', varName: string, min: number, max: number) =>
      (e: PointerEvent<HTMLDivElement>) => {
        e.preventDefault();
        const shell = shellRef.current;
        if (!shell) return;
        const rect = shell.getBoundingClientRect();

        // For the nav divider the width is measured from the left edge; for the
        // inspector's left-edge handle it is measured from just-inside the comms
        // strip (the inspector's right edge).
        const widthAt = (clientX: number) =>
          edge === 'nav'
            ? clientX - rect.left
            : rect.right - COMMS_W - clientX;
        const clamp = (w: number) => Math.max(min, Math.min(max, w));

        const onMove = (ev: globalThis.PointerEvent) => {
          const w = clamp(widthAt(ev.clientX));
          // Ghost x is the divider's screen position (left→right).
          const x = edge === 'nav' ? rect.left + w : rect.right - COMMS_W - w;
          setGhost({ x: x - rect.left });
        };
        const onUp = (ev: globalThis.PointerEvent) => {
          const w = clamp(widthAt(ev.clientX));
          shell.style.setProperty(varName, `${w}px`);
          setGhost(null);
          window.removeEventListener('pointermove', onMove);
          window.removeEventListener('pointerup', onUp);
        };
        window.addEventListener('pointermove', onMove);
        window.addEventListener('pointerup', onUp);
      },
    [],
  );

  const handleStyle: CSSProperties = {
    position: 'absolute',
    top: 0,
    bottom: 0,
    width: '4px',
    cursor: 'col-resize',
    background: 'transparent',
    zIndex: 30,
  };

  return (
    <div
      ref={shellRef}
      data-density={density}
      style={{
        position: 'relative',
        height: '100vh',
        overflow: 'hidden',
        color: 'var(--text-primary)',
        fontFamily: 'var(--sans)',
        background: 'var(--surface-canvas)',
        display: 'grid',
        gridTemplateColumns: 'var(--nav-w) 1fr var(--comms-w)',
      }}
    >
      <NavRail active={module} onSelect={onModule} />
      <Workspace title={workspaceTitle}>{children}</Workspace>
      <CommsStrip onSummon={onSummon} />

      {/* Inspector overlay — floats over the workspace, left of the comms strip */}
      <Inspector open={inspectorOpen} onClose={onInspectorClose}>
        {inspector}
      </Inspector>

      {/* Edge-resize handles (A12). Inspector handle only while it's open. */}
      <div
        aria-hidden
        onPointerDown={startResize('nav', '--nav-w', NAV_MIN, NAV_MAX)}
        style={{ ...handleStyle, left: 'calc(var(--nav-w) - 2px)' }}
      />
      {inspectorOpen ? (
        <div
          aria-hidden
          onPointerDown={startResize('inspector', '--insp-w', INSP_MIN, INSP_MAX)}
          style={{ ...handleStyle, right: 'calc(var(--comms-w) + var(--insp-w) - 2px)' }}
        />
      ) : null}

      {/* 1px ghost guide (only during an active drag) */}
      {ghost ? (
        <div
          aria-hidden
          style={{
            position: 'absolute',
            top: 0,
            bottom: 0,
            left: `${ghost.x}px`,
            width: '1px',
            background: 'var(--edge-focus)',
            zIndex: 40,
            pointerEvents: 'none',
          }}
        />
      ) : null}
    </div>
  );
}
