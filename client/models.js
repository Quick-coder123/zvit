// Процедурні моделі сучасної війни з фототекстурами: бійці, техніка, дрони, вогневі позиції.
// Кожна модель зливається в кілька сіток (вершинні кольори) — мало викликів малювання.
import * as THREE from 'three';
import { mergeGeometries } from 'three/addons/utils/BufferGeometryUtils.js';
import { A, texRep } from './assets.js';
import { patchWeather } from './scene.js';

// ───────── матеріали ─────────
let M = null;
function camoTexture(cols, seed) {
  const c = document.createElement('canvas');
  c.width = c.height = 256;
  const g = c.getContext('2d');
  g.fillStyle = cols[0]; g.fillRect(0, 0, 256, 256);
  let s = seed;
  const r = () => { s = (s * 16807) % 2147483647; return s / 2147483647; };
  for (let k = 1; k < cols.length; k++) {
    g.fillStyle = cols[k];
    for (let i = 0; i < 40; i++) {
      const x = r() * 256, y = r() * 256, w = 8 + r() * 30, h = 5 + r() * 16;
      g.beginPath(); g.ellipse(x, y, w, h, r() * 3, 0, Math.PI * 2); g.fill();
      g.beginPath(); g.ellipse((x + 128) % 256, y, w * 0.6, h * 0.6, r() * 3, 0, Math.PI * 2); g.fill();
    }
  }
  const t = new THREE.CanvasTexture(c);
  t.colorSpace = THREE.SRGBColorSpace;
  t.wrapS = t.wrapT = THREE.RepeatWrapping;
  t.repeat.set(3, 3);
  return t;
}
function mats() {
  if (M) return M;
  const nor = texRep('denim_fabric', 'nor', 6);
  const cloth = (cols, seed) => patchWeather(new THREE.MeshStandardMaterial({ map: camoTexture(cols, seed), normalMap: nor, normalScale: new THREE.Vector2(0.6, 0.6), roughness: 0.92, vertexColors: true }));
  M = {
    clothAlly: cloth(['#6b6a4c', '#4f5338', '#867c5a', '#3a3a2c', '#5d6a44'], 11),
    clothEnemy: cloth(['#7d7766', '#5f5b4e', '#9a917b', '#4a4840'], 23),
    clothDRG: cloth(['#4b4a38', '#3a3a2c', '#5a5642', '#2e2f25'], 37),
    plain: patchWeather(new THREE.MeshStandardMaterial({ roughness: 0.75, metalness: 0.05, vertexColors: true })),
    gun: patchWeather(new THREE.MeshStandardMaterial({ roughness: 0.45, metalness: 0.6, vertexColors: true })),
    paint: patchWeather(new THREE.MeshStandardMaterial({ map: texRep('green_metal_rust', 'diff', 1), normalMap: texRep('green_metal_rust', 'nor', 1), roughnessMap: texRep('green_metal_rust', 'rough', 1), metalness: 0.35, vertexColors: true })),
    steel: patchWeather(new THREE.MeshStandardMaterial({ map: texRep('rusty_metal_02', 'diff', 1), normalMap: texRep('rusty_metal_02', 'nor', 1), roughness: 0.7, metalness: 0.55, vertexColors: true })),
    rubber: patchWeather(new THREE.MeshStandardMaterial({ roughness: 0.95, metalness: 0, vertexColors: true })),
    glass: new THREE.MeshStandardMaterial({ color: 0x1a2228, roughness: 0.08, metalness: 0.6 }),
    canvas: patchWeather(new THREE.MeshStandardMaterial({ normalMap: nor, roughness: 0.95, vertexColors: true })),
  };
  M.burnt = new THREE.MeshStandardMaterial({ map: texRep('rusty_metal_02', 'diff', 1), color: 0x3a2f29, roughness: 0.95, metalness: 0.3 });
  return M;
}
export function burntMaterial() { return mats().burnt; }

// ───────── збирач геометрії ─────────
const _m = new THREE.Matrix4(), _q = new THREE.Quaternion(), _e = new THREE.Euler(), _v = new THREE.Vector3(), _s = new THREE.Vector3();
class B {
  constructor() { this.parts = {}; }
  add(mat, geo, color = 0xffffff, x = 0, y = 0, z = 0, rx = 0, ry = 0, rz = 0, sx = 1, sy = 1, sz = 1) {
    const g = geo.index ? geo.toNonIndexed() : geo.clone();
    _e.set(rx, ry, rz); _q.setFromEuler(_e);
    g.applyMatrix4(_m.compose(_v.set(x, y, z), _q, _s.set(sx, sy, sz)));
    const c = new THREE.Color(color);
    const n = g.attributes.position.count, col = new Float32Array(n * 3);
    for (let i = 0; i < n; i++) { col[i * 3] = c.r; col[i * 3 + 1] = c.g; col[i * 3 + 2] = c.b; }
    g.setAttribute('color', new THREE.BufferAttribute(col, 3));
    if (!g.attributes.uv) g.setAttribute('uv', new THREE.BufferAttribute(new Float32Array(n * 2), 2));
    for (const k of Object.keys(g.attributes)) if (!['position', 'normal', 'uv', 'color'].includes(k)) g.deleteAttribute(k);
    (this.parts[mat] ||= []).push(g);
    return this;
  }
  // циліндр між двома точками
  limb(mat, a, b, r, color, r2 = r, seg = 8) {
    const va = new THREE.Vector3(...a), vb = new THREE.Vector3(...b);
    const len = va.distanceTo(vb);
    const g = new THREE.CylinderGeometry(r2, r, len, seg);
    g.translate(0, len / 2, 0);
    const q = new THREE.Quaternion().setFromUnitVectors(new THREE.Vector3(0, 1, 0), vb.clone().sub(va).normalize());
    g.applyQuaternion(q).translate(va.x, va.y, va.z);
    return this.add(mat, g, color);
  }
  build(shadow = true) {
    const grp = new THREE.Group();
    const m = mats();
    for (const [k, list] of Object.entries(this.parts)) {
      const mesh = new THREE.Mesh(mergeGeometries(list), m[k]);
      mesh.castShadow = shadow; mesh.receiveShadow = true;
      grp.add(mesh);
    }
    return grp;
  }
}
const box = (w, h, d) => new THREE.BoxGeometry(w, h, d);
const cyl = (a, b, h, s = 10) => new THREE.CylinderGeometry(a, b, h, s);
const sph = (r, w = 12, h = 8) => new THREE.SphereGeometry(r, w, h);
const cap = (r, l) => new THREE.CapsuleGeometry(r, l, 4, 8);
function extrude(points, depth, bevel = 0.05) {
  const sh = new THREE.Shape(points.map(([x, y]) => new THREE.Vector2(x, y)));
  const g = new THREE.ExtrudeGeometry(sh, { depth, bevelEnabled: bevel > 0, bevelSize: bevel, bevelThickness: bevel, bevelSegments: 1 });
  g.translate(0, 0, -depth / 2);
  g.rotateY(-Math.PI / 2); // профіль по осі X → уздовж Z (вперед)
  return g;
}

// ───────── зброя ─────────
function rifle(b, mat, x, y, z, ry = 0, len = 1, kind = 'ar') {
  const add = (g, c, px, py, pz, rx = 0) => b.add(mat, g, c, x + Math.sin(ry) * pz + Math.cos(ry) * px, y + py, z + Math.cos(ry) * pz - Math.sin(ry) * px, rx, ry);
  const dark = 0x222320, poly = 0x2c2b27;
  add(box(0.055, 0.1, 0.36 * len), dark, 0, 0, 0);
  add(box(0.06, 0.07, 0.26 * len), poly, 0, 0.01, 0.3 * len);
  add(cyl(0.013, 0.013, 0.32 * len, 6), dark, 0, 0.015, 0.55 * len, Math.PI / 2);
  add(box(0.035, 0.15, 0.06), poly, 0, -0.1, 0.06, 0.25);
  add(box(0.045, 0.09, 0.24), poly, 0, -0.02, -0.28);
  add(cyl(0.022, 0.022, 0.12, 8), dark, 0, 0.085, 0.04, Math.PI / 2);
  if (kind === 'mg') { add(box(0.09, 0.1, 0.12), 0x3b3d30, 0.06, -0.08, 0.02); add(cyl(0.016, 0.016, 0.25, 6), dark, 0, 0.015, 0.75 * len, Math.PI / 2); }
}

// ───────── боєць ─────────
const SKIN = 0xc49a7a, BOOT = 0x2a2620, GEAR = 0xb8b39a;
export function makeSoldier({ side = 'ally', color = 0xff0000, backpack = false, kind = 'ar', crew = false, pose = 'stand' } = {}) {
  const cloth = side === 'ally' ? 'clothAlly' : side === 'drg' ? 'clothDRG' : 'clothEnemy';
  const root = new THREE.Group();
  const up = new B();
  // тулуб
  up.add(cloth, cap(0.16, 0.34), 0xffffff, 0, 1.28, 0, 0, 0, 0, 1.15, 1, 0.78);
  up.add(cloth, box(0.4, 0.42, 0.3), GEAR, 0, 1.3, 0.01);
  for (const px of [-0.12, 0, 0.12]) up.add(cloth, box(0.1, 0.14, 0.07), GEAR * 1 + 0, px, 1.17, 0.18);
  up.add('plain', box(0.36, 0.06, 0.26), 0x3a3628, 0, 1.06, 0);
  // голова
  up.add('plain', sph(0.1, 12, 10), side === 'drg' ? 0x2a2a26 : SKIN, 0, 1.64, 0.02, 0, 0, 0, 1, 1.1, 1);
  up.add(cloth, new THREE.SphereGeometry(0.13, 14, 8, 0, Math.PI * 2, 0, Math.PI * 0.55), 0xd0ccb0, 0, 1.67, 0.0);
  up.add('plain', cyl(0.134, 0.134, 0.03, 14), 0x45463a, 0, 1.64, 0);
  if (side !== 'drg') up.add('plain', box(0.14, 0.03, 0.04), 0x1c1c1c, 0, 1.66, 0.11);
  if (backpack) {
    up.add(cloth, box(0.3, 0.4, 0.18), 0xa8a48a, 0, 1.32, -0.24);
    up.add('plain', cyl(0.008, 0.008, 0.5, 4), 0x1a1a1a, 0.1, 1.7, -0.28);
  }
  // руки тримають зброю
  const sh = [[-0.22, 1.45, 0], [0.22, 1.45, 0]];
  if (crew) {
    up.limb(cloth, sh[0], [-0.2, 1.2, 0.25], 0.055, 0xffffff);
    up.limb(cloth, [-0.2, 1.2, 0.25], [-0.08, 1.25, 0.45], 0.048, 0xffffff);
    up.limb(cloth, sh[1], [0.2, 1.2, 0.2], 0.055, 0xffffff);
    up.limb(cloth, [0.2, 1.2, 0.2], [0.08, 1.25, 0.4], 0.048, 0xffffff);
  } else {
    up.limb(cloth, sh[0], [-0.18, 1.22, 0.22], 0.055, 0xffffff);
    up.limb(cloth, [-0.18, 1.22, 0.22], [-0.02, 1.3, 0.48], 0.048, 0xffffff);
    up.limb(cloth, sh[1], [0.2, 1.2, 0.05], 0.055, 0xffffff);
    up.limb(cloth, [0.2, 1.2, 0.05], [0.06, 1.28, 0.2], 0.048, 0xffffff);
    up.add('plain', sph(0.045, 6, 4), SKIN, -0.02, 1.3, 0.48);
    up.add('plain', sph(0.045, 6, 4), SKIN, 0.06, 1.28, 0.2);
    rifle(up, 'gun', 0.03, 1.32, 0.3, 0, 1, kind);
  }
  // пов'язка
  up.add('plain', cyl(0.064, 0.064, 0.06, 10), color, -0.21, 1.36, 0.04, 0, 0, 0.35);
  if (side === 'ally') up.add('plain', cyl(0.064, 0.064, 0.06, 10), color, 0.21, 1.36, 0.04, 0, 0, -0.35);
  const upper = up.build();
  root.add(upper);
  // ноги (шарнір у кульшовому суглобі)
  const legs = [];
  for (const sx of [-1, 1]) {
    const lb = new B();
    lb.add(cloth, cap(0.085, 0.34), 0xffffff, 0, -0.24, 0);
    lb.add(cloth, cap(0.07, 0.34), 0xffffff, 0, -0.66, 0.0);
    lb.add('plain', box(0.12, 0.1, 0.06), 0x3a3a30, 0, -0.45, 0.07);
    lb.add('plain', box(0.12, 0.11, 0.28), BOOT, 0, -0.9, 0.05);
    const leg = lb.build();
    const pivot = new THREE.Group();
    pivot.position.set(sx * 0.1, 0.96, 0);
    pivot.add(leg);
    if (pose === 'kneel') { pivot.rotation.x = sx < 0 ? -1.4 : -0.2; }
    root.add(pivot);
    legs.push(pivot);
  }
  if (pose === 'kneel') { upper.position.y = -0.45; legs.forEach((l) => (l.position.y -= 0.45)); }
  root.userData = { legs, upper, kind: 'soldier' };
  return root;
}

export function makeHero(color) {
  const g = new THREE.Group();
  const s = makeSoldier({ side: 'ally', color, backpack: true });
  s.scale.setScalar(1.25);
  g.add(s);
  const ring = new THREE.Mesh(new THREE.RingGeometry(0.85, 1.0, 40), new THREE.MeshBasicMaterial({ color, transparent: true, opacity: 0.35, depthWrite: false, side: THREE.DoubleSide }));
  ring.rotation.x = -Math.PI / 2; ring.position.y = 0.05;
  g.add(ring);
  const muzzle = new THREE.Mesh(sph(0.12, 8, 6), new THREE.MeshBasicMaterial({ color: 0xffc070 }));
  muzzle.position.set(0.04, 1.65, 1.4);
  muzzle.visible = false;
  g.add(muzzle);
  g.userData = { body: s, legs: s.userData.legs, arms: s.userData.upper, muzzle, ring, kind: 'hero' };
  return g;
}

export function makeDrone() {
  const b = new B();
  for (const a of [Math.PI / 4, -Math.PI / 4]) b.add('plain', box(0.9, 0.03, 0.06), 0x222222, 0, 0, 0, 0, a);
  b.add('plain', box(0.18, 0.08, 0.26), 0x2f3229, 0, 0.03, 0);
  b.add('gun', cyl(0.04, 0.04, 0.22, 8), 0x55523e, 0, -0.07, 0.05, Math.PI / 2);
  const g = b.build(false);
  const props = [];
  for (const [x, z] of [[0.32, 0.32], [-0.32, 0.32], [0.32, -0.32], [-0.32, -0.32]]) {
    const p = new THREE.Mesh(new THREE.CircleGeometry(0.16, 12), new THREE.MeshBasicMaterial({ color: 0x111111, transparent: true, opacity: 0.35, side: THREE.DoubleSide, depthWrite: false }));
    p.rotation.x = -Math.PI / 2; p.position.set(x, 0.05, z);
    g.add(p); props.push(p);
  }
  g.scale.setScalar(1.3);
  g.userData.props = props;
  return g;
}

// ───────── техніка ─────────
function wheels(b, positions, r, w) {
  for (const [x, z] of positions) {
    b.add('rubber', cyl(r, r, w, 16), 0x1c1c1a, x, r, z, 0, 0, Math.PI / 2);
    b.add('steel', cyl(r * 0.5, r * 0.5, w + 0.02, 10), 0x6a6a58, x, r, z, 0, 0, Math.PI / 2);
  }
}
function makeBTR(tint, scale = 1) {
  const b = new B();
  const hull = extrude([[-3.6, 0.55], [2.9, 0.55], [3.7, 1.1], [3.2, 1.85], [-3.3, 1.95], [-3.6, 1.7]], 2.6, 0.06);
  b.add('paint', hull, tint);
  const wp = [];
  for (const z of [-2.4, -0.9, 0.9, 2.2]) for (const x of [-1.3, 1.3]) wp.push([x, z]);
  wheels(b, wp, 0.55, 0.42);
  b.add('paint', box(0.06, 0.25, 6.2), tint * 1, 1.33, 1.1, -0.2);
  b.add('paint', box(0.06, 0.25, 6.2), tint, -1.33, 1.1, -0.2);
  for (const z of [-1.5, 0, 1.4]) for (const x of [-1, 1]) b.add('glass', box(0.04, 0.18, 0.18), 0, x * 1.31, 1.5, z);
  const turret = new B();
  turret.add('paint', cyl(0.75, 0.85, 0.5, 12), tint, 0, 0.25, 0);
  turret.add('steel', cyl(0.06, 0.06, 2.2, 8), 0x3a3a34, 0, 0.35, 1.2, Math.PI / 2);
  turret.add('steel', cyl(0.04, 0.04, 0.9, 6), 0x3a3a34, 0.25, 0.35, 0.7, Math.PI / 2);
  const g = b.build();
  const t = turret.build(); t.position.set(0, 1.95, 0.6);
  g.add(t);
  g.scale.setScalar(scale);
  g.userData.turret = t;
  return g;
}
function makeTank(tint, heavy = false) {
  const b = new B();
  b.add('paint', extrude([[-3.4, 0.55], [3.0, 0.55], [3.5, 1.0], [3.2, 1.5], [-3.4, 1.55]], 2.3, 0.05), tint);
  for (const x of [-1.45, 1.45]) {
    b.add('rubber', box(0.6, 0.75, 6.6), 0x2a2925, x, 0.42, 0);
    for (let i = 0; i < 6; i++) b.add('steel', cyl(0.36, 0.36, 0.62, 12), 0x4a4a40, x, 0.38, -2.5 + i * 1.0, 0, 0, Math.PI / 2);
    b.add('paint', box(0.65, 0.06, 6.8), tint, x, 0.85, 0);
  }
  // динамічний захист (ERA)
  for (let i = 0; i < 6; i++) b.add('paint', box(0.36, 0.14, 0.3), tint * 1, -1 + i * 0.4, 1.12, 3.15, -0.6);
  const tb = new B();
  tb.add('paint', sph(1.35, 16, 8).scale(1, 0.42, 1.15), tint, 0, 0.4, 0);
  tb.add('steel', cyl(0.11, 0.13, 5.2, 10), 0x3d3d34, 0, 0.55, 3.3, Math.PI / 2);
  tb.add('steel', cyl(0.17, 0.17, 0.7, 10), 0x3d3d34, 0, 0.55, 2.6, Math.PI / 2);
  for (let i = 0; i < 5; i++) for (const s of [-1, 1]) tb.add('paint', box(0.32, 0.22, 0.26), tint, s * (0.35 + i * 0.18), 0.55, 1.15 - i * 0.12, 0, s * 0.5);
  tb.add('steel', cyl(0.04, 0.04, 1.0, 6), 0x333333, 0.6, 1.0, 0.3, Math.PI / 2);
  tb.add('paint', cyl(0.28, 0.3, 0.25, 10), tint, 0.55, 0.85, -0.2);
  if (heavy) for (let i = 0; i < 8; i++) tb.add('paint', box(0.4, 0.25, 0.3), tint, -1.2 + i * 0.35, 0.75, -1.2, 0.3);
  const g = b.build();
  const t = tb.build(); t.position.set(0, 1.55, -0.3);
  g.add(t);
  if (heavy) g.scale.setScalar(1.3);
  g.userData.turret = t;
  return g;
}
function makeTruck(tint, kind) {
  const b = new B();
  const L = kind === 'transport' ? 8 : 6.5;
  b.add('paint', extrude([[L / 2 - 2.2, 0.9], [L / 2, 0.9], [L / 2, 2.1], [L / 2 - 0.7, 2.9], [L / 2 - 2.2, 2.9]], 2.4, 0.05), tint);
  b.add('glass', box(2.2, 0.55, 0.05), 0, 0, 2.45, L / 2 - 0.42, -0.6);
  b.add('steel', box(2.3, 0.3, L - 0.5), 0x3a3a32, 0, 0.95, -0.2);
  const wp = kind === 'transport' ? [[-1.1, L / 2 - 1.2], [1.1, L / 2 - 1.2], [-1.1, -0.8], [1.1, -0.8], [-1.1, -2.3], [1.1, -2.3], [-1.1, -3.4], [1.1, -3.4]] : [[-1.1, L / 2 - 1.2], [1.1, L / 2 - 1.2], [-1.1, -1.2], [1.1, -1.2], [-1.1, -2.4], [1.1, -2.4]];
  wheels(b, wp, 0.6, 0.4);
  if (kind === 'transport') {
    b.add('canvas', box(2.5, 1.8, L - 2.6), 0x5a5d44, 0, 2.1, -1.3);
    b.add('canvas', new THREE.CylinderGeometry(1.25, 1.25, L - 2.6, 12, 1, false, -Math.PI / 2, Math.PI), 0x5a5d44, 0, 3.0, -1.3, Math.PI / 2, 0, 0, 1, 1, 0.5);
  } else {
    b.add('paint', box(2.3, 0.7, L - 2.5), tint, 0, 1.45, -1.2);
    b.add('steel', box(0.25, 0.25, 3.4), 0x8a7a3a, 0, 2.6, -1.0, -0.35);
    b.add('steel', box(0.15, 1.2, 0.15), 0x8a7a3a, 0, 2.1, -2.4);
  }
  const g = b.build();
  return g;
}
function makeMoto(tint) {
  const b = new B();
  for (const z of [-0.65, 0.65]) {
    b.add('rubber', new THREE.TorusGeometry(0.32, 0.09, 8, 16), 0x161616, 0, 0.36, z, 0, Math.PI / 2);
  }
  b.add('paint', box(0.22, 0.32, 1.1), tint, 0, 0.72, 0);
  b.add('steel', box(0.28, 0.12, 0.4), 0x333333, 0, 0.55, 0.1);
  b.add('steel', cyl(0.025, 0.025, 0.8, 6), 0x222222, 0, 0.95, 0.55, 0, 0, Math.PI / 2);
  const g = b.build();
  const rider = makeSoldier({ side: 'enemy', color: 0xb03030, crew: true });
  rider.position.set(0, -0.12, -0.1);
  rider.userData.legs.forEach((l, i) => { l.rotation.x = -1.1; l.rotation.z = (i ? -1 : 1) * 0.25; });
  g.add(rider);
  return g;
}
function makeFPV() {
  const b = new B();
  for (const a of [Math.PI / 4, -Math.PI / 4]) b.add('plain', box(0.55, 0.025, 0.05), 0x1e1e1e, 0, 0, 0, 0, a);
  b.add('plain', box(0.1, 0.06, 0.18), 0x2a2a2a, 0, 0.03, 0);
  b.add('gun', cyl(0.05, 0.035, 0.32, 8), 0x4c4f3e, 0, -0.06, 0.12, Math.PI / 2);
  b.add('gun', new THREE.ConeGeometry(0.05, 0.12, 8), 0x4c4f3e, 0, -0.06, 0.34, Math.PI / 2);
  const g = b.build(false);
  const props = [];
  for (const [x, z] of [[0.2, 0.2], [-0.2, 0.2], [0.2, -0.2], [-0.2, -0.2]]) {
    const p = new THREE.Mesh(new THREE.CircleGeometry(0.11, 10), new THREE.MeshBasicMaterial({ color: 0x111111, transparent: true, opacity: 0.35, side: THREE.DoubleSide, depthWrite: false }));
    p.rotation.x = -Math.PI / 2; p.position.set(x, 0.04, z);
    g.add(p); props.push(p);
  }
  g.scale.setScalar(1.8);
  g.userData.props = props;
  return g;
}
function makeShahed() {
  const b = new B();
  b.add('paint', cyl(0.18, 0.2, 3.2, 12), 0xc4c4bc, 0, 0, 0, Math.PI / 2);
  b.add('paint', new THREE.ConeGeometry(0.18, 0.5, 12), 0xc4c4bc, 0, 0, 1.85, Math.PI / 2);
  const wing = new THREE.ExtrudeGeometry(new THREE.Shape([new THREE.Vector2(0, -1.0), new THREE.Vector2(1.25, 1.4), new THREE.Vector2(-1.25, 1.4)]), { depth: 0.06, bevelEnabled: false });
  wing.rotateX(-Math.PI / 2);
  b.add('paint', wing, 0xb8b8b0, 0, 0, -0.4);
  for (const s of [-1, 1]) b.add('paint', box(0.04, 0.5, 0.45), 0xb0b0a8, s * 1.25, 0.2, -1.5);
  const g = b.build(false);
  const prop = new THREE.Mesh(new THREE.CircleGeometry(0.45, 14), new THREE.MeshBasicMaterial({ color: 0x111111, transparent: true, opacity: 0.3, side: THREE.DoubleSide, depthWrite: false }));
  prop.position.z = -1.68;
  g.add(prop);
  g.userData.props = [];
  return g;
}
function makeHeli(tint) {
  const b = new B();
  b.add('paint', extrude([[-2.5, -0.9], [2.2, -0.9], [3.6, -0.3], [3.4, 0.5], [1.5, 1.2], [-2.6, 1.0]], 2.0, 0.15), tint, 0, 0, 0);
  b.add('glass', sph(0.9, 12, 8).scale(1, 0.75, 1.5), 0, 0, 0.6, 2.2);
  b.add('paint', cyl(0.25, 0.55, 6.5, 10), tint, 0, 0.4, -5.5, Math.PI / 2);
  b.add('paint', box(0.12, 1.8, 1.2), tint, 0, 1.2, -8.4, -0.3);
  b.add('paint', box(2.6, 0.08, 0.7), tint, 0, 0.6, -8.2);
  for (const s of [-1, 1]) {
    b.add('paint', box(1.6, 0.12, 0.8), tint, s * 1.6, 0, 0);
    b.add('steel', cyl(0.25, 0.25, 1.4, 10), 0x2c2c28, s * 2.2, -0.3, 0.1, Math.PI / 2);
    b.add('steel', box(0.08, 0.08, 3), 0x222222, s * 0.7, -1.4, 0.3);
  }
  b.add('steel', cyl(0.25, 0.3, 0.5, 10), 0x333333, 0, 1.4, 0.2);
  const g = b.build();
  const rotor = new THREE.Group();
  const rb = new B();
  for (let i = 0; i < 5; i++) rb.add('plain', box(0.3, 0.04, 7.2), 0x1c1c1a, 0, 0, 3.6, 0, (i / 5) * Math.PI * 2);
  rotor.add(rb.build(false));
  rotor.children[0].children.forEach((m) => { m.geometry.translate(0, 0, 0); });
  rotor.position.set(0, 1.7, 0.2);
  g.add(rotor);
  const tail = new THREE.Mesh(new THREE.CircleGeometry(0.8, 12), new THREE.MeshBasicMaterial({ color: 0x111111, transparent: true, opacity: 0.35, side: THREE.DoubleSide, depthWrite: false }));
  tail.rotation.y = Math.PI / 2; tail.position.set(0.15, 1.3, -8.4);
  g.add(tail);
  g.userData.rotor = rotor;
  g.userData.props = [];
  return g;
}

const ENEMY_TINT = 0xb9b7a6;
export function makeEnemy(kind) {
  const g = new THREE.Group();
  let body;
  let legs = [], props = [], turret = null, rotor = null;
  switch (kind) {
    case 'infantry': body = makeSoldier({ side: 'enemy', color: 0xb03030 }); body.scale.setScalar(1.15); break;
    case 'w_rifle': body = makeSoldier({ side: 'drg', color: 0xb03030 }); body.scale.setScalar(1.15); break;
    case 'w_mg': body = makeSoldier({ side: 'drg', color: 0xb03030, kind: 'mg' }); body.scale.setScalar(1.15); break;
    case 'w_officer': body = makeSoldier({ side: 'drg', color: 0xd8b040, backpack: true }); body.scale.setScalar(1.25); break;
    case 'moto': body = makeMoto(ENEMY_TINT); body.scale.setScalar(1.2); break;
    case 'fpv': body = makeFPV(); break;
    case 'shahed': body = makeShahed(); break;
    case 'btr': body = makeBTR(ENEMY_TINT); break;
    case 'tank': body = makeTank(ENEMY_TINT); break;
    case 'repair': body = makeTruck(ENEMY_TINT, 'repair'); break;
    case 'boss_heavy': body = makeTank(0x9a9a8a, true); break;
    case 'boss_transport': body = makeTruck(ENEMY_TINT, 'transport'); body.scale.setScalar(1.15); break;
    case 'boss_heli': body = makeHeli(0x8f9280); break;
  }
  g.add(body);
  legs = body.userData.legs || [];
  props = body.userData.props || [];
  turret = body.userData.turret || null;
  rotor = body.userData.rotor || null;
  const flying = ['fpv', 'shahed', 'boss_heli'].includes(kind);
  g.userData = { body, legs, props, turret, rotor, kind, flying, boss: kind.startsWith('boss'), human: !!body.userData.upper, vehicle: !!turret || ['moto', 'repair', 'boss_transport'].includes(kind) };
  return g;
}

// ───────── вогневі позиції ─────────
function bagRing(b, r, layers, gap = 0) {
  const a = A.glb.cement_bag;
  if (!a) return;
  for (let l = 0; l < layers; l++) {
    const n = Math.floor((2 * Math.PI * r) / 0.7);
    for (let i = 0; i < n; i++) {
      const ang = ((i + (l % 2) * 0.5) / n) * Math.PI * 2;
      if (gap && Math.abs(Math.sin((ang - Math.PI) / 2)) < gap) continue;
      b.push({ x: Math.cos(ang) * r, z: Math.sin(ang) * r, y: l * 0.17, ry: -ang });
    }
  }
}
function bagMesh(list) {
  const a = A.glb.cement_bag;
  if (!a || !list.length) return new THREE.Group();
  const im = new THREE.InstancedMesh(a.geo, patchWeather(a.mat), list.length);
  const m = new THREE.Matrix4();
  list.forEach((p, i) => { m.compose(new THREE.Vector3(p.x, p.y, p.z), new THREE.Quaternion().setFromEuler(new THREE.Euler(0, p.ry, 0)), new THREE.Vector3(1.6, 1.25, 1.05)); im.setMatrixAt(i, m); });
  im.castShadow = im.receiveShadow = true;
  return im;
}
function propMesh(id, x, y, z, ry = 0, s = 1) {
  const a = A.glb[id];
  if (!a) return new THREE.Group();
  const me = new THREE.Mesh(a.geo, patchWeather(a.mat));
  me.position.set(x, y, z); me.rotation.y = ry; me.scale.setScalar(s);
  me.castShadow = me.receiveShadow = true;
  return me;
}
function tripod(b, h = 0.6) {
  for (let i = 0; i < 3; i++) {
    const a = (i / 3) * Math.PI * 2 + Math.PI / 2;
    b.limb('gun', [0, h, 0], [Math.cos(a) * 0.55, 0, Math.sin(a) * 0.55 - 0.1], 0.02, 0x2a2b26);
  }
}

export function makeTower(type) {
  const g = new THREE.Group();
  const head = new THREE.Group();
  const hb = new B();
  const muzzles = [];
  const bags = [];
  const base = new THREE.Group();
  const OL = 0xd8d4c0;
  let crewPos = [0, -0.75], crewPose = 'kneel';
  switch (type) {
    case 'mg': {
      bagRing(bags, 1.7, 2, 0.3);
      tripod(hb, 0.55);
      hb.add('gun', box(0.11, 0.14, 0.7), 0x232421, 0, 0.62, 0.15);
      hb.add('gun', cyl(0.022, 0.022, 0.75, 8), 0x1d1d1b, 0, 0.64, 0.85, Math.PI / 2);
      hb.add('gun', cyl(0.035, 0.03, 0.12, 8), 0x1d1d1b, 0, 0.64, 1.25, Math.PI / 2);
      hb.add('paint', box(0.55, 0.38, 0.03), OL, 0, 0.72, 0.42);
      muzzles.push(new THREE.Vector3(0, 0.64, 1.3));
      base.add(propMesh('ammo_box', 0.6, 0, -0.3, 0.5, 1.8));
      break;
    }
    case 'ags': {
      bagRing(bags, 1.7, 2, 0.3);
      tripod(hb, 0.45);
      hb.add('gun', box(0.18, 0.18, 0.5), 0x2a2c26, 0, 0.55, 0.1);
      hb.add('gun', cyl(0.03, 0.03, 0.5, 8), 0x1d1d1b, 0, 0.58, 0.55, Math.PI / 2);
      hb.add('gun', cyl(0.16, 0.16, 0.12, 14), 0x3d4032, 0.18, 0.5, 0.05, 0, 0, Math.PI / 2);
      muzzles.push(new THREE.Vector3(0, 0.58, 0.82));
      base.add(propMesh('old_military_crate', -0.4, 0, -1.0, 0.3, 0.8));
      break;
    }
    case 'aa': {
      bagRing(bags, 2.3, 2, 0.25);
      hb.add('paint', box(1.4, 0.3, 1.6), OL, 0, 0.55, 0);
      hb.add('paint', box(0.9, 0.7, 0.6), OL, 0, 0.95, 0.1);
      for (const s of [-1, 1]) {
        hb.add('steel', cyl(0.05, 0.05, 2.6, 8), 0x2d2e28, s * 0.22, 1.25, 1.4, Math.PI / 2 - 0.12);
        hb.add('steel', cyl(0.08, 0.08, 0.3, 8), 0x2d2e28, s * 0.22, 1.42, 2.7, Math.PI / 2 - 0.12);
        hb.add('rubber', cyl(0.38, 0.38, 0.25, 14), 0x1a1a18, s * 0.85, 0.38, -0.3, 0, 0, Math.PI / 2);
        hb.add('paint', box(0.35, 0.5, 0.35), OL, s * 0.55, 1.1, -0.4);
        muzzles.push(new THREE.Vector3(s * 0.22, 1.43, 2.75));
      }
      crewPos = [0.9, -1.2];
      break;
    }
    case 'atgm': {
      bagRing(bags, 1.7, 2, 0.3);
      tripod(hb, 0.7);
      hb.add('paint', cyl(0.09, 0.09, 1.3, 12), OL, 0, 0.85, 0.2, Math.PI / 2);
      hb.add('gun', box(0.22, 0.2, 0.3), 0x2a2b26, 0.18, 0.72, -0.1);
      hb.add('gun', cyl(0.05, 0.05, 0.16, 8), 0x111111, 0.18, 0.78, 0.08, Math.PI / 2);
      muzzles.push(new THREE.Vector3(0, 0.85, 0.9));
      base.add(propMesh('wooden_military_crate', 0.5, 0, -1.0, 0.4, 0.8));
      break;
    }
    case 'mortar': {
      bagRing(bags, 1.9, 3, 0);
      hb.add('steel', cyl(0.4, 0.45, 0.06, 16), 0x343530, 0, 0.05, -0.2);
      hb.limb('steel', [0, 0.08, -0.2], [0, 1.25, 0.55], 0.065, 0x2f302b);
      hb.limb('gun', [0, 0.7, 0.3], [-0.3, 0, 0.6], 0.018, 0x2f302b);
      hb.limb('gun', [0, 0.7, 0.3], [0.3, 0, 0.6], 0.018, 0x2f302b);
      muzzles.push(new THREE.Vector3(0, 1.28, 0.57));
      base.add(propMesh('ammo_box', -0.9, 0, 0.6, 0.3, 2));
      base.add(propMesh('ammo_box', -1.0, 0, 0.9, 0.6, 2));
      crewPos = [0.7, -0.4];
      break;
    }
    case 'howitzer': {
      bagRing(bags, 3.0, 2, 0.35);
      for (const s of [-1, 1]) {
        hb.limb('paint', [s * 0.3, 0.45, -0.2], [s * 1.0, 0.05, -3.2], 0.09, OL);
        hb.add('rubber', cyl(0.62, 0.62, 0.3, 18), 0x1b1b19, s * 1.0, 0.62, 0.2, 0, 0, Math.PI / 2);
        hb.add('steel', cyl(0.3, 0.3, 0.32, 12), 0x5a5a4a, s * 1.0, 0.62, 0.2, 0, 0, Math.PI / 2);
      }
      hb.add('paint', box(1.9, 1.0, 0.06), OL, 0, 1.15, 0.5, 0.1);
      hb.add('paint', box(0.6, 0.5, 1.4), OL, 0, 0.95, 0);
      hb.add('steel', cyl(0.1, 0.13, 4.6, 12), 0x3a3b33, 0, 1.25, 2.4, Math.PI / 2 - 0.08);
      hb.add('steel', box(0.36, 0.22, 0.4), 0x3a3b33, 0, 1.43, 4.65, -0.08);
      muzzles.push(new THREE.Vector3(0, 1.45, 4.9));
      base.add(propMesh('wooden_military_crate', 2.0, 0, -1.6, 0.8));
      crewPos = [1.6, -0.6];
      break;
    }
  }
  base.add(bagMesh(bags));
  const crew = makeSoldier({ side: 'ally', color: 0x3060c0, crew: true, pose: crewPose });
  crew.position.set(crewPos[0], 0, crewPos[1]);
  crew.scale.setScalar(1.1);
  const hm = hb.build();
  head.add(hm);
  head.add(crew);
  g.add(base, head);
  // покращення за рівнем
  const upgrades = new THREE.Group();
  g.add(upgrades);
  const auraRing = new THREE.Mesh(new THREE.RingGeometry(2.6, 2.75, 40), new THREE.MeshBasicMaterial({ color: 0xffd27a, transparent: true, opacity: 0.25, depthWrite: false, side: THREE.DoubleSide }));
  auraRing.rotation.x = -Math.PI / 2; auraRing.position.y = 0.06; auraRing.visible = false;
  g.add(auraRing);
  g.userData = { head, muzzles, type, auraRing, upgrades, level: 0, ringR: { mg: 1.7, ags: 1.7, aa: 2.3, atgm: 1.7, mortar: 1.9, howitzer: 3 }[type] };
  return g;
}
let netMat = null;
export function setTowerLevel(g, level) {
  const u = g.userData;
  if (u.level === level) return;
  u.level = level;
  u.upgrades.clear();
  if (level >= 4) {
    const b = [];
    for (let i = 0; i < Math.floor(2 * Math.PI * (u.ringR + 0.55) / 0.7); i++) {
      const a = (i / Math.floor(2 * Math.PI * (u.ringR + 0.55) / 0.7)) * Math.PI * 2;
      b.push({ x: Math.cos(a) * (u.ringR + 0.55), z: Math.sin(a) * (u.ringR + 0.55), y: 0, ry: -a });
      if (level >= 12) b.push({ x: Math.cos(a) * (u.ringR + 0.55), z: Math.sin(a) * (u.ringR + 0.55), y: 0.17, ry: -a + 0.3 });
    }
    u.upgrades.add(bagMesh(b));
  }
  if (level >= 8) {
    if (!netMat) {
      const c = document.createElement('canvas'); c.width = c.height = 128;
      const x = c.getContext('2d');
      for (let i = 0; i < 300; i++) { x.fillStyle = ['#4c5a33', '#5d6b3c', '#3d4a2b', '#6b6a44'][i % 4]; x.beginPath(); x.ellipse(Math.random() * 128, Math.random() * 128, 3 + Math.random() * 5, 2 + Math.random() * 3, Math.random() * 3, 0, 7); x.fill(); }
      const t = new THREE.CanvasTexture(c); t.colorSpace = THREE.SRGBColorSpace; t.wrapS = t.wrapT = THREE.RepeatWrapping; t.repeat.set(2, 2);
      netMat = patchWeather(new THREE.MeshStandardMaterial({ map: t, alphaTest: 0.3, side: THREE.DoubleSide, roughness: 1 }), true);
    }
    const r = u.ringR + 1;
    const geo = new THREE.PlaneGeometry(r * 2, r * 2, 6, 6);
    const p = geo.attributes.position;
    for (let i = 0; i < p.count; i++) p.setZ(i, -Math.max(0, 0.9 - Math.hypot(p.getX(i), p.getY(i)) / r) * 1.2);
    geo.computeVertexNormals();
    const net = new THREE.Mesh(geo, netMat);
    net.rotation.x = -Math.PI / 2; net.position.y = 2.6;
    net.castShadow = true;
    u.upgrades.add(net);
    for (const [sx, sz] of [[1, 1], [-1, 1], [1, -1], [-1, -1]]) {
      const pole = new THREE.Mesh(cyl(0.04, 0.05, 2.6, 5), mats().steel);
      pole.position.set(sx * r * 0.7, 1.3, sz * r * 0.7);
      u.upgrades.add(pole);
    }
  }
  if (level >= 16) {
    for (let i = 0; i < 3; i++) u.upgrades.add(propMesh('concrete_road_barrier_02', Math.cos(i * 2.1 + 0.5) * (u.ringR + 1.6), 0, Math.sin(i * 2.1 + 0.5) * (u.ringR + 1.6), -i * 2.1, 1.2));
  }
}

// ───────── трофеї ─────────
const RCOL = [0xc8d0d8, 0x4dff7a, 0x3aa6ff, 0xc057ff, 0xffa02a];
export function makeLoot(rarity) {
  const g = new THREE.Group();
  const crate = propMesh(rarity >= 2 ? 'wooden_military_crate' : 'ammo_box', 0, 0, 0, Math.random() * 6, rarity >= 2 ? 0.55 : 2.2);
  g.add(crate);
  const col = RCOL[rarity];
  const beam = new THREE.Mesh(new THREE.CylinderGeometry(0.08, 0.25, 3 + rarity * 2, 8, 1, true), new THREE.MeshBasicMaterial({ color: col, transparent: true, opacity: 0.22 + rarity * 0.04, blending: THREE.AdditiveBlending, depthWrite: false }));
  beam.position.y = (3 + rarity * 2) / 2;
  g.add(beam);
  const ring = new THREE.Mesh(new THREE.RingGeometry(0.7, 0.85, 24), new THREE.MeshBasicMaterial({ color: col, transparent: true, opacity: 0.5, depthWrite: false, side: THREE.DoubleSide }));
  ring.rotation.x = -Math.PI / 2; ring.position.y = 0.05;
  g.add(ring);
  g.userData = { gem: beam };
  return g;
}

export function makeGhost(type) {
  const g = makeTower(type);
  const gm = new THREE.MeshBasicMaterial({ color: 0xffffff, transparent: true, opacity: 0.35, depthWrite: false });
  g.traverse((o) => { if (o.isMesh) { o.material = gm; o.castShadow = false; } });
  const range = new THREE.Mesh(new THREE.RingGeometry(0.985, 1, 96), new THREE.MeshBasicMaterial({ color: 0xffffff, transparent: true, opacity: 0.6, depthWrite: false, side: THREE.DoubleSide }));
  range.rotation.x = -Math.PI / 2; range.position.y = 0.15;
  const fill = new THREE.Mesh(new THREE.CircleGeometry(1, 96), new THREE.MeshBasicMaterial({ color: 0xffffff, transparent: true, opacity: 0.05, depthWrite: false }));
  fill.rotation.x = -Math.PI / 2; fill.position.y = 0.14;
  g.add(range, fill);
  g.userData.range = range; g.userData.fill = fill; g.userData.ghostMat = gm;
  return g;
}
