import { act } from 'react';
import { afterEach, expect, it, vi } from 'vitest';
import { mount, render, flush, unmount } from '../commissioner/commissionerDom';
import { useNativeLineup } from './hqNativeLineup';
import { heldLineup } from '../data/lineupTestData';
import type { HostedPort } from '../commissioner/hostedRoom';
import type { HostedResponse } from '../commissioner/api';
import wire from '../commissioner/hosted.wire.testdata.json';
import preview from '../commissioner/preview.wire.testdata.json';

afterEach(unmount);
it('previews native lineup membership with the read revision; discard never executes', async () => {
  const readings = wire as Record<string, HostedResponse>;
  const port = vi.fn<HostedPort>(async r => r.mode === 'preview'
    ? { ...readings['lineup.read'], data: preview.confirm } : readings[r.command]);
  let editor!: ReturnType<typeof useNativeLineup>;
  const lineup = { ...heldLineup(), week: 1 };
  function Probe() { editor = useNativeLineup('native.hosted', 'franchise.01', lineup, port);
    return <p>{editor.message}</p>; }
  await mount(render(Probe, {})); await flush();
  expect(editor.canEdit).toBe(true);
  await act(async () => editor.assign([lineup.bench[0].id]));
  await act(async () => { await editor.save(); });
  expect(port).toHaveBeenCalledWith(expect.objectContaining({ command: 'commish.lineup.set',
    mode: 'preview', arguments: expect.objectContaining({ expected: (readings['lineup.read'].data as { revision: number }).revision, visibility: 'private',
      args: { lineup: { season: 2026, franchise: 'franchise.01', week: 1,
        starters: [lineup.bench[0].id] } } }) }), readings['lineup.read'].football);
  await act(async () => editor.discard());
  expect(editor.edited?.starters).toEqual(lineup.starters);
  expect(port.mock.calls.some(([r]) => r.mode === 'execute')).toBe(false);
});
