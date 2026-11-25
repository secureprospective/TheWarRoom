import { lazy, Suspense, useEffect, useState } from 'react';
import { nativeAction, nativeStartup, subscribeNativeUpdate, type NativeView } from './nativeApi';
import { errorCodeFor, newClientErrorCode } from '../data/errorCodes';
import { suspendHostedReads } from '../data/hostedReads';
import { startPwaUpdates } from './pwaUpdate';
import './pwa.css';

const FailureAlert = lazy(() => import('../data/FailureAlert')
  .then(module => ({ default: module.FailureAlert })));
const UpdateRequired = lazy(() => import('./UpdateRequired'));
const NativeAccount = lazy(() => import('./NativeAccount'));

export function NativeStartup() {
  const [view, setView] = useState<NativeView | undefined>();
  const [error, setError] = useState('');
  const [errorCode, setErrorCode] = useState<string>();
  const [online, setOnline] = useState(() => navigator.onLine);
  useEffect(() => {
    let active = true;
    const unsubscribe = subscribeNativeUpdate(setView);
    const offline = () => {
      if (!active) return;
      setOnline(false);
      suspendHostedReads();
    };
    const onlineAgain = () => {
      if (!active) return;
      setOnline(true);
      void nativeAction('status').then(setView).catch(() => {});
      startPwaUpdates();
    };
    window.addEventListener('offline', offline);
    window.addEventListener('twr:offline', offline);
    window.addEventListener('online', onlineAgain);
    if (navigator.onLine) startPwaUpdates();
    nativeStartup().then(value => { if (active) setView(value); }).catch(cause => {
      if (!active) return;
      setErrorCode(errorCodeFor(cause) ?? newClientErrorCode());
      setError('Your league server could not be reached. Refresh to try again.');
    });
    return () => {
      active = false;
      unsubscribe();
      window.removeEventListener('offline', offline);
      window.removeEventListener('twr:offline', offline);
      window.removeEventListener('online', onlineAgain);
    };
  }, []);
  if (!online) return <main className="pwa-offline">
    <p role="status">You're offline</p>
    <Suspense fallback={<p role="alert">You're offline.</p>}>
      <FailureAlert
        message="You're offline. Check your internet connection, then try again."
        screen="Offline"
        route="home/overview"
        allowSend={false}
      />
    </Suspense>
  </main>;
  if (error) return <Suspense fallback={<p role="alert">{error}</p>}>
    <FailureAlert
      message={error}
      code={errorCode}
      screen="Startup"
      route="home/overview"
      allowSend={false}
    />
  </Suspense>;
  if (view === undefined) return <p role="status">Opening The War Room…</p>;
  return <Suspense fallback={<p role="status">Opening your league…</p>}>
    {view?.update ? <UpdateRequired initial={view} onRetry={setView} />
      : <NativeAccount initial={view} />}
  </Suspense>;
}
