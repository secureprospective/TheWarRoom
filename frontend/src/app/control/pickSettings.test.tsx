import { afterEach, expect, it, vi } from 'vitest';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { commands } from '../commands/registry';
import { dataState } from '../datasources/state';
import PickSettings from './PickSettings';

afterEach(() => vi.restoreAllMocks());
const param = (key: string, label: string, value: number) => ({ key, label, value, group: 'Picks',
  position: '', description: 'Weight from zero to one', default: value, min: 0, max: 1,
  overridden: false, calibrated: false });

it('shows the two pick settings in League with the ruled label, defaults, range and normal command', () => {
  const dispatch = vi.spyOn(commands, 'canDispatch').mockReturnValue(true);
  vi.spyOn(dataState, 'use').mockReturnValue({ pending: false, error: '', busy: '', result: '', outcomes: {},
    params: { convertValues: (v: unknown) => v, ok: true, error: '', params: [
      param('power.pick_weight', 'How much a future draft pick counts', .6),
      param('pickvalue.discount', 'Weight of each later draft', .75),
      param('dynasty.discount', 'Do not show player knobs', .75),
    ] } });
  const html = renderToStaticMarkup(createElement(PickSettings));
  expect(html).toContain('How much a future draft pick counts');
  expect(html).toContain('default 0.6');
  expect(html).toContain('default 0.75');
  expect(html).toContain('min="0" max="1"');
  expect(html).not.toContain('Do not show player knobs');
  expect(html).toContain('They do not count in this season');
  expect(dispatch).toHaveBeenCalledWith('data.parameter');
});

it('keeps a failed parameter read visible instead of inventing defaults', () => {
  vi.spyOn(dataState, 'use').mockReturnValue({ pending: false, error: 'Settings unavailable', busy: '',
    result: '', outcomes: {} });
  const html = renderToStaticMarkup(createElement(PickSettings));
  expect(html).toContain('role="alert"');
  expect(html).toContain('Settings unavailable');
  expect(html).not.toContain('type="number"');
});
