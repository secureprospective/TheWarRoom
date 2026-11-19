import { readFileSync } from 'node:fs';
import { expect, it } from 'vitest';

const styles = readFileSync(new URL('../control/room/controlRoom.css', import.meta.url), 'utf8');
const phoneRules = styles.match(/@media \(max-width:699px\)\{([\s\S]*?)\n\}/)?.[1] ?? '';

it('gives the hosted Control Room a contained, touch-sized phone layout below 700px', () => {
  expect(phoneRules).not.toBe('');
  expect(phoneRules).toContain('.crx.hosted-control-room{display:block;width:100%;height:100%;');
  expect(phoneRules).toContain('min-height:0;margin:0;border:0;overflow:hidden}');
  expect(phoneRules).toContain('.hosted-control-content{min-height:0;padding:10px;');
  expect(phoneRules).toContain('overflow:auto;overscroll-behavior:contain}');
  expect(phoneRules).toContain('.crx.hosted-control-room .ws-tabs button{flex:none;min-height:48px;');
  expect(phoneRules).toContain('.crx .hosted-job-controls select{min-height:44px;');
  expect(phoneRules).toContain('.crx .cr-job{grid-template-columns:8px minmax(0,1fr);');
  expect(phoneRules).toContain('.crx .cr-chip,.crx .cr-sit button');
  expect(phoneRules).toContain('.hosted-situation-steps .cr-btn{min-height:44px}');
  expect(phoneRules).toContain('min-width:0');
});
