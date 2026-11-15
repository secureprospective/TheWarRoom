import { type ReactNode, type CSSProperties, type AriaRole } from 'react';
import { commands, dispatch, type CommandId, type CommandArgs } from './registry';
type Variant = 'rail' | 'tab' | 'density' | 'icon' | 'text' | 'button' | 'card' | 'command' | 'surface';
type ActProps<K extends CommandId> = {
  verb: K;
  args: CommandArgs<K>;
  variant: Variant;
  children?: ReactNode;
  active?: boolean;
  current?: boolean | 'page';
  label?: string;
  expanded?: boolean;
  controls?: string;
  disabled?: boolean;
  submit?: boolean;
  densityKey?: string;
  className?: string;
  style?: CSSProperties;
  role?: AriaRole;
  selected?: boolean;
  attributes?: Record<`data-${string}`, string | undefined>;
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
  densityKey, className, style, role, selected, attributes,
}: ActProps<K> & { refusal?: string }) {
  disabled = disabled || Boolean(refusal) || !commands.canDispatch(verb);
  const accessibleLabel = refusal ?? label ?? commands.registry[verb].label;
  const content = refusal ?? children ?? accessibleLabel;
  const title = (disabled && variant !== 'command' && variant !== 'rail') ||
    (variant === 'icon' && typeof children !== 'string')
    ? accessibleLabel : undefined;
  return (
    <button
      {...attributes}
      style={style} role={role} aria-selected={selected}
      type={submit ? 'submit' : 'button'}
      disabled={disabled}
      className={`act act-${variant}${active ? ' on' : ''}${className ? ` ${className}` : ''}`}
      data-d={densityKey}
      aria-label={accessibleLabel}
      aria-pressed={active}
      aria-current={current ? (current === true ? 'true' : current) : undefined}
      aria-expanded={expanded}
      aria-controls={controls}
      title={title}
      onClick={submit ? undefined : () => dispatch(verb, args)}
    >
      {content}
    </button>
  );
}
