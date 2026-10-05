import {MONTHS,csv,playerLabel} from './model.mjs';
import {DEFAULTS,PHASES,MEASURES,CHANNELS,summarize,selection,facetMatch,csvRows,peerRows,weeklySelection,acquisitions,mean} from './behavior.mjs';
import {renderLab,renderCompare,renderDossier,renderWeekly} from './behavior_views.mjs';
import {renderLeague,renderPlayers,renderArchive} from './history_views.mjs';

const $ = id => document.getElementById(id);
const escape = value => String(value ?? '').replace(/[&<>"']/g,char => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[char]));
const fmt = (value,digits = 0) => value === null || value === undefined || !Number.isFinite(Number(value)) ? 'Unavailable' : Number(value).toLocaleString(undefined,{maximumFractionDigits:digits});
const state = {...DEFAULTS,route:'calendar',page:0,sort:'newest',season:null,rosterYear:null,query:'',archiveQuery:'',health:''};
let data,indexRevision = null,teamMap,labels,installPrompt,saved = [],storageProblem = '',questionChoice = '';
const names = {
  calendar:['When the league moves.','Fourteen years of decisions. Compare choices, then trace the assets behind each move.'],
  compare:['Same context. Different choices.','Compare franchise-slot behavior under identical filters—not personalities, prices or trade winners.'],
  teams:['What this franchise chooses.','Directions, exchange shapes, counterpart concentration and changes through time, with evidence attached.'],
  weekly:['From receipt to the lineup.','Observed starts and reconciled weekly decisions. Missing follow-up remains unknown, not failure.'],
  league:['The league, season by season.','Champions, results and recorded drafts. Whole-season outcomes are context, not a causal score.'],
  players:['A player’s league history.','Draft and movement records, reported roster states and league scoring.'],
  archive:['What the evidence permits.','Metric readiness, source coverage, denominators and deliberate limits.']
};
const teamName = (id,year) => (year && teamMap.get(id)?.names[String(year)]) || teamMap.get(id)?.name || `Franchise ${id}`;
const playerName = id => labels.get(id)?.split(' · ')[0] || `MFL player ${id}`;
const teamLink = (id,year) => `<button class="text-button team-link" data-team="${escape(id)}">${escape(teamName(id,year))}</button>`;
const playerLink = id => `<button class="text-button" data-player="${escape(id)}">${escape(playerName(id))}</button>`;
const filePath = ref => typeof ref === 'number' ? data.files[ref]?.file || 'Unresolved source' : ref;
const recordSource = ref => typeof ref !== 'object' ? filePath(ref) : `${filePath(ref.file)}${ref.record === undefined ? '' : ` · record ${ref.record+1}`}${ref.week ? ` · week ${ref.week}, block ${ref.block}, matchup ${ref.matchup}, franchise ${ref.team}` : ''}`;
const portableSources = refs => refs.map(ref => typeof ref === 'object' ? {...ref,file:filePath(ref.file)} : filePath(ref));
const sourceDetails = refs => `<details><summary>Sources (${refs.length})</summary><ul>${refs.map(ref => `<li><code>${escape(recordSource(ref))}</code></li>`).join('')}</ul></details>`;
const table = (headers,rows) => `<div class="scroll" tabindex="0" aria-label="Scrollable data table"><table><thead><tr>${headers.map(h => `<th scope="col">${h}</th>`).join('')}</tr></thead><tbody>${rows.join('')}</tbody></table>${rows.length ? '' : '<p class="empty">No matching observations; absence from this table is not a zero outcome.</p>'}</div>`;
const options = (values,selected) => values.map(([value,label]) => `<option value="${escape(value)}" ${String(value) === String(selected) ? 'selected' : ''}>${escape(label)}</option>`).join('');
const select = (id,label,values) => `<label>${label}<select id="${id}">${options(values,state[id])}</select></label>`;
const assetText = a => a.kind === 'player' ? `${playerLink(a.id)}${a.position ? ` <small>${escape(a.position)}</small>` : ''}` : `<span class="tag">${a.kind === 'pick' ? `${a.year} R${a.round}${a.pick ? ` · pick ${a.pick}` : ''}${a.original ? ` · original ${escape(teamMap.get(a.original)?.abbrev || a.original)}` : ''}` : 'Unparsed: '+escape(a.token)}</span>`;
const context = () => ({data,state,escape,fmt,teamName,playerName,teamLink,playerLink,sourceDetails,table,assetText});

function normalize() {
  if (!Object.hasOwn(names,state.route)) state.route = 'calendar';
  if (!teamMap.has(state.team)) state.team = '';
  if (!teamMap.has(state.compare) || state.compare === state.team) state.compare = '';
  if (!teamMap.has(state.partner)) state.partner = '';
  if (!labels.has(state.player)) state.player = '';
  if (!Object.hasOwn(CHANNELS,state.category)) state.category = 'trade';
  if (!Object.hasOwn(MEASURES,state.measure)) state.measure = 'deals';
  if (!['bilateral','all','ambiguous'].includes(state.policy)) state.policy = 'bilateral';
  if (!['either','in','out'].includes(state.direction)) state.direction = 'either';
  if (state.phase && !PHASES.includes(state.phase)) state.phase = '';
  if (!['','pick','player'].includes(state.asset)) state.asset = '';
  if (!['','1','2','3','4','5','6','7','8'].includes(state.round)) state.round = '';
  if (!['','0','1','2','3','4'].includes(state.horizon)) state.horizon = '';
  if (!['','QB','RB','WR','TE','PK','DT','DE','LB','CB','S'].includes(state.position)) state.position = '';
  if (!['','pick-swap','player-swap','players-for-picks','picks-for-players','mixed','unresolved'].includes(state.shape)) state.shape = '';
  if (!['','W','L','T'].includes(state.result)) state.result = '';
  if (!['','high','middle','low','unknown'].includes(state.role)) state.role = '';
  for (const key of ['from','to']) if (state[key]) {
    const date = new Date(state[key]+'T00:00:00Z');
    if (!/^\d{4}-\d{2}-\d{2}$/.test(state[key]) || !Number.isFinite(date.getTime()) || date.toISOString().slice(0,10) !== state[key]) state[key] = '';
  }
  if (state.year !== null && !data.years.includes(Number(state.year))) state.year = null;
  if (state.month !== null && (!Number.isInteger(Number(state.month)) || Number(state.month) < 0 || Number(state.month) > 11)) state.month = null;
  if (state.day !== null && (!Number.isInteger(Number(state.day)) || Number(state.day) < 1 || Number(state.day) > 31)) state.day = null;
  if (state.day && state.year && state.month !== null && state.day > new Date(Date.UTC(state.year,state.month+1,0)).getUTCDate()) state.day = null;
  const tenure = data.behavior.tenures.find(t => t.id === state.tenure);
  if (!tenure) state.tenure = ''; else state.team = tenure.team;
  if (!state.team) {state.result = ''; state.role = '';}
  if (state.token && !/^(FP_\d+_\d{4}_\d+|DP_\d+_\d+)$/.test(state.token)) state.token = '';
}
function readURL() {
  const [route,query = ''] = location.hash.slice(1).split('?'), params = new URLSearchParams(query);
  Object.assign(state,DEFAULTS,{route:route || 'calendar',page:0});
  for (const key of Object.keys(DEFAULTS)) if (params.has(key)) state[key] = ['year','month','day'].includes(key) ? Number(params.get(key)) : params.get(key);
  normalize();
}
function saveURL() {
  const params = new URLSearchParams();
  for (const key of Object.keys(DEFAULTS)) if (state[key] !== DEFAULTS[key] && state[key] !== '' && state[key] !== null) params.set(key,state[key]);
  history.replaceState(null,'',`#${state.route}${params.size ? '?'+params : ''}`);
}
function filters() {
  const teams = data.teams.map(t => [t.id,t.name]);
  const tradeFacets = ['asset','direction','round','horizon','position','partner','shape','token'];
  const hasTrades = ['trade','all'].includes(state.category);
  const active = ['from','to','phase','asset','round','horizon','position','partner','shape','result','role','token','tenure'].filter(k => state[k] && (hasTrades || !tradeFacets.includes(k))).length;
  const open = $('advanced')?.open ?? active > 0;
  const questionsOpen = $('questions')?.open || false;
  const tokens = [...new Set(data.events.filter(e => e.type === 'TRADE').flatMap(e => e.sides.flat().filter(a => a.kind === 'pick').map(a => a.token)))];
  $('filters').innerHTML = `<div class="entity-filters">${select('team','Franchise slot',[['','All franchises'],...teams])}${select('compare','Compare with',[['','Peer median'],...teams.filter(([id]) => id !== state.team)])}${select('category','Activity',Object.entries(CHANNELS))}${select('measure','Calendar measure',Object.entries(MEASURES))}${select('policy','Trade evidence policy',[['bilateral','Two encoded sides · no flags'],['all','All reported trades'],['ambiguous','Flagged / ambiguous only']])}<label class="player-search">Player<input id="player" type="search" list="player-options" value="${escape(state.player ? labels.get(state.player) : state.query)}" placeholder="Name or exact MFL ID" autocomplete="off"><datalist id="player-options"></datalist></label></div><div class="filter-actions"><button id="clear-player" ${state.player || state.query ? '' : 'hidden'}>Clear player</button><button id="reset">Reset filters</button><button id="export-findings">Export findings JSON</button><details id="questions" ${questionsOpen ? 'open' : ''}><summary>Questions &amp; saved lenses</summary><div class="inline-tools"><label>Open a question<select id="question"><option value="">Choose a question</option><option value="firsts">Who receives first-round picks?</option><option value="future">Who sends picks one draft cycle later?</option><option value="loss">What is recorded after a loss?</option><option value="usage">Are received players subsequently started?</option>${saved.map((q,i) => `<option value="saved-${i}" ${questionChoice === 'saved-'+i ? 'selected' : ''}>${escape(q.name)}</option>`).join('')}</select></label><label>Name this lens<input id="question-name" placeholder="Optional name" maxlength="80"></label><button id="question-save">Save current lens</button><button id="question-delete">Delete selected saved lens</button></div><p id="storage-status" class="chart-note">${escape(storageProblem || 'Saved only in this browser. Shared URLs contain the full analysis context.')}</p></details></div><details id="advanced" class="advanced" ${open ? 'open' : ''}><summary>Refine the evidence ${active ? '· '+active+' active facets' : '· dates, assets, context'}</summary><div class="advanced-grid"><label>From (UTC)<input id="from" type="date" value="${escape(state.from)}"></label><label>Through (UTC)<input id="to" type="date" value="${escape(state.to)}"></label>${select('phase','League phase',[['','All phases'],...PHASES.map(p => [p,p])])}${select('asset','Asset kind',[['','All encoded assets'],['pick','Picks'],['player','Players']])}${select('direction','Asset facet direction',[['either','Received or sent'],['in','Received'],['out','Sent']])}${select('round','Pick round',[['','Any round'],...[1,2,3,4,5,6,7,8].map(n => [n,'Round '+n])])}${select('horizon','Draft-cycle horizon',[['','Any / unknown'],['0','Current / next draft'],['1','One cycle later'],['2','Two cycles later'],['3','Three cycles later'],['4','Four cycles later']])}${select('position','Season-listed position',[['','Any / unknown'],...['QB','RB','WR','TE','PK','DT','DE','LB','CB','S'].map(p => [p,p])])}${select('partner','Counterpart franchise',[['','Any counterpart'],...teams])}${select('shape','Exchange shape',[['','Any shape'],['pick-swap','Pick-only exchange'],['player-swap','Player-only exchange'],['players-for-picks','Sent players, received picks'],['picks-for-players','Sent picks, received players'],['mixed','Mixed package'],['unresolved','Unresolved consideration']])}${select('result','Last safely anchored result',[['','Any / unknown'],['W','Win'],['L','Loss'],['T','Scored tie']])}${select('role','Observed prior regular record',[['','Any / unknown'],['high','≥65% wins, ≥4 games'],['middle','35–65% wins, ≥4 games'],['low','≤35% wins, ≥4 games'],['unknown','Unknown / fewer than 4 games']])}<label>Exact pick token<input id="token" list="pick-tokens" value="${escape(state.token)}" placeholder="FP_0025_2027_1"><datalist id="pick-tokens">${tokens.map(t => `<option value="${escape(t)}"></option>`).join('')}</datalist></label>${data.behavior.tenures.length ? select('tenure','Verified owner tenure',[['','Franchise slot only'],...data.behavior.tenures.map(t => [t.id,`${t.name} · ${t.from} → ${t.to}`])]) : ''}</div><p class="chart-note">Facets select records, not inferred motivations. Asset/counterpart/package facets apply only to completed trades; they do not hide draft or roster observations in other channels. Asset round/horizon/position filters constrain counted trade legs; pick-share denominators remain contextual deals before asset facets. Record bands require complete, safely anchored prior regular-season results; source corrections may be retrospective.</p></details>`;
  for (const id of tradeFacets) $(id).disabled = !hasTrades;
  $('policy').disabled = !hasTrades;
  $('result').disabled = !state.team;
  $('role').disabled = !state.team;
  $('measure').disabled = state.category !== 'trade';
  $('compare').disabled = !state.team;
  $('entity-status').textContent = `${state.tenure ? 'Verified tenure' : 'Franchise-slot analysis · owner continuity unverified'} / ${state.team ? teamName(state.team) : 'All franchises'} / ${state.player ? playerName(state.player) : 'All players'}${state.year || state.month !== null ? ` / selected ${state.month === null ? '' : MONTHS[state.month]+' '}${state.year || 'across all years'}${state.day ? ' day '+state.day : ''}` : ''}`;
}
function render() {
  const active = document.activeElement;
  const focusId = active?.id;
  const focusData = ['cell','day','monthColumn'].map(key => [key,active?.dataset?.[key]]).find(([,value]) => value !== undefined);
  normalize(); saveURL(); filters();
  $('title').textContent = names[state.route][0]; $('intro').textContent = names[state.route][1];
  document.querySelectorAll('nav a').forEach(a => a.hash === '#'+state.route ? a.setAttribute('aria-current','page') : a.removeAttribute('aria-current'));
  const views = {calendar:renderLab,compare:renderCompare,teams:renderDossier,weekly:renderWeekly,league:renderLeague,players:renderPlayers,archive:renderArchive};
  const conflictingAssets = ((state.round || state.horizon || state.token) && (state.position || state.asset === 'player')) || (state.asset === 'pick' && state.position) || (state.category === 'trade' && ((state.measure.includes('pick') && (state.asset === 'player' || state.position)) || (state.measure.startsWith('players') && (state.asset === 'pick' || state.round || state.horizon || state.token))));
  $('view').innerHTML = (conflictingAssets ? '<p class="notice" role="alert">The trade asset facets and measure target incompatible asset kinds. Facets match one asset, not separate legs of a mixed package. Clear the asset facets or use Exchange shape to study mixed packages.</p><button id="clear-asset-facets">Clear asset facets</button>' : '')+(state.from && state.to && state.from > state.to ? '<p class="notice" role="alert">The end date is before the start date. Adjust the date range.</p>' : '')+views[state.route](context());
  const focusTarget = focusId ? $(focusId) : focusData ? [...document.querySelectorAll('button')].find(b => b.dataset[focusData[0]] === focusData[1]) : null;
  if (focusTarget && !focusTarget.disabled) focusTarget.focus({preventScroll:true});
}
function clearPeriod() {state.year = state.month = state.day = null; state.page = 0;}
function navigate(route) {state.route = route; state.page = 0; render(); $('main').focus(); window.scrollTo({top:0});}
function reset() {questionChoice = ''; Object.assign(state,DEFAULTS,{page:0,query:'',archiveQuery:'',health:''}); render();}
function download(content,type,name) {
  const url = URL.createObjectURL(new Blob([content],{type})), a = document.createElement('a'); a.href = url; a.download = name; a.click(); setTimeout(() => URL.revokeObjectURL(url),1000);
}
function exportSelection() {
  let exportState = state.route === 'weekly' ? {...state,category:'trade'} : state;
  if (state.route === 'league') exportState = {...state,category:'draft',year:null,month:null,day:null,phase:'',shape:'',result:'',role:''};
  return selection(data,exportState).filter(v => facetMatch(v,state) && (state.route !== 'league' || v.event.season === Number(state.season || state.year || data.years.at(-2))));
}
function findings() {
  const reportState = state.route === 'weekly' ? {...state,category:'trade'} : state;
  const r = summarize(data,reportState), exported = exportSelection();
  const isTrade = state.route !== 'league' && reportState.category === 'trade';
  const peerReports = isTrade ? peerRows(data,reportState).map(p => ({franchise:p.team,selected:p.views.length,contextual:p.base.length,picksIn:p.pin,picksOut:p.pout,pickShare:p.pickShare,pickNumerator:p.incomingDeals,pickDenominator:p.pickDenominator,playersIn:p.playersIn,playersOut:p.playersOut,playerShare:p.playerShare,playerNumerator:p.incomingPlayerDeals,playerDenominator:p.playerDenominator,partnerHHI:p.hhi,topPartnerShare:p.topPartner,calendarDates:p.dates,per30:p.per30,verifiedOwner:false,eventIds:p.views.map(v => v.event.id)})) : [];
  const weekly = state.route === 'weekly' ? weeklySelection(data,state) : [];
  const eligibleWeeks = weekly.filter(w => w.gap !== null);
  const usage = state.route === 'weekly' ? acquisitions(data,state,state.team) : [];
  return {format:'legacy-nfl-behavior-findings',version:1,indexSchema:data.schema,indexRevision,behaviorDefinitionVersion:data.behavior.version,generatedAt:new Date().toISOString(),archiveAt:data.archiveAt,context:Object.fromEntries(Object.keys(DEFAULTS).map(k => [k,state[k]])),route:state.route,
    subject:state.tenure ? data.behavior.tenures.find(t => t.id === state.tenure) : {kind:'franchise-slot',id:state.team || null,verifiedOwner:false},
    definitions:{pickShare:'Deals receiving matching encoded picks / contextual deals from source seasons with observed pick tokens, before asset facets',pickNet:'Matching picks received minus matching picks sent; subject only',per30:'Qualifying dated records / selected intersecting calendar dates × 30; not permitted trading exposure',policy:state.policy,resultAvailability:data.behavior.resultAvailability,recordBasis:data.behavior.recordBasis,positions:data.behavior.positionBasis},
    counts:{selected:exported.length,contextual:state.route === 'league' ? null : r.base.length,selectedCalendarDates:r.dates,...(isTrade ? {picksIn:r.pin,picksOut:r.pout,net:r.net,playersIn:r.playersIn,playersOut:r.playersOut,playerReceivingNumerator:r.incomingPlayerDeals,playerDenominator:r.playerDenominator,playerShare:r.playerShare,pickReceivingNumerator:r.incomingDeals,pickDenominator:r.pickDenominator,pickShare:r.pickShare,per30:r.per30,partnerHHI:r.hhi,topPartnerShare:r.topPartner} : {tradeMeasures:'Not applicable to this evidence channel'})},
    scope:state.route === 'league' ? {category:'draft',sourceSeason:Number(state.season || state.year || data.years.at(-2)),clickedCalendarPeriodCleared:true,tradeContextFacetsCleared:true} : {category:reportState.category},
    peers:peerReports,distributions:{partners:Object.fromEntries(r.partners),packageShapes:Object.fromEntries(r.shapes),receivedSeasonListedPositions:Object.fromEntries(r.positions)},
    ...(state.route === 'weekly' ? {weeklyReport:{scope:'Franchise/tenure/player/calendar filters only; first NFL kickoff anchors each week. Trade-only facets do not constrain team-week observations.',scoredWeeks:weekly.length,eligibleWeeks:eligibleWeeks.length,meanUnusedPoints:mean(eligibleWeeks.map(w => w.gap)),meanOptimalOverlap:mean(eligibleWeeks.map(w => w.starters.filter(id => w.optimalPlayers.includes(id)).length/w.starters.length)),receipts:weekly.map(w => ({season:w.season,week:w.week,team:w.team,kickoff:w.kickoff,score:w.score,optimal:w.optimal,adjustment:w.adjustment,residual:w.residual,optimalResidual:w.optimalResidual,gap:w.gap,flags:w.flags,omittedScores:w.omittedScores,sources:portableSources(w.sources)})),usage:usage.map(a => ({event:a.view.event.id,player:a.asset.id,observedWeeks:a.observed.map(w => ({season:w.season,week:w.week,sources:portableSources(w.sources)})),nextFinalizedWeeks:a.future.length,firstObservedStart:a.start ? {season:a.start.season,week:a.start.week} : null,censored:a.censored,nextRecordedExit:a.exit ? {id:a.exit.id,date:a.exit.date,flags:a.exit.flags,sources:portableSources(a.exit.sources)} : null}))}} : {}),
    exclusions:{pickFieldUnobserved:r.pickExcluded,flaggedIncluded:r.ambiguous,unknownIncomingPositions:r.unknownPlayerPositions},coverage:data.behavior.coverage.map(row => ({...row,positionsSource:filePath(row.positionsSource),rulesSource:filePath(row.rulesSource),transactionsSource:filePath(row.transactionsSource)})),
    receipts:exported.map(v => ({id:v.event.id,activity:v.event.type,observedPlayers:v.event.players,movements:v.event.movements,draft:v.event.type === 'DRAFT' ? {round:v.event.round,selection:v.event.pick,notes:v.event.notes} : null,pickEncodingObserved:v.event.pickEncodingObserved,date:v.event.date,sourceSeason:v.event.season,analysisSeason:v.event.analysisSeason,subject:v.team,partner:v.partner,received:v.received,sent:v.sent,sources:portableSources(v.event.sources),flags:v.event.flags,priorContext:v.context ? {...v.context,sources:portableSources(v.context.sources)} : null}))};
}
function persistSaved() {
  try {localStorage.setItem('legacy-nfl-behavior-lenses-v1',JSON.stringify(saved)); storageProblem = '';}
  catch (error) {storageProblem = `Cannot persist saved lenses: ${error.message}. Export findings or use the shared URL instead.`;}
}
function question(value) {
  questionChoice = value;
  if (value.startsWith('saved-')) {const q = saved[Number(value.slice(6))]; if (q) Object.assign(state,DEFAULTS,q.context,{route:q.route,page:0,query:''}); render(); return;}
  const team = state.team, compare = state.compare;
  Object.assign(state,DEFAULTS,{team,compare,page:0,query:'',route:'compare'});
  if (value === 'firsts') Object.assign(state,{asset:'pick',round:'1',direction:'in',measure:'picks-in'});
  if (value === 'future') Object.assign(state,{asset:'pick',horizon:'1',direction:'out',measure:'picks-out'});
  if (value === 'loss') Object.assign(state,{result:'L',phase:'in-season',route:team ? 'teams' : 'compare'});
  if (value === 'usage') state.route = 'weekly';
  render();
}
$('workspace').addEventListener('change',event => {
  const id = event.target.id;
  if (Object.hasOwn(DEFAULTS,id) && !['player','year','month','day'].includes(id)) {
    state[id] = event.target.value; state.page = 0;
    if (id === 'team') state.tenure = '';
    render();
  } else if (id === 'player') {
    const value = event.target.value.trim(), match = [...labels].find(([key,label]) => key === value || label.toLowerCase() === value.toLowerCase());
    if (match || !value) {state.player = match?.[0] || ''; state.query = value; state.page = 0; render();}
    else $('entity-status').textContent = 'Player filter not applied. Choose a suggestion or enter an exact MFL ID.';
  } else if (id === 'question') {if (event.target.value) question(event.target.value);}
  else if (['roster-year','season','event-sort','health'].includes(id)) {
    state[{'roster-year':'rosterYear','event-sort':'sort'}[id] || id] = event.target.value; state.page = 0; render();
  }
});
$('workspace').addEventListener('input',event => {
  if (event.target.id === 'player') {
    const query = event.target.value.trim().toLowerCase(); state.query = event.target.value;
    $('player-options').innerHTML = [...labels].filter(([,label]) => label.toLowerCase().includes(query)).slice(0,35).map(([,label]) => `<option value="${escape(label)}"></option>`).join('');
    $('clear-player').hidden = !query && !state.player;
    if (state.route === 'players' && !state.player) $('view').innerHTML = renderPlayers(context());
    if (!query && state.player) {state.player = ''; state.page = 0; render();}
  } else if (event.target.id === 'archive-query') {
    const position = event.target.selectionStart; state.archiveQuery = event.target.value; state.page = 0;
    $('view').innerHTML = renderArchive(context()); $('archive-query').focus(); $('archive-query').setSelectionRange(position,position);
  }
});
$('workspace').addEventListener('click',event => {
  const b = event.target.closest('button'); if (!b) return;
  if (b.dataset.cell) {const [year,month] = b.dataset.cell.split(',').map(Number); Object.assign(state,{year,month,day:null,page:0}); render();}
  else if (b.dataset.monthColumn !== undefined) {Object.assign(state,{year:null,month:Number(b.dataset.monthColumn),day:null,page:0}); render();}
  else if (b.dataset.day) {state.day = state.day === Number(b.dataset.day) ? null : Number(b.dataset.day); state.page = 0; render();}
  else if (b.dataset.team) {state.team = b.dataset.team; state.tenure = ''; navigate('teams');}
  else if (b.dataset.player) {state.player = b.dataset.player; navigate('players');}
  else if (b.dataset.season) {state.season = b.dataset.season; state.page = 0; render();}
  else if (b.dataset.page) {state.page += Number(b.dataset.page); render();}
  else if (b.id === 'reset') reset();
  else if (b.id === 'clear-asset-facets') {for (const key of ['asset','round','horizon','position','token']) state[key] = DEFAULTS[key]; state.page = 0; render();}
  else if (b.id === 'trade-analysis') {state.category = 'trade'; state.page = 0; render();}
  else if (b.id === 'clear-period') {clearPeriod(); render();}
  else if (b.id === 'clear-player') {state.player = state.query = ''; state.page = 0; render();}
  else if (b.id === 'export') download('\uFEFF'+csv(csvRows(exportSelection(),state,data.files)),'text/csv;charset=utf-8',`legacy-nfl-${state.route}-events.csv`);
  else if (b.id === 'export-findings') download(JSON.stringify(findings(),null,2),'application/json','legacy-nfl-behavior-findings.json');
  else if (b.id === 'question-save') {
    if (saved.length >= 20) {$('storage-status').textContent = 'Twenty lenses saved. Delete one before saving another.'; return;}
    const name = $('question-name').value.trim() || `${state.team ? teamName(state.team) : 'League'} · ${MEASURES[state.measure]}`;
    saved.push({name,route:state.route,context:Object.fromEntries(Object.keys(DEFAULTS).map(k => [k,state[k]]))}); questionChoice = 'saved-'+(saved.length-1); persistSaved(); render();
  } else if (b.id === 'question-delete') {
    const value = $('question').value;
    if (!value.startsWith('saved-')) {$('storage-status').textContent = 'Select a saved lens before deleting it.'; return;}
    saved.splice(Number(value.slice(6)),1); questionChoice = ''; persistSaved(); render();
  }
});
document.addEventListener('click',event => {
  const a = event.target.closest('a[href^="#"]'); if (!a || a.hash === '#main') return;
  const route = a.hash.slice(1); if (!Object.hasOwn(names,route) || !data) return;
  event.preventDefault(); navigate(route);
});
window.addEventListener('hashchange',() => {if (data) {readURL(); render();}});

async function offlineSetup() {
  if (!('serviceWorker' in navigator) || !window.isSecureContext) {$('connection').textContent = 'Online-only · use localhost or private HTTPS for installation'; return;}
  try {
    const registration = await navigator.serviceWorker.register('./sw.js'); await navigator.serviceWorker.ready;
    (registration.active || navigator.serviceWorker.controller)?.postMessage({type:'CACHE_ARCHIVE'});
  } catch (error) {$('connection').textContent = `Offline cache unavailable: ${error.message}`;}
}
async function load() {
  $('loading').hidden = false; $('failure').hidden = true; $('workspace').hidden = true;
  try {
    const response = await fetch('./data/archive.json'); if (!response.ok) throw new Error(`Archive index returned HTTP ${response.status}`);
    const bytes = await response.arrayBuffer();
    data = JSON.parse(new TextDecoder('utf-8',{fatal:true}).decode(bytes));
    indexRevision = globalThis.crypto?.subtle ? [...new Uint8Array(await crypto.subtle.digest('SHA-256',bytes))].map(n => n.toString(16).padStart(2,'0')).join('').slice(0,12) : null;
    if (data.schema !== 2 || data.behavior?.version !== 1 || !Array.isArray(data.years) || !data.years.length || !Number.isFinite(Date.parse(data.archiveAt?.slice(0,10)+'T00:00:00Z')) || !Array.isArray(data.events) || !Array.isArray(data.behavior.weeks)) throw new Error('Unsupported index format. Rebuild the behavioral archive.');
    data.teams.sort((a,b) => a.name.localeCompare(b.name)); teamMap = new Map(data.teams.map(t => [t.id,t])); labels = new Map(Object.values(data.players).map(p => [p.id,playerLabel(p)]));
    try {const stored = JSON.parse(localStorage.getItem('legacy-nfl-behavior-lenses-v1') || '[]'); saved = Array.isArray(stored) ? stored.filter(q => typeof q.name === 'string' && Object.hasOwn(names,q.route) && q.context && typeof q.context === 'object').slice(0,20) : [];}
    catch (error) {storageProblem = `Saved lenses could not be opened: ${error.message}. Use URLs or exports.`;}
    readURL(); render(); $('loading').hidden = true; $('workspace').hidden = false;
    $('asof').textContent = `Archive through ${data.archiveAt.replace('T',' ')} · source clock; newest season partial`;
    $('connection').textContent = 'Local archive loaded · preparing offline cache…'; await offlineSetup();
  } catch (error) {$('loading').hidden = true; $('failure').hidden = false; $('failure-message').textContent = `${error.message}. Run python3 build_archive.py, then python3 serve.py from tools/league-history. First offline use requires a successful online visit.`; $('connection').textContent = 'Archive unavailable';}
}
window.addEventListener('beforeinstallprompt',event => {event.preventDefault(); installPrompt = event; $('install').hidden = false;});
$('install').addEventListener('click',async () => {if (installPrompt) {await installPrompt.prompt(); installPrompt = null; $('install').hidden = true;}});
window.addEventListener('appinstalled',() => {$('install').hidden = true;});
navigator.serviceWorker?.addEventListener('message',event => {if (event.data?.type === 'CACHE_READY') $('connection').textContent = 'Offline ready · full archive cached on this device'; else if (event.data?.type === 'CACHE_ERROR') $('connection').textContent = `Archive loaded · offline caching failed: ${event.data.message}`;});
window.addEventListener('offline',() => {$('connection').textContent = 'Network offline · cached content needs completed offline preparation';});
window.addEventListener('online',offlineSetup);
$('retry').addEventListener('click',load);
load();
