import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { act } from 'react';
import { createHold } from './hold';
import { HoldToFire } from './HoldToFire';
import { commands } from '../commands/registry';
import { all, flush, mount, render, rerender, unmount, type Element } from './commissionerDom';

let sent: { mock: { calls: unknown[][] }; mockRestore(): void };
beforeEach(() => {
  vi.useFakeTimers();
  sent = vi.spyOn(commands, 'dispatch').mockImplementation(() => undefined);
});
afterEach(async () => { await unmount(); vi.useRealTimers(); sent.mockRestore(); });

type Handler = (event: unknown) => void;
const fire = (button: Element, name: string, event: object = {}) =>
  act(async () => { (button.props()[name] as Handler)({ preventDefault() {}, button: 0, ...event }); });
const wait = (ms: number) => act(async () => { vi.advanceTimersByTime(ms); });
const press = async (disabled = false) => {
  const host = await mount(render(HoldToFire, { label: 'Confirm', body: 'B', disabled }));
  return { host, button: all(host, 'span')[0] };
};
const fired = () => sent.mock.calls.filter(c => c[0] === 'commissioner.execute');
const said = (host: Element) => all(host, 'span').find(s => s.getAttribute('role') === 'status')?.textContent;

describe('Hold to fire', () => {
  it('fires once after a full hold and never when released early', () => {
    const fired = vi.fn();
    const hold = createHold(fired);
    hold.start();
    vi.advanceTimersByTime(599);
    expect(hold.release()).toBe('early');
    vi.advanceTimersByTime(5000);
    expect(fired).not.toHaveBeenCalled();
    hold.start(); hold.start();
    vi.advanceTimersByTime(600);
    expect(fired).toHaveBeenCalledTimes(1);
    vi.advanceTimersByTime(5000);
    expect(fired).toHaveBeenCalledTimes(1);
    hold.start();
    vi.advanceTimersByTime(5000);
    expect(fired).toHaveBeenCalledTimes(1);
    expect(hold.release()).toBe('done');
    hold.start();
    vi.advanceTimersByTime(600);
    expect(fired).toHaveBeenCalledTimes(2);
  });
  it('does not fire when the pointer is let go early, and says so', async () => {
    const { host, button } = await press();
    await fire(button, 'onPointerDown');
    expect(said(host)).toContain('Keep holding');
    await wait(400);
    await fire(button, 'onPointerUp');
    await wait(2000);
    expect(fired()).toHaveLength(0);
    expect(said(host)).toBe('Released early. Nothing was sent.');
  });
  it('fires once when the pointer is held for the full time', async () => {
    const { host, button } = await press();
    await fire(button, 'onPointerDown');
    await wait(600);
    await flush();
    expect(fired()).toEqual([['commissioner.execute', { body: 'B' }]]);
    await fire(button, 'onPointerUp');
    await wait(2000);
    expect(fired()).toEqual([['commissioner.execute', { body: 'B' }]]);
    expect(said(host)).toContain('Confirmed');
  });
  it('fires from a held Space or Enter, ignoring key repeat, and cancels on release', async () => {
    const { button } = await press();
    for (const key of ['Enter', ' ']) {
      await fire(button, 'onKeyDown', { key, repeat: false });
      await wait(300);
      await fire(button, 'onKeyDown', { key, repeat: true });
      await wait(299);
      await fire(button, 'onKeyUp', { key });
    }
    expect(fired()).toHaveLength(0);
    await fire(button, 'onKeyDown', { key: ' ', repeat: false });
    await wait(300);
    await fire(button, 'onKeyDown', { key: ' ', repeat: true });
    await wait(400);
    expect(fired()).toEqual([['commissioner.execute', { body: 'B' }]]);
    await fire(button, 'onKeyUp', { key: ' ' });
  });
  it('never fires while disabled, even when the handlers are called', async () => {
    const { host, button } = await press(true);
    expect(all(host, 'button')[0].attributes.has('disabled')).toBe(true);
    await fire(button, 'onPointerDown');
    await fire(button, 'onKeyDown', { key: 'Enter', repeat: false });
    await wait(5000);
    expect(fired()).toHaveLength(0);
  });
  it('lets go when it becomes disabled in the middle of a hold', async () => {
    const host = await mount(render(HoldToFire, { label: 'Confirm', body: 'B' }));
    await fire(all(host, 'span')[0], 'onPointerDown');
    await wait(300);
    await rerender(render(HoldToFire, { label: 'Confirm', body: 'B', disabled: true }));
    await wait(5000);
    expect(fired()).toHaveLength(0);
  });
});

const css = readFileSync(new URL('./commissioner.css', import.meta.url), 'utf8');
const style = (selector: string) => css.slice(css.indexOf(selector)).split('}')[0];
describe('Hold presentation and lifetime', () => {
  it('fills the button itself, with full width and visible help and status on separate lines', async () => {
    const { host } = await press();
    const button = all(host, 'button')[0];
    const fill = all(host, 'span').find(s => s.getAttribute('class') === 'hold-fill');
    expect(fill?.parentNode).toBe(button);
    expect(style('.twr-app .hold-to-fire {')).toContain('display: grid');
    expect(style('.twr-app .hold-to-fire .act {')).toContain('width: 100%');
    expect(style('.twr-app .hold-to-fire .act {')).toContain('overflow: hidden');
    expect(css).not.toMatch(/clip:|width: 1px/);
    expect(host.textContent).toContain('Press and hold to confirm');
    expect(all(host, 'span').find(s => s.getAttribute('role') === 'status')?.parentNode)
      .toBe(button.parentNode);
    expect(button.props().onClick).toBeUndefined();
  });
  it('clears confirmation when an answer arrives or the inputs change; cancels an edited hold', async () => {
    const { host, button } = await press();
    await fire(button, 'onPointerDown');
    await wait(600);
    expect(said(host)).toContain('Confirmed');
    await rerender(render(HoldToFire, { label: 'Confirm', body: 'B', answer: 'done' }));
    expect(said(host)).toBe('');
    await fire(button, 'onPointerDown');
    await wait(600);
    expect(said(host)).toContain('Confirmed');
    await rerender(render(HoldToFire, { label: 'Confirm', body: 'C', answer: 'done' }));
    expect(said(host)).toBe('');
    await fire(button, 'onPointerDown');
    await wait(300);
    await rerender(render(HoldToFire, { label: 'Confirm', body: 'D', answer: 'done' }));
    await wait(600);
    expect(fired()).toHaveLength(2);
    expect(said(host)).toBe('');
  });
});
