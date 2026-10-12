import { useEffect, useState } from 'react';
import { parsePlayer, type Player } from './contract';
import type * as Bindings from '../../../wailsjs/go/main/App';

export default function ResearchPlayerInspector({ id }: { id: string }) {
  const [player, setPlayer] = useState<Player>();
  const [error, setError] = useState('');
  useEffect(() => {
    let disposed = false;
    setPlayer(undefined);
    setError('');
    const app = (window as unknown as { go?: { main: { App: typeof Bindings } } }).go?.main.App;
    if (!app) { setError('Player research requires the desktop app.'); return; }
    void app.TargetResearchPlayer(id).then(value => {
      if (!disposed) setPlayer(parsePlayer(value, 'Research player'));
    }).catch(cause => { if (!disposed) setError(String(cause)); });
    return () => { disposed = true; };
  }, [id]);
  return <section aria-label="Research player Inspector">
    {error && <p role="alert">{error}</p>}
    {!player && !error && <p role="status">Reading player identity…</p>}
    {player && <>
      <h4>{player.name}</h4>
      <p>{player.position} · {player.team}</p>
      <p>Source: held MFL player directory</p>
      <p>Open this player from your roster for contracts, values and roster Acts.</p>
    </>}
  </section>;
}
