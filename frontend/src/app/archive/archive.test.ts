import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { ArchiveView } from './Archive';
import type { Archive } from './state';

const report = (id: string, title: string) => ({ id, title, note: '', status: 'ok', source: 'Source: test',
  asOf: '', how: 'Counted.', columns: ['Season'], rows: [{ cells: ['2016'] }], summary: [] });
const archive = (): Archive => ({ season: 2016, seasons: [{ value: '2016', label: '2016' }], team: '',
  other: '', teams: [], board: { title: 'Pools / Archive', loading: '', refresh: 'Reread', howLabel: 'How',
    reports: [report('M-080', 'League history'), report('M-111', 'Pools')] } } as unknown as Archive);

describe('League archive', () => {
  it('answers every pool endpoint with the one Pools report, listed once', () => {
    for (const id of ['M-111', 'M-112', 'M-113', 'M-174', 'M-200', 'M-201']) {
      const markup = renderToStaticMarkup(createElement(ArchiveView, { archive: archive(), selected: [id] }));
      expect(markup).toContain('aria-label="Pools" aria-pressed="true"');
      expect(markup.match(/aria-label="Pools" aria-pressed/g)).toHaveLength(1);
      expect(markup).toContain('Season');
    }
  });
});
