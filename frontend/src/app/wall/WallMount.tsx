import { useEffect, useState } from 'react';
import { hostedWorkflow } from '../commissioner/api';
import { nativeStartup } from '../control/nativeApi';
import { LiveWall } from './LiveWall';
import { createDemoFeed } from './fixtures';

/** WallMount is the /app/wall entry: it reads the signed-in league, then shows the wall. `?demo`
 *  (dev server only) shows a live 12-game fixture slate with scoring plays arriving every ~20 s. */
export default function WallMount() {
  const [league, setLeague] = useState('');
  const [error, setError] = useState('');
  const demo = import.meta.env.DEV && new URLSearchParams(window.location.search).has('demo');
  const [feed] = useState(() => (demo ? createDemoFeed() : null));
  useEffect(() => {
    if (demo) return;
    nativeStartup().then(v => v?.league ? setLeague(v.league) : setError('Sign in to the league first.'))
      .catch(() => setError('The league server could not be reached.'));
  }, [demo]);
  if (demo && feed) {
    return <LiveWall league="league" subscribe={feed.subscribe} port={feed.port} />;
  }
  if (error) return <p role="alert">{error}</p>;
  if (!league) return <p role="status">Opening the wall…</p>;
  return <LiveWall league={league} port={hostedWorkflow} />;
}
