import type { ReactNode } from 'react';
import type { CountdownSlot, Gravity, LabelledSignal } from '../look/channels';
import { gravityClasses } from '../look/channels';
import { Countdown, SignalChip } from '../look/Slots';

export const DENSITIES = ['narrative', 'tactical', 'matrix'] as const;
export type Density = (typeof DENSITIES)[number];
export type VerdictSlot = LabelledSignal & { beyondChance: true };
export type CardProps = {
  gravity: Gravity;
  header: ReactNode;
  primaryZone?: ReactNode;
  children?: ReactNode;
  actionTray?: ReactNode;
  status?: readonly LabelledSignal[];
  verdict?: VerdictSlot;
  countdown?: CountdownSlot;
  provenance?: LabelledSignal;
};

export function Card({
  gravity,
  header,
  primaryZone,
  children,
  actionTray,
  status,
  verdict,
  countdown,
  provenance,
}: CardProps) {
  return (
    <article className={`twr-card ${gravityClasses[gravity]}`}>
      <header className="card-header">
        {header}
        {provenance && (
          <span className="card-provenance">
            <SignalChip {...provenance} />
          </span>
        )}
      </header>
      <div className="card-body">
        {primaryZone}
        {status && status.length > 0 && (
          <div className="card-status">
            {status.map((chip) => (
              <SignalChip key={chip.label} {...chip} />
            ))}
          </div>
        )}
        {children}
        {verdict && (
          <div>
            <SignalChip signal={verdict.signal} label={verdict.label} />
          </div>
        )}
        {countdown && (
          <div>
            <Countdown {...countdown} />
          </div>
        )}
      </div>
      <div className="card-action-tray">{actionTray}</div>
    </article>
  );
}
