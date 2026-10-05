export const MONTHS = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
export const CATEGORIES = {
  picks: 'Trades involving picks', assets: 'Individual picks moved', trades: 'All completed trades',
  draft: 'Draft selections', waiver: 'Adds & drops', roster: 'IR & taxi moves',
  proposals: 'Trade proposals', notes: 'Comment-only pick references', all: 'All recorded events'
};

export function matchesCategory(event, category) {
  switch (category) {
    case 'picks': case 'assets': return event.type === 'TRADE' && event.picks > 0;
    case 'trades': return event.type === 'TRADE';
    case 'draft': return event.type === 'DRAFT';
    case 'waiver': return ['FREE_AGENT','WAIVER'].includes(event.type);
    case 'roster': return ['IR','TAXI'].includes(event.type);
    case 'proposals': return event.type === 'TRADE_PROPOSAL';
    case 'notes': return event.type === 'TRADE' && event.pickNote && !event.picks;
    case 'all': return true;
    default: return false;
  }
}

export function filterEvents(events, filters, drilldown = false) {
  return events.filter(event => {
    if (filters.team && !event.teams.includes(filters.team)) return false;
    if (filters.player && !event.players.includes(filters.player)) return false;
    if (filters.category && !matchesCategory(event, filters.category)) return false;
    if (!event.date) return !filters.from && !filters.to && !(drilldown && (filters.year || filters.month !== null && filters.month !== undefined || filters.day));
    const date = event.date.slice(0,10);
    if (filters.from && date < filters.from) return false;
    if (filters.to && date > filters.to) return false;
    if (drilldown && filters.year && Number(date.slice(0,4)) !== filters.year) return false;
    if (drilldown && filters.month !== null && Number(date.slice(5,7)) - 1 !== filters.month) return false;
    if (drilldown && filters.day && Number(date.slice(8,10)) !== filters.day) return false;
    return true;
  });
}

export function monthlyCounts(events, category) {
  const result = new Map();
  for (const event of events) {
    if (!event.date) continue;
    const key = event.date.slice(0,7);
    result.set(key, (result.get(key) || 0) + (category === 'assets' ? event.picks : 1));
  }
  return result;
}

export function csv(rows) {
  return rows.map(row => row.map(value => {
    let text = String(value ?? '');
    if (/^[\s]*[=+\-@]/.test(text)) text = "'" + text;
    return '"' + text.replaceAll('"','""') + '"';
  }).join(',')).join('\r\n');
}

export function heatLevel(value, maximum) {
  if (!value || !maximum) return 0;
  return Math.max(1, Math.min(5, Math.ceil(Math.log1p(value) / Math.log1p(maximum) * 5)));
}

export function playerLabel(player) {
  const [last, first] = player.name.split(',').map(part => part.trim());
  const name = first ? `${first} ${last}` : last;
  return `${name} · ${player.position} · ${player.id}`;
}
