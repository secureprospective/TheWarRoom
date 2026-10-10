import { Children, type ReactNode } from 'react';
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
  const accessibleLabel = label ?? commands.registry[verb].label;
  const content = children ?? accessibleLabel;
  // A tooltip says why a button is disabled, names an icon or rail Act, or completes shortened text.
  // Cards and other self-describing content get none: a repeated tooltip lingers over the next card.
  const parts = Children.toArray(content);
  const shortened = parts.every((part) => typeof part === 'string') && parts.join('') !== accessibleLabel;
  const title = disabled || variant === 'icon' || variant === 'rail' || shortened
    ? accessibleLabel : undefined;
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
      title={title}
      onClick={() => dispatch(verb, args)}
    >
      {content}
    </button>
  );
}
