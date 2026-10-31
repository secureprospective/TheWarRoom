import { expect, it } from 'vitest';
import { renderToStaticMarkup } from 'react-dom/server';
import { HQShell, lockLabel } from './HQShell';
it('renders the actual league name, live count and a settings-clock lock time', () => {
  const at = new Date(2026, 10, 1, 12, 0).toISOString();
  expect(lockLabel(at)).toBe('Locks Sun 12:00');
  const html = renderToStaticMarkup(<HQShell tab="lineup-and-roster" leagueName="Held league"
    week={9} liveCount={3} lock={{ at }} navigate={() => {}} navigateNode={() => {}}
    inspector={null}>{null}</HQShell>);
  expect(html).toContain('Held league'); expect(html).toContain('3 live');
  expect(html).toContain('Locks Sun 12:00');
});
