import { useEffect, useState } from 'react';
import { hostedWorkflow } from '../commissioner/api';
import { nativeStartup } from '../control/nativeApi';
import { LiveWall } from './LiveWall';
import { fakeWallPort, wallGame } from './fixtures';

/** WallMount is the /app/wall entry: it reads the signed-in league, then shows the wall. `?demo`
 *  (dev server only) shows six fixture games instead of the server. */
export default function WallMount() {
  const [league, setLeague] = useState('');
  const [error, setError] = useState('');
  const demo = import.meta.env.DEV && new URLSearchParams(window.location.search).has('demo');
  useEffect(() => {
    if (demo) return;
    nativeStartup().then(v => v?.league ? setLeague(v.league) : setError('Sign in to the league first.'))
      .catch(() => setError('The league server could not be reached.'));
  }, [demo]);
  if (demo) return <LiveWall league="league" subscribe={() => () => {}}
    port={fakeWallPort([0, 1, 2, 3, 4, 5].map(i => wallGame(i)))} />;
  if (error) return <p role="alert">{error}</p>;
  if (!league) return <p role="status">Opening the wall…</p>;
  return <LiveWall league={league} port={hostedWorkflow} />;
}
