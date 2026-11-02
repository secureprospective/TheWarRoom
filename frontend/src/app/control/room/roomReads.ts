import type { CommissionerBindings, commissioner, market } from '../../commissioner/api';
import type { accounts, control } from '../../../../wailsjs/go/models';

/** The commissioner reads the Control Room shows beside the league state (jobs, rules, people, ledger,
 *  notices), which leagueState already holds. Each is an existing binding; nothing here decides a rule. */
export type RoomData = {
  waivers?: market.Market; bids?: market.Market; trades?: market.Market;
  statQueue?: commissioner.StatQueue; calendar?: commissioner.Calendar;
  corrections?: commissioner.Corrections; assets?: commissioner.NativeAssets;
  directory?: commissioner.NativeDirectory; schedule?: commissioner.ManagedSchedule;
  houseRules?: commissioner.NativeHouseRules; week?: commissioner.NativeWeek;
  /** The status strip's source health (TargetShellStatus). */
  status?: control.ShellStatus;
  /** The next rookie draft as Go orders it (draft.read, TargetNativeDraft). */
  draft?: commissioner.NativeDraft;
  /** The engine's calibration parameters (GetParams). */
  params?: control.ParamsResult;
  /** Each local identity's authenticator and last activity (TargetSignInFacts, the accounts store). */
  signIns?: accounts.SignInFact[];
  errors: Partial<Record<RoomSource, string>>;
};
export type RoomSource = 'waivers' | 'bids' | 'trades' | 'statQueue' | 'calendar' | 'corrections'
  | 'assets' | 'directory' | 'schedule' | 'houseRules' | 'week' | 'status' | 'draft' | 'params' | 'signIns';
export type RoomPort = Pick<CommissionerBindings, 'TargetCommissionerMarket' | 'TargetStatQueue'
  | 'TargetLeagueCalendar' | 'TargetCorrections' | 'TargetNativeAssets' | 'TargetNativeDirectory'
  | 'TargetCommissionerSchedule' | 'TargetNativeHouseRules' | 'TargetNativeWeek' | 'TargetNativeDraft' | 'TargetSignInFacts'>
  & { TargetShellStatus(): Promise<control.ShellStatus>; GetParams(): Promise<control.ParamsResult> };

const say = (cause: unknown) => (cause instanceof Error ? cause.message : String(cause)).replace(/^Error: /, '');

/** readRoom reads every source at once; a source that fails leaves its field empty and its words in
 *  errors, so one lost read never blanks the room. */
export async function readRoom(league: string, port: RoomPort): Promise<RoomData> {
  const out: RoomData = { errors: {} };
  const take = async <K extends RoomSource>(source: K, read: () => Promise<RoomData[K]>) => {
    try { out[source] = await read(); } catch (cause) { out.errors[source] = say(cause); }
  };
  await Promise.all([
    take('directory', () => port.TargetNativeDirectory(league)),
    take('assets', () => port.TargetNativeAssets(league)),
    take('waivers', () => port.TargetCommissionerMarket(league, 'waivers.process')),
    take('bids', () => port.TargetCommissionerMarket(league, 'bids.process')),
    take('trades', () => port.TargetCommissionerMarket(league, 'trade.review')),
    take('corrections', () => port.TargetCorrections(league)),
    take('schedule', () => port.TargetCommissionerSchedule(league)),
    take('houseRules', () => port.TargetNativeHouseRules(league)),
    take('status', () => port.TargetShellStatus()),
    take('signIns', () => port.TargetSignInFacts()),
    take('params', async () => {
      const result = await port.GetParams();
      if (!result.ok) throw new Error(result.error || 'The engine parameters could not be read');
      return result;
    }),
  ]);
  const season = out.directory?.rosterSeason;
  if (season) {
    const zone = Intl.DateTimeFormat().resolvedOptions().timeZone;
    // The War Room's calendar read (warroomCalendar.ts): the trade action's windows and the deadline.
    await Promise.all([take('statQueue', () => port.TargetStatQueue(league, season)),
      take('calendar', () => port.TargetLeagueCalendar({ league, season, franchise: '', action: 'trade',
        localZone: zone } as commissioner.RawLeagueCalendarScope)),
      // League Pulse's week read, through the first franchise: the league week is the same for every seat.
      take('week', () => port.TargetNativeWeek({ league, season, localZone: zone,
        franchise: out.directory?.franchises[0]?.franchise ?? '', at: new Date().toISOString() } as
        unknown as commissioner.RawNativeWeekScope)),
      // The Draft room's read (draftroom/state.ts): season 0 is the draft that is next; a zero time is now.
      take('draft', () => port.TargetNativeDraft({ league, season: 0, at: '0001-01-01T00:00:00Z',
        franchise: out.directory?.franchises[0]?.franchise ?? '' } as unknown as commissioner.RawNativeDraftScope))]);
  } else {
    const why = out.errors.directory ?? 'The league has no roster season yet';
    out.errors.statQueue = why;
    out.errors.calendar = why;
    out.errors.week = why;
    out.errors.draft = why;
  }
  return out;
}

/** A franchise's name from the league directory (HQ's identity block), never its id. */
export function teamName(data: RoomData, franchise: string): string {
  return data.directory?.franchises.find(f => f.franchise === franchise)?.name ?? franchise;
}

/** One person's sign-in facts: the people reading names a principal, an account or a linked identity. */
export const signInOf = (data: RoomData, person: string) => data.signIns?.find(f => f.principal === person);

/** The mock's SEEN column: minutes, hours or days since the person's last session activity. */
export function seenAgo(iso: string | undefined, now: number): string {
  const at = iso ? Date.parse(iso) : NaN;
  if (!Number.isFinite(at)) return '—';
  const minutes = Math.max(0, Math.floor((now - at) / 60_000));
  if (minutes < 60) return `${minutes}m`;
  if (minutes < 60 * 24) return `${Math.floor(minutes / 60)}h`;
  return `${Math.floor(minutes / 1440)}d`;
}
