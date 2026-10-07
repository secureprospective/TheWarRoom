import { afterEach, describe, expect, it, vi } from 'vitest';
import type { Deadline } from '../data/contract';
import { connectClock, nextDelay } from './ticker';
import { clockDom } from './testDom';

const deadline = (at?: string): Deadline => ({
  id: 'trade', label: 'TRADE_DEADLINE', at, urgency: at ? 'U2' : 'U0', pinned: false, promoted: false,
});
const now = Date.parse('2026-10-06T12:00:00Z');
const at = (remaining: number) => new Date(now + remaining).toISOString();

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe('next clock delay', () => {
  it.each([
    { remaining: [], delay: undefined },
    { remaining: [undefined], delay: undefined },
    { remaining: [-1, -60_000], delay: undefined },
    { remaining: [0], delay: undefined },
    { remaining: [1], delay: 1000 },
    { remaining: [3_599_999], delay: 1000 },
    { remaining: [3_600_000], delay: 60_025 },
    { remaining: [3_600_001], delay: 26 },
    { remaining: [7_245_000, 3_610_000], delay: 10_025 },
    { remaining: [3_610_000, 7_245_000], delay: 10_025 },
    { remaining: [-1, undefined, 7_245_000], delay: 45_025 },
    { remaining: [3_600_001, 3_599_999], delay: 1000 },
  ])('schedules $remaining in $delay ms', ({ remaining, delay }) => {
    const deadlines = remaining.map((ms) => deadline(ms === undefined ? undefined : at(ms)));
    expect(nextDelay(deadlines, now)).toBe(delay);
  });
});

describe('single DOM clock ticker', () => {
  it('updates both mounts, crosses the hour boundary, and clears its sole timeout', () => {
    vi.useFakeTimers();
    vi.setSystemTime(now);
    const { root, element, document } = clockDom(['trade', 'trade']);
    const set = vi.spyOn(globalThis, 'setTimeout');
    const stop = connectClock(element, [deadline(at(3_600_000))]);
    expect(set).toHaveBeenCalledTimes(1);
    expect(set.mock.calls[0][1]).toBe(60_025);
    for (const node of root.querySelectorAll('[data-deadline]')) {
      expect(node.getAttribute('data-urgency')).toBe('U2');
      expect(node.textContent).toBe('1h 0m');
    }
    vi.advanceTimersByTime(60_025);
    expect(set).toHaveBeenCalledTimes(2);
    expect(set.mock.calls[1][1]).toBe(1000);
    expect(vi.getTimerCount()).toBe(1);
    for (const node of root.querySelectorAll('[data-deadline]')) {
      expect(node.getAttribute('data-urgency')).toBe('U3');
      expect(node.textContent).toBe('58m');
    }
    vi.advanceTimersByTime(10_000);
    expect(set).toHaveBeenCalledTimes(12);
    expect(vi.getTimerCount()).toBe(1);
    stop();
    expect(vi.getTimerCount()).toBe(0);
    expect(document.listeners.size).toBe(0);
    const text = root.textContent;
    vi.advanceTimersByTime(3_600_000);
    expect(root.textContent).toBe(text);
  });
  it('flips the minute after the guard, then reschedules from the new remaining time', () => {
    vi.useFakeTimers();
    vi.setSystemTime(now);
    const { root, element } = clockDom(['trade']);
    const set = vi.spyOn(globalThis, 'setTimeout');
    const stop = connectClock(element, [deadline(at(7_210_000))]);
    expect(root.textContent).toBe('2h 0m');
    expect(set.mock.calls[0][1]).toBe(10_025);
    vi.advanceTimersByTime(10_024);
    expect(root.textContent).toBe('2h 0m');
    vi.advanceTimersByTime(1);
    expect(root.textContent).toBe('1h 59m');
    expect(set.mock.calls[1][1]).toBe(60_000);
    expect(vi.getTimerCount()).toBe(1);
    stop();
  });
  it.each([[[]], [[deadline()]]])('does not schedule undated or empty deadlines: %j', (deadlines) => {
    vi.useFakeTimers();
    const { root, element } = clockDom(['trade']);
    const set = vi.spyOn(globalThis, 'setTimeout');
    const stop = connectClock(element, deadlines);
    expect(set).not.toHaveBeenCalled();
    expect(vi.getTimerCount()).toBe(0);
    if (deadlines.length) expect(root.textContent).toBe('—');
    stop();
  });
  it('paints passed deadlines without scheduling a timer', () => {
    vi.useFakeTimers();
    vi.setSystemTime(now);
    const { root, element } = clockDom(['trade']);
    const set = vi.spyOn(globalThis, 'setTimeout');
    const stop = connectClock(element, [deadline(at(-1))]);
    expect(root.textContent).toBe('passed');
    expect(root.querySelector('[data-deadline]')?.getAttribute('data-urgency')).toBe('U3');
    expect(set).not.toHaveBeenCalled();
    expect(vi.getTimerCount()).toBe(0);
    stop();
  });
  it('stops rescheduling after the last future deadline passes', () => {
    vi.useFakeTimers();
    vi.setSystemTime(now);
    const { root, element } = clockDom(['trade']);
    const stop = connectClock(element, [deadline(at(500))]);
    expect(vi.getTimerCount()).toBe(1);
    vi.advanceTimersByTime(1000);
    expect(root.textContent).toBe('passed');
    expect(vi.getTimerCount()).toBe(0);
    stop();
  });
  it('pauses while hidden and refreshes immediately on return', () => {
    vi.useFakeTimers();
    vi.setSystemTime(now);
    const { root, element, document } = clockDom(['trade']);
    const stop = connectClock(element, [deadline(at(7_200_000))]);
    document.visibility(true);
    expect(vi.getTimerCount()).toBe(0);
    vi.advanceTimersByTime(7_200_001);
    expect(root.textContent).toBe('2h 0m');
    document.visibility(false);
    expect(root.textContent).toBe('passed');
    expect(vi.getTimerCount()).toBe(0);
    stop();
    document.visibility(false);
    expect(vi.getTimerCount()).toBe(0);
  });
  it('clears the pending timer before ticking on visibility return', () => {
    vi.useFakeTimers();
    vi.setSystemTime(now);
    const { root, element, document } = clockDom(['trade']);
    const clear = vi.spyOn(globalThis, 'clearTimeout');
    const set = vi.spyOn(globalThis, 'setTimeout');
    const stop = connectClock(element, [deadline(at(7_210_000))]);
    vi.setSystemTime(now + 20_000);
    document.visibility(false);
    expect(root.textContent).toBe('1h 59m');
    expect(clear).toHaveBeenCalledWith(set.mock.results[0].value);
    expect(clear.mock.invocationCallOrder[0]).toBeLessThan(set.mock.invocationCallOrder[1]);
    expect(vi.getTimerCount()).toBe(1);
    stop();
  });
  it('starts hidden without a timeout and chooses the fastest deadline across the app', () => {
    vi.useFakeTimers();
    vi.setSystemTime(now);
    const { element, document } = clockDom(['trade']);
    document.hidden = true;
    const set = vi.spyOn(globalThis, 'setTimeout');
    const stop = connectClock(element, [
      { ...deadline(at(259_200_000)), id: 'draft' },
      deadline(at(1_800_000)),
    ]);
    expect(set).not.toHaveBeenCalled();
    document.visibility(false);
    expect(set).toHaveBeenCalledTimes(1);
    expect(set.mock.calls[0][1]).toBe(1000);
    expect(vi.getTimerCount()).toBe(1);
    stop();
  });
});
