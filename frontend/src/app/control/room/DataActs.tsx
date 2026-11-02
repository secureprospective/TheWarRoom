import { useRoomSurface } from './useRoomSurface';
import { Act } from '../../commands/Act';
import { useEffect, useState } from 'react';
import { sourcesAPI } from '../../datasources/api';
import { dataScopes } from './dataScopes';

type Scope = { id: string; label: string; note: string };

/** Refresh a data source or clear a cache: the Data and sources commands (TargetDataRefresh,
 *  TargetCacheClear), each on a scope Go lists. */
export function DataActs({ mode, api = sourcesAPI }: { mode: 'refresh' | 'clear'; api?: typeof sourcesAPI }) {
  const [scopes, setScopes] = useState<Scope[]>();
  const [scope, setScope] = useState('');
  const [said, setSaid] = useState('');
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    let active = true;
    dataScopes(mode, api).then(
      (s) => {
        if (active) {
          setScopes(s);
          setScope(s[0]?.id ?? '');
        }
      },
      (e) => {
        if (active) setSaid(e instanceof Error ? e.message : String(e));
      },
    );
    return () => {
      active = false;
    };
  }, [mode, api]);
  const go = async () => {
    setBusy(true);
    try {
      const r = mode === 'clear' ? await api.cacheClear(scope) : await api.refresh(scope);
      setSaid(r.ok ? r.summary : r.error);
    } catch (e) {
      setSaid(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };
  useRoomSurface('room-data', () => go());
  return (
    <section className="league-form" aria-label={mode === 'clear' ? 'Clear a cache' : 'Refresh now'}>
      <label>
        {mode === 'clear' ? 'Cache' : 'Source'}
        <select value={scope} onChange={(e) => setScope(e.target.value)}>
          {(scopes ?? []).map((s) => (
            <option key={s.id} value={s.id}>
              {s.label}
            </option>
          ))}
        </select>
      </label>
      {scopes?.find((s) => s.id === scope)?.note && <p>{scopes.find((s) => s.id === scope)!.note}</p>}
      <Act
        className="cr-btn pri"
        disabled={!scope || busy}
        verb="commissioner.surface"
        variant="surface"
        label={mode === 'clear' ? 'CLEAR' : 'REFRESH'}
        args={{ screen: 'room-data', action: 'check', value: '' }}
      >
        {mode === 'clear' ? 'CLEAR' : 'REFRESH'}
      </Act>
      {said && <p role="status">{said}</p>}
    </section>
  );
}
