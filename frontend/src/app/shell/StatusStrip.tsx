import { useEffect, useState } from 'react';
import { selectProvider } from '../data/provider';
import { object, text, choice } from '../data/parseValues';
import { commands } from '../commands/registry';
import { SignalChip } from '../look/Slots';
import { Act } from '../commands/Act';
import { readShellStatus } from '../control/api';

type Status = { health: string; state: 'live' | 'stale' | 'fail'; gameday: string };
export function parseShellStatus(v: unknown): Status {
  const r = object(v, 'Status strip', ['health', 'state', 'gameday']);
  return { health: text(r.health, 'Health'), state: choice(r.state, 'Health', ['live', 'stale', 'fail']),
    gameday: text(r.gameday, 'Gameday') };
}
export default function StatusStrip() {
  const [status, setStatus] = useState<Status>();
  const [error, setError] = useState('');
  useEffect(() => {
    let active = true;
    let request = 0;
    const read = async () => {
      const revision = ++request;
      try {
        if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
          throw new Error('Source health needs the desktop app');
        const result = parseShellStatus(await readShellStatus());
        if (active && revision === request) { setStatus(result); setError(''); }
      } catch (cause) { if (active && revision === request) setError(String(cause)); }
    };
    void read();
    const stop = commands.alerts.onSeasonChange(read);
    const stopClock = selectProvider().onClockChange(read);
    return () => { active = false; stop(); stopClock(); };
  }, []);
  return <div className="shell-status" role="status">
    <span aria-label={error || status?.health || 'Source health loading'}>
      <SignalChip signal={error || status?.state === 'fail' ? 'red'
        : status?.state === 'stale' ? 'amber' : 'blue'}
        label={error ? 'Source health unavailable' : status?.health ?? 'Source health loading…'} />
    </span>
    <Act variant="text" verb="nav.open" args={{ node: 'pulse', workspace: 'now' }}>
      {status?.gameday ?? 'Gameday unavailable'}</Act>
  </div>;
}
