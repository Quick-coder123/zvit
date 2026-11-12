// Клієнт: мережа, керування, синхронізація сутностей, камера, оверлей.
import * as THREE from 'three';
import { World, terrainHeight } from './scene.js';
import { loadAssets } from './assets.js';
import { makeHero, makeTower, setTowerLevel, makeEnemy, makeLoot, makeGhost, makeDrone, burntMaterial } from './models.js';
import { FX } from './fx.js';
import { pushOutOfRects } from '/shared/collision.js';
import { Trauma, popScale } from './feel.js';
import { Audio } from './audio.js';
import { UI } from './ui.js';
import {
  ENEMIES, ENEMY_KINDS, TOWERS, TOWER_TYPES, DEPOSITS, CAMPS, LANES, BASE_R, ABILITIES, DIFFICULTY,
  canBuildAt, snap as snapGrid, towerStats, fmt, RARITIES, TOWER_MAX_LVL, towerUpgradeCost, extractorUpgradeCost, EXTRACTOR_COST,
  SEASONS, WEATHER,
} from '/shared/data.js';

const $ = (id) => document.getElementById(id);
const params = new URLSearchParams(location.search);
const hex = (n) => '#' + n.toString(16).padStart(6, '0');
const uiK = () => settings.uiScale || 1;
let overCanvas = false;

// ───────────── налаштування ─────────────
const DEFAULTS = { renderScale: 1, autoRes: true, quality: 'high', vol: 0.5, music: 0.35, dmgNums: true, shakeStr: 0.8, noFlash: false, uiScale: 1, fps: false };
let settings = { ...DEFAULTS };
let autoK = 1, autoAcc = 0, autoN = 0; // авто-роздільність
try { settings = { ...DEFAULTS, ...JSON.parse(localStorage.getItem('bastion.settings') || '{}') }; } catch { /* */ }
// RDP-сесія: за замовчуванням середня якість; безпечний режим після збою GPU — низька
{
  const q = new URLSearchParams(location.search);
  const stored = (() => { try { return localStorage.getItem('bastion.settings'); } catch { return null; } })();
  if (q.get('rdp') === '1' && !stored) settings.quality = 'medium';
  if (q.get('safe') === '1') settings.quality = 'low';
}
// помилки — на екран, а не мовчки
function showFatal(text) {
  let el = document.getElementById('fatal');
  if (!el) { el = document.createElement('div'); el.id = 'fatal'; document.body.appendChild(el); }
  el.innerHTML = `<b>Сталася помилка</b><div>${String(text).replace(/</g, '&lt;')}</div><small>Натисніть F12 і надішліть знімок вкладки Console розробнику. Esc — закрити.</small>`;
  el.onclick = () => el.remove();
}
window.addEventListener('error', (e) => showFatal(e.message + (e.filename ? ` (${e.filename.split('/').pop()}:${e.lineno})` : '')));
window.addEventListener('unhandledrejection', (e) => showFatal(e.reason && (e.reason.stack || e.reason.message) || e.reason));

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
    if (settings.shake === false) { settings.shake = undefined; settings.shakeStr = 0; }
    // масштаб інтерфейсу (game-ui-ux: текст має масштабуватися до 200 % без втрат)
    for (const id of ['hud', 'panel', 'tip', 'lobby', 'bigmap']) { const el = document.getElementById(id); if (el) el.style.zoom = settings.uiScale || 1; }
    world.noFlash = !!settings.noFlash;
    if (patch.renderScale != null || patch.quality) world.setRenderScale(settings.renderScale ?? 1);
    autoK = settings.renderScale ?? 1;
    $('fps').classList.toggle('hidden', !settings.fps);
  },
};
window.G = G;
if (params.get('debug') === '1') window.__dbg = {};

const renderer = new THREE.WebGLRenderer({ canvas: $('gl'), antialias: false, powerPreference: 'high-performance' });
$('gl').addEventListener('webglcontextlost', (e) => { e.preventDefault(); showFatal('Відеокарта скинула WebGL-контекст (перевантаження або збій драйвера). Гра спробує відновитися; якщо ні — знизьте якість у налаштуваннях.'); });
$('gl').addEventListener('webglcontextrestored', () => { const f = document.getElementById('fatal'); if (f) f.remove(); });
const audio = (G.audio = new Audio());
const ui = new UI(G);
$('lobby-status').textContent = 'Завантаження ресурсів… 0%';
await loadAssets(renderer, (p) => { $('lobby-status').textContent = `Завантаження ресурсів… ${Math.round(p * 100)}%`; });
$('lobby-status').textContent = 'Побудова світу…';
await new Promise((r) => setTimeout(r, 30));
const world = new World($('gl'), settings.quality, renderer);
const fx = new FX(world);
if (window.__dbg) Object.assign(window.__dbg, { world, renderer });
world.onThunder = () => setTimeout(() => audio.play('thunder'), 300 + Math.random() * 1500);
// прогрів шейдерів: компілюємо всі матеріали наперед, щоб перший вибух чи новий ворог не давав ривка
{
  $('lobby-status').textContent = 'Підготовка шейдерів…';
  const warm = new THREE.Group();
  for (const k of ENEMY_KINDS) warm.add(makeEnemy(k));
  for (const t of TOWER_TYPES) { const o = makeTower(t); setTowerLevel(o, 16); warm.add(o); }
  warm.add(makeHero(0xffffff), makeLoot(4), makeGhost('mg'), makeDrone());
  const burnt = makeEnemy('tank'); burnt.traverse((o) => { if (o.isMesh) o.material = burntMaterial(); }); warm.add(burnt);
  // розставляємо моделі перед камерою і робимо справжній кадр: так компілюються й шейдери тіней,
  // і варіанти для інстансів, яких renderer.compile() не покриває
  warm.children.forEach((o, i) => o.position.set((i % 8) * 6 - 21, 0, Math.floor(i / 8) * 6 - 8));
  world.scene.add(warm);
  world.camera.position.set(0, 40, 45); world.camera.lookAt(0, 0, 0);
  fx.explosion(0, 0, 1, 0.1); fx.eshot([0, 0], [1, 1], 1, 'mg'); fx.spawnProjectile({ id: -1, k: 'atgm', p: [0, 2, 0], v: [0, 0, 1], ttl: 0.01 });
  try { renderer.compile(world.scene, world.camera); world.render(); } catch (e) { console.warn(e); }
  world.scene.remove(warm);
}
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
  ws.onmessage = (e) => { const t1 = performance.now(); onMsg(JSON.parse(e.data)); PROF.net += performance.now() - t1; };
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
    case 'respawn': {
      me.x = m.x; me.z = m.z; me.vx = me.vz = 0;
      break;
    }
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
  ui.feed('Ласкаво просимо на позицію! Тримайте оборону, зачищайте ДРГ за периметром.', '#e8dcb8');
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
      obj.position.set(t.x, terrainHeight(t.x, t.z), t.z);
      world.scene.add(obj);
      o = { obj, data: t, rot: 0, level: 0, recoil: 0 };
      towers.set(t.id, o);
      fx.puff(t.x, 0.3, t.z, 0x8a7f6a, 20, { speed: 3, life: 1.5, size: 0.6, size1: 2.2, alpha: 0.5, up: 0.4 });
    }
    if (o.level !== t.level) { setTowerLevel(o.obj, t.level); o.level = t.level; }
    o.data = t;
  }
  for (const [id, o] of towers) if (!seen.has(id)) { world.scene.remove(o.obj); towers.delete(id); }
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
let curWx = null, curSea = null;
const dying = [];
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
        if (old && p.hp < old.hp - 1 && !p.d) { audio.play('hurt'); hurtFlash = Math.min(1, hurtFlash + 0.35); shake(Math.min(0.35, 0.08 + (old.hp - p.hp) / p.mhp)); }
        if (old && !old.d && p.d) shake(0.7);
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
    }
    if (e.hp != null && (hp < e.hp - 0.001 || (sh >= 0 && e.sh != null && sh < e.sh - 0.001))) e.flinch = 1;
    e.tx = x; e.tz = z; e.trot = rot; e.hp = hp; e.sh = sh; e.slow = slow; e.lvl = lvl;
  }
  for (const [id, e] of enemies) if (!seen.has(id)) { enemies.delete(id); startDying(e); }

  // вежі (поворот)
  for (const [id, rot, target, aura] of s.towers) {
    const o = towers.get(id);
    if (!o) continue;
    o.trot = rot; o.target = target;
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
  if (s.wx !== curWx || s.sea !== curSea) {
    world.setWeather(s.wx, s.sea, curWx == null);
    audio.setWeather(s.wx);
    curWx = s.wx; curSea = s.sea;
  }
  ui.updateHUD();
  world.setCoreHealth(s.core[0] / s.core[1]);
  audio.setIntensity(s.phase === 'wave' ? 1 : 0);
}

let hurtFlash = 0;
const TOWER_SND = { mg: 'mg', ags: 'ags', aa: 'aa', atgm: 'atgm', mortar: 'mortar', howitzer: 'howitzer' };
const MUZZLE_BIG = { mg: 0.8, ags: 1, aa: 1, atgm: 1.6, mortar: 1.8, howitzer: 3 };
function handleEvent(ev) {
  const near = (x, z) => Math.max(0.12, 1 - Math.hypot(x - me.x, z - me.z) / 110);
  switch (ev.e) {
    case 'ps':
      fx.spawnProjectile(ev);
      if (ev.k === 'hero') audio.play('rifle', near(ev.p[0], ev.p[2]));
      else if (ev.k === 'ebullet') audio.play('erifle', near(ev.p[0], ev.p[2]));
      else if (ev.k === 'rocket') audio.play('atgm', near(ev.p[0], ev.p[2]));
      break;
    case 'ph': fx.endProjectile(ev); break;
    case 'tf': {
      const o = towers.get(ev.id);
      if (!o) break;
      o.recoil = 1;
      const u = o.obj.userData;
      const dir = new THREE.Vector3(Math.sin(u.head.rotation.y), 0, Math.cos(u.head.rotation.y));
      for (const mz of u.muzzles) {
        const p = u.head.localToWorld(mz.clone());
        fx.muzzle(p.x, p.y, p.z, MUZZLE_BIG[u.type] || 1, dir);
      }
      if (u.type === 'atgm' || u.type === 'howitzer' || u.type === 'mortar') fx.puff(o.data.x, terrainHeight(o.data.x, o.data.z) + 0.3, o.data.z, 0x8a7f6a, 8, { speed: 3, life: 1.2, size: 0.6, size1: 2, alpha: 0.45, up: 0.2 });
      audio.play(TOWER_SND[u.type], near(o.data.x, o.data.z));
      break;
    }
    case 'boom': {
      const big = { agsg: 0.45, mortar: 1, shell: 1.8, rocket: 1, gren: 0.9, orbit: 1.6 }[ev.k] || 1;
      if (ev.k !== 'orbit') fx.explosion(ev.x, ev.z, ev.r * (ev.k === 'agsg' ? 0.7 : 0.6), big);
      if (ev.k !== 'orbit') audio.play(ev.k === 'agsg' ? 'smallboom' : big > 1.5 ? 'bigboom' : 'boom', near(ev.x, ev.z));
      if (big >= 1) shake(0.15 * big * near(ev.x, ev.z));
      break;
    }
    case 'die': {
      const d = ENEMIES[ENEMY_KINDS[ev.k]];
      if (d.boss) { audio.play('bigboom'); shake(0.8); ui.banner('ЦІЛЬ ЗНИЩЕНО', d.name, '#e8b24a', 3); }
      else if (d.vehicle || d.flying) audio.play('boom', near(ev.x, ev.z));
      if (ev.n === G.name) { hitMarks.push({ t: 0.35, max: 0.35, kill: true }); audio.play('ui'); }
      if (ev.c > 0 && Math.hypot(ev.x - me.x, ev.z - me.z) < 45) floaters.push({ x: ev.x, y: 2.5, z: ev.z, text: `+${fmt(ev.c)} ₴`, color: '#e8c35a', t: 1.1, size: 13 });
      break;
    }
    case 'dmg':
      if (ev.n === G.name) {
        if (settings.dmgNums) floaters.push({ x: ev.x + (Math.random() - 0.5), y: 2.5, z: ev.z, text: fmt(ev.v), color: ev.c ? '#ffd23a' : '#ffffff', t: 0.8, max: 0.8, size: ev.c ? 20 : 14, pop: true });
        hitMarks.push({ t: 0.18, max: 0.18, crit: !!ev.c, kill: false });
      }
      break;
    case 'spark': {
      let e = null, bd = 3;
      for (const q of enemies.values()) { const d = Math.hypot(q.x - ev.x, q.z - ev.z); if (d < bd) { bd = d; e = q; } }
      const y = e ? e.obj.position.y + (e.def.human ? 1.3 : 1) : terrainHeight(ev.x, ev.z) + 1;
      fx.hitSpark(ev.x, y, ev.z, e ? !!e.def.human : true);
      break;
    }
    case 'dash': fx.dash(ev.a, ev.b); audio.play('dash'); break;
    case 'gren': fx.grenade(ev.a, ev.b); break;
    case 'orbit': fx.artillery(ev.x, ev.z); audio.play('incoming'); setTimeout(() => { audio.play('bigboom'); shake(0.6); }, 1200); break;
    case 'drop': fx.explosion(ev.x, ev.z, 1.5, 0.4); audio.play('smallboom', near(ev.x, ev.z)); break;
    case 'eshot': fx.eshot(ev.a, ev.b, ev.y, ev.k); audio.play(ev.k === 'cannon' ? 'cannon' : 'emg', near(ev.a[0], ev.a[1])); break;
    case 'wallhit': fx.puff(ev.x, terrainHeight(ev.x, ev.z) + 1.3, ev.z, 0xa09a8c, 3, { speed: 2.5, life: 0.7, size: 0.25, size1: 0.9, alpha: 0.6 }); fx.emit(ev.x, terrainHeight(ev.x, ev.z) + 1.3, ev.z, 0xffd090, 2, { speed: 4, life: 0.12, size: 0.2 }); break;
    case 'core': fx.explosion(ev.x, ev.z, 3, 1); audio.play('boom'); shake(0.3); break;
    case 'heal': fx.emit(ev.x, terrainHeight(ev.x, ev.z) + 1.5, ev.z, 0xbfe0ff, 4, { speed: 3, life: 0.2, size: 0.2, grav: 1 }); break;
    case 'unload': fx.puff(ev.x, terrainHeight(ev.x, ev.z) + 0.5, ev.z, 0x8a7f6a, 14, { speed: 3, life: 1.5, size: 0.8, size1: 2.5, alpha: 0.5 }); break;
    case 'hdie': fx.puff(ev.x, 0.6, ev.z, 0x5a1a14, 10, { speed: 2, life: 0.8, size: 0.3, size1: 0.8, alpha: 0.8, grav: 0.5 }); audio.play('hurt'); break;
    case 'lvl': fx.emit(ev.x, terrainHeight(ev.x, ev.z) + 1.5, ev.z, 0xffe0a0, 25, { speed: 3, life: 1, size: 0.25, grav: -0.3, up: 1 }); break;
    case 'pick': {
      if (ev.n === G.name) { audio.play(ev.r >= 4 ? 'legend' : 'pick'); floaters.push({ x: ev.x, y: 3, z: ev.z, text: RARITIES[ev.r].name + '!', color: RARITIES[ev.r].color, t: 1.6, size: 16 }); }
      break;
    }
    case 'build': audio.play('build'); break;
    case 'up': fx.puff(ev.x, terrainHeight(ev.x, ev.z) + 0.4, ev.z, 0x8a7f6a, 12, { speed: 2, life: 1.2, size: 0.5, size1: 1.8, alpha: 0.45 }); audio.play('upgrade'); break;
    case 'sell': fx.puff(ev.x, terrainHeight(ev.x, ev.z) + 0.4, ev.z, 0x8a7f6a, 16, { speed: 3, life: 1.2, size: 0.6, size1: 2, alpha: 0.5 }); break;
    case 'wave': ui.banner(`ХВИЛЯ ${ev.w}`, ev.w % 10 === 0 ? 'Увага: важка ціль!' : 'противник атакує', '#e8a04a', 3); audio.play('siren'); break;
    case 'boss': ui.banner('⚠ ВАЖКА ЦІЛЬ', ENEMIES[ev.k].name, '#e85a4a', 3.5); audio.play('siren'); shake(0.3); break;
    case 'clear': ui.banner('ХВИЛЮ ВІДБИТО', 'час на підготовку', '#9ccf6a', 3); audio.play('clear'); break;
    case 'defeat': ui.banner('ШТАБ ЗНИЩЕНО', 'хвиля повториться', '#e85a4a', 4); audio.play('defeat'); shake(0.8); break;
    case 'camp': ui.feed('Позицію ДРГ зачищено! Згодом противник поверне її сильнішими силами.', '#e8c35a'); break;
    case 'season': ui.banner(SEASONS[ev.s].name.toUpperCase(), 'настала нова пора року', '#e8dcb8', 3); break;
  }
}

// ───────────── загиблі: падіння, вигоряння ─────────────
function startDying(e) {
  const o = e.obj, u = o.userData;
  const d = { obj: o, t: 0, human: u.human, vehicle: u.vehicle, flying: u.flying, rx: o.rotation.x, y0: o.position.y, vy: 0, side: Math.random() < 0.5 ? -1 : 1, life: 7 };
  if (u.flying) {
    if (e.kind === 'shahed' || e.kind === 'fpv') { fx.explosion(o.position.x, o.position.z, e.kind === 'shahed' ? 4 : 1.8, e.kind === 'shahed' ? 1.4 : 0.6, o.position.y); world.scene.remove(o); return; }
    d.life = 30;
  } else if (u.vehicle || u.boss) {
    o.traverse((m) => { if (m.isMesh && !m.isInstancedMesh) m.material = burntMaterial(); });
    fx.explosion(o.position.x, o.position.z, u.boss ? 6 : 3.5, u.boss ? 2.2 : 1.4, o.position.y + 1);
    fx.fire(o.position.x, o.position.z, u.boss ? 14 : 8, u.boss ? 2.5 : 1.4, o.position.y + 1.2);
    d.life = u.boss ? 18 : 12;
  }
  dying.push(d);
}
function updateDying(dt) {
  for (let i = dying.length - 1; i >= 0; i--) {
    const d = dying[i], o = d.obj;
    d.t += dt;
    if (d.human) {
      const k = Math.min(1, d.t / 0.45);
      o.rotation.x = d.rx - (Math.PI / 2) * k * k;
      o.position.y = d.y0 + 0.15 * Math.sin(k * Math.PI);
    } else if (d.flying) {
      d.vy -= 9.8 * dt;
      o.position.y += d.vy * dt;
      o.rotation.z += dt * 1.5 * d.side;
      const gy = terrainHeight(o.position.x, o.position.z);
      if (o.position.y <= gy + 0.5) {
        fx.explosion(o.position.x, o.position.z, 6, 2); fx.fire(o.position.x, o.position.z, 10, 2.5);
        o.position.y = gy + 0.5; d.flying = false; d.t = 0; d.life = 14;
        o.traverse((m) => { if (m.isMesh) m.material = burntMaterial(); });
      }
    }
    if (d.t > d.life - 2) o.position.y -= dt * (d.human ? 0.4 : 0.3);
    if (d.t > d.life) { world.scene.remove(o); dying.splice(i, 1); }
  }
}
// game-feel: тряска на травмі; сила — налаштування доступності, з відстанню слабшає
const trauma = new Trauma();
function shake(a, x = null, z = null) {
  const k = x == null ? 1 : Math.max(0, 1 - Math.hypot(x - me.x, z - me.z) / 80);
  trauma.add(a * k);
}
world.addTrauma = shake;
const hitMarks = [];

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
  overCanvas = e.target === gl || e.target === overlay;
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
  push(0, 0, 7.4);
  [me.x, me.z] = pushOutOfRects(me.x, me.z, world.colliders, 0.5);
  for (const t of G.world.towers) push(t.x, t.z, t.type === 'howitzer' ? 3.2 : 2.4);
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
const tmpV2 = new THREE.Vector3();
function nearestHeroTo(x, z, r) { let b = null, bd = r; for (const h of heroes.values()) { if (!h.obj.visible) continue; const d = Math.hypot(h.x - x, h.z - z); if (d < bd) { bd = d; b = h; } } return b; }
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
    if (h.drone) for (const p of h.drone.userData.props) p.rotation.z += dt * 50;
    o.rotation.y = h.rot;
    const sw = Math.min(1, sp / 6);
    u.legs[0].rotation.x = Math.sin(h.walk * 0.8) * 0.65 * sw;
    u.legs[1].rotation.x = -Math.sin(h.walk * 0.8) * 0.65 * sw;
    u.body.position.y = Math.abs(Math.sin(h.walk * 0.8)) * 0.05 * sw;
    const firing = h.data && h.data.f;
    u.muzzle.visible = firing && Math.sin(t * 70) > 0.3;
    if (firing && Math.random() < dt * 8) { const mp = u.muzzle.getWorldPosition(tmpV2); fx.puff(mp.x, mp.y, mp.z, 0x9a948a, 1, { speed: 0.5, life: 0.8, size: 0.2, size1: 0.9, alpha: 0.3, up: 1 }); }
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
    e.rot = lerpAngle(e.rot, e.trot, e.def.vehicle ? k * 0.5 : k);
    const o = e.obj, u = o.userData;
    const ground = terrainHeight(e.x, e.z);
    const fh = e.kind === 'fpv' ? 2.4 : e.kind === 'shahed' ? 7 : e.kind === 'boss_heli' ? 8 : 0;
    const y = u.flying ? ground + fh + Math.sin(t * 2 + e.seed) * (e.kind === 'fpv' ? 0.3 : 0.6) : ground;
    o.position.set(e.x, y, e.z);
    o.rotation.y = e.rot;
    if (e.flinch > 0) { e.flinch = Math.max(0, e.flinch - dt * 7); const fl = Math.sin(e.flinch * Math.PI) * (u.human ? 0.16 : 0.03); u.body.rotation.x = -fl; } else if (u.human) u.body.rotation.x = 0;
    const sp = Math.hypot(e.x - (e.px ?? e.x), e.z - (e.pz ?? e.z)) / Math.max(dt, 1e-3);
    e.px = e.x; e.pz = e.z;
    if (u.human) {
      e.walk = (e.walk || e.seed) + dt * Math.min(sp, 8) * 1.6;
      const sw = Math.min(1, sp / 2);
      u.legs[0].rotation.x = Math.sin(e.walk) * 0.6 * sw;
      u.legs[1].rotation.x = -Math.sin(e.walk) * 0.6 * sw;
      u.body.position.y = Math.abs(Math.sin(e.walk)) * 0.05 * sw;
    } else if (u.vehicle && !u.flying) {
      u.body.rotation.x = Math.sin(t * 3 + e.seed) * 0.01 * Math.min(1, sp / 2);
      if (u.turret) {
        const nh = nearestHeroTo(e.x, e.z, 25);
        const tr = nh ? Math.atan2(nh.x - e.x, nh.z - e.z) - e.rot : 0;
        u.turret.rotation.y = lerpAngle(u.turret.rotation.y, tr, 1 - Math.exp(-dt * 2));
      }
    }
    if (e.kind === 'fpv') o.rotation.x = 0.25;
    if (e.kind === 'boss_heli') o.rotation.x = 0.08;
    for (const p of u.props) p.rotation.z += dt * 60;
    if (u.rotor) u.rotor.rotation.y += dt * 18;
    if (e.slow && Math.random() < dt * 3) fx.puff(e.x, y + 0.5, e.z, 0x8a7f6a, 1, { speed: 1, life: 0.8, size: 0.4, size1: 1.2, alpha: 0.35 });
    if ((u.vehicle || u.boss) && e.hp < 0.5 && Math.random() < dt * (u.boss ? 10 : 4)) fx.puff(e.x, y + 2, e.z, 0x2c2926, 1, { speed: 0.6, life: 2.5, size: 0.8, size1: 3, alpha: 0.55, vy: 2 });
  }
  // вежі
  for (const o of towers.values()) {
    const u = o.obj.userData;
    if (o.trot != null) { o.rot = lerpAngle(o.rot, o.trot, 1 - Math.exp(-dt * 6)); u.head.rotation.y = o.rot; }
    o.recoil = Math.max(0, (o.recoil || 0) - dt * 5);
    u.head.position.z = -o.recoil * (u.type === 'howitzer' ? 0.35 : 0.06);
  }
  // лут
  for (const l of loots.values()) {
    l.obj.userData.gem.material.opacity = 0.18 + Math.sin(t * 2.5) * 0.06 + l.r * 0.04;
  }
  // примара
  if (ghost) {
    const x = snapGrid(mouse.world.x), z = snapGrid(mouse.world.z);
    ghost.position.set(x, 0, z);
    const ok = canBuildAt(x, z) && !G.world.towers.some((tt) => tt.x === x && tt.z === z);
    const col = ok ? 0xffffff : 0xff3344;
    ghost.userData.ghostMat.color.setHex(col);
    ghost.userData.range.material.color.setHex(col);
    ghost.position.y = terrainHeight(x, z);
    const r = towerStats(G.buildType, 1, G.world.research).range;
    ghost.userData.range.scale.setScalar(r);
    ghost.userData.fill.scale.setScalar(r);
  }
}

// ───────────── камера ─────────────
const camPos = new THREE.Vector3(0, 40, 30);
const camLook = new THREE.Vector3();
function updateCamera(dt) {
  const h = 25 * zoom, back = 19 * zoom;
  // легкий зсув у бік курсора
  const ox = THREE.MathUtils.clamp((mouse.world.x - me.x) * 0.18, -8, 8);
  const oz = THREE.MathUtils.clamp((mouse.world.z - me.z) * 0.18, -8, 8);
  const target = tmpV.set(me.x + ox, 0, me.z + oz);
  camLook.lerp(target, 1 - Math.exp(-dt * 6));
  camPos.set(camLook.x, camLook.y + h, camLook.z + back);
  trauma.strength = settings.shakeStr ?? 0.8;
  trauma.update(dt);
  world.camera.position.copy(camPos);
  world.camera.position.x += trauma.x; world.camera.position.y += trauma.y * 0.6;
  world.camera.lookAt(camLook);
  world.camera.rotateZ(trauma.roll);
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
    const top = e.obj.position.y + (e.def.human ? 2.6 : e.def.flying ? 1.5 : e.def.boss ? 5.5 : 3.2);
    const p = project(e.x, top, e.z);
    if (!p || p[0] < -50 || p[0] > W + 50 || p[1] < -50 || p[1] > H + 50) continue;
    if (e.def.boss) continue; // бос — окрема велика смуга
    if (e.hp >= 0.999 && !e.def.wild && !(e.sh >= 0 && e.sh < 0.999)) continue;
    const w = e.def.wild ? 40 : 30 * Math.max(0.8, e.def.size);
    octx.fillStyle = 'rgba(0,0,0,.6)';
    octx.fillRect(p[0] - w / 2 - 1, p[1] - 1, w + 2, 6);
    octx.fillStyle = e.def.wild ? '#e0a040' : '#d8463a';
    octx.fillRect(p[0] - w / 2, p[1], w * e.hp, 4);
    if (e.sh >= 0) { octx.fillStyle = '#6aa8ff'; octx.fillRect(p[0] - w / 2, p[1] - 3, w * e.sh, 2); }
    if (e.def.wild) { octx.fillStyle = '#ecdcb8'; octx.fillText(`${e.def.name} ${e.lvl}`, p[0], p[1] - 5); }
  }
  // бос: велика смуга вгорі
  const boss = [...enemies.values()].find((e) => e.def.boss);
  if (boss) {
    const w = Math.min(600, W * 0.45);
    const x = W / 2 - w / 2, y = 130;
    octx.fillStyle = 'rgba(0,0,0,.7)'; octx.fillRect(x - 2, y - 2, w + 4, 18);
    const gr = octx.createLinearGradient(x, 0, x + w, 0);
    gr.addColorStop(0, '#7a1c14'); gr.addColorStop(1, '#d8463a');
    octx.fillStyle = gr; octx.fillRect(x, y, w * boss.hp, 14);
    if (boss.sh >= 0) { octx.fillStyle = '#8a6aff'; octx.fillRect(x, y + 10, w * boss.sh, 4); }
    octx.font = '700 15px Bahnschrift, Segoe UI, sans-serif';
    octx.fillStyle = '#fff'; octx.fillText(`☠ ${boss.def.name} · ${Math.round(boss.hp * 100)}%`, W / 2, y - 6);
    octx.font = '600 12px Bahnschrift, Segoe UI, sans-serif';
  }
  // імена героїв
  for (const [n, h] of heroes) {
    if (!h.obj.visible) continue;
    const p = project(h.x, h.obj.position.y + 2.9, h.z);
    if (!p) continue;
    octx.fillStyle = hex(h.color);
    octx.fillText(`${n} · ${h.data?.l ?? ''}`, p[0], p[1] - 8);
    if (n !== G.name && h.data) {
      octx.fillStyle = 'rgba(0,0,0,.6)'; octx.fillRect(p[0] - 21, p[1] - 4, 42, 5);
      octx.fillStyle = '#9ccf6a'; octx.fillRect(p[0] - 20, p[1] - 3, 40 * (h.data.hp / h.data.mhp), 3);
    }
  }
  // мітки таборів і родовищ
  octx.font = '600 13px Bahnschrift, Segoe UI, sans-serif';
  CAMPS.forEach((c, i) => {
    if (Math.hypot(c.x - me.x, c.z - me.z) > 60) return;
    const p = project(c.x, terrainHeight(c.x, c.z) + 6, c.z);
    if (!p) return;
    const lv = G.world.camps[i]?.level ?? 1;
    octx.fillStyle = 'rgba(236,190,110,.9)';
    octx.fillText(`⌖ Позиція ДРГ · рів. ${lv}`, p[0], p[1]);
  });
  DEPOSITS.forEach((d, i) => {
    if (Math.hypot(d.x - me.x, d.z - me.z) > 55) return;
    const p = project(d.x, terrainHeight(d.x, d.z) + 10, d.z);
    if (!p) return;
    const lv = G.world.extractors[i]?.level || 0;
    octx.fillStyle = 'rgba(190,210,220,.95)';
    octx.fillText(lv ? `⚙ Майстерня · рів. ${lv}` : '⚙ Покинутий склад [F]', p[0], p[1]);
  });
  // підписи вогневих позицій поруч
  octx.font = '600 11px Bahnschrift, Segoe UI, sans-serif';
  for (const o of towers.values()) {
    if (Math.hypot(o.data.x - me.x, o.data.z - me.z) > 28) continue;
    const p = project(o.data.x, terrainHeight(o.data.x, o.data.z) + 3.6, o.data.z);
    if (!p) continue;
    octx.fillStyle = 'rgba(235,230,214,.8)';
    octx.fillText(`${TOWERS[o.data.type].name} · рів. ${o.data.level}`, p[0], p[1]);
  }
  // вибрана вежа
  if (ui.selTower) {
    const t = towers.get(ui.selTower);
    if (t) {
      const r = towerStats(t.data.type, t.data.level, G.world.research).range;
      const pts = [];
      for (let i = 0; i <= 64; i++) { const a = (i / 64) * Math.PI * 2; pts.push(project(t.data.x + Math.cos(a) * r, terrainHeight(t.data.x + Math.cos(a) * r, t.data.z + Math.sin(a) * r) + 0.2, t.data.z + Math.sin(a) * r)); }
      octx.strokeStyle = 'rgba(255,240,200,.7)'; octx.lineWidth = 2; octx.beginPath();
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
    const ps = f.pop ? popScale(Math.min(1, (f.max - f.t) / 0.22)) : 1;
    octx.font = `700 ${Math.round(f.size * ps * uiK())}px Tektur, Bahnschrift, Segoe UI, sans-serif`;
    octx.strokeStyle = 'rgba(0,0,0,.8)'; octx.lineWidth = 3;
    octx.strokeText(f.text, p[0], p[1]);
    octx.fillStyle = f.color; octx.fillText(f.text, p[0], p[1]);
  }
  octx.globalAlpha = 1;
  if (floaters.length > 120) floaters.splice(0, floaters.length - 120);
  // приціл і маркери влучання (game-feel: миттєве підтвердження кожного влучання)
  if (G.snap && !G.buildType && !ui.isOpen() && overCanvas) {
    const cx = mouse.x, cy = mouse.y, r = 9 * uiK();
    octx.strokeStyle = 'rgba(235,230,214,.85)'; octx.lineWidth = 1.5;
    octx.beginPath();
    for (const [dx, dy] of [[1, 0], [-1, 0], [0, 1], [0, -1]]) { octx.moveTo(cx + dx * r * 0.45, cy + dy * r * 0.45); octx.lineTo(cx + dx * r, cy + dy * r); }
    octx.stroke();
    for (let i = hitMarks.length - 1; i >= 0; i--) {
      const hm = hitMarks[i];
      hm.t -= dt;
      if (hm.t <= 0) { hitMarks.splice(i, 1); continue; }
      const k = hm.t / hm.max, s = (hm.kill ? 14 : hm.crit ? 11 : 8) * uiK() * (1 + (1 - k) * 0.3);
      octx.globalAlpha = k;
      octx.strokeStyle = hm.kill ? '#e8463a' : hm.crit ? '#ffd23a' : '#ffffff'; octx.lineWidth = hm.kill ? 3 : 2;
      octx.beginPath();
      for (const [dx, dy] of [[1, 1], [-1, 1], [1, -1], [-1, -1]]) { octx.moveTo(cx + dx * s * 0.45, cy + dy * s * 0.45); octx.lineTo(cx + dx * s, cy + dy * s); }
      octx.stroke();
    }
    octx.globalAlpha = 1;
  }
  if (hitMarks.length > 12) hitMarks.splice(0, hitMarks.length - 12);
  // червоний спалах при отриманні шкоди (вимикається в «менше спалахів»)
  if (hurtFlash > 0 && settings.noFlash) hurtFlash = 0;
  if (hurtFlash > 0) {
    hurtFlash = Math.max(0, hurtFlash - dt * 2);
    const g = octx.createRadialGradient(W / 2, H / 2, Math.min(W, H) * 0.3, W / 2, H / 2, Math.max(W, H) * 0.7);
    g.addColorStop(0, 'rgba(160,0,0,0)'); g.addColorStop(1, `rgba(160,0,0,${hurtFlash * 0.3})`);
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
  c.fillStyle = 'rgba(40,44,30,.95)'; c.fillRect(0, 0, S, S);
  // база
  let [bx, bz] = P(0, 0);
  c.fillStyle = 'rgba(120,110,80,.5)'; c.beginPath(); c.arc(bx, bz, BASE_R * sc, 0, Math.PI * 2); c.fill();
  c.fillStyle = 'rgba(90,90,85,.6)'; for (const b of world.colliders) { const [qx, qz] = P(b.x, b.z); c.save(); c.translate(qx, qz); c.rotate(-b.rot); c.fillRect(-b.w * sc / 2, -b.d * sc / 2, b.w * sc, b.d * sc); c.restore(); }
  // дороги
  c.strokeStyle = 'rgba(150,125,90,.85)'; c.lineWidth = Math.max(2, 4 * sc);
  for (const l of LANES) { c.beginPath(); l.forEach(([x, z], i) => { const [px, pz] = P(x, z); i ? c.lineTo(px, pz) : c.moveTo(px, pz); }); c.stroke(); }
  // родовища
  DEPOSITS.forEach((d, i) => { const [x, z] = P(d.x, d.z); c.fillStyle = G.world.extractors[i]?.level ? '#d8dccc' : 'rgba(216,220,204,.35)'; c.fillRect(x - 4, z - 4, 8, 8); });
  // табори
  c.font = `600 ${S > 300 ? 15 : 9}px Bahnschrift, sans-serif`; c.textAlign = 'center';
  CAMPS.forEach((cp, i) => { const [x, z] = P(cp.x, cp.z); c.fillStyle = ['#e0a040', '#d88030', '#c85a30', '#b03a28'][cp.tier]; c.beginPath(); c.arc(x, z, 3.5, 0, Math.PI * 2); c.fill(); if (S > 300) c.fillText(G.world.camps[i]?.level ?? '', x, z - 7); });
  // вежі
  for (const t of G.world.towers) { const [x, z] = P(t.x, t.z); c.fillStyle = hex(TOWERS[t.type].color); c.fillRect(x - 2, z - 2, 4, 4); }
  // вороги
  for (const e of enemies.values()) {
    const [x, z] = P(e.x, e.z);
    c.fillStyle = e.def.wild ? 'rgba(224,160,64,.85)' : e.def.boss ? '#ff4030' : '#d8463a';
    c.beginPath(); c.arc(x, z, e.def.boss ? 6 : 2, 0, Math.PI * 2); c.fill();
  }
  // лут
  for (const l of loots.values()) { const [x, z] = P(l.obj.position.x, l.obj.position.z); c.fillStyle = RARITIES[l.r].color; c.fillRect(x - 2, z - 2, 4, 4); }
  // ядро
  [bx, bz] = P(0, 0);
  c.fillStyle = '#e8dcb8'; c.fillRect(bx - 5, bz - 5, 10, 10);
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
  try { frameBody(now); } catch (e) { if (!frame.err) { frame.err = 1; console.error('frame error', e.message, e.stack); showFatal(e.stack || e.message); } }
}
// Профайлер (F3): середній час кожної частини кадру в мс — щоб бачити, що саме гальмує
const PROF = { on: false, acc: {}, n: 0, last: 0, net: 0 };
const pt = (k, t0) => { if (PROF.on) PROF.acc[k] = (PROF.acc[k] || 0) + performance.now() - t0; };
window.addEventListener('keydown', (e) => { if (e.code === 'F3') { PROF.on = !PROF.on; PROF.acc = {}; PROF.n = 0; $('prof').classList.toggle('hidden', !PROF.on); e.preventDefault(); } });
function profReport(now) {
  if (!PROF.on) return;
  PROF.n++;
  if (now - PROF.last < 1000) return;
  const inf = renderer.info.render, mem = renderer.info.memory;
  const rows = Object.entries(PROF.acc).map(([k, v]) => `${k.padEnd(10)} ${(v / PROF.n).toFixed(2)} мс`).join('\n');
  const pr = renderer.getPixelRatio(), sz = renderer.getDrawingBufferSize(new THREE.Vector2());
  $('prof').textContent = `FPS ${(PROF.n * 1000 / (now - PROF.last)).toFixed(0)}  кадр ${((now - PROF.last) / PROF.n).toFixed(1)} мс\n${rows}\nмережа    ${(PROF.net / PROF.n).toFixed(2)} мс\nвиклики   ${inf.calls}\nтрикутн.  ${(inf.triangles / 1000).toFixed(0)}k\nбуфер     ${sz.x}×${sz.y} (×${pr.toFixed(2)})\nгеом/текс ${mem.geometries}/${mem.textures}\nякість    ${settings.quality}`;
  PROF.acc = {}; PROF.n = 0; PROF.last = now; PROF.net = 0;
}
function frameBody(now) {
  let t0 = performance.now();
  const dt = Math.min(0.1, (now - last) / 1000);
  last = now;
  const t = now / 1000;
  if (G.snap) {
    updateMouseWorld();
    moveMe(dt);
    sendInput(dt);
    for (const k of ['q', 'e', 'r']) G.localCds[k] = Math.max(0, G.localCds[k] - dt);
    t0 = performance.now(); updateEntities(dt, t); pt('сутності', t0);
    updateCamera(dt);
    updatePrompt();
    mapAcc += dt;
    if (mapAcc > 0.1) {
      t0 = performance.now();
      mapAcc = 0;
      drawMap($('minimap'), 70, me);
      if (!$('bigmap').classList.contains('hidden')) drawMap($('bigmap-c'), 132);
      pt('мінікарта', t0);
    }
    panelAcc += dt;
    if (panelAcc > 0.5) { panelAcc = 0; ui.renderPanel(); }
  } else {
    // обертання камери в лобі
    world.camera.position.set(Math.cos(t * 0.05) * 70, 38, Math.sin(t * 0.05) * 70);
    world.camera.lookAt(0, 4, 0);
  }
  t0 = performance.now(); world.update(dt, me); pt('погода', t0);
  gl.style.cursor = G.snap && !G.buildType ? 'none' : 'default';
  updateDying(dt);
  if (world.coreDamage > 0.3 && Math.random() < dt * world.coreDamage * 8) fx.puff((Math.random() - 0.5) * 4, 3, (Math.random() - 0.5) * 4, 0x2c2926, 1, { speed: 0.5, life: 4, size: 1, size1: 4, alpha: 0.6, vy: 2.5 });
  t0 = performance.now(); fx.update(dt); pt('ефекти', t0);
  t0 = performance.now(); world.render(); pt('рендер(CPU)', t0);
  t0 = performance.now(); drawOverlay(dt); pt('оверлей', t0);
  profReport(now);
  // автопідлаштування роздільності: тримаємо ≥ 30 FPS, знижуючи кількість пікселів (не нижче 50 %)
  if (G.snap && settings.autoRes) {
    autoAcc += dt; autoN++;
    if (autoAcc > 3) {
      const fps = autoN / autoAcc;
      const base = settings.renderScale ?? 1;
      if (fps < 30 && autoK > 0.5) autoK = Math.max(0.5, autoK - 0.1);
      else if (fps > 52 && autoK < base) autoK = Math.min(base, autoK + 0.05);
      world.setRenderScale(autoK);
      autoAcc = 0; autoN = 0;
    }
  }
  fpsAcc += dt; fpsN++;
  if (fpsAcc > 1) { $('fps').textContent = `${Math.round(fpsN / fpsAcc)} FPS · ${enemies.size} ворогів`; fpsAcc = 0; fpsN = 0; }
  void lastSnapAt;
}
requestAnimationFrame(frame);
void EXTRACTOR_COST; void extractorUpgradeCost; void TOWER_MAX_LVL; void towerUpgradeCost;
