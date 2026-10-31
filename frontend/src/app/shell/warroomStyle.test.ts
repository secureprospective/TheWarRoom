import { expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import postcss from 'postcss';
it('retains every locked mock CSS rule and declaration under a War Room-only scope', () => {
  const mock = readFileSync('../docs/ui/mocks/warroom-mock-v6.html', 'utf8')
    .split('<style>')[1].split('</style>')[0];
  const actual = postcss.parse(readFileSync('src/app/shell/warroom.css', 'utf8'));
  const expected = postcss.parse(mock);
  const declarations = (rule: postcss.Rule) => rule.nodes.filter(n => n.type === 'decl')
    .map(n => n.toString().replace(/warroom-/g, ''));
  const rules: postcss.Rule[] = [];
  actual.walkRules(r => { rules.push(r); });
  expected.walkRules(rule => {
    const keyframe = rule.parent?.type === 'atrule' && rule.parent.name === 'keyframes';
    const selector = keyframe ? rule.selector : rule.selector.split(',').map(s => {
      s = s.trim();
      if ([':root', 'html', 'body', 'body.room'].includes(s)) return '.wr-v6';
      return s.startsWith('body.') ? s.replace('body.', '.wr-v6.') : `.wr-v6 ${s}`;
    }).join(',');
    expect(rules.some(r => r.selector === selector && JSON.stringify(declarations(r)) ===
      JSON.stringify(declarations(rule))), selector).toBe(true);
  });
});
