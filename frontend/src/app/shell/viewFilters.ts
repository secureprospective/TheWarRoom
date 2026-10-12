import { browserStorage, type SettingsStorage } from './franchise';

// S-10: a workspace's filters survive moving between sub-sections and restarting the app. They
// are kept per workspace in local storage and validated when read back: a stored filter the
// current build does not recognise is dropped, never half-applied.
export const FILTER_KEYS = {
  market: 'thewarroom.filters.market',
  draft: 'thewarroom.filters.draft',
} as const;

export function restoreFilter<T>(key: string, parse: (v: unknown) => T | null, fallback: T,
  storage: SettingsStorage | undefined = browserStorage()): T {
  const stored = storage?.getItem(key);
  if (!stored) return fallback;
  try {
    return parse(JSON.parse(stored)) ?? fallback;
  } catch {
    return fallback;
  }
}

export function keepFilter<T>(key: string, value: T,
  storage: SettingsStorage | undefined = browserStorage()) {
  storage?.setItem(key, JSON.stringify(value));
}

// subset reads a stored list that must hold only allowed ids, in the allowed order.
export function subset(v: unknown, allowed: readonly string[]): string[] | undefined {
  if (!Array.isArray(v) || !v.every(x => typeof x === 'string' && allowed.includes(x))) return undefined;
  return allowed.filter(id => v.includes(id));
}
