// Ефекти: трасери, дульні спалахи, вибухи (вогонь + дим + земля), сліди ракет, кіптява, пожежі.
import * as THREE from 'three';
import { terrainHeight } from './scene.js';

function softTex() {
  const c = document.createElement('canvas');
  c.width = c.height = 64;
  const g = c.getContext('2d');
  const gr = g.createRadialGradient(32, 32, 0, 32, 32, 32);
  gr.addColorStop(0, 'rgba(255,255,255,1)'); gr.addColorStop(0.45, 'rgba(255,255,255,0.55)'); gr.addColorStop(1, 'rgba(255,255,255,0)');
  g.fillStyle = gr; g.fillRect(0, 0, 64, 64);
  return new THREE.CanvasTexture(c);
}
function smokeTex() {
  const c = document.createElement('canvas');
  c.width = c.height = 128;
  const g = c.getContext('2d');
  for (let i = 0; i < 26; i++) {
    const x = 64 + (Math.random() - 0.5) * 50, y = 64 + (Math.random() - 0.5) * 50, r = 18 + Math.random() * 26;
    const gr = g.createRadialGradient(x, y, 0, x, y, r);
    gr.addColorStop(0, 'rgba(255,255,255,0.22)'); gr.addColorStop(1, 'rgba(255,255,255,0)');
    g.fillStyle = gr; g.fillRect(0, 0, 128, 128);
  }
  return new THREE.CanvasTexture(c);
}

class Particles {
  constructor(scene, N, blending, tex) {
    this.N = N;
    const geo = new THREE.BufferGeometry();
    this.pos = new Float32Array(N * 3); this.col = new Float32Array(N * 3); this.size = new Float32Array(N); this.alpha = new Float32Array(N);
    this.rot = new Float32Array(N);
    this.vel = new Float32Array(N * 3); this.life = new Float32Array(N); this.max = new Float32Array(N); this.grav = new Float32Array(N);
    this.drag = new Float32Array(N); this.s0 = new Float32Array(N); this.s1 = new Float32Array(N); this.a0 = new Float32Array(N);
    geo.setAttribute('position', new THREE.BufferAttribute(this.pos, 3).setUsage(THREE.DynamicDrawUsage));
    geo.setAttribute('color', new THREE.BufferAttribute(this.col, 3).setUsage(THREE.DynamicDrawUsage));
    geo.setAttribute('size', new THREE.BufferAttribute(this.size, 1).setUsage(THREE.DynamicDrawUsage));
    geo.setAttribute('alpha', new THREE.BufferAttribute(this.alpha, 1).setUsage(THREE.DynamicDrawUsage));
    geo.setAttribute('rot', new THREE.BufferAttribute(this.rot, 1));
    const additive = blending === THREE.AdditiveBlending;
    this.mat = new THREE.ShaderMaterial({
      transparent: true, depthWrite: false, blending,
      uniforms: { scale: { value: window.innerHeight / 2 }, map: { value: tex }, fogColor: { value: new THREE.Color() }, fogDensity: { value: 0 } },
      vertexShader: `attribute float size; attribute float alpha; attribute float rot; attribute vec3 color;
        varying vec3 vC; varying float vA; varying float vR; varying float vFog; uniform float scale;
        void main(){ vC = color; vA = alpha; vR = rot; vec4 mv = modelViewMatrix * vec4(position, 1.0);
          gl_PointSize = size * scale / -mv.z; gl_Position = projectionMatrix * mv; vFog = -mv.z; }`,
      fragmentShader: `varying vec3 vC; varying float vA; varying float vR; varying float vFog; uniform sampler2D map; uniform vec3 fogColor; uniform float fogDensity;
        void main(){ vec2 p = gl_PointCoord - 0.5; float c = cos(vR), s = sin(vR); p = vec2(c*p.x - s*p.y, s*p.x + c*p.y) + 0.5;
          vec4 t = texture2D(map, p); float f = 1.0 - exp(-fogDensity*fogDensity*vFog*vFog);
          ${additive ? 'gl_FragColor = vec4(vC * t.a * vA * (1.0 - f), 1.0);' : 'gl_FragColor = vec4(mix(vC, fogColor, f), t.a * vA);'} }`,
    });
    if (additive) this.mat.blending = THREE.AdditiveBlending;
    this.points = new THREE.Points(geo, this.mat);
    this.points.frustumCulled = false;
    this.points.renderOrder = additive ? 3 : 2;
    scene.add(this.points);
    this.next = 0;
    window.addEventListener('resize', () => (this.mat.uniforms.scale.value = window.innerHeight / 2));
  }
  emit(x, y, z, color, count, o) {
    const c = new THREE.Color(color);
    const sp = o.speed ?? 4, life = o.life ?? 1, size = o.size ?? 1, size1 = o.size1 ?? size, up = o.up ?? 0.5, spread = o.spread ?? 0;
    for (let k = 0; k < count; k++) {
      const i = this.next; this.next = (this.next + 1) % this.N;
      const th = Math.random() * Math.PI * 2, ph = Math.acos(2 * Math.random() - 1);
      const s = sp * (0.25 + Math.random() * 0.75);
      this.pos[i * 3] = x + (Math.random() - 0.5) * spread; this.pos[i * 3 + 1] = y + Math.random() * spread * 0.3; this.pos[i * 3 + 2] = z + (Math.random() - 0.5) * spread;
      this.vel[i * 3] = Math.sin(ph) * Math.cos(th) * s + (o.vx || 0);
      this.vel[i * 3 + 1] = Math.abs(Math.cos(ph)) * s * up + (o.vy || 0);
      this.vel[i * 3 + 2] = Math.sin(ph) * Math.sin(th) * s + (o.vz || 0);
      const v = (o.vary ?? 0.25);
      const m = 1 - v / 2 + Math.random() * v;
      this.col[i * 3] = c.r * m; this.col[i * 3 + 1] = c.g * m; this.col[i * 3 + 2] = c.b * m;
      const l = life * (0.6 + Math.random() * 0.7);
      this.life[i] = l; this.max[i] = l;
      this.s0[i] = size * (0.7 + Math.random() * 0.6); this.s1[i] = size1 * (0.7 + Math.random() * 0.6);
      this.a0[i] = o.alpha ?? 1;
      this.grav[i] = o.grav ?? 0; this.drag[i] = o.drag ?? 1.5;
      this.rot[i] = Math.random() * 6.28;
    }
    this.points.geometry.attributes.rot.needsUpdate = true;
  }
  update(dt, fog) {
    for (let i = 0; i < this.N; i++) {
      if (this.life[i] <= 0) { if (this.alpha[i] !== 0) { this.alpha[i] = 0; this.size[i] = 0; } continue; }
      this.life[i] -= dt;
      const k = Math.max(0, this.life[i] / this.max[i]);
      const d = Math.exp(-this.drag[i] * dt);
      this.vel[i * 3] *= d; this.vel[i * 3 + 1] = this.vel[i * 3 + 1] * d - this.grav[i] * 9.8 * dt; this.vel[i * 3 + 2] *= d;
      this.pos[i * 3] += this.vel[i * 3] * dt; this.pos[i * 3 + 1] += this.vel[i * 3 + 1] * dt; this.pos[i * 3 + 2] += this.vel[i * 3 + 2] * dt;
      if (this.pos[i * 3 + 1] < 0.05) { this.pos[i * 3 + 1] = 0.05; this.vel[i * 3 + 1] *= -0.2; this.vel[i * 3] *= 0.5; this.vel[i * 3 + 2] *= 0.5; }
      this.alpha[i] = this.a0[i] * Math.min(1, k * 1.6) * Math.min(1, (1 - k) * 12 + 0.2);
      this.size[i] = this.s1[i] + (this.s0[i] - this.s1[i]) * k;
    }
    const a = this.points.geometry.attributes;
    a.position.needsUpdate = a.color.needsUpdate = a.size.needsUpdate = a.alpha.needsUpdate = true;
    if (fog) { this.mat.uniforms.fogColor.value.copy(fog.color); this.mat.uniforms.fogDensity.value = fog.density; }
  }
}

const _t = new THREE.Vector3();
const TRACER = { hero: 0xffc070, bullet: 0xffb050, aa: 0xffd890, ebullet: 0xff8060 };

export class FX {
  constructor(world) {
    this.world = world;
    this.scene = world.scene;
    this.mul = world.q.particles;
    this.add = new Particles(this.scene, 5000, THREE.AdditiveBlending, softTex());
    this.smoke = new Particles(this.scene, 5000, THREE.NormalBlending, smokeTex());
    this.projectiles = new Map();
    this.tracers = [];
    this.pending = [];
    this.fires = [];
    this.tracerGeo = new THREE.BoxGeometry(0.05, 0.05, 1);
    this.shellGeo = new THREE.CylinderGeometry(0.06, 0.06, 0.4, 6).rotateX(Math.PI / 2);
    this.missileGeo = new THREE.CylinderGeometry(0.07, 0.07, 0.9, 8).rotateX(Math.PI / 2);
    this.tracerMats = {};
    this.shellMat = new THREE.MeshStandardMaterial({ color: 0x3a3b33, roughness: 0.5, metalness: 0.5 });
    // кіптява на землі
    const sc = document.createElement('canvas'); sc.width = sc.height = 128;
    const g = sc.getContext('2d');
    for (let i = 0; i < 18; i++) {
      const x = 64 + (Math.random() - 0.5) * 30, y = 64 + (Math.random() - 0.5) * 30, r = 20 + Math.random() * 40;
      const gr = g.createRadialGradient(x, y, 0, x, y, r);
      gr.addColorStop(0, 'rgba(20,16,12,0.35)'); gr.addColorStop(1, 'rgba(20,16,12,0)');
      g.fillStyle = gr; g.fillRect(0, 0, 128, 128);
    }
    this.decalTex = new THREE.CanvasTexture(sc);
    this.decalGeo = new THREE.PlaneGeometry(1, 1).rotateX(-Math.PI / 2);
    this.decals = [];
  }
  // пул сіток: снаряди й трасери не створюються заново на кожен постріл
  mesh(geo, mat) {
    const key = geo.uuid + mat.uuid;
    const list = (this._pool ||= {})[key] ||= [];
    let m = list.pop();
    if (!m) { m = new THREE.Mesh(geo, mat); m.userData.poolKey = key; this.scene.add(m); }
    m.visible = true; m.scale.set(1, 1, 1);
    return m;
  }
  release(m) { m.visible = false; this._pool[m.userData.poolKey].push(m); }
  emit(x, y, z, color, count, o = {}) { this.add.emit(x, y, z, color, Math.max(1, Math.round(count * this.mul)), o); }
  puff(x, y, z, color, count, o = {}) { this.smoke.emit(x, y, z, color, Math.max(1, Math.round(count * this.mul)), o); }

  tracerMat(color) {
    if (!this.tracerMats[color]) this.tracerMats[color] = new THREE.MeshBasicMaterial({ color: new THREE.Color(color).multiplyScalar(2.2), transparent: true, opacity: 0.95, blending: THREE.AdditiveBlending, depthWrite: false });
    return this.tracerMats[color];
  }
  muzzle(x, y, z, big = 1, dir = null) {
    this.emit(x, y, z, 0xffc070, 3, { speed: 2, life: 0.06, size: 0.7 * big, size1: 0.2, drag: 6, vx: dir ? dir.x * 6 : 0, vz: dir ? dir.z * 6 : 0 });
    this.puff(x, y, z, 0x9a948a, 1 + Math.round(big), { speed: 0.8, life: 0.9 * big, size: 0.5 * big, size1: 1.6 * big, alpha: 0.35, up: 1, grav: -0.03 });
    if (big >= 1.5) this.world.flash(x, y, z, 0xffb060, 25 * big, 0.08);
  }
  spawnProjectile(ev) {
    const k = ev.k;
    let m;
    if (TRACER[k]) {
      m = this.mesh(this.tracerGeo, this.tracerMat(k === 'hero' ? 0xffd080 : TRACER[k]));
      m.scale.z = k === 'aa' ? 3.5 : 2.6;
    } else if (k === 'atgm' || k === 'rocket') {
      m = this.mesh(this.missileGeo, this.shellMat);
    } else {
      m = this.mesh(this.shellGeo, this.shellMat);
      if (k === 'agsg') m.scale.setScalar(0.6);
    }
    m.position.set(ev.p[0], ev.p[1], ev.p[2]);
    const v = new THREE.Vector3(...ev.v);
    if (v.lengthSq() > 0) m.lookAt(_t.copy(m.position).add(v));
    this.projectiles.set(ev.id, { m, v, g: ev.g || 0, ttl: ev.ttl + 0.1, k });
    if (k === 'atgm' || k === 'rocket') this.puff(ev.p[0], ev.p[1], ev.p[2], 0xc8c4bc, 6, { speed: 2, life: 1.2, size: 0.8, size1: 2.5, alpha: 0.5, up: 0.6 });
  }
  endProjectile(ev) {
    const p = this.projectiles.get(ev.id);
    if (!p) return;
    this.release(p.m);
    this.projectiles.delete(ev.id);
    const pos = p.m.position;
    if (p.k === 'mortar' || p.k === 'shell' || p.k === 'agsg' || p.k === 'rocket') return; // окремий вибух
    if (p.k === 'atgm') { this.explosion(pos.x, pos.z, 2.5, 0.8, pos.y); return; }
    // влучання кулі: іскри або пил
    const y = Math.max(0.3, pos.y);
    this.emit(pos.x, y, pos.z, 0xffc070, 2, { speed: 4, life: 0.15, size: 0.25, grav: 1 });
    this.puff(pos.x, y, pos.z, 0x8a7f6a, 2, { speed: 1.2, life: 0.6, size: 0.3, size1: 1.0, alpha: 0.45, up: 1 });
  }
  hitSpark(x, y, z, human) {
    if (human) this.puff(x, y, z, 0x5a1a14, 2, { speed: 1.5, life: 0.4, size: 0.25, size1: 0.6, alpha: 0.7, grav: 0.6 });
    else this.emit(x, y, z, 0xffd090, 3, { speed: 5, life: 0.18, size: 0.25, grav: 1 });
  }
  scorch(x, z, r) {
    let d = this.decals.find((q) => q.life <= 0);
    if (!d) {
      if (this.decals.length < 80) {
        const m = new THREE.Mesh(this.decalGeo, new THREE.MeshBasicMaterial({ map: this.decalTex, transparent: true, depthWrite: false, polygonOffset: true, polygonOffsetFactor: -4 }));
        m.renderOrder = 1;
        this.scene.add(m);
        d = { m, life: 0 };
        this.decals.push(d);
      } else d = this.decals.reduce((a, b) => (a.life < b.life ? a : b));
    }
    d.life = 60;
    d.m.position.set(x, terrainHeight(x, z) + 0.06, z);
    d.m.scale.set(r * 2.2, 1, r * 2.2);
    d.m.rotation.y = Math.random() * 6;
    d.m.material.opacity = 1;
  }
  explosion(x, z, r, big = 1, y0 = null) {
    const y = y0 ?? terrainHeight(x, z) + 0.3;
    // спалах і вогонь
    this.emit(x, y + 0.4, z, 0xffd28a, 6 * big, { speed: r * 1.2, life: 0.12, size: r * 1.3, size1: r * 0.6, drag: 8 });
    this.emit(x, y + 0.5, z, 0xff7a28, 22 * big, { speed: r * 2.4, life: 0.45, size: r * 0.7, size1: r * 0.2, up: 0.9, drag: 4 });
    // земля і уламки
    this.puff(x, y, z, 0x4e4234, 26 * big, { speed: r * 3.2, life: 1.2, size: 0.25, size1: 0.2, alpha: 0.9, up: 1.6, grav: 1.1, drag: 0.4 });
    this.emit(x, y, z, 0xffa040, 10 * big, { speed: r * 4, life: 0.6, size: 0.12, up: 1.4, grav: 1, drag: 0.6 });
    // дим
    this.puff(x, y + 0.4, z, 0x5a534b, 14 * big, { speed: r * 0.8, life: 3.2, size: r * 0.6, size1: r * 2.2, alpha: 0.42, up: 1, grav: -0.06, drag: 1.2, spread: r * 0.4 });
    this.puff(x, y + 0.2, z, 0x9a8f7e, 10 * big, { speed: r * 1.6, life: 2.0, size: r * 0.5, size1: r * 1.8, alpha: 0.35, up: 0.2, drag: 1.5, spread: r * 0.5 });
    this.world.flash(x, y + 2, z, 0xff9a40, 70 * big, 0.25);
    if (y0 == null || y0 < 1) this.scorch(x, z, r * 0.8);
    this.world.shake = Math.max(this.world.shake || 0, 0.12 * big * Math.min(1.5, r / 3));
  }
  fire(x, z, dur, size = 1, y = null) { this.fires.push({ x, z, y: y ?? terrainHeight(x, z) + 0.6, t: dur, size }); }
  artillery(x, z) {
    for (let i = 0; i < 5; i++) {
      const ox = (Math.random() - 0.5) * 9, oz = (Math.random() - 0.5) * 9;
      this.pending.push({ t: 1.2 + i * 0.12, fn: () => this.explosion(x + ox, z + oz, 4, 1.3) });
    }
  }
  dash(a, b) {
    for (let i = 0; i <= 6; i++) { const t = i / 6; this.puff(a[0] + (b[0] - a[0]) * t, 0.3, a[1] + (b[1] - a[1]) * t, 0x8a7f6a, 2, { speed: 1, life: 0.8, size: 0.5, size1: 1.5, alpha: 0.4 }); }
  }
  grenade(a, b) {
    const start = new THREE.Vector3(a[0], 1.6, a[1]), end = new THREE.Vector3(b[0], terrainHeight(b[0], b[1]) + 0.1, b[1]);
    const m = new THREE.Mesh(new THREE.SphereGeometry(0.07, 8, 6), this.shellMat);
    this.scene.add(m);
    let t = 0;
    this.pending.push({ every: true, fn: (dt) => {
      t += dt / 0.6;
      const p = start.clone().lerp(end, Math.min(1, t));
      p.y += Math.sin(Math.min(1, t) * Math.PI) * 4;
      m.position.copy(p);
      if (t >= 1) { this.scene.remove(m); return true; }
      return false;
    } });
  }
  eshot(a, b, y, k) {
    const ya = terrainHeight(a[0], a[1]) + y;
    const from = new THREE.Vector3(a[0], ya, a[1]), to = new THREE.Vector3(b[0], terrainHeight(b[0], b[1]) + 1.2, b[1]);
    const dir = to.clone().sub(from).normalize();
    this.muzzle(from.x + dir.x, from.y, from.z + dir.z, k === 'cannon' ? 3 : 1, dir);
    const n = k === 'mg' ? 3 : 1;
    for (let i = 0; i < n; i++) this.tracer(from, to, TRACER.ebullet, i * 0.07);
  }
  tracer(from, to, color, delay = 0) {
    const m = this.mesh(this.tracerGeo, this.tracerMat(color));
    m.scale.z = 2.5; m.visible = false;
    const len = from.distanceTo(to);
    this.tracers.push({ m, from: from.clone(), to: to.clone(), t: -delay, dur: Math.max(0.05, len / 120) });
  }

  update(dt) {
    const fog = this.scene.fog;
    this.add.update(dt, fog);
    this.smoke.update(dt, fog);
    for (const [id, p] of this.projectiles) {
      p.ttl -= dt;
      p.m.position.x += p.v.x * dt; p.m.position.y += p.v.y * dt - 0.5 * p.g * dt * dt; p.m.position.z += p.v.z * dt;
      p.v.y -= p.g * dt;
      if (p.g) p.m.lookAt(_t.copy(p.m.position).add(p.v));
      if (p.k === 'atgm' || p.k === 'rocket') {
        this.emit(p.m.position.x, p.m.position.y, p.m.position.z, 0xffb060, 1, { speed: 0.2, life: 0.08, size: 0.5 });
        this.puff(p.m.position.x, p.m.position.y, p.m.position.z, 0xbdb8ae, 1, { speed: 0.3, life: 1.6, size: 0.4, size1: 1.6, alpha: 0.45, up: 0.3 });
      }
      if (p.ttl <= 0) { this.release(p.m); this.projectiles.delete(id); }
    }
    for (let i = this.tracers.length - 1; i >= 0; i--) {
      const t = this.tracers[i];
      t.t += dt;
      if (t.t < 0) continue;
      const k = t.t / t.dur;
      if (k >= 1) { this.release(t.m); this.tracers.splice(i, 1); continue; }
      t.m.visible = true;
      t.m.position.copy(t.from).lerp(t.to, k);
      t.m.lookAt(t.to);
    }
    for (let i = this.fires.length - 1; i >= 0; i--) {
      const f = this.fires[i];
      f.t -= dt;
      if (f.t <= 0) { this.fires.splice(i, 1); continue; }
      if (Math.random() < dt * 30) this.emit(f.x + (Math.random() - 0.5) * f.size, f.y, f.z + (Math.random() - 0.5) * f.size, 0xff8a30, 1, { speed: 1, life: 0.5, size: 0.9 * f.size, size1: 0.2, up: 1, vy: 2, drag: 1 });
      if (Math.random() < dt * 12) this.puff(f.x, f.y + 0.8, f.z, 0x3a3532, 1, { speed: 0.6, life: 4, size: 1 * f.size, size1: 4.5 * f.size, alpha: 0.4, vy: 2.5, drag: 0.4 });
    }
    for (const d of this.decals) if (d.life > 0) { d.life -= dt; d.m.material.opacity = Math.min(1, d.life / 10); if (d.life <= 0) d.m.visible = false; else d.m.visible = true; }
    for (let i = this.pending.length - 1; i >= 0; i--) {
      const p = this.pending[i];
      if (p.every) { if (p.fn(dt)) this.pending.splice(i, 1); continue; }
      p.t -= dt;
      if (p.t <= 0) { this.pending.splice(i, 1); p.fn(); }
    }
  }
}
