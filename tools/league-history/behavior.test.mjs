import test from 'node:test';
import assert from 'node:assert/strict';
import {DEFAULTS,ledgerView,selection,summarize,monthlyGrid,peerRows,packageShape,dateMatch,acquisitions,weeklySelection,resultWindows,csvRows,annualProfile,packageSize} from './behavior.mjs';
const pick = (year,round,original = '0002') => ({kind:'pick',year,round,original,token:`FP_${original}_${year}_${round}`});
const player = (id,position = 'QB') => ({kind:'player',id,position});
function event(id,date,sides,extra = {}) {return {id,date,season:2025,pickEncodingObserved:true,analysisSeason:2025,phase:'pre-draft',cycle:2025,type:'TRADE',teams:['0001','0002'],sides,players:sides.flat().filter(a => a.kind === 'player').map(a => a.id),picks:sides.flat().filter(a => a.kind === 'pick').length,flags:[],context:{},sources:[{file:0,record:0}],movements:{},...extra};}
function fixture() {
  return {years:[2013,2025,2026],archiveAt:'2026-10-05T12:00:00',teams:[{id:'0001'},{id:'0002'},{id:'0003'}],events:[
    event('a','2025-03-05T10:00:00+00:00',[[player('p')],[pick(2026,1),pick(2026,2)]]),
    event('b','2025-03-06T10:00:00+00:00',[[pick(2025,1)],[player('q')]]),
    event('c','2025-03-07T10:00:00+00:00',[[],[pick(2026,1)]],{flags:['empty-encoded-side']}),
    event('old','2013-03-07T10:00:00+00:00',[[player('p')],[player('q')]],{season:2013,pickEncodingObserved:false,analysisSeason:2013,cycle:null})
  ],behavior:{version:1,coverage:[{season:2013,pickTokensObserved:false},{season:2025,pickTokensObserved:true},{season:2026,pickTokensObserved:true}],weeks:[],tenures:[],anchors:{2025:{kickoff:'2025-09-01T00:00:00+00:00'}}}};
}
const lens = patch => ({...DEFAULTS,...patch});
test('received / sent reverses by subject; league conserved without a subject net',() => {
  const data = fixture(), a = ledgerView(data.events[0],'0001'), b = ledgerView(data.events[0],'0002');
  assert.deepEqual(a.received,b.sent); assert.deepEqual(a.sent,b.received);
  assert.equal(summarize(data,lens({team:'0001'})).pin,2);
  assert.equal(summarize(data,lens({team:'0001'})).pout,1);
  assert.equal(summarize(data,lens({team:'0002'})).net,-1);
  assert.equal(summarize(data,lens({})).net,null);
  assert.equal(ledgerView(data.events[0],'0003'),null);
});
test('unknown consideration is flagged; policy excludes it but never commissioner entry',() => {
  const data = fixture(); data.events[0].commissioner = true;
  assert.equal(selection(data,lens({team:'0001'})).length,3);
  assert.equal(selection(data,lens({team:'0001',policy:'all'})).length,4);
  assert.equal(selection(data,lens({team:'0001',policy:'ambiguous'})).length,1);
  assert.equal(packageShape(ledgerView(data.events[2],'0001')),'unresolved');
});
test('asset facets constrain numerator legs, not the contextual receiving-share denominator',() => {
  const data = fixture(), s = lens({team:'0001',asset:'pick',round:'1',horizon:'1',direction:'in'}), r = summarize(data,s);
  assert.equal(r.views.length,1); assert.equal(r.pin,1); assert.equal(r.pickDenominator,2);
  assert.equal(r.incomingDeals,1); assert.equal(r.pickShare,.5); assert.equal(r.pickExcluded,1);
  assert.equal(summarize(data,{...s,team:'0002'}).views.length,0);
});
test('player choices have their own counts, denominator and coverage, not fabricated pick zeroes',() => {
  const data = fixture(), s = lens({team:'0001',asset:'player',position:'QB',direction:'in',measure:'players-in'});
  let r = summarize(data,s); assert.equal(r.playersIn,2); assert.equal(r.playersOut,1);
  assert.equal(r.playerShare,2/3); assert.equal(r.pickShare,null); assert.equal(r.pin,null);
  data.events[1].sides[1][0].position = null;
  r = summarize(data,s); assert.equal(r.unknownPlayerPositions,1); assert.equal(r.playerDenominator,2); assert.equal(r.playerShare,.5);
});
test('early pick coverage stays unknown, while covered zero is zero and share zero-denominator is unknown',() => {
  const data = fixture(), s = lens({team:'0001',measure:'picks-in'}), grid = monthlyGrid(data,s);
  assert.equal(grid.get('2013-03'),null); assert.equal(grid.get('2025-03'),2); assert.equal(grid.get('2025-04'),0);
  assert.equal(monthlyGrid(data,{...s,measure:'pick-share'}).get('2025-04'),null);
  assert.equal(grid.get('2026-11'),null);
  assert.equal(monthlyGrid(data,{...s,from:'2025-03-01',to:'2025-03-31'}).get('2025-04'),null);
});
test('calendar retains overview; clicked month/day narrows every dossier/peer receipt',() => {
  const data = fixture(), s = lens({team:'0001',year:2025,month:2,day:5});
  assert.equal(summarize(data,s).views.length,1);
  assert.equal(monthlyGrid(data,s).get('2025-03'),2);
  assert.equal(peerRows(data,s).find(r => r.team === '0002').views.length,1);
  assert.equal(dateMatch(null,s),false);
});
test('explicit tenure attribution is half-open, independent of franchise aliases; peers use same dates',() => {
  const data = fixture(); data.behavior.tenures = [{id:'gm-a',name:'Same Name',team:'0001',from:'2025-03-05',to:'2025-03-06',evidence:'operator map'}];
  const s = lens({team:'0001',tenure:'gm-a'});
  assert.equal(summarize(data,s).views.length,1);
  assert.equal(peerRows(data,s).find(r => r.team === '0002').views.length,1);
});
test('unknown prior outcomes fail result filters; prior records are subject-relative',() => {
  const data = fixture(); data.events[0].context = {'0001':{lastResult:'L',wins:1,losses:5,ties:0,week:6,recordComplete:true}};
  assert.equal(summarize(data,lens({team:'0001',result:'L',role:'low'})).views.length,1);
  assert.equal(summarize(data,lens({team:'0002',result:'L'})).views.length,0);
  data.events[0].context['0001'].recordComplete = false;
  assert.equal(summarize(data,lens({team:'0001',result:'L',role:'low'})).views.length,0);
});
test('annual trajectory preserves the same calendar slice and excludes other months',() => {
  const data = fixture(); data.events.push(event('april','2025-04-01T10:00:00+00:00',[[player('p')],[pick(2026,1)]]));
  const rows = annualProfile(data,lens({team:'0001',year:2025,month:2}),'0001');
  assert.equal(rows.find(r => r.year === 2025).views.length,2);
  assert.equal(rows.find(r => r.year === 2013).views.length,1);
});
test('package size is encoded inventory only and fails closed on ambiguous consideration',() => {
  const data = fixture();
  assert.equal(packageSize(ledgerView(data.events[0],'0001')),'More assets received');
  assert.equal(packageSize(ledgerView(data.events[0],'0002')),'Fewer assets received');
  assert.equal(packageSize(ledgerView(data.events[2],'0001')),null);
  assert.equal(packageSize(ledgerView(data.events[3],'0001')),null);
  assert.equal(packageShape(ledgerView(data.events[3],'0001')),'unresolved');
});
test('manual and proposal channels remain distinct from completed trades',() => {
  const data = fixture(); data.events.push({...data.events[0],id:'manual',type:'LOAD_ROSTERS',movements:{added:['p']},sides:[],flags:['manual-roster-adjustment']});
  data.events.push({...data.events[0],id:'offer',type:'TRADE_PROPOSAL'});
  assert.equal(summarize(data,lens({category:'manual',asset:'pick',round:'1',shape:'mixed',partner:'0002'})).views.length,1);
  assert.equal(summarize(data,lens({category:'proposals'})).views.length,1);
  assert.equal(summarize(data,lens({category:'trade'})).views.length,3);
});
function week(n,kickoff,safeAt,starter = false) {return {season:2025,week:n,team:'0001',kickoff,safeAt,result:'L',starters:starter ? ['q'] : [],bench:starter ? [] : ['q'],gap:2};}
test('usage never turns incomplete follow-up into non-use; exit interrupts observation',() => {
  const data = fixture(); data.events = [data.events[1]];
  data.behavior.weeks = [week(1,'2025-09-01T00:00:00+00:00','2025-09-06T00:00:00+00:00',true)];
  let a = acquisitions(data,lens({team:'0001'}),'0001')[0]; assert(a.start); assert(a.censored);
  data.behavior.weeks.push(week(2,'2025-09-08T00:00:00+00:00','2025-09-13T00:00:00+00:00'),week(3,'2025-09-15T00:00:00+00:00','2025-09-20T00:00:00+00:00'));
  a = acquisitions(data,lens({team:'0001'}),'0001')[0]; assert.equal(a.censored,false);
  data.events.push(event('exit','2025-09-07T10:00:00+00:00',[[player('q')],[pick(2026,2)]]));
  a = acquisitions(data,lens({team:'0001'}),'0001')[0]; assert(a.censored); assert.equal(a.observed.length,1);
});
test('missing team-week is censored, not replaced by a later observation',() => {
  const data = fixture(); data.events = [data.events[1]];
  data.behavior.weeks = [week(1,'2025-09-01T00:00:00+00:00','2025-09-06T00:00:00+00:00',true),
    {...week(2,'2025-09-08T00:00:00+00:00','2025-09-13T00:00:00+00:00'),team:'0002'},
    week(3,'2025-09-15T00:00:00+00:00','2025-09-20T00:00:00+00:00'),
    week(4,'2025-09-22T00:00:00+00:00','2025-09-27T00:00:00+00:00')];
  const a = acquisitions(data,lens({team:'0001'}),'0001')[0];
  assert.equal(a.future.length,3); assert.equal(a.observed.length,2); assert(a.censored);
});
test('verified tenure cannot credit a successor’s starts or exits, or a crossing week',() => {
  const data = fixture(); data.events = [data.events[1],event('later-exit','2025-09-20T00:00:00+00:00',[[player('q')],[pick(2026,2)]])];
  data.behavior.tenures = [{id:'a',name:'Owner A',team:'0001',from:'2025-03-01',to:'2025-09-12',evidence:'operator map'}];
  data.behavior.weeks = [week(1,'2025-09-01T00:00:00+00:00','2025-09-06T00:00:00+00:00',true),
    week(2,'2025-09-08T00:00:00+00:00','2025-09-13T00:00:00+00:00',true),
    week(3,'2025-09-15T00:00:00+00:00','2025-09-20T00:00:00+00:00',true)];
  const s = lens({team:'0001',tenure:'a'}), a = acquisitions(data,s,'0001')[0];
  assert.equal(a.observed.length,1); assert(a.censored); assert.equal(a.exit,undefined);
  assert.equal(weeklySelection(data,s).length,1);
});
test('post-result exposure crosses calendar boundaries and is clipped by next result',() => {
  const data = fixture(); data.behavior.weeks = [week(1,'2025-08-28T00:00:00+00:00','2025-08-31T12:00:00+00:00'),week(2,'2025-09-02T00:00:00+00:00','2025-09-04T12:00:00+00:00')];
  const rows = resultWindows(data,lens({team:'0001',year:2025,month:8}),'0001');
  const loss = rows.find(r => r.result === 'L'); assert.equal(loss.windows,2); assert.equal(loss.dates,10.5);
  assert.equal(weeklySelection(data,lens({team:'0001',year:2025,month:8})).length,1);
});
test('CSV receipts preserve exact directions, flags and proxy rather than infer a price',() => {
  const data = fixture(), s = lens({team:'0001',policy:'all'}), row = csvRows(selection(data,s),s)[1];
  assert.equal(row[6],'0001'); assert.equal(row[7],'0002'); assert.equal(row[9],'FP_0002_2026_1; FP_0002_2026_2');
  const portable = csvRows(selection(data,s),s,[{file:'raw/2025/transactions.json'}])[1];
  assert(portable.at(-1).includes('raw/2025/transactions.json'));
  const old = csvRows([ledgerView(data.events[3],'0001')],s)[1]; assert.equal(old[11],''); assert.equal(old[12],'');
  const draftEvent = {...data.events[0],type:'DRAFT',sides:[],teams:['0001'],players:['q'],round:1,pick:12};
  const [headers,draft] = csvRows([ledgerView(draftEvent,'0001')],s);
  assert.equal(draft[11],''); assert.equal(draft[headers.indexOf('Observed player IDs')],'q');
  assert.equal(draft[headers.indexOf('Draft selection')],12);
});
