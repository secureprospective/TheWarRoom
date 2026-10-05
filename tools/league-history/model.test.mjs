import {test} from 'node:test';
import assert from 'node:assert/strict';
import {filterEvents,monthlyCounts,matchesCategory,csv,heatLevel,playerLabel} from './model.mjs';

const events = [
  {id:'1',type:'TRADE',date:'2025-07-01T23:00:00+00:00',season:2024,teams:['0001','0025'],players:['0835'],picks:3,pickNote:false},
  {id:'2',type:'TRADE_PROPOSAL',date:'2025-07-02T00:00:00+00:00',teams:['0001','0025'],players:['0835'],picks:4},
  {id:'3',type:'TRADE',date:'2025-08-02T00:00:00+00:00',teams:['0001','0002'],players:['1234'],picks:0,pickNote:true},
  {id:'4',type:'DRAFT',date:'2026-07-01T00:00:00+00:00',teams:['0025'],players:['1234'],picks:0},
  {id:'5',type:'TRADE',date:null,teams:['0002'],players:[],picks:1}
];

test('completed pick trades exclude proposals and comment-only references',() => {
  assert.deepEqual(filterEvents(events,{category:'picks'}).map(e => e.id),['1','5']);
  assert.deepEqual(filterEvents(events,{category:'notes'}).map(e => e.id),['3']);
  assert.equal(matchesCategory(events[1],'trades'),false);
});
test('entity, inclusive UTC dates and calendar drilldown compose',() => {
  const filters = {category:'all',team:'0025',player:'0835',from:'2025-07-01',to:'2025-07-01',year:2025,month:6,day:1};
  assert.deepEqual(filterEvents(events,filters,true).map(e => e.id),['1']);
  assert.deepEqual(filterEvents(events,{...filters,month:7},true),[]);
  assert.deepEqual(filterEvents(events,{...filters,from:'2025-08-01'}),[]);
  assert.deepEqual(filterEvents(events,{category:'all',month:6},true).map(e => e.id),['1','2','4']);
});
test('heatmap aggregates actual dates and counts assets separately from deals',() => {
  const filtered = filterEvents(events,{category:'picks'});
  assert.equal(monthlyCounts(filtered,'picks').get('2025-07'),1);
  assert.equal(monthlyCounts(filtered,'assets').get('2025-07'),3);
  assert.equal(monthlyCounts(filtered,'picks').has('2024-07'),false);
});
test('CSV quotes correctly and prevents spreadsheet formula injection',() => {
  assert.equal(csv([['name','comma, quote"','line\nnext','=HYPERLINK("x")',' @SUM(1)']]),'"name","comma, quote""","line\nnext","\'=HYPERLINK(""x"")","\' @SUM(1)"');
});
test('heat levels remain bounded and IDs remain strings',() => {
  assert.equal(heatLevel(0,100),0);
  assert.equal(heatLevel(100,100),5);
  assert.equal(heatLevel(1,100),1);
  assert.equal(playerLabel({name:'Allen, Josh',position:'QB',id:'0835'}),'Josh Allen · QB · 0835');
});
