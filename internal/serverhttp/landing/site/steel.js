/* steel.js: hand-built 3D polygon meshes (portal frame, goal post) drawn on a 2D canvas.
   World: x right, y DOWN (floor +430), z forward. Meshes are built in local space and
   drawn at a depth offset d0. Every portal part is built for x >= 0 and mirrored, so the
   left and right halves are exact mirrors by construction. */
(function (root) {
  'use strict';
  const TAU = Math.PI * 2;
  const unit = (x, y, z) => { const l = Math.hypot(x, y, z); return [x / l, y / l, z / l]; };
  const KEY = unit(0.25, -0.8, -0.55);   // toward the key light: above, camera side
  const FILL = unit(0, 0.6, -0.8);       // weak fill from below-front
  const MI = { steel: 0, edge: 1, hazard: 2, lamp: 4, bolt: 5, post: 6, pad: 7 };

  // ---------- face construction ----------
  // Newell normal; flipped if needed so it points away from `inside` (a point within the solid).
  function mkFace(pts, mat, det, inside, extra) {
    let nx = 0, ny = 0, nz = 0, cx = 0, cy = 0, cz = 0;
    const n = pts.length;
    for (let i = 0; i < n; i++) {
      const a = pts[i], b = pts[(i + 1) % n];
      nx += (a[1] - b[1]) * (a[2] + b[2]); ny += (a[2] - b[2]) * (a[0] + b[0]); nz += (a[0] - b[0]) * (a[1] + b[1]);
      cx += a[0]; cy += a[1]; cz += a[2];
    }
    cx /= n; cy /= n; cz /= n;
    let [ux, uy, uz] = unit(nx, ny, nz);
    if (ux * (cx - inside[0]) + uy * (cy - inside[1]) + uz * (cz - inside[2]) < 0) {
      pts = pts.slice().reverse(); ux = -ux; uy = -uy; uz = -uz;
    }
    const f = { pts, n: [ux, uy, uz], c: [cx, cy, cz], mat, detail: det ? 1 : 0 };
    return extra ? Object.assign(f, extra) : f;
  }
  function Mesh() { this.faces = []; }
  Mesh.prototype.add = function (pts, mat, det, inside, extra) {
    const f = mkFace(pts, mat, det, inside, extra); this.faces.push(f); return f;
  };

  // Sutherland-Hodgman clip of a convex polygon against plane p[ax] = val (keeps >= if ge).
  function clip(poly, ax, val, ge) {
    const out = [], n = poly.length;
    for (let i = 0; i < n; i++) {
      const a = poly[i], b = poly[(i + 1) % n];
      const ia = ge ? a[ax] >= val : a[ax] <= val, ib = ge ? b[ax] >= val : b[ax] <= val;
      if (ia) out.push(a);
      if (ia !== ib) { const t = (val - a[ax]) / (b[ax] - a[ax]); out.push(a.map((v, j) => v + (b[j] - v) * t)); }
    }
    return out;
  }
  const area2 = (p) => { let s = 0; for (let i = 0; i < p.length; i++) { const a = p[i], b = p[(i + 1) % p.length]; s += a[0] * b[1] - b[0] * a[1]; } return Math.abs(s / 2); };

  // Prism: convex cross-section sec [[u,v]...] extruded along axis from a0 to a1.
  // mats/dets: per side edge (null mat = side omitted). cap0/cap1: material or null.
  function prism(M, sec, axis, a0, a1, mats, dets, cap0, cap1, extra) {
    const P = (u, v, a) => axis === 'x' ? [a, u, v] : axis === 'y' ? [u, a, v] : [u, v, a];
    let su = 0, sv = 0; for (const p of sec) { su += p[0]; sv += p[1]; }
    const ins = P(su / sec.length, sv / sec.length, (a0 + a1) / 2), n = sec.length, out = [];
    const capDet = Array.isArray(dets) ? 0 : dets;
    for (let k = 0; k < n; k++) {
      const m = Array.isArray(mats) ? mats[k] : mats;
      if (!m) { out.push(null); continue; }
      const p = sec[k], q = sec[(k + 1) % n];
      out.push(M.add([P(p[0], p[1], a0), P(q[0], q[1], a0), P(q[0], q[1], a1), P(p[0], p[1], a1)], m,
        Array.isArray(dets) ? dets[k] : dets, ins, extra));
    }
    if (cap0) out.push(M.add(sec.map((p) => P(p[0], p[1], a0)), cap0, capDet, ins, extra));
    if (cap1) out.push(M.add(sec.map((p) => P(p[0], p[1], a1)), cap1, capDet, ins, extra));
    return out;
  }

  // Axis-aligned box; spec maps nx/px/ny/py/nz/pz to a material (missing = omitted).
  function box(M, x0, x1, y0, y1, z0, z1, spec, det, extra) {
    const ins = [(x0 + x1) / 2, (y0 + y1) / 2, (z0 + z1) / 2], r = {};
    const q = (k, pts) => { if (spec[k]) r[k] = M.add(pts, spec[k], det, ins, extra); };
    q('nx', [[x0, y0, z0], [x0, y1, z0], [x0, y1, z1], [x0, y0, z1]]);
    q('px', [[x1, y0, z0], [x1, y1, z0], [x1, y1, z1], [x1, y0, z1]]);
    q('ny', [[x0, y0, z0], [x1, y0, z0], [x1, y0, z1], [x0, y0, z1]]);
    q('py', [[x0, y1, z0], [x1, y1, z0], [x1, y1, z1], [x0, y1, z1]]);
    q('nz', [[x0, y0, z0], [x1, y0, z0], [x1, y1, z0], [x0, y1, z0]]);
    q('pz', [[x0, y0, z1], [x1, y0, z1], [x1, y1, z1], [x0, y1, z1]]);
    return r;
  }
  const all = (mat, omit) => { const s = {}; for (const k of ['nx', 'px', 'ny', 'py', 'nz', 'pz']) if (!(omit || '').includes(k)) s[k] = mat; return s; };

  // Rectangle section [u0,u1]x[v0,v1] with chosen corners chamfered (mask bit i = corner i:
  // 0 (u0,v0), 1 (u1,v0), 2 (u1,v1), 3 (u0,v1)). side = mats for faces v0, u1, v1, u0.
  function chamf(u0, u1, v0, v1, c, mask, side) {
    const C = [[u0, v0, 1, 1], [u1, v0, -1, 1], [u1, v1, -1, -1], [u0, v1, 1, -1]];
    const sec = [], mats = [], dets = [];
    for (let i = 0; i < 4; i++) {
      const [u, v, du, dv] = C[i];
      if (mask & (1 << i)) {
        const pin = i % 2 ? [u + du * c, v] : [u, v + dv * c], pout = i % 2 ? [u, v + dv * c] : [u + du * c, v];
        sec.push(pin); mats.push(side[(i + 3) % 4] && side[i] ? 'edge' : null); dets.push(1);
        sec.push(pout);
      } else sec.push([u, v]);
      mats.push(side[i]); dets.push(0);
    }
    return { sec, mats, dets };
  }

  // Diagonal hazard stripes (bright parallelograms) drawn as decals over a black host face.
  // map(x,t) -> 3D point on the host plane; stripes are clipped to [x0,x1], phase is global.
  function stripes(M, host, map, x0, x1, t0, t1, pitch, shear) {
    const ins = host.c.map((v, i) => v - host.n[i]);
    for (let k = Math.floor((x0 - Math.abs(shear)) / pitch) - 1; k * pitch < x1; k++) {
      const xa = k * pitch;
      let p = [[xa, t0], [xa + pitch / 2, t0], [xa + pitch / 2 + shear, t1], [xa + shear, t1]];
      p = clip(clip(p, 0, x0, true), 0, x1, false);
      if (p.length < 3 || area2(p) < 30) continue;
      M.add(p.map((q) => map(q[0], q[1])), 'hazard', 1, ins, { hz: 1, host });
    }
  }
  // Hexagon section around (cu,cv).
  const hex = (cu, cv, r) => { const s = []; for (let i = 0; i < 6; i++) { const a = i * TAU / 6 + Math.PI / 6; s.push([cu + r * Math.cos(a), cv + r * Math.sin(a)]); } return s; };

  // Mirror faces [from, to) across x = 0 (winding reversed so normals stay outward).
  function mirror(M, from, to) {
    const map = new Map();
    for (let i = from; i < to; i++) {
      const f = M.faces[i], g = Object.assign({}, f);
      g.pts = f.pts.map((p) => [-p[0], p[1], p[2]]).reverse();
      g.n = [-f.n[0], f.n[1], f.n[2]]; g.c = [-f.c[0], f.c[1], f.c[2]];
      if (f.host) g.host = map.get(f.host);
      map.set(f, g); M.faces.push(g);
    }
  }

  // Precompute draw data: static shade, material slot, z range, decal lists.
  function finalize(M) {
    const F = M.faces, idx = new Map();
    F.forEach((f, i) => idx.set(f, i));
    M.roots = []; M.kids = F.map(() => null);
    F.forEach((f, i) => {
      const n = f.n;
      const kx = KEY[0] * (f.c[0] < 0 ? -1 : 1);   // the key light's sideways lean is mirrored, so both halves shade alike
      const bounce = Math.max(0, -n[0] * Math.sign(f.c[0])) * 0.34 + Math.max(0, -n[2]) * 0.1;
      f.s = Math.min(1, 0.14 + 0.8 * Math.max(0, n[0] * kx + n[1] * KEY[1] + n[2] * KEY[2]) +
        0.2 * Math.max(0, n[0] * FILL[0] + n[1] * FILL[1] + n[2] * FILL[2]) + bounce);
      f.mi = MI[f.mat] + (f.mat === 'hazard' && f.hz ? 1 : 0);
      f.zmin = Math.min(...f.pts.map((p) => p[2])); f.zmax = Math.max(...f.pts.map((p) => p[2]));
      if (f.host) { const h = idx.get(f.host); (M.kids[h] = M.kids[h] || []).push(i); delete f.host; }
      else M.roots.push(i);
    });
    M.key = new Float64Array(F.length); M.ord = [];
    return M;
  }

  // ---------- Model 1: welded steel portal frame ----------
  function portal(o) {
    const M = new Mesh(), hs = o.halfSpan, fy = o.floorY, by = o.beamY;
    const xi = hs - 190, xe = xi - 20;           // column inner face; girder end (end plate face)
    const gt = by - 190, CH = 5;                 // girder top; chamfer size
    const S = 'steel';
    // Columns: H-section, flanges in y-z planes, web in x-y plane; base plate top to girder top.
    const yb = fy - 16;
    for (const [xa, xb] of [[xi, xi + 24], [hs - 24, hs]]) {
      const c = chamf(xa, xb, -85, 85, CH, 15, [S, S, S, S]);
      prism(M, c.sec, 'y', gt, yb, c.mats, c.dets, null, null);
    }
    box(M, xi + 24, hs - 24, gt, yb, -8, 8, { nz: S, pz: S });
    // Base plate with four anchor bolts (flush hex heads on square washers).
    const bc = hs - 95, bp = box(M, bc - 135, bc + 135, yb, fy, -125, 125, all(S, 'py'));
    for (const sx of [-1, 1]) for (const sz of [-1, 1]) {
      const u = bc + sx * 105, v = sz * 95;
      M.add([[u - 16, yb, v - 16], [u + 16, yb, v - 16], [u + 16, yb, v + 16], [u - 16, yb, v + 16]], 'bolt', 1, [u, fy, v], { host: bp.ny });
      M.add(hex(u, v, 11).map((q) => [q[0], yb, q[1]]), 'bolt', 1, [u, fy, v], { host: bp.ny });
    }
    // Cap plate over column top and end plate.
    box(M, xe, hs + 10, gt - 16, gt, -95, 95, all(S));
    // End (splice) plate at the girder end, 2x4 flush hex bolt heads on its -x face, z = -50 and +50.
    const ep = box(M, xe, xi, gt, by, -95, 95, all(S, 'ny px'));
    for (const z of [-50, 50]) for (const dy of [150, 115, 75, 40]) {
      M.add(hex(by - dy, z, 10).map((q) => [xe, q[0], q[1]]), 'bolt', 1, [xi, by - dy, z], { host: ep.nx });
    }
    // Girder: split at mid-span so painter sorting holds; flanges chamfered on exposed edges.
    const mid = Math.round(xe / 2);
    for (const [a0, a1] of [[0, mid], [mid, xe]]) {
      const top = chamf(gt, gt + 24, -85, 85, CH, 15, [S, S, S, S]);
      prism(M, top.sec, 'x', a0, a1, top.mats, top.dets, null, null);
      const bot = chamf(by - 24, by, -85, 85, CH, 12, [null, 'hazard', S, S]);
      const bf = prism(M, bot.sec, 'x', a0, a1, bot.mats, bot.dets, null, null);
      const under = bf.find((f) => f && f.mat === 'hazard');
      stripes(M, under, (x, t) => [x, by, t], a0, a1, -85, 80, 150, 165);
      box(M, a0, a1, gt + 24, by - 24, -8, 8, { nz: S, pz: S });
      // Front fascia plate (faces the approaching camera) carrying hazard stripes.
      const fa = box(M, a0, a1, by - 30, by, -93, -85, { nz: 'hazard', ny: S, py: S });
      stripes(M, fa.nz, (x, t) => [x, t, -93], a0, a1, by - 30, by, 150, 30);
    }
    // Web stiffeners (both faces of the web) at light positions and the quarter point.
    const st = [];
    for (const x of o.lights.filter((x) => x >= 0).concat([xe / 2])) if (!st.some((s) => Math.abs(s - x) < 80)) st.push(x);
    for (const s of st) for (const sz of [-1, 1]) {
      const z0 = sz < 0 ? -85 : 8, z1 = sz < 0 ? -8 : 85, sp = { px: S }; if (s > 0) sp.nx = S; sp[sz < 0 ? 'nz' : 'pz'] = S;
      box(M, s > 0 ? s - 6 : 0, s + 6, gt + 24, by - 24, z0, z1, sp, 1);
    }
    // Knee haunch: triangular web plate plus a flange strip along the hypotenuse.
    const H = 330, T = 20 * Math.SQRT2;
    prism(M, [[xi, by], [xi, by + H], [xi - H, by]], 'z', -8, 8, null, 0, S, S);
    const hf = prism(M, [[xi, by + H], [xi, by + H + T], [xi - H - T, by], [xi - H, by]], 'z', -85, 85, [null, S, null, S], 0, S, S);
    for (const [za, zb] of [[-85, -85 + CH + 2], [85 - CH - 2, 85]]) {
      M.add([[xi, by + H + T, za], [xi - H - T, by, za], [xi - H - T, by, zb], [xi, by + H + T, zb]], 'edge', 1, [xi - 40, by + 40, 0], { host: hf[1] });
    }
    // Weld toes: short plates where the haunch flange lands on the column and on the girder, so the joint is not a bare butt.
    box(M, xi - 14, xi, by + H - 10, by + H + T + 26, -85, 85, all(S, 'px'), 1);
    box(M, xi - H - T - 26, xi - H + 10, by, by + 14, -85, 85, all(S, 'ny'), 1);
    // Light bars on two hanger rods each; bottom and front faces emissive.
    for (const lx of o.lights.filter((x) => x >= 0)) {
      const x0 = lx > 0 ? lx - 150 : 0, sp = { px: S, ny: S, py: 'lamp', nz: 'lamp', pz: S };
      if (lx > 0) sp.nx = S;
      box(M, x0, lx + 150, by + 38, by + 66, -35, 35, sp);
      const hood = { px: S, ny: S, py: S, nz: S, pz: S }; if (lx > 0) hood.nx = S;
      box(M, lx > 0 ? x0 - 8 : 0, lx + 158, by + 26, by + 38, -43, 43, hood);
      for (const hx of lx > 0 ? [lx - 110, lx + 110] : [lx + 110]) box(M, hx - 3, hx + 3, by, by + 26, -3, 3, { nx: S, px: S, nz: S, pz: S }, 1);
    }
    mirror(M, 0, M.faces.length);
    return finalize(M);
  }

  // ---------- Model 2: sling-shot goal post ----------
  const NS = 14, OFF = Math.PI / 2 + Math.PI / NS;    // ring phase chosen so rings are x-mirror symmetric
  function ring(P, u, v, r) {
    const out = [];
    for (let k = 0; k < NS; k++) {
      const a = OFF + k * TAU / NS, c = Math.cos(a) * r, s = Math.sin(a) * r;
      out.push([P[0] + c * u[0] + s * v[0], P[1] + c * u[1] + s * v[1], P[2] + c * u[2] + s * v[2]]);
    }
    return out;
  }
  // Quads between two rings; inside = point on the axis; matFn(k) may vary the material.
  function band(M, A, B, inside, mat, det, extraFn) {
    const out = [];
    for (let k = 0; k < NS; k++) {
      const j = (k + 1) % NS;
      out.push(M.add([A[k], A[j], B[j], B[k]], mat, det || 0, inside, extraFn ? extraFn(k) : null));
    }
    return out;
  }
  const midp = (a, b) => [(a[0] + b[0]) / 2, (a[1] + b[1]) / 2, (a[2] + b[2]) / 2];
  // Straight tube from P0 to P1 split into ~300-unit segments (rings share the u,v frame).
  function tube(M, P0, P1, u, v, r, mat, seg) {
    const n = Math.max(1, Math.ceil(Math.hypot(P1[0] - P0[0], P1[1] - P0[1], P1[2] - P0[2]) / (seg || 300)));
    let prev = ring(P0, u, v, r), pp = P0;
    for (let i = 1; i <= n; i++) {
      const P = P0.map((a, j) => a + (P1[j] - a) * i / n), R = ring(P, u, v, r);
      band(M, prev, R, midp(pp, P), mat); prev = R; pp = P;
    }
    return prev;
  }

  function goalpost(o) {
    const M = new Mesh(), fy = o.floorY, cy = o.crossbarY, hw = o.halfWidth, ty = o.topY, sb = o.setback;
    const X = [1, 0, 0], Y = [0, 1, 0], Z = [0, 0, 1], UP = [0, -1, 0];
    const R = Math.min(sb * 0.55, (fy - cy) * 0.4), y1 = cy + R;   // gooseneck sweep radius, bend start
    const rp = 22, rpad = 46, rfl = 66, flT = 12, rc = 26;
    const padTop = fy - Math.min(260, (fy - y1) * 0.7);
    const C = (y) => [0, y, sb];
    // Floor flange ring with anchor bolts.
    const f0 = ring(C(fy), X, Z, rfl), f1 = ring(C(fy - flT), X, Z, rfl);
    band(M, f0, f1, C(fy - flT / 2), 'steel');
    const p0 = ring(C(fy - flT), X, Z, rpad);
    const ann = band(M, f1, p0, C(fy), 'steel');
    for (let k = 0; k < 4; k++) {
      const a = Math.PI / 4 + k * Math.PI / 2, u = Math.cos(a) * 56, v = sb + Math.sin(a) * 56;
      let host = ann[0], best = 1e9;
      for (const f of ann) { const d = Math.hypot(f.c[0] - u, f.c[2] - v); if (d < best) { best = d; host = f; } }
      prism(M, hex(u, v, 7), 'y', fy - flT, fy - flT - 6, 'bolt', 1, null, 'bolt', { host });
    }
    // Pad: body, hazard band near the top (alternating blocks), capped with an annulus to the post.
    const hz0 = padTop + 44, hz1 = padTop + 14;
    const pa = ring(C(hz0), X, Z, rpad), pb = ring(C(hz1), X, Z, rpad), pt = ring(C(padTop), X, Z, rpad);
    band(M, p0, pa, C((fy + hz0) / 2), 'pad');
    band(M, pa, pb, C((hz0 + hz1) / 2), 'hazard', 0, (k) => ({ hz: k % 2 }));
    band(M, pb, pt, C((hz1 + padTop) / 2), 'pad');
    band(M, pt, ring(C(padTop), X, Z, rp), C(padTop + 5), 'pad');
    // Base post up to the bend, then a 90-degree gooseneck (8 segments, shared x axis: no twist).
    let prev = tube(M, C(padTop), C(y1), X, Z, rp, 'post'), pp = C(y1);
    for (let i = 1; i <= 8; i++) {
      const a = i / 8 * Math.PI / 2, P = [0, y1 - R * Math.sin(a), sb - R + R * Math.cos(a)];
      const Rg = ring(P, X, [0, -Math.sin(a), Math.cos(a)], rp);
      band(M, prev, Rg, midp(pp, P), 'post'); prev = Rg; pp = P;
    }
    // Horizontal arm forward to the collar; its capped end sits just inside the collar surface.
    const zEnd = Math.sqrt(rc * rc - rp * rp), end = [0, cy, zEnd];
    const armEnd = tube(M, pp, end, X, UP, rp, 'post');
    M.add(armEnd, 'post', 0, [0, cy, zEnd + 10]);
    const half = M.faces.length;
    // Right half: collar, crossbar, mitre-free swept elbow, upright, domed cap.
    const xc = 58, Rc = 54, O = (x) => [x, cy, 0];
    tube(M, O(0), O(xc), Z, Y, rc, 'post');
    band(M, ring(O(xc), Z, Y, rc), ring(O(xc), Z, Y, 16), O(xc - 5), 'post');
    prev = tube(M, O(xc), O(hw - Rc), Z, Y, 16, 'post', 400); pp = O(hw - Rc);
    for (let i = 1; i <= 3; i++) {
      const a = i / 3 * Math.PI / 2, P = [hw - Rc + Rc * Math.sin(a), cy - Rc + Rc * Math.cos(a), 0];
      const Rg = ring(P, Z, [Math.sin(a), Math.cos(a), 0], 16 - i);
      band(M, prev, Rg, midp(pp, P), 'post'); prev = Rg; pp = P;
    }
    const top = tube(M, pp, [hw, ty, 0], Z, X, 13, 'post', 450);
    const dome = ring([hw, ty - 4, 0], Z, X, 8.5);
    band(M, top, dome, [hw, ty + 4, 0], 'post');
    M.add(dome, 'post', 0, [hw, ty, 0]);
    mirror(M, half, M.faces.length);
    return finalize(M);
  }

  // ---------- drawing ----------
  const FOG = [8, 9, 12];   // hsl(220,22%,4%)
  function hsl(h, s, l) {
    s /= 100; l /= 100;
    const k = (n) => (n + h / 30) % 12, a = s * Math.min(l, 1 - l);
    const f = (n) => 255 * (l - a * Math.max(-1, Math.min(k(n) - 3, 9 - k(n), 1)));
    return [f(0), f(8), f(4)];
  }
  // Material colour at shade s (0..1) and light level on (0..1), before fog.
  function matRGB(mi, s, on) {
    const e = s * (0.3 + 0.7 * on), ds = 0.45 + 0.55 * on;
    switch (mi) {
      case 0: return hsl(220, 12 * ds, 5 + 27 * e);
      case 1: return hsl(220, 8 * ds, 8 + 54 * e);
      case 2: return hsl(220, 25, 1.5 + 2 * e);
      case 3: return hsl(220, 10 - 2 * on, (28 + 54 * on) * (0.38 + 0.62 * s));
      case 4: return hsl(220, 10 - 2 * on, 16 + 72 * on);
      case 5: return hsl(220, 10 * ds, 7 + 29 * e);
      case 6: {
        const a = hsl(30 + 8 * s, 74 + 18 * s, 16 + 44 * s), b = hsl(220, 10, 8 + 20 * s);
        return a.map((v, i) => b[i] + (v - b[i]) * on);
      }
      default: return hsl(220, 18 * ds, 4 + 10 * e);
    }
  }
  const CACHE = [];
  function colour(f, d, on, far) {
    const si = Math.round(f.s * 23), oi = Math.round(on * 10);
    const fg = Math.pow(Math.max(0, 1 - Math.max(0, d) / far), 1.7), fi = Math.round(fg * 31);
    const key = ((f.mi * 24 + si) * 11 + oi) * 32 + fi;
    let c = CACHE[key];
    if (!c) {
      const rgb = matRGB(f.mi, si / 23, oi / 10), w = fi / 31;
      c = CACHE[key] = 'rgb(' + rgb.map((v, i) => Math.round(FOG[i] + (v - FOG[i]) * w)).join(',') + ')';
    }
    return c;
  }

  function paint(ctx, f, v, d0, on, zNear) {
    let pts = f.pts;
    if (f.zmin < zNear) { pts = clip(pts, 2, zNear, true); if (pts.length < 3) return; }
    const hx = v.W / 2, hy = v.H / 2 + v.yOff;
    ctx.beginPath();
    for (let i = 0; i < pts.length; i++) {
      const p = pts[i], k = v.P / (v.P + d0 + p[2]) * v.S;
      const sx = (hx + (p[0] - v.camX) * k) * v.dpr, sy = (hy + (p[1] - v.camY) * k) * v.dpr;
      if (i) ctx.lineTo(sx, sy); else ctx.moveTo(sx, sy);
    }
    ctx.closePath();
    ctx.fillStyle = ctx.strokeStyle = colour(f, d0 + f.c[2], on, v.far);
    ctx.fill(); ctx.stroke();
  }

  // Cull, depth-sort (painter's), shade, fog, near-clip and fill each visible face.
  function draw(ctx, mesh, v, d0, on) {
    ctx.globalAlpha = 1;
    if (d0 > v.far) return;
    const F = mesh.faces, key = mesh.key, ord = mesh.ord, roots = mesh.roots;
    const ex = v.camX, ey = v.camY, ez = -v.P - d0, zNear = v.near - d0, lim = 0.22;
    const vis = (f) => {
      if (f.zmax < zNear) return false;
      const n = f.n, c = f.c;
      if (n[0] * (ex - c[0]) + n[1] * (ey - c[1]) + n[2] * (ez - c[2]) <= 0) return false;
      return !f.detail || v.P / (v.P + Math.max(v.near, d0 + c[2])) * v.S * v.dpr >= lim;
    };
    let m = 0;
    for (let r = 0; r < roots.length; r++) {
      const i = roots[r], f = F[i];
      if (!vis(f)) continue;
      const dx = ex - f.c[0], dy = ey - f.c[1], dz = ez - f.c[2];
      key[i] = dx * dx + dy * dy + dz * dz; ord[m++] = i;
    }
    ord.length = m;
    ord.sort((a, b) => key[b] - key[a]);
    ctx.lineWidth = 1; ctx.lineJoin = 'round';
    for (let j = 0; j < m; j++) {
      const i = ord[j];
      paint(ctx, F[i], v, d0, on, zNear);
      const kids = mesh.kids[i];
      if (kids) for (const q of kids) if (vis(F[q])) paint(ctx, F[q], v, d0, on, zNear);
    }
    ctx.globalAlpha = 1;
  }

  root.TWRSteel = { portal, goalpost, draw };
})(typeof window !== 'undefined' ? window : globalThis);
