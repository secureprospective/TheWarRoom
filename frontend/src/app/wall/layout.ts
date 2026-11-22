import { useEffect, useState, type RefObject } from 'react';

export const PHONE_W = 390;
export const PHONE_H = 844;
export const BEZEL = 8;
export const FRAME_W = PHONE_W + 2 * BEZEL;
export const FRAME_H = PHONE_H + 2 * BEZEL;
export const TICKER_H = 48;
export const WALL_GAP = 10;
export const WALL_PAD = 10;

const HEAD_BASE = 28;
const HEAD_MIN = 20;
const TILE_GAP_BASE = 4;
const HEAD_TOTAL_BASE = HEAD_BASE + TILE_GAP_BASE;
const HEAD_TOTAL_MIN = HEAD_MIN + TILE_GAP_BASE;
const MIN_RIGHT_RAIL = 240;
const MIN_BOTTOM_RAIL = 150;

export type RailMode = 'right' | 'bottom' | 'none';
export type Viewport = { w: number; h: number };

export type WallLayout = {
  cols: number; rows: number; scale: number;
  bezel: number; headH: number; tileGap: number; gap: number; pad: number;
  phoneW: number; phoneH: number; tileW: number; tileH: number;
  gridW: number; gridH: number;
  rail: RailMode; railW: number; railH: number;
};

type Candidate = { cols: number; rows: number; scale: number; headH: number; tileGap: number };

function fitCandidate(cols: number, n: number, stageW: number, stageH: number, gap: number): Candidate {
  const rows = Math.ceil(n / cols);
  const cellW = Math.max(40, (stageW - (cols - 1) * gap) / cols);
  const cellH = Math.max(80, (stageH - (rows - 1) * gap) / rows);
  const sProp = Math.min(cellW / FRAME_W, cellH / (FRAME_H + HEAD_TOTAL_BASE));
  if (HEAD_TOTAL_BASE * sProp >= HEAD_TOTAL_MIN) {
    const scale = Math.max(0.15, sProp);
    return { cols, rows, scale, headH: HEAD_BASE * scale, tileGap: TILE_GAP_BASE * scale };
  }
  const scale = Math.max(0.15, Math.min(cellW / FRAME_W, (cellH - HEAD_TOTAL_MIN) / FRAME_H));
  return { cols, rows, scale, headH: HEAD_MIN, tileGap: TILE_GAP_BASE };
}

/** chooseWallLayout picks cols x rows for n phones to maximise phone size inside the live viewport,
 *  then assigns any leftover aspect-mismatch band to a right or bottom Big Plays rail. */
export function chooseWallLayout(n: number, viewportW: number, viewportH: number,
  tickerH = TICKER_H, gap = WALL_GAP, pad = WALL_PAD): WallLayout {
  const count = Math.max(1, Math.floor(n));
  const stageW = Math.max(200, viewportW - 2 * pad);
  const stageH = Math.max(200, viewportH - tickerH - 2 * pad);
  let best = fitCandidate(1, count, stageW, stageH, gap);
  for (let cols = 2; cols <= count; cols++) {
    if (Math.ceil(count / (cols - 1)) === Math.ceil(count / cols)) continue;
    const next = fitCandidate(cols, count, stageW, stageH, gap);
    const diff = next.scale - best.scale;
    const betterTie = Math.abs(diff) < 1e-4 && (
      next.cols * next.rows - count < best.cols * best.rows - count ||
      (next.cols * next.rows === best.cols * best.rows && next.rows < best.rows)
    );
    if (diff > 1e-4 || betterTie) best = next;
  }
  const phoneW = FRAME_W * best.scale;
  const phoneH = FRAME_H * best.scale;
  const tileW = phoneW;
  const tileH = phoneH + best.headH + best.tileGap;
  const gridW = best.cols * tileW + (best.cols - 1) * gap;
  const gridH = best.rows * tileH + (best.rows - 1) * gap;
  const rightSpace = stageW - gridW - gap;
  const bottomSpace = stageH - gridH - gap;
  const canRight = rightSpace >= MIN_RIGHT_RAIL && stageH >= 320;
  const canBottom = bottomSpace >= MIN_BOTTOM_RAIL && stageW >= 320;
  let rail: RailMode = 'none';
  if (canRight && canBottom) {
    rail = rightSpace >= 280 || rightSpace * stageH >= stageW * bottomSpace ? 'right' : 'bottom';
  } else if (canRight) {
    rail = 'right';
  } else if (canBottom) {
    rail = 'bottom';
  }
  const railW = rail === 'right' ? Math.floor(rightSpace) : rail === 'bottom' ? Math.floor(stageW) : 0;
  const railH = rail === 'bottom' ? Math.floor(bottomSpace) : rail === 'right' ? Math.floor(stageH) : 0;
  return { ...best, bezel: BEZEL, gap, pad, phoneW, phoneH, tileW, tileH, gridW, gridH,
    rail, railW, railH };
}

const windowViewport = (): Viewport => ({
  w: typeof window !== 'undefined' && window.innerWidth > 0 ? window.innerWidth : 1920,
  h: typeof window !== 'undefined' && window.innerHeight > 0 ? window.innerHeight : 1080,
});

/** useViewport observes the wall container with ResizeObserver and recomputes on every resize. */
export function useViewport(ref: RefObject<HTMLElement | null>): Viewport {
  const [size, setSize] = useState<Viewport>(windowViewport);
  useEffect(() => {
    const el = ref.current;
    const apply = (w: number, h: number) => {
      const rw = Math.round(w), rh = Math.round(h);
      if (rw <= 0 || rh <= 0) return;
      setSize(prev => prev.w === rw && prev.h === rh ? prev : { w: rw, h: rh });
    };
    if (el) {
      const rect = el.getBoundingClientRect();
      if (rect.width > 0 && rect.height > 0) apply(rect.width, rect.height);
    }
    const onWin = () => { const v = windowViewport(); apply(v.w, v.h); };
    window.addEventListener('resize', onWin);
    if (typeof ResizeObserver === 'undefined' || !el) {
      return () => window.removeEventListener('resize', onWin);
    }
    const obs = new ResizeObserver(entries => {
      const entry = entries[entries.length - 1];
      if (entry) apply(entry.contentRect.width, entry.contentRect.height);
    });
    obs.observe(el);
    return () => { obs.disconnect(); window.removeEventListener('resize', onWin); };
  }, [ref]);
  return size;
}
