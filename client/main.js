// Клієнт: мережа, керування, синхронізація сутностей, камера, оверлей.
import * as THREE from 'three';
import { World, terrainHeight } from './scene.js';
import { makeHero, makeTower, setTowerLevel, makeEnemy, makeLoot, makeGhost, makeDrone } from './models.js';
import { FX, PCOLORS } from './fx.js';
import { Audio } from './audio.js';
import { UI } from './ui.js';
import {
  ENEMIES, ENEMY_KINDS, TOWERS, TOWER_TYPES, DEPOSITS, CAMPS, LANES, BASE_R, ABILITIES, DIFFICULTY,
  canBuildAt, snap as snapGrid, towerStats, fmt, RARITIES, TOWER_MAX_LVL, towerUpgradeCost, extractorUpgradeCost, EXTRACTOR_COST,
} from '/shared/data.js';

const $ = (id) => document.getElementById(id);
const params = new URLSearchParams(location.search);
const hex = (n) => '#' + n.toString(16).padStart(6, '0');

// ───────────── налаштування ─────────────
const DEFAULTS = { quality: 'high', vol: 0.5, music: 0.35, dmgNums: true, shake: true, fps: false };
let settings = { ...DEFAULTS };
try { settings = { ...DEFAULTS, ...JSON.parse(localStorage.getItem('bastion.settings') || '{}') }; } catch { /* */ }

const G = {
  name: null, host: false, me: null, snap: null,
  world: { towers: [], extractors: [], research: {}, camps: [], cfg: { prepTime: 120, difficulty: 'normal' }, stats: {} },
  buildType: null, settings, localCds: { q: 0, e: 0, r: 0 },
  send(obj) { if (ws && ws.readyState === 1) ws.send(JSON.stringify(obj)); },
  selectBuild(type) {
    const d = TOWERS[type];
    if (d.unlock && !G.world.research[d.unlock]) { ui.feed(`«${d.name}» потребує дослідження (J).`, '#f66'); audio.play('error'); return; }
    G.buildType = G.buildType === type ? null : type;
    ui.selectTower(null);
    updateGhost();
    audio.play('ui');
  },
  applySettings(patch) {
    Object.assign(settings, patch);
    try { localStorage.setItem('bastion.settings', JSON.stringify(settings)); } catch { /* */ }
    if (patch.quality) { world.setQuality(patch.quality); fx.mul = world.q.particles; }
    audio.setVolume(settings.vol);
    audio.setMusic(settings.music);
    $('fps').classList.toggle('hidden', !settings.fps);
  },
};
window.G = G;

const world = new World($('gl'), settings.quality);
const fx = new FX(world);
const audio = (G.audio = new Audio());
const ui = new UI(G);
G.applySettings({});
const overlay = $('overlay');
const octx = overlay.getContext('2d');

// ───────────── мережа ─────────────
let ws = null;
let everConnected = false;
function connect() {
  ws = new WebSocket(`ws://${location.host}/ws`);
  ws.onopen = () => {
    everConnected = true;
    $('disconnected').classList.add('hidden');
    G.send({ t: 'join', name: params.get('user') || localStorage.getItem('bastion.name') || 'Гравець' });
  };
  ws.onmessage = (e) => onMsg(JSON.parse(e.data));
  ws.onclose = () => {
    if (!everConnected) { $('lobby-status').textContent = 'Сервер недоступний. Повтор…'; setTimeout(connect, 1500); return; }
    $('disconnected').classList.remove('hidden');
    setTimeout(async () => {
      if (window.bastion) await window.bastion.reconnect();
      else location.reload();
    }, 1500 + Math.random() * 1500);
  };
}
connect();

function onMsg(m) {
  switch (m.t) {
    case 'hello': onHello(m); break;
    case 'world': onWorld(m); break;
    case 'me': {
      const prevLvl = G.me?.profile.level;
      G.me = m;
      if (prevLvl && m.profile.level > prevLvl) { ui.banner(`РІВЕНЬ ${m.profile.level}`, 'отримано очко таланту [K]', '#ffd23a', 2.5); audio.play('lvl'); }
      ui.renderPanel();
      break;
    }
    case 'snap': onSnap(m); break;
    case 'msg': ui.feed(m.text, m.color); break;
    case 'respawn': me.x = m.x; me.z = m.z; me.vx = me.vz = 0; break;
  }
}

function onHello(m) {
  G.name = m.name;
  G.host = m.host;
  if (m.needSetup) {
    $('lobby').classList.remove('hidden');
    $('hud').classList.add('hidden');
    if (m.host) {
      $('lobby-status').textContent = `Вітаємо, ${m.name}! Ви — хост. Налаштуйте гру:`;
      $('setup').classList.remove('hidden');
      if (m.save) {
        $('save-info').innerHTML = `Знайдено збереження: <b>хвиля ${m.save.wave}</b> · ${DIFFICULTY[m.save.difficulty]?.name || ''} · герої: ${m.save.heroes.join(', ') || '—'}`;
        $('btn-continue').classList.remove('hidden');
        $('new-warn').classList.remove('hidden');
      } else $('save-info').textContent = 'Збережень ще немає — почніть нову гру.';
    } else {
      $('lobby-status').textContent = `Вітаємо, ${m.name}! Чекаємо, поки хост почне гру…`;
    }
    return;
  }
  $('lobby').classList.add('hidden');
  $('hud').classList.remove('hidden');
  ui.feed('Ласкаво просимо на форпост! Відбивайте напади, качайтесь у зонах за периметром.', '#3ae0ff');
}
// лобі
let setupDiff = 'normal', setupPrep = 120;
const diffHints = { easy: 'Вороги посилюються на ~7% за хвилю. Для розслабленої гри.', normal: 'Вороги посилюються на ~9% за хвилю. Потрібен регулярний грінд.', hard: 'Вороги посилюються на ~11% за хвилю. Для досвідчених.' };
$('diff-hint').textContent = diffHints.normal;
document.querySelectorAll('#diff button').forEach((b) => b.addEventListener('click', () => {
  setupDiff = b.dataset.v;
  document.querySelectorAll('#diff button').forEach((x) => x.classList.toggle('on', x === b));
  $('diff-hint').textContent = diffHints[setupDiff];
}));
document.querySelectorAll('#prep button').forEach((b) => b.addEventListener('click', () => {
  setupPrep = +b.dataset.v;
  document.querySelectorAll('#prep button').forEach((x) => x.classList.toggle('on', x === b));
}));
$('btn-new').onclick = () => { audio.init(); G.send({ t: 'setup', mode: 'new', difficulty: setupDiff, prepTime: setupPrep }); };
$('btn-continue').onclick = () => { audio.init(); G.send({ t: 'setup', mode: 'continue', prepTime: setupPrep }); };

// ───────────── сутності ─────────────
const heroes = new Map();   // name -> {obj, x, z, rot, drone}
const enemies = new Map();  // id -> {obj, x, z, tx, tz, rot, ...}
const towers = new Map();   // id -> {obj, data, rot, laser}
const loots = new Map();
const floaters = [];
const me = { x: 0, z: 10, vx: 0, vz: 0, rot: 0, ax: 0, az: 0, dashT: 0, dashFrom: null, dashTo: null };

function onWorld(m) {
  G.world = m;
  // вежі
  const seen = new Set();
  for (const t of m.towers) {
    seen.add(t.id);
    let o = towers.get(t.id);
    if (!o) {
      const obj = makeTower(t.type);
      obj.position.set(t.x, 0, t.z);
      world.scene.add(obj);
      o = { obj, data: t, rot: 0, level: 0, recoil: 0 };
      towers.set(t.id, o);
      fx.emit(t.x, 1, t.z, TOWERS[t.type].color, 30, { speed: 6, life: 0.6, size: 0.8 });
      fx.ring(t.x, t.z, 3, TOWERS[t.type].color, 0.5);
    }
    if (o.level !== t.level) { setTowerLevel(o.obj, t.level); o.level = t.level; }
    o.data = t;
  }
  for (const [id, o] of towers) if (!seen.has(id)) { world.scene.remove(o.obj); if (o.laser) world.scene.remove(o.laser); towers.delete(id); }
  world.setExtractors(m.extractors);
  ui.renderPanel();
  ui.updateBuildBar();
}

function ensureHero(p) {
  let h = heroes.get(p.n);
  if (!h) {
    const obj = makeHero(p.c);
    world.scene.add(obj);
    h = { obj, x: p.x, z: p.z, rot: p.r, drone: null, walk: 0, px: p.x, pz: p.z, color: p.c };
    heroes.set(p.n, h);
  }
  return h;
}

let lastSnapAt = 0;
function onSnap(s) {
  const prev = G.snap;
  G.snap = s;
  lastSnapAt = performance.now();
  // герої
  const names = new Set();
  for (const p of s.players) {
    names.add(p.n);
    const h = ensureHero(p);
    h.data = p;
    if (p.n !== G.name) { h.tx = p.x; h.tz = p.z; h.trot = p.r; }
    h.obj.visible = !p.d;
    if (p.dr && !h.drone) { h.drone = makeDrone(p.c); world.scene.add(h.drone); }
    if (p.n === G.name) {
      for (const k of ['q', 'e', 'r']) if (!(G.localCds[k] > p.cds[k] + 0.4)) G.localCds[k] = p.cds[k];
      if (prev) {
        const old = prev.players.find((x) => x.n === G.name);
        if (old && p.hp < old.hp - 1 && !p.d) { audio.play('hurt'); hurtFlash = Math.min(1, hurtFlash + 0.35); }
      }
    }
  }
  for (const [n, h] of heroes) if (!names.has(n)) { world.scene.remove(h.obj); if (h.drone) world.scene.remove(h.drone); heroes.delete(n); }

  // вороги
  const seen = new Set();
  for (const a of s.enemies) {
    const [id, ki, x, z, hp, sh, slow, rot, lvl] = a;
    seen.add(id);
    let e = enemies.get(id);
    if (!e) {
      const kind = ENEMY_KINDS[ki];
      const obj = makeEnemy(kind);
      world.scene.add(obj);
      e = { obj, kind, x, z, rot, seed: Math.random() * 10, def: ENEMIES[kind] };
      enemies.set(id, e);
      obj.position.set(x, terrainHeight(x, z), z);
      if (!e.def.wild) fx.emit(x, 2, z, 0xff2266, 12, { speed: 4, life: 0.5, size: 0.8 });
    }
    if (e.hp != null && hp < e.hp) e.obj.userData.hitFlash = 0.12;
    e.tx = x; e.tz = z; e.trot = rot; e.hp = hp; e.sh = sh; e.slow = slow; e.lvl = lvl;
  }
  for (const [id, e] of enemies) if (!seen.has(id)) { world.scene.remove(e.obj); enemies.delete(id); }

  // вежі (поворот, лазер)
  for (const [id, rot, laserT, aura, ramp] of s.towers) {
    const o = towers.get(id);
    if (!o) continue;
    o.trot = rot; o.laserT = laserT; o.ramp = ramp;
    o.obj.userData.auraRing.visible = !!aura;
  }
  // лут
  const ls = new Set();
  for (const [id, x, z, r] of s.loot) {
    ls.add(id);
    if (!loots.has(id)) {
      const obj = makeLoot(r);
      obj.position.set(x, terrainHeight(x, z), z);
      world.scene.add(obj);
      loots.set(id, { obj, r });
    }
  }
  for (const [id, l] of loots) if (!ls.has(id)) { world.scene.remove(l.obj); loots.delete(id); }

  for (const ev of s.ev) handleEvent(ev);
  ui.updateHUD();
  world.setCoreHealth(s.core[0] / s.core[1]);
  audio.setIntensity(s.phase === 'wave' ? 1 : 0);
}

let hurtFlash = 0;
function handleEvent(ev) {
  const near = (x, z) => Math.max(0.15, 1 - Math.hypot(x - me.x, z - me.z) / 90);
  switch (ev.e) {
    case 'ps': fx.spawnProjectile(ev); if (ev.k === 'hero') audio.play('shoot', near(ev.p[0], ev.p[2])); else if (ev.k === 'blaster') audio.play('blaster', near(ev.p[0], ev.p[2])); else if (ev.k === 'mortar') audio.play('mortar', near(ev.p[0], ev.p[2])); break;
    case 'ph': fx.endProjectile(ev); break;
    case 'beam': {
      const col = PCOLORS[ev.k] || 0xffffff;
      fx.beam(ev.a, ev.b, col, ev.k === 'rail' ? 0.35 : ev.k === 'drone' ? 0.06 : 0.18, ev.k === 'rail' ? 0.3 : 0.12);
      fx.emit(ev.b[0], ev.b[1], ev.b[2], col, ev.k === 'rail' ? 4 : 8, { speed: 5, life: 0.35, size: 0.6 });
      if (ev.k === 'rail') { audio.play('rail', near(ev.a[0], ev.a[2])); for (let i = 0; i < 12; i++) { const t = i / 12; fx.emit(ev.a[0] + (ev.b[0] - ev.a[0]) * t, 1.5, ev.a[2] + (ev.b[2] - ev.a[2]) * t, 0xffee55, 1, { speed: 1, life: 0.5, size: 0.5 }); } }
      if (ev.k === 'cryo') audio.play('cryo', near(ev.a[0], ev.a[2]));
      break;
    }
    case 'chain': fx.lightning(ev.p, 0xbb77ff); audio.play('tesla', near(ev.p[0][0], ev.p[0][2])); break;
    case 'tf': {
      const o = towers.get(ev.id);
      if (o) {
        o.recoil = 1;
        const mz = o.obj.userData.muzzles[0];
        if (mz) {
          const p = o.obj.userData.head.localToWorld(mz.clone());
          fx.emit(p.x, p.y, p.z, TOWERS[o.data.type].color, 3, { speed: 3, life: 0.15, size: 0.9 });
        }
      }
      break;
    }
    case 'boom': {
      const col = { gren: 0xff9933, orbit: 0x66ccff, mortar: 0xff7733 }[ev.k] || 0xff8833;
      fx.explosion(ev.x, ev.z, ev.r, col, ev.k === 'orbit' ? 0 : ev.k === 'gren' ? 1.4 : 1);
      audio.play('boom', near(ev.x, ev.z));
      if (ev.k === 'orbit') audio.play('bigboom');
      if (ev.k !== 'mortar') shake(0.35);
      break;
    }
    case 'die': {
      const e = enemies.get(-1);
      void e;
      const d = ENEMIES[ENEMY_KINDS[ev.k]];
      const big = d.boss ? 4 : d.size;
      fx.emit(ev.x, 1 + (d.flying ? 3 : 0), ev.z, d.color, 18 * big, { speed: 7 * Math.sqrt(big), life: 0.7, size: 0.7 * big, grav: 1, up: 1 });
      fx.emit(ev.x, 1, ev.z, 0xffffff, 6 * big, { speed: 3, life: 0.25, size: 1.2 * big });
      if (d.boss) { fx.explosion(ev.x, ev.z, 12, d.color, 3); audio.play('bigboom'); shake(1); ui.banner('БОСА ЗНИЩЕНО', d.name, '#ffa02a', 3); }
      else audio.play('die', near(ev.x, ev.z));
      if (ev.c > 0 && Math.hypot(ev.x - me.x, ev.z - me.z) < 45) floaters.push({ x: ev.x, y: 2.5, z: ev.z, text: `+${fmt(ev.c)}◈`, color: '#ffd23a', t: 1.1, size: 13 });
      break;
    }
    case 'dmg':
      if (ev.n === G.name && settings.dmgNums) floaters.push({ x: ev.x + (Math.random() - 0.5), y: 2.5, z: ev.z, text: fmt(ev.v), color: ev.c ? '#ffd23a' : '#ffffff', t: 0.8, size: ev.c ? 20 : 14 });
      if (ev.n === G.name) audio.play('hit', 0.6);
      break;
    case 'spark': fx.emit(ev.x, ev.y, ev.z, ev.c ?? 0xffffff, 4, { speed: 5, life: 0.25, size: 0.5 }); break;
    case 'dash': fx.dash(ev.a, ev.b, ev.c); break;
    case 'gren': fx.grenade(ev.a, ev.b); break;
    case 'orbit': fx.orbital(ev.x, ev.z); break;
    case 'core': fx.explosion(ev.x, ev.z, 3, 0xff2244, 0.6); fx.emit(0, 7, 0, 0xff2244, 20, { speed: 8, life: 0.5, size: 1 }); audio.play('core'); shake(0.25); break;
    case 'swing': fx.ring(ev.x, ev.z, 1.5 + ev.s, 0xff5533, 0.3, 0.3); break;
    case 'stomp': fx.ring(ev.x, ev.z, ev.r, 0xff2222, 0.8, 0.3); fx.emit(ev.x, 0.5, ev.z, 0x886655, 50, { speed: 14, life: 0.8, size: 1.6, up: 0.3 }); audio.play('stomp'); shake(0.6); break;
    case 'heal': fx.ring(ev.x, ev.z, 7, 0x44ffaa, 0.6, 0.4); fx.emit(ev.x, 1.5, ev.z, 0x44ffaa, 8, { speed: 2, life: 0.8, size: 0.6, grav: -0.3 }); break;
    case 'hdie': fx.explosion(ev.x, ev.z, 4, 0xff4a5a, 1); audio.play('boom'); break;
    case 'lvl': fx.emit(ev.x, 1, ev.z, 0xffd23a, 60, { speed: 4, life: 1.2, size: 0.8, grav: -0.6, up: 1 }); fx.ring(ev.x, ev.z, 4, 0xffd23a, 0.8); break;
    case 'pick': {
      if (ev.n === G.name) { audio.play(ev.r >= 4 ? 'legend' : 'pick'); floaters.push({ x: ev.x, y: 3, z: ev.z, text: RARITIES[ev.r].name + '!', color: RARITIES[ev.r].color, t: 1.6, size: 16 }); }
      fx.emit(ev.x, 1, ev.z, new THREE.Color(RARITIES[ev.r].color).getHex(), 20, { speed: 4, life: 0.6, size: 0.7, grav: -0.5 });
      break;
    }
    case 'build': audio.play('build'); break;
    case 'up': fx.emit(ev.x, 2, ev.z, 0x4dff7a, 30, { speed: 3, life: 0.9, size: 0.6, grav: -0.8 }); fx.ring(ev.x, ev.z, 3, 0x4dff7a, 0.6); audio.play('upgrade'); break;
    case 'sell': fx.emit(ev.x, 1, ev.z, 0xffd23a, 25, { speed: 5, life: 0.7, size: 0.6 }); break;
    case 'wave': ui.banner(`ХВИЛЯ ${ev.w}`, ev.w % 10 === 0 ? 'Увага: бос!' : 'вороги атакують', '#ff8a3a', 3); audio.play('wave'); break;
    case 'boss': ui.banner('⚠ БОС ⚠', ENEMIES[ev.k].name, '#ff3355', 3.5); audio.play('boss'); shake(0.5); break;
    case 'clear': ui.banner('ХВИЛЮ ВІДБИТО', 'час на прокачку', '#4dff7a', 3); audio.play('clear'); break;
    case 'defeat': ui.banner('ЯДРО ВПАЛО', 'хвиля повториться', '#ff3355', 4); audio.play('defeat'); shake(1); break;
    case 'camp': fx.ring(ev.x, ev.z, 10, 0xffd23a, 1); ui.feed('Табір зачищено! Він відродиться сильнішим.', '#ffd23a'); break;
  }
}
function shake(a) { if (settings.shake) world.shake = Math.max(world.shake || 0, a); }

// ───────────── введення ─────────────
const keys = new Set();
const mouse = { x: 0, y: 0, nx: 0, ny: 0, down: false, world: new THREE.Vector3() };
let zoom = 1;
const raycaster = new THREE.Raycaster();
const groundPlane = new THREE.Plane(new THREE.Vector3(0, 1, 0), 0);

window.addEventListener('keydown', (e) => {
  audio.init();
  const chat = $('chat');
  if (document.activeElement === chat) {
    if (e.key === 'Enter') { if (chat.value.trim()) G.send({ t: 'chat', text: chat.value.trim() }); chat.value = ''; chat.classList.add('hidden'); chat.blur(); }
    if (e.key === 'Escape') { chat.value = ''; chat.classList.add('hidden'); chat.blur(); }
    return;
  }
  if (!G.snap) return;
  const k = e.key.toLowerCase();
  const code = e.code;
  if (code === 'Tab') { e.preventDefault(); ui.openPanel('hero'); return; }
  if (code === 'Escape') {
    if (G.buildType) { G.buildType = null; updateGhost(); return; }
    if (ui.selTower) { ui.selectTower(null); return; }
    if (ui.isOpen()) { ui.closePanel(); return; }
    ui.openPanel('settings');
    return;
  }
  if (code === 'Enter') { $('chat').classList.remove('hidden'); $('chat').focus(); return; }
  if (code === 'KeyK') return ui.openPanel('talents');
  if (code === 'KeyJ') return ui.openPanel('research');
  if (code === 'KeyM') { $('bigmap').classList.toggle('hidden'); drawMap($('bigmap-c'), 132); return; }
  if (code === 'KeyG') { G.send({ t: 'ready' }); return; }
  if (code === 'KeyF') { tryExtractor(); return; }
  if (code === 'KeyU' && ui.selTower) { G.send({ t: 'upgrade', id: ui.selTower }); return; }
  if (/^Digit[1-6]$/.test(code)) { G.selectBuild(TOWER_TYPES[+code.slice(5) - 1]); return; }
  if (code === 'KeyQ' || code === 'KeyE' || code === 'KeyR') { useAbility(code.slice(3).toLowerCase()); return; }
  keys.add(code);
  void k;
});
window.addEventListener('keyup', (e) => keys.delete(e.code));
window.addEventListener('blur', () => { keys.clear(); mouse.down = false; });
window.addEventListener('mousemove', (e) => {
  mouse.x = e.clientX; mouse.y = e.clientY;
  mouse.nx = (e.clientX / window.innerWidth) * 2 - 1;
  mouse.ny = -(e.clientY / window.innerHeight) * 2 + 1;
});
const gl = $('gl');
gl.addEventListener('mousedown', (e) => {
  audio.init();
  if (e.button === 0) {
    if (G.buildType) { placeTower(); return; }
    mouse.down = true;
  } else if (e.button === 2) {
    if (G.buildType) { G.buildType = null; updateGhost(); return; }
    const t = pickTower();
    ui.selectTower(t ? t : null);
    if (t) audio.play('ui');
  }
});
window.addEventListener('mouseup', (e) => { if (e.button === 0) mouse.down = false; });
window.addEventListener('contextmenu', (e) => e.preventDefault());
gl.addEventListener('wheel', (e) => { zoom = THREE.MathUtils.clamp(zoom * (e.deltaY > 0 ? 1.1 : 0.9), 0.55, 1.7); }, { passive: true });

function updateMouseWorld() {
  raycaster.setFromCamera(new THREE.Vector2(mouse.nx, mouse.ny), world.camera);
  raycaster.ray.intersectPlane(groundPlane, mouse.world);
}
function pickTower() {
  let best = null, bd = 2.8;
  for (const [id, o] of towers) {
    const d = Math.hypot(o.data.x - mouse.world.x, o.data.z - mouse.world.z);
    if (d < bd) { bd = d; best = id; }
  }
  return best;
}

// примара будівництва
let ghost = null, ghostType = null;
function updateGhost() {
  if (ghost && ghostType !== G.buildType) { world.scene.remove(ghost); ghost = null; }
  if (G.buildType && !ghost) {
    ghost = makeGhost(G.buildType);
    ghostType = G.buildType;
    world.scene.add(ghost);
  }
  world.gridOverlay.visible = !!G.buildType;
  ui.updateBuildBar();
}
function placeTower() {
  const x = snapGrid(mouse.world.x), z = snapGrid(mouse.world.z);
  if (!canBuildAt(x, z) || G.world.towers.some((t) => t.x === x && t.z === z)) { audio.play('error'); return; }
  G.send({ t: 'build', type: G.buildType, x, z });
  if (!keys.has('ShiftLeft') && !keys.has('ShiftRight')) { /* лишаємось у режимі будівництва для швидкості */ }
}

function nearestDeposit() {
  let best = null, bd = 12;
  for (const d of DEPOSITS) { const dd = Math.hypot(d.x - me.x, d.z - me.z); if (dd < bd) { bd = dd; best = d; } }
  return best;
}
function tryExtractor() {
  const d = nearestDeposit();
  if (!d) return;
  G.send({ t: 'extract', id: d.id });
}

function useAbility(k) {
  const meP = G.snap?.players.find((p) => p.n === G.name);
  if (!meP || meP.d || !G.me) return;
  if (G.localCds[k] > 0) { audio.play('error'); return; }
  G.localCds[k] = ABILITIES[k].cd * (1 - G.me.stats.cdr);
  if (k === 'q') {
    let dx = 0, dz = 0;
    if (keys.has('KeyW')) dz -= 1; if (keys.has('KeyS')) dz += 1; if (keys.has('KeyA')) dx -= 1; if (keys.has('KeyD')) dx += 1;
    if (!dx && !dz) { dx = mouse.world.x - me.x; dz = mouse.world.z - me.z; }
    const l = Math.hypot(dx, dz) || 1;
    const from = { x: me.x, z: me.z };
    const to = clampPos(me.x + (dx / l) * 11, me.z + (dz / l) * 11);
    me.dashFrom = from; me.dashTo = to; me.dashT = 0.16;
    G.send({ t: 'ab', k, fx: from.x, fz: from.z, x: to.x, z: to.z });
    audio.play('dash');
  } else {
    G.send({ t: 'ab', k, x: mouse.world.x, z: mouse.world.z });
    if (k === 'e') audio.play('mortar');
  }
}
function clampPos(x, z) {
  const r = Math.hypot(x, z);
  if (r > 126) { x *= 126 / r; z *= 126 / r; }
  return { x, z };
}

function moveMe(dt) {
  const meP = G.snap?.players.find((p) => p.n === G.name);
  if (!meP || meP.d || !G.me) return;
  if (me.dashT > 0) {
    me.dashT -= dt;
    const t = 1 - Math.max(0, me.dashT) / 0.16;
    me.x = me.dashFrom.x + (me.dashTo.x - me.dashFrom.x) * t;
    me.z = me.dashFrom.z + (me.dashTo.z - me.dashFrom.z) * t;
  } else if (document.activeElement !== $('chat')) {
    let dx = 0, dz = 0;
    if (keys.has('KeyW') || keys.has('ArrowUp')) dz -= 1;
    if (keys.has('KeyS') || keys.has('ArrowDown')) dz += 1;
    if (keys.has('KeyA') || keys.has('ArrowLeft')) dx -= 1;
    if (keys.has('KeyD') || keys.has('ArrowRight')) dx += 1;
    const l = Math.hypot(dx, dz);
    const sp = G.me.stats.speed;
    const tvx = l ? (dx / l) * sp : 0, tvz = l ? (dz / l) * sp : 0;
    const a = 1 - Math.exp(-dt * 14);
    me.vx += (tvx - me.vx) * a; me.vz += (tvz - me.vz) * a;
    me.x += me.vx * dt; me.z += me.vz * dt;
  }
  // колізії: ядро, вежі, межі світу
  const push = (cx, cz, r) => {
    const dx = me.x - cx, dz = me.z - cz, d = Math.hypot(dx, dz);
    if (d < r && d > 0.001) { me.x = cx + (dx / d) * r; me.z = cz + (dz / d) * r; }
  };
  push(0, 0, 7.6);
  for (const t of G.world.towers) push(t.x, t.z, 2.1);
  const c = clampPos(me.x, me.z); me.x = c.x; me.z = c.z;
  me.ax = mouse.world.x; me.az = mouse.world.z;
  me.rot = Math.atan2(me.ax - me.x, me.az - me.z);
}

let sendAcc = 0;
function sendInput(dt) {
  sendAcc += dt;
  if (sendAcc < 0.045) return;
  sendAcc = 0;
  const firing = mouse.down && !G.buildType && !ui.isOpen();
  G.send({ t: 'in', x: +me.x.toFixed(2), z: +me.z.toFixed(2), ax: +me.ax.toFixed(2), az: +me.az.toFixed(2), fire: firing, rot: +me.rot.toFixed(2) });
}

// ───────────── оновлення сцени ─────────────
const tmpV = new THREE.Vector3();
function lerpAngle(a, b, t) { let d = b - a; while (d > Math.PI) d -= Math.PI * 2; while (d < -Math.PI) d += Math.PI * 2; return a + d * t; }

function updateEntities(dt, t) {
  const k = 1 - Math.exp(-dt * 12);
  // герої
  for (const [n, h] of heroes) {
    const isMe = n === G.name;
    if (isMe) { h.x = me.x; h.z = me.z; h.rot = me.rot; }
    else if (h.tx != null) { h.x += (h.tx - h.x) * k; h.z += (h.tz - h.z) * k; h.rot = lerpAngle(h.rot, h.trot, k); }
    const sp = Math.hypot(h.x - h.px, h.z - h.pz) / Math.max(dt, 1e-3);
    h.px = h.x; h.pz = h.z;
    h.walk += dt * Math.min(sp, 14) * 0.9;
    const o = h.obj, u = o.userData;
    o.position.set(h.x, terrainHeight(h.x, h.z), h.z);
    o.rotation.y = h.rot;
    const sw = Math.min(1, sp / 6);
    u.legs[0].rotation.x = Math.sin(h.walk) * 0.7 * sw;
    u.legs[1].rotation.x = -Math.sin(h.walk) * 0.7 * sw;
    u.body.position.y = Math.abs(Math.sin(h.walk)) * 0.08 * sw;
    const firing = h.data && h.data.f;
    u.muzzle.visible = firing && Math.sin(t * 60) > 0;
    u.arms.position.z = firing ? Math.sin(t * 50) * 0.04 : 0;
    u.ring.material.opacity = 0.4 + Math.sin(t * 3) * 0.2;
    if (h.drone) {
      h.drone.visible = o.visible;
      h.drone.position.set(h.x - 1.6 + Math.sin(t * 1.3) * 0.4, o.position.y + 3.2 + Math.sin(t * 2.2) * 0.3, h.z - 1.2 + Math.cos(t * 1.1) * 0.4);
      h.drone.rotation.y = t * 2;
    }
  }
  // вороги
  for (const e of enemies.values()) {
    e.x += (e.tx - e.x) * k; e.z += (e.tz - e.z) * k;
    e.rot = lerpAngle(e.rot, e.trot, k);
    const o = e.obj, u = o.userData;
    const fly = u.flying;
    const ground = terrainHeight(e.x, e.z);
    const y = fly ? ground + (u.boss ? 6 : 4) + Math.sin(t * 3 + e.seed) * 0.4 : ground;
    o.position.set(e.x, y, e.z);
    o.rotation.y = e.rot;
    const ph = t * (e.def.speed * 1.6) + e.seed;
    u.legs.forEach((l, i) => (l.rotation.x = Math.sin(ph + i * Math.PI * 0.5 + (i % 2) * Math.PI) * 0.6));
    if (!fly) u.body.position.y = Math.abs(Math.sin(ph)) * 0.08 * e.def.size;
    u.wings.forEach((w, i) => {
      if (u.kind === 'flyer') w.rotation.z = Math.sin(t * 28 + e.seed) * 0.7 * (i ? 1 : -1);
      else w.rotation.y += dt * (1.5 + i);
    });
    if (u.shield) { u.shield.visible = e.sh > 0.02; u.shield.material.opacity = 0.08 + e.sh * 0.18 + Math.sin(t * 6) * 0.03; }
    if (u.hitFlash > 0) { u.hitFlash -= dt; u.body.scale.setScalar((u.boss ? 1.15 : e.def.size) * 1.06); } else u.body.scale.setScalar(u.boss ? (u.kind === 'boss_sentinel' ? 1 : 1.15) : e.def.size);
    if (e.slow && Math.random() < dt * 6) fx.emit(e.x, y + 1, e.z, 0x9fd6ff, 1, { speed: 1, life: 0.5, size: 0.5 });
  }
  // вежі
  for (const o of towers.values()) {
    const u = o.obj.userData;
    if (o.trot != null) { o.rot = lerpAngle(o.rot, o.trot, 1 - Math.exp(-dt * 10)); u.head.rotation.y = o.rot; }
    if (u.type === 'tesla') u.head.rotation.y = t * 0.8;
    o.recoil = Math.max(0, (o.recoil || 0) - dt * 6);
    u.head.position.z = 0;
    u.head.children.forEach(() => {});
    u.head.position.y = 2.6 - o.recoil * 0.12;
    // лазер
    const tgt = o.laserT > 0 ? enemies.get(o.laserT) : null;
    if (tgt) {
      if (!o.laser) {
        o.laser = new THREE.Mesh(new THREE.CylinderGeometry(1, 1, 1, 8, 1, true).rotateX(Math.PI / 2), new THREE.MeshBasicMaterial({ color: new THREE.Color(0xff3388).multiplyScalar(3), transparent: true, blending: THREE.AdditiveBlending, depthWrite: false }));
        world.scene.add(o.laser);
      }
      const a = u.head.localToWorld(u.muzzles[0].clone());
      const b = tmpV.set(tgt.x, tgt.obj.position.y + (tgt.obj.userData.flying ? 0 : 1), tgt.z);
      o.laser.visible = true;
      o.laser.position.copy(a).lerp(b, 0.5);
      o.laser.lookAt(b);
      const w = 0.08 + (o.ramp || 0) * 0.08 + Math.sin(t * 40) * 0.02;
      o.laser.scale.set(w, w, a.distanceTo(b));
      if (Math.random() < 0.5) fx.emit(b.x, b.y, b.z, 0xff3388, 1, { speed: 4, life: 0.25, size: 0.6 });
    } else if (o.laser) o.laser.visible = false;
  }
  // лут
  for (const l of loots.values()) {
    l.obj.userData.gem.rotation.y += dt * 2;
    l.obj.userData.gem.position.y = 1 + Math.sin(t * 2.5) * 0.2;
  }
  // примара
  if (ghost) {
    const x = snapGrid(mouse.world.x), z = snapGrid(mouse.world.z);
    ghost.position.set(x, 0, z);
    const ok = canBuildAt(x, z) && !G.world.towers.some((tt) => tt.x === x && tt.z === z);
    const col = ok ? 0x3ae0ff : 0xff3344;
    ghost.traverse((m) => { if (m.isMesh) m.material.color.setHex(col); });
    const r = towerStats(G.buildType, 1, G.world.research).range;
    ghost.userData.range.scale.setScalar(r);
    ghost.userData.fill.scale.setScalar(r);
  }
}

// ───────────── камера ─────────────
const camPos = new THREE.Vector3(0, 40, 30);
const camLook = new THREE.Vector3();
function updateCamera(dt) {
  const h = 30 * zoom, back = 21 * zoom;
  // легкий зсув у бік курсора
  const ox = THREE.MathUtils.clamp((mouse.world.x - me.x) * 0.18, -8, 8);
  const oz = THREE.MathUtils.clamp((mouse.world.z - me.z) * 0.18, -8, 8);
  const target = tmpV.set(me.x + ox, 0, me.z + oz);
  camLook.lerp(target, 1 - Math.exp(-dt * 6));
  camPos.set(camLook.x, camLook.y + h, camLook.z + back);
  world.camera.position.copy(camPos);
  if (world.shake > 0) {
    world.camera.position.x += (Math.random() - 0.5) * world.shake;
    world.camera.position.y += (Math.random() - 0.5) * world.shake;
    world.shake = Math.max(0, world.shake - dt * 2.5);
  }
  world.camera.lookAt(camLook);
}

// ───────────── оверлей (смужки здоров’я, імена, цифри) ─────────────
function project(x, y, z) {
  tmpV.set(x, y, z).project(world.camera);
  if (tmpV.z > 1) return null;
  return [(tmpV.x * 0.5 + 0.5) * overlay.width, (-tmpV.y * 0.5 + 0.5) * overlay.height];
}
function drawOverlay(dt) {
  if (overlay.width !== window.innerWidth || overlay.height !== window.innerHeight) { overlay.width = window.innerWidth; overlay.height = window.innerHeight; }
  const W = overlay.width, H = overlay.height;
  octx.clearRect(0, 0, W, H);
  octx.textAlign = 'center';
  octx.font = '600 12px Bahnschrift, Segoe UI, sans-serif';
  // вороги
  for (const e of enemies.values()) {
    const top = e.obj.position.y + (e.def.boss ? 9 : 2.6 * e.def.size + (e.def.wild ? 0.5 : 0));
    const p = project(e.x, top, e.z);
    if (!p || p[0] < -50 || p[0] > W + 50 || p[1] < -50 || p[1] > H + 50) continue;
    if (e.def.boss) continue; // бос — окрема велика смуга
    if (e.hp >= 0.999 && !e.def.wild && !(e.sh >= 0 && e.sh < 0.999)) continue;
    const w = e.def.wild ? 40 : 30 * Math.max(0.8, e.def.size);
    octx.fillStyle = 'rgba(0,0,0,.6)';
    octx.fillRect(p[0] - w / 2 - 1, p[1] - 1, w + 2, 6);
    octx.fillStyle = e.def.wild ? '#ffb02e' : '#ff3355';
    octx.fillRect(p[0] - w / 2, p[1], w * e.hp, 4);
    if (e.sh >= 0) { octx.fillStyle = '#6aa8ff'; octx.fillRect(p[0] - w / 2, p[1] - 3, w * e.sh, 2); }
    if (e.def.wild) { octx.fillStyle = '#ffd9a0'; octx.fillText(`${e.def.name} ${e.lvl}`, p[0], p[1] - 5); }
  }
  // бос: велика смуга вгорі
  const boss = [...enemies.values()].find((e) => e.def.boss);
  if (boss) {
    const w = Math.min(600, W * 0.45);
    const x = W / 2 - w / 2, y = 130;
    octx.fillStyle = 'rgba(0,0,0,.7)'; octx.fillRect(x - 2, y - 2, w + 4, 18);
    const gr = octx.createLinearGradient(x, 0, x + w, 0);
    gr.addColorStop(0, '#a0102a'); gr.addColorStop(1, '#ff3355');
    octx.fillStyle = gr; octx.fillRect(x, y, w * boss.hp, 14);
    if (boss.sh >= 0) { octx.fillStyle = '#8a6aff'; octx.fillRect(x, y + 10, w * boss.sh, 4); }
    octx.font = '700 15px Bahnschrift, Segoe UI, sans-serif';
    octx.fillStyle = '#fff'; octx.fillText(`☠ ${boss.def.name} · ${Math.round(boss.hp * 100)}%`, W / 2, y - 6);
    octx.font = '600 12px Bahnschrift, Segoe UI, sans-serif';
  }
  // імена героїв
  for (const [n, h] of heroes) {
    if (!h.obj.visible) continue;
    const p = project(h.x, h.obj.position.y + 3.2, h.z);
    if (!p) continue;
    octx.fillStyle = hex(h.color);
    octx.fillText(`${n} · ${h.data?.l ?? ''}`, p[0], p[1] - 8);
    if (n !== G.name && h.data) {
      octx.fillStyle = 'rgba(0,0,0,.6)'; octx.fillRect(p[0] - 21, p[1] - 4, 42, 5);
      octx.fillStyle = '#4dff7a'; octx.fillRect(p[0] - 20, p[1] - 3, 40 * (h.data.hp / h.data.mhp), 3);
    }
  }
  // мітки таборів і родовищ
  octx.font = '600 13px Bahnschrift, Segoe UI, sans-serif';
  CAMPS.forEach((c, i) => {
    if (Math.hypot(c.x - me.x, c.z - me.z) > 60) return;
    const p = project(c.x, terrainHeight(c.x, c.z) + 6, c.z);
    if (!p) return;
    const lv = G.world.camps[i]?.level ?? 1;
    octx.fillStyle = 'rgba(255,170,60,.85)';
    octx.fillText(`☠ Табір · рів. ${lv}`, p[0], p[1]);
  });
  DEPOSITS.forEach((d, i) => {
    if (Math.hypot(d.x - me.x, d.z - me.z) > 55) return;
    const p = project(d.x, terrainHeight(d.x, d.z) + 10, d.z);
    if (!p) return;
    const lv = G.world.extractors[i]?.level || 0;
    octx.fillStyle = 'rgba(195,140,255,.95)';
    octx.fillText(lv ? `◆ Шахта · рів. ${lv}` : '◆ Родовище кристалів', p[0], p[1]);
  });
  // вибрана вежа
  if (ui.selTower) {
    const t = towers.get(ui.selTower);
    if (t) {
      const r = towerStats(t.data.type, t.data.level, G.world.research).range;
      const pts = [];
      for (let i = 0; i <= 64; i++) { const a = (i / 64) * Math.PI * 2; pts.push(project(t.data.x + Math.cos(a) * r, 0.2, t.data.z + Math.sin(a) * r)); }
      octx.strokeStyle = 'rgba(58,224,255,.7)'; octx.lineWidth = 2; octx.beginPath();
      pts.forEach((p, i) => p && (i ? octx.lineTo(p[0], p[1]) : octx.moveTo(p[0], p[1])));
      octx.stroke();
    }
  }
  // плаваючі цифри
  for (let i = floaters.length - 1; i >= 0; i--) {
    const f = floaters[i];
    f.t -= dt; f.y += dt * 2.2;
    if (f.t <= 0) { floaters.splice(i, 1); continue; }
    const p = project(f.x, f.y, f.z);
    if (!p) continue;
    octx.globalAlpha = Math.min(1, f.t * 2.5);
    octx.font = `700 ${f.size}px Bahnschrift, Segoe UI, sans-serif`;
    octx.strokeStyle = 'rgba(0,0,0,.8)'; octx.lineWidth = 3;
    octx.strokeText(f.text, p[0], p[1]);
    octx.fillStyle = f.color; octx.fillText(f.text, p[0], p[1]);
  }
  octx.globalAlpha = 1;
  if (floaters.length > 120) floaters.splice(0, floaters.length - 120);
  // червоний спалах при отриманні шкоди
  if (hurtFlash > 0) {
    hurtFlash = Math.max(0, hurtFlash - dt * 2);
    const g = octx.createRadialGradient(W / 2, H / 2, Math.min(W, H) * 0.3, W / 2, H / 2, Math.max(W, H) * 0.7);
    g.addColorStop(0, 'rgba(255,0,30,0)'); g.addColorStop(1, `rgba(255,0,30,${hurtFlash * 0.5})`);
    octx.fillStyle = g; octx.fillRect(0, 0, W, H);
  }
}

// ───────────── карта ─────────────
function drawMap(cv, R = 132, focus = null) {
  const c = cv.getContext('2d');
  const S = cv.width;
  const cx = focus ? focus.x : 0, cz = focus ? focus.z : 0;
  const sc = S / 2 / R;
  const P = (x, z) => [S / 2 + (x - cx) * sc, S / 2 + (z - cz) * sc];
  c.clearRect(0, 0, S, S);
  c.save();
  c.beginPath(); c.arc(S / 2, S / 2, S / 2 - 1, 0, Math.PI * 2); c.clip();
  c.fillStyle = 'rgba(8,14,26,.95)'; c.fillRect(0, 0, S, S);
  // база
  let [bx, bz] = P(0, 0);
  c.fillStyle = 'rgba(58,224,255,.12)'; c.beginPath(); c.arc(bx, bz, BASE_R * sc, 0, Math.PI * 2); c.fill();
  // дороги
  c.strokeStyle = 'rgba(255,70,90,.6)'; c.lineWidth = Math.max(2, 4 * sc);
  for (const l of LANES) { c.beginPath(); l.forEach(([x, z], i) => { const [px, pz] = P(x, z); i ? c.lineTo(px, pz) : c.moveTo(px, pz); }); c.stroke(); }
  // родовища
  DEPOSITS.forEach((d, i) => { const [x, z] = P(d.x, d.z); c.fillStyle = G.world.extractors[i]?.level ? '#c38cff' : 'rgba(195,140,255,.35)'; c.fillRect(x - 4, z - 4, 8, 8); });
  // табори
  c.font = `600 ${S > 300 ? 15 : 9}px Bahnschrift, sans-serif`; c.textAlign = 'center';
  CAMPS.forEach((cp, i) => { const [x, z] = P(cp.x, cp.z); c.fillStyle = ['#ff9944', '#ff6633', '#ff3366', '#ffcc22'][cp.tier]; c.beginPath(); c.arc(x, z, 3.5, 0, Math.PI * 2); c.fill(); if (S > 300) c.fillText(G.world.camps[i]?.level ?? '', x, z - 7); });
  // вежі
  for (const t of G.world.towers) { const [x, z] = P(t.x, t.z); c.fillStyle = hex(TOWERS[t.type].color); c.fillRect(x - 2, z - 2, 4, 4); }
  // вороги
  for (const e of enemies.values()) {
    const [x, z] = P(e.x, e.z);
    c.fillStyle = e.def.wild ? 'rgba(255,170,60,.8)' : e.def.boss ? '#ff00ff' : '#ff3355';
    c.beginPath(); c.arc(x, z, e.def.boss ? 6 : 2, 0, Math.PI * 2); c.fill();
  }
  // лут
  for (const l of loots.values()) { const [x, z] = P(l.obj.position.x, l.obj.position.z); c.fillStyle = RARITIES[l.r].color; c.fillRect(x - 2, z - 2, 4, 4); }
  // ядро
  [bx, bz] = P(0, 0);
  c.fillStyle = '#3ae0ff'; c.beginPath(); c.arc(bx, bz, 5, 0, Math.PI * 2); c.fill();
  // герої
  for (const [n, h] of heroes) {
    const [x, z] = P(h.x, h.z);
    c.save(); c.translate(x, z); c.rotate(-h.rot + Math.PI);
    c.fillStyle = hex(h.color); c.strokeStyle = n === G.name ? '#fff' : '#000'; c.lineWidth = 1.5;
    c.beginPath(); c.moveTo(0, -7); c.lineTo(5, 5); c.lineTo(-5, 5); c.closePath(); c.fill(); c.stroke();
    c.restore();
  }
  c.restore();
}

// ───────────── підказки поруч ─────────────
function updatePrompt() {
  if (G.buildType) {
    const d = TOWERS[G.buildType];
    ui.prompt(`Будівництво: <b>${d.name}</b> — ЛКМ поставити · ПКМ/Esc скасувати`);
    return;
  }
  const d = nearestDeposit();
  if (d) { ui.prompt(ui.extractorPrompt(G.world.extractors[d.id] || { level: 0 }, d)); return; }
  ui.prompt(null);
}

// ───────────── головний цикл ─────────────
let last = performance.now(), fpsAcc = 0, fpsN = 0, panelAcc = 0, mapAcc = 0;
function frame(now) {
  requestAnimationFrame(frame);
  try { frameBody(now); } catch (e) { if (!frame.err) { frame.err = 1; console.error('frame error', e.message, e.stack); } }
}
function frameBody(now) {
  const dt = Math.min(0.05, (now - last) / 1000);
  last = now;
  const t = now / 1000;
  if (G.snap) {
    updateMouseWorld();
    moveMe(dt);
    sendInput(dt);
    for (const k of ['q', 'e', 'r']) G.localCds[k] = Math.max(0, G.localCds[k] - dt);
    updateEntities(dt, t);
    updateCamera(dt);
    updatePrompt();
    mapAcc += dt;
    if (mapAcc > 0.1) {
      mapAcc = 0;
      drawMap($('minimap'), 70, me);
      if (!$('bigmap').classList.contains('hidden')) drawMap($('bigmap-c'), 132);
    }
    panelAcc += dt;
    if (panelAcc > 0.5) { panelAcc = 0; ui.renderPanel(); }
  } else {
    // обертання камери в лобі
    world.camera.position.set(Math.cos(t * 0.05) * 70, 38, Math.sin(t * 0.05) * 70);
    world.camera.lookAt(0, 4, 0);
  }
  world.update(dt, me);
  fx.update(dt);
  world.render();
  drawOverlay(dt);
  fpsAcc += dt; fpsN++;
  if (fpsAcc > 1) { $('fps').textContent = `${Math.round(fpsN / fpsAcc)} FPS · ${enemies.size} ворогів`; fpsAcc = 0; fpsN = 0; }
  void lastSnapAt;
}
requestAnimationFrame(frame);
void EXTRACTOR_COST; void extractorUpgradeCost; void TOWER_MAX_LVL; void towerUpgradeCost;
