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
console.log(JSON.stringify({ties,knifeEdges:margins,minimum},null,2));
if(margins.length)throw new Error('non-identity decision margins below 1e-6; see named rows above');
