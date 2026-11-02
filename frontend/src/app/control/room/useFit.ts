import { useLayoutEffect, useRef, useState } from 'react';

/** The mock's fitN(): how many table rows fit the panel, so a page never scrolls. */
export function useFit(initial: number) {
  const panel = useRef<HTMLDivElement>(null);
  const [per, setPer] = useState(initial);
  useLayoutEffect(() => {
    const p = panel.current;
    if (!p || typeof ResizeObserver === 'undefined') return;
    const fit = () => {
      const tr = p.querySelector('tbody tr') as HTMLElement | null;
      const th = p.querySelector('thead') as HTMLElement | null;
      const pg = p.querySelector('.cr-pager') as HTMLElement | null;
      if (!tr || !th || !pg || !tr.offsetHeight) return;
      const n = Math.max(8, Math.floor((p.clientHeight - th.offsetHeight - pg.offsetHeight - 6) / tr.offsetHeight));
      setPer(was => (was === n ? was : n));
    };
    const ro = new ResizeObserver(fit);
    ro.observe(p);
    fit();
    return () => ro.disconnect();
  });
  return { panel, per };
}
