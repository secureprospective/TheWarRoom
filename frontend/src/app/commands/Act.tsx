import { type ReactNode } from 'react';
import { commands, dispatch, type CommandId, type CommandArgs } from './registry';
type Variant = 'rail' | 'tab' | 'density' | 'icon' | 'text' | 'button' | 'card' | 'command';
type ActProps<K extends CommandId> = {
  verb: K;
  args: CommandArgs<K>;
  variant: Variant;
  children?: ReactNode;
  active?: boolean;
  current?: boolean;
  label?: string;
  expanded?: boolean;
  controls?: string;
  disabled?: boolean;
  submit?: boolean;
  densityKey?: string;
};
export function Act<K extends CommandId>(props: ActProps<K>) {
  return commands.isTeamAction(props.verb, props.args) ? <TeamAct {...props} /> : ActButton(props);
}
function TeamAct<K extends CommandId>(props: ActProps<K>) {
  const refusal = commands.useTeamActionReason(props.verb, props.args);
  return ActButton({ ...props, refusal });
}
function ActButton<K extends CommandId>({
  verb,
  args,
  refusal,
  variant,
  children,
  active,
  current,
  label,
  expanded,
  controls,
  disabled,
  submit = false,
  densityKey,
}: ActProps<K> & { refusal?: string }) {
  disabled = disabled || Boolean(refusal);
  const accessibleLabel = refusal ?? label ?? commands.registry[verb].label;
  const content = refusal ?? children ?? accessibleLabel;
  const title = (disabled && variant !== 'command' && variant !== 'rail') ||
    (variant === 'icon' && typeof children !== 'string')
    ? accessibleLabel : undefined;
  return (
    <button
      type={submit ? 'submit' : 'button'}
      disabled={disabled}
      className={`act act-${variant}${active ? ' on' : ''}`}
      data-d={densityKey}
      aria-label={accessibleLabel}
      aria-pressed={active}
      aria-current={current ? 'true' : undefined}
      aria-expanded={expanded}
      aria-controls={controls}
      title={title}
      onClick={submit ? undefined : () => dispatch(verb, args)}
    >
      {content}
    </button>
  );
}
