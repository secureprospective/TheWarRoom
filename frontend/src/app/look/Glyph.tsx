import type { Position } from '../data/contract';

export const TIERS = ['contender', 'hunt', 'rebuild'] as const;
export const POSTURES = ['now', 'later', 'like-for-like'] as const;
export const PRIVATE_MARKERS = ['watch', 'bait'] as const;
export type GlyphName =
  | Position
  | (typeof TIERS)[number]
  | (typeof POSTURES)[number]
  | (typeof PRIVATE_MARKERS)[number];

export const glyphPaths: Record<GlyphName, string> = {
  QB: 'M2 8 6 3l4 1 4 4-4 4-4 1Z M5 8h6 M8 5v6',
  RB: 'M2 12 6 8 3 5 M6 8h5l3-5 M10 8l3 5',
  WR: 'M2 3v7l6 4 6-4V3 M2 6l6 5 6-5',
  TE: 'M3 3h10v10H3Z M3 8h10 M8 3v10',
  K: 'M2 13h12 M4 11l4-4 5 2 M8 7 7 2 M11 3h3',
  DT: 'M2 4h12v8H2Z M5 4v8 M11 4v8',
  DE: 'M2 13V3h5 M14 13V3H9 M2 8h4 M14 8h-4',
  LB: 'M8 2 14 5v5l-6 4-6-4V5Z M5 8h6',
  CB: 'M3 2v12 M3 3h7l3 5-3 5H3',
  S: 'M8 2 14 8l-6 6-6-6Z M8 5v6 M5 8h6',
  FLAG: 'M3 14V2 M3 2h10l-3 3 3 3H3',
  contender: 'M2 4h3V2h6v2h3v4l-3 2H5L2 8Z M8 10v4 M5 14h6',
  hunt: 'M8 2a6 6 0 1 0 0 12 6 6 0 0 0 0-12 M8 5v6 M5 8h6',
  rebuild: 'M2 8 8 2l6 6 M4 6v8h8V6 M6 14V9h4v5',
  now: 'M9 2 3 9h5l-1 5 6-7H8Z',
  later: 'M3 2h10 M3 14h10 M4 2v3l8 6v3 M12 2v3l-8 6v3',
  'like-for-like': 'M2 5h12l-3-3 M14 11H2l3 3',
  watch: 'M1 8s3-5 7-5 7 5 7 5-3 5-7 5-7-5-7-5Z M8 6a2 2 0 1 0 0 4 2 2 0 0 0 0-4',
  bait: 'M9 2v8a4 4 0 0 1-8 0V8l3 3 M7 2h4',
};

export function Glyph({ name }: { name: GlyphName }) {
  return (
    <svg
      className="classification-glyph"
      viewBox="0 0 16 16"
      role="img"
      aria-label={name}
    >
      <title>{name}</title>
      <path d={glyphPaths[name]} />
    </svg>
  );
}
