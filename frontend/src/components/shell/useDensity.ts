import { useState, useEffect, useCallback } from 'react';
import type { Density } from './types';

// Density persists across launches (localStorage). Default tactical — the
// operate altitude. Callers bind keyboard 1/2/3 (Session-B map) to setDensity.
const STORAGE_KEY = 'twr.density';

export function useDensity(): {
  density: Density;
  setDensity: (d: Density) => void;
  cycle: () => void;
} {
  const [density, setDensityState] = useState<Density>(() => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY) as Density | null;
      return stored === 'narrative' || stored === 'tactical' || stored === 'matrix'
        ? stored
        : 'tactical';
    } catch {
      return 'tactical';
    }
  });

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, density);
    } catch {
      // ignore write failures (private mode, quota)
    }
  }, [density]);

  const setDensity = useCallback((d: Density) => setDensityState(d), []);

  const cycle = useCallback(() => {
    setDensityState((prev) =>
      prev === 'narrative' ? 'tactical' : prev === 'tactical' ? 'matrix' : 'narrative',
    );
  }, []);

  return { density, setDensity, cycle };
}
