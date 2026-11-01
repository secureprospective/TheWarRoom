import type { HostedPort } from '../commissioner/hostedRoom';
import { object, text } from '../data/parseValues';
import { nativeEnvelope, type NativeRequest } from '../commands/native';
import { commissioner, market, nativeBindings } from '../commissioner/api';

export const localWorkflow: HostedPort = async (request: NativeRequest, football = '') => {
  if (typeof window === 'undefined' || !(window as { go?: unknown }).go)
    throw new Error('League changes require the desktop app.');
  const envelope = nativeEnvelope(request);
  const api = nativeBindings();
  let direct: (() => Promise<unknown>) | undefined;
  if (request.command === 'native.cap.check')
    direct = () => api.TargetNativeCapCheck(request.arguments.league, request.arguments.viewerZone);
  else if (request.command === 'contract.quote')
    direct = () => api.TargetNativeContractQuote(market.RawContractQuote.createFrom(request.arguments));
  else if (request.command === 'week.read')
    direct = () => api.TargetNativeWeek(commissioner.RawNativeWeekScope.createFrom(request.arguments));
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

export const localReadOnlyWorkflow: HostedPort = async (request, football = '') => {
  if (request.mode !== 'read') throw new Error('No franchise is assigned to you. This view is read-only.');
  const result = await localWorkflow(request, football);
  return { ...result, writable: false };
};
