import type { ComponentProps } from 'react';
import { commands } from '../commands/registry';
import { useLineup } from './lineups';
import LineupEditor from './LineupEditor';

export function LineupRoster(props: Omit<ComponentProps<typeof LineupEditor>, 'reading' | 'error'>) {
  const { reading: cached, error } = useLineup(commands.lineups, props.franchiseId, props.snapshot);
  const reading = error === undefined ? cached : undefined;
  return <LineupEditor {...props} reading={reading} error={error} />;
}
