// Візуальні ефекти: частинки, снаряди, промені, блискавки, ударні хвилі.
import * as THREE from 'three';

const PCOLORS = {
  blaster: 0x33e0ff, cryo: 0x9fd6ff, tesla: 0xbb77ff, rail: 0xffee55, mortar: 0xff7733, laser: 0xff3388,
  acid: 0x88ff44, evil: 0xc057ff, drone: 0x66ffcc, gren: 0xff9933, orbit: 0x66ccff,
};

export class FX {
  constructor(world) {
    this.world = world;
    this.scene = world.scene;
    this.mul = world.q.particles;
    this.initParticles(9000);
    this.projectiles = new Map();
    this.beams = [];
    this.rings = [];
    this.pending = [];
    this.projGeo = new THREE.SphereGeometry(0.22, 8, 6);
    this.projMats = {};
    this.beamGeo = new THREE.CylinderGeometry(1, 1, 1, 8, 1, true).rotateX(Math.PI / 2);
    this.ringGeo = new THREE.RingGeometry(0.85, 1, 64).rotateX(-Math.PI / 2);
    this.discGeo = new THREE.CircleGeometry(1, 48).rotateX(-Math.PI / 2);
    this.beamPool = [];
  }

  initParticles(N) {
    this.N = N;
    const geo = new THREE.BufferGeometry();
    this.pPos = new Float32Array(N * 3);
    this.pCol = new Float32Array(N * 3);
    this.pSize = new Float32Array(N);
    this.pAlpha = new Float32Array(N);
    this.pVel = new Float32Array(N * 3);
    this.pLife = new Float32Array(N);
    this.pMax = new Float32Array(N);
    this.pGrav = new Float32Array(N);
    this.pDrag = new Float32Array(N);
    this.pS0 = new Float32Array(N);
    geo.setAttribute('position', new THREE.BufferAttribute(this.pPos, 3).setUsage(THREE.DynamicDrawUsage));
    geo.setAttribute('color', new THREE.BufferAttribute(this.pCol, 3).setUsage(THREE.DynamicDrawUsage));
    geo.setAttribute('size', new THREE.BufferAttribute(this.pSize, 1).setUsage(THREE.DynamicDrawUsage));
    geo.setAttribute('alpha', new THREE.BufferAttribute(this.pAlpha, 1).setUsage(THREE.DynamicDrawUsage));
    const mat = new THREE.ShaderMaterial({
      transparent: true, depthWrite: false, blending: THREE.AdditiveBlending,
      uniforms: { scale: { value: window.innerHeight / 2 } },
      vertexShader: `attribute float size; attribute float alpha; attribute vec3 color; varying vec3 vC; varying float vA;
        uniform float scale;
        void main(){ vC = color; vA = alpha; vec4 mv = modelViewMatrix*vec4(position,1.0); gl_PointSize = size * scale / -mv.z; gl_Position = projectionMatrix*mv; }`,
      fragmentShader: `varying vec3 vC; varying float vA;
        void main(){ vec2 p = gl_PointCoord*2.0-1.0; float d = dot(p,p); if(d>1.0) discard; float f = pow(1.0-d, 1.6); gl_FragColor = vec4(vC*f*1.6, f*vA); }`,
    });
    this.pMat = mat;
    this.points = new THREE.Points(geo, mat);
    this.points.frustumCulled = false;
    this.scene.add(this.points);
    this.pNext = 0;
    window.addEventListener('resize', () => (mat.uniforms.scale.value = window.innerHeight / 2));
  }

  emit(x, y, z, color, count, o = {}) {
    const c = new THREE.Color(color);
    const n = Math.max(1, Math.round(count * this.mul));
    const sp = o.speed ?? 6, life = o.life ?? 0.6, size = o.size ?? 0.6, grav = o.grav ?? 0, drag = o.drag ?? 2, up = o.up ?? 0.5, spread = o.spread ?? 0;
    for (let k = 0; k < n; k++) {
      const i = this.pNext;
      this.pNext = (this.pNext + 1) % this.N;
      const th = Math.random() * Math.PI * 2, ph = Math.acos(2 * Math.random() - 1);
      const s = sp * (0.3 + Math.random() * 0.7);
      this.pPos[i * 3] = x + (Math.random() - 0.5) * spread;
      this.pPos[i * 3 + 1] = y + (Math.random() - 0.5) * spread * 0.5;
      this.pPos[i * 3 + 2] = z + (Math.random() - 0.5) * spread;
      this.pVel[i * 3] = Math.sin(ph) * Math.cos(th) * s + (o.vx || 0);
      this.pVel[i * 3 + 1] = Math.abs(Math.cos(ph)) * s * up + (o.vy || 0) + (up > 0.5 ? s * 0.3 : 0);
      this.pVel[i * 3 + 2] = Math.sin(ph) * Math.sin(th) * s + (o.vz || 0);
      const v = 0.75 + Math.random() * 0.5;
      this.pCol[i * 3] = c.r * v; this.pCol[i * 3 + 1] = c.g * v; this.pCol[i * 3 + 2] = c.b * v;
      const l = life * (0.5 + Math.random() * 0.8);
      this.pLife[i] = l; this.pMax[i] = l;
      this.pS0[i] = size * (0.6 + Math.random() * 0.8);
      this.pGrav[i] = grav; this.pDrag[i] = drag;
    }
  }

  // ───── Снаряди ─────
  projMat(k, c) {
    const key = k + (c || '');
    if (!this.projMats[key]) this.projMats[key] = new THREE.MeshBasicMaterial({ color: new THREE.Color(c ?? PCOLORS[k] ?? 0xffffff).multiplyScalar(3) });
    return this.projMats[key];
  }
  spawnProjectile(ev) {
    const m = new THREE.Mesh(this.projGeo, this.projMat(ev.k, ev.c));
    const size = { hero: 0.9, blaster: 1.1, mortar: 2.6, acid: 1.6, evil: 2.4 }[ev.k] || 1;
    m.scale.set(size * 0.8, size * 0.8, size * (ev.k === 'hero' || ev.k === 'blaster' ? 3.5 : 1));
    m.position.set(ev.p[0], ev.p[1], ev.p[2]);
    this.scene.add(m);
    this.projectiles.set(ev.id, { m, v: new THREE.Vector3(...ev.v), g: ev.g || 0, ttl: ev.ttl + 0.15, k: ev.k, c: ev.c ?? PCOLORS[ev.k] });
    const dir = new THREE.Vector3(...ev.v);
    if (dir.lengthSq() > 0) m.lookAt(m.position.clone().add(dir));
  }
  endProjectile(ev) {
    const p = this.projectiles.get(ev.id);
    if (!p) return;
    this.scene.remove(p.m);
    this.projectiles.delete(ev.id);
    const pos = p.m.position;
    if (ev.k === 'mortar') return; // вибух буде окремою подією
    this.emit(pos.x, Math.max(0.3, pos.y), pos.z, p.c, ev.k === 'hero' ? 4 : 6, { speed: 5, life: 0.3, size: 0.45 });
  }

  // ───── Промені ─────
  beam(a, b, color, width = 0.12, life = 0.15) {
    const mat = new THREE.MeshBasicMaterial({ color: new THREE.Color(color).multiplyScalar(2.5), transparent: true, opacity: 1, blending: THREE.AdditiveBlending, depthWrite: false });
    const m = new THREE.Mesh(this.beamGeo, mat);
    const va = new THREE.Vector3(...a), vb = new THREE.Vector3(...b);
    const len = va.distanceTo(vb);
    m.position.copy(va).lerp(vb, 0.5);
    m.lookAt(vb);
    m.scale.set(width, width, len);
    this.scene.add(m);
    const core = new THREE.Mesh(this.beamGeo, new THREE.MeshBasicMaterial({ color: 0xffffff, transparent: true, blending: THREE.AdditiveBlending, depthWrite: false }));
    core.position.copy(m.position); core.quaternion.copy(m.quaternion); core.scale.set(width * 0.35, width * 0.35, len);
    this.scene.add(core);
    this.beams.push({ ms: [m, core], life, max: life, w: width });
  }
  lightning(points, color) {
    for (let i = 0; i < points.length - 1; i++) {
      const a = new THREE.Vector3(...points[i]), b = new THREE.Vector3(...points[i + 1]);
      const segs = 6;
      let prev = a.clone();
      for (let s = 1; s <= segs; s++) {
        const p = a.clone().lerp(b, s / segs);
        if (s < segs) p.add(new THREE.Vector3((Math.random() - 0.5) * 1.2, (Math.random() - 0.5) * 1.2, (Math.random() - 0.5) * 1.2));
        this.beam(prev.toArray(), p.toArray(), color, 0.07, 0.18);
        prev = p;
      }
      this.emit(b.x, b.y, b.z, color, 6, { speed: 6, life: 0.3, size: 0.5 });
    }
  }
  ring(x, z, r, color, life = 0.6, y = 0.2, width = 1) {
    const mat = new THREE.MeshBasicMaterial({ color: new THREE.Color(color).multiplyScalar(2), transparent: true, blending: THREE.AdditiveBlending, depthWrite: false, side: THREE.DoubleSide });
    const m = new THREE.Mesh(this.ringGeo, mat);
    m.position.set(x, y, z);
    this.scene.add(m);
    this.rings.push({ m, life, max: life, r, w: width });
  }
  explosion(x, z, r, color, big = 1) {
    const y = 0.8;
    this.emit(x, y, z, color, 40 * big, { speed: r * 2.5, life: 0.7, size: 1.4, up: 0.8, drag: 3 });
    this.emit(x, y, z, 0xffffff, 14 * big, { speed: r * 1.2, life: 0.35, size: 2.2, drag: 4 });
    this.emit(x, y, z, 0x333344, 16 * big, { speed: r * 0.7, life: 1.4, size: 2.6, up: 1, grav: -1, drag: 1 });
    this.ring(x, z, r, color, 0.5);
    this.world.flash(x, 2, z, color, 80 * big, 0.3);
  }
  orbital(x, z) {
    // маркер прицілу
    this.ring(x, z, 9, 0x66ccff, 1.2, 0.25);
    this.pending.push({ t: 1.2, fn: () => {
      this.beam([x, 120, z], [x, 0, z], 0x66ccff, 3.5, 0.6);
      this.beam([x, 120, z], [x, 0, z], 0xffffff, 1.2, 0.5);
      this.explosion(x, z, 9, 0x66ccff, 3);
      this.world.shake = Math.max(this.world.shake || 0, 0.8);
    } });
  }
  dash(a, b, color) {
    const steps = 10;
    for (let i = 0; i <= steps; i++) {
      const t = i / steps;
      this.emit(a[0] + (b[0] - a[0]) * t, 1.2, a[1] + (b[1] - a[1]) * t, color, 3, { speed: 1.5, life: 0.45, size: 1.1 });
    }
  }
  grenade(a, b) {
    const start = new THREE.Vector3(a[0], 1.6, a[1]);
    const end = new THREE.Vector3(b[0], 0.3, b[1]);
    const m = new THREE.Mesh(this.projGeo, this.projMat('gren'));
    m.scale.setScalar(1.6);
    this.scene.add(m);
    let t = 0;
    this.pending.push({ t: 0, every: true, fn: (dt) => {
      t += dt / 0.6;
      const p = start.clone().lerp(end, Math.min(1, t));
      p.y += Math.sin(Math.min(1, t) * Math.PI) * 5;
      m.position.copy(p);
      this.emit(p.x, p.y, p.z, 0xff9933, 1, { speed: 0.5, life: 0.3, size: 0.5 });
      if (t >= 1) { this.scene.remove(m); return true; }
      return false;
    } });
  }

  update(dt) {
    // частинки
    for (let i = 0; i < this.N; i++) {
      if (this.pLife[i] <= 0) { if (this.pAlpha[i] !== 0) { this.pAlpha[i] = 0; this.pSize[i] = 0; } continue; }
      this.pLife[i] -= dt;
      const k = Math.max(0, this.pLife[i] / this.pMax[i]);
      const d = Math.exp(-this.pDrag[i] * dt);
      this.pVel[i * 3] *= d; this.pVel[i * 3 + 1] = this.pVel[i * 3 + 1] * d - this.pGrav[i] * dt * 9.8; this.pVel[i * 3 + 2] *= d;
      this.pPos[i * 3] += this.pVel[i * 3] * dt;
      this.pPos[i * 3 + 1] = Math.max(0.05, this.pPos[i * 3 + 1] + this.pVel[i * 3 + 1] * dt);
      this.pPos[i * 3 + 2] += this.pVel[i * 3 + 2] * dt;
      this.pAlpha[i] = k;
      this.pSize[i] = this.pS0[i] * (0.4 + k * 0.6);
    }
    const ga = this.points.geometry.attributes;
    ga.position.needsUpdate = ga.color.needsUpdate = ga.size.needsUpdate = ga.alpha.needsUpdate = true;

    // снаряди
    for (const [id, p] of this.projectiles) {
      p.ttl -= dt;
      p.m.position.x += p.v.x * dt;
      p.m.position.y += p.v.y * dt - 0.5 * p.g * dt * dt;
      p.m.position.z += p.v.z * dt;
      p.v.y -= p.g * dt;
      if (p.k === 'mortar' || p.k === 'acid' || p.k === 'evil') this.emit(p.m.position.x, p.m.position.y, p.m.position.z, p.c, 1, { speed: 0.6, life: 0.4, size: p.k === 'mortar' ? 1.2 : 0.8 });
      if (p.ttl <= 0) { this.scene.remove(p.m); this.projectiles.delete(id); }
    }
    // промені
    for (let i = this.beams.length - 1; i >= 0; i--) {
      const b = this.beams[i];
      b.life -= dt;
      const k = Math.max(0, b.life / b.max);
      for (const m of b.ms) m.material.opacity = k;
      b.ms[0].scale.x = b.ms[0].scale.y = b.w * (0.5 + k * 0.5);
      if (b.life <= 0) { for (const m of b.ms) { this.scene.remove(m); m.material.dispose(); } this.beams.splice(i, 1); }
    }
    for (let i = this.rings.length - 1; i >= 0; i--) {
      const r = this.rings[i];
      r.life -= dt;
      const k = Math.max(0, r.life / r.max);
      const s = r.r * (1.05 - k * 0.75);
      r.m.scale.set(s, 1, s);
      r.m.material.opacity = k;
      if (r.life <= 0) { this.scene.remove(r.m); r.m.material.dispose(); this.rings.splice(i, 1); }
    }
    for (let i = this.pending.length - 1; i >= 0; i--) {
      const p = this.pending[i];
      if (p.every) { if (p.fn(dt)) this.pending.splice(i, 1); continue; }
      p.t -= dt;
      if (p.t <= 0) { this.pending.splice(i, 1); p.fn(); }
    }
  }
}
export { PCOLORS };
