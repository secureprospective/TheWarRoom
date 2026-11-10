// TheWarRoom landing page: one camera flying 110 yards down a field.
// Scroll position drives the camera; the DOM scenes and the canvas field share one projection.
(() => {
  'use strict';
  const P = 1000;            // perspective distance, design units (matches .lens in site.css)
  const D = 2600;            // depth between scenes = 10 yards
  const YARD = D / 10;
  const FLOOR = 430, SIDE = 1500, WALL = 2100, HASH = 420, FAR = 9500;
  const HY = [0, 0, 0, 0, 0, -240, -150, -150, -150, -110, -300, -220];   // camera height at each scene (negative is up)
  const SPAN = [1.15, 1.15, 1.15, 1.2, 1.3, 1.5, 1.5, 1.5, 1.4, 1.5, 1.3];   // scroll length of each flight, in viewport heights: the case against the old way moves faster than the proof
  const TOP = SPAN.reduce((a, v) => (a.push(a[a.length - 1] + v), a), [0]);   // where each scene sits, in viewport heights
  const still = matchMedia('(prefers-reduced-motion: reduce)').matches;

  const $ = (id) => document.getElementById(id);
  const lens = $('lens'), world = $('world'), cv = $('field'), ctx = cv.getContext('2d'), fx = $('fx'), fxc = fx.getContext('2d');
  const hud = $('hud'), hudCh = $('hud-ch'), hudYd = $('hud-yd'), hudUs = $('hud-us');
  const css = getComputedStyle(document.documentElement);
  const col = (n) => css.getPropertyValue(n).trim();
  const C = { canvas: col('--canvas'), sunken: col('--sunken'), edge: col('--edge'), hi: col('--edge-hi'), amber: col('--amber'), black: col('--black') };

  // ---- scenes and their layers
  const scenes = [...world.querySelectorAll('.scene')].map((el, i) => {
    el.style.transform = `translate3d(0,${HY[i]}px,${-i * D}px)`;
    const layers = [...el.querySelectorAll('.ly')].map((l) => ({ el: l, z: parseFloat(l.style.getPropertyValue('--z')) || 0, o: -1, junk: l.querySelector('.junk'), broke: false }));
    el.inert = true;
    return { el, i, layers, near: false, seen: false, on: false, fall: el.querySelector('[data-fall]'), fell: false };
  });
  const N = scenes.length;
  const motes = Array.from({ length: 110 }, (_, i) => ({ x: (i * 7919) % 3800 - 1900, y: -900 + (i * 104729) % 1330, z: (i * 3571) % 7000, s: 6 + (i % 4) * 4, c: i % 7 }));

  // ---- layout
  let W = 0, H = 0, S = 1, dpr = 1, ROOF = -900, WEAVE = 150;
  function resize() {
    const keep = W ? progress() : 0;   // a resize (rotation, phone toolbars) must not move the visitor along the story
    W = innerWidth; H = innerHeight; dpr = Math.min(devicePixelRatio || 1, 2);
    const portrait = H > W;
    S = portrait ? Math.min(W / 580, H / 1060) : Math.min(W / 1440, H / 860);
    ROOF = portrait ? -1700 : -900;   // a taller hall on phones keeps the beams clear of the words
    lens.style.transform = `translateY(23px) scale(${S})`;
    cv.width = fx.width = W * dpr; cv.height = fx.height = H * dpr;
    WEAVE = portrait ? 55 : 150;
    $('track').style.height = `${(TOP[N - 1] + 1) * innerHeight}px`;
    if (keep) scrollTo(0, topOf(keep));
  }
  const topOf = (p) => { const i = Math.min(N - 2, Math.floor(p)); return (TOP[i] + (p - i) * SPAN[i]) * H; };
  function progress() {
    const v = scrollY / H; let i = 0;
    while (i < N - 2 && v >= TOP[i + 1]) i++;
    return Math.max(0, Math.min(N - 1, i + (v - TOP[i]) / SPAN[i]));
  }
  function goTo(i) { scrollTo({ top: topOf(i), behavior: still ? 'auto' : 'smooth' }); }
  window.twrTop = topOf;

  // ---- camera
  let cam = 0, camX = 0, camY = 0, px = 0, vel = 0, mx = 0, my = 0, shake = 0, litZ = 0, litGoal = 0, last = 0;
  const smooth = (t) => t * t * t * (t * (t * 6 - 15) + 10);
  function camH(z) {
    const p = Math.max(0, Math.min(N - 1, z / D)), i = Math.min(N - 2, Math.floor(p));
    return HY[i] + (HY[i + 1] - HY[i]) * smooth(p - i);
  }
  // Between scenes the camera cuts to one side and back, like a back slashing through a gap. It leans toward the old tool it is about to hit.
  const CUT = [1, -1, 1, -1, 1, -1, 1, -1, 1, -1, 0];
  function weave(z) {
    const p = Math.max(0, Math.min(N - 1, z / D)), i = Math.min(N - 2, Math.floor(p)), t = p - i;
    return CUT[i] * Math.sin(Math.PI * Math.min(1, t * 1.25)) ** 2 * (1 - t * 0.35);
  }
  function target() {
    const p = progress();
    if (still) return Math.round(p) * D;
    const i = Math.min(N - 2, Math.floor(p)), t = p - i;
    const e = smooth(Math.max(0, Math.min(1, (t - 0.18) / 0.64)));
    return i * D + e * (D - 90) + t * 90;   // holds at each scene, with a slow drift so it never freezes
  }

  // ---- canvas field: floor, walls, portals and motes, drawn far to near in 5-yard slices
  const K = (d) => (P / (P + d)) * S;
  const X = (x, d) => (W / 2 + (x - camX) * K(d)) * dpr;
  const Y = (y, d) => (H / 2 + 23 + (y - camY) * K(d)) * dpr;
  const fog = (d) => Math.max(0, 1 - Math.max(0, d) / FAR) ** 1.7;
  const NEAR = -P * 0.86;
  const lit = (z) => z >= 5 * D - 1 && z <= litZ;
  function seg(x1, y1, d1, x2, y2, d2, w, color, a) {
    if (d1 < NEAR && d2 < NEAR) return;
    if (d1 < NEAR) { const t = (NEAR - d1) / (d2 - d1); x1 += (x2 - x1) * t; y1 += (y2 - y1) * t; d1 = NEAR; }
    if (d2 < NEAR) { const t = (NEAR - d2) / (d1 - d2); x2 += (x1 - x2) * t; y2 += (y1 - y2) * t; d2 = NEAR; }
    const dm = (d1 + d2) / 2;
    ctx.globalAlpha = a * fog(dm); ctx.strokeStyle = color; ctx.lineWidth = Math.max(1, w * K(dm) * dpr);
    ctx.beginPath(); ctx.moveTo(X(x1, d1), Y(y1, d1)); ctx.lineTo(X(x2, d2), Y(y2, d2)); ctx.stroke();
  }
  function quad(pts, fill, a) {
    ctx.globalAlpha = a; ctx.fillStyle = fill; ctx.beginPath();
    pts.forEach(([x, y, d], i) => (i ? ctx.lineTo(X(x, d), Y(y, d)) : ctx.moveTo(X(x, d), Y(y, d))));
    ctx.fill();
  }
  const rnd = (n) => { const v = Math.sin(n * 127.1) * 43758.5453; return v - Math.floor(v); };
  const SIGNAL = [C.hi, C.hi, col('--green'), C.amber, col('--blue')];

  // the steel: real meshes from steel.js, rebuilt when the hall changes height
  const steel = window.TWRSteel, view = { P, S: 1, W: 0, H: 0, dpr: 1, yOff: 23, camX: 0, camY: 0, near: NEAR, far: FAR };
  let frameMesh = null, postMesh = null, meshRoof = 0;
  function meshes() {
    if (meshRoof === ROOF) return;
    meshRoof = ROOF;
    frameMesh = steel.portal({ halfSpan: WALL, floorY: FLOOR, beamY: ROOF + 150, lights: [-1410, -940, -470, 0, 470, 940, 1410] });
    postMesh = steel.goalpost({ floorY: FLOOR, crossbarY: FLOOR - 520, halfWidth: 520, topY: ROOF - 500, setback: 430 });
  }
  function portal(d, on, label) {
    if (d < NEAR + 80 || d > FAR) return;
    const k = K(d) * dpr, a = fog(d);
    const x0 = X(-WALL, d), x1 = X(WALL, d), top = Y(ROOF - 700, d), bot = Y(ROOF - 40, d);
    ctx.globalAlpha = a; ctx.fillStyle = on ? 'hsl(220,18%,9%)' : 'hsl(220,18%,6%)';   // the bulkhead the girder is hung under
    ctx.fillRect(x0, top, x1 - x0, bot - top);
    if (d < 2000) {
      ctx.fillStyle = on ? C.hi : C.edge; ctx.font = `700 ${150 * k}px "JetBrains Mono",monospace`; ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
      ctx.fillText(label, X(0, d), bot - 120 * k);
    }
    steel.draw(ctx, frameMesh, view, d, on ? 1 : 0);
  }

  function drawField(now) {
    meshes(); Object.assign(view, { S, W, H, dpr, camX, camY });
    ctx.globalAlpha = 1; ctx.fillStyle = C.sunken; ctx.fillRect(0, 0, cv.width, cv.height);
    const SL = 5 * YARD, td = cam > 10 * D - 700;
    const sMax = Math.min(25, Math.ceil((cam + FAR) / SL)), sMin = Math.max(-2, Math.floor((cam + NEAR) / SL));
    for (let s = sMax; s >= sMin; s--) {
      const yd = s * 5, z = s * SL, d1 = Math.max(z - cam, NEAR), d2 = z + SL - cam;
      if (d2 <= d1) continue;
      const f = fog((d1 + d2) / 2), on = lit(z), ez = yd < 0 || yd >= 100, onField = yd >= -10 && yd < 110;
      const line = on ? C.hi : C.edge;
      if (s === 24) {   // goalposts behind the end zone
        const gd = z - cam;
        if (H > W && gd < 4200) continue;   // on phones the posts would cross the last screen's words
        steel.draw(ctx, postMesh, view, gd, Math.max(0, Math.min(1, (litZ - 96 * YARD) / (8 * YARD))));
      }
      // floor: apron wall to wall, then the field
      quad([[-WALL, FLOOR, d1], [WALL, FLOOR, d1], [WALL, FLOOR, d2], [-WALL, FLOOR, d2]], 'hsl(220,20%,5%)', f);
      if (onField) {
        const dark = s & 1;
        const fill = on ? (ez ? 'hsl(220,18%,9%)' : dark ? 'hsl(138,36%,10%)' : 'hsl(138,36%,13%)') : dark ? 'hsl(220,18%,6.5%)' : 'hsl(220,18%,8%)';
        quad([[-SIDE, FLOOR, d1], [SIDE, FLOOR, d1], [SIDE, FLOOR, d2], [-SIDE, FLOOR, d2]], fill, f);
        if (ez) for (let x = -SIDE; x < SIDE; x += 375) { const run = Math.min(SL, SIDE - x); seg(x, FLOOR, z - cam, x + run, FLOOR, z + run - cam, 26, line, 0.55); }
        for (const x of [-SIDE, SIDE]) seg(x, FLOOR, z - cam, x, FLOOR, z + SL - cam, 12, line, 1);
        if (!ez) for (let h = 1; h < 5; h++) for (const hx of [-HASH, HASH, -SIDE + 60, SIDE - 60]) seg(hx - 36, FLOOR, z + h * YARD - cam, hx + 36, FLOOR, z + h * YARD - cam, 5, line, 0.75);
      }
      // walls, with rails; past midfield they carry screens
      for (const sx of [-1, 1]) {
        const x = sx * WALL;
        quad([[x, ROOF, d1], [x, FLOOR, d1], [x, FLOOR, d2], [x, ROOF, d2]], on ? 'hsl(220,18%,8%)' : 'hsl(220,20%,4.5%)', f);
        for (let r = 1; r < 5; r++) { const y = ROOF + (FLOOR - ROOF) * r / 5; seg(x, y, z - cam, x, y, z + SL - cam, 5, C.edge, 0.8); }
        seg(x, ROOF, z - cam, x, FLOOR, z - cam, 5, C.edge, 0.8);
        if (on && z - cam > NEAR) for (let r = 1; r < 4; r++) {
          const h = rnd(s * 9 + r * 3 + sx); if (h < 0.35) continue;
          const ya = ROOF + (FLOOR - ROOF) * r / 5 + 40, yb = ROOF + (FLOOR - ROOF) * (r + 1) / 5 - 40, za = z - cam + 150, zb = z - cam + SL - 150;
          quad([[x, ya, za], [x, yb, za], [x, yb, zb], [x, ya, zb]], 'hsl(220,16%,14%)', f);
          for (let b = 0; b < 4; b++) { const y = ya + (yb - ya) * (b + 1) / 5; seg(x, y, za + 80, x, y, za + 80 + (zb - za - 160) * (0.3 + 0.7 * rnd(s + r * 7 + b + sx)), 9, SIGNAL[Math.floor(rnd(s * 3 + r + b) * 5)], 0.9); }
        }
      }
      // what stands on the near edge of this slice
      if (z - cam > NEAR && yd >= 0 && yd <= 100) {
        seg(-SIDE, FLOOR, z - cam, SIDE, FLOOR, z - cam, yd % 10 === 0 ? 10 : 6, line, 1);
        if (yd % 10 === 0 && yd > 0 && yd < 100) {   // painted numbers, flattened onto the floor
          const d = z - cam + 120, hgt = Y(FLOOR, d) - Y(FLOOR, d + 520), k = K(d) * dpr;
          if (hgt > 4 && d > NEAR + 200) {
            ctx.globalAlpha = 0.5 * fog(d) * Math.min(1, Math.max(0, (d - 500) / 900)); ctx.fillStyle = line; ctx.textAlign = 'center'; ctx.textBaseline = 'alphabetic'; ctx.font = '700 100px "JetBrains Mono",monospace';
            for (const nx of [-930, 930]) { ctx.save(); ctx.translate(X(nx, d), Y(FLOOR, d)); ctx.scale(k * 4.2, hgt / 74); ctx.fillText(String(yd <= 50 ? yd : 100 - yd), 0, 0); ctx.restore(); }
          }
        }
      }
      if (yd % 10 === 0 && yd >= 0 && yd <= 110) { const n = yd + 10; portal(z + 650 - cam, on, n === 50 ? '50' : n >= 100 ? 'GOAL' : String(n < 50 ? n : 100 - n)); }
    }
    // motes hanging in the air; confetti once the ball is in the end zone
    for (const m of motes) {
      const d = ((m.z - cam) % 7000 + 7000) % 7000 - 700, party = td && !still && cam + d > 96 * YARD;
      const y = party ? -900 + ((m.y + 900 + now * 0.22 * (1 + m.c / 5)) % 1330) : m.y;
      const sz = m.s * (party ? 2.6 : 1) * K(d) * dpr;
      ctx.globalAlpha = fog(d) * (d < -300 ? (d + 700) / 400 : 1) * (party ? 0.95 : 0.45);
      ctx.fillStyle = party ? SIGNAL[m.c % 4 + 1] : lit(cam + d) ? C.hi : C.edge;
      ctx.fillRect(X(m.x, d), Y(y, d), sz * (party ? 1.8 : 1), sz);
    }
    ctx.globalAlpha = 1;
  }

  // ---- per-frame: place the camera, fade layers by distance, wake scenes
  function frame(now) {
    const dt = Math.min(0.05, (now - last) / 1000 || 0.016); last = now;
    const tg = target();
    const was = cam;
    cam = still ? tg : cam + (tg - cam) * (1 - Math.exp(-dt * 9));
    vel = (cam - was) / dt;
    if (!still) {
      px += (mx * 90 - px) * (1 - Math.exp(-dt * 5));
      camX = px + weave(cam) * WEAVE;
      shake *= Math.exp(-dt * 14);
      }
    camY = camH(cam) + (still ? 0 : my * 45 + Math.abs(weave(cam)) * 16 + (shake > 0.3 ? Math.sin(now / 16) * shake : 0));
    litZ += (litGoal - litZ) * (1 - Math.exp(-dt * (litGoal > litZ ? 2.2 : 12)));
    world.style.transform = `translate3d(${-camX}px,${-camY}px,${cam}px)`;

    let cur = 0, best = 1e9;
    for (const s of scenes) {
      const d = s.i * D - cam;
      if (Math.abs(d) < best) { best = Math.abs(d); cur = s.i; }
      const near = d < 3200 && d > -2400;
      if (near !== s.near) { s.near = near; s.el.classList.toggle('near', near); }
      if (!near) { if (s.seen) { s.seen = false; s.el.classList.remove('seen'); } continue; }
      const on = Math.abs(d) < 700;
      if (on !== s.on) { s.on = on; s.el.classList.toggle('on', on); s.el.inert = !on; }
      if (d < 750 && d > -1500 && !s.seen) {
        s.seen = true; s.el.classList.add('seen');
        if ('jolt' in s.el.dataset && !still) shake = 9;
      } else if (s.seen && (d > 2300 || d < -1900)) { s.seen = false; s.el.classList.remove('seen'); }
      if (s.fall && !still) {   // the 1998 page drops apart as the camera reaches it
        const hit = d < -70;
        if (hit !== s.fell) { s.fell = hit; s.fall.classList.toggle('fall', hit); if (hit && vel > 300) { shake = Math.max(shake, 5); thud(); } }
      }
      for (const l of s.layers) {
        const dl = d - l.z;
        if (l.junk && !still) {   // old tools shatter when the camera goes through them
          if (!l.broke && dl < -300 && vel > 500) { l.broke = true; shatter(l.junk); }
          else if (l.broke && dl > -240) l.broke = false;
        }
        const farF = still ? (on ? 1 : 0) : dl <= 300 ? 1 : dl >= 1500 ? 0 : (1 - (dl - 300) / 1200) ** 2;
        const nearF = dl >= -330 ? 1 : dl <= -760 ? 0 : (dl + 760) / 430;
        const o = l.broke ? 0 : Math.round(farF * nearF * 100) / 100;
        if (o !== l.o) { l.o = o; l.el.style.opacity = o; }
      }
    }
    litGoal = cam > 5 * D - 900 ? 126 * YARD : 0;
    updateHud(cur);
    const sig = `${cam.toFixed(1)}|${camX.toFixed(1)}|${camY.toFixed(1)}|${litZ.toFixed(0)}|${W}x${H}`, party = !still && cam > 10 * D - 700;
    if (sig !== lastSig || party) { lastSig = sig; drawField(now); }
    drawShards(dt);
    requestAnimationFrame(frame);
  }

  // ---- breaking the old tools: a window's own rectangle splits into triangles that fly past the camera
  const shards = [];
  let fxDirty = false;
  function shatter(el) {
    const b = el.getBoundingClientRect();
    if (b.right < 0 || b.left > W || b.bottom < 0 || b.top > H || b.width < 40) return;
    const nx = 5, ny = 4, pt = (i, j) => [b.left + b.width * (i / nx + (i % nx ? (rnd(i * 7 + j) - 0.5) * 0.16 : 0)), b.top + b.height * (j / ny + (j % ny ? (rnd(j * 5 + i * 3) - 0.5) * 0.16 : 0))];
    for (let i = 0; i < nx; i++) for (let j = 0; j < ny; j++) {
      const q = [pt(i, j), pt(i + 1, j), pt(i + 1, j + 1), pt(i, j + 1)];
      for (const tri of [[q[0], q[1], q[2]], [q[0], q[2], q[3]]]) {
        const cx = (tri[0][0] + tri[1][0] + tri[2][0]) / 3, cy = (tri[0][1] + tri[1][1] + tri[2][1]) / 3;
        const ox = cx - W / 2, oy = cy - H / 2, len = Math.hypot(ox, oy) || 1, sp = 500 + rnd(cx + cy) * 900, h = rnd(cx * 3 + cy);
        shards.push({ pts: tri.map(([x, y]) => [x - cx, y - cy]), x: cx, y: cy, vx: ox / len * sp, vy: oy / len * sp - 120, a: 0, va: (h - 0.5) * 9, s: 1, t: 0,
          c: j === 0 ? 'hsl(220,6%,52%)' : h < 0.18 ? 'hsl(220,8%,30%)' : h < 0.6 ? 'hsl(220,7%,66%)' : 'hsl(220,7%,58%)' });
      }
    }
    shake = Math.max(shake, 7); crack();
  }
  function drawShards(dt) {
    if (!shards.length) { if (fxDirty) { fxc.clearRect(0, 0, fx.width, fx.height); fxDirty = false; } return; }
    fxc.clearRect(0, 0, fx.width, fx.height); fxDirty = true;
    for (let n = shards.length - 1; n >= 0; n--) {
      const q = shards[n]; q.t += dt;
      if (q.t > 0.85) { shards.splice(n, 1); continue; }
      q.vy += 900 * dt; q.x += q.vx * dt; q.y += q.vy * dt; q.a += q.va * dt; q.s += 1.6 * dt;
      const c = Math.cos(q.a) * q.s * dpr, sn = Math.sin(q.a) * q.s * dpr;
      fxc.globalAlpha = Math.min(1, (0.85 - q.t) * 4); fxc.fillStyle = q.c; fxc.strokeStyle = 'hsl(220,6%,36%)'; fxc.lineWidth = dpr;
      fxc.beginPath();
      q.pts.forEach(([x, y], k) => { const X2 = q.x * dpr + x * c - y * sn, Y2 = q.y * dpr + x * sn + y * c; k ? fxc.lineTo(X2, Y2) : fxc.moveTo(X2, Y2); });
      fxc.closePath(); fxc.fill(); fxc.stroke();
    }
    fxc.globalAlpha = 1;
  }
  // the contract spreadsheet: cut it in two along a jagged line; the halves part when the scene lands
  document.querySelectorAll('[data-crack]').forEach((el) => {
    const cut = [[58, 0], [52, 18], [61, 31], [47, 49], [56, 64], [44, 82], [50, 100]];
    const poly = (side) => `polygon(${[...cut.map(([x, y]) => `${x}% ${y}%`), ...(side ? ['100% 100%', '100% 0'] : ['0 100%', '0 0'])].join(',')})`;
    const twin = el.cloneNode(true);
    twin.removeAttribute('data-crack'); twin.classList.add('half', 'b'); el.classList.add('half', 'a');
    el.style.clipPath = poly(0); twin.style.clipPath = poly(1);
    el.after(twin);
  });

  // ---- sound: a quiet crowd, and a cheer that swells and dies when a button is pressed. Made here, no audio files.
  let ac = null, crowd = null, noise = null, soundOn = false;
  function audio() {
    if (ac) return ac;
    ac = new (window.AudioContext || window.webkitAudioContext)();
    noise = ac.createBuffer(1, ac.sampleRate * 4, ac.sampleRate);
    const ch = noise.getChannelData(0); let b0 = 0, b1 = 0, b2 = 0;
    for (let i = 0; i < ch.length; i++) {   // pink-ish noise
      const w = Math.random() * 2 - 1; b0 = 0.997 * b0 + 0.029 * w; b1 = 0.985 * b1 + 0.032 * w; b2 = 0.95 * b2 + 0.048 * w; ch[i] = (b0 + b1 + b2 + w * 0.02) * 2.2;
    }
    return ac;
  }
  function voice(freq, q, gain) {   // one band of crowd: looping noise through a band-pass
    const src = ac.createBufferSource(), f = ac.createBiquadFilter(), g = ac.createGain();
    src.buffer = noise; src.loop = true; src.playbackRate.value = 0.8 + Math.random() * 0.4;
    f.type = 'bandpass'; f.frequency.value = freq; f.Q.value = q; g.gain.value = gain;
    src.connect(f).connect(g); src.start(0, Math.random() * 3);
    return { src, f, g };
  }
  function setSound(on) {
    soundOn = on;
    const b = $('snd'); b.setAttribute('aria-pressed', on); b.lastElementChild.textContent = on ? 'ON' : 'OFF';
    if (!on) { if (crowd) { crowd.out.gain.setTargetAtTime(0, ac.currentTime, 0.2); } return; }
    audio().resume();
    if (!crowd) {
      const out = ac.createGain(); out.gain.value = 0; out.connect(ac.destination);
      const bands = [voice(420, 0.7, 0.5), voice(900, 0.9, 0.32), voice(1700, 1.2, 0.12)];
      const lfo = ac.createOscillator(), depth = ac.createGain(); lfo.frequency.value = 0.13; depth.gain.value = 0.012; lfo.connect(depth).connect(out.gain); lfo.start();
      bands.forEach((v) => v.g.connect(out));
      crowd = { out };
    }
    crowd.out.gain.setTargetAtTime(0.05, ac.currentTime, 0.8);
  }
  function burst(bands, peak, rise, fall) {
    if (!soundOn || !ac) return;
    const t = ac.currentTime, out = ac.createGain();
    out.gain.setValueAtTime(0.0001, t); out.gain.exponentialRampToValueAtTime(peak, t + rise); out.gain.setTargetAtTime(0.0001, t + rise + 0.15, fall / 4);
    out.connect(ac.destination);
    const vs = bands.map(([f, q, g, glide]) => { const v = voice(f, q, g); if (glide) v.f.frequency.exponentialRampToValueAtTime(f * glide, t + rise + fall * 0.5); v.g.connect(out); return v; });
    setTimeout(() => { vs.forEach((v) => v.src.stop()); out.disconnect(); }, (rise + fall + 0.6) * 1000);
  }
  const cheer = () => burst([[700, 0.8, 0.5, 1.25], [1500, 1.1, 0.4, 1.35], [2600, 1.4, 0.22, 1.2], [3800, 2, 0.08, 1]], 0.2, 0.28, 2.6);
  const crack = () => burst([[2400, 0.6, 0.5, 0.5], [5200, 0.8, 0.3, 0.6]], 0.12, 0.012, 0.3);
  const thud = () => burst([[110, 0.9, 1, 0.6], [320, 0.7, 0.3, 0.7]], 0.3, 0.015, 0.45);
  $('snd').addEventListener('click', () => setSound(!soundOn));
  addEventListener('click', (e) => { if (e.target.closest('button,.btn,.go')) cheer(); });

  // ---- scorebug
  let hudKey = '', lastSig = '';
  function updateHud(cur) {
    const yd = Math.round(cam / YARD);
    const where = yd <= 0 ? 'OWN GOAL LINE' : yd < 50 ? `BALL ON OWN ${yd}` : yd === 50 ? 'MIDFIELD' : yd < 100 ? `BALL ON THEIR ${100 - yd}` : 'IN THE END ZONE';
    const us = cur >= 11 ? 7 : cam > 10 * D - 500 ? 6 : 0;
    const key = `${cur}|${where}|${us}`;
    hud.style.setProperty('--p', Math.min(1, cam / (10 * D)).toFixed(4));
    if (key === hudKey) return;
    hudKey = key;
    hudCh.textContent = scenes[cur].el.dataset.ch;
    hudYd.textContent = where;
    hudUs.textContent = us;
  }

  // ---- game day: plays keep landing while you watch
  const plays = [
    ['Montano to Berry Rice, 18 yds over the middle', '+2.8', 1, 10], ['Walter Peyton up the middle for 4 yds', '+0.4', 1, 3],
    ['Dan Merino short right to Largeant for 6 yds', '', 0, 4], ['Reggie Wight sacks the quarterback, -8 yds', '+2.0', 1, -6],
    ['Berry Sanders left end for 9 yds', '+0.9', 1, 6], ['Brett Favor deep left to Berry Rice, 31 yds', '+4.1', 1, 14],
    ['Emmett Smyth right side for 3 yds', '', 0, 2], ['John Elwood to Sharp, 22 yds, down at the 9', '+3.2', 1, 9],
  ];
  let playN = 0, usPts = 101.2, los = 68, clock = 8 * 60 + 42;
  function play() {
    if (!scenes[6].seen || document.hidden) return;
    const [text, pts, mine, gain] = plays[playN++ % plays.length];
    clock = Math.max(5, clock - 23 - (playN % 3) * 9);
    const li = document.createElement('li');
    li.className = `new${mine ? ' mine' : ''}`;
    li.innerHTML = `<span class="n">Q3 ${Math.floor(clock / 60)}:${String(clock % 60).padStart(2, '0')}</span><span></span><em></em>`;
    li.children[1].textContent = text; li.children[2].textContent = pts;
    $('clock').textContent = `Q3 ${Math.floor(clock / 60)}:${String(clock % 60).padStart(2, '0')}`;
    const ol = $('pbp'); ol.prepend(li); while (ol.children.length > 7) ol.lastChild.remove();
    $('lastplay').textContent = text;
    if (pts) { usPts += parseFloat(pts); $('us').textContent = usPts.toFixed(1); $('usbar').style.width = `${Math.min(92, 53 + (usPts - 101.2) * 1.4)}%`; }
    los = los + gain > 90 || los + gain < 34 ? 40 : los + gain;
    $('los').style.left = `${los}%`; $('drive').style.width = `${Math.max(2, los - 30)}%`;
  }

  // ---- any league: the franchise wall follows the TEAMS switch
  const names = ['Gridiron Syndicate', 'Burnt Orange Militia', 'Cascade Hammers', 'Desert Wolves', 'Lakeshore Ironworks', 'Frozen Tundra FC', 'Motor City Rust', 'North Woods Lumber', 'Brotherly Shove', 'Capital Punishment', 'Big Apple Bruisers', 'Lone Star Outlaws', 'Bayou Bandits', 'Peach State Pistons', 'Gulf Coast Gators', 'Carolina Kudzu', 'Steel Curtain Call', 'Ravens Remorse', 'Queen City Cats', 'Dawg Pound Diesel', 'Mile High Mayhem', 'Arrowhead Anvils', 'Silver & Black Ops', 'Bolt Throwers', 'Buffalo Stampede', 'Flight Risk', 'Duval Dynasty', 'Lucas Oil Slick', 'Music City Mafia', 'Patriot Missiles', 'South Beach Sharks', 'Iron Curtain'];
  const fr = $('fr');
  names.forEach((n, j) => { const s = document.createElement('span'); s.textContent = n; s.style.setProperty('--j', j); fr.append(s); });
  function press(group, btn) { for (const b of group.children) b.setAttribute('aria-pressed', b === btn); }
  document.querySelectorAll('[data-one],[data-teams]').forEach((g) => g.addEventListener('click', (e) => {
    const b = e.target.closest('.mbtn'); if (!b) return;
    press(g, b);
    if ('teams' in g.dataset) {
      const n = +b.textContent;
      [...fr.children].forEach((s, j) => s.classList.toggle('off', j >= n));
      $('teamcount').textContent = `${n} franchises`;
    }
  }));

  // ---- extra point: the card for the group chat
  const verdicts = ['Either you are lying or you are already in The War Room.', 'One thing. Suspiciously sane.', 'Two things that have never met each other.',
    'Three things and a commissioner holding it together.', 'Four things. Your commissioner deserves a raise.', 'Five things. This is a second job.', 'All six. This is the dumbest way possible.'];
  function card() {
    const on = [...$('picks').children].filter((b) => b.getAttribute('aria-pressed') === 'true').map((b) => b.textContent);
    $('card-n').textContent = on.length;
    $('card-h').textContent = on.length === 1 ? 'thing' : 'different things';
    $('card-l').replaceChildren(...on.map((t) => Object.assign(document.createElement('li'), { textContent: t })));
    $('card-v').textContent = verdicts[on.length];
    return `My league runs on ${on.length} different things: ${on.join(', ').toLowerCase()}. ${verdicts[on.length]} How dumb is yours? https://warroom.secureprospective.com`;
  }
  $('picks').addEventListener('click', (e) => {
    const b = e.target.closest('.mbtn'); if (!b) return;
    b.setAttribute('aria-pressed', b.getAttribute('aria-pressed') !== 'true'); card();
  });
  $('share').addEventListener('click', async (e) => {
    const text = card(), b = e.currentTarget;
    try { if (navigator.share) { await navigator.share({ text }); b.textContent = 'Sent. Go start something.'; } else { await navigator.clipboard.writeText(text); b.textContent = 'Copied. Go start something.'; } }
    catch (err) { if (err.name !== 'AbortError') b.textContent = 'Could not copy. Try again.'; }
  });
  // the same card as a picture, drawn at 1080 x 1350 for a group chat or a story
  function cardImage() {
    const on = [...$('picks').children].filter((b) => b.getAttribute('aria-pressed') === 'true').map((b) => b.textContent);
    const c = document.createElement('canvas'), g = c.getContext('2d'); c.width = 1080; c.height = 1350;
    const ink = col('--ink'), ink2 = col('--ink2'), ink3 = col('--ink3');
    g.fillStyle = C.sunken; g.fillRect(0, 0, 1080, 1350);
    g.fillStyle = C.canvas; g.fillRect(60, 84, 960, 1182);
    g.save(); g.beginPath(); g.rect(60, 60, 960, 24); g.clip(); g.fillStyle = C.black; g.fillRect(60, 60, 960, 24); g.fillStyle = C.hi;   // hazard strip
    for (let x = 30; x < 1060; x += 48) { g.beginPath(); g.moveTo(x, 84); g.lineTo(x + 24, 84); g.lineTo(x + 48, 60); g.lineTo(x + 24, 60); g.fill(); }
    g.restore();
    g.strokeStyle = C.hi; g.lineWidth = 3; g.strokeRect(60, 84, 960, 1182);
    g.textBaseline = 'alphabetic'; g.fillStyle = ink3; g.font = '700 30px "JetBrains Mono",monospace'; g.fillText('MY LEAGUE RUNS ON', 120, 190);
    g.fillStyle = ink; g.font = '700 330px "JetBrains Mono",monospace'; g.fillText(String(on.length), 104, 480);
    g.font = '700 76px Inter,sans-serif'; g.fillText(on.length === 1 ? 'THING' : 'DIFFERENT THINGS', 120, 580);
    g.font = '400 40px "JetBrains Mono",monospace';
    on.forEach((t, i) => { g.fillStyle = col('--red'); g.fillText('×', 120, 680 + i * 62); g.fillStyle = ink2; g.fillText(t, 170, 680 + i * 62); });
    g.fillStyle = C.edge; g.fillRect(120, 1060, 840, 2);
    g.fillStyle = ink; g.font = '600 44px Inter,sans-serif';
    const words = verdicts[on.length].split(' '); let line = '', y = 1124;
    for (const w of words) { if (g.measureText(line + w).width > 840) { g.fillText(line, 120, y); y += 54; line = ''; } line += `${w} `; }
    g.fillText(line, 120, y);
    g.fillStyle = C.hi; g.font = '700 30px "JetBrains Mono",monospace'; g.fillText('THE WAR ROOM', 120, 1232);
    g.fillStyle = ink3; g.textAlign = 'right'; g.fillText('WARROOM.SECUREPROSPECTIVE.COM', 960, 1232);
    return c;
  }
  $('shareimg').addEventListener('click', (e) => {
    const b = e.currentTarget;
    cardImage().toBlob(async (blob) => {
      const file = new File([blob], 'my-league-setup.png', { type: 'image/png' });
      try {
        if (navigator.canShare && navigator.canShare({ files: [file] })) await navigator.share({ files: [file], text: 'How dumb is yours? https://warroom.secureprospective.com' });
        else { const a = document.createElement('a'); a.href = URL.createObjectURL(blob); a.download = file.name; a.click(); setTimeout(() => URL.revokeObjectURL(a.href), 4000); }
        b.textContent = 'Saved. Post it.';
      } catch (err) { if (err.name !== 'AbortError') b.textContent = 'Could not save. Try again.'; }
    }, 'image/png');
  });

  // ---- wiring
  document.querySelectorAll('[data-goto]').forEach((b) => b.addEventListener('click', (e) => { e.preventDefault(); goTo(+b.dataset.goto); }));
  addEventListener('pointermove', (e) => { if (e.pointerType === 'mouse') { mx = e.clientX / W - 0.5; my = e.clientY / H - 0.5; } });
  addEventListener('resize', resize);
  resize(); card();
  cam = target();
  setInterval(play, 1700);
  for (let i = 0; i < 4; i++) { const seen = scenes[6].seen; scenes[6].seen = true; play(); scenes[6].seen = seen; }
  requestAnimationFrame(frame);
})();
