import { useState, useRef, useEffect } from 'react';
import type { Snapshot } from '../data/contract';
import { commands } from './registry';
import { Act } from './Act';
import { searchCandidates, rankResults, runResult } from './search';

export function CommandBar({ snapshot }: { snapshot: Snapshot }) {
  const [query, setQuery] = useState('');
  const [index, setIndex] = useState(0);
  const input = useRef<HTMLInputElement>(null);
  const results = rankResults(query, searchCandidates(commands, snapshot));
  useEffect(() => {
    const previous = document.activeElement;
    input.current?.focus();
    return () => {
      if (previous instanceof HTMLElement && previous.isConnected) previous.focus();
    };
  }, []);
  return (
    <section className="command-bar" aria-label="Command bar">
      <input
        ref={input}
        aria-label="Search commands, places and players"
        aria-controls="command-results"
        placeholder="Type a command, place or player…"
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
            <span>{result.label}</span>
            <small>{result.kind}</small>
          </Act>
        ))}
        {!results.length && (
          <p role="status">No matching commands, places or rostered players.</p>
        )}
      </div>
      {(!query.trim() || 'preset: admin'.includes(query.trim().toLowerCase())) && (
        <p className="not-wired">
          Preset: Admin · unavailable: commissioner or admin role required
        </p>
      )}
      <Act verb="commandbar.close" args={{}}>
        Close · Escape
      </Act>
    </section>
  );
}
