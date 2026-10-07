import type { ReactNode } from 'react';
import { commands, dispatch, type CommandId, type CommandArgs } from './registry';
type Variant = 'rail' | 'tab' | 'density' | 'icon' | 'text' | 'command';
type ActProps<K extends CommandId> = {
  verb: K;
  args: CommandArgs<K>;
  variant?: Variant;
  children?: ReactNode;
  active?: boolean;
  label?: string;
  expanded?: boolean;
  controls?: string;
  disabled?: boolean;
  densityKey?: string;
};
export function Act<K extends CommandId>({
  verb,
  args,
  variant = 'text',
  children,
  active,
  label,
  expanded,
  controls,
  disabled,
  densityKey,
}: ActProps<K>) {
  return (
    <button
      type="button"
      disabled={disabled}
      className={`act act-${variant}${active ? ' on' : ''}`}
      data-d={densityKey}
      aria-label={label ?? commands.registry[verb].label}
      aria-pressed={active}
      aria-expanded={expanded}
      aria-controls={controls}
      title={label ?? commands.registry[verb].label}
      onClick={() => dispatch(verb, args)}
    >
      {children ?? label ?? commands.registry[verb].label}
    </button>
  );
}
