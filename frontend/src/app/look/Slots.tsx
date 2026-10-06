import { signalClasses, urgencyClasses } from './channels';
import type { CountdownSlot, LabelledSignal } from './channels';

export function SignalChip({ signal, label, detail }: LabelledSignal) {
  return <span className={`channel-chip ${signalClasses[signal]}`} title={detail}>{label}</span>;
}

export function Countdown({ urgency, label }: CountdownSlot) {
  return <span className={`channel-countdown ${urgencyClasses[urgency]}`}>{label}</span>;
}
