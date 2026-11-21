import { describe, expect, it } from 'vitest';
import { activeNoticesAndNotes, ledgerEntryDetail, noticeKindTag } from './CtrlLeagueV3';
import type { LeagueNotes, Note } from '../commissioner/notes';

const note = (o: Partial<Note> = {}): Note => ({
  id: 'n1', kind: 'notice', text: 'Lineups lock Sunday', pinned: false,
  who: 'Commissioner', when: '2026-11-17T12:00:00Z', withdrawn: false, change: 'c1', ...o,
});

describe('activeNoticesAndNotes', () => {
  it('returns empty when reading is absent', () => {
    expect(activeNoticesAndNotes(undefined)).toEqual([]);
  });

  it('filters withdrawn items and orders notices before league notes', () => {
    const reading: LeagueNotes = {
      league: 'l1', revision: 1,
      notices: [note({ id: 'n1' }), note({ id: 'n2', withdrawn: true })],
      notes: [note({ id: 'm1', kind: 'note' }), note({ id: 'm2', kind: 'note', withdrawn: true })],
      releases: [], mail: { words: '', recorded: [] },
    };
    expect(activeNoticesAndNotes(reading).map(item => item.id)).toEqual(['n1', 'm1']);
  });
});

describe('noticeKindTag', () => {
  it('labels pinned and unpinned notices and league notes', () => {
    expect(noticeKindTag({ kind: 'notice', pinned: true })).toBe('PINNED NOTICE');
    expect(noticeKindTag({ kind: 'notice', pinned: false })).toBe('NOTICE');
    expect(noticeKindTag({ kind: 'note', pinned: true })).toBe('PINNED NOTE');
    expect(noticeKindTag({ kind: 'note', pinned: false })).toBe('LEAGUE NOTE');
  });
});

describe('ledgerEntryDetail', () => {
  it('includes reason when present and falls back to actor alone', () => {
    expect(ledgerEntryDetail({ who: 'Commissioner', why: 'Rule vote' })).toBe('Commissioner · Rule vote');
    expect(ledgerEntryDetail({ who: 'Commissioner', why: '' })).toBe('Commissioner');
  });

  it('maps account ids to member display names and keeps the id as fallback', () => {
    const people = { seats: [{ franchise: 'f25', name: 'Arizona Cardinals', abbreviation: 'ARI',
      owner: 'acct.az' }] };
    expect(ledgerEntryDetail({ who: 'acct.az', why: 'Updated scoring' }, people))
      .toBe('Arizona Cardinals · Updated scoring');
    expect(ledgerEntryDetail({ who: 'via AI/CLI, acting for acct.az', why: '' }, people))
      .toBe('via AI/CLI, acting for Arizona Cardinals');
    expect(ledgerEntryDetail({ who: 'acct.unknown', why: 'Manual fix' }, people))
      .toBe('acct.unknown · Manual fix');
  });
});
