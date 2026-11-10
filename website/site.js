// TheWarRoom landing page: one camera flying 110 yards down a field.
// Scroll position drives the camera; the DOM scenes and the canvas field share one projection.
(() => {
  'use strict';
  const P = 1000;            // perspective distance, design units (matches .lens in site.css)
  const D = 2600;            // depth between scenes = 10 yards
  const YARD = D / 10;
  const FLOOR = 430, SIDE = 1500, WALL = 2100, HASH = 420, FAR = 9500;
  const HY = [0, 0, 0, 0, 0, -240, -150, -150, -150, -110, -300, -220];   // camera height at each scene (negative is up)
  const SEG_VH = 1.5;        // scroll length per scene, in viewport heights
  const still = matchMedia('(prefers-reduced-motion: reduce)').matches;

  const $ = (id) => document.getElementById(id);
  const lens = $('lens'), world = $('world'), cv = $('field'), ctx = cv.getContext('2d');
  const hud = $('hud'), hudCh = $('hud-ch'), hudYd = $('hud-yd'), hudUs = $('hud-us');
  const css = getComputedStyle(document.documentElement);
  const col = (n) => css.getPropertyValue(n).trim();
  const C = { canvas: col('--canvas'), sunken: col('--sunken'), edge: col('--edge'), hi: col('--edge-hi'), amber: col('--amber'), black: col('--black') };

  // ---- scenes and their layers
  const scenes = [...world.querySelectorAll('.scene')].map((el, i) => {
    el.style.transform = `translate3d(0,${HY[i]}px,${-i * D}px)`;
    const layers = [...el.querySelectorAll('.ly')].map((l) => ({ el: l, z: parseFloat(l.style.getPropertyValue('--z')) || 0, o: -1 }));
    return { el, i, layers, near: false, seen: false, on: false };
  });
  const N = scenes.length;
  const motes = Array.from({ length: 110 }, (_, i) => ({ x: (i * 7919) % 3800 - 1900, y: -900 + (i * 104729) % 1330, z: (i * 3571) % 7000, s: 6 + (i % 4) * 4, c: i % 7 }));

  // ---- layout
  let W = 0, H = 0, S = 1, dpr = 1, ROOF = -900;
  function resize() {
    W = innerWidth; H = innerHeight; dpr = Math.min(devicePixelRatio || 1, 2);
    const portrait = H > W;
    S = portrait ? Math.min(W / 580, H / 1060) : Math.min(W / 1440, H / 860);
    ROOF = portrait ? -1700 : -900;   // a taller hall on phones keeps the beams clear of the words
    lens.style.transform = `translateY(23px) scale(${S})`;
    cv.width = W * dpr; cv.height = H * dpr;
    $('track').style.height = `${((N - 1) * SEG_VH + 1) * H}px`;
  }
  const segPx = () => SEG_VH * H;
  function goTo(i) { scrollTo({ top: i * segPx(), behavior: still ? 'auto' : 'smooth' }); }

  // ---- camera
  let cam = 0, camX = 0, camY = 0, mx = 0, my = 0, shake = 0, litZ = 0, litGoal = 0, last = 0;
  const smooth = (t) => t * t * t * (t * (t * 6 - 15) + 10);
  function camH(z) {
    const p = Math.max(0, Math.min(N - 1, z / D)), i = Math.min(N - 2, Math.floor(p));
    return HY[i] + (HY[i + 1] - HY[i]) * smooth(p - i);
  }
  function target() {
    const p = Math.max(0, Math.min(N - 1, scrollY / segPx()));
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

  function portal(d, on, label) {
    if (d < NEAR + 80 || d > FAR) return;
    const k = K(d) * dpr, a = fog(d);
    const x0 = X(-WALL, d), x1 = X(WALL, d), top = Y(ROOF - 700, d), bot = Y(ROOF + 150, d), floor = Y(FLOOR, d), cw = 190 * k;
    ctx.globalAlpha = a; ctx.fillStyle = on ? 'hsl(220,18%,10%)' : 'hsl(220,18%,7%)';
    ctx.fillRect(x0, top, x1 - x0, bot - top); ctx.fillRect(x0, bot, cw, floor - bot); ctx.fillRect(x1 - cw, bot, cw, floor - bot);
    ctx.strokeStyle = on ? C.hi : C.edge; ctx.lineWidth = Math.max(1, 3 * k);
    ctx.beginPath(); ctx.moveTo(x0 + cw, floor); ctx.lineTo(x0 + cw, bot); ctx.lineTo(x1 - cw, bot); ctx.lineTo(x1 - cw, floor); ctx.stroke();
    const hz = 30 * k;   // hazard edge under the beam
    if (hz > 3) {
      ctx.save(); ctx.beginPath(); ctx.rect(x0 + cw, bot - hz, x1 - x0 - 2 * cw, hz); ctx.clip();
      ctx.fillStyle = C.black; ctx.fillRect(x0, bot - hz, x1 - x0, hz); ctx.fillStyle = on ? C.hi : C.edge;
      for (let x = x0; x < x1; x += hz * 1.6) { ctx.beginPath(); ctx.moveTo(x, bot); ctx.lineTo(x + hz * 0.8, bot); ctx.lineTo(x + hz * 1.8, bot - hz); ctx.lineTo(x + hz, bot - hz); ctx.fill(); }
      ctx.restore();
    }
    for (let b = -3; b <= 3; b++) {   // light bars
      const w = 300 * k, h = 40 * k, x = X(b * 470, d) - w / 2, y = bot + 26 * k;
      if (on) { ctx.fillStyle = C.hi; ctx.fillRect(x, y, w, h); } else { ctx.strokeStyle = C.edge; ctx.lineWidth = Math.max(1, 3 * k); ctx.strokeRect(x, y, w, h); }
    }
    if (d < 2000) {
      ctx.fillStyle = on ? C.hi : C.edge; ctx.font = `700 ${150 * k}px "JetBrains Mono",monospace`; ctx.textAlign = 'center'; ctx.textBaseline = 'middle';
      ctx.fillText(label, X(0, d), bot - 130 * k);
    }
  }

  function drawField(now) {
    ctx.globalAlpha = 1; ctx.fillStyle = C.sunken; ctx.fillRect(0, 0, cv.width, cv.height);
    const SL = 5 * YARD, td = cam > 10 * D - 700;
    const sMax = Math.min(25, Math.ceil((cam + FAR) / SL)), sMin = Math.max(-2, Math.floor((cam + NEAR) / SL));
    for (let s = sMax; s >= sMin; s--) {
      const yd = s * 5, z = s * SL, d1 = Math.max(z - cam, NEAR), d2 = z + SL - cam;
      if (d2 <= d1) continue;
      const f = fog((d1 + d2) / 2), on = lit(z), ez = yd < 0 || yd >= 100, onField = yd >= -10 && yd < 110;
      const line = on ? C.hi : C.edge;
      if (s === 24) {   // goalposts behind the end zone
        const gd = z - cam, c = litZ > 100 * YARD ? C.amber : C.edge;
        if (H > W && gd < 4200) continue;   // on phones the posts would cross the last screen's words
        seg(0, FLOOR, gd, 0, FLOOR - 330, gd, 26, c, 1); seg(-430, FLOOR - 330, gd, 430, FLOOR - 330, gd, 26, c, 1);
        seg(-430, FLOOR - 330, gd, -430, ROOF - 300, gd, 22, c, 1); seg(430, FLOOR - 330, gd, 430, ROOF - 300, gd, 22, c, 1);
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
      const d = ((m.z - cam) % 7000 + 7000) % 7000 - 700, party = td && cam + d > 96 * YARD;
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
    cam = still ? tg : cam + (tg - cam) * (1 - Math.exp(-dt * 9));
    if (!still) {
      camX += (mx * 90 - camX) * (1 - Math.exp(-dt * 5));
      shake *= Math.exp(-dt * 14);
      }
    camY = camH(cam) + (still ? 0 : my * 45 + (shake > 0.3 ? Math.sin(now / 16) * shake : 0));
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
      if (on !== s.on) { s.on = on; s.el.classList.toggle('on', on); }
      if (d < 750 && d > -1500 && !s.seen) {
        s.seen = true; s.el.classList.add('seen');
        if ('jolt' in s.el.dataset && !still) shake = 9;
      } else if (s.seen && (d > 2300 || d < -1900)) { s.seen = false; s.el.classList.remove('seen'); }
      for (const l of s.layers) {
        const dl = d - l.z;
        const farF = still ? (on ? 1 : 0) : dl <= 300 ? 1 : dl >= 1500 ? 0 : (1 - (dl - 300) / 1200) ** 2;
        const nearF = dl >= -330 ? 1 : dl <= -760 ? 0 : (dl + 760) / 430;
        const o = Math.round(farF * nearF * 100) / 100;
        if (o !== l.o) { l.o = o; l.el.style.opacity = o; }
      }
    }
    litGoal = cam > 5 * D - 900 ? 126 * YARD : 0;
    updateHud(cur);
    drawField(now);
    requestAnimationFrame(frame);
  }

  // ---- scorebug
  let hudKey = '';
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
    ['Montana to Rice, 18 yds over the middle', '+2.8', 1, 10], ['Payton up the middle for 4 yds', '+0.4', 1, 3],
    ['Marino short right to Largent for 6 yds', '', 0, 4], ['Reggie White sacks the quarterback, -8 yds', '+2.0', 1, -6],
    ['Barry Sanders left end for 9 yds', '+0.9', 1, 6], ['Montana deep left to Rice, 31 yds', '+4.1', 1, 14],
    ['Emmitt Smith right side for 3 yds', '', 0, 2], ['Elway to Sharpe, 22 yds, down at the 9', '+3.2', 1, 9],
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
    try { if (navigator.share) await navigator.share({ text }); else await navigator.clipboard.writeText(text); b.textContent = 'Copied. Go start something.'; }
    catch { b.textContent = 'Could not copy. Try again.'; }
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
