import { useEffect, useState } from 'react';
import { kickoffText } from '../now/LiveGame';
import {
  parseHostedGame, parseHostedGames, parseHostedPregame, type Game, type LiveView,
} from '../now/liveContract';
import type { usePulseLive } from './hqLive';
import type { PulseScope } from './pulseReads';

/** useNow is the current time, re-read every `ms` (one interval, cleared on unmount), for text that ages. */
export function useNow(ms: number) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), ms);
    return () => clearInterval(id);
  }, [ms]);
  return now;
}

/** followKey is how useHostedGames names a game to follow: away-home abbreviations. */
export function followKey(g: Game) { return `${g.away.abbreviation}-${g.home.abbreviation}`; }
/** clockWords is a game's short state: kickoff time, quarter and clock, or FINAL. */
export function clockWords(g: Game) {
  if (g.status === 'post') return 'FINAL';
  if (g.status === 'pre') return kickoffText(g.kickoff);
  return `${g.period > 4 ? 'OT' : `Q${g.period}`} ${g.clock}`;
}

export function useHostedGames(league: string, port: NonNullable<PulseScope['port']>,
  onChange: (load: () => void) => () => void):
  ReturnType<typeof usePulseLive> & { prefer: (key: string) => void; heldAt?: number } {
  const [held, setHeld] = useState<{ view?: LiveView; error: string; at?: number }>({ error: '' });
  const [chosen, choose] = useState('');
  const [preferred, prefer] = useState('');
  useEffect(() => {
    let live = true;
    let generation = 0;
    const load = async () => {
      const request = ++generation;
      const current = () => live && request === generation;
      try {
        const reply = await port({ command: 'nfl.games.read', mode: 'read', arguments: { league } });
        const view = parseHostedGames(reply.data, league);
        const key = chosen || preferred;
        const selected = view.games.find(g => `${g.away.abbreviation}-${g.home.abbreviation}` === key)
          ?? view.games.find(g => g.status === 'in');
        if (current()) setHeld(prior => ({ view: { ...view,
          followed: prior.view?.followed?.id === selected?.id ? prior.view?.followed ?? null : null },
          error: '', at: Date.now() }));
        if (!selected) return;
        const detail = await port({ command: 'nfl.game.read', mode: 'read',
          arguments: { league, game: selected.id } });
        let followed = parseHostedGame(detail.data, league, selected.id)
          ?? (selected.status === 'pre' ? selected : null);
        if (followed && followed.status === 'pre') {
          try {
            const pre = await port({ command: 'nfl.game.pregame.read', mode: 'read',
              arguments: { league, game: selected.id } });
            followed = { ...followed, ...parseHostedPregame(pre.data, league, selected.id) };
          } catch (error: unknown) {
            // Older servers or unavailable pre-game cache still render the preview without these fields.
            console.debug('Pre-game read failed:', error);
          }
        }
        view.followed = followed;
        if (current()) setHeld({ view, error: '', at: Date.now() });
      } catch (e: unknown) {
        const error = e instanceof Error ? e.message : String(e);
        if (current()) setHeld(prior => ({ view: prior.view, error, at: prior.at }));
      }
    };
    void load();
    const off = onChange(() => { void load(); });
    return () => { live = false; off(); };
  }, [league, port, onChange, chosen, preferred]);
  // heldAt: when the last held board read resolved (the view's asOf is only a display string).
  return { view: held.view, error: held.error, heldAt: held.at, follow: choose, prefer };
}
