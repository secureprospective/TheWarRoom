import { lazy, Suspense } from 'react';
import { commands } from './app/commands/registry';
import { Act } from './app/commands/Act';
import { TargetApp } from './app/shell/TargetApp';
import { useShellBindings } from './app/shell/useShellBindings';
const App = lazy(() => import('./App'));
export function TargetMount() {
  const { harness } = commands.use();
  useShellBindings();
  return harness ? (
    <Suspense fallback={<p>Opening harness…</p>}>
      <App />
      <div className="twr-app harness-return">
        <Act verb="target.open" args={{}}>Target UI</Act>
      </div>
    </Suspense>
  ) : <TargetApp />;
}
