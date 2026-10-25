import { lazy, Suspense } from 'react';
import { Act } from '../commands/Act';
import { presets, presetIds, presetAllowed } from './presets';

const MFLConnection = lazy(() => import('./MFLConnection'));
const WhatsNew = lazy(() => import('../control/WhatsNew').then((m) => ({ default: m.WhatsNew })));
const BackupNow = lazy(() => import('../commissioner/BackupNow').then((m) => ({ default: m.BackupNow })));

export function AppSettings() {
  return (
    <section className="app-settings">
      <Suspense fallback={<section className="mfl-connection">Checking the keyring…</section>}>
        <MFLConnection />
      </Suspense>
      <Suspense fallback={null}>
        <BackupNow />
      </Suspense>
      <h4>View presets</h4>
      {presetIds.map((preset) => (
        <div key={preset}>
          <Act variant="button" verb="preset.apply" args={{ preset }} disabled={!presetAllowed(preset)}>
            {presets[preset].label}
          </Act>
          {!presetAllowed(preset) && (
            <p className="not-wired">
              Unavailable: commissioner or admin role required · running as GM
            </p>
          )}
        </div>
      ))}
      <Suspense fallback={<p role="status">Reading MFL's site news…</p>}>
        <WhatsNew />
      </Suspense>
    </section>
  );
}
