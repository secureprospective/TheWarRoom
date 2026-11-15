// @vitest-environment jsdom
import { act, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { expect, it, vi } from 'vitest';
import SideTray from './SideTray';

it('closes once on document Escape after focus moves to the body, and restores focus', async () => {
  Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
  const host = document.createElement('div');
  document.body.append(host);
  const root = createRoot(host);
  const button = document.createElement('button');
  document.body.append(button);
  const onClose = vi.fn();
  function Example() {
    const [open, setOpen] = useState(true);
    return open ? <SideTray title="Change" screen="test" opener={button} onClose={() => {
      onClose(); setOpen(false);
    }}><p>In force now</p></SideTray> : null;
  }
  try {
    button.focus();
    await act(async () => root.render(<Example />));
    expect(host.querySelector('[role="dialog"]')).not.toBeNull();
    await act(async () => (document.activeElement as HTMLElement).blur());
    expect(document.activeElement).toBe(document.body);
    await act(async () => document.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'Escape', bubbles: true, cancelable: true,
    })));
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(host.querySelector('[role="dialog"]')).toBeNull();
    expect(document.activeElement).toBe(button);
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
    expect(onClose).toHaveBeenCalledTimes(1);
  } finally {
    await act(async () => root.unmount());
    host.remove(); button.remove();
  }
});
