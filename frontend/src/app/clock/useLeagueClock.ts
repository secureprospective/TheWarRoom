import { useEffect, useState, type RefObject } from 'react';
import type { ClockReading, Sourced } from '../data/contract';
import { selectProvider } from '../data/provider';
import { connectClock } from './ticker';

export function useLeagueClock(root: RefObject<HTMLElement>): Sourced<ClockReading> | undefined {
  const [reading, setReading] = useState<Sourced<ClockReading>>();
  useEffect(() => {
    let active = true;
    let request = 0;
    const provider = selectProvider();
    const read = () => {
      const current = ++request;
      void provider.clock().then((value) => {
        if (active && current === request) setReading(value);
      }).catch(() => {
        // A rejected refresh leaves the last reading intact; the next event retries.
      });
    };
    const unsubscribe = provider.onClockChange(read);
    read();
    return () => {
      active = false;
      unsubscribe();
    };
  }, []);
  useEffect(() => {
    if (reading && root.current) return connectClock(root.current, reading.value.deadlines);
  }, [reading, root]);
  return reading;
}
