// Реалістичний світ: HDRI-освітлення, фотоскан-текстури, сезони, погода, лісосмуги, зруйноване місто, укріплена база.
import * as THREE from 'three';
import { EffectComposer } from 'three/addons/postprocessing/EffectComposer.js';
import { RenderPass } from 'three/addons/postprocessing/RenderPass.js';
import { UnrealBloomPass } from 'three/addons/postprocessing/UnrealBloomPass.js';
import { OutputPass } from 'three/addons/postprocessing/OutputPass.js';
import { ShaderPass } from 'three/addons/postprocessing/ShaderPass.js';
import { mergeGeometries } from 'three/addons/utils/BufferGeometryUtils.js';
import { A, texRep } from './assets.js';
import { LANES, DEPOSITS, CAMPS, BASE_R, distToLanes, canBuildAt, GRID, inCity, CITY } from '/shared/data.js';
import { cityBuildings } from '/shared/collision.js';

export const QUALITY = {
  low: { name: 'Низька', shadows: 0, bloom: false, grade: false, pr: 1, particles: 0.5, trees: 0.45, grass: 0, rain: 0.5 },
  medium: { name: 'Середня', shadows: 1024, bloom: false, grade: false, pr: 1, particles: 0.75, trees: 0.7, grass: 200, rain: 0.75 },
  high: { name: 'Висока', shadows: 2048, bloom: true, grade: true, pr: 1, particles: 1, trees: 1, grass: 450, rain: 1 },
  ultra: { name: 'Ультра', shadows: 4096, bloom: true, grade: true, pr: 1.5, particles: 1.2, trees: 1.25, grass: 900, rain: 1.3 },
};

// ───────── шум ─────────
function hash(x, y) { const s = Math.sin(x * 127.1 + y * 311.7) * 43758.5453; return s - Math.floor(s); }
function vnoise(x, y) {
  const xi = Math.floor(x), yi = Math.floor(y), xf = x - xi, yf = y - yi;
  const u = xf * xf * (3 - 2 * xf), v = yf * yf * (3 - 2 * yf);
  const a = hash(xi, yi), b = hash(xi + 1, yi), c = hash(xi, yi + 1), d = hash(xi + 1, yi + 1);
  return a + (b - a) * u + (c - a) * v + (a - b - c + d) * u * v;
}
export function fbm(x, y, o = 4) { let s = 0, a = 0.5, f = 1; for (let i = 0; i < o; i++) { s += vnoise(x * f, y * f) * a; f *= 2; a *= 0.5; } return s; }
function srand(seed) { let s = seed >>> 0 || 1; return () => { s = (s * 16807) % 2147483647; return s / 2147483647; }; }
const smooth = (e0, e1, x) => { const t = Math.max(0, Math.min(1, (x - e0) / (e1 - e0))); return t * t * (3 - 2 * t); };

// ───────── кратери (детерміновані) ─────────
export const CRATERS = [];
{
  const r = srand(777);
  for (let i = 0; i < 70; i++) {
    const a = r() * Math.PI * 2, d = BASE_R + 8 + r() * 85;
    const x = Math.cos(a) * d, z = Math.sin(a) * d;
    const city = inCity(x, z);
    if (r() > 0.35 + city * 0.6 && distToLanes(x, z) > 9) continue;
    CRATERS.push({ x, z, r: 1.5 + r() * 2.8 });
  }
}

export function terrainHeight(x, z) {
  const r = Math.hypot(x, z);
  const flat = smooth(BASE_R + 4, BASE_R + 22, r) * Math.min(1, distToLanes(x, z) / 9);
  let h = (fbm(x * 0.012 + 5, z * 0.012 - 3, 4) - 0.45) * 3.2 * flat;
  h *= 1 - inCity(x, z) * 0.85;
  h += smooth(122, 160, r) * (6 + fbm(x * 0.03, z * 0.03, 3) * 10);
  for (const c of CRATERS) {
    const d = Math.hypot(x - c.x, z - c.z);
    if (d < c.r * 1.3) { const k = d / c.r; h += k < 1 ? -(1 - k * k) * c.r * 0.28 : (1.3 - k) * 0.6; }
  }
  return h;
}

// ───────── спільні уніформи погоди для всіх матеріалів ─────────
export const WXU = { uSnow: { value: 0 }, uWet: { value: 0 }, uAutumn: { value: 0 }, uSpring: { value: 0 }, uDry: { value: 0 } };
const patchedMats = new WeakSet();
export function patchWeather(mat, veg = false) {
  if (patchedMats.has(mat)) return mat;
  patchedMats.add(mat);
  const prev = mat.onBeforeCompile;
  mat.onBeforeCompile = (sh, r) => {
    if (prev) prev(sh, r);
    Object.assign(sh.uniforms, WXU);
    sh.fragmentShader = sh.fragmentShader
      .replace('#include <common>', '#include <common>\nuniform float uSnow, uWet, uAutumn, uSpring, uDry;')
      .replace('#include <normal_fragment_maps>', `#include <normal_fragment_maps>
        {
          vec3 wN = normalize((vec4(normal, 0.0) * viewMatrix).xyz);
          ${veg ? `float lum = dot(diffuseColor.rgb, vec3(0.3, 0.59, 0.11));
          vec3 autumnC = lum * vec3(2.3, 1.25, 0.35);
          diffuseColor.rgb = mix(diffuseColor.rgb, autumnC, uAutumn);
          diffuseColor.rgb *= mix(vec3(1.0), vec3(1.08, 1.15, 0.85), uSpring);
          diffuseColor.rgb = mix(diffuseColor.rgb, lum * vec3(1.35, 1.2, 0.6), uDry * 0.35);` : ''}
          float sCover = uSnow * smoothstep(0.45, 0.8, wN.y);
          diffuseColor.rgb = mix(diffuseColor.rgb, vec3(0.9, 0.92, 0.96), sCover);
          diffuseColor.rgb *= 1.0 - uWet * 0.28 * (1.0 - sCover);
          roughnessFactor = mix(roughnessFactor, roughnessFactor * 0.45, uWet * (1.0 - sCover));
          roughnessFactor = mix(roughnessFactor, 0.75, sCover);
        }`);
  };
  mat.customProgramCacheKey = () => (veg ? 'wxveg' : 'wx') + (mat.alphaTest || 0);
  return mat;
}

function pbr(id, rep, opts = {}) {
  const m = new THREE.MeshStandardMaterial({
    map: texRep(id, 'diff', rep), normalMap: texRep(id, 'nor', rep), roughnessMap: texRep(id, 'rough', rep),
    roughness: 1, metalness: 0, ...opts,
  });
  return patchWeather(m);
}

// Box-UV у метрах для злитої геометрії
function boxUV(geo, scale = 2.5) {
  const p = geo.attributes.position, n = geo.attributes.normal;
  const uv = new Float32Array(p.count * 2);
  for (let i = 0; i < p.count; i++) {
    const nx = Math.abs(n.getX(i)), ny = Math.abs(n.getY(i));
    let u, v;
    if (ny > 0.7) { u = p.getX(i); v = p.getZ(i); } else if (nx > 0.7) { u = p.getZ(i); v = p.getY(i); } else { u = p.getX(i); v = p.getY(i); }
    uv[i * 2] = u / scale; uv[i * 2 + 1] = v / scale;
  }
  geo.setAttribute('uv', new THREE.BufferAttribute(uv, 2));
  return geo;
}
function colorGeo(geo, c) {
  const n = geo.attributes.position.count;
  const col = new Float32Array(n * 3);
  for (let i = 0; i < n; i++) { col[i * 3] = c; col[i * 3 + 1] = c; col[i * 3 + 2] = c; }
  geo.setAttribute('color', new THREE.BufferAttribute(col, 3));
  return geo;
}
function boxAt(w, h, d, x, y, z, ry = 0, rx = 0, rz = 0) {
  const g = new THREE.BoxGeometry(w, h, d);
  const m = new THREE.Matrix4().compose(new THREE.Vector3(x, y, z), new THREE.Quaternion().setFromEuler(new THREE.Euler(rx, ry, rz)), new THREE.Vector3(1, 1, 1));
  return g.applyMatrix4(m);
}

// ───────────────────────── Світ ─────────────────────────
export class World {
  constructor(canvas, quality = 'high', renderer) {
    this.canvas = canvas;
    this.q = QUALITY[quality] || QUALITY.high;
    this.qualityKey = quality;
    const r = (this.renderer = renderer);
    r.setPixelRatio(Math.min(window.devicePixelRatio, this.q.pr));
    r.toneMapping = THREE.ACESFilmicToneMapping;
    r.toneMappingExposure = 0.9;
    r.outputColorSpace = THREE.SRGBColorSpace;
    r.shadowMap.enabled = this.q.shadows > 0;
    r.shadowMap.type = THREE.PCFSoftShadowMap;

    this.scene = new THREE.Scene();
    this.scene.fog = new THREE.FogExp2(0xb8c2cc, 0.004);
    this.camera = new THREE.PerspectiveCamera(42, 1, 0.5, 1200);
    this.camera.position.set(0, 40, 30);
    this.clock = 0;
    this.pmrem = new THREE.PMREMGenerator(r);
    this.envs = {};
    for (const [k, t] of Object.entries(A.hdr)) this.envs[k] = this.pmrem.fromEquirectangular(t).texture;
    this.colliders = []; // прямокутники будівель для руху героя

    this.buildLights();
    this.buildTerrain();
    this.buildBase();
    this.buildCity();
    this.buildTreeLines();
    this.buildProps();
    this.buildDeposits();
    this.buildCamps();
    this.buildPrecip();
    this.buildGridOverlay();
    this.buildPost();
    this.wx = { cur: null, season: 0, kind: 'clear', snow: 0, wet: 0, autumn: 0, spring: 0, dry: 0, rain: 0, snowfall: 0, fog: 0.004, sun: 2.6, hemi: 0.6, env: 0.9, leaves: 1, flash: 0, nextBolt: 5 };
    this.setWeather('clear', 1, true);
    this.resize();
    window.addEventListener('resize', () => this.resize());
  }

  setQuality(key) {
    this.q = QUALITY[key] || QUALITY.high;
    this.qualityKey = key;
    this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, this.q.pr));
    this.renderer.shadowMap.enabled = this.q.shadows > 0;
    if (this.q.shadows) {
      this.sun.shadow.mapSize.set(this.q.shadows, this.q.shadows);
      if (this.sun.shadow.map) { this.sun.shadow.map.dispose(); this.sun.shadow.map = null; }
    }
    if (this.grass) this.grass.count = Math.min(this.grassMax, this.q.grass);
    this.scene.traverse((o) => { if (o.material) { const ms = Array.isArray(o.material) ? o.material : [o.material]; ms.forEach((m) => (m.needsUpdate = true)); } });
    this.resize();
  }

  resize() {
    const w = window.innerWidth, h = window.innerHeight;
    this.renderer.setSize(w, h, false);
    this.camera.aspect = w / h;
    this.camera.updateProjectionMatrix();
    if (this.composer) { this.composer.setPixelRatio(this.renderer.getPixelRatio()); this.composer.setSize(w, h); }
  }

  buildLights() {
    this.hemi = new THREE.HemisphereLight(0xc8d8ff, 0x5a5040, 0.5);
    this.scene.add(this.hemi);
    const s = (this.sun = new THREE.DirectionalLight(0xfff1dd, 2.6));
    s.position.set(-60, 90, 40);
    s.castShadow = this.q.shadows > 0;
    s.shadow.mapSize.set(this.q.shadows || 1024, this.q.shadows || 1024);
    const sc = s.shadow.camera;
    sc.left = -60; sc.right = 60; sc.top = 60; sc.bottom = -60; sc.near = 10; sc.far = 260;
    s.shadow.bias = -0.0005;
    s.shadow.normalBias = 0.05;
    this.scene.add(s, s.target);
    this.flashLights = [];
    for (let i = 0; i < 6; i++) {
      const l = new THREE.PointLight(0xffaa55, 0, 26, 2);
      l.userData.life = 0;
      this.scene.add(l);
      this.flashLights.push(l);
    }
  }
  flash(x, y, z, color, intensity = 60, life = 0.2) {
    y += 1.2; // світло не впритул до поверхні: 1/d² біля нуля дає переповнення
    if (this.noFlash) intensity *= 0.25;
    const l = this.flashLights.find((f) => f.userData.life <= 0) || this.flashLights.reduce((a, b) => (a.userData.life < b.userData.life ? a : b));
    l.position.set(x, y, z);
    l.color.set(color);
    l.userData.life = life; l.userData.max = life; l.userData.I = intensity;
  }

  // ───── місцевість ─────
  buildTerrain() {
    // маска: R дорога, G місто, B база, A рілля
    const N = 512, S = 320;
    const data = new Uint8Array(N * N * 4);
    const fieldCell = 48;
    for (let j = 0; j < N; j++) {
      for (let i = 0; i < N; i++) {
        const x = (i / N - 0.5) * S, z = (j / N - 0.5) * S;
        const dl = distToLanes(x, z);
        let road = 1 - smooth(2.4, 3.6, dl + (hash(Math.floor(x * 2), Math.floor(z * 2)) - 0.5) * 0.4);
        const city = inCity(x, z);
        // вулиці міста — сітка вздовж осі міста
        if (city > 0.2) {
          const ca = CITY.angle;
          const u = x * Math.cos(ca) + z * Math.sin(ca), v = -x * Math.sin(ca) + z * Math.cos(ca);
          const su = Math.abs(((u - 6) % 34 + 34) % 34 - 17), sv = Math.abs(((v + 3) % 30 + 30) % 30 - 15);
          road = Math.max(road, (1 - smooth(13, 15, 17 - Math.min(su, 17))) * 0, (1 - smooth(3, 4.5, Math.min(17 - su, 15 - sv))) * city);
        }
        const r = Math.hypot(x, z);
        const base = 1 - smooth(BASE_R + 2, BASE_R + 6, r);
        // поля: чергування ріллі та трави по клітинках
        const fx = Math.floor((x + 1000) / fieldCell), fz = Math.floor((z + 1000) / fieldCell);
        const plowCell = hash(fx * 3.1, fz * 1.7) > 0.62 ? 1 : 0;
        const edge = Math.min(((x + 1000) % fieldCell), fieldCell - ((x + 1000) % fieldCell), ((z + 1000) % fieldCell), fieldCell - ((z + 1000) % fieldCell));
        const plow = plowCell * smooth(3, 7, edge) * (1 - city) * (1 - base) * smooth(BASE_R + 8, BASE_R + 16, r);
        let crater = 0;
        for (const c of CRATERS) { const d = Math.hypot(x - c.x, z - c.z); if (d < c.r * 1.6) crater = Math.max(crater, 1 - d / (c.r * 1.6)); }
        const k = (j * N + i) * 4;
        data[k] = Math.round(road * 255);
        data[k + 1] = Math.round(Math.max(city, crater * 0.9) * 255);
        data[k + 2] = Math.round(base * 255);
        data[k + 3] = Math.round(plow * 255);
      }
    }
    const mask = new THREE.DataTexture(data, N, N, THREE.RGBAFormat);
    mask.magFilter = THREE.LinearFilter; mask.minFilter = THREE.LinearFilter;
    mask.needsUpdate = true;
    // У masks.y=0 відповідає z=-160 (рядки по z)

    const geo = new THREE.PlaneGeometry(S, S, 240, 240);
    geo.rotateX(-Math.PI / 2);
    const pos = geo.attributes.position;
    for (let i = 0; i < pos.count; i++) pos.setY(i, terrainHeight(pos.getX(i), pos.getZ(i)));
    geo.computeVertexNormals();
    const t = A.tex;
    const mat = new THREE.MeshStandardMaterial({ roughness: 1, metalness: 0, map: t.leafy_grass.diff });
    const U = {
      tGrass: { value: t.leafy_grass.diff }, tGrassN: { value: t.leafy_grass.nor },
      tDry: { value: t.withered_grass.diff }, tMud: { value: t.brown_mud_02.diff }, tMudN: { value: t.brown_mud_02.nor },
      tRoad: { value: t.muddy_tracks.diff }, tSnow: { value: t.snow_02.diff }, tAsph: { value: t.asphalt_02.diff },
      tAsphN: { value: t.asphalt_02.nor }, tRubble: { value: t.rubble.diff }, tMask: { value: mask },
    };
    mat.onBeforeCompile = (sh) => {
      Object.assign(sh.uniforms, U, WXU);
      sh.vertexShader = sh.vertexShader
        .replace('#include <common>', '#include <common>\nvarying vec3 vWP;')
        .replace('#include <begin_vertex>', '#include <begin_vertex>\nvWP = (modelMatrix * vec4(transformed, 1.0)).xyz;');
      sh.fragmentShader = sh.fragmentShader
        .replace('#include <common>', `#include <common>
          varying vec3 vWP;
          uniform sampler2D tGrass, tGrassN, tDry, tMud, tMudN, tRoad, tSnow, tAsph, tAsphN, tRubble, tMask;
          uniform float uSnow, uWet, uAutumn, uSpring, uDry;
          float h2(vec2 p){ return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453); }
          float vn(vec2 p){ vec2 i = floor(p), f = fract(p); f = f*f*(3.0-2.0*f);
            return mix(mix(h2(i), h2(i+vec2(1,0)), f.x), mix(h2(i+vec2(0,1)), h2(i+vec2(1,1)), f.x), f.y); }
          vec3 tri(sampler2D t, vec2 uv){ return mix(texture2D(t, uv).rgb, texture2D(t, uv * 0.31 + vec2(0.37, 0.71)).rgb, 0.42); }
          float gSnow; vec4 gMask;`)
        .replace('#include <map_fragment>', `
          vec2 wuv = vWP.xz;
          gMask = texture2D(tMask, vWP.xz / 320.0 + 0.5);
          float road = gMask.r, city = gMask.g, base = gMask.b, plow = gMask.a;
          float macro = vn(wuv * 0.035) * 0.6 + vn(wuv * 0.13) * 0.4;
          vec3 grass = tri(tGrass, wuv * 0.22);
          vec3 dry = tri(tDry, wuv * 0.2);
          float dryK = clamp(uDry * 0.3 + uAutumn * 0.85 + (macro - 0.5) * 0.7 - uSpring * 0.3, 0.0, 1.0);
          vec3 col = mix(grass, dry, dryK);
          col *= mix(vec3(1.0), vec3(1.04, 1.1, 0.88), uSpring);
          vec3 mud = tri(tMud, wuv * 0.2);
          col = mix(col, mud * vec3(0.92, 0.86, 0.78), plow);
          vec3 asph = tri(tAsph, wuv * 0.18);
          vec3 rub = tri(tRubble, wuv * 0.22);
          vec3 cityC = mix(rub, asph * 0.9, smoothstep(0.45, 0.6, vn(wuv * 0.07)));
          col = mix(col, cityC, city);
          col = mix(col, mud * vec3(1.05, 1.0, 0.92), base * 0.85);
          vec3 roadC = mix(tri(tRoad, wuv * 0.22) * vec3(0.78, 0.74, 0.68), asph, city);
          col = mix(col, roadC, road);
          gSnow = uSnow * smoothstep(0.3, 0.7, uSnow * 1.15 + (macro - 0.5) * 0.9 - road * 0.45 - base * 0.25);
          col = mix(col, tri(tSnow, wuv * 0.2) * 1.05, gSnow);
          col *= 1.0 - uWet * 0.3 * (1.0 - gSnow);
          diffuseColor.rgb = col;`)
        .replace('#include <roughnessmap_fragment>', `float roughnessFactor = mix(0.95, 0.38, uWet * (1.0 - gSnow) * (0.6 + gMask.r * 0.4 + gMask.g * 0.3));
          roughnessFactor = mix(roughnessFactor, 0.7, gSnow);`)
        .replace('#include <normal_fragment_maps>', `
          {
            vec3 tg = texture2D(tGrassN, wuv * 0.22).xyz * 2.0 - 1.0;
            vec3 tm = texture2D(tMudN, wuv * 0.2).xyz * 2.0 - 1.0;
            vec3 ta = texture2D(tAsphN, wuv * 0.18).xyz * 2.0 - 1.0;
            vec3 tn = mix(tg, tm, clamp(gMask.a + gMask.r + gMask.b, 0.0, 1.0));
            tn = mix(tn, ta, gMask.g * 0.8);
            tn = mix(tn, vec3(0.0, 0.0, 1.0), gSnow * 0.7);
            vec3 gw = normalize((vec4(normal, 0.0) * viewMatrix).xyz);
            vec3 wn = normalize(gw * tn.z + vec3(tn.x, 0.0, -tn.y) * 1.1);
            normal = normalize((viewMatrix * vec4(wn, 0.0)).xyz);
          }`);
    };
    mat.customProgramCacheKey = () => 'terrain';
    const ground = (this.ground = new THREE.Mesh(geo, mat));
    ground.receiveShadow = true;
    this.scene.add(ground);
  }

  instancedGeo(geo, mat, list, { shadow = true } = {}) {
    if (!list.length) return null;
    const im = new THREE.InstancedMesh(geo, mat, list.length);
    const m = new THREE.Matrix4(), q = new THREE.Quaternion(), e = new THREE.Euler();
    list.forEach((it, i) => {
      e.set(it.rx || 0, it.ry || 0, it.rz || 0); q.setFromEuler(e);
      m.compose(new THREE.Vector3(it.x, it.y ?? terrainHeight(it.x, it.z), it.z), q, new THREE.Vector3(it.sx ?? it.s ?? 1, it.sy ?? it.s ?? 1, it.sz ?? it.s ?? 1));
      im.setMatrixAt(i, m);
    });
    im.castShadow = shadow; im.receiveShadow = true;
    im.computeBoundingSphere();
    this.scene.add(im);
    return im;
  }
  // ───── інстанси моделей ─────
  instanced(id, list, { veg = false, shadow = true, receive = true, matOverride = null } = {}) {
    const a = A.glb[id];
    if (!a || !list.length) return null;
    const mat = matOverride || patchWeather(a.mat.clone(), veg);
    if (veg) { mat.alphaTest = 0.5; mat.transparent = false; mat.side = THREE.DoubleSide; mat.alphaMap = a.mat.alphaMap; }
    const im = new THREE.InstancedMesh(a.geo, mat, list.length);
    const m = new THREE.Matrix4(), q = new THREE.Quaternion(), e = new THREE.Euler();
    list.forEach((it, i) => {
      e.set(it.rx || 0, it.ry || 0, it.rz || 0);
      q.setFromEuler(e);
      m.compose(new THREE.Vector3(it.x, it.y ?? terrainHeight(it.x, it.z), it.z), q, new THREE.Vector3(it.sx ?? it.s ?? 1, it.sy ?? it.s ?? 1, it.sz ?? it.s ?? 1));
      im.setMatrixAt(i, m);
    });
    im.castShadow = shadow; im.receiveShadow = receive;
    im.computeBoundingSphere();
    this.scene.add(im);
    return im;
  }

  // ───── укріплена база ─────
  buildBase() {
    const bags = [];
    const R = BASE_R + 2.6;
    for (let layer = 0; layer < 2; layer++) {
      const n = Math.floor((2 * Math.PI * R) / 0.72);
      for (let i = 0; i < n; i++) {
        const a = ((i + layer * 0.5) / n) * Math.PI * 2;
        const x = Math.cos(a) * R, z = Math.sin(a) * R;
        if (distToLanes(x, z) < 5) continue;
        bags.push({ x, z, y: terrainHeight(x, z) + layer * 0.17, ry: -a, s: 1, sx: 1.6, sy: 1.25, sz: 1.05 });
        bags.push({ x: Math.cos(a) * (R + 0.5), z: Math.sin(a) * (R + 0.5), y: terrainHeight(x, z) + layer * 0.17, ry: -a + 0.05, sx: 1.6, sy: 1.25, sz: 1.05 });
      }
    }
    this.instanced('cement_bag', bags);
    // бетонні блоки на в'їздах
    const blocks = [];
    for (const lane of LANES) {
      for (let k = 0; k < lane.length - 1; k++) {
        const [ax, az] = lane[k], [bx, bz] = lane[k + 1];
        const ra = Math.hypot(ax, az), rb = Math.hypot(bx, bz);
        if ((ra - R) * (rb - R) > 0) continue;
        const t = (ra - R) / (ra - rb);
        const x = ax + (bx - ax) * t, z = az + (bz - az) * t;
        const dirA = Math.atan2(bz - az, bx - ax);
        for (const side of [-1, 1]) {
          const ox = Math.cos(dirA + Math.PI / 2) * side * 4.2, oz = Math.sin(dirA + Math.PI / 2) * side * 4.2;
          blocks.push({ x: x + ox, z: z + oz, ry: -dirA + Math.PI / 2, s: 1.3 });
          blocks.push({ x: x + ox * 1.35 + Math.cos(dirA) * 2, z: z + oz * 1.35 + Math.sin(dirA) * 2, ry: -dirA + Math.PI / 2 + 0.2, s: 1.3 });
        }
      }
    }
    this.instanced('concrete_road_barrier_02', blocks);
    this.buildHQ();
  }

  buildHQ() {
    const g = (this.core = new THREE.Group());
    const concrete = pbr('concrete_wall_004', 1, { vertexColors: true });
    // заглиблений бліндаж: стіни, перекриття, вхід
    const parts = [
      colorGeo(boxUV(boxAt(8, 1.9, 6, 0, 0.75, 0)), 0.8),
      colorGeo(boxUV(boxAt(8.6, 0.35, 6.6, 0, 1.85, 0)), 0.7),
      colorGeo(boxUV(boxAt(2.2, 1.7, 2.6, 2.6, 0.6, 4.1)), 0.75),
      colorGeo(boxUV(boxAt(0.3, 1.2, 2.6, 1.45, 0.55, 4.1)), 0.7),
      colorGeo(boxUV(boxAt(0.3, 1.2, 2.6, 3.75, 0.55, 4.1)), 0.7),
    ];
    const bunker = new THREE.Mesh(mergeGeometries(parts), concrete);
    bunker.castShadow = bunker.receiveShadow = true;
    g.add(bunker);
    // ґрунт, присипаний на дах
    const soil = pbr('brown_mud_02', 1.5);
    const roof = new THREE.Mesh(boxUV(new THREE.BoxGeometry(7.8, 0.35, 5.8), 2), soil);
    roof.position.y = 2.2; roof.castShadow = roof.receiveShadow = true;
    g.add(roof);
    // маскувальна сітка на стійках
    const netTex = makeNetTexture();
    const netMat = patchWeather(new THREE.MeshStandardMaterial({ map: netTex, alphaTest: 0.45, side: THREE.DoubleSide, roughness: 1 }), true);
    const netGeo = new THREE.PlaneGeometry(13, 10, 18, 14);
    const np = netGeo.attributes.position;
    for (let i = 0; i < np.count; i++) {
      const x = np.getX(i), y = np.getY(i);
      const edge = Math.max(Math.abs(x) / 6.5, Math.abs(y) / 5);
      np.setZ(i, 3.4 - Math.max(0, edge - 0.75) * 9 + Math.sin(x * 1.3 + y * 0.7) * 0.12 - Math.abs(Math.sin(x * 0.9)) * 0.25);
    }
    netGeo.computeVertexNormals();
    netGeo.rotateX(-Math.PI / 2);
    const net = new THREE.Mesh(netGeo, netMat);
    net.castShadow = true; net.receiveShadow = true;
    g.add(net);
    this.netMat = netMat;
    const steel = patchWeather(new THREE.MeshStandardMaterial({ color: 0x3b3e38, metalness: 0.6, roughness: 0.5 }));
    for (const [px, pz] of [[-5, -3.8], [5, -3.8], [-5, 3.8], [5, 3.8], [0, 0]]) {
      const pole = new THREE.Mesh(new THREE.CylinderGeometry(0.05, 0.06, 3.5, 6), steel);
      pole.position.set(px, 1.7, pz);
      g.add(pole);
    }
    // антенна щогла
    const mast = new THREE.Mesh(new THREE.CylinderGeometry(0.05, 0.09, 11, 6), steel);
    mast.position.set(-3, 7.5, -2); mast.castShadow = true;
    g.add(mast);
    const dish = new THREE.Mesh(new THREE.CylinderGeometry(0.5, 0.15, 0.2, 14), steel);
    dish.position.set(-3, 12.6, -2); dish.rotation.x = 0.6;
    g.add(dish);
    const wires = new THREE.BufferGeometry().setFromPoints([
      new THREE.Vector3(-3, 12.5, -2), new THREE.Vector3(-8, 0.1, -6), new THREE.Vector3(-3, 12.5, -2), new THREE.Vector3(2, 0.1, -6.5),
      new THREE.Vector3(-3, 12.5, -2), new THREE.Vector3(-6, 0.1, 3),
    ]);
    g.add(new THREE.LineSegments(wires, new THREE.LineBasicMaterial({ color: 0x2a2a28 })));
    this.scene.add(g);
    // мішки: стінка довкола і ряди на даху
    const bags = [];
    for (let i = 0; i < 52; i++) {
      const a = (i / 52) * Math.PI * 2;
      if (Math.abs(a - Math.PI / 2 + 0.38) < 0.22) continue; // вхід
      for (let l = 0; l < 4; l++) bags.push({ x: Math.cos(a) * 6.2, z: Math.sin(a) * 5.0, y: l * 0.17, ry: -a + (l % 2) * 0.15, sx: 1.6, sy: 1.25, sz: 1.05 });
    }
    for (let i = 0; i < 18; i++) for (const zz of [-2.6, 2.6]) bags.push({ x: -3.6 + i * 0.42, z: zz, y: 2.38, ry: Math.PI / 2, sx: 1.6, sy: 1.25, sz: 1.05 });
    this.instanced('cement_bag', bags);
    const props = [
      ['portable_generator', [{ x: -6.5, z: 4.5, ry: 0.6, s: 1.3 }]],
      ['metal_jerrycan_green', [{ x: -5.6, z: 5.6, ry: 0.2 }, { x: -5.2, z: 5.7, ry: 0.5 }, { x: -5.4, z: 6.1, ry: 1.8 }]],
      ['wooden_military_crate', [{ x: 6.4, z: -4.2, ry: 0.3 }, { x: 6.6, z: -3.1, ry: 0.1 }, { x: 6.5, z: -3.6, y: 0.42, ry: 0.4 }]],
      ['ammo_box', [{ x: 5.2, z: 5.4, ry: 0.4, s: 1.6 }, { x: 5.6, z: 5.6, ry: 1.2, s: 1.6 }]],
      ['barrel_03', [{ x: -7.4, z: -1.5, s: 1 }, { x: -7.6, z: -2.4, s: 1, ry: 1 }]],
    ];
    for (const [id, l] of props) this.instanced(id, l.map((p) => ({ ...p, y: p.y ?? 0 })));
    // дим над штабом при пошкодженні
    this.coreSmoke = 0;
  }
  setCoreHealth(f) { this.coreDamage = 1 - f; }

  // ───── зруйноване місто ─────
  buildCity() {
    const rnd = srand(9001);
    const ca = CITY.angle;
    const plaster = pbr('rough_plaster_broken', 1, { vertexColors: true });
    const brick = pbr('plaster_brick_01', 1, { vertexColors: true });
    const conc = pbr('concrete_wall_004', 1, { vertexColors: true });
    const interior = new THREE.MeshStandardMaterial({ color: 0x15130f, roughness: 1 });
    const geos = { plaster: [], brick: [], conc: [], dark: [] };
    const placed = [];
    // розміщення спільне з сервером (shared/collision.js): стіни на екрані = стіни в симуляції
    for (const h of cityBuildings()) {
      const brnd = srand(h.seed);
      const b = makeRuin(h.w, h.d, h.floors, brnd, h.big);
      const m = new THREE.Matrix4().compose(new THREE.Vector3(h.x, terrainHeight(h.x, h.z) - 0.1, h.z), new THREE.Quaternion().setFromEuler(new THREE.Euler(0, h.rot, 0)), new THREE.Vector3(1, 1, 1));
      for (const gg of b.walls) geos[h.kind].push(gg.applyMatrix4(m));
      for (const gg of b.slabs) geos.conc.push(gg.applyMatrix4(m));
      for (const gg of b.dark) geos.dark.push(gg.applyMatrix4(m));
      placed.push({ x: h.x, z: h.z, rad: Math.hypot(h.w, h.d) / 2 + 1.5, rubble: b.rubble.map(([lx, lz, sz]) => { const v = new THREE.Vector3(lx, 0, lz).applyMatrix4(m); return [v.x, v.z, sz]; }) });
      this.colliders.push(h);
    }
    const add = (list, mat) => { if (!list.length) return; const me = new THREE.Mesh(mergeGeometries(list), mat); me.castShadow = me.receiveShadow = true; this.scene.add(me); };
    add(geos.plaster.map((g) => boxUV(g, 3)), plaster);
    add(geos.brick.map((g) => boxUV(g, 2)), brick);
    add(geos.conc.map((g) => boxUV(g, 3)), conc);
    add(geos.dark, interior);
    // купи уламків
    const rubbleMat = pbr('rubble', 1);
    const chunks = [];
    const piles = [];
    for (const p of placed) {
      for (const [x, z, s] of p.rubble) {
        const pg = new THREE.SphereGeometry(s, 10, 6, 0, Math.PI * 2, 0, Math.PI / 2);
        const pp = pg.attributes.position;
        for (let i = 0; i < pp.count; i++) { const k = 0.75 + hash(i * 1.3, s) * 0.5; pp.setX(i, pp.getX(i) * k); pp.setZ(i, pp.getZ(i) * k); pp.setY(i, pp.getY(i) * 0.45 * k); }
        pg.computeVertexNormals();
        pg.translate(x, terrainHeight(x, z) - 0.1, z);
        piles.push(boxUV(pg, 2));
        for (let i = 0; i < 6; i++) chunks.push({ x: x + (rnd() - 0.5) * s * 2, z: z + (rnd() - 0.5) * s * 2, s: 0.4 + rnd() * 0.9, rx: rnd() * 3, ry: rnd() * 3, rz: rnd() * 3 });
      }
    }
    if (piles.length) { const me = new THREE.Mesh(mergeGeometries(piles), rubbleMat); me.castShadow = me.receiveShadow = true; this.scene.add(me); }
    const chunkGeo = new THREE.BoxGeometry(1, 0.35, 0.8);
    const chunkMesh = new THREE.InstancedMesh(boxUV(chunkGeo, 1.5), conc, chunks.length);
    const mm = new THREE.Matrix4();
    chunks.forEach((c, i) => { mm.compose(new THREE.Vector3(c.x, terrainHeight(c.x, c.z) + 0.1, c.z), new THREE.Quaternion().setFromEuler(new THREE.Euler(c.rx, c.ry, c.rz)), new THREE.Vector3(c.s, c.s, c.s)); chunkMesh.setMatrixAt(i, mm); });
    chunkGeo.setAttribute('color', new THREE.BufferAttribute(new Float32Array(chunkGeo.attributes.position.count * 3).fill(0.8), 3));
    chunkMesh.castShadow = chunkMesh.receiveShadow = true;
    this.scene.add(chunkMesh);
    // вигорілі авто, блоки, ліхтарі, шини
    const burnt = patchWeather(new THREE.MeshStandardMaterial({ map: texRep('rusty_metal_02', 'diff', 2), normalMap: texRep('rusty_metal_02', 'nor', 2), color: 0x4a3c34, roughness: 0.9, metalness: 0.4 }));
    const cars = [], lamps = [], tyres = [], blk = [];
    for (let i = 0; i < 160 && cars.length < 14; i++) {
      const rr = CITY.rMin + 4 + rnd() * (CITY.rMax - CITY.rMin - 6), aa = ca + (rnd() - 0.5) * CITY.spread * 1.8;
      const x = Math.cos(aa) * rr, z = Math.sin(aa) * rr;
      if (inCity(x, z) < 0.5 || distToLanes(x, z) < 5 || placed.some((p) => Math.hypot(p.x - x, p.z - z) < p.rad + 2)) continue;
      cars.push({ x, z, ry: rnd() * 6.28, rz: (rnd() - 0.5) * 0.1, s: 1, y: terrainHeight(x, z) + 0.28 });
      if (rnd() < 0.6) lamps.push({ x: x + 3, z: z + 2, ry: rnd() * 6, rz: (rnd() - 0.5) * 0.4, s: 1.6 });
      for (let k = 0; k < 2; k++) tyres.push({ x: x + (rnd() - 0.5) * 6, z: z + (rnd() - 0.5) * 6, rx: Math.PI / 2, rz: rnd() * 6, y: terrainHeight(x, z) + 0.08, s: 1.2 });
      if (rnd() < 0.5) blk.push({ x: x - 3, z: z + 3, ry: rnd() * 6, s: 1.3 });
    }
    this.instanced('covered_car', cars, { matOverride: burnt });
    this.instanced('street_lamp_01', lamps);
    this.instanced('old_tyre', tyres);
    this.instanced('concrete_road_barrier', blk);
  }

  // ───── лісосмуги ─────
  buildTreeLines() {
    const rnd = srand(31337);
    const trees = [];
    const spacing = 5.2 / Math.sqrt(this.q.trees);
    const ok = (x, z, pad = 0) => {
      const r = Math.hypot(x, z);
      if (r < BASE_R + 12 || r > 158) return false;
      if (inCity(x, z) > 0.25) return false;
      if (distToLanes(x, z) < 6.5 + pad) return false;
      for (const c of CAMPS) if (Math.hypot(c.x - x, c.z - z) < 10) return false;
      for (const d of DEPOSITS) if (Math.hypot(d.x - x, d.z - z) < 11) return false;
      return true;
    };
    const cell = 48;
    for (let k = -4; k <= 4; k++) {
      for (const axis of [0, 1]) {
        const c0 = k * cell - 1000 % cell + (axis ? 4 : -2);
        for (let t = -165; t < 165; t += spacing) {
          const along = t + (rnd() - 0.5) * 1.5;
          const across = c0 + (rnd() - 0.5) * 2.2;
          const x = axis ? along : across, z = axis ? across : along;
          if (ok(x, z) && rnd() > 0.06) trees.push({ x, z, s: 0.8 + rnd() * 0.55, ry: rnd() * 6.28, kind: rnd() });
        }
      }
    }
    // ліс по краю карти
    for (let i = 0; i < 520 * this.q.trees; i++) {
      const a = rnd() * Math.PI * 2, r = 128 + rnd() * 30;
      const x = Math.cos(a) * r, z = Math.sin(a) * r;
      if (ok(x, z, 2)) trees.push({ x, z, s: 0.9 + rnd() * 0.6, ry: rnd() * 6.28, kind: rnd() });
    }
    this.treeCount = trees.length;
    // стовбури
    const bark = pbr('bark_brown_02', 1);
    bark.map.repeat.set(1, 3); bark.normalMap.repeat.set(1, 3); bark.roughnessMap.repeat.set(1, 3);
    const trunkGeo = new THREE.CylinderGeometry(0.12, 0.22, 5, 7, 1);
    trunkGeo.translate(0, 2.5, 0);
    const branchGeo = new THREE.CylinderGeometry(0.04, 0.1, 2.6, 5);
    branchGeo.translate(0, 1.3, 0);
    const trunks = new THREE.InstancedMesh(trunkGeo, bark, trees.length);
    const branches = new THREE.InstancedMesh(branchGeo, bark, trees.length * 4);
    const crowns = [];
    const m = new THREE.Matrix4(), q = new THREE.Quaternion(), e = new THREE.Euler(), v = new THREE.Vector3(), sc = new THREE.Vector3();
    trees.forEach((tr, i) => {
      const y = terrainHeight(tr.x, tr.z);
      const h = 0.9 + tr.kind * 0.5;
      e.set((rnd() - 0.5) * 0.08, tr.ry, (rnd() - 0.5) * 0.08); q.setFromEuler(e);
      m.compose(v.set(tr.x, y - 0.2, tr.z), q, sc.set(tr.s, tr.s * h, tr.s));
      trunks.setMatrixAt(i, m);
      for (let b = 0; b < 4; b++) {
        e.set(0.7 + rnd() * 0.4, tr.ry + b * 1.57 + rnd() * 0.5, 0, 'YXZ'); q.setFromEuler(e);
        m.compose(v.set(tr.x, y + 3.2 * tr.s * h + b * 0.4, tr.z), q, sc.set(tr.s, tr.s, tr.s));
        branches.setMatrixAt(i * 4 + b, m);
      }
      const cs = tr.s * (1.3 + rnd() * 0.5);
      crowns.push({ x: tr.x, z: tr.z, y: y + 3.6 * tr.s * h, ry: rnd() * 6.28, sx: cs, sy: cs * (1.1 + rnd() * 0.4), sz: cs });
      if (tr.kind > 0.55) crowns.push({ x: tr.x + (rnd() - 0.5) * 1.6, z: tr.z + (rnd() - 0.5) * 1.6, y: y + 4.8 * tr.s * h, ry: rnd() * 6.28, s: cs * 0.75 });
    });
    trunks.castShadow = branches.castShadow = true;
    trunks.receiveShadow = true;
    this.scene.add(trunks, branches);
    const foliage = foliageMaterial();
    this.crowns = this.instancedGeo(crownGeometry(26, 1), foliage, crowns.map((c) => ({ ...c, sx: (c.sx ?? c.s) * 1.6, sy: (c.sy ?? c.s) * 1.3, sz: (c.sz ?? c.s) * 1.6, y: c.y + 0.8 })));
    this.crownCount = crowns.length;
    // кущі, трава, пні, колоди, каміння
    const shrubs = [], grass = [], stumps = [], logs = [], rocks = [], stones = [];
    for (const tr of trees) {
      if (rnd() < 0.5) shrubs.push({ x: tr.x + (rnd() - 0.5) * 4, z: tr.z + (rnd() - 0.5) * 4, ry: rnd() * 6.28, s: 2.2 + rnd() * 1.8 });
      if (rnd() < 0.05) stumps.push({ x: tr.x + 2, z: tr.z + 1, ry: rnd() * 6, s: 0.8 + rnd() * 0.4 });
      if (rnd() < 0.04) logs.push({ x: tr.x + 3, z: tr.z - 2, ry: rnd() * 6, s: 1 + rnd() * 0.4 });
    }
    for (let i = 0; i < 1600; i++) {
      const x = (rnd() - 0.5) * 300, z = (rnd() - 0.5) * 300;
      if (!ok(x, z, -2) && !(Math.hypot(x, z) > BASE_R + 6 && Math.hypot(x, z) < 160 && distToLanes(x, z) > 4 && inCity(x, z) < 0.2)) continue;
      grass.push({ x, z, ry: rnd() * 6.28, s: 3 + rnd() * 3 });
      if (rnd() < 0.08) rocks.push({ x: x + 3, z: z + 3, ry: rnd() * 6.28, rx: rnd() * 0.5, s: 5 + rnd() * 8 });
      if (rnd() < 0.08) stones.push({ x: x - 2, z: z + 2, ry: rnd() * 6.28, s: 8 + rnd() * 14 });
    }
    this.instancedGeo(crownGeometry(12, 2), foliage, shrubs.map((b) => ({ ...b, s: b.s * 0.45, y: terrainHeight(b.x, b.z) + b.s * 0.25 })));
    // трав'яні купини — маленькі низькі пластини
    this.grassMax = grass.length;
    this.grass = this.instancedGeo(crownGeometry(5, 3, true), foliage, grass.map((g) => ({ ...g, s: 0.5, sy: 0.35, sx: 0.6, sz: 0.6, y: terrainHeight(g.x, g.z) + 0.15 })), { shadow: false });
    if (this.grass) this.grass.count = Math.min(grass.length, this.q.grass);
    this.instanced('tree_stump_01', stumps);
    this.instanced('dead_tree_trunk_02', logs);
    this.instanced('rock_07', rocks);
    this.instanced('stone_01', stones);
  }

  buildProps() {
    // уздовж доріг: опори ЛЕП
    const rnd = srand(55);
    const poleMat = pbr('bark_brown_02', 1);
    const poles = [];
    const lane = LANES[2];
    for (let i = 1; i < lane.length - 1; i++) {
      const [ax, az] = lane[i - 1], [bx, bz] = lane[i];
      const len = Math.hypot(bx - ax, bz - az);
      for (let t = 0; t < len; t += 18) {
        const k = t / len;
        const dx = (bx - ax) / len, dz = (bz - az) / len;
        const x = ax + (bx - ax) * k - dz * 7, z = az + (bz - az) * k + dx * 7;
        if (Math.hypot(x, z) < BASE_R + 6) continue;
        poles.push([x, terrainHeight(x, z), z, (rnd() - 0.5) * 0.15]);
      }
    }
    const pg = [];
    for (const [x, y, z, tilt] of poles) {
      pg.push(new THREE.CylinderGeometry(0.13, 0.17, 8, 7).translate(0, 4, 0).applyMatrix4(new THREE.Matrix4().makeRotationZ(tilt)).translate(x, y, z));
      pg.push(new THREE.BoxGeometry(2.2, 0.14, 0.14).translate(0, 7.4, 0).applyMatrix4(new THREE.Matrix4().makeRotationZ(tilt)).translate(x, y, z));
    }
    if (pg.length) { const me = new THREE.Mesh(mergeGeometries(pg.map((g) => boxUV(g, 1.5))), poleMat); me.castShadow = true; this.scene.add(me); }
    // провиси дротів
    const pts = [];
    for (let i = 0; i < poles.length - 1; i++) {
      const [x1, y1, z1] = poles[i], [x2, y2, z2] = poles[i + 1];
      if (Math.hypot(x2 - x1, z2 - z1) > 30) continue;
      for (const off of [-1, 1]) {
        for (let s = 0; s < 8; s++) {
          const a = s / 8, b = (s + 1) / 8;
          const sag = (t) => -Math.sin(t * Math.PI) * 0.9;
          pts.push(new THREE.Vector3(x1 + (x2 - x1) * a + off * 0.9, y1 + 7.45 + (y2 - y1) * a + sag(a), z1 + (z2 - z1) * a));
          pts.push(new THREE.Vector3(x1 + (x2 - x1) * b + off * 0.9, y1 + 7.45 + (y2 - y1) * b + sag(b), z1 + (z2 - z1) * b));
        }
      }
    }
    this.scene.add(new THREE.LineSegments(new THREE.BufferGeometry().setFromPoints(pts), new THREE.LineBasicMaterial({ color: 0x1b1b1b })));
    // спалена техніка біля доріг
    const burnt = patchWeather(new THREE.MeshStandardMaterial({ map: texRep('rusty_metal_02', 'diff', 2), color: 0x3d342e, roughness: 0.9, metalness: 0.4 }));
    const wrecks = [];
    for (const l of LANES) {
      for (const k of [1, 3]) {
        const [x, z] = l[k];
        wrecks.push({ x: x + 6, z: z + 4, ry: rnd() * 6, rz: 0.08, y: terrainHeight(x + 6, z + 4) + 0.25 });
      }
    }
    this.instanced('covered_car', wrecks, { matOverride: burnt });
  }

  // ───── майстерні ─────
  buildDeposits() {
    this.depositObjs = [];
    const brick = pbr('plaster_brick_01', 1, { vertexColors: true });
    const conc = pbr('concrete_wall_004', 1, { vertexColors: true });
    for (const d of DEPOSITS) {
      const g = new THREE.Group();
      const y = terrainHeight(d.x, d.z);
      g.position.set(d.x, y, d.z);
      g.rotation.y = Math.atan2(d.x, d.z);
      const rnd = srand(d.id * 977 + 3);
      // руїна складу
      const walls = [];
      const W = 12, D = 9;
      const floor = new THREE.Mesh(boxUV(boxAt(W, 0.3, D, 0, 0.15, 0), 3), pbr('concrete_wall_004', 1));
      floor.receiveShadow = true;
      g.add(floor);
      const wallH = (x) => 2.5 + Math.sin(x * 0.9 + d.id) * 1.2 + rnd() * 0.6;
      for (let i = 0; i < 6; i++) {
        const x = -W / 2 + 1 + i * 2;
        walls.push(colorGeo(boxAt(2, wallH(x), 0.35, x, wallH(x) / 2, -D / 2), 0.7 + rnd() * 0.3));
        if (i % 2 === 0) walls.push(colorGeo(boxAt(2, wallH(x + 7) * 0.6, 0.35, x, wallH(x + 7) * 0.3, D / 2), 0.7));
      }
      walls.push(colorGeo(boxAt(0.35, 3.2, D, -W / 2, 1.6, 0), 0.8));
      const wm = new THREE.Mesh(mergeGeometries(walls.map((gg) => boxUV(gg, 2))), brick);
      wm.castShadow = wm.receiveShadow = true;
      g.add(wm);
      void conc;
      // обладнання майстерні (видиме після будівництва)
      const rig = new THREE.Group();
      const addProp = (id, x, z, ry = 0, s = 1, yy = 0) => {
        const a = A.glb[id];
        if (!a) return;
        const me = new THREE.Mesh(a.geo, patchWeather(a.mat));
        me.position.set(x, yy, z); me.rotation.y = ry; me.scale.setScalar(s);
        me.castShadow = me.receiveShadow = true;
        rig.add(me);
      };
      addProp('portable_generator', -2, -2.5, 0.4, 1.4);
      addProp('wooden_military_crate', 2, -2.8, 0.1);
      addProp('wooden_military_crate', 2.2, -1.8, -0.2);
      addProp('old_military_crate', 3.5, 1.5, 0.6, 1.2);
      addProp('barrel_03', -4, 2.5);
      addProp('barrel_03', -3.3, 3, 0, 1);
      addProp('metal_jerrycan_green', -1.5, 3, 0.4);
      addProp('ammo_box', 0.5, 2.5, 0.9, 1.6);
      // тент
      const tarp = new THREE.Mesh(new THREE.PlaneGeometry(7, 5, 6, 4), patchWeather(new THREE.MeshStandardMaterial({ color: 0x4d5536, roughness: 0.95, side: THREE.DoubleSide, normalMap: texRep('denim_fabric', 'nor', 4) })));
      const tp = tarp.geometry.attributes.position;
      for (let i = 0; i < tp.count; i++) tp.setZ(i, -Math.abs(tp.getX(i)) * 0.3 + Math.sin(i) * 0.05);
      tarp.geometry.computeVertexNormals();
      tarp.rotation.x = -Math.PI / 2 + 0.05; tarp.position.set(0.5, 3.0, -0.5);
      tarp.castShadow = true;
      rig.add(tarp);
      rig.visible = false;
      g.add(rig);
      this.scene.add(g);
      this.depositObjs.push({ g, rig, def: d, level: 0 });
    }
  }
  setExtractors(list) {
    list.forEach((ex, i) => {
      const o = this.depositObjs[i];
      if (!o) return;
      o.rig.visible = ex.level > 0;
      o.level = ex.level;
    });
  }

  // ───── позиції ДРГ ─────
  buildCamps() {
    this.campObjs = [];
    const bags = [], crates = [], logs = [];
    for (const c of CAMPS) {
      const rnd = srand(c.id * 131 + 7);
      // кілька дуг окопів з мішків
      const arcs = 3 + Math.min(2, c.tier);
      for (let k = 0; k < arcs; k++) {
        const a0 = (k / arcs) * Math.PI * 2 + rnd() * 0.6, span = 0.7 + rnd() * 0.4;
        const r = 6.5 + rnd() * 2;
        const n = Math.floor((span * r) / 0.68);
        for (let i = 0; i < n; i++) {
          const a = a0 + (i / n) * span;
          const x = c.x + Math.cos(a) * r, z = c.z + Math.sin(a) * r;
          const y = terrainHeight(x, z);
          for (let l = 0; l < 2 + (c.tier >= 2 ? 1 : 0); l++) bags.push({ x, z, y: y + l * 0.17, ry: -a + (l % 2) * 0.12, sx: 1.6, sy: 1.25, sz: 1.05 });
        }
      }
      for (let i = 0; i < 2 + c.tier; i++) crates.push({ x: c.x + (rnd() - 0.5) * 6, z: c.z + (rnd() - 0.5) * 6, ry: rnd() * 6 });
      if (c.tier >= 1) logs.push({ x: c.x + 3, z: c.z - 3, ry: rnd() * 6, s: 0.9 });
      this.campObjs.push({ def: c });
    }
    this.instanced('cement_bag', bags);
    this.instanced('old_military_crate', crates);
    this.instanced('dead_tree_trunk_02', logs);
  }

  // ───── опади ─────
  buildPrecip() {
    const N = Math.floor(7000 * this.q.rain);
    const rg = new THREE.BufferGeometry();
    const rp = new Float32Array(N * 6);
    for (let i = 0; i < N; i++) {
      const x = (Math.random() - 0.5) * 120, y = Math.random() * 60, z = (Math.random() - 0.5) * 120;
      rp.set([x, y, z, x + 0.15, y - 1.1, z], i * 6);
    }
    rg.setAttribute('position', new THREE.BufferAttribute(rp, 3));
    this.rain = new THREE.LineSegments(rg, new THREE.LineBasicMaterial({ color: 0xaab4c0, transparent: true, opacity: 0.35, depthWrite: false, fog: false }));
    this.rain.frustumCulled = false;
    this.rain.visible = false;
    this.scene.add(this.rain);
    const sg = new THREE.BufferGeometry();
    const sp = new Float32Array(N * 3);
    for (let i = 0; i < N; i++) sp.set([(Math.random() - 0.5) * 120, Math.random() * 50, (Math.random() - 0.5) * 120], i * 3);
    sg.setAttribute('position', new THREE.BufferAttribute(sp, 3));
    const flake = document.createElement('canvas');
    flake.width = flake.height = 32;
    const fc = flake.getContext('2d');
    const gr = fc.createRadialGradient(16, 16, 0, 16, 16, 16);
    gr.addColorStop(0, 'rgba(255,255,255,1)'); gr.addColorStop(1, 'rgba(255,255,255,0)');
    fc.fillStyle = gr; fc.fillRect(0, 0, 32, 32);
    this.snowfall = new THREE.Points(sg, new THREE.PointsMaterial({ size: 0.28, map: new THREE.CanvasTexture(flake), transparent: true, depthWrite: false, opacity: 0.9 }));
    this.snowfall.frustumCulled = false;
    this.snowfall.visible = false;
    this.scene.add(this.snowfall);
  }

  buildGridOverlay() {
    const cells = [];
    const M = Math.floor(BASE_R / GRID) * GRID;
    for (let x = -M; x <= M; x += GRID) for (let z = -M; z <= M; z += GRID) if (canBuildAt(x, z)) cells.push([x, z]);
    const geo = new THREE.PlaneGeometry(GRID * 0.9, GRID * 0.9);
    geo.rotateX(-Math.PI / 2);
    const mat = new THREE.MeshBasicMaterial({ color: 0xffffff, transparent: true, opacity: 0.08, depthWrite: false });
    const im = new THREE.InstancedMesh(geo, mat, cells.length);
    const m = new THREE.Matrix4();
    cells.forEach(([x, z], i) => { m.makeTranslation(x, terrainHeight(x, z) + 0.08, z); im.setMatrixAt(i, m); });
    im.visible = false;
    this.gridOverlay = im;
    this.scene.add(im);
  }

  buildPost() {
    const size = this.renderer.getDrawingBufferSize(new THREE.Vector2());
    const rt = new THREE.WebGLRenderTarget(size.x, size.y, { type: THREE.HalfFloatType, samples: 4 });
    this.composer = new EffectComposer(this.renderer, rt);
    this.composer.addPass(new RenderPass(this.scene, this.camera));
    // санітарний прохід: Inf/NaN (переповнення half-float на справжніх GPU) не повинні потрапити в bloom
    this.composer.addPass(new ShaderPass({
      uniforms: { tDiffuse: { value: null } },
      vertexShader: `varying vec2 vUv; void main(){ vUv = uv; gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0); }`,
      fragmentShader: `uniform sampler2D tDiffuse; varying vec2 vUv;
        void main(){ vec4 c = texture2D(tDiffuse, vUv);
          bvec3 bad = bvec3(c.r != c.r || c.r > 6.0e4, c.g != c.g || c.g > 6.0e4, c.b != c.b || c.b > 6.0e4);
          vec3 col = any(bad) ? vec3(0.0) : clamp(c.rgb, 0.0, 48.0);
          gl_FragColor = vec4(col, 1.0); }`,
    }));
    this.bloom = new UnrealBloomPass(new THREE.Vector2(size.x, size.y), 0.35, 0.4, 0.95);
    this.composer.addPass(this.bloom);
    // кольорокорекція «плівки» в лінійному просторі, перед OutputPass (тонмапінг + sRGB)
    this.grade = new ShaderPass({
      uniforms: { tDiffuse: { value: null }, sat: { value: 1 }, contrast: { value: 1.04 }, tint: { value: new THREE.Vector3(1, 1, 1) }, vignette: { value: 0.32 }, grain: { value: 0.035 }, time: { value: 0 } },
      vertexShader: `varying vec2 vUv; void main(){ vUv = uv; gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0); }`,
      fragmentShader: `uniform sampler2D tDiffuse; uniform float sat, contrast, vignette, grain, time; uniform vec3 tint; varying vec2 vUv;
        float h(vec2 p){ return fract(sin(dot(p, vec2(12.9898, 78.233)) + time * 61.7) * 43758.5453); }
        void main(){
          vec4 c = texture2D(tDiffuse, vUv);
          float l = dot(c.rgb, vec3(0.2126, 0.7152, 0.0722));
          vec3 col = mix(vec3(l), c.rgb, sat) * tint;
          // контраст довкола середньо-сірого в лог-просторі (стабільно для HDR)
          col = exp2((log2(max(col, 1e-5)) - log2(0.18)) * contrast + log2(0.18));
          vec2 d = vUv - 0.5; col *= 1.0 - vignette * smoothstep(0.25, 0.85, dot(d, d) * 2.2);
          col *= 1.0 + (h(vUv * 931.0) - 0.5) * grain;
          gl_FragColor = vec4(col, c.a);
        }`,
    });
    this.composer.addPass(this.grade);
    this.composer.addPass(new OutputPass());
  }

  // ───── погода ─────
  setWeather(kind, season, instant = false) {
    const P = {
      clear: { gSat: 1.06, gTint: [1.03, 1.0, 0.95], env: 'clear', sun: 2.2, hemi: 0.45, envI: 0.75, fog: 0.0028, fogC: 0xb9c6d2, exp: 0.8, rain: 0, snow: 0, wet: 0 },
      cloudy: { gSat: 0.88, gTint: [0.99, 1.0, 1.02], env: 'overcast', sun: 1.0, hemi: 0.6, envI: 0.9, fog: 0.0042, fogC: 0x9ea6ad, exp: 0.95, rain: 0, snow: 0, wet: 0 },
      rain: { gSat: 0.8, gTint: [0.96, 0.99, 1.04], env: 'overcast', sun: 0.5, hemi: 0.7, envI: 0.85, fog: 0.0075, fogC: 0x8b9298, exp: 1.0, rain: 1, snow: 0, wet: 1 },
      storm: { gSat: 0.72, gTint: [0.94, 0.98, 1.05], env: 'overcast', sun: 0.3, hemi: 0.55, envI: 0.6, fog: 0.009, fogC: 0x6c7379, exp: 0.95, rain: 1.6, snow: 0, wet: 1 },
      fog: { gSat: 0.8, gTint: [1.0, 1.0, 1.0], env: 'misty', sun: 0.7, hemi: 0.8, envI: 1.0, fog: 0.019, fogC: 0xb0b4b6, exp: 1.05, rain: 0, snow: 0, wet: 0.4 },
      snow: { gSat: 0.85, gTint: [0.95, 0.99, 1.06], env: 'winter', sun: 0.8, hemi: 0.8, envI: 1.0, fog: 0.0085, fogC: 0xc3c9d0, exp: 0.95, rain: 0, snow: 1, wet: 0 },
      blizzard: { gSat: 0.75, gTint: [0.94, 0.99, 1.08], env: 'winter', sun: 0.35, hemi: 0.75, envI: 0.85, fog: 0.022, fogC: 0xc9ced4, exp: 0.95, rain: 0, snow: 2.2, wet: 0 },
    }[kind] || {};
    if (season === 3 && kind === 'clear') P.env = 'winter';
    this.wx.kind = kind; this.wx.season = season; this.wx.target = P;
    this.wx.tSnowCover = season === 3 ? 1 : 0;
    this.wx.tAutumn = season === 2 ? 1 : 0;
    this.wx.tSpring = season === 0 ? 1 : 0;
    this.wx.tDry = season === 1 && (kind === 'clear' || kind === 'cloudy') ? 1 : 0;
    this.wx.tLeaves = season === 3 ? 0 : 1;
    if (P.env && this.wx.envKey !== P.env && this.envs[P.env]) {
      this.wx.envKey = P.env;
      this.scene.environment = this.envs[P.env];
      this.scene.background = A.hdr[P.env];
    }
    if (instant) {
      Object.assign(this.wx, { sun: P.sun, hemi: P.hemi, envI: P.envI, fog: P.fog, exp: P.exp, rainA: P.rain, snowA: P.snow, wet: P.wet,
        snow: this.wx.tSnowCover, autumn: this.wx.tAutumn, spring: this.wx.tSpring, dry: this.wx.tDry, leaves: this.wx.tLeaves });
      this.wx.fogC = new THREE.Color(P.fogC);
    }
  }

  updateWeather(dt, focus) {
    const w = this.wx, P = w.target;
    if (!P) return;
    const k = 1 - Math.exp(-dt * 0.35);
    const ks = 1 - Math.exp(-dt * 0.08);
    w.sun += (P.sun - w.sun) * k; w.hemi += (P.hemi - w.hemi) * k; w.envI += (P.envI - w.envI) * k;
    w.fog += (P.fog - w.fog) * k; w.exp += (P.exp - w.exp) * k;
    w.rainA += (P.rain - w.rainA) * k * 2; w.snowA += (P.snow - w.snowA) * k * 2;
    w.wet += ((P.wet > w.wet ? P.wet : P.wet) - w.wet) * (P.wet > w.wet ? k : k * 0.3);
    w.fogC.lerp(new THREE.Color(P.fogC), k);
    // покрив снігу наростає під час снігопаду або взимку
    const snowTarget = Math.max(w.tSnowCover * (w.snowA > 0.2 ? 1 : 0.85), 0);
    w.snow += (snowTarget - w.snow) * ks;
    w.autumn += (w.tAutumn - w.autumn) * ks; w.spring += (w.tSpring - w.spring) * ks; w.dry += (w.tDry - w.dry) * ks;
    w.leaves += (w.tLeaves - w.leaves) * ks;
    WXU.uSnow.value = w.snow; WXU.uWet.value = w.wet; WXU.uAutumn.value = w.autumn; WXU.uSpring.value = w.spring; WXU.uDry.value = w.dry;
    // блискавки
    let bolt = 0;
    if (w.kind === 'storm') {
      w.nextBolt -= dt;
      if (w.nextBolt <= 0) { w.nextBolt = 6 + Math.random() * 10; w.flash = 1; this.onThunder && this.onThunder(); }
    }
    if (w.flash > 0) { w.flash -= dt * 3; bolt = Math.max(0, w.flash) * (this.noFlash ? 0.15 : Math.random() > 0.3 ? 1 : 0.2); }
    this.sun.intensity = w.sun;
    this.hemi.intensity = w.hemi + bolt * 3;
    this.scene.environmentIntensity = w.envI + bolt * 2;
    this.scene.backgroundIntensity = w.envI * 0.9 + bolt;
    this.scene.fog.density = w.fog;
    this.scene.fog.color.copy(w.fogC);
    this.renderer.toneMappingExposure = w.exp;
    if (this.grade) {
      const U = this.grade.uniforms, kk = 1 - Math.exp(-dt * 0.5);
      U.sat.value += ((P.gSat ?? 1) * (w.autumn > 0.5 ? 1.05 : 1) - U.sat.value) * kk;
      const tt = P.gTint || [1, 1, 1];
      U.tint.value.x += (tt[0] - U.tint.value.x) * kk; U.tint.value.y += (tt[1] - U.tint.value.y) * kk; U.tint.value.z += (tt[2] - U.tint.value.z) * kk;
      U.time.value = this.clock % 100;
    }
    // крони дерев: листя взимку зникає
    if (this.crowns) {
      this.crowns.visible = w.leaves > 0.05;
      this.crowns.count = Math.floor(this.crownCount * Math.min(1, w.leaves * 1.1));
    }
    if (this.grass) this.grass.visible = w.snow < 0.6;
    // опади слідують за камерою
    const cx = focus ? focus.x : 0, cz = focus ? focus.z : 0;
    this.rain.visible = w.rainA > 0.02;
    if (this.rain.visible) {
      const p = this.rain.geometry.attributes.position, arr = p.array;
      const n = arr.length / 6, active = Math.floor(n * Math.min(1, w.rainA / 1.6));
      const sp = 38 * dt;
      for (let i = 0; i < n; i++) {
        const o = i * 6;
        if (i >= active) { arr[o + 1] = -100; arr[o + 4] = -100; continue; }
        let y = arr[o + 1] - sp;
        let x = arr[o], z = arr[o + 2];
        if (y < -1 || y < -50 || Math.abs(x - cx) > 60 || Math.abs(z - cz) > 60) {
          x = cx + (Math.random() - 0.5) * 120; z = cz + (Math.random() - 0.5) * 120; y = 30 + Math.random() * 30;
        }
        arr[o] = x; arr[o + 1] = y; arr[o + 2] = z;
        arr[o + 3] = x + 0.2 * (1 + (w.kind === 'storm') * 2); arr[o + 4] = y - 1.2; arr[o + 5] = z;
      }
      p.needsUpdate = true;
      this.rain.material.opacity = 0.18 + Math.min(1, w.rainA) * 0.22;
    }
    this.snowfall.visible = w.snowA > 0.02;
    if (this.snowfall.visible) {
      const p = this.snowfall.geometry.attributes.position, arr = p.array;
      const n = arr.length / 3, active = Math.floor(n * Math.min(1, w.snowA / 2.2) * (w.snowA > 1.5 ? 1 : 0.6));
      const t = this.clock, wind = w.kind === 'blizzard' ? 9 : 1.2;
      for (let i = 0; i < n; i++) {
        const o = i * 3;
        if (i >= active) { arr[o + 1] = -100; continue; }
        let x = arr[o] + (wind + Math.sin(t + i) * 0.6) * dt, y = arr[o + 1] - (1.6 + (i % 7) * 0.15) * dt, z = arr[o + 2] + Math.cos(t * 0.7 + i) * 0.5 * dt;
        if (y < 0 || Math.abs(x - cx) > 60 || Math.abs(z - cz) > 60) { x = cx + (Math.random() - 0.5) * 120; z = cz + (Math.random() - 0.5) * 120; y = 20 + Math.random() * 30; }
        arr[o] = x; arr[o + 1] = y; arr[o + 2] = z;
      }
      p.needsUpdate = true;
    }
  }

  update(dt, focus) {
    this.clock += dt;
    this.updateWeather(dt, focus);
    for (const l of this.flashLights) {
      if (l.userData.life > 0) { l.userData.life -= dt; l.intensity = Math.max(0, l.userData.life / l.userData.max) * l.userData.I; } else l.intensity = 0;
    }
    if (focus) {
      this.sun.position.set(focus.x - 50, 90, focus.z + 35);
      this.sun.target.position.set(focus.x, 0, focus.z);
    }
  }

  render() {
    if (this.bloom) this.bloom.enabled = !!this.q.bloom;
    if (this.q.bloom || this.q.grade) this.composer.render();
    else this.renderer.render(this.scene, this.camera);
  }
}

// ───────── зруйнована будівля ─────────
function makeRuin(W, D, floors, rnd, panel) {
  const FH = 3, T = 0.35;
  const walls = [], slabs = [], dark = [], rubble = [];
  const H = floors * FH;
  // профіль руйнування: V-подібний обвал у випадковому місці
  const cx = (rnd() - 0.5) * W * 0.8, cw = W * (0.2 + rnd() * 0.35), depth = H * (0.3 + rnd() * 0.7);
  const cz = (rnd() - 0.5) * D * 0.6;
  const maxH = (x, z) => {
    const dd = Math.max(0, 1 - Math.hypot((x - cx) / cw, (z - cz) / (D * 0.8)));
    return H - depth * Math.min(1, dd * 1.6) - (rnd() < 0.08 ? FH : 0);
  };
  const win = panel ? 1.5 : 1.2, pier = panel ? 1.6 : 1.4;
  const facade = (len, place) => {
    const cols = Math.max(1, Math.floor(len / (win + pier)));
    const step = len / cols;
    for (let c = 0; c < cols; c++) {
      const u0 = -len / 2 + c * step;
      for (let f = 0; f < floors; f++) {
        const y0 = f * FH;
        const mid = u0 + step / 2;
        const lim = place(mid, 0, true);
        if (y0 + FH > lim + 0.5) {
          // зламаний верх — короткий уламок
          if (y0 < lim) {
            const hh = Math.max(0.3, lim - y0);
            walls.push(place(u0 + step * 0.25, y0 + hh / 2, false, step * 0.5, hh));
          }
          continue;
        }
        const tint = 0.55 + rnd() * 0.45 - (f === floors - 1 ? 0.1 : 0);
        // простінок, підвіконня, перемичка
        walls.push(colorGeo(place(u0 + (step - win) / 4, y0 + FH / 2, false, (step - win) / 2, FH), tint));
        walls.push(colorGeo(place(u0 + step - (step - win) / 4, y0 + FH / 2, false, (step - win) / 2, FH), tint));
        if (rnd() > 0.06) walls.push(colorGeo(place(mid, y0 + 0.45, false, win, 0.9), tint));
        walls.push(colorGeo(place(mid, y0 + FH - 0.25, false, win, 0.5), tint * 0.95));
      }
    }
  };
  for (const side of [-1, 1]) {
    facade(W, (u, y, q, w, h) => {
      if (q) return maxH(u, side * D / 2);
      return boxAt(w, h, T, u, y, side * D / 2);
    });
    facade(D, (u, y, q, w, h) => {
      if (q) return maxH(side * W / 2, u);
      return boxAt(T, h, w, side * W / 2, y, u);
    });
  }
  walls.forEach((g, i) => { if (!g.attributes.color) walls[i] = colorGeo(g, 0.5); });
  // перекриття
  for (let f = 1; f <= floors; f++) {
    const y = f * FH - 0.1;
    const segs = 4;
    for (let s = 0; s < segs; s++) {
      const x0 = -W / 2 + (s + 0.5) * (W / segs);
      if (maxH(x0, 0) < y - 0.2) {
        if (rnd() < 0.5) slabs.push(colorGeo(boxAt(W / segs, 0.22, D * 0.9, x0, Math.max(0.6, maxH(x0, 0) - 0.5), 0, 0, 0, (rnd() - 0.5) * 0.7), 0.6));
        continue;
      }
      slabs.push(colorGeo(boxAt(W / segs, 0.22, D - 0.1, x0, y, 0), 0.7 + rnd() * 0.2));
    }
  }
  // темний «інтер'єр» за вікнами
  const ih = Math.max(1, H - depth - 0.6);
  dark.push(boxAt(W - 1.2, ih, D - 1.2, 0, ih / 2, 0));
  rubble.push([cx, D / 2 + 2.5, 2.5 + depth * 0.4], [cx * 0.7, -D / 2 - 2, 1.8 + depth * 0.25]);
  return { walls, slabs, dark, rubble };
}

function makeNetTexture() {
  const c = document.createElement('canvas');
  c.width = c.height = 256;
  const g = c.getContext('2d');
  g.clearRect(0, 0, 256, 256);
  const cols = ['#4c5a33', '#5d6b3c', '#3d4a2b', '#6b6a44', '#55613a'];
  for (let i = 0; i < 420; i++) {
    g.fillStyle = cols[i % cols.length];
    const x = Math.random() * 256, y = Math.random() * 256, s = 3 + Math.random() * 7;
    g.beginPath();
    g.ellipse(x, y, s, s * 0.5, Math.random() * 3, 0, Math.PI * 2);
    g.fill();
  }
  g.strokeStyle = 'rgba(45,51,32,0.9)'; g.lineWidth = 1;
  for (let i = 0; i < 256; i += 8) { g.beginPath(); g.moveTo(i, 0); g.lineTo(i, 256); g.stroke(); g.beginPath(); g.moveTo(0, i); g.lineTo(256, i); g.stroke(); }
  const t = new THREE.CanvasTexture(c);
  t.colorSpace = THREE.SRGBColorSpace;
  t.wrapS = t.wrapT = THREE.RepeatWrapping;
  t.repeat.set(3, 3);
  return t;
}

// ───────── листя: пластини з процедурною текстурою ─────────
let _foliage = null;
function foliageMaterial() {
  if (_foliage) return _foliage;
  const c = document.createElement('canvas');
  c.width = c.height = 512;
  const g = c.getContext('2d');
  const r = srand(4242);
  const greens = [[52, 74, 30], [66, 92, 36], [80, 104, 44], [44, 62, 28], [92, 112, 52], [60, 84, 34]];
  // густий центр + рвані краї
  for (let i = 0; i < 2600; i++) {
    const a = r() * Math.PI * 2, d = Math.pow(r(), 0.65) * 230;
    const x = 256 + Math.cos(a) * d, y = 256 + Math.sin(a) * d * 0.92;
    const [cr, cg, cb] = greens[Math.floor(r() * greens.length)];
    const shade = 0.65 + (1 - d / 230) * 0.25 + r() * 0.25 - (y > 300 ? 0.15 : 0);
    g.fillStyle = `rgb(${cr * shade | 0},${cg * shade | 0},${cb * shade | 0})`;
    g.save(); g.translate(x, y); g.rotate(r() * Math.PI * 2);
    const L = 7 + r() * 9;
    g.beginPath(); g.ellipse(0, 0, L, L * 0.42, 0, 0, Math.PI * 2); g.fill();
    g.restore();
  }
  // гілочки
  g.strokeStyle = 'rgba(60,45,30,0.6)'; g.lineWidth = 2;
  for (let i = 0; i < 14; i++) { g.beginPath(); g.moveTo(256, 300); g.lineTo(256 + (r() - 0.5) * 300, 256 + (r() - 0.5) * 300); g.stroke(); }
  const t = new THREE.CanvasTexture(c);
  t.colorSpace = THREE.SRGBColorSpace;
  t.anisotropy = 4;
  _foliage = patchWeather(new THREE.MeshStandardMaterial({ map: t, alphaTest: 0.5, side: THREE.DoubleSide, roughness: 0.85 }), true);
  return _foliage;
}
// Набір перехресних пластин в еліпсоїді з «сферичними» нормалями (об'ємне освітлення крони)
function crownGeometry(n, seed, flat = false) {
  const r = srand(seed * 999 + 1);
  const pos = [], nor = [], uv = [];
  for (let i = 0; i < n; i++) {
    const a = r() * Math.PI * 2, el = flat ? 0 : (r() - 0.35) * 1.2, d = flat ? r() * 0.6 : Math.pow(r(), 0.5) * 0.85;
    const cx = Math.cos(a) * Math.cos(el) * d, cy = Math.sin(el) * d * 0.8, cz = Math.sin(a) * Math.cos(el) * d;
    const s = flat ? 0.9 : 0.9 + r() * 0.6;
    const q = new THREE.Quaternion().setFromEuler(new THREE.Euler(flat ? 0 : (r() - 0.5) * 2.4, r() * Math.PI * 2, flat ? 0 : (r() - 0.5) * 1.2));
    const corners = [[-1, -1], [1, -1], [1, 1], [-1, -1], [1, 1], [-1, 1]];
    for (const [u, v] of corners) {
      const p = new THREE.Vector3(u * s * 0.5, v * s * 0.5, 0).applyQuaternion(q).add(new THREE.Vector3(cx, cy, cz));
      if (flat) p.y = Math.max(-0.05, p.y);
      pos.push(p.x, p.y, p.z);
      const nn = new THREE.Vector3(p.x, flat ? 1 : p.y + 0.35, p.z).normalize();
      nor.push(nn.x, nn.y, nn.z);
      uv.push((u + 1) / 2, (v + 1) / 2);
    }
  }
  const geo = new THREE.BufferGeometry();
  geo.setAttribute('position', new THREE.Float32BufferAttribute(pos, 3));
  geo.setAttribute('normal', new THREE.Float32BufferAttribute(nor, 3));
  geo.setAttribute('uv', new THREE.Float32BufferAttribute(uv, 2));
  geo.computeBoundingSphere();
  return geo;
}
