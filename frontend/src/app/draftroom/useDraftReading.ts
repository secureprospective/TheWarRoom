import { useEffect, useRef, useState } from 'react';
import { draftRoom } from './state';

// The room looks at the draft again this often, to see other franchises' picks; the clock running out
// makes it look at once.
const lookEveryMs = 15_000;

// useNow is the time a countdown is read against, ticking each second while there is anything to count.
export function useNow(counting: boolean): number {
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    if (!counting) return undefined;
    setNow(Date.now());
    const timer = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(timer);
  }, [counting]);
  return now;
}

// useDraftReading keeps the draft read for one franchise: now, on a steady look, and the moment something is
// due, which is the pick on the clock running out (Go skips it when the draft is read, so the next reading
// shows the skip) or the draft's start arriving. due is that instant, or empty when nothing is counting.
export function useDraftReading(league: string, franchise: string, due: string, now: number, reread: string,
  startup = false) {
  const looked = useRef('');
  useEffect(() => {
    if (!league || !franchise) return undefined;
    void draftRoom.read(league, franchise);
    const timer = setInterval(() => void draftRoom.read(league, franchise), lookEveryMs);
    return () => clearInterval(timer);
  }, [league, franchise, reread, startup]);
  useEffect(() => {
    if (!due || !league || now < Date.parse(due) || looked.current === due) return;
    looked.current = due;
    void draftRoom.read(league, franchise);
  }, [now, due, league, franchise]);
}
