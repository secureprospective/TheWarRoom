import {readFileSync, writeFileSync, mkdirSync} from 'node:fs';
import {resolve, join} from 'node:path';
import {pathToFileURL} from 'node:url';
import {createHash} from 'node:crypto';
import {gunzipSync, gzipSync} from 'node:zlib';

if (process.version !== 'v24.21.0') throw new Error(`Node v24.21.0 required; got ${process.version}`);
const lab = resolve(process.argv[2] || '');
if (!process.argv[2]) throw new Error('usage: node tools/labparity/gen-goldens.mjs LAB');
const enginePath = join(lab, 'app/js/engine.js');
const E = await import(pathToFileURL(enginePath));
const draftPath = join(lab, 'app/js/draftlab.js');
const D = await import(pathToFileURL(draftPath));
const {decode, PHASES} = await import(pathToFileURL(join(lab, 'app/js/model.js')));
const bytes = gunzipSync(readFileSync('internal/lab/testdata/lab.json.gz'));
const sha = b => createHash('sha256').update(b).digest('hex');
const raw = JSON.parse(bytes), data = decode(raw);
const header = {dataSHA: sha(bytes), engineSHA: sha(readFileSync(enginePath)), node: process.version};
const fit = f => f && ({prices: Object.fromEntries(f.prices), kinds: [...f.prices.keys()], covariance: [...f.prices.keys()].map(a => [...f.prices.keys()].map(b => f.cov(a,b))), perAsset:f.perAsset, trades:f.trades});
const clean = value => JSON.parse(JSON.stringify(value, (_,v) => v instanceof Map ? Object.fromEntries(v) : v));
const row = r => ({...r, halves:r.halves?.map(h=>h && row(h)), trend:r.trend || undefined, holds:r.holds || undefined, cheapest:r.cheapest || undefined, dearest:r.dearest || undefined});
const market = m => ({...m, phases:m.phases ?? null, rows:m.rows?.map(row) ?? null});
const phases = PHASES.map(p=>p.id);
const scopes = [
 ['all',{from:2013,to:2026}], ['measured',{from:2017,to:2026}],
 ['last3',{from:2024,to:2026}], ['early',{from:2013,to:2016}],
 ['pick-only',{from:2017,to:2026,assets:['pick1:next','pick1:later','pick2:next','pick2:later','pick3:next','pick3:later']}],
 ['age',{from:2017,to:2026,byAge:true}], ['age-last3',{from:2024,to:2026,byAge:true}],
 ['season',{from:2017,to:2026,phases:['early','midseason','deadline'],windows:['season']}],
 ['winter',{from:2017,to:2026,phases:['winter']}], ['empty',{from:2030,to:2030}],
];
let minimum={margin:Infinity,row:''};
const margins=[], ties=[];
function margin(value,bound,label,identity=false,displayOrder=false) {
 const m=Math.abs(value-bound);
 if(displayOrder && m<1e-12) { ties.push({row:label,margin:m,status:'tie (stable input order)'}); return; }
 if(m===0 && identity)return; // Only explicitly identified structural identities are exempt.
 if(m<minimum.margin)minimum={margin:m,row:label};
 if(m<1e-6)margins.push({row:label,margin:m});
}
function orderingMargins(items,label,number) {
 const ordered=items.filter(x=>(x.kind??x.id)!=='player:unknown').map(x=>({id:x.kind??x.id,value:number(x)})).filter(x=>Number.isFinite(x.value)).sort((a,b)=>a.value-b.value);
 for(let i=1;i<ordered.length;i++)margin(ordered[i].value,ordered[i-1].value,`${label}:${ordered[i-1].id}/${ordered[i].id}`,false,true);
}
function scan(value,path) {
 if(!value || typeof value!=='object')return;
 if(Array.isArray(value) && value.length && value[0]?.kind && !path.endsWith('.halves')) {
  if(value[0].price)orderingMargins(value,path+'.price-order',x=>x.price.price);
  if(value[0].range)orderingMargins(value,path+'.range-order',x=>x.range?.value);
  if(value[0].ratio!=null)orderingMargins(value,path+'.ratio-order',x=>x.ratio);
  if(value[0].wins!=null)orderingMargins(value,path+'.wins-order',x=>x.wins);
 }
 if(value.prices && !Array.isArray(value.prices))orderingMargins(Object.entries(value.prices).map(([kind,p])=>({kind,...p})),path+'.price-order',x=>x.price);
 if(value.price!=null && typeof value.price==='number')margin(value.price,0,path+'.price>0');
 if(value.ratio!=null)margin(value.ratio,0,path+'.ratio>0');
 if(value.spread!=null){margin(value.spread,2.5,path+'.spread<=2.5');margin(value.spread,6,path+'.spread<=6');}
 if(value.cost && typeof value.cost==='object'){margin(value.cost.lo,1,path+'.cost.lo>1');margin(value.cost.hi,1,path+'.cost.hi<1');}
 if(value.byDelivered!=null){margin(value.lo,1,path+'.lo>1');margin(value.hi,1,path+'.hi<1');}
 if(value.clear!=null)margin(value.hi,1,path+'.clock.hi<1');
 if(value.price!=null && typeof value.price==='number' && value.se!=null){margin(value.se,0.5*Math.max(value.price,0.2),path+'.time.se');margin(value.se,0.6*Math.max(value.price,0.2),path+'.stretch.se');}
 for(const [k,v] of Object.entries(value))scan(v,`${path}.${k}`);
}
function timingMargins(items,label,field) {
 const known=items.filter(x=>x[field]!=null);
 for(let i=0;i<known.length;i++)for(let j=i+1;j<known.length;j++) {
  const a=known[i],b=known[j],av=field==='paid'?a.paid.value:a.cost,bv=field==='paid'?b.paid.value:b.cost;
  const ase=field==='paid'?a.paid.se:a.se,bse=field==='paid'?b.paid.se:b.se;
  margin(Math.abs(av-bv),1.96*Math.hypot(ase,bse),`${label}:${a.phase||a.window}/${b.phase||b.window}`);
 }
}
mkdirSync('internal/lab/testdata/goldens',{recursive:true});
const writeGolden = (name, text) => writeFileSync(`internal/lab/testdata/goldens/${name}.json.gz`, gzipSync(text, {level:9,mtime:0}));
for(const [name,scope] of scopes) {
 const keep=E.inScope(scope);
 const vt=E.valuedTrades(data,t=>keep(t)&&(name!=='pick-only'||t.legs.every(l=>l.xClass?.startsWith('pick'))));
 const f=E.fitPrices(vt), stretches=E.stretchPrices(data,scope,phases);
 const m=market(E.leagueMarket(data,scope));
 const pl=E.priceList(data,scope);
 if(pl)for(const r of pl.rows)delete r.evidence;
 const kinds=scope.byAge?['LB:young','LB:prime','LB:older','pick1:next','pick1:later','missing']:['LB','QB','pick1:next','pick1:later','missing'];
 const profiles=Object.fromEntries(kinds.map(k=>[k,E.assetProfile(data,scope,k,stretches)]));
 // Missing-kind result has only kind and n in the reference.
 for(const k of kinds)if(profiles[k] && profiles[k].trades==null)profiles[k]={kind:k,n:0,moved:0,trades:0,paid:null,delivered:null,when:null,against:null,back:null,buyers:{mine:{contender:0,middle:0,rebuilder:0},all:{contender:0,middle:0,rebuilder:0},n:0}};
 const records=data.delivered.filter(r=>keep(r.t));
 const deliveries={},ranges={},grades={};
 for(const k of [...f.prices.keys()]) {
  const rs=records.filter(r=>r.class===k);deliveries[k]=E.deliveredRatio(rs);ranges[k]=E.perDelivered(f.prices.get(k),deliveries[k]);grades[k]=E.evidence(ranges[k]);
  margin(rs.reduce((s,r)=>s+r.expected,0),0.5,`${name}:delivered:${k}:expected`);
 }
 const pc=E.tradesByPhase(data,scope);
 const values={valuedTrades:vt.map(v=>({id:v.t.id,sides:v.sides})),fitPrices:fit(f),deliveredRatio:deliveries,perDelivered:ranges,evidence:grades,priceList:pl,meanVar:E.meanVar(vt.flatMap(v=>v.sides.flat().map(a=>a.value))),groupOf:Object.fromEntries(f.prices.keys().map(k=>[k,E.groupOf(k)])),ageOf:Object.fromEntries(f.prices.keys().map(k=>[k,E.ageOf(k)])),assetOf:Object.fromEntries(f.prices.keys().map(k=>[k,E.assetOf(k)])),inScope:data.trades.filter(keep).map(t=>t.id),leagueMarket:m,tradesByPhase:{...pc,order:[...pc.phases.keys()]},pickClock:E.pickClock(data,scope),contenderTrades:E.contenderTrades(data,scope),stretchPrices:[...stretches].map(([phase,sx])=>({...sx,phase,fit:fit(sx.fit)})),assetProfile:profiles};
 scan(clean(values),name);
 for(const r of m.rows||[])timingMargins(r.times,`${name}:${r.kind}:times`,'cost');
 for(const [k,p] of Object.entries(profiles))if(p?.when)timingMargins(p.when,`${name}:${k}:when`,'paid');
 writeGolden(name,JSON.stringify({header:{...header,scope,name},compilerPrices:raw.market.classes.map(c=>({id:c.id,price:c.price})),values},(_,v)=>v instanceof Map?Object.fromEntries(v):v,2)+'\n');
}
// Threshold fixture guarantees a value/null cell even if real scopes lack n=7.
writeGolden('boundaries',JSON.stringify({header,records:Array.from({length:7},()=>({expected:1,delivered:1})),deliveredRatio:E.deliveredRatio(Array.from({length:7},()=>({expected:1,delivered:1})))},null,2)+'\n');
const draftPick = p => ({id:p.id, segment:p.segment, nfl:p.nfl, judged:p.judged, got:p.got, expected:p.expected});
const cell = c => Object.fromEntries(Object.entries(c).filter(([k,v])=>k!=='picks' && v!==undefined && (v!==null || k==='ratio') && (v!==false || !['best','worst'].includes(k))));
const position = p => ({...p, all:p.all.map(draftPick), where:p.where.map(cell), when:p.when.map(cell), nfl:p.nfl.map(cell), age:p.age.map(cell)});
const draftScopes = [
 ['measured',{from:2017,to:2026,seasons:3}], ['two',{from:2017,to:2026,seasons:2}],
 ['four',{from:2017,to:2026,seasons:4}], ['early',{from:2013,to:2016,seasons:3}],
 ['recent',{from:2024,to:2026,seasons:3}], ['round3',{from:2017,to:2026,seasons:3,segments:['r3']}],
 ['nfl1',{from:2017,to:2026,seasons:3,nfl:['n1']}], ['empty',{from:2100,to:2101,seasons:3}],
];
const groups=['QB','RB','WR','TE','PK','DL','LB','DB'];
const draftFixtures=draftScopes.map(([name,scope])=>{
 const picks=D.draftScope(data,scope), curve=D.slotCurve(picks);
 const positions=Object.fromEntries(groups.map(g=>[g,position(D.positionDraft(picks,g))]));
 const map=D.draftMap(picks,groups.map(id=>({id}))).map(r=>({...r,when:r.when.map(cell),where:r.where.map(c=>({segment:c.segment,n:c.n,reading:c.ratio==null?null:Object.fromEntries(Object.entries(c).filter(([k])=>!['segment'].includes(k)))}))}));
 const ageMarket=E.leagueMarket(data,{from:Math.max(2017,scope.from),to:scope.to,byAge:true});
 const buys=Object.fromEntries(groups.map(g=>{
  const rows=[`${g}:young`,`${g}:prime`,`${g}:older`,'pick1:next','pick2:next','pick3:next'].map(k=>ageMarket.rows?.find(r=>r.kind===k)).filter(r=>r?.cost);
  const ages=rows.filter(r=>r.kind.startsWith(g+':')).sort((a,b)=>a.cost.value-b.cost.value);
  const rounds=new Map();for(const p of picks.filter(p=>p.group===g && p.judged)){const k=`pick${Math.min(3,p.round)}:next`;rounds.set(k,(rounds.get(k)||0)+1);}
  const usual=[...rounds].sort((a,b)=>b[1]-a[1])[0]?.[0];
  const via=rows.find(r=>r.kind===usual)||rows.find(r=>r.kind.startsWith('pick'));
  return [g,{rows:rows.map(row),cheapest:ages[0]?.kind??null,via:via?.kind??null}];
 }));
 const values={draftScope:picks.map(draftPick),vsSlot:D.vsSlot(picks),bySeason:D.bySeason(picks).map(cell),slotCurve:curve,playsLike:[null,0,1,10].map(v=>D.playsLike(curve,v)),hitRate:D.hitRate(picks,picks),positionDraft:positions,draftMap:map,draftBuy:buys};
 for(const [g,p] of Object.entries(positions)) {
  const mine=picks.filter(x=>x.group===g && x.judged);
  if(p.hits) {
   const base=new Map(D.SEGMENTS.map(seg=>{const xs=picks.filter(x=>x.judged && x.segment===seg.id);return [seg.id,xs.length?xs.filter(x=>['top','starter'].includes(x.best)).length/xs.length:null]}));
   const expected=mine.reduce((t,x)=>t+base.get(x.segment),0),v=mine.reduce((t,x)=>{const b=base.get(x.segment);return t+b*(1-b)},0),hits=mine.filter(x=>['top','starter'].includes(x.best)).length;
   margin(Math.abs(hits-expected),1.96*Math.sqrt(v),`draft:${name}:${g}:hit-tone`,v===0 && hits===expected);
  }
  for(const c of [...p.where,...p.nfl,...p.age])if(c.like && c.ratio!=null){
   const target=curve[c.at]*c.ratio;
   for(const o of [c.like-1,c.like])if(o>0 && curve[o]!=null)margin(target,curve[o],`draft:${name}:${g}:playsLike:${c.at}/${o}`,c.ratio===1);
  }
  for(const [axis,cs] of Object.entries({where:p.where,when:p.when,nfl:p.nfl,age:p.age})) {
   for(const [i,c] of cs.entries()) {
    const cohort=axis==='where'?mine.filter(x=>x.segment===c.segment):axis==='nfl'?mine.filter(x=>x.nfl===c.nfl):axis==='age'?mine.filter(x=>x.age!=null && (c.age==='a21'?x.age<22:c.age==='a22'?x.age>=22 && x.age<23:x.age>=23)):picks.filter(x=>x.group===g && x.wins[c.season-1]!=null && x.slotWins[c.season-1]!=null);
    const expected=cohort.reduce((sum,x)=>sum+(axis==='when'?x.slotWins[c.season-1]:x.expected),0);
    margin(expected,0.5,`draft:${name}:${g}:${axis}:${i}:slot-total`);
   }
   for(const [i,c] of cs.entries())if(c.ratio!=null){
    margin(c.lo,1,`draft:${name}:${g}:${axis}:${i}:lo>1`);margin(c.hi,1,`draft:${name}:${g}:${axis}:${i}:hi<1`);
   }
   const known=cs.filter(c=>c.ratio!=null);
   for(let i=0;i<known.length;i++)for(let j=i+1;j<known.length;j++) {
    const a=known[i],b=known[j];margin(Math.abs(a.ratio-b.ratio),Math.hypot((a.hi-a.lo)/3.92,(b.hi-b.lo)/3.92)*1.96,`draft:${name}:${g}:${axis}:${i}/${j}:extreme`);
   }
  }
 }
 return {name,scope,values};
});
const boundaryPicks=D.draftScope(data,{from:2017,to:2017,seasons:3}).filter(p=>p.judged).slice(0,7);
writeGolden('draft',JSON.stringify({header:{...header,draftSHA:sha(readFileSync(draftPath))},fixtures:draftFixtures,boundary:{picks:boundaryPicks.map(draftPick),hitRate:D.hitRate(boundaryPicks,boundaryPicks)}},null,2)+'\n');
const teamsPath=join(lab,'app/js/teams.js');
const T=await import(pathToFileURL(teamsPath));
// Full side objects appear once (tradeSides); everywhere else a side is the compact
// "tradeID:fid" reference, so the golden stays a fraction of the raw facts.
const side=s=>({tradeID:s.t.id,fid:s.fid,other:s.other,season:s.season,window:s.window,
 phase:s.phase,tier:s.tier||'',otherTier:s.otherTier||'',rank:s.rank,now:s.now,real:s.real,
 players:s.players,picks:s.picks,picksIn:s.picksIn,picksOut:s.picksOut,
 ageIn:s.ageIn,ageOut:s.ageOut,got:s.got.map(l=>({class:l.xClass,value:l.xValue,age:l.age})),
 gave:s.gave.map(l=>({class:l.xClass,value:l.xValue,age:l.age})),posture:s.posture});
const ref=s=>`${s.t.id}:${s.fid}`;
const teamClean=value=>JSON.parse(JSON.stringify(value,(k,v)=>{
 if(k==='sides')return v.map(ref);
 if(k==='base' && v===null)return undefined;
 return v;
}));
function teamMargins(value,path) {
 if(!value || typeof value!=='object')return;
 if(value.tone && value.lo!=null) {
  const base=value.p!=null ? (value.base??null) : 0;
  if(base!=null) {
   const identity=value.mean===0 && value.lo===0 && value.hi===0;
   margin(value.lo,base,path+'.lo',identity);margin(value.hi,base,path+'.hi',identity);
  }
 }
 for(const [k,v] of Object.entries(value))teamMargins(v,path+'.'+k);
}
const teamScopes=[
 ['all',{from:2013,to:2026}],['measured',{from:2017,to:2026},true],
 ['recent',{from:2024,to:2026}],['early',{from:2013,to:2016}],
 ['rebuild-season',{from:2017,to:2026,tiers:['rebuild'],windows:['season']}],
 ['later',{from:2017,to:2026,postures:['later']}],
 ['one-team',{from:2017,to:2026,team:'0001'}],
 ['empty',{from:2100,to:2101}],
];
const franchiseIds=()=>[...data.franchises.keys()];
// The measured scope profiles every franchise; other scopes a fixed few plus the
// busiest, so scope filtering is proven without re-serialising every profile.
const profileIds=(sides,all)=>{
 const ids=franchiseIds();
 if(all)return ids;
 const count=id=>sides.filter(s=>s.fid===id).length;
 const busiest=ids.reduce((best,id)=>count(id)>count(best)?id:best,ids[0]);
 return [...new Set([...ids.slice(0,4),busiest])];
};
const teamFixtures=teamScopes.map(([name,scope,all])=>{
 const league=T.sidesInScope(data,{from:scope.from,to:scope.to}),sides=T.sidesInScope(data,scope);
 const profiles=Object.fromEntries(profileIds(sides,all).map(id=>[id,T.teamProfile(data,sides,league,id)]));
 const trends=T.tierTrends(sides);
 for(const tier of trends.tiers)for(const [id,shift] of Object.entries(tier.shift))if(shift){
  const mine=sides.filter(s=>s.tier===tier.tier),a=mine.filter(s=>s.season<=shift.from[1]),b=mine.filter(s=>s.season>shift.from[1]);
  margin(Math.abs(shift.late-shift.early),1.96*Math.sqrt(shift.early*(1-shift.early)/a.length+shift.late*(1-shift.late)/b.length),`teams:${name}:${tier.tier}:${id}:shift`,shift.early===shift.late && [0,1].includes(shift.early));
 }
 const values=teamClean({sidesInScope:sides.map(ref),behaviour:T.behaviour(sides),againstLeague:T.behaviour(sides,league),tierTrends:trends,tierPairs:T.tierPairs(sides),teamRows:T.teamRows(sides,league),teamProfile:profiles});
 teamMargins(values,'teams:'+name);
 return {name,scope,values};
});
const teamSides=T.tradeSides(data);
for(const s of teamSides){margin(s.players,0.05,`teams:${s.t.id}:${s.fid}:players>EDGE`);margin(s.players,-0.05,`teams:${s.t.id}:${s.fid}:players<-EDGE`);margin(s.picks,0.05,`teams:${s.t.id}:${s.fid}:picks>EDGE`);margin(s.picks,-0.05,`teams:${s.t.id}:${s.fid}:picks<-EDGE`);}
// Rank 0 stands for the reference's null rank (unclassed); the boundaries are pinned at 10/22/23.
const tierOf=[null,1,10,11,22,23,32].map(r=>r==null?'':T.tierOf(r));
writeGolden('teams',JSON.stringify({header:{...header,teamsSHA:sha(readFileSync(teamsPath))},tradeSides:teamSides.map(side),tierOf,fixtures:teamFixtures},null,2)+'\n');
console.log(JSON.stringify({ties,knifeEdges:margins,minimum},null,2));
if(margins.length)throw new Error('non-identity decision margins below 1e-6; see named rows above');
