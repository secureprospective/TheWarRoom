// The viewer's own time zone, read the way the browser reports it. Every time a person reads is in it.
export const viewerZone = () => Intl.DateTimeFormat().resolvedOptions().timeZone;

export function localTime(iso: string, extra: Intl.DateTimeFormatOptions = {}): string {
  return new Date(iso).toLocaleString(undefined, {
    weekday: 'short', month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit',
    timeZoneName: 'short', ...extra, timeZone: viewerZone(),
  });
}
