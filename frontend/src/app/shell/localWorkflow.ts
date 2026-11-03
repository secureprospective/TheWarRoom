import { joinRead } from '../data/query';
import type { HostedPort } from '../commissioner/hostedRoom';
import { object, text } from '../data/parseValues';
import { nativeEnvelope, type NativeRequest } from '../commands/native';
import { commissioner, market, nativeBindings } from '../commissioner/api';

// Mounts and change events may request the same reading from several screens.
// Keep at most one trailing read so a mid-flight change is not lost.
export const localWorkflow: HostedPort = (request, football = '') => request.mode === 'read'
  ? joinRead(runLocalWorkflow, JSON.stringify([request.command,
    Object.entries(request.arguments).sort(([a], [b]) => a.localeCompare(b)), football]),
    () => runLocalWorkflow(request, football)) : runLocalWorkflow(request, football);

const runLocalWorkflow: HostedPort = async (request: NativeRequest, football = '') => {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('League changes require the desktop app.');
  const envelope = nativeEnvelope(request);
  const api = nativeBindings();
  let direct: (() => Promise<unknown>) | undefined;
  if (request.command === 'native.cap.check')
    direct = () => api.TargetNativeCapCheck(request.arguments.league, request.arguments.viewerZone);
  else if (request.command === 'contract.quote')
    direct = () => api.TargetNativeContractQuote(market.RawContractQuote.createFrom(request.arguments));
  else if (request.command === 'calendar.read')
    direct = () => api.TargetLeagueCalendar(
      commissioner.RawLeagueCalendarScope.createFrom(request.arguments));
  else if (request.command === 'week.read')
    direct = () => api.TargetNativeWeek(commissioner.RawNativeWeekScope.createFrom(request.arguments));
  else if (request.command === 'lineupdelta.read')
    direct = () => api.TargetNativeLineupDelta(request.arguments);
  else if (request.command === 'native.directory.read')
    direct = () => api.TargetNativeDirectory(request.arguments.league);
  // The bounded workflow envelope (internal/commandcatalog/workflow_commands.go) does not carry the
  // ledger or the jobs board, so TargetWorkflow refuses them; both have their own league-view binding.
  else if (request.command === 'commish.ledger.read')
    direct = () => api.TargetCommissionerLedger(request.arguments.league);
  else if (request.command === 'commish.jobs.read')
    direct = () => api.TargetCommissionerJobs(request.arguments.league);
  else if (request.command === 'bracket.read')
    direct = () => api.TargetNativeBracket(request.arguments.league, request.arguments.season);
  else if (request.command === 'projections.read')
    direct = () => api.TargetNativeProjections(request.arguments.league, request.arguments.players,
      request.arguments.season, request.arguments.week);
  const authority = direct ? { command: 'native.assets.read', version: '1', mode: 'read',
    arguments: { league: request.arguments.league } } : envelope;
  const raw = await api.TargetWorkflow(JSON.stringify({ ...authority, football }));
  const r = object(raw, 'Local league', ['label', 'writable', 'football', 'data']);
  if (typeof r.writable !== 'boolean') throw new Error('Unreadable league authority. Refresh to try again.');
  return {
    label: text(r.label, 'League location'),
    writable: r.writable,
    football: text(r.football, 'Football evidence'),
    data: direct ? await direct() : r.data,
  };
};
