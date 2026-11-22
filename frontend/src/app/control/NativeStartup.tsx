import { lazy, Suspense, useEffect, useState } from 'react';
import { nativeAction, nativeStartup, subscribeNativeUpdate, type NativeView } from './nativeApi';
import { isWebTransport } from './transport';
import { suspendHostedReads } from '../data/hostedReads';
import './pwa.css';

const UpdateRequired = lazy(() => import('./UpdateRequired'));
const NativeAccount = lazy(() => import('./NativeAccount'));
const LocalApp = lazy(() => import('../../TargetMount').then(m => ({ default: m.TargetMount })));

// Do not mount the local shell or its refresh/binding hooks in a hosted window.
export function NativeStartup() {
  const browser = isWebTransport();
  const [view, setView] = useState<NativeView | null | undefined>();
  const [error, setError] = useState('');
  const [online, setOnline] = useState(() => !browser || navigator.onLine);
  useEffect(() => {
    let active = true;
    const unsubscribe = subscribeNativeUpdate(setView);
    const offline = () => {
      if (!active || !browser) return;
      setOnline(false);
      suspendHostedReads();
    };
    const onlineAgain = () => {
      if (!active || !browser) return;
      setOnline(true);
      void nativeAction('status').then(setView).catch(() => {});
      void navigator.serviceWorker?.register('/app/sw.js', { scope: '/app/' }).catch(() => {});
    };
    if (browser) {
      window.addEventListener('offline', offline);
      window.addEventListener('twr:offline', offline);
      window.addEventListener('online', onlineAgain);
    }
    if (browser && navigator.onLine) {
      void navigator.serviceWorker?.register('/app/sw.js', { scope: '/app/' }).catch(() => {});
    }
    nativeStartup().then(value => { if (active) setView(value); }).catch(() => {
      if (!active) return;
      setError(browser ? 'Your league server could not be reached. Refresh to try again.'
        : 'Desktop startup could not be read. Close and reopen the app.');
    });
    return () => {
      active = false;
      unsubscribe();
      if (browser) {
        window.removeEventListener('offline', offline);
        window.removeEventListener('twr:offline', offline);
        window.removeEventListener('online', onlineAgain);
      }
    };
  }, [browser]);
  if (browser && !online) return <main className="pwa-offline"><p role="status">You're offline</p></main>;
  if (error) return <p role="alert">{error}</p>;
  if (view === undefined) return <p role="status">Opening The War Room…</p>;
  return <Suspense fallback={<p role="status">Opening your league…</p>}>
    {view?.update ? <UpdateRequired initial={view} onRetry={setView} />
      : view ? <NativeAccount initial={view} /> : <LocalApp />}
  </Suspense>;
}
