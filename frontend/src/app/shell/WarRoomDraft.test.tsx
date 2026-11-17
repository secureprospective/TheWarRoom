// @vitest-environment jsdom
import { act } from 'react';
import type { ReactNode } from 'react';
import { createRoot } from 'react-dom/client';
import { afterEach, expect, it, vi } from 'vitest';

const testState = vi.hoisted(() => ({
  save: vi.fn(async (_native: unknown, players: string[]) => players),
  reading: {
    draft: {
      league: 'native.test', words: 'Draft is open', start: '2026-11-17T00:00:00Z',
      owners: [], order: [{ round: 1, words: '', picks: [] }], picks: [], onClock: null,
      players: {
        '0099': { id: '0099', name: 'Player A', position: 'QB', team: 'DAL' },
        '0100': { id: '0100', name: 'Player B', position: 'WR', team: 'NYG' },
      },
      available: [
        { id: '0099', name: 'Player A', position: 'QB', team: 'DAL' },
        { id: '0100', name: 'Player B', position: 'WR', team: 'NYG' },
      ],
    },
    queue: ['0099', '0100'],
  },
}));

vi.mock('./warroomReads', () => ({ useWarRoomRead: () => ({ reading: testState.reading }) }));
vi.mock('./warroomDraftReads', () => ({
  readWarRoomDraft: vi.fn(), saveWarRoomDraftQueue: testState.save,
}));
vi.mock('./HQAct', () => ({ HQAct: ({ onAction, children, label, disabled }: {
  onAction?: () => void; children?: ReactNode; label?: string; disabled?: boolean;
}) => <button aria-label={label ?? (typeof children === 'string' ? children : undefined)}
  disabled={disabled} onClick={() => onAction?.()}>{children}</button> }));

import { WarRoomDraft } from './WarRoomDraft';

afterEach(() => vi.clearAllMocks());

it('shows the hosted queue and saves a reorder as one ordered replacement', async () => {
  const container = document.createElement('div');
  const root = createRoot(container);
  const native = { league: 'native.test', mine: 'franchise.02', leave: () => {} };
  try {
    await act(async () => {
      root.render(<WarRoomDraft context={{ native, mine: 'franchise.02', week: 1,
        snapshot: { franchises: { value: [] } } } as never} select={() => {}} />);
    });
    expect(container.textContent).toContain('Player A');
    expect(container.textContent).toContain('Player B');
    const moveUp = container.querySelector('[aria-label="Move Player B up"]');
    expect(moveUp).not.toBeNull();
    await act(async () => {
      (moveUp as HTMLButtonElement).click();
      await new Promise(resolve => setTimeout(resolve, 0));
    });
    expect(testState.save).toHaveBeenCalledWith(native, ['0100', '0099']);
    const queuedNames = [...container.querySelectorAll('.qi > span:nth-child(3)')]
      .map(node => node.textContent);
    expect(queuedNames).toEqual(['Player B', 'Player A']);
  } finally {
    await act(async () => { root.unmount(); });
  }
});
