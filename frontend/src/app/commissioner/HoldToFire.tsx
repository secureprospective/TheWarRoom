import { useEffect, useId, useRef, useState } from 'react';
import { Act } from '../commands/Act';
import { commands } from '../commands/registry';
import { createHold, holdMs } from './hold';

type Props = { label: string; body: string; disabled?: boolean; ms?: number; answer?: string;
  onConfirm?(): void }; 
const holdKey = (key: string) => key === ' ' || key === 'Enter';

// HoldToFire confirms a change that needs a deliberate act: hold the pointer, Space or Enter for about
// 600 ms. Releasing early cancels. Held long enough it sends `body` through commissioner.execute, once.
// It is self-contained so chunk 4 can move it to the shared shell.
export function HoldToFire({ label, body, disabled = false, ms = holdMs, answer = '', onConfirm }: Props) {
  const latest = useRef(body);
  latest.current = body;
  const confirm = useRef(onConfirm);
  confirm.current = onConfirm;
  const help = useId();
  const [said, setSaid] = useState('');
  const [holding, setHolding] = useState(false);
  const hold = useRef<ReturnType<typeof createHold>>();
  if (!hold.current) {
    hold.current = createHold(() => {
      setHolding(false);
      setSaid('Confirmed. Sending the change.');
      if (confirm.current) confirm.current();
      else commands.dispatch('commissioner.execute', { body: latest.current });
    }, ms);
  }
  const start = () => {
    if (disabled) return;
    hold.current?.start();
    if (hold.current?.phase() !== 'holding') return;
    setHolding(true);
    setSaid('Holding. Keep holding to confirm.');
  };
  const release = () => {
    setHolding(false);
    if (hold.current?.release() === 'early') setSaid('Released early. Nothing was sent.');
  };
  useEffect(() => { if (disabled) release(); }, [disabled]);
  useEffect(() => {
    hold.current?.release();
    setHolding(false);
    setSaid('');
  }, [body, answer]);
  useEffect(() => () => { hold.current?.release(); }, []);
  // The button is an Act that never fires on a click; the wrapper times the hold from the events
  // it bubbles.
  return (
    <span className={`hold-to-fire${holding ? ' holding' : ''}`}
      onPointerDown={e => { if (e.button === 0) start(); }}
      onPointerUp={release} onPointerLeave={release} onPointerCancel={release} onBlur={release}
      onKeyDown={e => { if (holdKey(e.key)) { e.preventDefault(); start(); } }}
      onKeyUp={e => { if (holdKey(e.key)) release(); }}>
      <Act verb="commissioner.execute" args={{ body }} variant="button" label={label}
        disabled={disabled} submit>
        <span className="hold-fill" aria-hidden="true"
          style={{ transitionDuration: holding ? `${ms}ms` : '120ms' }} />
        <span className="hold-label">{label}</span>
      </Act>
      <span id={help} className="hold-help">Press and hold to confirm</span>
      <span role="status" aria-live="polite" className="hold-said">{said}</span>
    </span>
  );
}
