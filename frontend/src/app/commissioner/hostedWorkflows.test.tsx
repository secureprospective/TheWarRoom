import { expect, it } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { HostedWorkflows, type HostedTask } from './HostedWorkflows';
import { createHostedRoom } from './hostedRoom';
import type { HostedResponse } from './api';
import wire from './hosted.wire.testdata.json';

const readings = wire as Record<string, HostedResponse>;
it.each(['bids', 'waivers', 'trades', 'results', 'rollover'] as HostedTask[])(
  'renders the hosted %s task with named choices',
  async (task) => {
    const room = createHostedRoom(
      'native.hosted',
      'franchise.01',
      async (r) => readings[r.command],
    );
    await room.refresh();
    const reading = room.peek().reading!;
    expect(reading).toBeDefined();
    const html = renderToStaticMarkup(
      <HostedWorkflows
        task={task}
        room={room}
        reading={reading}
        franchise="franchise.01"
        commissioner
      />,
    );
    const visible = html.replace(/<[^>]*>/g, '');
    expect(visible).not.toMatch(/franchise\.\d|native\.hosted|6001|0099/);
    expect(visible).toContain('Reason for this change');
    expect(visible).toContain('Check this change');
    if (task === 'bids') expect(visible).toContain('Not even the commissioner can peek');
    if (task === 'results') expect(visible).toContain('Computed standings');
  },
);
it('does not offer commissioner-only corrections or rollover to an ordinary owner', async () => {
  const room = createHostedRoom('native.hosted', 'franchise.01', async (r) => readings[r.command]);
  await room.refresh();
  const reading = room.peek().reading!;
  const html = renderToStaticMarkup(
    <HostedWorkflows
      task="rollover"
      room={room}
      reading={reading}
      franchise="franchise.01"
      commissioner={false}
    />,
  );
  expect(html).toContain('Only your commissioner');
  expect(html).not.toContain('Check this change');
});

it('uses held names for preview targets rather than showing internal identifiers', async () => {
  const { hostedWords } = await import('./hostedWords');
  const room = createHostedRoom('native.hosted', 'franchise.01', async r => readings[r.command]);
  await room.refresh();
  expect(hostedWords(room.peek().reading, 'franchise.01, player 0099'))
    .toBe('Practice team 1, player Practice quarterback');
});
