import { useEffect, useRef, useState } from 'react';
import type { HostedPort } from '../commissioner/hostedRoom';
import { createHomeLayoutStore, defaultLayout, parseLayout, type Layout } from './homeLayout';
import type { SettingsStorage } from './franchise';

export type LayoutSyncStatus = 'local' | 'loading' | 'saving' | 'synced' | 'offline';

export function useHomeLayout(port: HostedPort, key: string, storage: SettingsStorage | undefined,
  enabled: boolean, report: (message: string) => void) {
  const [layout, setLayout] = useState<Layout>(() => enabled
    ? defaultLayout() : parseLayout(storage?.getItem(key)));
  const [status, setStatus] = useState<LayoutSyncStatus>(enabled ? 'loading' : 'local');
  const store = useRef<ReturnType<typeof createHomeLayoutStore> | null>(null);
  const saveSequence = useRef(0);

  useEffect(() => {
    if (!enabled) {
      saveSequence.current++;
      store.current = null;
      setStatus('local');
      return;
    }
    let live = true;
    const remote = createHomeLayoutStore(port);
    store.current = remote;
    setStatus('loading');
    void remote.read().then(result => {
      if (!live) return;
      setLayout(result.layout);
      setStatus(result.offline ? 'offline' : 'synced');
    });
    return () => {
      live = false;
      saveSequence.current++;
      if (store.current === remote) store.current = null;
    };
  }, [enabled, port]);

  const save = (next: Layout) => {
    setLayout(next);
    report('');
    if (!enabled) {
      setStatus('local');
      try { storage?.setItem(key, JSON.stringify(next)); } catch (cause) {
        report(`Layout not saved · ${String(cause)}`);
      }
      return;
    }
    const remote = store.current;
    if (!remote) {
      setStatus('loading');
      report('Layout not saved · account preference is still loading');
      return;
    }
    const sequence = ++saveSequence.current;
    setStatus('saving');
    void remote.write(next).then(result => {
      if (sequence !== saveSequence.current) return;
      setLayout(result.layout);
      setStatus(result.offline ? 'offline' : 'synced');
      if (result.conflict) report('Layout changed on another device · latest copy loaded');
      else if (result.offline) report('Server unavailable · layout change could not be saved');
    }).catch(cause => {
      if (sequence !== saveSequence.current) return;
      setStatus('offline');
      report(`Layout not saved · ${String(cause)}`);
    });
  };

  return { layout, status, save };
}
