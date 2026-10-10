import { it, expect } from 'vitest';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { DraftControls, DraftReading } from './Draft';
import { countNoun } from './ReadingView';
import type { SurfaceReading } from './contract';
import { unavailable } from './provider';

it('Draft values are visible while long action names remain accessible', () => {
  const html = renderToStaticMarkup(createElement(DraftControls));
  for (const [label, value] of [
    ['Filter draft stretch 1.01–1.10', '1.01–1.10'],
    ['Filter NFL 1st', '1st'],
    ['Draft profile QB', 'QB'],
    ['Use classes 2017–2026', '2017–2026'],
    ['Judge on first 3 seasons', 'First 3 seasons'],
  ]) {
    expect(html).toContain(`aria-label="${label}"`);
    expect(html).toContain(`>${value}</button>`);
    expect(html).not.toContain(`>${label}</button>`);
  }
});
it('every row tone is display prose, not a raw enum caption, and a single record stays singular', () => {
  for (const [tone, words] of [
    ['down', 'below its slot beyond chance'],
    ['even', 'within chance of its slot'],
    ['up', 'above its slot beyond chance'],
  ]) {
    const reading: SurfaceReading = {
      ...unavailable(''),
      status: 'ok',
      absenceReason: null,
      count: 1,
      value: {
        summary: '1 pick',
        sections: [
          {
            id: 'L-16',
            title: 'Returns',
            note: '',
            rows: [
              {
                key: 'QB',
                label: 'QB',
                count: 1,
                range: null,
                detail: `Against the same slots · ${words}`,
                mark: tone,
              },
            ],
          },
        ],
      },
    };
    const html = renderToStaticMarkup(createElement(DraftReading, { reading }));
    expect(html).toContain(words);
    expect(html).not.toContain(`· ${tone}`);
    expect(html).toContain('1 record');
    expect(html).not.toContain('1 records');
  }
  for (const noun of ['record', 'pick', 'trade']) {
    expect(countNoun(1, noun)).toBe(`1 ${noun}`);
    expect(countNoun(0, noun)).toBe(`0 ${noun}s`);
    expect(countNoun(2, noun)).toBe(`2 ${noun}s`);
  }
});
