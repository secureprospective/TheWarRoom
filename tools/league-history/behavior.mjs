/* Pure, dependency-free analysis. Every unit is distinct; unknown never becomes zero. */
export const PHASES = ['pre-draft','draft','post-draft','in-season','postseason','unknown'];
export const MEASURES = {
  deals:'Recorded deals', 'picks-in':'Picks received', 'picks-out':'Picks sent',
  'pick-net':'Net picks (in − out)', 'players-in':'Players received', 'players-out':'Players sent',
  'pick-share':'Deals receiving matching picks (%)', 'per30':'Deals per 30 selected calendar dates'
};
export const CHANNELS = {trade:'Completed trades',draft:'Draft selections',adds:'Add/drop records',manual:'Manual roster adjustments',status:'Roster status changes',proposals:'Proposal/status notices',all:'All archived events'};
export const DEFAULTS = {team:'',compare:'',tenure:'',player:'',category:'trade',measure:'deals',policy:'bilateral',from:'',to:'',year:null,month:null,day:null,phase:'',asset:'',direction:'either',round:'',horizon:'',position:'',partner:'',shape:'',result:'',role:'',token:''};
export const BLOCKED = [
  ['Identity prerequisite','Verified GM attribution','Explicit owner-tenure evidence is required. Franchise aliases do not establish identity; no second league archive.'],
  ['B02','Permitted-opportunity rates','Calendar-date rates are descriptive only; historical trading windows and exceptions are unverified.'],
  ['B03 / B04 / B05','Peer-mean activity lift, phase-share and rolling-burst statistics','Matched median comparisons and calendar/phase slices are shown; the proposed mean ratio and rolling seven-day burst share are not calculated.'],
  ['B10 / B11','Decision-time horizons and position preferences','Archive-backed cycle and season-listed position subsets are shown; exact decision-time attributes and available alternatives are unverified.'],
  ['B16','Effective-partner count','HHI and partner breadth are reported; its reciprocal is not a separate shipped measure.'],
  ['B17 / B26','Dated draft-selection mix and draft-run response','Recorded selections are inspectable; draft-time position histories, remaining alternatives and comparable pick opportunities are unverified.'],
  ['B18 / B19 / B20','Eligibility-adjusted usage and continuous custody','Censored observed starts and next recorded exits are shown; injury eligibility, custody continuity and event-time exposure are unverified.'],
  ['B21 / B22','Official result clocks and full competitive state','Seven-day recording windows use a disclosed proxy; prior regular record is shown, not verified decision-time knowledge or all-play/points percentiles.'],
  ['B24 / B25','Feasible score utilization ratio and starter persistence','Unused points and optimal-set overlap are shown; the proposed score ratio and week-to-week persistence statistic are not calculated.'],
  ['B27 / B28','Modeled stability and partner preference beyond activity','Trajectories and concentration are descriptive; no held-out tendency or temporal-network null model is fitted.'],
  ['B29','Age / experience preference','Validated birthdates, experience and dated contextual attributes are not compiled.'],
  ['B30','Revealed pick-exchange price curve','Packages are inventory, not value. Dated market beliefs, slot uncertainty and utility objectives are unresolved.'],
  ['B31 / B32','Retention after underperformance and realized net outcomes','Incomplete custody and dated expectations/constraints prevent attribution or winner labels; observed usage is not return on investment.'],
  ['Additional questions','Acceptance, response latency, initiation and cap-sensitive choices','No reliable offer/actor clocks or event-time budget, contract and exception histories.']
];
const picks = a => a.kind === 'pick';
const players = a => a.kind === 'player';
export const sum = values => values.reduce((n,v) => n + v,0);
export const mean = values => values.length ? sum(values)/values.length : null;
export function median(values) {const v = [...values].sort((a,b) => a-b); return v.length ? (v[Math.floor((v.length-1)/2)]+v[Math.floor(v.length/2)])/2 : null;}
export function ratio(n,d) {return d > 0 ? n/d : null;}
export function ledgerView(event,team = '') {
  const index = event.teams.indexOf(team);
  if (team && index < 0) return null;
  const resolved = event.type === 'TRADE' && event.teams.length === 2 && new Set(event.teams).size === 2;
  const sent = team ? event.sides[index] || [] : event.sides.flat();
  const received = resolved ? (team ? event.sides[1-index] || [] : event.sides.flat()) : [];
  return {event,team,partner:team && resolved ? event.teams[1-index] : '',sent,received,resolved,
    context:team ? event.context?.[team] : null};
}
export function packageShape(view) {
  if (!view.resolved || view.event.pickEncodingObserved !== true || view.event.flags.length) return 'unresolved';
  const ip = view.received.some(picks), op = view.sent.some(picks), ia = view.received.some(players), oa = view.sent.some(players);
  if (!ia && !oa && (ip || op)) return 'pick-swap';
  if (!ip && !op && (ia || oa)) return 'player-swap';
  if (ip && oa && !ia && !op) return 'players-for-picks';
  if (ia && op && !ip && !oa) return 'picks-for-players';
  return 'mixed';
}
export function packageSize(view) {
  if (!view.team || !view.resolved || view.event.pickEncodingObserved !== true || view.event.flags.length || !view.received.length || !view.sent.length) return null;
  return view.received.length < view.sent.length ? 'Fewer assets received' : view.received.length > view.sent.length ? 'More assets received' : 'Same asset count';
}
export function assetMatch(asset,event,state) {
  if (state.token && asset.token !== state.token) return false;
  if (state.asset && asset.kind !== state.asset) return false;
  if (state.round && (asset.kind !== 'pick' || asset.round !== Number(state.round))) return false;
  if (state.horizon && (asset.kind !== 'pick' || event.cycle === null || asset.year - event.cycle !== Number(state.horizon))) return false;
  if (state.position && (asset.kind !== 'player' || asset.position !== state.position)) return false;
  return true;
}
export function matchingAssets(view,state,direction) {return view[direction].filter(a => assetMatch(a,view.event,state));}
export function dateMatch(date,state,drill = true) {
  if (!date) return !(state.from || state.to || (drill && (state.year || state.month !== null || state.day)));
  const day = date.slice(0,10);
  if ((state.from && day < state.from) || (state.to && day > state.to)) return false;
  return !drill || ((!state.year || Number(day.slice(0,4)) === state.year) && (state.month === null || Number(day.slice(5,7))-1 === state.month) && (!state.day || Number(day.slice(8,10)) === state.day));
}
export function channelMatch(event,category) {
  if (category === 'trade') return event.type === 'TRADE';
  if (category === 'draft') return event.type === 'DRAFT';
  if (category === 'adds') return event.type === 'FREE_AGENT';
  if (category === 'manual') return event.type === 'LOAD_ROSTERS';
  if (category === 'status') return ['IR','TAXI'].some(t => event.type.includes(t));
  if (category === 'proposals') return /TRADE_/.test(event.type);
  return true;
}
export function baseMatch(view,state,data,drill = true) {
  const e = view.event;
  if (!channelMatch(e,state.category) || !dateMatch(e.date,state,drill)) return false;
  if (state.player && !e.players.includes(state.player)) return false;
  if (e.type === 'TRADE' && ((state.policy === 'bilateral' && e.flags.length) || (state.policy === 'ambiguous' && !e.flags.length))) return false;
  if (state.phase && e.phase !== state.phase) return false;
  if (e.type === 'TRADE' && state.partner && view.partner !== state.partner && (view.team || !e.teams.includes(state.partner))) return false;
  if (e.type === 'TRADE' && state.shape && packageShape(view) !== state.shape) return false;
  const tenure = data.behavior.tenures.find(t => t.id === state.tenure);
  if (tenure && (!e.date || e.date.slice(0,10) < tenure.from || e.date.slice(0,10) >= tenure.to || view.team !== tenure.team)) return false;
  if (state.result && view.context?.lastResult !== state.result) return false;
  if (state.role) {
    const c = view.context, games = c ? c.wins+c.losses+c.ties : 0;
    const band = games < 4 || c.recordComplete !== true ? 'unknown' : c.wins/games >= .65 ? 'high' : c.wins/games <= .35 ? 'low' : 'middle';
    if (band !== state.role) return false;
  }
  return true;
}
export function facetMatch(view,state) {
  if (view.event.type !== 'TRADE') return true;
  if (![state.asset,state.round,state.horizon,state.position,state.token].some(Boolean)) return true;
  const directions = state.direction === 'in' ? ['received'] : state.direction === 'out' ? ['sent'] : ['sent','received'];
  return directions.some(d => matchingAssets(view,state,d).length);
}
export function selection(data,state,team = state.team,drill = true) {
  return data.events.map(e => ledgerView(e,team)).filter(v => v && baseMatch(v,state,data,drill));
}
export function observedPickSeason(data,event) {return data.behavior.coverage.some(c => c.season === event.season && c.pickTokensObserved);}
export function measureValue(views,state,data) {
  if (state.category !== 'trade' || state.measure === 'deals') return views.length;
  if ((state.measure.includes('pick') && (state.asset === 'player' || state.position)) || (state.measure.startsWith('players') && (state.asset === 'pick' || state.round || state.horizon || state.token))) return null;
  const valid = views.filter(v => observedPickSeason(data,v.event));
  if (state.measure === 'pick-share') return valid.length ? 100*sum(valid.map(v => matchingAssets(v,state,'received').some(picks)))/valid.length : null;
  if (state.measure.startsWith('pick')) {
    if (!valid.length && views.length) return null;
    const incoming = sum(valid.map(v => matchingAssets(v,state,'received').filter(picks).length));
    const outgoing = sum(valid.map(v => matchingAssets(v,state,'sent').filter(picks).length));
    return state.measure === 'picks-in' ? incoming : state.measure === 'picks-out' ? outgoing : state.team ? incoming-outgoing : null;
  }
  if (state.measure.startsWith('players')) return sum(views.map(v => matchingAssets(v,state,state.measure === 'players-in' ? 'received' : 'sent').filter(players).length));
  return null;
}
export function selectedDates(data,state,drill = true) {
  const first = `${data.years[0]}-01-01`, last = data.archiveAt.slice(0,10);
  const tenure = data.behavior.tenures.find(t => t.id === state.tenure);
  let days = 0;
  for (let t = Date.parse(first+'T00:00:00Z'); t <= Date.parse(last+'T00:00:00Z'); t += 86400000) {
    const day = new Date(t).toISOString();
    if (dateMatch(day,state,drill) && (!tenure || (day.slice(0,10) >= tenure.from && day.slice(0,10) < tenure.to))) days++;
  }
  return days;
}
export function summarize(data,state,team = state.team,drill = true) {
  const base = selection(data,state,team,drill);
  const views = base.filter(v => facetMatch(v,state));
  const dates = selectedDates(data,state,drill);
  const pickBase = base.filter(v => observedPickSeason(data,v.event));
  const pickWindow = data.behavior.coverage.some(c => c.pickTokensObserved && (!state.year || c.season === state.year) && (!state.from || `${c.season}-12-31` >= state.from) && (!state.to || `${c.season}-01-01` <= state.to));
  const pickApplicable = state.asset !== 'player' && !state.position;
  const playerApplicable = state.asset !== 'pick' && !state.round && !state.horizon && !state.token;
  const pickKnown = pickApplicable && dates > 0 && (pickBase.length > 0 || (base.length === 0 && pickWindow));
  const pin = pickKnown ? sum(views.filter(v => observedPickSeason(data,v.event)).map(v => matchingAssets(v,state,'received').filter(picks).length)) : null;
  const pout = pickKnown ? sum(views.filter(v => observedPickSeason(data,v.event)).map(v => matchingAssets(v,state,'sent').filter(picks).length)) : null;
  const incomingDeals = pickApplicable ? pickBase.filter(v => matchingAssets(v,state,'received').some(picks)).length : null;
  const playerBase = base.filter(v => !state.position || v.received.filter(players).every(a => a.position));
  const incomingPlayerDeals = playerApplicable ? playerBase.filter(v => matchingAssets(v,state,'received').some(players)).length : null;
  const playersIn = playerApplicable ? sum(views.map(v => matchingAssets(v,state,'received').filter(players).length)) : null;
  const playersOut = playerApplicable ? sum(views.map(v => matchingAssets(v,state,'sent').filter(players).length)) : null;
  const partners = new Map();
  for (const v of views) if (v.partner) partners.set(v.partner,(partners.get(v.partner)||0)+1);
  const shares = [...partners.values()].map(n => n/views.length);
  const shapes = new Map();
  for (const v of views) {const shape = packageShape(v); shapes.set(shape,(shapes.get(shape)||0)+1);}
  const positions = new Map();
  for (const v of views) for (const a of matchingAssets(v,state,'received').filter(players)) positions.set(a.position || 'Unknown',(positions.get(a.position || 'Unknown')||0)+1);
  return {team,base,views,count:views.length,pin,pout,net:team && pin !== null && pout !== null ? pin-pout : null,incomingDeals,pickDenominator:pickBase.length,
    pickShare:pickApplicable ? ratio(incomingDeals,pickBase.length) : null,
    playersIn,playersOut,incomingPlayerDeals,playerDenominator:playerBase.length,
    playerShare:playerApplicable ? ratio(incomingPlayerDeals,playerBase.length) : null,
    unknownPlayerPositions:base.length-playerBase.length,partners,shapes,positions,
    hhi:shares.length ? sum(shares.map(p => p*p)) : null,
    topPartner:shares.length ? Math.max(...shares) : null,
    dates,per30:dates ? 30*views.filter(v => v.event.date).length/dates : null,
    ambiguous:base.filter(v => v.event.flags.length).length,
    pickExcluded:base.length-pickBase.length};
}
export function peerRows(data,state) {
  // When tenure is selected, compare franchise slots over exactly that tenure's dates.
  const tenure = data.behavior.tenures.find(t => t.id === state.tenure);
  const peerState = {...state,tenure:''};
  if (tenure) {
    peerState.from = !state.from || state.from < tenure.from ? tenure.from : state.from;
    const end = new Date(Date.parse(tenure.to+'T00:00:00Z')-86400000).toISOString().slice(0,10);
    peerState.to = !state.to || state.to > end ? end : state.to;
  }
  return data.teams.map(team => summarize(data,{...peerState,team:team.id},team.id));
}
export function monthlyGrid(data,state) {
  const base = selection(data,state,state.team,false);
  const groups = new Map();
  for (const view of base) if (view.event.date) {
    const key = view.event.date.slice(0,7); if (!groups.has(key)) groups.set(key,[]); groups.get(key).push(view);
  }
  const grid = new Map();
  for (const year of data.years) for (let month = 0; month < 12; month++) {
    const key = `${year}-${String(month+1).padStart(2,'0')}`;
    if (key > data.archiveAt.slice(0,7)) {grid.set(key,null); continue;}
    const monthEnd = new Date(Date.UTC(year,month+1,0)).toISOString().slice(0,10);
    const tenure = data.behavior.tenures.find(t => t.id === state.tenure);
    if ((state.from && monthEnd < state.from) || (state.to && key+'-01' > state.to) || (tenure && (monthEnd < tenure.from || key+'-01' >= tenure.to))) {grid.set(key,null); continue;}
    const all = groups.get(key)||[], matched = all.filter(v => facetMatch(v,state));
    let value;
    if (state.category === 'trade' && state.measure.includes('pick') && (!data.behavior.coverage.some(c => c.season === year && c.pickTokensObserved) || (all.length > 0 && !all.some(v => observedPickSeason(data,v.event))) || (state.measure === 'pick-net' && !state.team))) value = null;
    else if (state.category === 'trade' && state.measure === 'per30') {
      const dates = selectedDates(data,{...state,year,month,day:null}); value = dates ? 30*matched.length/dates : null;
    } else value = measureValue(state.measure === 'pick-share' ? all : matched,state,data);
    grid.set(key,value);
  }
  return grid;
}
export function comparisonBaseline(rows,team,key) {
  return median(rows.filter(r => r.team !== team).map(r => r[key]).filter(v => v !== null && Number.isFinite(v)));
}
function wholeWeekInTenure(week,tenure) {
  return !tenure || (week.kickoff && week.safeAt && week.kickoff.slice(0,10) >= tenure.from && week.safeAt.slice(0,10) < tenure.to);
}
export function weeklySelection(data,state,team = state.team) {
  const tenure = data.behavior.tenures.find(t => t.id === state.tenure);
  return data.behavior.weeks.filter(w => (!team || w.team === team) && dateMatch(w.kickoff,state) && (!state.player || w.starters.includes(state.player) || w.bench.includes(state.player)) && (!tenure || (w.team === tenure.team && wholeWeekInTenure(w,tenure))));
}
export function resultWindows(data,state,team) {
  const observed = data.behavior.weeks.filter(w => w.team === team && w.safeAt).sort((a,b) => a.safeAt.localeCompare(b.safeAt));
  const tenure = data.behavior.tenures.find(t => t.id === state.tenure);
  const stats = new Map(['W','L','T'].map(r => [r,{result:r,windows:0,dates:0,deals:0}]));
  const pool = selection(data,{...state,result:'',role:''},team).filter(v => facetMatch(v,state));
  // Capture timezone was not preserved. Stop at the preceding UTC date, rather
  // than pretending the source-local capture clock is a verified UTC timestamp.
  const archiveEnd = Date.parse(data.archiveAt.slice(0,10)+'T00:00:00Z')-86400000;
  for (const [index,week] of observed.entries()) {
    const start = Date.parse(week.safeAt), next = observed[index+1];
    const end = Math.min(start+7*86400000,archiveEnd,next?.season === week.season ? Date.parse(next.safeAt) : Infinity);
    if (end <= start) continue;
    const record = week.record, games = record ? record.wins+record.losses+record.ties : 0;
    const band = games < 4 || record.complete !== true ? 'unknown' : record.wins/games >= .65 ? 'high' : record.wins/games <= .35 ? 'low' : 'middle';
    if (state.role && state.role !== band) continue;
    // Split on UTC midnight; a window may cross into a selected month even when
    // its game's kickoff is outside it. The next safely available result ends it.
    let days = 0;
    for (let time = start; time < end;) {
      const date = new Date(time).toISOString(), nextMidnight = Date.parse(date.slice(0,10)+'T00:00:00Z')+86400000;
      const segmentEnd = Math.min(end,nextMidnight);
      const anchor = data.behavior.anchors[week.season] || {};
      const phase = anchor.playoffs && date >= anchor.playoffs ? 'postseason' : 'in-season';
      if (dateMatch(date,state) && (!state.phase || state.phase === phase) && (!tenure || (date.slice(0,10) >= tenure.from && date.slice(0,10) < tenure.to))) days += (segmentEnd-time)/86400000;
      time = segmentEnd;
    }
    if (!days) continue;
    const row = stats.get(week.result); row.windows++; row.dates += days;
    row.deals += pool.filter(v => v.context?.week === week.week && v.event.analysisSeason === week.season && v.context?.lastResult === week.result && Date.parse(v.event.date) >= start && Date.parse(v.event.date) < end).length;
  }
  return [...stats.values()].map(s => ({...s,per30:s.dates ? 30*s.deals/s.dates : null}));
}
export function acquisitions(data,state,team) {
  if (!team) return [];
  const tenure = data.behavior.tenures.find(t => t.id === state.tenure);
  const views = selection(data,{...state,category:'trade'},team).filter(v => facetMatch(v,state));
  const weeks = data.behavior.weeks.filter(w => w.team === team && w.safeAt && w.kickoff).sort((a,b) => a.kickoff.localeCompare(b.kickoff));
  const calendar = Object.values(data.behavior.anchors).flatMap(a => a.weekDates || []);
  const fallback = [...new Map(data.behavior.weeks.filter(w => w.kickoff && w.safeAt).map(w => [`${w.season}-${w.week}`,w])).values()];
  const finalizedCalendar = (calendar.length ? calendar : fallback).filter(w => !data.behavior.completedThrough || w.safeAt <= data.behavior.completedThrough).sort((a,b) => a.kickoff.localeCompare(b.kickoff));
  const allMoves = data.events.filter(e => e.date && e.teams.includes(team)).sort((a,b) => a.date.localeCompare(b.date));
  return views.flatMap(v => matchingAssets(v,state,'received').filter(players).map(asset => {
    const exit = allMoves.find(e => e.date > v.event.date && (!tenure || e.date.slice(0,10) < tenure.to) && ((e.type === 'TRADE' && ledgerView(e,team)?.sent.some(a => a.kind === 'player' && a.id === asset.id)) || e.movements?.dropped?.includes(asset.id)));
    const future = finalizedCalendar.filter(w => w.kickoff > v.event.date && w.season === v.event.analysisSeason).slice(0,3);
    const follow = future.filter(w => wholeWeekInTenure(w,tenure) && (!exit || exit.date >= w.kickoff)).map(w => weeks.find(row => row.season === w.season && row.week === w.week)).filter(Boolean);
    const observed = follow.filter(w => w.starters.includes(asset.id) || w.bench.includes(asset.id));
    const start = observed.find(w => w.starters.includes(asset.id));
    const censored = future.length < 3 || observed.length !== future.length;
    return {view:v,asset,future,observed,start,exit,censored,daysToExit:exit ? (Date.parse(exit.date)-Date.parse(v.event.date))/86400000 : null};
  }));
}
export function annualProfile(data,state,team) {
  const base = {...state,year:null};
  return data.years.map(year => ({year,...summarize(data,{...base,year},team)}));
}
export function pearson(pairs) {
  if (pairs.length < 4) return null;
  const a = mean(pairs.map(p => p[0])), b = mean(pairs.map(p => p[1]));
  const numerator = sum(pairs.map(p => (p[0]-a)*(p[1]-b)));
  const d = Math.sqrt(sum(pairs.map(p => (p[0]-a)**2))*sum(pairs.map(p => (p[1]-b)**2)));
  return d ? numerator/d : null;
}
export function csvRows(views,state,files = []) {
  return [['Event ID','Date UTC','Archive season','Analysis season','Phase','Draft cycle','Subject franchise','Partner franchise','Activity','Assets received','Assets sent','Matching encoded picks received','Matching encoded picks sent','Flags','Last observed result','Prior W-L-T','Prior week','Result availability proxy','Prior record complete','Observed player IDs','Draft round','Draft selection','Recorded add IDs','Recorded drop IDs','Sources'],...views.map(v => [v.event.id,v.event.date,v.event.season,v.event.analysisSeason,v.event.phase,v.event.cycle,v.team,v.partner,v.event.type,
    v.received.map(a => a.token || a.id).join('; '),v.sent.map(a => a.token || a.id).join('; '),v.event.type === 'TRADE' && v.event.pickEncodingObserved === true ? matchingAssets(v,state,'received').filter(picks).length : '',v.event.type === 'TRADE' && v.event.pickEncodingObserved === true ? matchingAssets(v,state,'sent').filter(picks).length : '',v.event.flags.join('; '),v.context?.lastResult,v.context ? `${v.context.wins}-${v.context.losses}-${v.context.ties}` : '',v.context?.week,v.context?.safeAt,v.context?.recordComplete,v.event.players.join('; '),v.event.round,v.event.pick,(v.event.movements?.added || []).join('; '),(v.event.movements?.dropped || []).join('; '),JSON.stringify(v.event.sources.map(ref => ({...ref,file:files[ref.file]?.file || ref.file})))])];
}
