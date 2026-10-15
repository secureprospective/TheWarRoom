import { useEffect, useId, useRef, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { Act } from '../commands/Act';

const focusable = 'button, input, select, textarea, a[href], [tabindex]';

// SideTray is the one tray every commissioner change opens in. Its Close is an Act on the screen that owns
// the tray (screen), so closing goes through the command registry like every other control.
export default function SideTray({
  title, screen, opener, background, onClose, children, closeLabel = 'Close',
}: {
  title: string; screen: string; opener: HTMLElement; background?: HTMLElement | null;
  onClose(): void; children: ReactNode; closeLabel?: string;
}) {
  const id = useId();
  const panel = useRef<HTMLElement>(null);
  useEffect(() => {
    background?.setAttribute('inert', '');
    const first = () => panel.current?.querySelector<HTMLElement>(focusable)?.focus();
    first();
    const keepInside = (event: FocusEvent) => {
      if (panel.current && !panel.current.contains(event.target as Node)) first();
    };
    document.addEventListener('focusin', keepInside);
    return () => {
      document.removeEventListener('focusin', keepInside);
      background?.removeAttribute('inert');
      opener.focus();
    };
  }, [opener, background]);
  const tray = (
    <div className="tray-shade">
      <section ref={panel} className="side-tray" role="dialog" aria-modal="true" aria-labelledby={id}
        onKeyDown={event => {
          if (event.key === 'Escape') { event.preventDefault(); event.stopPropagation(); onClose(); }
          if (event.key !== 'Tab' || !panel.current) return;
          const items = Array.from(panel.current.querySelectorAll<HTMLElement>(focusable))
            .filter(e => !e.hasAttribute('disabled') && e.tabIndex >= 0 && !e.closest('[hidden]'));
          const first = items[0];
          const last = items[items.length - 1];
          if (event.shiftKey && document.activeElement === first) {
            event.preventDefault(); last?.focus();
          } else if (!event.shiftKey && document.activeElement === last) {
            event.preventDefault(); first?.focus();
          }
        }}>
        <header className="tray-heading"><h3 id={id}>{title}</h3>
          <Act verb="commissioner.surface" variant="text" label={closeLabel}
            args={{ screen, action: 'close', value: '' }}>Close</Act>
        </header>
        {children}
      </section>
    </div>
  );
  // Drawn at the app root, so no scrolling screen beneath can paint its scroll bar over the tray. Outside
  // the app (a bare test host) there is no root to draw at, and the tray stays where it was opened.
  const layer = opener.closest?.('.twr-app');
  return layer ? createPortal(tray, layer) : tray;
}
