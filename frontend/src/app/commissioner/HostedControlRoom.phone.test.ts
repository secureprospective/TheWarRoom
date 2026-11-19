import { readFileSync } from 'node:fs';
import { expect, it } from 'vitest';

const styles = readFileSync(new URL('../control/room/controlRoom.css', import.meta.url), 'utf8');
const phoneRules = styles.match(/@media \(max-width:699px\)\{([\s\S]*?)\n\}/)?.[1] ?? '';

it('gives Control Room jobs a touch-sized phone layout that scrolls with the page', () => {
  expect(phoneRules).not.toBe('');
  // The phone shell content area is the one scroller; the Control Room must not trap it.
  expect(phoneRules).toContain('.crx.hosted-control-room{width:100%;height:auto;');
  expect(phoneRules).not.toContain('overflow:hidden}');
  expect(phoneRules).toContain('.crx .hosted-job-controls select{min-height:44px;');
  expect(phoneRules).toContain('.crx .cr-job{grid-template-columns:8px minmax(0,1fr);');
  expect(phoneRules).toContain('.crx .cr-chip,.crx .cr-sit button');
  expect(phoneRules).toContain('.hosted-situation-steps .cr-btn{min-height:44px}');
  expect(phoneRules).toContain('min-width:0');
});
