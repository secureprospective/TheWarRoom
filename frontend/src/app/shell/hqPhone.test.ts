import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import postcss, { type AtRule } from 'postcss';

describe('HQ phone layout at 390px', () => {
  it('uses a single-column embedded board and reachable, touch-sized lineup actions', () => {
    const css = readFileSync(new URL('./hq.css', import.meta.url), 'utf8');
    const root = postcss.parse(css);
    const phone = root.nodes.find(node => node.type === 'atrule' &&
      node.name === 'media' && node.params === '(max-width:699px)');
    expect(phone, 'mobile-only HQ rules must cover a 390px viewport').toBeDefined();
    const rules = (phone as AtRule).nodes?.filter(node => node.type === 'rule')
      .map(node => node.toString()).join('\n') ?? '';
    expect(rules).toContain('.hq-v10.shell.embedded');
    expect(rules).toContain('min-width:0');
    expect(rules).toContain('flex-direction:column');
    expect(rules).toContain('max-height:40vh');
    expect(rules).toContain('min-height:48px');
    expect(rules).toContain('env(safe-area-inset-bottom)');
    expect(rules).not.toContain('min-width:1040px');
  });
});
