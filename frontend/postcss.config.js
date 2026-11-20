// Desktop v3 isolation. The v3 frame (#app, app/v3/v3.css) shares 181 class names with the older
// stylesheets. Every rule outside app/v3 is fenced so it never reaches the v3 frame, except inside
// .legacy-host: the slot where a not-yet-reskinned screen still renders. :where keeps each rule's
// original specificity, so legacy screens outside the frame render exactly as before.
const FENCE = ':where(:not(#app, #app *), .legacy-host, .legacy-host *)';
const ROOTISH = /^(html|body|:root|\*|::?[\w-]+)$/;

function fence(selector) {
  const s = selector.trim();
  if (!s || ROOTISH.test(s) || /^(html|body|:root)\b/.test(s) || s.includes('#app')) return s;
  const pe = s.search(/::[\w-]+(\([^)]*\))?\s*$/);
  return pe > 0 ? s.slice(0, pe) + FENCE + s.slice(pe) : s + FENCE;
}

const v3Isolation = () => ({
  postcssPlugin: 'twr-v3-isolation',
  Rule(rule) {
    const file = rule.source?.input?.file ?? '';
    if (!file || file.includes('/app/v3/') || rule.__fenced) return;
    if (rule.parent?.type === 'atrule' && /keyframes/i.test(rule.parent.name)) return;
    rule.selectors = rule.selectors.map(fence);
    rule.__fenced = true;
  },
});
v3Isolation.postcss = true;

export default {
  plugins: [
    (await import('@tailwindcss/postcss')).default(),
    v3Isolation(),
  ],
};
