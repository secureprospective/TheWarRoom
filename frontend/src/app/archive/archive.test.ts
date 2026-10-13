import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { ReportView } from '../reports/ReportView';
import type { Report } from '../reports/contract';
import { ArchiveView } from './Archive';
import type { Archive } from './state';

const report = (id: string, title: string) => ({ id, title, note: '', status: 'ok', source: 'Source: test',
  asOf: '', how: 'Counted.', columns: ['Season'], rows: [{ cells: ['2016'] }], summary: [] });
const archive = (): Archive => ({ season: 2016, seasons: [{ value: '2016', label: '2016' }], team: '',
  other: '', teams: [], board: { title: 'Pools / Archive', loading: '', refresh: 'Reread', howLabel: 'How',
    reports: [report('M-080', 'League history'), report('M-111', 'Pools')] } } as unknown as Archive);

describe('League archive', () => {
  it('puts oddity counts in closed disclosures with names grouped by franchise', () => {
    const oddity = { ...report('M-089', 'Recorded oddities'),
      status: 'ok' as const, link: null, columns: ['When', 'Recorded oddity', 'Who'],
      rows: [{ cells: ['2026 week 1', 'Zero-point starter · 2',
        'Cardinals: Coleman; Shakir\nSaints: Olave'], range: null }] } as Report;
    const markup = renderToStaticMarkup(createElement(ReportView, { report: oddity, howLabel: 'How' }));
    expect(markup).toContain('<summary>2026 week 1 · Zero-point starter · 2</summary>');
    expect(markup).toContain('<p>Cardinals: Coleman; Shakir</p><p>Saints: Olave</p>');
    expect(markup).not.toContain('<details open');
    expect(markup).not.toContain('<td>');
    expect(markup).not.toContain('ring');
  });
  it('answers every pool endpoint with the one Pools report, listed once', () => {
    for (const id of ['M-111', 'M-112', 'M-113', 'M-174', 'M-200', 'M-201']) {
      const markup = renderToStaticMarkup(createElement(ArchiveView, { archive: archive(), selected: [id] }));
      expect(markup).toContain('aria-label="Pools" aria-pressed="true"');
      expect(markup.match(/aria-label="Pools" aria-pressed/g)).toHaveLength(1);
      expect(markup).toContain('Season');
    }
  });
});
