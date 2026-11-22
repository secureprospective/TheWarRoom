import type { EligibilityDecision, Provenance } from '../data/contract';

// The owner-facing name of each feed token; Go sends the token, this is the only label table.
const FEED_LABELS: Record<EligibilityDecision['source'], string> = {
  injuries: 'MFL injuries',
  players: 'MFL players',
  rosters: 'MFL rosters',
};

function asOf(feed: Provenance | undefined): string {
  if (!feed?.freshness.fetchedAt) return '';
  const at = new Date(feed.freshness.fetchedAt);
  if (Number.isNaN(at.getTime())) return '';
  return new Intl.DateTimeFormat(undefined, { hour: '2-digit', minute: '2-digit' }).format(at);
}

// One plain line per direction, e.g. "IR: not eligible, not on NFL injured reserve
// (MFL injuries, 11:02)". Eligible activations and promotions name their direction so an
// on-IR player's line cannot read as permission to deactivate.
export function eligibilityLine(
  slot: 'IR' | 'Taxi',
  decision: EligibilityDecision,
  feeds: Record<EligibilityDecision['source'], Provenance> | undefined,
): string {
  const feed = FEED_LABELS[decision.source];
  const when = asOf(feeds?.[decision.source]);
  const provenance = when ? ` (${feed}, ${when})` : ` (${feed})`;
  if (decision.verdict === 'eligible') {
    const direction = decision.direction === 'off_ir'
      ? ' to activate'
      : decision.direction === 'off_taxi'
        ? ' to promote'
        : '';
    return `${slot}: eligible${direction}${provenance}`;
  }
  const reason = decision.reason ?? '';
  if (decision.verdict === 'unknown') {
    return `${slot}: not checked, ${reason}; MFL will decide${provenance}`;
  }
  return `${slot}: not eligible, ${reason}${provenance}`;
}

// The Act tooltip text for an ineligible direction; undefined leaves the Act's own label.
export function ineligibleLabel(
  action: string,
  decision: EligibilityDecision | undefined,
): string | undefined {
  return decision?.verdict === 'ineligible' ? `${action}: ${decision.reason}` : undefined;
}
