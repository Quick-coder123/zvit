// Колізії зі статичною геометрією світу (будинки міста). Спільне для сервера і клієнта,
// щоб будинки однаково зупиняли бійців, ДРГ і кулі.
import { CITY, inCity, distToLanes, CAMPS, DEPOSITS } from './data.js';

function srand(seed) { let s = seed >>> 0 || 1; return () => { s = (s * 16807) % 2147483647; return s / 2147483647; }; }

// rot — поворот об'єкта навколо Y у конвенції three.js (локальна X → (cos, 0, −sin)).
export function toLocal(x, z, b) {
  const dx = x - b.x, dz = z - b.z, c = Math.cos(b.rot), s = Math.sin(b.rot);
  return [dx * c - dz * s, dx * s + dz * c];
}
export function toWorld(lx, lz, b) {
  const c = Math.cos(b.rot), s = Math.sin(b.rot);
  return [b.x + lx * c + lz * s, b.z - lx * s + lz * c];
}
export function insideRect(x, z, b, pad = 0) {
  const [lx, lz] = toLocal(x, z, b);
  return Math.abs(lx) < b.w / 2 + pad && Math.abs(lz) < b.d / 2 + pad;
}
// Виштовхує точку з усіх прямокутників на найближчу грань (з відступом pad).
export function pushOutOfRects(x, z, rects, pad = 0) {
  for (const b of rects) {
    const hw = b.w / 2 + pad, hd = b.d / 2 + pad;
    const [lx, lz] = toLocal(x, z, b);
    if (Math.abs(lx) >= hw || Math.abs(lz) >= hd) continue;
    let nx = lx, nz = lz;
    if (hw - Math.abs(lx) < hd - Math.abs(lz)) nx = Math.sign(lx || 1) * hw; else nz = Math.sign(lz || 1) * hd;
    [x, z] = toWorld(nx, nz, b);
  }
  return [x, z];
}
// Чи перетинає відрізок AB прямокутник (метод плит у локальних координатах).
export function segmentHitsRect(ax, az, bx, bz, b, pad = 0) {
  const [x0, z0] = toLocal(ax, az, b), [x1, z1] = toLocal(bx, bz, b);
  const hw = b.w / 2 + pad, hd = b.d / 2 + pad;
  let t0 = 0, t1 = 1;
  for (const [p, d, h] of [[x0, x1 - x0, hw], [z0, z1 - z0, hd]]) {
    if (Math.abs(d) < 1e-9) { if (Math.abs(p) >= h) return false; continue; }
    let ta = (-h - p) / d, tb = (h - p) / d;
    if (ta > tb) [ta, tb] = [tb, ta];
    t0 = Math.max(t0, ta); t1 = Math.min(t1, tb);
    if (t0 > t1) return false;
  }
  return true;
}
export function segmentBlocked(ax, az, bx, bz, rects, pad = 0) {
  for (const b of rects) if (segmentHitsRect(ax, az, bx, bz, b, pad)) return b;
  return null;
}

// Детерміноване розміщення будинків зруйнованого міста.
let _city = null;
export function cityBuildings() {
  if (_city) return _city;
  const rnd = srand(9001);
  const ca = CITY.angle;
  const blockers = [...CAMPS.map((c) => [c.x, c.z, 8]), ...DEPOSITS.map((d) => [d.x, d.z, 10])];
  const out = [];
  for (let i = 0; i < 1200 && out.length < 30; i++) {
    const rr = CITY.rMin + 8 + rnd() * (CITY.rMax - CITY.rMin - 12);
    const aa = ca + (rnd() - 0.5) * CITY.spread * 1.7;
    const x = Math.cos(aa) * rr, z = Math.sin(aa) * rr;
    const big = rnd() < 0.45;
    const w = big ? 16 + rnd() * 10 : 7 + rnd() * 5, d = big ? 10 : 6 + rnd() * 4;
    const rot = -(ca + (rnd() < 0.5 ? 0 : Math.PI / 2) + (rnd() - 0.5) * 0.08);
    const floors = big ? 3 + Math.floor(rnd() * 3) : 1 + Math.floor(rnd() * 2);
    const kind = big ? (rnd() < 0.6 ? 'plaster' : 'conc') : 'brick';
    const seed = Math.floor(rnd() * 1e9);
    if (inCity(x, z) < 0.55) continue;
    const rad = Math.hypot(w, d) / 2;
    if (distToLanes(x, z) < rad + 4) continue;
    if (blockers.some(([bx, bz, br]) => Math.hypot(bx - x, bz - z) < rad + br)) continue;
    if (out.some((p) => Math.hypot(p.x - x, p.z - z) < rad + Math.hypot(p.w, p.d) / 2 + 1.5)) continue;
    out.push({ x, z, w, d, rot, floors, kind, big, seed });
  }
  _city = out;
  return out;
}
