import { useState } from 'react';
import { describe, expect, it } from 'vitest';
import { all, click, mount, render, settle, type Element } from '../../commissioner/commissionerDom';
import { open, stub } from '../../commissioner/commissionerStub';
import ControlRoom from './ControlRoom';
import { data, league, roomHooks } from './roomTestKit';
import peopleWire from '../../commissioner/people.wire.testdata.json';

roomHooks();

// The shell's rail as TargetApp gives it: a slot under the Control Room item and the rail footer.
function Shell() {
  const [tree, setTree] = useState<HTMLDivElement | null>(null);
  const [footer, setFooter] = useState<HTMLDivElement | null>(null);
  return (
    <>
      <nav>
        <div data-slot="tree" ref={setTree} />
        <div data-slot="footer" ref={setFooter} />
      </nav>
      <ControlRoom league={league} initial={data} tree={tree} footer={footer} />
    </>
  );
}
const slot = (host: Element, name: string) =>
  all(host, 'div').find((d) => d.getAttribute('data-slot') === name)!;

describe('Control Room · rail tree and Signed in footer', () => {
  it('offers a visitor the sign-in form', async () => {
    const s = stub({ TargetCommissionerSeat: async () => ({ hasAccount: true }) },
      { people: { ...peopleWire.commissioner, yourRole: 'visitor' } });
    const host = await mount(render(Shell, {}), s.app);
    await open(league);
    await settle();
    const link = all(slot(host, 'footer'), 'button').find(b => b.textContent ===
      'Sign in to claim your seat');
    expect(link).toBeTruthy();
    await click(link!);
    await settle();
    expect(host.textContent).toContain('Restored from another computer?');
  });

  it(
    "puts the nine leaves under the rail item and the viewer's role in the footer; " +
      'a leaf opens its branch',
    async () => {
      const s = stub({});
      const host = await mount(render(Shell, {}), s.app);
      await open(league);
      await settle();
      const leaves = () => all(slot(host, 'tree'), 'button');
      expect(leaves().map((b) => b.textContent)).toEqual([
        '├Jobs',
        '├Situations',
        '├Rules',
        '├Season desk',
        '├Fix',
        '├Picks',
        '├People',
        '├League',
        '└Server',
      ]);
      expect(slot(host, 'footer').textContent).toMatch(/^Signed in: [A-Z—]/);
      await click(leaves()[6]!);
      expect(
        all(host, 'button').find(
          (b) => b.getAttribute('role') === 'tab' && b.getAttribute('aria-selected') === 'true',
        )!.textContent,
      ).toBe('PEOPLE');
      expect(leaves().find((b) => b.getAttribute('aria-current') === 'page')!.textContent).toBe('├People');
    },
  );
});
