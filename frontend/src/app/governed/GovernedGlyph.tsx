import { useEffect, useId, useState } from 'react';
import { Act } from '../commands/Act';
import { localTime } from '../clock/localTime';
import { surfaceActions } from '../commissioner/surfaceActions';
import type { Governed } from './governed';
import './governed.css';

// GovernedGlyph marks a value the commissioner set. Pressing it opens who set it, when (in the viewer's own
// time) and with which recorded change; pressing it again closes that. `subject` names the value for a
// screen reader.
export function GovernedGlyph({ governed, subject }: { governed: Governed; subject: string }) {
  const id = useId();
  const screen = `governed.${id}`;
  const [open, setOpen] = useState(false);
  useEffect(() => surfaceActions.register(screen, args => {
    if (args.action === 'view') setOpen(was => !was);
  }), [screen]);
  return (
    <span className="governed">
      <Act verb="commissioner.surface" variant="icon" args={{ screen, action: 'view', value: '' }}
        expanded={open} controls={`${id}-detail`} label={`Who set ${subject}, when, and with which change`}>
        ◆
      </Act>
      {open && (
        <dl id={`${id}-detail`} className="governed-detail">
          <dt>Set by</dt><dd>{governed.who}</dd>
          <dt>When</dt>
          <dd><time dateTime={governed.when}>{localTime(governed.when, { year: 'numeric' })}</time></dd>
          <dt>With this change</dt><dd>{governed.change}</dd>
        </dl>
      )}
    </span>
  );
}
