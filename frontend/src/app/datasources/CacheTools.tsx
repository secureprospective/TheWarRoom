import { useEffect } from 'react';
import { Act } from '../commands/Act';
import { cacheState } from './cacheState';

// CacheTools reads a held copy afresh when it looks stale. Go owns the list and runs the same refresh each
// screen already has, ignoring the copy held; a source that does not answer leaves the copy as it was.
export function CacheTools() {
  const { scopes, busy, error, outcomes } = cacheState.use();
  useEffect(() => { void cacheState.load(); }, []);
  return (
    <section aria-label="Read data again" data-endpoint="M-137">
      <h4>Read data again</h4>
      <p>Use these when a screen shows data that looks out of date. Each one asks for a fresh copy and
        keeps the copy held if the source does not answer.</p>
      {error && <p role="alert">{error}</p>}
      <dl className="refresh-scopes">
        {scopes.map(scope => (
          <div key={scope.id}>
            <dt>{scope.label}</dt>
            <dd>
              <p>{scope.note}</p>
              <div className="data-action">
                <Act variant="button" verb="cache.run" args={{ scope: scope.id }} disabled={busy !== ''}>
                  {scope.action}
                </Act>
                <span role="status" aria-live="polite">
                  {busy === scope.id ? scope.progress : outcomes[scope.id]}
                </span>
              </div>
            </dd>
          </div>
        ))}
      </dl>
    </section>
  );
}
