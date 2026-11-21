/* The mock v1 "Broadcast" field, ported from .scratch/TheWarRoom-mobile-mock-v1.html (VIEW, SKIN v1:
   night turf, HUD, sticks, grain, border, big play "e", scoring plate). A real 3D field built with CSS 3D
   transforms; every moving piece is a transform or opacity change through the Web Animations API or one
   requestAnimationFrame tween. Imperative on purpose: React owns the host element, this owns its insides. */
import type { FieldPlay, Side } from './livePlays';
import { clubColours, type Club } from './liveClubs';

export type FieldTeams = Record<Side, Club>;
export type FieldGame = { poss: Side; x: number; lat: number; down: number; dist: number; driveStart: number };
type Pt = { x: number; y: number; z: number };
type Opts = KeyframeAnimationOptions;

const clamp = (v: number, a: number, b: number) => Math.max(a, Math.min(b, v));
const wait = (ms: number) => new Promise<void>(r => setTimeout(r, ms));
const ease = { lin: (t: number) => t, out: (t: number) => 1 - Math.pow(1 - t, 3) };
const ORD = ['1ST', '2ND', '3RD', '4TH'];
const GLYPH = '▓▒░#/\\<>01';
const pick = (s: string) => s[Math.floor(Math.random() * s.length)];
const t3 = (x: number, y: number, z = 0) => `translate3d(${x.toFixed(1)}px,${y.toFixed(1)}px,${z.toFixed(1)}px)`;
function el(tag: string, cls?: string, html?: string): HTMLElement {
  const e = document.createElement(tag);
  if (cls) e.className = cls;
  if (html != null) e.innerHTML = html;
  return e;
}
const esc = (s: string) => s.replace(/[&<>"]/g, c => `&#${c.charCodeAt(0)};`);
export const surnameOf = (n: string) => (n.split(/[ .]/).filter(Boolean).pop() ?? n).toUpperCase();

/* camera and world constants from the mock: world pixels per yard, aprons in yards, camera tilt */
const U = 10, APX = 34, APY = 6, APN = 4, FW = 53.3, TILT = 66, PAN: [number, number] = [9, 91], LEAD = 3;
const WW = (120 + 2 * APX) * U, WH = (FW + APY + APN) * U, FT_ = APY * U, FH = FW * U;
const wx = (yd: number) => (APX + 10 + yd) * U;
const wy = (lat: number) => FT_ + FH / 2 + lat * 22 * U;
const ydOf = (px: number) => px / U - APX - 10;
const TURF = ['hsl(142,34%,10.5%)', 'hsl(142,36%,13.5%)', 'hsl(150,20%,3%)'];
const BIG = 20, FIT = 276, WALL = 170, PYW = 9, PYH = 30;
const runSpeed = (t: number) => 1 - Math.pow(1 - t, 1.7);

export type FieldView = ReturnType<typeof createFieldView>;

/** createFieldView builds the field inside host and returns the mock's interface: sync, play, banner. */
export function createFieldView(host: HTMLElement, teams: FieldTeams, reducedMotion: () => boolean) {
  const reduced = reducedMotion;
  let alive = true;
  const tween = (ms: number, fn: (k: number) => void, e: (t: number) => number = ease.out) => {
    if (reduced() || ms <= 0 || !alive) { fn(1); return Promise.resolve(); }
    return new Promise<void>(res => {
      const t0 = performance.now();
      const step = (now: number) => {
        if (!alive) return res();
        const t = clamp((now - t0) / ms, 0, 1); fn(e(t));
        if (t < 1) requestAnimationFrame(step); else res();
      };
      requestAnimationFrame(step);
    });
  };
  /* Web Animations wrapper: runs on the compositor; with reduced motion it lands on the last keyframe */
  const anim = (node: Element, frames: Keyframe[], opts: number | Opts): Promise<void> => {
    const o: Opts = typeof opts === 'number' ? { duration: opts } : { ...opts };
    if (reduced()) { o.duration = 0; o.delay = 0; }
    o.fill = o.fill || 'forwards'; o.easing = o.easing || 'cubic-bezier(.2,.8,.2,1)';
    if (typeof node.animate !== 'function') return Promise.resolve();
    return node.animate(frames, o).finished.then(() => undefined, () => undefined);
  };
  const hold = (ms: number) => reduced() || !alive ? Promise.resolve() : wait(ms);
  const TEAM = { L: clubColours(teams.L), R: clubColours(teams.R) };
  const name = (k: Side) => TEAM[k].name;

  function turf() {
    const [c0, c1, apron] = TURF;
    let s = '<defs><pattern id="grA" width="5" height="5" patternUnits="userSpaceOnUse"><path d="M1 0V5" '
      + 'stroke="#fff" stroke-opacity=".055" stroke-width="1.4"/></pattern><pattern id="grB" width="5" '
      + 'height="5" patternUnits="userSpaceOnUse"><path d="M0 1H5" stroke="#000" stroke-opacity=".16" '
      + 'stroke-width="1.4"/></pattern></defs>';
    for (let yd = 0; yd < 100; yd += 5) s += `<rect x="${wx(yd)}" y="${FT_}" width="${5 * U}" height="${FH}" `
      + `fill="${(yd / 5) % 2 ? c0 : c1}"/>`;
    for (let yd = 0; yd < 100; yd += 5) s += `<rect x="${wx(yd)}" y="${FT_}" width="${5 * U}" height="${FH}" `
      + `fill="url(#gr${(yd / 5) % 2 ? 'A' : 'B'})"/>`;
    /* the club's own colour fills its end zone; the lettering is black or white, whichever reads better */
    const ez = (k: Side, yd: number, rot: number) => {
      const t = TEAM[k], size = Math.min(74, 470 / (t.name.length * .74));
      return `<rect x="${wx(yd)}" y="${FT_}" width="${10 * U}" height="${FH}" fill="${t.fillc}"/>`
        + `<rect x="${wx(yd) + 9}" y="${FT_ + 9}" width="${10 * U - 18}" height="${FH - 18}" fill="none" `
        + `stroke="${t.trimc}" stroke-width="5"/><text class="ezt" fill="${t.onc}" font-size="${size}" `
        + `transform="translate(${wx(yd + 5)} ${FT_ + FH / 2}) rotate(${rot})" dy="${size * .34}">`
        + `${esc(t.name)}</text>`;
    };
    s += ez('L', -10, -90) + ez('R', 100, 90);
    for (let yd = 0; yd <= 100; yd += 5) s += `<rect class="yl" data-yd="${yd}" x="${wx(yd) - (yd % 100 ? 1.5 : 3)}" `
      + `y="${FT_}" width="${yd % 100 ? 3 : 6}" height="${FH}" fill="#fff" opacity="${yd % 10 ? .5 : .85}"/>`;
    const dash = (y: number, w: number) => `<path d="M${wx(0)} ${y}H${wx(100)}" stroke="#fff" stroke-opacity=".7" `
      + `stroke-width="${w}" stroke-dasharray="2 8" fill="none"/>`;
    for (const off of [-4.4, 4.4]) s += dash(FT_ + FH / 2 + off * U, 12);
    for (const y of [FT_ + 5, FT_ + FH - 5]) s += dash(y, 10);
    for (let yd = 10; yd <= 90; yd += 10) {
      const n = yd <= 50 ? yd : 100 - yd;
      s += `<text class="num" x="${wx(yd)}" y="${FT_ + FH - 9.5 * U}">${n}</text>`
        + `<text class="num far" x="${wx(yd)}" y="${FT_ + 14 * U}">${n}</text>`;
    }
    s += `<g class="logo" transform="translate(${wx(50)} ${FT_ + FH / 2})"><circle r="62" /><circle r="50"/>`
      + '<text y="15">TWR</text></g>';
    const sbd = (x: number, y: number, w: number, h: number) =>
      `<rect class="sbd" x="${x}" y="${y}" width="${w}" height="${h}"/>`;
    const bx = wx(-10) - 2 * U;
    return `<svg class="v2-turf" width="${WW}" height="${WH}" viewBox="0 0 ${WW} ${WH}">`
      + `<rect width="${WW}" height="${WH}" fill="${apron}"/>${s}`
      + sbd(bx, FT_ - 2 * U, 124 * U, 2 * U) + sbd(bx, FT_ + FH, 124 * U, 2 * U) + sbd(bx, FT_, 2 * U, FH)
      + sbd(wx(110), FT_, 2 * U, FH)
      + `<rect x="${wx(-10)}" y="${FT_}" width="${120 * U}" height="${FH}" fill="none" stroke="#fff" `
      + 'stroke-width="6"/></svg>';
  }
  /* a pylon: a real orange box standing on the corner, faces shaded so it reads as a solid as the camera pans */
  const pylon = (x: number, y: number) => `<div class="v2-py" style="transform:${t3(x, y)}">`
    + `<i class="f" style="transform:translateY(${PYW / 2}px) rotateX(-90deg)"></i>`
    + `<i class="b" style="transform:translateY(${-PYW / 2}px) rotateX(-90deg)"></i>`
    + `<i class="s" style="transform:translateX(${-PYW / 2}px) rotateZ(90deg) rotateX(-90deg)"></i>`
    + `<i class="s" style="transform:translateX(${PYW / 2}px) rotateZ(90deg) rotateX(-90deg)"></i>`
    + `<i class="t" style="transform:translateZ(${PYH}px)"></i></div>`;
  const post = (k: Side, yd: number) => `<div class="v2-post" data-side="${k}" style="transform:${t3(wx(yd), wy(0))} `
    + `rotateX(-90deg) rotateY(${k === 'L' ? 62 : -62}deg)"><i class="stem"></i><i class="fork"></i></div>`;
  const zone = (cls: string, extra = '') => `<div class="${cls}" style="top:${FT_}px;height:${FH}px;${extra}"></div>`;
  const sy = FT_ + FH + 1.1 * U;

  host.innerHTML = `<div class="v2-stage skin-night" id="v2stage" style="--wh:${WALL}px">`
    + `<div class="v2-cam" id="v2cam"><div class="v2-world" id="v2world" style="width:${WW}px;height:${WH}px">`
    + `<div class="v2-wall" style="width:${WW}px"><div class="crowd"></div>`
    + `<div class="board" id="v2board" style="left:${wx(50)}px"></div><div class="ribbon" id="v2rib"></div></div>`
    + turf()
    + `<div class="v2-tint" id="v2tint" style="left:${wx(0)}px;top:${FT_}px;width:${100 * U}px;height:${FH}px"></div>`
    + zone('v2-zone drv', `width:${100 * U}px`).replace('<div', '<div id="v2drv"')
    + zone('v2-zone togo', `width:${100 * U}px`).replace('<div', '<div id="v2togo"')
    + zone('v2-ezfx', `left:${wx(-10)}px;width:${10 * U}px`).replace('<div', '<div id="v2ezfxL"')
    + zone('v2-ezfx', `left:${wx(100)}px;width:${10 * U}px`).replace('<div', '<div id="v2ezfxR"')
    + '<div class="v2-paint" id="v2paint"></div>'
    + zone('v2-band los').replace('<div', '<div id="v2los"')
    + zone('v2-band first').replace('<div', '<div id="v2first"')
    + `<div class="v2-dd sticks" id="v2dd" style="top:${FT_ + FH - 7.2 * U}px"><b></b></div>`
    + `<div class="v2-chain sticks" id="v2chain" style="top:${sy}px"></div>`
    + `<div class="v2-pad" id="v2pad" style="top:${sy - 6}px"></div>`
    + '<div class="bb v2-stk sticks" id="v2stk"><div class="bb-in"><div class="dbox"><b id="stkDn"></b></div>'
    + '<i class="slv"></i><i class="tip"></i></div></div>'
    + '<div class="bb v2-stk sticks two" id="v2stk2"><div class="bb-in"><i class="disc"></i><i class="pen">'
    + '<i></i><i></i><i></i><i></i></i><i class="tip"></i></div></div>'
    + post('L', -10) + post('R', 110)
    + [0, -10, 100, 110].flatMap(yd => [FT_ - PYW / 2, FT_ + FH + PYW / 2].map(y => pylon(wx(yd), y))).join('')
    + '<div class="bb v2-mk" id="v2mk"><i class="disc"></i><div class="bb-in"><div class="pin">'
    + '<svg viewBox="0 0 30 20" width="30" height="20"><ellipse cx="15" cy="10" rx="13.5" ry="8.2"/>'
    + '<path d="M8 10h14M12 7.4v5.2M15 7v6M18 7.4v5.2"/></svg><i></i></div></div></div>'
    + '</div></div>'
    + '<div class="v2-fx" id="v2fx"></div><div class="v2-third" id="v2third" hidden></div>'
    + '<div class="v2-hud" id="v2hud"><div class="hc"><i>DOWN</i><b id="hDown"></b></div>'
    + '<div class="hc"><i>TO GO</i><b id="hDist"></b></div><div class="hc w"><i>BALL ON</i><b id="hSpot"></b></div>'
    + '<div class="hc w"><i>TO GOAL</i><b id="hGoal"></b></div></div>'
    + '<div class="v2-cam over" aria-hidden="true"><div class="v2-world" id="v2world2"><div class="v2-up" id="v2up">'
    + '</div></div></div><div class="v1-ban" id="v1ban" hidden></div></div>';

  const q = <T extends Element = HTMLElement>(s: string) => host.querySelector(s) as T;
  const stage = q('#v2stage'), world = q('#v2world'), world2 = q('#v2world2'), cam = q('#v2cam');
  const cam2 = q('.v2-cam.over'), up = q('#v2up'), paint = q('#v2paint'), fx = q('#v2fx'), mk = q('#v2mk');
  const los = q('#v2los'), first = q('#v2first'), dd = q('#v2dd'), drv = q('#v2drv'), togo = q('#v2togo');
  let cur = { x: 0, y: 0 }, focus = 50;
  stage.style.setProperty('--tilt', `${TILT}deg`);

  /* the whole scene scales from one number, so the same field fills a phone or a desktop pane */
  function resize() { stage.style.setProperty('--k', (stage.clientHeight / FIT || 1).toFixed(3)); }
  const teamVars = (k: Side, pre: string) => {
    const t = TEAM[k];
    stage.style.setProperty(`--${pre}c`, t.color); stage.style.setProperty(`--${pre}turf`, t.turf);
    stage.style.setProperty(`--${pre}fill`, t.fillc); stage.style.setProperty(`--${pre}on`, t.onc);
    stage.style.setProperty(`--${pre}trim`, t.trimc);
  };
  /* camera: slide the world under a fixed lens; yd is the yard the lens centres on */
  function look(yd: number, ms = 700, z = 1) {
    focus = clamp(yd, PAN[0], PAN[1]);
    for (const n of [world, world2, cam, cam2]) n.style.transitionDuration = `${reduced() ? 0 : ms}ms`;
    world.style.transform = world2.style.transform = `translate3d(${-wx(focus)}px,0,0)`;
    stage.style.setProperty('--zoom', String(z));
  }
  /* a billboard: stands upright at a spot on the field, facing the lens */
  function bb(x: number, y: number, z: number, html: string, cls = '') {
    const a = el('div', `bb ${cls}`, `<div class="bb-in">${html}</div>`);
    a.style.transform = t3(x, y, z); up.appendChild(a); return a;
  }
  function label(x: number, y: number, text: string, cls = '', z = 46, delay = 0) {
    const a = bb(x, y, z, `<span class="tagl ${cls}">${esc(text)}</span>`, 'lab');
    void anim(a.firstElementChild!.firstElementChild!, [{ transform: 'scale(.2)', opacity: 0 },
      { transform: 'scale(1)', opacity: 1 }], { duration: 280, delay, easing: 'cubic-bezier(.2,1.6,.4,1)',
      fill: 'both' });
    return a;
  }
  /* a stripe painted on the turf from one spot to another */
  function line(x0: number, y0: number, x1: number, y1: number, cls = '') {
    const n = el('div', `gl ${cls}`), len = Math.hypot(x1 - x0, y1 - y0);
    n.style.cssText = `width:${len}px;transform:${t3(x0, y0, 1)} rotate(${Math.atan2(y1 - y0, x1 - x0)}rad)`;
    paint.appendChild(n);
    const inn = el('i'); n.appendChild(inn); return inn;
  }
  const grow = (inn: Element, ms: number, easing = 'cubic-bezier(.45,0,.2,1)') =>
    anim(inn, [{ transform: 'scaleX(0)' }, { transform: 'scaleX(1)' }], { duration: ms, easing });
  const moveMk = (x: number, y: number, ms: number, easing = 'cubic-bezier(.45,0,.2,1)') => {
    const from = t3(cur.x, cur.y); cur = { x, y };
    return anim(mk, [{ transform: from }, { transform: t3(x, y) }], { duration: ms, easing });
  };
  const setMk = (x: number, y: number) => {
    mk.getAnimations?.().forEach(a => a.cancel()); mk.style.transform = t3(x, y); cur = { x, y };
  };
  function cross(x: number, y: number) {
    const c = el('div', 'gx'); c.style.transform = t3(x, y, 1.5); c.innerHTML = '<i></i><i></i>';
    paint.appendChild(c); void anim(c, [{ opacity: 0, scale: '.3' }, { opacity: 1, scale: '1' }], 200);
  }
  /* a hard-edged slash of colour crosses the picture: the cut between one moment and the next */
  function wipe(cls: string, ms = 560) {
    if (reduced()) return;
    const f = el('i', `wp ${cls}`); fx.appendChild(f);
    void anim(f, [{ transform: 'translateX(-140%) skewX(-20deg)' }, { transform: 'translateX(330%) skewX(-20deg)' }],
      { duration: ms, easing: 'cubic-bezier(.6,0,.3,1)' }).then(() => f.remove());
  }
  /* a ball in the air: dots hang along the arc and fill in as the ball passes; the shadow runs along the turf */
  async function flight(a: Pt, b: Pt, peak: number, ms: number, cls = '') {
    const N = 16;
    const at = (t: number): Pt => ({ x: a.x + (b.x - a.x) * t, y: a.y + (b.y - a.y) * t,
      z: a.z + (b.z - a.z) * t + 4 * peak * t * (1 - t) });
    const frames: Keyframe[] = [], sfr: Keyframe[] = [];
    for (let i = 0; i <= N; i++) {
      const t = i / N, p = at(t);
      frames.push({ transform: t3(p.x, p.y, p.z) });
      sfr.push({ transform: t3(p.x, p.y, 1), opacity: .6 - .35 * Math.sin(Math.PI * t) });
    }
    const dots: HTMLElement[] = [];
    for (let i = 1; i < N; i++) {
      const p = at(i / N), dot = bb(p.x, p.y, p.z, '<i></i>', `dot ${cls}`); dots.push(dot);
      void anim(dot.firstElementChild!, [{ opacity: 0 }, { opacity: 1 }],
        { duration: 60, delay: ms * i / N, easing: 'linear', fill: 'both' });
    }
    const sh = el('i', 'bsh'); paint.appendChild(sh);
    const ball = bb(a.x, a.y, a.z, '<i></i>', 'ball');
    void anim(sh, sfr, { duration: ms, easing: 'linear' });
    await anim(ball, frames, { duration: ms, easing: 'linear' });
    sh.remove();
    return { ball, dots, apex: at(.5) };
  }
  function clearPlay() {
    up.replaceChildren();
    for (const n of [...paint.children]) { if (n.classList.contains('old')) n.remove(); else n.classList.add('old'); }
  }
  const spotText = (x: number) => { x = Math.round(x); return x === 50 ? '50' : x < 50 ? `${TEAM.L.ab} ${x}`
    : `${TEAM.R.ab} ${100 - x}`; };
  const toEnd = (x: number, s: Side) => s === 'L' ? 100 - x : x;
  const setNum = (node: HTMLElement | null, text: string, flash = true) => {
    if (!node || node.textContent === text) return;
    node.textContent = text;
    if (flash) { node.classList.remove('chg'); void node.offsetWidth; node.classList.add('chg'); }
  };

  async function sync(G: FieldGame, o: { instant?: boolean; flip?: boolean; first?: boolean } = {}) {
    const d = G.poss === 'L' ? 1 : -1, goal = toEnd(G.x, G.poss) <= G.dist, x = wx(G.x), y = wy(G.lat);
    const fxd = goal ? wx(d > 0 ? 100 : 0) : wx(G.x + d * G.dist);
    const ddText = `${ORD[G.down - 1] ?? '—'} & ${goal ? 'GOAL' : G.dist}`;
    stage.classList.toggle('instant', !!o.instant);
    teamVars(G.poss, 'p');
    if (o.flip) {                                       // possession change: marker spins round in the new colour
      clearPlay(); paint.replaceChildren();
      void anim(mk.querySelector('.pin')!, [{ transform: 'rotateY(0) scale(1)' },
        { transform: 'rotateY(540deg) scale(1.5)', offset: .5 }, { transform: 'rotateY(1080deg) scale(1)' }],
      { duration: 800, easing: 'ease-in-out', fill: 'none' });
      wipe('poss', 620);
    }
    los.style.transform = `translateX(${x}px)`;
    first.style.transform = `translateX(${fxd}px)`; first.style.opacity = goal ? '0' : '1';
    const a = wx(G.driveStart);
    const span = (n: HTMLElement, p: number, r: number) => { n.style.transform = `translateX(${Math.min(p, r)}px) `
      + `scaleX(${Math.max(Math.abs(r - p) / (100 * U), .0001)})`; };
    span(drv, a, x); span(togo, x, fxd);
    dd.dataset.dir = d > 0 ? 'r' : 'l'; dd.style.transform = `translateX(${x}px) translateZ(1px)`;
    dd.firstElementChild!.textContent = d > 0 ? `${ddText} ▸` : `◂ ${ddText}`;
    dd.style.opacity = '1';
    /* the real sticks on the near sideline: down box at the ball, line-to-gain stick at the first-down line */
    const sa = q('#v2stk'), sb = q('#v2stk2'), sc = q('#v2chain'), pad = q('#v2pad'), n = q('#stkDn');
    const dn = String(G.down);
    sa.style.transform = t3(x, sy, 0); sb.style.transform = t3(fxd, sy, 0);
    pad.style.transform = `translateX(${fxd}px) translateZ(1px)`;
    sc.style.transform = `translateX(${Math.min(x, fxd)}px) translateZ(1px)`; sc.style.width = `${Math.abs(fxd - x)}px`;
    for (const e of [sb, sc, pad]) e.classList.toggle('away', goal);
    if (n.textContent !== dn) {
      n.textContent = dn;
      if (!o.instant && !reduced()) void anim(n, [{ transform: 'rotateX(90deg)' }, { transform: 'rotateX(0)' }],
        { duration: 300, delay: 300, easing: 'cubic-bezier(.2,.9,.3,1.2)', fill: 'backwards' });
    }
    setNum(q<HTMLElement>('#hDown'), ORD[G.down - 1] ?? '—', !o.instant);
    setNum(q<HTMLElement>('#hDist'), goal ? 'GOAL' : String(G.dist), !o.instant);
    setNum(q<HTMLElement>('#hSpot'), spotText(G.x), !o.instant);
    setNum(q<HTMLElement>('#hGoal'), `${toEnd(G.x, G.poss)} YD`, false);
    q('#v2rib').innerHTML = `<span>${Array(14).fill(`${esc(name(G.poss))} BALL &nbsp;◆&nbsp; ${ddText} `
      + `&nbsp;◆&nbsp; ${spotText(G.x)} &nbsp;◆&nbsp; `).join('')}</span>`;
    const ms = o.instant ? 0 : o.flip ? 850 : 460;
    look(G.x + d * LEAD, ms + 140);
    if (o.first && !o.flip) {
      stage.classList.add('firstdown'); setTimeout(() => stage.classList.remove('firstdown'), 1100);
      fdBlocks(x, fxd, d);
    }
    await moveMk(x, y, ms);
    stage.classList.remove('instant');
  }

  /* first down: the new ten yards build up on the turf in the club's colour, a column of blocks per yard */
  function fdBlocks(xa: number, xb: number, d: number) {
    if (reduced() || Math.abs(xb - xa) < U) return;
    const n = Math.max(1, Math.round(Math.abs(xb - xa) / U)), cw = Math.abs(xb - xa) / n, box = el('div', 'fdb');
    box.style.cssText = `left:${Math.min(xa, xb)}px;top:${FT_}px;width:${Math.abs(xb - xa)}px;height:${FH}px`;
    world.appendChild(box);
    for (let i = 0; i < n; i++) {
      const c = el('i'); c.style.cssText = `left:${(d > 0 ? i : n - 1 - i) * cw + 1.5}px;width:${cw - 3}px`;
      box.appendChild(c);
      void anim(c, [{ clipPath: 'inset(0 0 100% 0)' }, { clipPath: 'inset(0 0 0 0)', offset: .22 },
        { clipPath: 'inset(0 0 0 0)', offset: .74 }, { clipPath: 'inset(100% 0 0 0)' }],
      { duration: 1500, delay: 380 + i * 55, easing: 'cubic-bezier(.5,0,.3,1)', fill: 'both' });
    }
    setTimeout(() => box.remove(), 380 + n * 55 + 1600);
  }

  /* ===== Big plays (20+ yards from scrimmage): the camera chases the ball, each yard line he crosses fires in
     the club's colour and a streak paints behind him; then the blast doors slam shut (BIG | PLAY). ===== */
  const isBig = (p: FieldPlay) => ['run', 'pass', 'td'].includes(p.type) && p.gain >= BIG;
  type Chase = { p: FieldPlay; d: number; td: boolean; gain: number; x0: number; y0: number; x1: number;
    y1: number; ms: number; fired: Set<number>; sl: HTMLElement; big: boolean };
  function ride(set: ((x: number, y: number) => void) | null, xa: number, ya: number, xb: number, yb: number,
    ms: number, c: Chase, e = runSpeed, z = 1.12) {
    return tween(ms, k => {
      const x = xa + (xb - xa) * k, y = ya + (yb - ya) * k;
      if (set) set(x, y);
      look(ydOf(x) + c.d * 7, 160, z); step(c, (x - c.x0) * c.d / U, x, y);
    }, e);
  }
  const jolt = (px = 4) => anim(stage, [{ transform: `translateY(${px}px)` }, { transform: `translateY(${-px / 2}px)` },
    { transform: 'none' }], { duration: 180, easing: 'steps(3,end)' });
  function step(c: Chase, yds: number, x: number, y: number) {
    c.sl.style.width = `${Math.hypot(x - c.x0, y - c.y0)}px`;
    c.sl.style.transform = `${t3(c.x0, c.y0, 1)} rotate(${Math.atan2(y - c.y0, x - c.x0)}rad)`;
    world.querySelectorAll<HTMLElement>('.yl').forEach(l => {
      const yd = Number(l.dataset.yd), sx = c.p.start.x;
      const crossed = c.d > 0 ? ydOf(x) >= yd && yd > sx : ydOf(x) <= yd && yd < sx;
      if (crossed && !c.fired.has(yd)) {
        c.fired.add(yd); l.style.animationDelay = '0ms'; l.classList.add('fire');
        setTimeout(() => l.classList.remove('fire'), 1000);
      }
    });
    if (yds >= BIG && !c.big) { c.big = true; wipe('bp', 520); }
  }
  /* blast doors: two grip slabs slam shut, the seam is the club's colour, the yards punch out of the seam */
  async function doors(c: Chase) {
    const st = el('div', 'bpx bp-doors', '<div class="dr l"><b>BIG</b><i class="bolt"></i><i class="bolt"></i></div>'
      + '<div class="dr r"><b>PLAY</b><i class="bolt"></i><i class="bolt"></i></div><div class="seam"></div>'
      + `<div class="yd"><b>+${c.gain}</b><span>${c.td ? 'TOUCHDOWN' : 'YARDS'}</span></div>`);
    stage.appendChild(st);
    const [l, r] = [...st.querySelectorAll('.dr')], sm = st.querySelector('.seam')!, yd = st.querySelector('.yd')!;
    const shut = { duration: 300, easing: 'cubic-bezier(.7,0,1,1)' };
    await Promise.all([anim(l, [{ transform: 'translateX(-101%)' }, { transform: 'none' }], shut),
      anim(r, [{ transform: 'translateX(101%)' }, { transform: 'none' }], shut)]);
    void jolt(6); void anim(sm, [{ transform: 'scaleY(0)' }, { transform: 'scaleY(1)' }],
      { duration: 160, easing: 'steps(3,end)' });
    await hold(260);
    await anim(yd, [{ transform: 'translate(-50%,-50%) scale(0)' },
      { transform: 'translate(-50%,-50%) scale(1.25)', offset: .7 }, { transform: 'translate(-50%,-50%) scale(1)' }],
    { duration: 260, easing: 'steps(5,end)', fill: 'both' });
    void jolt(3); await hold(c.td ? 1200 : 1800);
    void anim(yd, [{ opacity: 1 }, { opacity: 0 }], { duration: 120, easing: 'steps(2,end)', fill: 'forwards' });
    const part = { duration: 280, easing: 'cubic-bezier(.4,0,.6,1)', fill: 'forwards' as FillMode };
    await Promise.all([anim(l, [{ transform: 'none' }, { transform: 'translateX(-101%)' }], part),
      anim(r, [{ transform: 'none' }, { transform: 'translateX(101%)' }], part)]);
    st.remove();
  }
  async function bigPlay(p: FieldPlay) {
    const d = p.dir, td = p.type === 'td', x0 = wx(p.start.x), y0 = cur.y, y1 = wy(p.lat1);
    const x1 = wx(p.start.x + d * p.gain) + (td ? d * 4 * U : 0), pass = p.type === 'pass' || (td && p.how === 'pass');
    const streak = line(x0, y0, x0 + d, y0, 'bp-streak');
    const c: Chase = { p, d, td, gain: p.gain, x0, y0, x1, y1, ms: clamp(900 + p.gain * 22, 1400, 2600),
      fired: new Set(), sl: streak.parentElement!, big: false };
    if (!pass) {
      void grow(line(x0, y0, x1, y1, ''), c.ms, 'linear');
      await ride((x, y) => setMk(x, y), x0, y0, x1, y1, c.ms, c);
    } else {
      const qx = x0 - d * 5 * U, xc = wx(p.start.x + d * p.air), fms = clamp(700 + p.air * 16, 900, 1600);
      await grow(line(x0, y0, qx, y0, 'thin'), 260);
      const fl = flight({ x: qx, y: y0, z: 16 }, { x: xc, y: y1, z: 12 }, clamp(50 + p.air * 3.6, 60, 200), fms);
      await ride(null, qx, y0, xc, y1, fms, c, ease.lin);
      const f = await fl;
      f.ball.remove(); setMk(x1, y1);
    }
    if (!c.td) { label(c.x1, c.y1, `+${c.gain}`, 'team', 62); await hold(300); }
    await doors(c);
    if (!c.td) { look(ydOf(c.x1), 600, 1); await hold(500); }
  }

  async function play(p: FieldPlay) {
    if (!alive) return;
    clearPlay();
    if (isBig(p)) return bigPlay(p);
    const d = p.dir, x0 = wx(p.start.x), y0 = cur.y, y1 = wy(p.lat1);
    if (['run', 'sack', 'fumble'].includes(p.type) || (p.type === 'td' && p.how === 'run')) {
      const x1 = wx(p.start.x + d * p.gain) + (p.type === 'td' ? d * 4 * U : 0);
      const ms = clamp(540 + Math.abs(p.gain) * 18, 600, 1000);
      look(p.start.x + d * (p.gain * .7 + 4), ms + 260);
      void grow(line(x0, y0, x1, y1, p.gain < 0 ? 'loss' : ''), ms);
      await moveMk(x1, y1, ms);
      if (p.type === 'td') return;
      const yd = (p.gain > 0 ? '+' : p.gain < 0 ? '−' : '') + Math.abs(p.gain);
      label(x1, y1, p.type === 'sack' ? `SACK ${p.gain}` : yd, p.gain < 0 ? 'red' : 'team', 52, 90);
      if (p.type === 'fumble') { await hold(260); cross(x1, y1); label(x1, y1, 'FUMBLE', 'red', 92, 120); }
      await hold(260);
    } else if (['pass', 'inc', 'int', 'td'].includes(p.type)) {
      const qx = x0 - d * 5 * U, xc = wx(p.start.x + d * p.air), done = p.type === 'pass' || p.type === 'td';
      const ms = clamp(480 + p.air * 10, 540, 840);
      look(p.start.x + d * (p.air * (done ? .65 : .85) + 2), ms + 300);
      await grow(line(x0, y0, qx, y0, 'thin'), 220);                       // the drop-back, then the throw
      const f = await flight({ x: qx, y: y0, z: 16 }, { x: xc, y: y1, z: done ? 12 : 2 },
        clamp(34 + p.air * 3.4, 44, 170), ms, done ? '' : 'miss');
      if (!done) {
        cross(xc, y1); f.ball.remove(); f.dots.forEach(n => n.classList.add('dead'));
        label(xc, y1, p.type === 'int' ? 'INTERCEPTED' : 'INCOMPLETE', 'red', 44, 120); await hold(380); return;
      }
      const x1 = wx(p.start.x + d * p.gain) + (p.type === 'td' ? d * 4 * U : 0);
      f.ball.remove();
      if (p.type === 'td') { setMk(x1, y1); return; }
      label(x1, y1, `+${p.gain}`, 'team', 62, 160); await hold(300);
    } else if (p.type === 'penalty') {
      const x1 = wx(p.start.x + d * p.gain), a = Math.min(x0, x1), w = Math.abs(x1 - x0);
      const flag = bb(x0 + d * 30, y0 - 40, 0, '<i></i>', 'flag');
      await anim(flag, [{ transform: t3(x0 - d * 60, y0 - 120, 190) },
        { transform: t3(x0 + d * 10, y0 - 60, 130), offset: .55 }, { transform: t3(x0 + d * 30, y0 - 40, 0) }],
      { duration: 460, easing: 'cubic-bezier(.4,0,.9,.6)' });
      stage.classList.add('flagged'); wipe('flagw', 520);
      await hold(200);
      const hz = el('div', 'hz');
      hz.style.cssText = `left:${a}px;top:${FT_}px;width:${w}px;height:${FH}px;`
        + `transform-origin:${p.gain < 0 ? '100%' : '0'} 50%`;
      paint.appendChild(hz);
      void anim(hz, [{ transform: 'translateZ(1px) scaleX(0)' }, { transform: 'translateZ(1px) scaleX(1)' }], 320);
      const moved = p.pen?.moved ?? Math.abs(p.gain);
      const call = p.pen ? `${p.pen.name.toUpperCase()} ${p.gain > 0 ? '+' : '−'}${moved}` : 'FLAG';
      label(a + w / 2, wy(-.75), call, 'flagc', 70);
      if (p.pen) label(a + w / 2, wy(-.75), `ON ${p.pen.team}`, 'mini', 44, 220);
      await hold(380);
      const steps = Math.max(1, Math.round(moved / 5));
      for (let i = 1; i <= steps; i++) {                                     // walked off five yards at a time
        const tx = x0 + (x1 - x0) * i / steps; los.style.transform = `translateX(${tx}px)`;
        await moveMk(tx, cur.y, 170, 'ease-out'); await hold(110);
      }
      setTimeout(() => stage.classList.remove('flagged'), 1100);
    } else if (['fg', 'fgmiss', 'punt'].includes(p.type)) {
      const punt = p.type === 'punt', kx = x0 - d * (punt ? 12 : 7) * U, ms = punt ? 1100 : 1000;
      const tyd = punt ? clamp(p.start.x + d * p.kick, -4, 104) : (d > 0 ? 110 : -10);
      const tx = wx(tyd), ty = punt ? y1 : wy(p.lat1 * 2.4);
      await grow(line(x0, y0, kx, y0, 'thin'), 240);
      look(punt ? tyd : (d > 0 ? PAN[1] : PAN[0]), ms + 100);
      const f = await flight({ x: kx, y: y0, z: 4 }, { x: tx, y: ty, z: punt ? 4 : 52 }, punt ? 210 : 120, ms,
        p.type === 'fgmiss' ? 'miss' : 'kick');
      f.ball.remove();
      if (p.type === 'fg') {
        const goalPost = world.querySelector(`.v2-post[data-side=${d > 0 ? 'R' : 'L'}]`)!;
        goalPost.classList.add('good'); setTimeout(() => goalPost.classList.remove('good'), 6200);
      } else if (p.type === 'fgmiss') {
        label(tx - d * 40, ty, 'NO GOOD', 'red', 70); f.dots.forEach(n => n.classList.add('dead')); await hold(500);
      } else { label(tx, ty, `PUNT ${p.kick}`, '', 40); await hold(450); }
    }
  }

  /* the scoring plate (mock v1's moment, in the club's own colours): the yard lines fire in sequence away from
     the end zone, then a plate opens over the field between two moving colour rails and the word decodes. */
  async function banner(p: FieldPlay, scorer: string) {
    if (!alive || !p.scoring) return;
    const td = p.scoring === 'td', side: Side = p.dir > 0 ? 'R' : 'L', word = td ? 'TOUCHDOWN' : 'FIELD GOAL';
    const t = TEAM[p.poss];
    teamVars(p.poss, 't');
    look(side === 'R' ? 82 : 18, 700, 1);
    if (td) {
      q(`#v2ezfx${side}`).classList.add('lit');
      world.querySelectorAll<HTMLElement>('.yl').forEach(l => {
        const yd = Number(l.dataset.yd), i = (side === 'R' ? 100 - yd : yd) / 5;
        l.style.animationDelay = `${i * 34}ms`; l.classList.add('fire');
        setTimeout(() => l.classList.remove('fire'), 1900);
      });
      await hold(720);
    }
    const b = q('#v1ban'); b.hidden = false; b.className = `v1-ban ${td ? 'td' : 'fg'}`;
    const who = scorer ? `${esc(surnameOf(scorer))} · ` : '';
    b.innerHTML = '<div class="rail t"><i></i></div>'
      + `<div class="pl"><div class="bk"><b>${t.ab}</b><span>${td ? 'SCORING PLAY' : 'KICK IS GOOD'} · `
      + `Q${p.q} ${esc(p.clock)}</span></div><div class="bw" id="v1bw">${word}</div>`
      + `<div class="bs">${who}${td ? `${p.gain}-YD ${p.how.toUpperCase()}` : `${p.kick} YARDS`}</div>`
      + `<div class="bsc"><b class="${p.poss === 'L' ? 'hot' : ''}">${TEAM.L.ab} ${p.score.L}</b><i></i>`
      + `<b class="${p.poss === 'R' ? 'hot' : ''}">${TEAM.R.ab} ${p.score.R}</b></div></div>`
      + '<div class="rail b"><i></i></div>';
    const bw = q('#v1bw'), n = word.length, rows = [...b.querySelectorAll('.bk,.bs,.bsc')];
    rows.forEach((r, i) => void anim(r, [{ opacity: 0 }, { opacity: 1 }],
      { duration: 240, delay: [260, 1050, 1300][i], fill: 'both' }));
    void anim(b, [{ clipPath: 'inset(50% 0 50% 0)' }, { clipPath: 'inset(0 0 0 0)' }],
      { duration: 340, easing: 'cubic-bezier(.2,.9,.2,1)' });
    void anim(b.querySelector('.rail.t')!, [{ transform: 'translateX(-100%)' }, { transform: 'none' }], 460);
    void anim(b.querySelector('.rail.b')!, [{ transform: 'translateX(100%)' }, { transform: 'none' }], 460);
    await hold(200);
    await tween(760, k => {
      let s = '';
      for (let i = 0; i < n; i++) s += word[i] === ' ' ? ' ' : k >= (i + 1.5) / (n + 1.5) ? word[i] : pick(GLYPH);
      bw.textContent = s;
    }, ease.lin);
    bw.textContent = word; bw.classList.add('set');
    await hold(4200);
    await anim(b, [{ clipPath: 'inset(0 0 0 0)' }, { clipPath: 'inset(50% 0 50% 0)' }],
      { duration: 300, easing: 'ease-in' });
    b.hidden = true; b.getAnimations?.().forEach(a => a.cancel());
    world.querySelectorAll('.v2-ezfx.lit').forEach(e => e.classList.remove('lit'));
  }

  resize();
  return { sync, play, banner, resize, destroy() { alive = false; host.replaceChildren(); } };
}
