// The hold-to-fire timer behind HoldToFire: held long enough it fires once; released early it does not.
export const holdMs = 600;
type Timers = { set(run: () => void, ms: number): unknown; clear(timer: unknown): void };
const browser: Timers = {
  set: (run, ms) => setTimeout(run, ms),
  clear: timer => clearTimeout(timer as ReturnType<typeof setTimeout>),
};
type HoldPhase = 'idle' | 'holding' | 'fired';

export function createHold(onFire: () => void, ms = holdMs, timers: Timers = browser) {
  let timer: unknown;
  let phase: HoldPhase = 'idle';
  return {
    phase: () => phase,
    start() {
      if (phase !== 'idle') return;
      phase = 'holding';
      timer = timers.set(() => { phase = 'fired'; timer = undefined; onFire(); }, ms);
    },
    // release says how the hold ended: 'early' (nothing fired), 'done' (it had fired) or 'none'.
    release(): 'early' | 'done' | 'none' {
      const was = phase;
      if (timer !== undefined) timers.clear(timer);
      timer = undefined;
      phase = 'idle';
      return was === 'holding' ? 'early' : was === 'fired' ? 'done' : 'none';
    },
  };
}
