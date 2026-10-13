import { useEffect } from 'react';
import { Glyph } from '../look/Glyph';
import { Act } from '../commands/Act';
import { personalState } from './state';

export function WatchButton({ id, name, mine }: { id: string; name: string; mine: string }) {
  const s = personalState.useReading(mine);
  useEffect(() => { if (mine) void personalState.ensure(mine); }, [mine]);
  const watched = s.value?.entries.some(e => e.kind === 'watch' && e.id === id);
  return <Act variant="button" verb="watch.add" disabled={!mine || s.pending}
    args={{ id, entity: 'player', action: watched ? 'remove' : 'add' }}
    label={s.error || `${watched ? 'Unwatch' : 'Watch'} ${name}`}>
    {watched && <Glyph name="watch" />} {watched ? 'Unwatch' : 'Watch'}</Act>;
}
