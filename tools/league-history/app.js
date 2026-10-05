import {MONTHS,CATEGORIES,filterEvents,monthlyCounts,csv,heatLevel,playerLabel} from './model.mjs';

const $ = id => document.getElementById(id);
const escape = value => String(value ?? '').replace(/[&<>"']/g, char => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[char]));
const fmt = (value, digits = 0) => value === null || value === undefined ? 'Unavailable' : Number(value).toLocaleString(undefined,{maximumFractionDigits:digits});
const state = {route:'calendar',team:'',player:'',query:'',category:'picks',from:'',to:'',year:null,month:null,day:null,page:0,sort:'newest',season:null,rosterYear:null,archiveQuery:'',health:''};
let data, teamMap, labels, shownEvents = [], installPrompt;
const PAGE_SIZE = 40;
const palette = ['#182331','#233d3e','#2d5850','#326d5b','#68b698','#a5dcc7'];
const names = {calendar:['Fourteen years. Twelve months.','Find when the league moves its draft capital. Select a month to see the trades behind it.'],league:['The league, season by season.','Champions, results and draft activity from the archive. The newest season is still in progress.'],teams:['A franchise through time.','Follow results, roster observations and the moves made by each team.'],players:['A player’s league history.','Find draft selections, trades, roster observations and season scoring.'],archive:['What the archive can tell us.','Source coverage, expected gaps and the rules behind the numbers.']};
const filePath = ref => typeof ref === 'number' ? data.files[ref].file : ref;
const teamName = (id, year) => (year && teamMap.get(id)?.names[String(year)]) || teamMap.get(id)?.name || `Franchise ${id}`;
const playerName = id => labels.get(id)?.split(' · ')[0] || `MFL player ${id}`;
const teamLink = (id, year) => `<button class="text-button team-link" data-team="${escape(id)}">${escape(teamName(id,year))}</button>`;
const playerLink = id => `<button class="text-button" data-player="${escape(id)}">${escape(playerName(id))}</button>`;
const recordSource = item => `${filePath(item.file)}${item.record === undefined ? '' : ` · record ${item.record + 1}`}`;
const sourceDetails = sources => `<details><summary>Sources (${sources.length})</summary><ul>${sources.map(item => `<li><code>${escape(typeof item === 'object' ? recordSource(item) : filePath(item))}</code></li>`).join('')}</ul></details>`;
const table = (headers, rows) => `<div class="scroll" tabindex="0" aria-label="Scrollable data table"><table><thead><tr>${headers.map(h => `<th scope="col">${h}</th>`).join('')}</tr></thead><tbody>${rows.join('')}</tbody></table></div>`;
const empty = (title, text) => `<div class="empty"><strong>${escape(title)}</strong>${escape(text)}</div>`;
const options = (values, selected) => values.map(([value,label]) => `<option value="${escape(value)}" ${String(value) === String(selected) ? 'selected' : ''}>${escape(label)}</option>`).join('');
const byEntity = events => filterEvents(events,{team:state.team,player:state.player});
const eventsInSeason = year => data.events.filter(event => event.sources.some(item => data.files[item.file].year === year));
const seasonStatus = year => year === data.years.at(-1) ? 'Partial' : 'Archived';

function routeState() {
  const [route, search = ''] = location.hash.slice(1).split('?');
  if (names[route]) state.route = route;
  const params = new URLSearchParams(search);
  for (const key of ['team','player','category','from','to']) {
    if (params.has(key)) state[key] = params.get(key);
  }
  for (const key of ['year','month','day']) state[key] = params.has(key) ? Number(params.get(key)) : null;
  if (!teamMap.has(state.team)) state.team = '';
  if (!data.players[state.player]) state.player = '';
  if (!CATEGORIES[state.category]) state.category = 'picks';
  for (const key of ['from','to']) if (state[key] && !/^\d{4}-\d{2}-\d{2}$/.test(state[key])) state[key] = '';
  if (state.year && !data.years.includes(state.year)) state.year = null;
  if (state.month !== null && (state.month < 0 || state.month > 11)) state.month = null;
  if (state.day && (state.day < 1 || state.day > 31)) state.day = null;
}

function saveURL() {
  const params = new URLSearchParams();
  for (const key of ['team','player','category','from','to','year','month','day']) {
    if (state[key] !== '' && state[key] !== null) params.set(key,state[key]);
  }
  history.replaceState(null,'',`#${state.route}?${params}`);
}

function render() {
  saveURL();
  $('title').textContent = names[state.route][0];
  $('intro').textContent = names[state.route][1];
  document.querySelectorAll('nav a').forEach(a => a.getAttribute('href') === `#${state.route}` ? a.setAttribute('aria-current','page') : a.removeAttribute('aria-current'));
  $('team').value = state.team;
  if (state.player) $('player').value = labels.get(state.player) || state.player;
  $('clear-player').hidden = !state.player && !$('player').value;
  $('entity-status').textContent = [state.team ? teamName(state.team) : 'All franchises',state.player ? playerName(state.player) : 'All players'].join(' / ');
  const views = {calendar:renderCalendar,league:renderLeague,teams:renderTeams,players:renderPlayers,archive:renderArchive};
  views[state.route]();
}

function calendarEvents(drilldown = false) {return filterEvents(data.events,state,drilldown);}
function clearDrilldown() {state.year = state.month = state.day = null; state.page = 0;}

function renderCalendar() {
  const events = calendarEvents();
  const counts = monthlyCounts(events,state.category);
  const maximum = Math.max(0,...counts.values());
  const trades = byEntity(data.events).filter(e => e.type === 'TRADE' && (!state.from || e.date?.slice(0,10) >= state.from) && (!state.to || e.date?.slice(0,10) <= state.to));
  const pickTrades = trades.filter(e => e.picks);
  const noteTrades = trades.filter(e => e.pickNote && !e.picks);
  const lastMonth = data.archiveAt.slice(0,7);
  let cells = '<span></span>' + MONTHS.map((month,index) => `<button class="month-head" data-month-column="${index}" title="Filter all years to ${month}">${month}</button>`).join('');
  for (const year of data.years) {
    cells += `<div class="year-head">${year}${year === data.years.at(-1) ? '<small>partial</small>' : ''}</div>`;
    for (let month = 0; month < 12; month++) {
      const key = `${year}-${String(month + 1).padStart(2,'0')}`;
      const count = counts.get(key) || 0;
      const future = key > lastMonth;
      const selected = (state.year === year || !state.year) && state.month === month;
      const level = heatLevel(count,maximum);
      cells += `<button class="heat-cell ${future ? 'future' : ''} ${key === lastMonth ? 'partial' : ''} ${selected ? 'selected' : ''}" style="--heat:${palette[level]};--ink:${level >= 4 ? '#0c211d' : '#e5edf5'}" data-cell="${year},${month}" aria-label="${MONTHS[month]} ${year}: ${future ? 'not yet observed' : `${count} ${CATEGORIES[state.category].toLowerCase()}`}${key === lastMonth ? ', partial month' : ''}" aria-pressed="${selected}" ${future ? 'disabled' : ''}>${future ? '—' : fmt(count)}</button>`;
    }
  }
  const peak = [...counts].sort((a,b) => b[1]-a[1])[0];
  const highest = peak && peak[1] ? `<p class="insight">Most activity in this selection: <strong>${MONTHS[Number(peak[0].slice(5))-1]} ${peak[0].slice(0,4)}</strong> · ${fmt(peak[1])} ${escape(CATEGORIES[state.category].toLowerCase())}.</p>` : '';
  $('view').innerHTML = `<div class="calendar-tools"><label>Activity<select id="category">${options(Object.entries(CATEGORIES),state.category)}</select></label><label>From (UTC)<input id="from" type="date" value="${escape(state.from)}"></label><label>Through (UTC)<input id="to" type="date" value="${escape(state.to)}"></label></div>
    ${state.from && state.to && state.from > state.to ? '<p class="notice"><strong>The end date is before the start date.</strong> Adjust the date range to see events.</p>' : ''}
    <div class="metrics"><div><strong>${fmt(trades.length)}</strong><span>completed trades</span></div><div><strong>${fmt(pickTrades.length)}</strong><span>trades with encoded picks</span></div><div><strong>${fmt(pickTrades.reduce((n,e) => n+e.picks,0))}</strong><span>individual picks moved</span></div></div>
    <div class="chart-top"><h2>${escape(CATEGORIES[state.category])} · calendar year</h2><div class="legend">0 ${palette.map(color => `<span class="swatch" style="--heat:${color}" aria-hidden="true"></span>`).join('')} ${fmt(maximum)} max</div></div>
    <div class="scroll" tabindex="0" aria-label="Fourteen-year monthly activity heatmap. Scroll horizontally on smaller screens."><div class="heatmap">${cells}</div></div>
    <p class="chart-note">Cells use actual transaction dates in UTC, not archive-season labels. Counts are exact; color uses a logarithmic scale for visibility. A dash is future/unobserved, not zero. The newest month is partial. Click a month heading to compare that month across all years.</p>
    <p class="notice"><strong>${fmt(noteTrades.length)} additional completed trades have comment-only pick references.</strong> These are excluded from encoded-pick counts; select “Comment-only pick references” to inspect them. Zero means no matching dated records, not proof that nothing happened.</p>${highest}
    <div id="day-calendar"></div><div id="ledger"></div>`;
  if (state.year && state.month !== null) renderDays(events);
  renderLedger(calendarEvents(true),'calendar');
}

function renderDays(events) {
  const days = new Date(Date.UTC(state.year,state.month+1,0)).getUTCDate();
  const offset = (new Date(Date.UTC(state.year,state.month,1)).getUTCDay()+6)%7;
  const counts = new Map();
  for (const event of events) {
    if (!event.date || Number(event.date.slice(0,4)) !== state.year || Number(event.date.slice(5,7))-1 !== state.month) continue;
    const day = Number(event.date.slice(8,10));
    counts.set(day,(counts.get(day)||0)+(state.category === 'assets' ? event.picks : 1));
  }
  const max = Math.max(0,...counts.values());
  const headers = ['Mon','Tue','Wed','Thu','Fri','Sat','Sun'].map(d => `<div class="month-head">${d}</div>`).join('');
  let cells = '<span></span>'.repeat(offset);
  for (let day = 1; day <= days; day++) {
    const value = counts.get(day)||0, level = heatLevel(value,max);
    const key = `${state.year}-${String(state.month+1).padStart(2,'0')}-${String(day).padStart(2,'0')}`;
    const future = key > data.archiveAt.slice(0,10);
    cells += `<button class="heat-cell ${state.day === day ? 'selected' : ''}" style="--heat:${palette[level]};--ink:${level >= 4 ? '#0c211d' : '#e5edf5'}" data-day="${day}" aria-pressed="${state.day === day}" aria-label="${MONTHS[state.month]} ${day}: ${future ? 'unobserved' : value + ' matching activity'}" ${future ? 'disabled' : ''}>${day}<small style="display:block">${future ? '—' : value}</small></button>`;
  }
  $('day-calendar').innerHTML = `<h3>${MONTHS[state.month]} ${state.year} · daily detail (UTC)</h3><div class="day-grid">${headers}${cells}</div>`;
}

function assetText(asset) {
  if (asset.kind === 'player') return playerLink(asset.id);
  if (asset.kind === 'unknown') return `<span class="tag">Unparsed: ${escape(asset.token)}</span>`;
  const position = asset.pick ? ` · pick ${asset.pick}` : '';
  return `<span class="tag pick-tag">${asset.year} R${asset.round}${position}${asset.original ? ` · original ${escape(teamMap.get(asset.original)?.abbrev || asset.original)}` : ''}</span>`;
}

function eventDetails(event) {
  if (event.type === 'DRAFT') return `<strong>R${event.round}.${String(event.pick).padStart(2,'0')}</strong> · ${event.players.map(playerLink).join(', ')}${event.lineage ? `<details><summary>Draft pick notes</summary><p>${escape(event.lineage)}</p></details>` : ''}`;
  if (event.sides.some(side => side.length)) return event.sides.map((side,index) => `<div><span class="muted">${escape(teamName(event.teams[index],event.season))} sent:</span> ${side.length ? side.map(assetText).join(', ') : 'No encoded assets'}</div>`).join('') + (event.pickNote && !event.picks ? '<span class="tag">Pick reference in source comments; quantity unknown</span>' : '');
  const movements = Object.entries(event.movements);
  return movements.length ? movements.map(([action,ids]) => `<div><span class="muted">${escape(action)}:</span> ${ids.map(playerLink).join(', ')}</div>`).join('') : '<span class="muted">No player-level movement encoded in this record.</span>';
}

function renderLedger(events, context, target = 'ledger') {
  shownEvents = [...events].sort((a,b) => state.sort === 'oldest' ? (a.date||'').localeCompare(b.date||'') : (b.date||'').localeCompare(a.date||''));
  const pages = Math.max(1,Math.ceil(shownEvents.length/PAGE_SIZE));
  state.page = Math.min(state.page,pages-1);
  const start = state.page*PAGE_SIZE;
  const period = context === 'calendar' && state.month !== null ? `${MONTHS[state.month]} ${state.year || '· all years'}${state.day ? ` · day ${state.day}` : ''}` : 'Matching events';
  const selected = shownEvents.slice(start,start+PAGE_SIZE);
  const rows = selected.map(event => `<tr><td>${escape(event.date?.slice(0,10) || 'Undated')}<br><small class="muted">${event.date ? event.date.slice(11,16)+' UTC' : ''}</small></td><td>${escape(event.type.replaceAll('_',' '))}<br><small class="muted">Archive season ${event.season}</small></td><td class="wrap">${event.teams.map(id => teamLink(id,event.season)).join('<br>')}</td><td class="wrap">${eventDetails(event)}</td><td>${sourceDetails(event.sources)}<details><summary>Indexed record</summary><pre>${escape(JSON.stringify(event,null,2))}</pre></details></td></tr>`);
  $(target).innerHTML = `<div class="ledger-heading"><h2>${escape(period)} <small class="muted">· ${fmt(events.length)} records</small></h2>${context === 'calendar' && state.month !== null ? '<button id="clear-period">Clear calendar selection</button>' : ''}<label class="sort-label"><span class="sr-only">Event order</span><select id="event-sort">${options([['newest','Newest first'],['oldest','Oldest first']],state.sort)}</select></label><button id="export" ${events.length ? '' : 'disabled'}>Export filtered CSV</button></div>
    ${rows.length ? table(['Date (UTC)','Activity','Franchises','Players & assets','Evidence'],rows) : empty('No matching records','Try a different activity, date range, franchise or player. Archive gaps are listed in Coverage & methodology.')}
    <div class="pagination"><button data-page="-1" ${state.page === 0 ? 'disabled' : ''}>Previous</button><span>${events.length ? `${fmt(start+1)}–${fmt(Math.min(start+PAGE_SIZE,events.length))} of ${fmt(events.length)}` : '0 records'}</span><button data-page="1" ${state.page >= pages-1 ? 'disabled' : ''}>Next</button></div>`;
}

function renderLeague() {
  const summaries = data.seasons.map(season => {
    const events = byEntity(eventsInSeason(season.year));
    const trades = events.filter(e => e.type === 'TRADE');
    return `<tr><td><button class="text-button" data-season="${season.year}">${season.year}</button><br><small class="muted">${seasonStatus(season.year)}</small></td><td>${season.champion ? teamLink(season.champion.team,season.year) : '<span class="muted">Not recorded / not decided</span>'}</td><td class="numeric">${fmt(trades.length)}</td><td class="numeric">${fmt(trades.filter(e => e.picks).length)}</td><td class="numeric">${fmt(trades.reduce((n,e) => n+e.picks,0))}</td><td class="numeric">${fmt(events.filter(e => e.type === 'DRAFT').length)}</td></tr>`;
  });
  const season = data.seasons.find(s => s.year === Number(state.season)) || data.seasons.at(-2);
  state.season = season.year;
  const standings = season.standings.filter(row => !state.team || row.team === state.team).sort((a,b) => (b.wins||0)-(a.wins||0) || (b.pf||0)-(a.pf||0));
  $('view').innerHTML = `<p class="history-note">Activity columns follow the selected franchise/player and the source’s archive season, not calendar filters. Champions always describe the whole league. Draft count means recorded selections; a zero may reflect unavailable results.</p>${table(['Season','Super Bowl winner','Completed trades','Trades with picks','Picks moved','Draft selections'],summaries)}
    <div class="inline-tools"><label>Season results<select id="season">${options(data.years.map(y => [y,`${y} · ${seasonStatus(y)}`]),season.year)}</select></label></div>
    <p class="history-note">MFL-reported records and points. Ordered by wins, then points—not MFL’s full tie-breaker ranking. YTD points can include playoff weeks. Player filters do not change franchise standings.</p>
    ${standingsTable(standings,season.year)}<h2>Recorded draft · ${season.year}</h2><div id="ledger"></div>`;
  renderLedger(byEntity(eventsInSeason(season.year)).filter(e => e.type === 'DRAFT'),'league');
}

function standingsTable(standings,year) {
  return standings.length ? table(['Franchise','W–L–T','Points for (YTD)','Points against','All-play %','Evidence'],standings.map(row => `<tr><td>${teamLink(row.team,year)}</td><td>${fmt(row.wins)}–${fmt(row.losses)}–${fmt(row.ties)}</td><td class="numeric">${fmt(row.pf,2)}</td><td class="numeric">${fmt(row.pa,2)}</td><td class="numeric">${row.allPlay === null ? 'Unavailable' : fmt(row.allPlay*100,1)+'%'}</td><td>${sourceDetails([row.source])}</td></tr>`)) : empty('No standings available','This source does not contain matching franchise results.');
}

function renderTeams() {
  if (!state.team) {
    $('view').innerHTML = `<p>Select a franchise above or open one below.</p>${table(['Franchise','MFL ID','Recorded Super Bowls','Completed trades'],data.teams.sort((a,b) => a.name.localeCompare(b.name)).map(team => `<tr><td>${teamLink(team.id)}</td><td>${escape(team.id)}</td><td class="numeric">${data.seasons.filter(s => s.champion?.team === team.id).length}</td><td class="numeric">${fmt(data.events.filter(e => e.type === 'TRADE' && e.teams.includes(team.id)).length)}</td></tr>`))}`;
    return;
  }
  const events = byEntity(data.events);
  const results = data.seasons.map(season => {
    const standing = season.standings.find(row => row.team === state.team);
    const trades = eventsInSeason(season.year).filter(e => e.type === 'TRADE' && e.teams.includes(state.team));
    return `<tr><td>${season.year}${season.champion?.team === state.team ? ' · Champion' : ''}</td><td>${escape(teamName(state.team,season.year))}</td><td>${standing ? `${fmt(standing.wins)}–${fmt(standing.losses)}–${fmt(standing.ties)}` : 'Unavailable'}</td><td class="numeric">${fmt(standing?.pf,2)}</td><td class="numeric">${trades.length}</td><td class="numeric">${trades.reduce((n,e) => n+e.picks,0)}</td><td>${standing ? sourceDetails([standing.source]) : 'No standings source'}</td></tr>`;
  });
  const year = Number(state.rosterYear || data.years.at(-1));
  const roster = Object.entries(data.snapshots).flatMap(([id,observations]) => observations.filter(row => row.season === year && row.team === state.team && (!state.player || state.player === id)).map(row => ({id,...row}))).sort((a,b) => playerName(a.id).localeCompare(playerName(b.id)));
  $('view').innerHTML = `<h2 class="entity-heading">${escape(teamName(state.team))}</h2><p class="history-note">Same MFL franchise ID across seasons; historical names are preserved. This tracks franchise slots, not verified owner continuity. Results below are franchise-wide.</p>${table(['Season','Name that season','W–L–T','Points for (YTD)','Trades','Picks moved','Evidence'],results)}
    <div class="inline-tools"><label>Roster observations in season<select id="roster-year">${options(data.years.map(y => [y,y]),year)}</select></label></div><p class="history-note">Each row is a distinct reported roster state. Weeks are observations, not proof of continuous ownership; duplicate states are grouped. A player may have several rows.</p>
    ${roster.length ? table(['Player','Position (latest)','Status','Salary (league units)','Contract year','Observed weeks','Evidence'],roster.map(row => `<tr><td>${playerLink(row.id)}</td><td>${escape(data.players[row.id]?.position || 'Unknown')}</td><td>${escape(row.status)}</td><td>${fmt(row.salary,2)}</td><td>${escape(row.contract || 'Unavailable')}</td><td class="wrap">${row.weeks.map(w => w ?? 'unspecified').join(', ')}</td><td>${sourceDetails(row.sources)}</td></tr>`)) : empty('No matching roster observations','Change the season or clear the player filter. Missing sources are not evidence of an empty roster.')}
    <h2>Franchise activity</h2><p class="history-note">All recorded activity; calendar activity/date filters do not apply here. Player filter applies when selected.</p><div id="ledger"></div>`;
  renderLedger(events,'teams');
}

function renderPlayers() {
  if (!state.player) {
    const query = state.query.toLowerCase().trim();
    const candidates = Object.values(data.players).filter(player => (!query || labels.get(player.id).toLowerCase().includes(query)) && (!state.team || data.snapshots[player.id]?.some(row => row.team === state.team))).sort((a,b) => playerName(a.id).localeCompare(playerName(b.id)));
    const active = query ? candidates : candidates.filter(p => data.snapshots[p.id]);
    $('view').innerHTML = `<p>Search above to open a player. ${state.team ? 'This list includes players observed on the selected franchise.' : 'With no search, this list includes players with league roster observations.'}</p><p class="muted">${fmt(active.length)} matches · showing first 60</p>${active.length ? table(['Player','MFL ID','Latest position','Latest NFL team'],active.slice(0,60).map(p => `<tr><td>${playerLink(p.id)}</td><td>${escape(p.id)}</td><td>${escape(p.position)}</td><td>${escape(p.team)}</td></tr>`)) : empty('No players found','Search by part of a name or an exact MFL ID.')}`;
    return;
  }
  const player = data.players[state.player];
  const observations = (data.snapshots[state.player] || []).filter(row => !state.team || row.team === state.team).sort((a,b) => b.season-a.season);
  const events = byEntity(data.events);
  const scores = [...(data.scores[state.player] || [])].sort((a,b) => b.season-a.season);
  $('view').innerHTML = `<h2 class="entity-heading">${escape(playerName(state.player))}</h2><p class="muted">MFL ID ${escape(state.player)} · latest listed position ${escape(player.position)} · latest NFL team ${escape(player.team)}</p>
    <div class="summary-strip"><div><strong>${new Set(observations.map(row => row.season)).size}</strong><span>seasons with matching roster observations</span></div><div><strong>${events.filter(e => e.type === 'TRADE').length}</strong><span>completed trades involving this player</span></div><div><strong>${events.filter(e => e.type === 'DRAFT').length}</strong><span>recorded draft selections</span></div></div>
    <h3>Season scoring</h3><p class="history-note">MFL YTD totals for the league’s scoring system; newest year is partial. Missing rows are unavailable, not zero. Totals are player-wide, regardless of franchise filter.</p>
    ${scores.length ? table(['Archive season','MFL YTD points','Evidence'],scores.map(row => `<tr><td>${row.season} · ${seasonStatus(row.season)}</td><td class="numeric">${fmt(row.score,2)}</td><td>${sourceDetails([row.source])}</td></tr>`)) : empty('No archived scoring totals','The player is absent from the archived YTD score exports.')}
    <h2>Roster observations</h2><p class="history-note">Reported weeks are snapshots, not inferred ownership intervals. Status, salary and contract changes produce separate rows.</p>
    ${observations.length ? table(['Season','Fantasy franchise','Status','Salary (league units)','Contract year','Observed weeks','Evidence'],observations.map(row => `<tr><td>${row.season}</td><td>${teamLink(row.team,row.season)}</td><td>${escape(row.status)}</td><td>${fmt(row.salary,2)}</td><td>${escape(row.contract || 'Unavailable')}</td><td class="wrap">${row.weeks.map(w => w ?? 'unspecified').join(', ')}</td><td>${sourceDetails(row.sources)}</td></tr>`)) : empty('No matching roster observations','This player may appear only in transaction or draft records. Try clearing the franchise filter.')}
    <h2>Draft & movement history</h2><p class="history-note">Calendar activity/date filters do not apply here. Proposals are labelled separately from completed trades.</p><div id="ledger"></div>`;
  renderLedger(events,'players');
}

function renderArchive() {
  const query = state.archiveQuery.toLowerCase();
  const files = data.files.filter(row => (!state.health || row.health === state.health) && (!query || `${row.file} ${row.type} ${row.status}`.toLowerCase().includes(query)));
  const start = state.page*PAGE_SIZE;
  $('view').innerHTML = `<div class="summary-strip"><div><strong>${fmt(data.files.length)}</strong><span>source files</span></div><div><strong>${fmt(data.counts.ok)}</strong><span>usable responses</span></div><div><strong>${fmt(data.counts.error)}</strong><span>expected MFL errors</span></div><div><strong>${fmt(data.counts.empty)}</strong><span>empty responses</span></div></div>
    <p class="notice">The expected errors cover unused features, unavailable MFL-wide lists, future 2026 weeks and membership-restricted 2013–2015 data. They are retained in this inventory, not treated as successful data.</p>
    <div class="inline-tools"><label>Find a source<input id="archive-query" type="search" placeholder="Year, file, data type or status" value="${escape(state.archiveQuery)}"></label><label>Status<select id="health">${options([['','All statuses'],['ok','Usable'],['error','MFL error'],['empty','Empty']],state.health)}</select></label></div><p class="muted">${fmt(files.length)} matching files. Team/player filters do not apply to source inventory.</p>
    ${files.length ? table(['Archive season','Source file','Data type','Status','Size','Detail'],files.slice(start,start+PAGE_SIZE).map(row => `<tr><td>${row.year}</td><td class="file-path"><code>${escape(row.file)}</code></td><td>${escape(row.type)}</td><td>${escape(row.health)}</td><td>${fmt(row.bytes/1024,1)} KB</td><td class="wrap">${escape(row.status)}</td></tr>`)) : empty('No matching files','Try a year such as 2025, “rosters”, or a different status.')}
    <div class="pagination"><button data-page="-1" ${state.page === 0 ? 'disabled' : ''}>Previous</button><span>${fmt(files.length ? start+1 : 0)}–${fmt(Math.min(start+PAGE_SIZE,files.length))} of ${fmt(files.length)}</span><button data-page="1" ${start+PAGE_SIZE >= files.length ? 'disabled' : ''}>Next</button></div>
    <section class="help"><h2>How to read the numbers</h2><ul><li><strong>Calendar vs. season.</strong> Calendar uses transaction timestamps in UTC. Historical season exports sometimes contain offseason moves dated the following year. League summaries use the archive season instead.</li><li><strong>Completed trades.</strong> Only MFL’s TRADE records count. Proposals, accept notices, rejections and revocations do not count as completed trades.</li><li><strong>Pick activity.</strong> FP_originalFranchise_year_round and DP_round_pick tokens are parsed. DP picks use the source season as draft year. A pick moved several times counts once for each recorded transfer. “Trades with picks” counts deals, not individual picks.</li><li><strong>Comment-only transfers.</strong> ${fmt(data.notes.commentOnlyPickTrades)} trades contain pick references in comments but no encoded picks. References are detected conservatively, not interpreted as quantities. They may include pick discussion rather than a transfer. Inspect the original local transaction record to verify.</li><li><strong>Snapshots.</strong> Archived roster sources are observations reported by MFL, not reconstructed continuous ownership intervals. Historical endpoints can reflect corrections made later.</li><li><strong>Coverage.</strong> A usable response can still lack a particular field. In particular, the early draft results are absent or contain unfilled picks. Unknown champions and missing results remain unavailable, not inferred. Zero activity means zero matching records.</li><li><strong>Provenance.</strong> “record” is a one-based index in the original MFL array. File references are relative to league-archive/. Indexed records are normalized, not verbatim raw JSON. Identical dated transaction records are deduplicated and retain all source references.</li><li><strong>Privacy.</strong> This PWA serves only a generated, limited index. Owner contacts, credentials, message boards, and arbitrary trade comments are excluded. Draft pick notes are retained. League data stays local and out of git.</li><li><strong>Offline.</strong> Open on localhost or HTTPS to cache the application and full index. Cache updates are revision-aware. A plain LAN HTTP URL is online-only. No live MFL requests are made.</li></ul></section>`;
}

function exportEvents() {
  const rows = [['Date UTC','Archive season','Activity','Franchises','Encoded picks','Player names','Assets sent by franchise 1','Assets sent by franchise 2','Sources']];
  for (const event of shownEvents) rows.push([event.date,event.season,event.type,event.teams.map(id => teamName(id,event.season)).join('; '),event.picks,event.players.map(playerName).join('; '),...([0,1].map(i => (event.sides[i]||[]).map(a => a.kind === 'player' ? playerName(a.id) : a.token).join('; '))),event.sources.map(recordSource).join('; ')]);
  const url = URL.createObjectURL(new Blob(['\uFEFF'+csv(rows)],{type:'text/csv;charset=utf-8'}));
  const anchor = document.createElement('a'); anchor.href = url; anchor.download = `legacy-nfl-${state.route}-events.csv`; anchor.click(); setTimeout(() => URL.revokeObjectURL(url),1000);
}

function attachEvents() {
  $('team').addEventListener('change',() => {state.team = $('team').value; state.page = 0; render();});
  $('player').addEventListener('input',() => {
    state.query = $('player').value; $('clear-player').hidden = !state.query && !state.player;
    if (!state.query) {state.player = ''; state.page = 0; render(); return;}
    const query = state.query.toLowerCase().trim();
    const matches = query ? [...labels].filter(([,label]) => label.toLowerCase().includes(query)).slice(0,35) : [];
    $('player-options').innerHTML = matches.map(([,label]) => `<option value="${escape(label)}"></option>`).join('');
    if (state.route === 'players' && !state.player) renderPlayers();
    $('entity-status').textContent = state.player ? `Current player filter: ${playerName(state.player)}. Choose a suggestion to replace it.` : 'Choose a suggested player to apply the filter.';
  });
  $('player').addEventListener('change',() => {
    const value = $('player').value.trim();
    const match = [...labels].find(([id,label]) => id === value || label.toLowerCase() === value.toLowerCase());
    state.page = 0;
    if (match || !value) {state.player = match?.[0] || ''; render();}
    else {$('entity-status').textContent = state.player ? `Still filtering to ${playerName(state.player)}. Choose a suggested player or clear the filter.` : 'Player filter not applied. Choose a suggestion, enter an exact MFL ID, or open Players to browse matches.';}
  });
  $('clear-player').addEventListener('click',() => {state.player = state.query = ''; $('player').value = ''; state.page = 0; render();});
  $('reset').addEventListener('click',() => {Object.assign(state,{team:'',player:'',query:'',category:'picks',from:'',to:'',page:0,archiveQuery:'',health:''}); clearDrilldown(); $('player').value = ''; render();});
  $('view').addEventListener('change',event => {
    const id = event.target.id;
    const keys = {'category':'category','from':'from','to':'to','season':'season','roster-year':'rosterYear','event-sort':'sort','health':'health'};
    if (!keys[id]) return;
    state[keys[id]] = event.target.value; state.page = 0;
    if (['category','from','to'].includes(id)) clearDrilldown();
    render();
  });
  $('view').addEventListener('input',event => {
    if (event.target.id !== 'archive-query') return;
    const input = event.target, position = input.selectionStart;
    state.archiveQuery = input.value; state.page = 0; renderArchive(); $('archive-query').focus(); $('archive-query').setSelectionRange(position,position);
  });
  $('view').addEventListener('click',event => {
    const button = event.target.closest('button'); if (!button) return;
    if (button.dataset.cell) {const [year,month] = button.dataset.cell.split(',').map(Number); Object.assign(state,{year,month,day:null,page:0}); render();}
    else if (button.dataset.monthColumn !== undefined) {Object.assign(state,{year:null,month:Number(button.dataset.monthColumn),day:null,page:0}); render();}
    else if (button.dataset.day) {state.day = state.day === Number(button.dataset.day) ? null : Number(button.dataset.day); state.page = 0; render();}
    else if (button.dataset.team) {state.team = button.dataset.team; state.route = 'teams'; state.page = 0; render(); window.scrollTo({top:0});}
    else if (button.dataset.player) {state.player = button.dataset.player; state.route = 'players'; state.page = 0; render(); window.scrollTo({top:0});}
    else if (button.dataset.season) {state.season = Number(button.dataset.season); state.page = 0; render();}
    else if (button.dataset.page) {state.page += Number(button.dataset.page); render();}
    else if (button.id === 'clear-period') {clearDrilldown(); render();}
    else if (button.id === 'export') exportEvents();
  });
  document.querySelector('nav').addEventListener('click',event => {
    const link = event.target.closest('a'); if (!link) return;
    event.preventDefault(); state.route = link.hash.slice(1); state.page = 0; render();
  });
  window.addEventListener('hashchange',() => {state.page = 0; routeState(); render();});
}

async function offlineSetup() {
  if (!('serviceWorker' in navigator) || !window.isSecureContext) {
    $('connection').textContent = 'Online-only · use localhost or HTTPS for offline installation'; return;
  }
  try {
    const registration = await navigator.serviceWorker.register('./sw.js');
    await navigator.serviceWorker.ready;
    const worker = registration.active || navigator.serviceWorker.controller;
    worker?.postMessage({type:'CACHE_ARCHIVE'});
  } catch (error) {$('connection').textContent = `Offline cache unavailable: ${error.message}`;}
}

async function load() {
  $('loading').hidden = false; $('failure').hidden = true; $('workspace').hidden = true;
  try {
    const response = await fetch('./data/archive.json');
    if (!response.ok) throw new Error(`Archive index returned HTTP ${response.status}.`);
    data = await response.json();
    if (data.schema !== 1 || !Array.isArray(data.events) || !Array.isArray(data.files) || !Array.isArray(data.years)) throw new Error('The generated archive index has an unsupported format. Rebuild it.');
    teamMap = new Map(data.teams.map(team => [team.id,team])); labels = new Map(Object.values(data.players).map(player => [player.id,playerLabel(player)]));
    $('team').innerHTML = '<option value="">All franchises</option>' + options(data.teams.sort((a,b) => a.name.localeCompare(b.name)).map(team => [team.id,team.name]),'');
    state.season = data.years.at(-2); routeState(); render();
    $('loading').hidden = true; $('workspace').hidden = false;
    $('asof').textContent = `Archive through ${data.archiveAt.replace('T',' ')} · source clock; newest season partial`;
    $('connection').textContent = navigator.onLine ? 'Local archive loaded · preparing offline cache…' : 'Offline · using cached archive';
    await offlineSetup();
  } catch (error) {
    $('loading').hidden = true; $('failure').hidden = false;
    $('failure-message').textContent = `${error.message} Run “python3 build_archive.py” and “python3 serve.py” from tools/league-history, then open the printed localhost URL. Offline use needs a successful first online visit.`;
    $('connection').textContent = 'Archive unavailable';
  }
}

window.addEventListener('beforeinstallprompt',event => {event.preventDefault(); installPrompt = event; $('install').hidden = false;});
$('install').addEventListener('click',async () => {if (!installPrompt) return; await installPrompt.prompt(); installPrompt = null; $('install').hidden = true;});
window.addEventListener('appinstalled',() => {$('install').hidden = true;});
navigator.serviceWorker?.addEventListener('message',event => {
  if (event.data.type === 'CACHE_READY') $('connection').textContent = 'Offline ready · full archive cached on this device';
  if (event.data.type === 'CACHE_ERROR') $('connection').textContent = `Archive loaded · offline caching failed: ${event.data.message}`;
});
window.addEventListener('offline',() => {$('connection').textContent = 'Network offline · cached content available only if offline preparation completed';});
window.addEventListener('online',() => {$('connection').textContent = 'Network available · checking offline archive'; offlineSetup();});
$('retry').addEventListener('click',load);
attachEvents();
load();
