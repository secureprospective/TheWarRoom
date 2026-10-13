import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import { ReportView } from '../reports/ReportView';
import type { Report } from '../reports/contract';
import { ArchiveView } from './Archive';
import type { Archive } from './state';

const report = (id: string, title: string) => ({ id, title, note: '', status: 'ok' as const, link: null,
  source: 'Source: test',
  asOf: '', how: 'Counted.', columns: ['Season'], rows: [{ cells: ['2016'], range: null }], summary: [] });
const archive = (): Archive => ({ season: 2016, seasons: [{ value: '2016', label: '2016' }], team: '',
  other: '', teams: [], board: { title: 'Archive', loading: '', refresh: 'Reread', howLabel: 'How',
    mflLabel: '', mflUrl: '',
    reports: [report('M-080', 'League history'), report('M-079', 'Player additions')] } });

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
  it('does not turn a retired pool selection into a Pools report', () => {
    const stale = archive();
    stale.board.reports.push(report('M-111', 'Pools'));
    const markup = renderToStaticMarkup(createElement(ArchiveView, {
      archive: stale, selected: ['M-112'],
    }));
    expect(markup).toContain('aria-label="League history" aria-pressed="true"');
    expect(markup).not.toContain('aria-label="Pools" aria-pressed="true"');
  });
  it('keeps the remaining season report selectable without pool aliases', () => {
    const markup = renderToStaticMarkup(createElement(ArchiveView, {
      archive: archive(), selected: ['M-079'],
    }));
    expect(markup).toContain('aria-label="Player additions" aria-pressed="true"');
    expect(markup).not.toContain('Pools');
  });
});
