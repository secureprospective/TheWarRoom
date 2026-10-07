import { useEffect, useState, type RefObject } from 'react';
import type { ClockReading, Sourced } from '../data/contract';
import { selectProvider } from '../data/provider';
import { connectClock } from './ticker';

export function useLeagueClock(root: RefObject<HTMLElement>): Sourced<ClockReading> | undefined {
  const [reading, setReading] = useState<Sourced<ClockReading>>();
  useEffect(() => {
    let active = true;
    selectProvider().clock().then((value) => {
      if (active) setReading(value);
    });
    return () => {
      active = false;
    };
  }, []);
  useEffect(() => {
    if (reading && root.current) return connectClock(root.current, reading.value.deadlines);
  }, [reading, root]);
  return reading;
}
