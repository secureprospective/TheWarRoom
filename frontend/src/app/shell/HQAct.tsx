import { useEffect, useId, useRef, type ReactNode, type CSSProperties, type AriaRole } from 'react';
import { Act } from '../commands/Act';
import { surfaceActions } from '../commissioner/surfaceActions';

// Local presentation actions use the existing command door, without putting callbacks into its args.
export function HQAct({ onAction, children, className, style, role, selected, expanded, active,
  disabled, title, label, submit, attributes }: {
  onAction?: () => void | Promise<unknown>; children?: ReactNode; className?: string;
  style?: CSSProperties; role?: AriaRole; selected?: boolean; expanded?: boolean; active?: boolean;
  disabled?: boolean; title?: string; label?: string; submit?: boolean;
  attributes?: Record<`data-${string}`, string | undefined>;
}) {
  const screen = useId();
  const current = useRef(onAction); current.current = onAction;
  useEffect(() => surfaceActions.register(screen, async () => { await current.current?.(); }), [screen]);
  return <Act verb="commissioner.surface" variant="button" args={{ screen, action: 'view', value: '' }}
    {...{ className, style, role, selected, expanded, active, disabled, submit, attributes }}
    label={label ?? (typeof children === 'string' ? children : title ?? 'HQ action')}>
    {children}
  </Act>;
}
