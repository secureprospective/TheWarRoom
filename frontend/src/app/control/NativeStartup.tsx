import { lazy, Suspense, useEffect, useState } from 'react';
import { nativeStartup, type NativeView } from './nativeApi';

const UpdateRequired = lazy(() => import('./UpdateRequired'));
const NativeAccount = lazy(() => import('./NativeAccount'));
const LocalApp = lazy(() => import('../../TargetMount').then(m => ({ default: m.TargetMount })));

// Do not mount the local shell or its refresh/binding hooks in a hosted window.
export function NativeStartup() {
  const [view, setView] = useState<NativeView | null | undefined>();
  const [error, setError] = useState('');
  useEffect(() => {
    let active = true;
    const update = (event: Event) => setView((event as CustomEvent<NativeView>).detail);
    window.addEventListener('native-update', update);
    nativeStartup().then(value => { if (active) setView(value); })
      .catch(() => { if (active) setError('Desktop startup could not be read. Close and reopen the app.'); });
    return () => { active = false; window.removeEventListener('native-update', update); };
  }, []);
  if (error) return <p role="alert">{error}</p>;
  if (view === undefined) return <p role="status">Opening The War Room…</p>;
  return <Suspense fallback={<p role="status">Opening your league…</p>}>
    {view?.update ? <UpdateRequired initial={view} onRetry={setView} />
      : view ? <NativeAccount initial={view} /> : <LocalApp />}
  </Suspense>;
}
