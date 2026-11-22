import type { Urgency } from '../look/channels';

export function grade(at: string | undefined, now: number): Urgency {
  if (at === undefined) return 'U0';
  const remaining = Date.parse(at) - now;
  if (remaining < 3_600_000) return 'U3';
  return remaining <= 172_800_000 ? 'U2' : 'U1';
}

export function countdown(at: string | undefined, now: number): string {
  if (at === undefined) return '—';
  const remaining = Date.parse(at) - now;
  if (remaining < 0) return 'passed';
  const minutes = Math.floor(remaining / 60_000);
  if (minutes < 1) return 'under 1m';
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  if (days) return `${days}d ${hours % 24}h`;
  if (hours) return `${hours}h ${minutes % 60}m`;
  return `${minutes}m`;
}
