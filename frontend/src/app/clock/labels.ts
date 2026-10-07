import type { ClockReading, Sourced } from '../data/contract';

export type ClockProps = { reading?: Sourced<ClockReading> };

export const phaseLabels: Record<ClockReading['phase'], string> = {
  OFFSEASON: 'Offseason',
  REGULAR_SEASON: 'Regular season',
  PLAYOFFS: 'Playoffs',
};
export const windowLabels: Record<ClockReading['windows'][number]['kind'], string> = {
  contract_options: 'Contract options',
  rfa_tender: 'RFA tender',
  ufa_bidding: 'UFA bidding',
  re_sign: 'Re-sign',
  cut_day: 'Cut day',
  trade_deadline: 'Trade deadline',
  draft: 'Draft',
};
const dates = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' });

export function deadlineLabel(label: string): string {
  if (!/^[A-Z0-9_]+$/.test(label)) return label;
  const words = label.toLowerCase().replaceAll('_', ' ');
  return words.charAt(0).toUpperCase() + words.slice(1);
}

export function localDate(at: string | undefined): string {
  return at === undefined ? 'Date unknown' : dates.format(new Date(at));
}
