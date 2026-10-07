import { lazy, Suspense } from 'react';
import { Act } from '../commands/Act';
import { presets, presetIds, presetAllowed } from './presets';

const MFLConnection = lazy(() => import('./MFLConnection'));

export function AppSettings() {
  return (
    <section className="app-settings">
      <Suspense fallback={<section className="mfl-connection">Checking the keyring…</section>}>
        <MFLConnection />
      </Suspense>
      <h4>View presets</h4>
      {presetIds.map((preset) => (
        <div key={preset}>
          <Act verb="preset.apply" args={{ preset }} disabled={!presetAllowed(preset)}>
            {presets[preset].label}
          </Act>
          {!presetAllowed(preset) && (
            <p className="not-wired">
              Unavailable: commissioner or admin role required · running as GM
            </p>
          )}
        </div>
      ))}
      <Act verb="harness.open" args={{}}>
        Open harness
      </Act>
    </section>
  );
}
