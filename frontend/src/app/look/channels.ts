export const GRAVITIES = ['G0', 'G1', 'G2', 'G3'] as const;
export const URGENCIES = ['U0', 'U1', 'U2', 'U3'] as const;
export const SIGNALS = ['green', 'blue', 'amber', 'red', 'grey'] as const;
export type Gravity = (typeof GRAVITIES)[number];
export type Urgency = (typeof URGENCIES)[number];
export type Signal = (typeof SIGNALS)[number];
// detail is the interrogate altitude: shown on hover/focus, never in the resting card.
export type LabelledSignal = { signal: Signal; label: string; detail?: string };
export type CountdownSlot = { urgency: Urgency; label: string };
export const gravityClasses: Record<Gravity, string> = {
  G0: 'gravity-g0',
  G1: 'gravity-g1',
  G2: 'gravity-g2',
  G3: 'gravity-g3',
};
export const urgencyClasses: Record<Urgency, string> = {
  U0: 'urgency-u0',
  U1: 'urgency-u1',
  U2: 'urgency-u2',
  U3: 'urgency-u3',
};
export const signalClasses: Record<Signal, string> = {
  green: 'signal-green',
  blue: 'signal-blue',
  amber: 'signal-amber',
  red: 'signal-red',
  grey: 'signal-grey',
};
