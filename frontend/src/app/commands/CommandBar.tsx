import { EndpointIndex } from '../registry/EndpointIndex';
import { useState, useRef, useEffect, useMemo } from 'react';
import type { Snapshot } from '../data/contract';
import { commands } from './registry';
import { Act } from './Act';
import { searchCandidates, rankResults, runResult } from './search';
import { Glyph } from '../look/Glyph';
import { assertShortList } from '../shell/shortList';

export function CommandBar({ snapshot }: { snapshot?: Snapshot }) {
  const open = commands.use().commandbar;
  const [query, setQuery] = useState('');
  const [index, setIndex] = useState(0);
  const input = useRef<HTMLInputElement>(null);
  const candidates = useMemo(() => searchCandidates(commands, snapshot), [snapshot]);
  const results = rankResults(query, candidates);
  if (import.meta.env.DEV) assertShortList(results.length, 8);
  useEffect(() => {
    if (!open) {
      setQuery('');
      setIndex(0);
      // A closed bar is only visually hidden (so it can transition); it must not keep focus, or
      // the next keystrokes land in an invisible input and every shortcut goes dead.
      if (document.activeElement === input.current) input.current?.blur();
      return;
    }
    const previous = document.activeElement;
    input.current?.focus();
    return () => {
      if (previous instanceof HTMLElement && previous.isConnected) previous.focus();
    };
  }, [open]);
  return (
    <section className="command-bar" data-open={open} aria-hidden={!open} aria-label="Command bar">
      <input
        ref={input}
        aria-label="Search commands, places, endpoints and players"
        aria-controls="command-results"
        placeholder="Type a command, place, endpoint or player…"
        value={query}
        onChange={(event) => {
          setQuery(event.target.value);
          setIndex(0);
        }}
        onKeyDown={(event) => {
          if (event.key === 'Escape') {
            event.preventDefault();
            commands.dispatch('commandbar.close', {});
          } else if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
            event.preventDefault();
            const step = event.key === 'ArrowDown' ? 1 : -1;
            setIndex((current) =>
              results.length ? (current + step + results.length) % results.length : 0,
            );
          } else if (event.key === 'Enter' && results[index]) {
            event.preventDefault();
            runResult(commands, results[index]);
          }
        }}
      />
      <div id="command-results">
        {results.map((result, position) => (
          <Act
            key={`${result.kind}-${result.label}-${position}`}
            verb={result.invocation.verb}
            args={result.invocation.args}
            active={position === index}
            label={result.label}
          >
            <span>{result.position && <Glyph name={result.position} />} {result.label}</span>
            <small>{result.kind}</small>
          </Act>
        ))}
        {!results.length && (
          <p role="status">No matching commands, places, endpoints or rostered players.</p>
        )}
      </div>
      {(!query.trim() || 'preset: admin'.includes(query.trim().toLowerCase())) && (
        <p className="not-wired">
          Preset: Admin · unavailable: commissioner or admin role required
        </p>
      )}
      {/* The bar's own map rows only on an empty query; typed results are the point. */}
      {!query.trim() && <EndpointIndex placement="Command bar" />}
      <Act verb="commandbar.close" args={{}}>
        Close · Escape
      </Act>
    </section>
  );
}
