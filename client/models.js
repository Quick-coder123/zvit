// Процедурні 3D-моделі: герої, вежі, вороги, боси.
import * as THREE from 'three';
import { RoundedBoxGeometry } from 'three/addons/geometries/RoundedBoxGeometry.js';
import { TOWERS, ENEMIES } from '/shared/data.js';

const matCache = new Map();
function metal(color = 0x3a4456, rough = 0.3, metalness = 0.85) {
  const k = `m${color}_${rough}_${metalness}`;
  if (!matCache.has(k)) matCache.set(k, new THREE.MeshStandardMaterial({ color, roughness: rough, metalness }));
  return matCache.get(k);
}
function glow(color, intensity = 2.5) {
  const k = `g${color}_${intensity}`;
  if (!matCache.has(k)) matCache.set(k, new THREE.MeshStandardMaterial({ color: 0x000000, emissive: color, emissiveIntensity: intensity }));
  return matCache.get(k);
}
function skin(color, emissiveK = 0.15) {
  const k = `s${color}_${emissiveK}`;
  if (!matCache.has(k)) {
    const c = new THREE.Color(color);
    matCache.set(k, new THREE.MeshStandardMaterial({ color: c.clone().multiplyScalar(0.35), roughness: 0.55, metalness: 0.25, emissive: c, emissiveIntensity: emissiveK }));
  }
  return matCache.get(k);
}
function mesh(geo, mat, x = 0, y = 0, z = 0, shadow = true) {
  const m = new THREE.Mesh(geo, mat);
  m.position.set(x, y, z);
  m.castShadow = shadow;
  return m;
}
const G = {
  box: (w, h, d) => new THREE.BoxGeometry(w, h, d),
  rbox: (w, h, d, r = 0.08) => new RoundedBoxGeometry(w, h, d, 2, r),
  cyl: (a, b, h, s = 12) => new THREE.CylinderGeometry(a, b, h, s),
  sph: (r, w = 16, h = 12) => new THREE.SphereGeometry(r, w, h),
};

// ───────────────────────── Герой ─────────────────────────
export function makeHero(color) {
  const g = new THREE.Group();
  const body = new THREE.Group();
  g.add(body);
  const armor = metal(0x2c3546, 0.35, 0.8);
  const dark = metal(0x161b24, 0.5, 0.6);
  const accent = glow(color, 2.6);
  const legs = [];
  for (const s of [-1, 1]) {
    const hip = new THREE.Group();
    hip.position.set(s * 0.32, 1.05, 0);
    const thigh = mesh(G.rbox(0.32, 0.6, 0.36), armor, 0, -0.3, 0);
    const shin = mesh(G.rbox(0.28, 0.55, 0.32), dark, 0, -0.78, 0.02);
    const foot = mesh(G.rbox(0.32, 0.16, 0.5), armor, 0, -1.0, 0.08);
    const kneeGlow = mesh(G.box(0.1, 0.1, 0.05), accent, 0, -0.55, 0.19, false);
    hip.add(thigh, shin, foot, kneeGlow);
    body.add(hip);
    legs.push(hip);
  }
  const torso = mesh(G.rbox(0.95, 0.85, 0.6, 0.12), armor, 0, 1.55, 0);
  const chestGlow = mesh(G.box(0.35, 0.12, 0.05), accent, 0, 1.68, 0.31, false);
  const belt = mesh(G.rbox(0.85, 0.18, 0.52), dark, 0, 1.12, 0);
  const pack = mesh(G.rbox(0.7, 0.75, 0.32), dark, 0, 1.6, -0.42);
  const packGlow = mesh(G.cyl(0.1, 0.1, 0.6, 8), accent, 0, 1.6, -0.6, false);
  packGlow.rotation.z = Math.PI / 2;
  const head = mesh(G.rbox(0.48, 0.44, 0.48, 0.14), armor, 0, 2.2, 0.02);
  const visor = mesh(G.rbox(0.42, 0.14, 0.1, 0.04), accent, 0, 2.23, 0.24, false);
  const antenna = mesh(G.cyl(0.02, 0.02, 0.5, 4), dark, 0.18, 2.55, -0.1, false);
  body.add(torso, chestGlow, belt, pack, packGlow, head, visor, antenna);
  for (const s of [-1, 1]) {
    const pad = mesh(G.rbox(0.38, 0.22, 0.5, 0.08), armor, s * 0.62, 1.95, 0);
    pad.rotation.z = s * -0.25;
    const padG = mesh(G.box(0.3, 0.04, 0.42), accent, s * 0.64, 2.07, 0, false);
    padG.rotation.z = s * -0.25;
    body.add(pad, padG);
  }
  // руки + гвинтівка
  const arms = new THREE.Group();
  arms.position.set(0, 1.6, 0);
  const armL = mesh(G.rbox(0.22, 0.22, 0.75), dark, -0.45, -0.05, 0.35);
  const armR = mesh(G.rbox(0.22, 0.22, 0.75), dark, 0.42, -0.05, 0.35);
  const gun = new THREE.Group();
  gun.position.set(0.15, 0, 0.75);
  gun.add(mesh(G.rbox(0.18, 0.26, 1.1, 0.05), metal(0x222833, 0.3, 0.9), 0, 0, 0.2));
  gun.add(mesh(G.cyl(0.06, 0.06, 0.6, 8).rotateX(Math.PI / 2), dark, 0, 0.03, 0.95));
  gun.add(mesh(G.box(0.05, 0.08, 0.7), accent, 0.1, 0.05, 0.2, false));
  gun.add(mesh(G.box(0.05, 0.08, 0.7), accent, -0.1, 0.05, 0.2, false));
  const muzzle = mesh(G.sph(0.18, 8, 6), glow(color, 6), 0, 0.03, 1.3, false);
  muzzle.visible = false;
  gun.add(muzzle);
  arms.add(armL, armR, gun);
  body.add(arms);
  // кільце під героєм
  const ring = new THREE.Mesh(new THREE.RingGeometry(0.9, 1.1, 32), new THREE.MeshBasicMaterial({ color, transparent: true, opacity: 0.6, depthWrite: false, side: THREE.DoubleSide }));
  ring.rotation.x = -Math.PI / 2;
  ring.position.y = 0.06;
  g.add(ring);
  const light = new THREE.PointLight(color, 6, 8, 2);
  light.position.set(0, 2.5, 0.5);
  g.add(light);
  body.scale.setScalar(1.3);
  g.userData = { body, legs, arms, muzzle, ring, kind: 'hero' };
  return g;
}

export function makeDrone(color) {
  const g = new THREE.Group();
  g.add(mesh(G.sph(0.35, 12, 8), metal(0x2c3546), 0, 0, 0));
  g.add(mesh(G.box(1.4, 0.06, 0.12), metal(0x222833), 0, 0.05, 0));
  g.add(mesh(G.box(0.12, 0.06, 1.4), metal(0x222833), 0, 0.05, 0));
  g.add(mesh(G.sph(0.14, 8, 6), glow(color, 4), 0, -0.05, 0.3, false));
  for (const [x, z] of [[0.7, 0], [-0.7, 0], [0, 0.7], [0, -0.7]]) g.add(mesh(new THREE.TorusGeometry(0.22, 0.03, 4, 16).rotateX(Math.PI / 2), glow(color, 2), x, 0.08, z, false));
  return g;
}

// ───────────────────────── Вежі ─────────────────────────
export function makeTower(type) {
  const def = TOWERS[type];
  const c = def.color;
  const g = new THREE.Group();
  const base = new THREE.Group();
  const plate = metal(0x2a3240, 0.35, 0.85);
  const dark = metal(0x151a22, 0.5, 0.7);
  base.add(mesh(G.cyl(1.7, 1.95, 0.5, 6), plate, 0, 0.25, 0));
  base.add(mesh(G.cyl(1.35, 1.6, 0.9, 6), dark, 0, 0.95, 0));
  base.add(mesh(new THREE.TorusGeometry(1.45, 0.06, 6, 6).rotateX(Math.PI / 2).rotateY(Math.PI / 6), glow(c, 2.5), 0, 0.52, 0, false));
  base.add(mesh(G.cyl(0.8, 1.0, 1.0, 12), plate, 0, 1.85, 0));
  g.add(base);
  const head = new THREE.Group();
  head.position.y = 2.6;
  g.add(head);
  const accent = glow(c, 3);
  const muzzles = [];
  switch (type) {
    case 'blaster': {
      head.add(mesh(G.rbox(1.3, 0.8, 1.4, 0.15), plate, 0, 0.3, 0));
      head.add(mesh(G.box(0.9, 0.12, 0.08), accent, 0, 0.45, 0.71, false));
      for (const s of [-1, 1]) {
        head.add(mesh(G.cyl(0.11, 0.13, 1.6, 8).rotateX(Math.PI / 2), dark, s * 0.32, 0.3, 1.2));
        head.add(mesh(G.cyl(0.14, 0.14, 0.12, 8).rotateX(Math.PI / 2), accent, s * 0.32, 0.3, 1.95, false));
        muzzles.push(new THREE.Vector3(s * 0.32, 0.3, 2.05));
      }
      break;
    }
    case 'cryo': {
      head.add(mesh(G.sph(0.75, 20, 14), plate, 0, 0.4, 0));
      const tank = mesh(G.sph(0.55, 16, 12), new THREE.MeshStandardMaterial({ color: 0x99ccff, emissive: c, emissiveIntensity: 1.2, transparent: true, opacity: 0.85, roughness: 0.05 }), 0, 0.85, -0.2);
      head.add(tank);
      head.add(mesh(G.cyl(0.18, 0.45, 1.1, 12).rotateX(-Math.PI / 2), dark, 0, 0.4, 0.9));
      head.add(mesh(new THREE.TorusGeometry(0.4, 0.06, 6, 18), accent, 0, 0.4, 1.45, false));
      muzzles.push(new THREE.Vector3(0, 0.4, 1.5));
      break;
    }
    case 'tesla': {
      head.add(mesh(G.cyl(0.3, 0.45, 2.2, 10), dark, 0, 1.1, 0));
      for (let i = 0; i < 4; i++) head.add(mesh(new THREE.TorusGeometry(0.62 - i * 0.08, 0.1, 8, 24).rotateX(Math.PI / 2), i % 2 ? plate : accent, 0, 0.4 + i * 0.5, 0, false));
      head.add(mesh(G.sph(0.5, 16, 12), glow(c, 4), 0, 2.5, 0, false));
      muzzles.push(new THREE.Vector3(0, 2.5, 0));
      break;
    }
    case 'rail': {
      head.add(mesh(G.rbox(1.0, 0.8, 1.6, 0.12), plate, 0, 0.3, -0.2));
      for (const s of [-1, 1]) head.add(mesh(G.box(0.14, 0.32, 3.2), dark, s * 0.24, 0.35, 1.6));
      for (let i = 0; i < 5; i++) head.add(mesh(G.box(0.62, 0.08, 0.1), accent, 0, 0.35, 0.5 + i * 0.6, false));
      head.add(mesh(G.cyl(0.08, 0.08, 3.2, 6).rotateX(Math.PI / 2), glow(c, 1.5), 0, 0.35, 1.6, false));
      muzzles.push(new THREE.Vector3(0, 0.35, 3.2));
      break;
    }
    case 'mortar': {
      head.add(mesh(G.rbox(1.6, 0.6, 1.6, 0.15), plate, 0, 0.2, 0));
      const tube = mesh(G.cyl(0.55, 0.65, 1.8, 14), dark, 0, 1.0, 0.3);
      tube.rotation.x = 0.55;
      head.add(tube);
      const rim = mesh(new THREE.TorusGeometry(0.56, 0.08, 6, 20), accent, 0, 1.75, 0.8, false);
      rim.rotation.x = Math.PI / 2 + 0.55;
      head.add(rim);
      for (const s of [-1, 1]) head.add(mesh(G.cyl(0.25, 0.25, 0.9, 10), glow(c, 1.5), s * 0.85, 0.35, -0.3));
      muzzles.push(new THREE.Vector3(0, 1.8, 0.8));
      break;
    }
    case 'laser': {
      head.add(mesh(G.rbox(1.1, 0.9, 1.1, 0.15), plate, 0, 0.35, 0));
      const dish = mesh(G.cyl(0.95, 0.3, 0.4, 20).rotateX(Math.PI / 2), dark, 0, 0.7, 0.75);
      head.add(dish);
      head.add(mesh(new THREE.OctahedronGeometry(0.32), glow(c, 5), 0, 0.7, 1.15, false));
      head.add(mesh(new THREE.TorusGeometry(0.9, 0.05, 6, 30), accent, 0, 0.7, 0.96, false));
      muzzles.push(new THREE.Vector3(0, 0.7, 1.2));
      break;
    }
  }
  head.traverse((o) => { if (o.isMesh) o.castShadow = true; });
  base.traverse((o) => { if (o.isMesh) { o.castShadow = true; o.receiveShadow = true; } });
  // індикатори рівня
  const pips = new THREE.Group();
  g.add(pips);
  const auraRing = new THREE.Mesh(new THREE.RingGeometry(2.0, 2.25, 6), new THREE.MeshBasicMaterial({ color: 0x3ae0ff, transparent: true, opacity: 0.7, depthWrite: false, side: THREE.DoubleSide }));
  auraRing.rotation.x = -Math.PI / 2;
  auraRing.position.y = 0.08;
  auraRing.visible = false;
  g.add(auraRing);
  g.userData = { head, muzzles, pips, type, auraRing, recoil: 0 };
  return g;
}
const pipGeo = new THREE.OctahedronGeometry(0.16);
export function setTowerLevel(g, level) {
  const { pips } = g.userData;
  pips.clear();
  const tiers = [0xffffff, 0x4dff7a, 0x3aa6ff, 0xc057ff, 0xffa02a];
  const tier = Math.min(4, Math.floor((level - 1) / 5));
  const n = ((level - 1) % 5) + 1;
  for (let i = 0; i < n; i++) {
    const a = (i / 5) * Math.PI * 2;
    const p = new THREE.Mesh(pipGeo, glow(tiers[tier], 3));
    p.position.set(Math.cos(a) * 1.25, 1.55, Math.sin(a) * 1.25);
    pips.add(p);
  }
  g.scale.setScalar(1 + Math.min(0.35, (level - 1) * 0.014));
}

// ───────────────────────── Вороги ─────────────────────────
export function makeEnemy(kind) {
  const d = ENEMIES[kind];
  const c = d.color;
  const g = new THREE.Group();
  const body = new THREE.Group();
  g.add(body);
  const sk = skin(c, 0.18);
  const carapace = metal(0x1d1620, 0.45, 0.4);
  const eye = glow(c, 5);
  const legs = [];
  const wings = [];
  let shield = null;
  const addLegs = (n, len, spread, y) => {
    for (let i = 0; i < n; i++) {
      const side = i % 2 ? 1 : -1;
      const row = Math.floor(i / 2) - (n / 2 - 1) / 2;
      const leg = new THREE.Group();
      leg.position.set(side * spread, y, row * 0.5);
      const seg = mesh(G.box(len, 0.12, 0.12), carapace, side * len / 2, 0, 0);
      seg.rotation.z = side * -0.6;
      leg.add(seg);
      body.add(leg);
      legs.push(leg);
    }
  };
  switch (kind) {
    case 'crawler': {
      body.add(mesh(G.sph(0.7, 16, 10).scale(1, 0.6, 1.3), sk, 0, 0.6, 0));
      body.add(mesh(G.sph(0.45, 12, 8).scale(1, 0.7, 1), carapace, 0, 0.75, -0.6));
      body.add(mesh(G.sph(0.12, 8, 6), eye, 0.2, 0.75, 0.82, false), mesh(G.sph(0.12, 8, 6), eye, -0.2, 0.75, 0.82, false));
      addLegs(6, 0.9, 0.5, 0.55);
      break;
    }
    case 'runner': {
      body.add(mesh(G.cyl(0.25, 0.35, 1.2, 10), sk, 0, 1.3, 0));
      body.children[0].rotation.x = 0.5;
      body.add(mesh(G.sph(0.3, 12, 8).scale(1, 0.8, 1.5), carapace, 0, 1.85, 0.4));
      body.add(mesh(G.box(0.3, 0.05, 0.05), eye, 0, 1.9, 0.82, false));
      for (const s of [-1, 1]) {
        const leg = new THREE.Group();
        leg.position.set(s * 0.25, 0.85, 0);
        leg.add(mesh(G.box(0.14, 0.9, 0.14), carapace, 0, -0.42, 0));
        body.add(leg);
        legs.push(leg);
      }
      break;
    }
    case 'swarm': {
      body.add(mesh(new THREE.OctahedronGeometry(0.45, 0), sk, 0, 0.6, 0));
      body.add(mesh(G.sph(0.14, 8, 6), eye, 0, 0.65, 0.32, false));
      for (let i = 0; i < 4; i++) {
        const sp = mesh(new THREE.ConeGeometry(0.08, 0.5, 4), carapace, 0, 0.6, 0);
        sp.rotation.z = (i / 4) * Math.PI * 2;
        sp.position.x = Math.sin(sp.rotation.z) * -0.4; sp.position.y = 0.6 + Math.cos(sp.rotation.z) * 0.4;
        body.add(sp);
      }
      break;
    }
    case 'brute': {
      body.add(mesh(G.rbox(1.5, 1.3, 1.2, 0.2), sk, 0, 1.6, 0));
      body.add(mesh(G.rbox(1.7, 0.6, 1.3, 0.2), carapace, 0, 2.35, -0.1));
      body.add(mesh(G.sph(0.25, 10, 8), eye, 0, 1.7, 0.62, false));
      body.add(mesh(G.rbox(0.6, 0.5, 0.6, 0.12), carapace, 0, 2.2, 0.55));
      for (const s of [-1, 1]) {
        const arm = new THREE.Group();
        arm.position.set(s * 1.0, 2.0, 0);
        arm.add(mesh(G.rbox(0.5, 1.4, 0.5, 0.12), carapace, 0, -0.6, 0.1));
        arm.add(mesh(G.rbox(0.65, 0.5, 0.65, 0.12), sk, 0, -1.35, 0.15));
        body.add(arm);
        legs.push(arm);
      }
      for (const s of [-1, 1]) {
        const leg = new THREE.Group();
        leg.position.set(s * 0.45, 0.95, 0);
        leg.add(mesh(G.rbox(0.45, 0.95, 0.5, 0.1), carapace, 0, -0.45, 0));
        body.add(leg);
        legs.push(leg);
      }
      break;
    }
    case 'flyer': {
      body.add(mesh(G.sph(0.45, 12, 8).scale(0.8, 0.7, 1.6), sk, 0, 0, 0));
      body.add(mesh(new THREE.ConeGeometry(0.2, 0.8, 6).rotateX(-Math.PI / 2), carapace, 0, 0, -0.95));
      body.add(mesh(G.sph(0.12, 8, 6), eye, 0, 0.05, 0.7, false));
      for (const s of [-1, 1]) {
        const w = new THREE.Group();
        w.position.set(s * 0.3, 0.15, 0);
        const wm = new THREE.Mesh(new THREE.PlaneGeometry(1.6, 0.8), new THREE.MeshStandardMaterial({ color: c, emissive: c, emissiveIntensity: 0.8, transparent: true, opacity: 0.55, side: THREE.DoubleSide, depthWrite: false }));
        wm.rotation.x = -Math.PI / 2;
        wm.position.x = s * 0.8;
        w.add(wm);
        body.add(w);
        wings.push(w);
      }
      break;
    }
    case 'shielder': {
      body.add(mesh(G.rbox(1.0, 1.2, 0.9, 0.2), sk, 0, 1.3, 0));
      body.add(mesh(G.sph(0.35, 12, 8), carapace, 0, 2.1, 0));
      body.add(mesh(G.box(0.4, 0.08, 0.05), eye, 0, 2.12, 0.33, false));
      body.add(mesh(G.rbox(1.3, 1.5, 0.15, 0.06), metal(0x334488, 0.3, 0.8), 0, 1.3, 0.7));
      addLegs(2, 0.3, 0.3, 0.7);
      legs.forEach((l) => { l.children[0].geometry = G.box(0.25, 0.8, 0.25); l.children[0].rotation.z = 0; l.children[0].position.set(0, -0.35, 0); });
      shield = new THREE.Mesh(G.sph(1.7, 24, 16), new THREE.MeshBasicMaterial({ color: 0x6688ff, transparent: true, opacity: 0.18, depthWrite: false, blending: THREE.AdditiveBlending }));
      shield.position.y = 1.3;
      g.add(shield);
      break;
    }
    case 'healer': {
      body.add(mesh(G.sph(0.6, 16, 12), sk, 0, 1.6, 0));
      body.add(mesh(G.sph(0.3, 12, 8), glow(0x44ffaa, 3), 0, 1.6, 0.4, false));
      for (let i = 0; i < 2; i++) {
        const r = mesh(new THREE.TorusGeometry(0.95 + i * 0.25, 0.04, 6, 32), glow(0x44ffaa, 2), 0, 1.6, 0, false);
        r.rotation.x = Math.PI / 2 + i * 0.6;
        body.add(r);
        wings.push(r);
      }
      body.add(mesh(new THREE.ConeGeometry(0.3, 1.0, 8).rotateX(Math.PI), carapace, 0, 0.75, 0));
      break;
    }
    case 'boss_titan': {
      body.add(mesh(G.rbox(2.2, 2.0, 1.7, 0.3), sk, 0, 2.6, 0));
      body.add(mesh(G.rbox(2.6, 0.9, 1.9, 0.3), carapace, 0, 3.8, -0.1));
      body.add(mesh(G.rbox(0.9, 0.8, 0.9, 0.2), carapace, 0, 3.7, 0.8));
      body.add(mesh(G.box(0.6, 0.14, 0.1), eye, 0, 3.75, 1.26, false));
      body.add(mesh(G.sph(0.45, 12, 8), glow(c, 4), 0, 2.6, 0.86, false));
      for (const s of [-1, 1]) {
        const horn = mesh(new THREE.ConeGeometry(0.25, 1.4, 6), carapace, s * 0.9, 4.6, 0.2);
        horn.rotation.z = s * -0.5;
        body.add(horn);
        for (let i = 0; i < 3; i++) body.add(mesh(new THREE.ConeGeometry(0.18, 0.8, 5), carapace, s * (0.4 + i * 0.4), 4.4, -0.7));
        const arm = new THREE.Group();
        arm.position.set(s * 1.5, 3.4, 0);
        arm.add(mesh(G.rbox(0.8, 2.2, 0.8, 0.2), carapace, 0, -1.0, 0.2));
        arm.add(mesh(G.rbox(1.0, 0.8, 1.0, 0.2), sk, 0, -2.2, 0.3));
        body.add(arm);
        legs.push(arm);
        const leg = new THREE.Group();
        leg.position.set(s * 0.7, 1.6, 0);
        leg.add(mesh(G.rbox(0.75, 1.6, 0.8, 0.15), carapace, 0, -0.8, 0));
        body.add(leg);
        legs.push(leg);
      }
      break;
    }
    case 'boss_hive': {
      body.add(mesh(G.sph(1.6, 24, 18).scale(1.1, 0.9, 1.4), sk, 0, 2.0, -0.4));
      body.add(mesh(G.sph(0.9, 16, 12), carapace, 0, 2.3, 1.4));
      for (let i = 0; i < 8; i++) {
        const a = (i / 8) * Math.PI * 2;
        body.add(mesh(G.sph(0.35, 10, 8), glow(c, 3), Math.cos(a) * 1.3, 2.6 + Math.sin(i) * 0.3, -0.4 + Math.sin(a) * 1.6, false));
      }
      body.add(mesh(G.sph(0.18, 8, 6), eye, 0.35, 2.5, 2.15, false), mesh(G.sph(0.18, 8, 6), eye, -0.35, 2.5, 2.15, false));
      addLegs(8, 2.0, 1.2, 1.4);
      break;
    }
    case 'boss_sentinel': {
      body.add(mesh(G.sph(1.3, 24, 18), carapace, 0, 0, 0));
      const eyeBall = mesh(G.sph(0.7, 16, 12), glow(c, 5), 0, 0, 0.85, false);
      body.add(eyeBall);
      for (let i = 0; i < 3; i++) {
        const r = mesh(new THREE.TorusGeometry(2.2 + i * 0.5, 0.12, 8, 48), i === 1 ? glow(c, 2.5) : metal(0x2a2040), 0, 0, 0);
        r.rotation.x = i * 1.1;
        body.add(r);
        wings.push(r);
      }
      for (let i = 0; i < 6; i++) {
        const a = (i / 6) * Math.PI * 2;
        const sp = mesh(new THREE.ConeGeometry(0.2, 1.4, 5), carapace, Math.cos(a) * 1.4, Math.sin(a) * 1.4, -0.3);
        sp.rotation.z = a - Math.PI / 2;
        body.add(sp);
      }
      shield = new THREE.Mesh(G.sph(3.6, 24, 16), new THREE.MeshBasicMaterial({ color: c, transparent: true, opacity: 0.15, depthWrite: false, blending: THREE.AdditiveBlending }));
      g.add(shield);
      break;
    }
    case 'w_mutant': {
      body.add(mesh(G.sph(0.7, 14, 10).scale(1, 0.85, 1.2), sk, 0, 1.2, 0));
      body.add(mesh(G.sph(0.4, 12, 8), carapace, 0, 1.55, 0.7));
      body.add(mesh(G.sph(0.09, 6, 4), eye, 0.15, 1.62, 1.05, false), mesh(G.sph(0.09, 6, 4), eye, -0.15, 1.62, 1.05, false));
      for (const s of [-1, 1]) {
        const arm = new THREE.Group();
        arm.position.set(s * 0.65, 1.3, 0.3);
        arm.add(mesh(G.box(0.18, 0.9, 0.18), carapace, 0, -0.4, 0.2));
        arm.add(mesh(new THREE.ConeGeometry(0.12, 0.5, 4).rotateX(Math.PI / 2), glow(c, 1), 0, -0.85, 0.45));
        body.add(arm);
        legs.push(arm);
        const leg = new THREE.Group();
        leg.position.set(s * 0.35, 0.75, 0);
        leg.add(mesh(G.box(0.2, 0.75, 0.2), carapace, 0, -0.37, 0));
        body.add(leg);
        legs.push(leg);
      }
      break;
    }
    case 'w_spitter': {
      body.add(mesh(G.sph(0.75, 16, 12), new THREE.MeshStandardMaterial({ color: 0x224411, emissive: c, emissiveIntensity: 0.6, transparent: true, opacity: 0.9, roughness: 0.1 }), 0, 1.1, -0.2));
      body.add(mesh(G.cyl(0.2, 0.3, 0.7, 8).rotateX(Math.PI / 2 - 0.4), carapace, 0, 1.3, 0.6));
      body.add(mesh(G.sph(0.1, 6, 4), eye, 0, 1.5, 0.75, false));
      addLegs(4, 0.8, 0.45, 0.6);
      break;
    }
    case 'w_alpha': {
      body.add(mesh(G.sph(1.0, 16, 12).scale(1.1, 0.9, 1.3), sk, 0, 1.7, 0));
      body.add(mesh(G.sph(0.55, 12, 8), carapace, 0, 2.1, 1.0));
      body.add(mesh(G.box(0.5, 0.08, 0.05), eye, 0, 2.2, 1.52, false));
      for (let i = 0; i < 5; i++) body.add(mesh(new THREE.ConeGeometry(0.16, 0.9, 5), glow(c, 1.3), 0, 2.5, -0.6 + i * 0.35));
      for (const s of [-1, 1]) {
        const arm = new THREE.Group();
        arm.position.set(s * 0.95, 1.9, 0.4);
        arm.add(mesh(G.box(0.3, 1.3, 0.3), carapace, 0, -0.6, 0.2));
        arm.add(mesh(new THREE.ConeGeometry(0.2, 0.8, 4).rotateX(Math.PI / 2), glow(c, 1.5), 0, -1.25, 0.55));
        body.add(arm);
        legs.push(arm);
        const leg = new THREE.Group();
        leg.position.set(s * 0.5, 1.0, 0);
        leg.add(mesh(G.box(0.3, 1.0, 0.3), carapace, 0, -0.5, 0));
        body.add(leg);
        legs.push(leg);
      }
      break;
    }
  }
  const s = kind.startsWith('boss') ? 1 : d.size;
  if (!kind.startsWith('boss')) body.scale.setScalar(s);
  else body.scale.setScalar(kind === 'boss_sentinel' ? 1 : 1.15);
  body.traverse((o) => { if (o.isMesh && o.material !== eye) o.castShadow = true; });
  g.userData = { body, legs, wings, shield, kind, flying: !!d.flying, boss: !!d.boss, hitFlash: 0 };
  return g;
}

// ───────────────────────── Лут ─────────────────────────
const RCOL = [0xc8d0d8, 0x4dff7a, 0x3aa6ff, 0xc057ff, 0xffa02a];
export function makeLoot(rarity) {
  const g = new THREE.Group();
  const col = RCOL[rarity];
  const gem = mesh(new THREE.OctahedronGeometry(0.4 + rarity * 0.06), glow(col, 3), 0, 1, 0, false);
  g.add(gem);
  const beamMat = new THREE.MeshBasicMaterial({ color: col, transparent: true, opacity: 0.35, blending: THREE.AdditiveBlending, depthWrite: false });
  const beam = new THREE.Mesh(new THREE.CylinderGeometry(0.12, 0.35, 6 + rarity * 3, 8, 1, true), beamMat);
  beam.position.y = (6 + rarity * 3) / 2;
  g.add(beam);
  const ring = new THREE.Mesh(new THREE.RingGeometry(0.6, 0.8, 24), new THREE.MeshBasicMaterial({ color: col, transparent: true, opacity: 0.7, depthWrite: false, side: THREE.DoubleSide }));
  ring.rotation.x = -Math.PI / 2;
  ring.position.y = 0.05;
  g.add(ring);
  g.userData = { gem };
  return g;
}

export function makeGhost(type) {
  const g = makeTower(type);
  g.traverse((o) => {
    if (o.isMesh) {
      o.material = new THREE.MeshBasicMaterial({ color: 0x3ae0ff, transparent: true, opacity: 0.35, depthWrite: false });
      o.castShadow = false;
    }
  });
  const range = new THREE.Mesh(new THREE.RingGeometry(0.97, 1, 96), new THREE.MeshBasicMaterial({ color: 0x3ae0ff, transparent: true, opacity: 0.6, depthWrite: false, side: THREE.DoubleSide }));
  range.rotation.x = -Math.PI / 2;
  range.position.y = 0.15;
  const fill = new THREE.Mesh(new THREE.CircleGeometry(1, 96), new THREE.MeshBasicMaterial({ color: 0x3ae0ff, transparent: true, opacity: 0.06, depthWrite: false }));
  fill.rotation.x = -Math.PI / 2;
  fill.position.y = 0.14;
  g.add(range, fill);
  g.userData.range = range;
  g.userData.fill = fill;
  return g;
}
