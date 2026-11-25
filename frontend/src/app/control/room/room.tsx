/** Month and day plus time, in the viewer's zone. */
export function when(iso: string | undefined): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime()) || d.getFullYear() < 1900) return '—';
  return d.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  });
}

/** A ledger sentence with its machine timestamps said as the viewer's own date and time. */
export function plainStamps(text: string): string {
  const stamp = /\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2}(\.\d+)?)?(Z|[+-]\d{2}:\d{2})/g;
  return text.replace(stamp, (iso) => when(iso));
}

