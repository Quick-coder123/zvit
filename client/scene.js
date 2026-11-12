// 3D-світ: рендерер, пост-обробка, місцевість, небо, ядро, дороги, портали, декор.
import * as THREE from 'three';
import { EffectComposer } from 'three/addons/postprocessing/EffectComposer.js';
import { RenderPass } from 'three/addons/postprocessing/RenderPass.js';
import { UnrealBloomPass } from 'three/addons/postprocessing/UnrealBloomPass.js';
import { OutputPass } from 'three/addons/postprocessing/OutputPass.js';
import { RoomEnvironment } from 'three/addons/environments/RoomEnvironment.js';
import { LANES, PORTALS, DEPOSITS, CAMPS, BASE_R, WORLD, distToLanes, canBuildAt, GRID } from '/shared/data.js';

export const QUALITY = {
  low: { name: 'Низька', shadows: 0, bloom: false, pr: 1, particles: 0.4, decor: 0.4 },
  medium: { name: 'Середня', shadows: 1024, bloom: true, pr: 1, particles: 0.75, decor: 0.75 },
  high: { name: 'Висока', shadows: 2048, bloom: true, pr: 1.25, particles: 1, decor: 1 },
  ultra: { name: 'Ультра', shadows: 4096, bloom: true, pr: 2, particles: 1.3, decor: 1.3 },
};

// детермінований шум
function hash(x, y) { const s = Math.sin(x * 127.1 + y * 311.7) * 43758.5453; return s - Math.floor(s); }
function vnoise(x, y) {
  const xi = Math.floor(x), yi = Math.floor(y), xf = x - xi, yf = y - yi;
  const u = xf * xf * (3 - 2 * xf), v = yf * yf * (3 - 2 * yf);
  const a = hash(xi, yi), b = hash(xi + 1, yi), c = hash(xi, yi + 1), d = hash(xi + 1, yi + 1);
  return a + (b - a) * u + (c - a) * v + (a - b - c + d) * u * v;
}
export function fbm(x, y, o = 5) { let s = 0, a = 0.5, f = 1; for (let i = 0; i < o; i++) { s += vnoise(x * f, y * f) * a; f *= 2; a *= 0.5; } return s; }
function srand(seed) { let s = seed; return () => { s = (s * 16807) % 2147483647; return s / 2147483647; }; }

export function terrainHeight(x, z) {
  const r = Math.hypot(x, z);
  const edge = THREE.MathUtils.smoothstep(r, 112, 150);
  const lane = Math.min(1, distToLanes(x, z) / 10);
  const hills = (fbm(x * 0.02 + 10, z * 0.02 - 4, 4) - 0.35) * 2.2 * THREE.MathUtils.smoothstep(r, BASE_R + 6, BASE_R + 30) * lane;
  return Math.max(0, hills) + edge * (16 + fbm(x * 0.03, z * 0.03, 4) * 36);
}

export class World {
  constructor(canvas, quality = 'high') {
    this.canvas = canvas;
    this.q = QUALITY[quality] || QUALITY.high;
    this.qualityKey = quality;
    const r = (this.renderer = new THREE.WebGLRenderer({ canvas, antialias: false, powerPreference: 'high-performance' }));
    r.setPixelRatio(Math.min(window.devicePixelRatio, this.q.pr));
    r.toneMapping = THREE.ACESFilmicToneMapping;
    r.toneMappingExposure = 1.05;
    r.outputColorSpace = THREE.SRGBColorSpace;
    r.shadowMap.enabled = this.q.shadows > 0;
    r.shadowMap.type = THREE.PCFSoftShadowMap;

    this.scene = new THREE.Scene();
    this.scene.fog = new THREE.FogExp2(0x0a0f22, 0.0062);
    // карта оточення для металевих поверхонь
    const pmrem = new THREE.PMREMGenerator(r);
    const envScene = new RoomEnvironment();
    envScene.traverse((o) => { if (o.isMesh && o.material && o.material.color) o.material.color.multiply(new THREE.Color(0.75, 0.8, 1.15)); });
    this.scene.environment = pmrem.fromScene(envScene, 0.04).texture;
    this.scene.environmentIntensity = 0.45;
    this.camera = new THREE.PerspectiveCamera(48, 1, 0.5, 900);
    this.camera.position.set(0, 40, 30);
    this.clock = 0;

    this.buildLights();
    this.buildSky();
    this.buildTerrain();
    this.buildBase();
    this.buildLanes();
    this.buildPortals();
    this.buildCore();
    this.buildDeposits();
    this.buildCamps();
    this.buildDecor();
    this.buildGridOverlay();
    this.buildPost();
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
    this.scene.traverse((o) => { if (o.material) { const ms = Array.isArray(o.material) ? o.material : [o.material]; ms.forEach((m) => (m.needsUpdate = true)); } });
    this.resize();
  }

  resize() {
    const w = window.innerWidth, h = window.innerHeight;
    this.renderer.setSize(w, h, false);
    this.camera.aspect = w / h;
    this.camera.updateProjectionMatrix();
    if (this.composer) {
      this.composer.setPixelRatio(this.renderer.getPixelRatio());
      this.composer.setSize(w, h);
    }
  }

  buildLights() {
    this.scene.add(new THREE.HemisphereLight(0x6f8cff, 0x1a0f20, 0.75));
    this.scene.add(new THREE.AmbientLight(0x223355, 0.35));
    const s = (this.sun = new THREE.DirectionalLight(0xffd6b0, 2.3));
    s.position.set(-60, 90, 40);
    s.castShadow = true;
    s.shadow.mapSize.set(this.q.shadows || 1024, this.q.shadows || 1024);
    const sc = s.shadow.camera;
    sc.left = -70; sc.right = 70; sc.top = 70; sc.bottom = -70; sc.near = 10; sc.far = 260;
    s.shadow.bias = -0.0004;
    s.shadow.normalBias = 0.04;
    this.scene.add(s, s.target);
    const rim = new THREE.DirectionalLight(0x9a5cff, 0.9);
    rim.position.set(80, 40, -90);
    this.scene.add(rim);
    // пул точкових світел для вибухів (фіксована кількість, щоб не перекомпілювати шейдери)
    this.flashLights = [];
    for (let i = 0; i < 6; i++) {
      const l = new THREE.PointLight(0xffffff, 0, 22, 2);
      l.userData.life = 0;
      this.scene.add(l);
      this.flashLights.push(l);
    }
  }
  flash(x, y, z, color, intensity = 60, life = 0.25) {
    let l = this.flashLights.find((f) => f.userData.life <= 0) || this.flashLights.reduce((a, b) => (a.userData.life < b.userData.life ? a : b));
    l.position.set(x, y, z);
    l.color.set(color);
    l.userData.life = life; l.userData.max = life; l.userData.I = intensity;
  }

  buildSky() {
    const geo = new THREE.SphereGeometry(600, 48, 24);
    const mat = new THREE.ShaderMaterial({
      side: THREE.BackSide, depthWrite: false, fog: false,
      uniforms: { time: { value: 0 } },
      vertexShader: `varying vec3 vP; void main(){ vP = normalize(position); gl_Position = projectionMatrix * modelViewMatrix * vec4(position,1.0); }`,
      fragmentShader: `
        varying vec3 vP; uniform float time;
        float h(vec3 p){ return fract(sin(dot(p, vec3(12.9898,78.233,45.164)))*43758.5453); }
        void main(){
          float y = vP.y;
          vec3 top = vec3(0.01,0.015,0.05);
          vec3 mid = vec3(0.06,0.04,0.16);
          vec3 hor = vec3(0.32,0.12,0.30);
          vec3 c = mix(hor, mid, smoothstep(-0.05, 0.25, y));
          c = mix(c, top, smoothstep(0.25, 0.9, y));
          // туманність
          float n = sin(vP.x*6.0+vP.z*3.0)*sin(vP.z*5.0-vP.y*4.0+1.3)*0.5+0.5;
          c += vec3(0.25,0.05,0.35) * pow(n,4.0) * smoothstep(0.05,0.6,y) * 0.6;
          c += vec3(0.0,0.25,0.35) * pow(1.0-n,6.0) * smoothstep(0.2,0.8,y) * 0.35;
          // зірки
          vec3 sp = floor(vP*320.0);
          float s = h(sp);
          float tw = 0.6 + 0.4*sin(time*2.0 + s*50.0);
          c += vec3(step(0.9975, s) * tw * smoothstep(0.0, 0.3, y));
          gl_FragColor = vec4(c,1.0);
        }`,
    });
    this.sky = new THREE.Mesh(geo, mat);
    this.scene.add(this.sky);

    // планета з атмосферою
    const pg = new THREE.SphereGeometry(70, 64, 32);
    const pm = new THREE.ShaderMaterial({
      fog: false,
      uniforms: { sunDir: { value: new THREE.Vector3(-0.6, 0.4, 0.7).normalize() } },
      vertexShader: `varying vec3 vN; varying vec3 vW; void main(){ vN = normalize(normalMatrix*normal); vW = normalize((modelMatrix*vec4(position,1.0)).xyz - cameraPosition); gl_Position = projectionMatrix*modelViewMatrix*vec4(position,1.0);} `,
      fragmentShader: `varying vec3 vN; varying vec3 vW; uniform vec3 sunDir;
        void main(){
          float l = clamp(dot(vN, normalize(vec3(-0.7,0.3,0.6))), 0.0, 1.0);
          float band = sin(vN.y*24.0 + vN.x*4.0)*0.5+0.5;
          vec3 base = mix(vec3(0.35,0.12,0.45), vec3(0.85,0.45,0.35), band*0.6);
          float rim = pow(1.0 - abs(dot(vN, vec3(0,0,1))), 3.0);
          vec3 c = base * (0.08 + l*1.1) + vec3(0.6,0.3,1.0)*rim*0.9;
          gl_FragColor = vec4(c,1.0);
        }`,
    });
    const planet = new THREE.Mesh(pg, pm);
    planet.position.set(-260, 150, -380);
    this.scene.add(planet);
    const ring = new THREE.Mesh(
      new THREE.RingGeometry(95, 140, 96),
      new THREE.MeshBasicMaterial({ color: 0xc89cff, transparent: true, opacity: 0.25, side: THREE.DoubleSide, fog: false, depthWrite: false }),
    );
    ring.position.copy(planet.position);
    ring.rotation.set(1.2, 0.3, 0.4);
    this.scene.add(ring);
    const moon = new THREE.Mesh(new THREE.SphereGeometry(14, 32, 16), new THREE.MeshBasicMaterial({ color: 0x8fb8ff, fog: false }));
    moon.position.set(200, 120, -420);
    this.scene.add(moon);
  }

  buildTerrain() {
    const S = 2048;
    const cv = document.createElement('canvas');
    cv.width = cv.height = 1024;
    const g = cv.getContext('2d');
    const img = g.createImageData(1024, 1024);
    const rough = document.createElement('canvas');
    rough.width = rough.height = 1024;
    const rg = rough.getContext('2d');
    const rimg = rg.createImageData(1024, 1024);
    for (let y = 0; y < 1024; y++) {
      for (let x = 0; x < 1024; x++) {
        const wx = (x / 1024 - 0.5) * 300, wz = (y / 1024 - 0.5) * 300;
        const n = fbm(wx * 0.05, wz * 0.05, 5);
        const n2 = fbm(wx * 0.3 + 7, wz * 0.3, 3);
        const cr = Math.pow(Math.abs(Math.sin(fbm(wx * 0.025 + 3, wz * 0.025, 3) * 18)), 8); // трещини
        let rC = 38 + n * 50 + n2 * 22, gC = 30 + n * 34 + n2 * 14, bC = 46 + n * 46 + n2 * 20;
        // дороги темніші
        const dl = distToLanes(wx, wz);
        if (dl < 6) { const k = 1 - dl / 6; rC *= 1 - 0.45 * k; gC *= 1 - 0.45 * k; bC *= 1 - 0.35 * k; }
        rC += cr * 40; gC += cr * 10; bC += cr * 70;
        const i = (y * 1024 + x) * 4;
        img.data[i] = rC; img.data[i + 1] = gC; img.data[i + 2] = bC; img.data[i + 3] = 255;
        const ro = 170 + n2 * 70 - cr * 120;
        rimg.data[i] = rimg.data[i + 1] = rimg.data[i + 2] = ro; rimg.data[i + 3] = 255;
      }
    }
    g.putImageData(img, 0, 0);
    rg.putImageData(rimg, 0, 0);
    const tex = new THREE.CanvasTexture(cv);
    tex.colorSpace = THREE.SRGBColorSpace;
    tex.anisotropy = 8;
    const rtex = new THREE.CanvasTexture(rough);
    // деталізація
    const det = document.createElement('canvas');
    det.width = det.height = 256;
    const dg = det.getContext('2d');
    const dimg = dg.createImageData(256, 256);
    for (let i = 0; i < 256 * 256; i++) {
      const x = i % 256, y = Math.floor(i / 256);
      const v = 128 + (fbm(x * 0.06, y * 0.06, 4) - 0.5) * 120 + (Math.random() - 0.5) * 30;
      dimg.data[i * 4] = dimg.data[i * 4 + 1] = dimg.data[i * 4 + 2] = v; dimg.data[i * 4 + 3] = 255;
    }
    dg.putImageData(dimg, 0, 0);
    const dtex = new THREE.CanvasTexture(det);
    dtex.wrapS = dtex.wrapT = THREE.RepeatWrapping;
    dtex.repeat.set(60, 60);

    const geo = new THREE.PlaneGeometry(300, 300, 220, 220);
    geo.rotateX(-Math.PI / 2);
    const pos = geo.attributes.position;
    for (let i = 0; i < pos.count; i++) pos.setY(i, terrainHeight(pos.getX(i), pos.getZ(i)));
    geo.computeVertexNormals();
    const mat = new THREE.MeshStandardMaterial({ map: tex, roughnessMap: rtex, roughness: 1, metalness: 0.15, bumpMap: dtex, bumpScale: 1.6 });
    const ground = (this.ground = new THREE.Mesh(geo, mat));
    ground.receiveShadow = true;
    this.scene.add(ground);
    void S;
  }

  buildBase() {
    // металева платформа бази з гексагональною сіткою
    const cv = document.createElement('canvas');
    cv.width = cv.height = 1024;
    const g = cv.getContext('2d');
    g.fillStyle = '#000'; g.fillRect(0, 0, 1024, 1024);
    const R = 22;
    g.lineWidth = 2;
    for (let row = -2; row < 1024 / (R * 1.5) + 2; row++) {
      for (let col = -2; col < 1024 / (R * Math.sqrt(3)) + 2; col++) {
        const cx = col * R * Math.sqrt(3) + (row % 2) * R * Math.sqrt(3) / 2, cy = row * R * 1.5;
        const d = Math.hypot(cx - 512, cy - 512) / 512;
        const a = Math.max(0, 0.22 - d * 0.12) + (hash(row, col) > 0.95 ? 0.35 : 0);
        g.strokeStyle = `rgba(58,224,255,${a})`;
        g.beginPath();
        for (let k = 0; k < 7; k++) { const an = Math.PI / 3 * k + Math.PI / 6; g.lineTo(cx + Math.cos(an) * R, cy + Math.sin(an) * R); }
        g.stroke();
      }
    }
    // кільця
    for (const [rr, a] of [[500, 0.9], [470, 0.3], [140, 0.6]]) { g.strokeStyle = `rgba(58,224,255,${a})`; g.lineWidth = 4; g.beginPath(); g.arc(512, 512, rr, 0, Math.PI * 2); g.stroke(); }
    const em = new THREE.CanvasTexture(cv);
    em.colorSpace = THREE.SRGBColorSpace;
    const mat = new THREE.MeshStandardMaterial({ color: 0x1a2230, metalness: 0.8, roughness: 0.35, emissive: 0xffffff, emissiveMap: em, emissiveIntensity: 0.6, transparent: false });
    this.baseMat = mat;
    const plat = new THREE.Mesh(new THREE.CylinderGeometry(BASE_R + 2, BASE_R + 3.5, 0.6, 96), mat);
    plat.position.y = -0.22;
    plat.receiveShadow = true;
    this.scene.add(plat);
    // опорні стовпи-маяки по периметру
    const pillarGeo = new THREE.CylinderGeometry(0.35, 0.5, 4, 8);
    const pillarMat = new THREE.MeshStandardMaterial({ color: 0x334055, metalness: 0.9, roughness: 0.3 });
    const capMat = new THREE.MeshStandardMaterial({ color: 0x000000, emissive: 0x3ae0ff, emissiveIntensity: 3 });
    for (let i = 0; i < 24; i++) {
      const a = (i / 24) * Math.PI * 2;
      const x = Math.cos(a) * (BASE_R + 2.6), z = Math.sin(a) * (BASE_R + 2.6);
      if (distToLanes(x, z) < 4) continue;
      const p = new THREE.Mesh(pillarGeo, pillarMat);
      p.position.set(x, 2, z);
      p.castShadow = true;
      this.scene.add(p);
      const c = new THREE.Mesh(new THREE.SphereGeometry(0.35, 12, 8), capMat);
      c.position.set(x, 4.1, z);
      this.scene.add(c);
    }
  }

  buildLanes() {
    // стрічки доріг з біжучими шевронами
    const cv = document.createElement('canvas');
    cv.width = 128; cv.height = 256;
    const g = cv.getContext('2d');
    g.fillStyle = '#000'; g.fillRect(0, 0, 128, 256);
    g.strokeStyle = 'rgba(255,90,60,0.9)'; g.lineWidth = 10;
    g.beginPath(); g.moveTo(24, 160); g.lineTo(64, 110); g.lineTo(104, 160); g.stroke();
    g.fillStyle = 'rgba(255,60,90,1)'; g.fillRect(0, 0, 6, 256); g.fillRect(122, 0, 6, 256);
    const tex = (this.laneTex = new THREE.CanvasTexture(cv));
    tex.wrapS = tex.wrapT = THREE.RepeatWrapping;
    tex.colorSpace = THREE.SRGBColorSpace;
    const mat = new THREE.MeshStandardMaterial({ color: 0x0c0d14, roughness: 0.6, metalness: 0.6, emissive: 0xffffff, emissiveMap: tex, emissiveIntensity: 1.1, transparent: true, opacity: 0.92, depthWrite: false, polygonOffset: true, polygonOffsetFactor: -2 });
    for (const lane of LANES) {
      // згладжена крива
      const pts = lane.map(([x, z]) => new THREE.Vector3(x, 0, z));
      const curve = new THREE.CatmullRomCurve3(pts, false, 'centripetal');
      const N = 200;
      const W = 2.6;
      const verts = [], uvs = [], idx = [];
      let len = 0, prev = curve.getPoint(0);
      for (let i = 0; i <= N; i++) {
        const p = curve.getPoint(i / N);
        len += p.distanceTo(prev); prev = p;
        const t = curve.getTangent(i / N);
        const nx = -t.z, nz = t.x;
        const y = terrainHeight(p.x, p.z) + 0.08;
        verts.push(p.x + nx * W, y, p.z + nz * W, p.x - nx * W, y, p.z - nz * W);
        uvs.push(0, len / 5, 1, len / 5);
        if (i < N) { const a = i * 2; idx.push(a, a + 2, a + 1, a + 1, a + 2, a + 3); }
      }
      const geo = new THREE.BufferGeometry();
      geo.setAttribute('position', new THREE.Float32BufferAttribute(verts, 3));
      geo.setAttribute('uv', new THREE.Float32BufferAttribute(uvs, 2));
      geo.setIndex(idx);
      geo.computeVertexNormals();
      const m = new THREE.Mesh(geo, mat);
      m.receiveShadow = true;
      m.renderOrder = 1;
      this.scene.add(m);
    }
  }

  buildPortals() {
    this.portals = [];
    const swirl = new THREE.ShaderMaterial({
      transparent: true, depthWrite: false, blending: THREE.AdditiveBlending, side: THREE.DoubleSide,
      uniforms: { time: { value: 0 } },
      vertexShader: `varying vec2 vUv; void main(){ vUv = uv; gl_Position = projectionMatrix*modelViewMatrix*vec4(position,1.0);} `,
      fragmentShader: `varying vec2 vUv; uniform float time;
        void main(){ vec2 p = vUv*2.0-1.0; float r = length(p); float a = atan(p.y,p.x);
          float s = sin(a*5.0 + r*14.0 - time*5.0)*0.5+0.5;
          float m = smoothstep(1.0,0.2,r);
          vec3 c = mix(vec3(1.0,0.1,0.35), vec3(0.6,0.1,1.0), s) * (s*0.8+0.4) * m;
          c += vec3(1.0,0.8,1.0) * smoothstep(0.35,0.0,r)*0.8;
          gl_FragColor = vec4(c, m); }`,
    });
    this.portalMat = swirl;
    const frameMat = new THREE.MeshStandardMaterial({ color: 0x221122, metalness: 0.9, roughness: 0.3, emissive: 0xff2266, emissiveIntensity: 1.2 });
    for (const [x, z] of PORTALS) {
      const grp = new THREE.Group();
      const y = terrainHeight(x, z);
      grp.position.set(x, y, z);
      grp.lookAt(0, y, 0);
      const ring = new THREE.Mesh(new THREE.TorusGeometry(6, 0.7, 12, 48), frameMat);
      ring.position.y = 6.5;
      ring.castShadow = true;
      grp.add(ring);
      const disc = new THREE.Mesh(new THREE.CircleGeometry(5.6, 48), swirl);
      disc.position.y = 6.5;
      grp.add(disc);
      for (const s of [-1, 1]) {
        const leg = new THREE.Mesh(new THREE.BoxGeometry(1.2, 5, 1.6), frameMat);
        leg.position.set(s * 5.5, 1.8, 0);
        leg.rotation.z = s * 0.35;
        grp.add(leg);
      }
      const l = new THREE.PointLight(0xff2266, 40, 30, 2);
      l.position.y = 6.5;
      grp.add(l);
      this.scene.add(grp);
      this.portals.push(grp);
    }
  }

  buildCore() {
    const g = (this.core = new THREE.Group());
    const metal = new THREE.MeshStandardMaterial({ color: 0x2a3446, metalness: 0.9, roughness: 0.25 });
    const glow = (this.coreGlowMat = new THREE.MeshStandardMaterial({ color: 0x000000, emissive: 0x3ae0ff, emissiveIntensity: 1.6 }));
    const base1 = new THREE.Mesh(new THREE.CylinderGeometry(6.5, 7.5, 1.2, 8), metal);
    base1.position.y = 0.6;
    const base2 = new THREE.Mesh(new THREE.CylinderGeometry(4.5, 5.8, 1.6, 8), metal);
    base2.position.y = 2;
    const ringG = new THREE.Mesh(new THREE.TorusGeometry(5.2, 0.15, 8, 64), glow);
    ringG.rotation.x = Math.PI / 2; ringG.position.y = 2.85;
    for (const m of [base1, base2]) { m.castShadow = m.receiveShadow = true; g.add(m); }
    g.add(ringG);
    // пілони
    for (let i = 0; i < 4; i++) {
      const a = (i / 4) * Math.PI * 2 + Math.PI / 4;
      const p = new THREE.Mesh(new THREE.BoxGeometry(0.8, 7, 0.8), metal);
      p.position.set(Math.cos(a) * 4.6, 5, Math.sin(a) * 4.6);
      p.rotation.z = Math.cos(a) * 0.2; p.rotation.x = -Math.sin(a) * 0.2;
      p.castShadow = true;
      g.add(p);
      const tip = new THREE.Mesh(new THREE.OctahedronGeometry(0.5), glow);
      tip.position.set(Math.cos(a) * 3.9, 8.8, Math.sin(a) * 3.9);
      g.add(tip);
    }
    const crystal = (this.coreCrystal = new THREE.Mesh(new THREE.IcosahedronGeometry(2.4, 1), new THREE.MeshStandardMaterial({ color: 0x0a1a28, emissive: 0x3ae0ff, emissiveIntensity: 0.55, metalness: 0.2, roughness: 0.1, flatShading: true })));
    crystal.position.y = 7;
    g.add(crystal);
    this.coreRings = [];
    for (let i = 0; i < 3; i++) {
      const r = new THREE.Mesh(new THREE.TorusGeometry(3.4 + i * 0.7, 0.09, 8, 80), glow);
      r.position.y = 7;
      g.add(r);
      this.coreRings.push(r);
    }
    // промінь у небо
    const beamMat = new THREE.ShaderMaterial({
      transparent: true, depthWrite: false, blending: THREE.AdditiveBlending,
      uniforms: { time: { value: 0 }, color: { value: new THREE.Color(0x3ae0ff) } },
      vertexShader: `varying vec2 vUv; void main(){ vUv=uv; gl_Position=projectionMatrix*modelViewMatrix*vec4(position,1.0);} `,
      fragmentShader: `varying vec2 vUv; uniform float time; uniform vec3 color;
        void main(){ float f = (1.0 - vUv.y); float s = 0.6+0.4*sin(vUv.y*40.0 - time*6.0); gl_FragColor = vec4(color * f * s * 0.9, f*0.8); }`,
    });
    this.coreBeamMat = beamMat;
    const beam = new THREE.Mesh(new THREE.CylinderGeometry(0.6, 1.2, 120, 16, 1, true), beamMat);
    beam.position.y = 67;
    g.add(beam);
    this.coreLight = new THREE.PointLight(0x3ae0ff, 120, 45, 1.6);
    this.coreLight.position.y = 7;
    g.add(this.coreLight);
    this.scene.add(g);
  }
  setCoreHealth(f) {
    const c = new THREE.Color(0x3ae0ff).lerp(new THREE.Color(0xff2244), 1 - f);
    this.coreGlowMat.emissive.copy(c);
    this.coreCrystal.material.emissive.copy(c);
    this.coreLight.color.copy(c);
    this.coreBeamMat.uniforms.color.value.copy(c);
  }

  buildDeposits() {
    this.depositObjs = [];
    const cmat = new THREE.MeshStandardMaterial({ color: 0x1a0a30, emissive: 0xb06cff, emissiveIntensity: 1.6, roughness: 0.15, metalness: 0.1, flatShading: true });
    this.crystalMat = cmat;
    for (const d of DEPOSITS) {
      const g = new THREE.Group();
      const y = terrainHeight(d.x, d.z);
      g.position.set(d.x, y, d.z);
      const rnd = srand(d.id * 977 + 3);
      for (let i = 0; i < 9; i++) {
        const h = 2 + rnd() * 5;
        const c = new THREE.Mesh(new THREE.CylinderGeometry(0, 0.5 + rnd() * 0.7, h, 6), cmat);
        const a = rnd() * 6.28, r = rnd() * 3.5;
        c.position.set(Math.cos(a) * r, h / 2 - 0.3, Math.sin(a) * r);
        c.rotation.set((rnd() - 0.5) * 0.8, rnd() * 6, (rnd() - 0.5) * 0.8);
        c.castShadow = true;
        g.add(c);
      }
      const l = new THREE.PointLight(0xb06cff, 25, 18, 2);
      l.position.y = 3;
      g.add(l);
      // бурова установка (з’являється після будівництва)
      const rig = new THREE.Group();
      const metal = new THREE.MeshStandardMaterial({ color: 0x3a4456, metalness: 0.85, roughness: 0.3 });
      const glowM = new THREE.MeshStandardMaterial({ color: 0, emissive: 0xb06cff, emissiveIntensity: 2.5 });
      for (let k = 0; k < 4; k++) {
        const a = k * Math.PI / 2 + Math.PI / 4;
        const leg = new THREE.Mesh(new THREE.BoxGeometry(0.4, 9, 0.4), metal);
        leg.position.set(Math.cos(a) * 2.2, 4.3, Math.sin(a) * 2.2);
        leg.rotation.set(-Math.sin(a) * 0.22, 0, Math.cos(a) * 0.22);
        leg.castShadow = true;
        rig.add(leg);
      }
      const top = new THREE.Mesh(new THREE.BoxGeometry(2.4, 1.2, 2.4), metal);
      top.position.y = 8.8;
      rig.add(top);
      const drill = new THREE.Mesh(new THREE.CylinderGeometry(0.5, 0.05, 7, 8), glowM);
      drill.position.y = 4.4;
      rig.add(drill);
      const tank = new THREE.Mesh(new THREE.CylinderGeometry(1.1, 1.1, 2.4, 16), metal);
      tank.position.set(3.6, 1.2, 0);
      tank.castShadow = true;
      rig.add(tank);
      const tankGlow = new THREE.Mesh(new THREE.TorusGeometry(1.15, 0.08, 6, 24), glowM);
      tankGlow.rotation.x = Math.PI / 2; tankGlow.position.set(3.6, 1.8, 0);
      rig.add(tankGlow);
      rig.visible = false;
      g.add(rig);
      this.scene.add(g);
      this.depositObjs.push({ g, rig, drill, def: d });
    }
  }
  setExtractors(list) {
    list.forEach((ex, i) => {
      const o = this.depositObjs[i];
      if (!o) return;
      o.rig.visible = ex.level > 0;
      o.level = ex.level;
      o.rig.scale.setScalar(1 + Math.min(0.5, ex.level * 0.03));
    });
  }

  buildCamps() {
    this.campObjs = [];
    const nestMat = new THREE.MeshStandardMaterial({ color: 0x2a1a14, roughness: 0.8, metalness: 0.1, emissive: 0x551100, emissiveIntensity: 0.4 });
    const ringMat = new THREE.MeshBasicMaterial({ color: 0xff5522, transparent: true, opacity: 0.18, depthWrite: false, side: THREE.DoubleSide });
    for (const c of CAMPS) {
      const g = new THREE.Group();
      const y = terrainHeight(c.x, c.z);
      g.position.set(c.x, y + 0.1, c.z);
      const ring = new THREE.Mesh(new THREE.RingGeometry(8.5, 9.2, 48), ringMat);
      ring.rotation.x = -Math.PI / 2;
      g.add(ring);
      const rnd = srand(c.id * 131 + 7);
      for (let i = 0; i < 5 + c.tier * 2; i++) {
        const h = 1.5 + rnd() * 3 + c.tier;
        const sp = new THREE.Mesh(new THREE.ConeGeometry(0.4 + rnd() * 0.5, h, 5), nestMat);
        const a = rnd() * 6.28, r = 4 + rnd() * 4;
        sp.position.set(Math.cos(a) * r, h / 2 - 0.2, Math.sin(a) * r);
        sp.rotation.set((rnd() - 0.5) * 0.9, 0, (rnd() - 0.5) * 0.9);
        sp.castShadow = true;
        g.add(sp);
      }
      const egg = new THREE.Mesh(new THREE.SphereGeometry(0.9 + c.tier * 0.3, 16, 12), new THREE.MeshStandardMaterial({ color: 0x220a00, emissive: [0xff8833, 0xff5522, 0xff2266, 0xffcc22][c.tier], emissiveIntensity: 1.4 }));
      egg.scale.y = 1.4;
      egg.position.y = 1;
      g.add(egg);
      this.scene.add(g);
      this.campObjs.push({ g, egg, def: c });
    }
  }

  buildDecor() {
    const rnd = srand(4242);
    const n = Math.floor(420 * this.q.decor);
    const rockGeo = new THREE.DodecahedronGeometry(1, 0);
    const rockMat = new THREE.MeshStandardMaterial({ color: 0x3a3448, roughness: 0.9, metalness: 0.1, flatShading: true });
    const rocks = new THREE.InstancedMesh(rockGeo, rockMat, n);
    const m = new THREE.Matrix4(), q = new THREE.Quaternion(), s = new THREE.Vector3(), p = new THREE.Vector3(), e = new THREE.Euler();
    const free = (x, z, pad) => {
      if (Math.hypot(x, z) < BASE_R + 5) return false;
      if (distToLanes(x, z) < 7 + pad) return false;
      for (const d of DEPOSITS) if (Math.hypot(d.x - x, d.z - z) < 8) return false;
      for (const c of CAMPS) if (Math.hypot(c.x - x, c.z - z) < 7) return false;
      return true;
    };
    let k = 0;
    for (let i = 0; i < n * 4 && k < n; i++) {
      const x = (rnd() - 0.5) * 290, z = (rnd() - 0.5) * 290;
      const r = Math.hypot(x, z);
      if (!free(x, z, 0) || r > 150) continue;
      const big = r > 110 ? 3 : 1;
      const sc = (0.5 + rnd() * rnd() * 3.2) * big;
      p.set(x, terrainHeight(x, z) + sc * 0.2, z);
      e.set(rnd() * 3, rnd() * 3, rnd() * 3); q.setFromEuler(e);
      s.set(sc * (0.8 + rnd() * 0.6), sc * (0.5 + rnd() * 0.7), sc * (0.8 + rnd() * 0.6));
      m.compose(p, q, s);
      rocks.setMatrixAt(k++, m);
    }
    rocks.count = k;
    rocks.castShadow = rocks.receiveShadow = true;
    this.scene.add(rocks);

    // світні кристали-рослини
    const nc = Math.floor(260 * this.q.decor);
    const cg = new THREE.OctahedronGeometry(0.5, 0);
    cg.scale(0.6, 2.2, 0.6);
    const cm = new THREE.MeshStandardMaterial({ color: 0x051018, emissive: 0xffffff, emissiveIntensity: 1.8, roughness: 0.2 });
    const crys = new THREE.InstancedMesh(cg, cm, nc);
    const palette = [new THREE.Color(0x33e0ff), new THREE.Color(0xb06cff), new THREE.Color(0x44ff99), new THREE.Color(0xff4fd8)];
    k = 0;
    for (let i = 0; i < nc * 5 && k < nc; i++) {
      const cx = (rnd() - 0.5) * 260, cz = (rnd() - 0.5) * 260;
      if (!free(cx, cz, 1) || Math.hypot(cx, cz) > 125) continue;
      const col = palette[Math.floor(rnd() * palette.length)];
      const cnt = 2 + Math.floor(rnd() * 4);
      for (let j = 0; j < cnt && k < nc; j++) {
        const x = cx + (rnd() - 0.5) * 2.5, z = cz + (rnd() - 0.5) * 2.5;
        const sc = 0.5 + rnd() * 1.3;
        p.set(x, terrainHeight(x, z) + sc * 0.8, z);
        e.set((rnd() - 0.5) * 0.7, rnd() * 3, (rnd() - 0.5) * 0.7); q.setFromEuler(e);
        s.setScalar(sc);
        m.compose(p, q, s);
        crys.setMatrixAt(k, m);
        crys.setColorAt(k, col);
        k++;
      }
    }
    crys.count = k;
    this.scene.add(crys);

    // уламки / контейнери біля бази
    const crateGeo = new THREE.BoxGeometry(2, 1.4, 1.4);
    const crateMat = new THREE.MeshStandardMaterial({ color: 0x3b4252, metalness: 0.7, roughness: 0.45 });
    const crates = new THREE.InstancedMesh(crateGeo, crateMat, 40);
    k = 0;
    for (let i = 0; i < 400 && k < 40; i++) {
      const a = rnd() * 6.28, r = BASE_R + 6 + rnd() * 14;
      const x = Math.cos(a) * r, z = Math.sin(a) * r;
      if (!free(x, z, 1)) continue;
      p.set(x, 0.7 + terrainHeight(x, z), z);
      e.set(0, rnd() * 6, 0); q.setFromEuler(e); s.setScalar(0.8 + rnd() * 0.6);
      m.compose(p, q, s);
      crates.setMatrixAt(k++, m);
    }
    crates.count = k;
    crates.castShadow = crates.receiveShadow = true;
    this.scene.add(crates);

    // пил у повітрі
    const dn = Math.floor(1500 * this.q.decor);
    const dg = new THREE.BufferGeometry();
    const dp = new Float32Array(dn * 3);
    for (let i = 0; i < dn; i++) { dp[i * 3] = (rnd() - 0.5) * 260; dp[i * 3 + 1] = rnd() * 30; dp[i * 3 + 2] = (rnd() - 0.5) * 260; }
    dg.setAttribute('position', new THREE.BufferAttribute(dp, 3));
    this.dust = new THREE.Points(dg, new THREE.PointsMaterial({ color: 0x88aaff, size: 0.18, transparent: true, opacity: 0.5, depthWrite: false, blending: THREE.AdditiveBlending }));
    this.scene.add(this.dust);
  }

  buildGridOverlay() {
    const cells = [];
    const M = Math.floor(BASE_R / GRID) * GRID;
    for (let x = -M; x <= M; x += GRID) for (let z = -M; z <= M; z += GRID) if (canBuildAt(x, z)) cells.push([x, z]);
    this.buildCells = cells;
    const geo = new THREE.PlaneGeometry(GRID * 0.86, GRID * 0.86);
    geo.rotateX(-Math.PI / 2);
    const mat = new THREE.MeshBasicMaterial({ color: 0x3ae0ff, transparent: true, opacity: 0.12, depthWrite: false, blending: THREE.AdditiveBlending });
    const im = new THREE.InstancedMesh(geo, mat, cells.length);
    const m = new THREE.Matrix4();
    cells.forEach(([x, z], i) => { m.makeTranslation(x, 0.12, z); im.setMatrixAt(i, m); });
    im.visible = false;
    this.gridOverlay = im;
    this.scene.add(im);
  }

  buildPost() {
    const size = this.renderer.getDrawingBufferSize(new THREE.Vector2());
    const rt = new THREE.WebGLRenderTarget(size.x, size.y, { type: THREE.HalfFloatType, samples: 4 });
    this.composer = new EffectComposer(this.renderer, rt);
    this.composer.addPass(new RenderPass(this.scene, this.camera));
    this.bloom = new UnrealBloomPass(new THREE.Vector2(size.x, size.y), 0.55, 0.5, 0.86);
    this.composer.addPass(this.bloom);
    this.composer.addPass(new OutputPass());
  }

  update(dt, focus) {
    this.clock += dt;
    const t = this.clock;
    this.sky.material.uniforms.time.value = t;
    this.portalMat.uniforms.time.value = t;
    this.coreBeamMat.uniforms.time.value = t;
    this.laneTex.offset.y -= dt * 0.6;
    this.coreCrystal.rotation.y += dt * 0.6;
    this.coreCrystal.position.y = 7 + Math.sin(t * 1.4) * 0.35;
    this.coreRings.forEach((r, i) => { r.rotation.x = t * (0.5 + i * 0.3) + i; r.rotation.y = t * (0.35 - i * 0.2); r.position.y = this.coreCrystal.position.y; });
    this.coreLight.intensity = 70 + Math.sin(t * 3) * 15;
    for (const o of this.depositObjs) if (o.rig.visible) o.drill.rotation.y += dt * (3 + (o.level || 0) * 0.3);
    for (const o of this.campObjs) o.egg.material.emissiveIntensity = 1.1 + Math.sin(t * 2 + o.def.id) * 0.5;
    this.baseMat.emissiveIntensity = 0.55 + Math.sin(t * 1.5) * 0.1;
    if (this.dust) this.dust.position.y = Math.sin(t * 0.2) * 1.5;
    for (const l of this.flashLights) {
      if (l.userData.life > 0) {
        l.userData.life -= dt;
        l.intensity = Math.max(0, l.userData.life / l.userData.max) * l.userData.I;
      } else l.intensity = 0;
    }
    // тіні слідують за камерою
    if (focus) {
      this.sun.position.set(focus.x - 60, 90, focus.z + 40);
      this.sun.target.position.set(focus.x, 0, focus.z);
    }
  }

  render() {
    if (this.q.bloom) this.composer.render();
    else this.renderer.render(this.scene, this.camera);
  }
}
export { WORLD };
