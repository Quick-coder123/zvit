// Авторитетна симуляція гри. Працює в Node (усередині Electron або окремо).
import fs from 'node:fs';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { cityBuildings, pushOutOfRects, segmentBlocked, toWorld } from '../shared/collision.js';
import {
  TICK, WORLD, LANES, PORTALS, DEPOSITS, CAMPS, CAMP_TIERS, ENEMIES, ENEMY_KINDS, TOWERS, TOWER_MAX_LVL,
  DIFFICULTY, waveHpMul, waveDmgMul, bountyMul, waveComposition, wildStats, towerStats, towerUpgradeCost,
  canBuildAt, snap, EXTRACTOR_COST, extractorUpgradeCost, extractorRate, RESEARCH, researchCost,
  TALENTS, xpToNext, heroStats, newHeroProfile, genItem, itemValue, itemSalvage, ABILITIES, HERO_COLORS, SLOTS,
  SEASONS, seasonOf, WEATHER, SEASON_WEATHER, TARGET_MODES, bossScale,
} from '../shared/data.js';

const SAVE_VERSION = 2;
const INV_MAX = 40;

function rng(seed) {
  let a = seed >>> 0;
  return () => {
    a = (a + 0x6d2b79f5) >>> 0;
    let t = a;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}
const rand = rng(Date.now());
const r1 = (v) => Math.round(v * 10) / 10;
const r2 = (v) => Math.round(v * 100) / 100;

export class Game {
  constructor(saveDir) {
    this.saveDir = saveDir;
    this.walls = cityBuildings(); // будинки міста: укриття й перешкоди
    fs.mkdirSync(saveDir, { recursive: true });
    shareFolder(saveDir);
    this.savePath = path.join(saveDir, 'world.json');
    this.clients = new Map(); // ws -> player
    this.running = false;
    this.nextId = 1;
    this.events = [];
    this.time = 0;
    this.lastSave = 0;
  }

  // ───────────── збереження ─────────────
  saveInfo() {
    try {
      const s = JSON.parse(fs.readFileSync(this.savePath, 'utf8'));
      return { wave: s.wave, difficulty: s.cfg.difficulty, heroes: Object.keys(s.heroes || {}), saved: s.savedAt };
    } catch { return null; }
  }
  newWorld(cfg) {
    const difficulty = DIFFICULTY[cfg.difficulty] ? cfg.difficulty : 'normal';
    this.w = {
      version: SAVE_VERSION,
      cfg: { difficulty, prepTime: clampPrep(cfg.prepTime) },
      wave: 1, credits: 450, crystals: 0,
      research: {},
      towers: [],
      extractors: DEPOSITS.map((d) => ({ id: d.id, level: 0 })),
      camps: CAMPS.map((c) => ({ id: c.id, level: CAMP_TIERS[c.tier].lvl })),
      heroes: {},
      stats: { kills: 0, bosses: 0, defeats: 0, bestWave: 0, playTime: 0 },
    };
  }
  load() {
    try {
      const s = JSON.parse(fs.readFileSync(this.savePath, 'utf8'));
      if (s.version !== SAVE_VERSION) throw new Error('version');
      this.w = s;
      return true;
    } catch { return false; }
  }
  save() {
    if (!this.running) return;
    // зберегти профілі онлайн-гравців
    for (const p of this.clients.values()) if (p.profile) this.w.heroes[p.name] = p.profile;
    this.w.towers = [...this.towers.values()].map((t) => ({ type: t.type, level: t.level, x: t.x, z: t.z, spent: t.spent, mode: t.mode }));
    this.w.savedAt = Date.now();
    const tmp = this.savePath + '.tmp';
    const data = JSON.stringify(this.w);
    try {
      fs.writeFileSync(tmp, data);
      fs.renameSync(tmp, this.savePath);
    } catch {
      // файл міг створити інший обліковий запис Windows — пробуємо перезаписати вміст напряму
      try { fs.writeFileSync(this.savePath, data); } catch (e) { console.error('save failed', e); return; }
    }
    try { fs.copyFileSync(this.savePath, path.join(this.saveDir, 'world.backup.json')); } catch { /* не критично */ }
  }

  start(cfg) {
    if (cfg.mode === 'continue' && this.load()) {
      if (cfg.prepTime) this.w.cfg.prepTime = clampPrep(cfg.prepTime);
    } else this.newWorld(cfg);
    this.towers = new Map();
    for (const t of this.w.towers) this.addTower(t.type, t.x, t.z, t.level, t.spent).mode = t.mode || 'first';
    this.enemies = new Map();
    this.projectiles = new Map();
    this.loot = new Map();
    this.timers = [];
    this.phase = 'prep';
    this.timer = this.w.cfg.prepTime;
    this.spawnQueue = [];
    this.coreMax = this.calcCoreMax();
    this.coreHp = this.coreMax;
    this.campState = CAMPS.map(() => ({ alive: new Set(), respawn: 0 }));
    this.running = true;
    if (!this.w.weather) this.rollWeather(false);
    for (const c of this.campState.keys()) this.spawnCamp(c);
    this.broadcastWorld();
    this.msg(`Гру запущено. Складність: ${DIFFICULTY[this.w.cfg.difficulty].name}. Хвиля ${this.w.wave}.`, '#3ae0ff');
  }

  // ───────────── мережа ─────────────
  send(ws, obj) { if (ws.readyState === 1) ws.send(JSON.stringify(obj)); }
  broadcast(obj) { const s = JSON.stringify(obj); for (const ws of this.clients.keys()) if (ws.readyState === 1) ws.send(s); }
  msg(text, color = '#cfe') { this.broadcast({ t: 'msg', text, color }); }

  onConnect(ws) {
    this.clients.set(ws, { ws, name: null, profile: null });
  }
  onClose(ws) {
    const p = this.clients.get(ws);
    const wasHost = this.isHost(ws);
    this.clients.delete(ws);
    if (!this.running && wasHost) {
      // права хоста переходять наступному гравцю в лобі
      const [nws, np] = [...this.clients.entries()][0] || [];
      if (nws && np.name) this.send(nws, { t: 'hello', name: np.name, host: true, needSetup: true, save: this.saveInfo() });
    }
    if (p && p.profile && this.running) {
      this.w.heroes[p.name] = p.profile;
      this.msg(`${p.name} покинув(ла) гру.`, '#889');
      this.save();
    }
  }
  isHost(ws) { return [...this.clients.keys()][0] === ws; }

  onMessage(ws, raw) {
    let m;
    try { m = JSON.parse(raw); } catch { return; }
    const p = this.clients.get(ws);
    if (!p) return;
    if (m.t === 'join') return this.join(ws, p, String(m.name || 'Гравець').slice(0, 24));
    if (m.t === 'setup') {
      if (this.running || !this.isHost(ws)) return;
      this.start(m);
      for (const [cws, cp] of this.clients) if (cp.name) this.enterGame(cws, cp);
      return;
    }
    if (!this.running || !p.profile) return;
    const h = p;
    switch (m.t) {
      case 'in': {
        if (h.dead) break;
        h.x = clamp(+m.x || 0, -WORLD, WORLD); h.z = clamp(+m.z || 0, -WORLD, WORLD);
        h.ax = +m.ax || 0; h.az = +m.az || 0; h.fire = !!m.fire; h.rot = +m.rot || 0;
        break;
      }
      case 'ab': this.ability(h, m); break;
      case 'build': this.build(h, m.type, m.x, m.z); break;
      case 'upgrade': this.upgradeTower(h, m.id); break;
      case 'sell': this.sellTower(h, m.id); break;
      case 'tmode': { const t = this.towers.get(m.id); if (t && TARGET_MODES.includes(m.mode)) { t.mode = m.mode; this.broadcastWorld(); } break; }
      case 'extract': this.extract(h, m.id); break;
      case 'research': this.research(h, m.id); break;
      case 'talent': this.talent(h, m.id); break;
      case 'respec': this.respec(h); break;
      case 'equip': this.equip(h, m.id); break;
      case 'unequip': this.unequip(h, m.slot); break;
      case 'sellItem': this.sellItem(h, m.id, false); break;
      case 'salvage': this.sellItem(h, m.id, true); break;
      case 'bulk': this.bulk(h, +m.max, m.mode === 'salvage'); break;
      case 'ready': h.ready = !h.ready; break;
      case 'cfg': if (this.isHost(ws)) { this.w.cfg.prepTime = clampPrep(m.prepTime); this.broadcastWorld(); this.msg(`Час підготовки: ${this.w.cfg.prepTime} с`); } break;
      case 'chat': this.msg(`${h.name}: ${String(m.text).slice(0, 140)}`, '#fff'); break;
      case 'saveNow': this.save(); this.send(ws, { t: 'msg', text: 'Гру збережено.', color: '#4dff7a' }); break;
    }
  }

  join(ws, p, name) {
    // унікальне ім'я серед онлайн
    const online = new Set([...this.clients.values()].filter((c) => c !== p && c.name).map((c) => c.name));
    let n = name, i = 2;
    while (online.has(n)) n = `${name} ${i++}`;
    p.name = n;
    if (!this.running) {
      this.send(ws, { t: 'hello', name: n, host: this.isHost(ws), needSetup: true, save: this.saveInfo() });
      return;
    }
    this.enterGame(ws, p);
  }
  enterGame(ws, p) {
    p.profile = this.w.heroes[p.name] || newHeroProfile(p.name);
    if (p.profile.color == null) {
      const used = new Set([...this.clients.values()].map((c) => c.profile && c.profile.color));
      p.profile.color = HERO_COLORS.find((c) => !used.has(c)) ?? HERO_COLORS[0];
    }
    this.w.heroes[p.name] = p.profile;
    p.stats = heroStats(p.profile, this.w.research);
    p.hp = p.stats.maxHp; p.dead = false; p.respawn = 0;
    const a = rand() * Math.PI * 2;
    p.x = Math.cos(a) * 12; p.z = Math.sin(a) * 12; p.ax = 0; p.az = 0; p.rot = 0;
    p.fireCd = 0; p.cds = { q: 0, e: 0, r: 0 }; p.droneCd = 0; p.ready = false; p.dirty = true; p.lastHit = 0;
    this.send(ws, { t: 'hello', name: p.name, host: this.isHost(ws), needSetup: false });
    this.send(ws, this.worldMsg());
    this.send(ws, { t: 'respawn', x: p.x, z: p.z });
    this.sendMe(p);
    this.msg(`${p.name} приєднався(лась) до оборони!`, '#4dff7a');
  }
  sendMe(p) {
    p.stats = heroStats(p.profile, this.w.research);
    this.send(p.ws, { t: 'me', profile: p.profile, stats: p.stats, cds: p.cds });
    p.dirty = false;
  }
  worldMsg() {
    return {
      t: 'world',
      towers: [...this.towers.values()].map((t) => ({ id: t.id, type: t.type, level: t.level, x: t.x, z: t.z, spent: t.spent, mode: t.mode || 'first' })),
      extractors: this.w.extractors,
      research: this.w.research,
      camps: this.w.camps,
      cfg: this.w.cfg,
      stats: this.w.stats,
    };
  }
  broadcastWorld() { this.broadcast(this.worldMsg()); }
  players() { return [...this.clients.values()].filter((p) => p.profile); }

  // ───────────── економіка / будівництво ─────────────
  thrift() { return Math.max(0, ...this.players().map((p) => p.stats.thrift)); }
  addTower(type, x, z, level = 1, spent = 0) {
    const t = { id: this.nextId++, type, x, z, level, spent: spent || TOWERS[type].cost, cd: rand(), rot: 0, target: -1, ramp: 0 };
    this.towers.set(t.id, t);
    return t;
  }
  build(h, type, x, z) {
    const def = TOWERS[type];
    if (!def) return;
    if (def.unlock && !this.w.research[def.unlock]) return this.send(h.ws, { t: 'msg', text: 'Цю вежу ще не досліджено.', color: '#f66' });
    x = snap(x); z = snap(z);
    if (!canBuildAt(x, z)) return;
    for (const t of this.towers.values()) if (t.x === x && t.z === z) return;
    const cost = Math.round(def.cost * (1 - this.thrift()));
    if (this.w.credits < cost) return this.send(h.ws, { t: 'msg', text: 'Недостатньо коштів.', color: '#f66' });
    this.w.credits -= cost;
    this.addTower(type, x, z, 1, cost);
    this.events.push({ e: 'build', x, z });
    this.broadcastWorld();
  }
  upgradeTower(h, id) {
    const t = this.towers.get(id);
    if (!t || t.level >= TOWER_MAX_LVL) return;
    const cost = Math.round(towerUpgradeCost(t.type, t.level) * (1 - this.thrift()));
    if (this.w.credits < cost) return this.send(h.ws, { t: 'msg', text: 'Недостатньо коштів.', color: '#f66' });
    this.w.credits -= cost; t.spent += cost; t.level++;
    this.events.push({ e: 'up', x: t.x, z: t.z });
    this.broadcastWorld();
  }
  sellTower(h, id) {
    const t = this.towers.get(id);
    if (!t) return;
    this.w.credits += Math.round(t.spent * 0.7);
    this.towers.delete(id);
    this.events.push({ e: 'sell', x: t.x, z: t.z });
    this.broadcastWorld();
  }
  extract(h, id) {
    const ex = this.w.extractors[id];
    if (!ex) return;
    const cost = ex.level === 0 ? EXTRACTOR_COST : extractorUpgradeCost(ex.level);
    if (this.w.credits < cost) return this.send(h.ws, { t: 'msg', text: 'Недостатньо коштів.', color: '#f66' });
    const d = DEPOSITS[id];
    if (Math.hypot(h.x - d.x, h.z - d.z) > 14) return this.send(h.ws, { t: 'msg', text: 'Підійдіть ближче до родовища.', color: '#f66' });
    this.w.credits -= cost; ex.level++;
    this.events.push({ e: 'up', x: d.x, z: d.z });
    this.broadcastWorld();
  }
  research(h, id) {
    const r = RESEARCH[id];
    if (!r) return;
    const lvl = this.w.research[id] || 0;
    if (lvl >= r.max) return;
    if (r.req && !this.w.research[r.req]) return;
    const cost = researchCost(id, lvl);
    if (this.w.crystals < cost) return this.send(h.ws, { t: 'msg', text: 'Недостатньо запчастин.', color: '#f66' });
    this.w.crystals -= cost;
    this.w.research[id] = lvl + 1;
    const oldMax = this.coreMax;
    this.coreMax = this.calcCoreMax();
    this.coreHp += this.coreMax - oldMax;
    this.msg(`Дослідження: ${r.name} → рівень ${lvl + 1}`, '#c057ff');
    for (const p of this.players()) p.dirty = true;
    this.broadcastWorld();
  }
  wx() { return WEATHER[this.w.weather?.kind] || WEATHER.clear; }
  rollWeather(announce = true) {
    const table = SEASON_WEATHER[seasonOf(this.w.wave)];
    const total = Object.values(table).reduce((a, b) => a + b, 0);
    let r = rand() * total, kind = 'clear';
    for (const [k, v] of Object.entries(table)) { r -= v; if (r <= 0) { kind = k; break; } }
    this.w.weather = { kind, t: 150 + rand() * 180 };
    if (announce) {
      const W = WEATHER[kind];
      this.msg(`Погода: ${W.name}${W.desc ? ' — ' + W.desc : ''}`, '#9cc3d9');
    }
  }
  calcCoreMax() { return 1000 * (1 + 0.15 * (this.w.research.r_core || 0)); }

  // ───────────── герой ─────────────
  talent(h, id) {
    const t = TALENTS[id];
    if (!t || h.profile.points <= 0) return;
    const cur = h.profile.talents[id] || 0;
    if (cur >= t.max) return;
    h.profile.talents[id] = cur + 1;
    h.profile.points--;
    const old = h.stats.maxHp;
    h.stats = heroStats(h.profile, this.w.research);
    h.hp *= h.stats.maxHp / old;
    h.dirty = true;
  }
  respec(h) {
    const spent = Object.values(h.profile.talents).reduce((s, v) => s + v, 0);
    const cost = 20 + spent * 5;
    if (!spent) return;
    if (this.w.crystals < cost) return this.send(h.ws, { t: 'msg', text: `Скидання талантів коштує ${cost} запчастин.`, color: '#f66' });
    this.w.crystals -= cost;
    h.profile.talents = {}; h.profile.points += spent; h.dirty = true;
  }
  equip(h, id) {
    const inv = h.profile.inv;
    const i = inv.findIndex((it) => it.id === id);
    if (i < 0) return;
    const it = inv[i];
    const old = h.profile.equip[it.slot];
    inv.splice(i, 1);
    if (old) inv.push(old);
    h.profile.equip[it.slot] = it;
    const oldMax = h.stats.maxHp;
    h.stats = heroStats(h.profile, this.w.research);
    h.hp = Math.min(h.stats.maxHp, h.hp * h.stats.maxHp / oldMax);
    h.dirty = true;
  }
  unequip(h, slot) {
    if (!SLOTS[slot] || !h.profile.equip[slot] || h.profile.inv.length >= INV_MAX) return;
    h.profile.inv.push(h.profile.equip[slot]);
    h.profile.equip[slot] = null;
    h.stats = heroStats(h.profile, this.w.research);
    h.hp = Math.min(h.hp, h.stats.maxHp);
    h.dirty = true;
  }
  sellItem(h, id, salvage) {
    const inv = h.profile.inv;
    const i = inv.findIndex((it) => it.id === id);
    if (i < 0) return;
    const it = inv.splice(i, 1)[0];
    if (salvage) this.w.crystals += itemSalvage(it); else this.w.credits += itemValue(it);
    h.dirty = true;
  }
  bulk(h, maxRarity, salvage) {
    const keep = [];
    let got = 0;
    for (const it of h.profile.inv) {
      if (it.rarity <= maxRarity) got += salvage ? itemSalvage(it) : itemValue(it);
      else keep.push(it);
    }
    h.profile.inv = keep;
    if (salvage) this.w.crystals += got; else this.w.credits += got;
    this.send(h.ws, { t: 'msg', text: salvage ? `Розібрано: +${got} запчастин` : `Продано: +${got} ₴`, color: '#4dff7a' });
    h.dirty = true;
  }
  giveXp(h, amount) {
    const pr = h.profile;
    pr.xp += amount * h.stats.xpMul;
    let up = false;
    while (pr.xp >= xpToNext(pr.level)) {
      pr.xp -= xpToNext(pr.level);
      pr.level++; pr.points++;
      up = true;
    }
    if (up) {
      h.stats = heroStats(pr, this.w.research);
      h.hp = h.stats.maxHp;
      this.events.push({ e: 'lvl', n: h.name, x: h.x, z: h.z });
      this.msg(`${h.name} досяг(ла) рівня ${pr.level}!`, '#ffd23a');
    }
    h.dirty = true;
  }
  ability(h, m) {
    if (h.dead) return;
    const k = m.k;
    if (!ABILITIES[k] || h.cds[k] > 0) return;
    h.cds[k] = ABILITIES[k].cd * (1 - h.stats.cdr);
    const s = h.stats;
    if (k === 'q') {
      const fx = +m.fx || h.x, fz = +m.fz || h.z;
      this.events.push({ e: 'dash', a: [r1(fx), r1(fz)], b: [r1(+m.x), r1(+m.z)], c: h.profile.color });
      if (s.dashDmg > 0) {
        for (const e of this.enemies.values()) {
          if (segDist(e.x, e.z, fx, fz, +m.x, +m.z) < 2.5 + e.size * 0.5) this.hit(e, s.dashDmg, h, false);
        }
      }
    } else if (k === 'e') {
      let tx = +m.x, tz = +m.z;
      const d = Math.hypot(tx - h.x, tz - h.z);
      if (d > 24) { tx = h.x + (tx - h.x) / d * 24; tz = h.z + (tz - h.z) / d * 24; }
      this.events.push({ e: 'gren', a: [r1(h.x), r1(h.z)], b: [r1(tx), r1(tz)] });
      this.later(0.6, () => this.explode(tx, tz, s.grenadeR, s.grenadeDmg, h, 'gren'));
    } else if (k === 'r') {
      const tx = +m.x, tz = +m.z;
      this.events.push({ e: 'orbit', x: r1(tx), z: r1(tz) });
      this.later(1.2, () => this.explode(tx, tz, 9, s.orbitalDmg, h, 'orbit'));
    }
    h.dirty = true;
  }
  later(t, fn) { this.timers.push({ t, fn }); }
  explode(x, z, r, dmg, h, kind) {
    this.events.push({ e: 'boom', x: r1(x), z: r1(z), r, k: kind });
    for (const e of [...this.enemies.values()]) {
      const d = Math.hypot(e.x - x, e.z - z);
      if (d < r + e.size * 0.5) this.hit(e, dmg * (d < r * 0.4 ? 1 : 0.75), h, false);
    }
  }
  damageHero(h, dmg) {
    if (h.dead) return;
    const a = h.stats.armor;
    dmg = dmg * (60 / (60 + a)) * (1 - h.stats.dr);
    h.hp -= dmg;
    h.lastHit = this.time;
    if (h.hp <= 0) {
      h.dead = true; h.hp = 0; h.respawn = 8;
      h.profile.deaths++;
      this.events.push({ e: 'hdie', x: r1(h.x), z: r1(h.z) });
      this.msg(`${h.name} загинув(ла)! Відродження через 8 с.`, '#ff5a4a');
    }
  }

  // ───────────── вороги ─────────────
  spawnEnemy(kind, x, z, opts = {}) {
    const d = ENEMIES[kind];
    const e = {
      id: this.nextId++, kind, ki: ENEMY_KINDS.indexOf(kind), x, z,
      hp: opts.hp, maxHp: opts.hp, armor: opts.armor ?? d.armor, speed: d.speed * (opts.speedMul || 1),
      size: d.size, flying: !!d.flying, wild: !!d.wild, boss: !!d.boss,
      shield: opts.shield || 0, maxShield: opts.shield || 0, shieldCd: 0,
      lane: opts.lane ?? 0, wp: 1, off: opts.off || 0, prog: 0,
      slowT: 0, slowF: 0, atkCd: 1 + rand(), abCd: 4 + rand() * 2,
      dmg: opts.dmg || 0, bounty: opts.bounty || 0, xp: opts.xp || 0, coreDmg: opts.coreDmg || 0,
      camp: opts.camp ?? -1, state: 'idle', target: null, homeX: x, homeZ: z, wanderT: 0, wx: x, wz: z,
      lvl: opts.lvl || 0, rot: 0,
    };
    this.enemies.set(e.id, e);
    return e;
  }
  spawnWaveEnemy(kind, lane) {
    const w = this.w.wave, diff = this.w.cfg.difficulty;
    const d = ENEMIES[kind];
    const hm = waveHpMul(w, diff) * (d.boss ? bossScale(w) : 1);
    const p = PORTALS[lane];
    const off = (rand() - 0.5) * 3.2;
    const e = this.spawnEnemy(kind, p[0] + (rand() - 0.5) * 2, p[1] + (rand() - 0.5) * 2, {
      hp: d.hp * hm, shield: (d.shield || 0) * hm, armor: d.armor * (1 + 0.04 * w),
      speedMul: 1 + Math.min(0.3, w * 0.005), lane, off,
      dmg: (d.melee || 0) * waveDmgMul(w, diff), coreDmg: d.core * waveDmgMul(w, diff),
      bounty: d.bounty * bountyMul(w, diff), xp: d.xp * (1 + 0.12 * (w - 1)) * Math.pow(1.03, w - 1),
    });
    if (d.boss) {
      this.msg(`⚠ Увага: ${d.name} наближається!`, '#ff3355');
      this.events.push({ e: 'boss', k: kind });
    }
    return e;
  }
  spawnCamp(ci) {
    const c = CAMPS[ci];
    const lvl = this.w.camps[ci].level;
    const st = this.campState[ci];
    const n = 3 + c.tier;
    const kinds = [];
    for (let i = 0; i < n; i++) kinds.push(i % 3 === 2 ? 'w_mg' : 'w_rifle');
    if (c.tier >= 1 || lvl >= 4) kinds[0] = 'w_officer';
    for (const k of kinds) {
      const s = wildStats(k, lvl);
      const a = rand() * Math.PI * 2, r = rand() * 5;
      const e = this.spawnEnemy(k, c.x + Math.cos(a) * r, c.z + Math.sin(a) * r, {
        hp: s.hp, dmg: s.dmg, bounty: s.bounty, xp: s.xp, armor: s.armor, camp: ci, lvl,
      });
      e.homeX = c.x; e.homeZ = c.z;
      st.alive.add(e.id);
    }
  }

  hit(e, dmg, h, canCrit = true, fromTower = false, ap = false) {
    if (!this.enemies.has(e.id)) return 0;
    let crit = false;
    if (h && canCrit && rand() < h.stats.crit) { dmg *= h.stats.critMul; crit = true; }
    if (e.shield > 0) {
      const s = Math.min(e.shield, dmg);
      e.shield -= s; dmg -= s;
      e.shieldCd = 3;
    }
    const real = ap ? dmg : Math.max(dmg * 0.15, dmg - e.armor);
    e.hp -= real;
    if (h) {
      if (h.stats.leech > 0 && !h.dead) h.hp = Math.min(h.stats.maxHp, h.hp + real * h.stats.leech);
      this.events.push({ e: 'dmg', x: r1(e.x), z: r1(e.z), v: Math.round(real), c: crit ? 1 : 0, n: h.name });
      if (e.wild && e.state !== 'leash') { e.state = 'chase'; e.target = h; }
    }
    if (e.hp <= 0) this.kill(e, h, fromTower);
    return real;
  }
  kill(e, h, fromTower) {
    this.enemies.delete(e.id);
    const res = this.w.research;
    const credits = e.bounty * (1 + 0.08 * (res.r_bounty || 0));
    this.w.credits += credits;
    this.w.stats.kills++;
    this.events.push({ e: 'die', x: r1(e.x), z: r1(e.z), k: e.ki, c: Math.round(credits), n: h ? h.name : null });
    // досвід: вбивця отримує повний, решта – половину
    for (const p of this.players()) {
      if (p.dead) continue;
      const share = p === h ? 1 : (e.wild ? (Math.hypot(p.x - e.x, p.z - e.z) < 40 ? 0.5 : 0) : 0.4);
      if (share > 0) this.giveXp(p, e.xp * share);
    }
    if (h) h.profile.kills++;
    // лут
    const luck = (res.r_luck || 0);
    let chance = e.boss ? 1 : e.kind === 'w_officer' ? 0.6 : e.wild ? 0.13 : 0.015;
    chance *= 1 + 0.1 * luck;
    const drops = e.boss ? 3 : 1;
    for (let i = 0; i < drops; i++) {
      if (rand() < chance) {
        const ilvl = e.wild ? e.lvl : this.w.wave;
        const it = genItem(ilvl, rand, luck, e.boss, e.boss && i === 0 ? Math.max(3, Math.floor(rand() * 5)) : -1);
        const a = rand() * Math.PI * 2;
        const l = { id: this.nextId++, x: e.x + Math.cos(a) * (i ? 2 : 0), z: e.z + Math.sin(a) * (i ? 2 : 0), item: it, t: 0 };
        this.loot.set(l.id, l);
      }
    }
    if (e.wild && rand() < 0.35) this.w.crystals += 1 + Math.floor(e.lvl * 0.4);
    if (e.boss) {
      this.w.stats.bosses++;
      this.w.crystals += 40 + this.w.wave * 3;
      this.msg(`★ «${ENEMIES[e.kind].name}» знищено! +${40 + this.w.wave * 3} запчастин`, '#ffa02a');
      if (h) h.profile.bossKills++;
    }
    if (e.camp >= 0) {
      const st = this.campState[e.camp];
      st.alive.delete(e.id);
      if (st.alive.size === 0) {
        st.respawn = 25;
        this.w.camps[e.camp].level++;
        const c = CAMPS[e.camp];
        this.w.crystals += 3 + c.tier * 3;
        this.events.push({ e: 'camp', x: r1(c.x), z: r1(c.z) });
        this.broadcastWorld();
      }
    }
  }

  // ───────────── хвилі ─────────────
  startWave() {
    this.phase = 'wave';
    this.spawnQueue = waveComposition(this.w.wave);
    this.spawnCd = 0;
    this.spawnIdx = 0;
    for (const p of this.players()) p.ready = false;
    this.events.push({ e: 'wave', w: this.w.wave });
    this.msg(`Хвиля ${this.w.wave} атакує! Ворогів: ${this.spawnQueue.length}`, '#ff8a3a');
  }
  waveCleared() {
    const w = this.w.wave;
    const bonus = Math.round((60 + w * 20) * bountyMul(w, this.w.cfg.difficulty));
    this.w.credits += bonus;
    this.w.crystals += 2 + Math.floor(w / 2);
    this.w.stats.bestWave = Math.max(this.w.stats.bestWave, w);
    this.msg(`Хвилю ${w} відбито! +${bonus} ₴, +${2 + Math.floor(w / 2)} запчастин`, '#4dff7a');
    const oldSeason = seasonOf(this.w.wave);
    this.w.wave++;
    if (seasonOf(this.w.wave) !== oldSeason) {
      const S = SEASONS[seasonOf(this.w.wave)];
      this.msg(`Настала ${S.name.toLowerCase()}.`, '#e8d9a8');
      this.events.push({ e: 'season', s: seasonOf(this.w.wave) });
      this.rollWeather();
    }
    this.phase = 'prep';
    this.timer = this.w.cfg.prepTime;
    this.events.push({ e: 'clear' });
    this.save();
  }
  defeat() {
    for (const e of [...this.enemies.values()]) if (!e.wild) this.enemies.delete(e.id);
    this.spawnQueue = [];
    const lc = Math.floor(this.w.credits * 0.25), lx = Math.floor(this.w.crystals * 0.25);
    this.w.credits -= lc; this.w.crystals -= lx;
    this.w.stats.defeats++;
    this.coreHp = this.coreMax;
    this.phase = 'prep';
    this.timer = this.w.cfg.prepTime;
    this.events.push({ e: 'defeat' });
    this.msg(`ШТАБ ЗНИЩЕНО! Втрачено ${lc} ₴ і ${lx} запчастин. Хвиля ${this.w.wave} повториться.`, '#ff3355');
    this.save();
  }

  // ───────────── головний цикл ─────────────
  tick() {
    if (!this.running) return;
    const players = this.players();
    if (!players.length) return; // нікого немає — світ на паузі
    const dt = TICK;
    this.time += dt;
    this.w.stats.playTime += dt;
    const res = this.w.research;

    // таймери
    for (let i = this.timers.length - 1; i >= 0; i--) {
      const t = this.timers[i];
      t.t -= dt;
      if (t.t <= 0) { this.timers.splice(i, 1); t.fn(); }
    }

    // погода
    this.w.weather.t -= dt;
    if (this.w.weather.t <= 0) this.rollWeather();

    // економіка
    this.w.credits += (1 + (res.r_income || 0)) * dt;
    for (const ex of this.w.extractors) this.w.crystals += extractorRate(ex.level, res) * dt;
    this.coreMax = this.calcCoreMax();
    let coreRegen = 0.004 * (res.r_regen || 0) * this.coreMax;
    for (const p of players) if (!p.dead && Math.hypot(p.x, p.z) < 16) coreRegen += p.stats.repair * this.coreMax;
    this.coreHp = Math.min(this.coreMax, this.coreHp + coreRegen * dt);

    // фази
    if (this.phase === 'prep') {
      this.timer -= dt;
      const allReady = players.length > 0 && players.every((p) => p.ready);
      if (allReady && this.timer > 3) {
        const bonus = Math.round(this.timer * 1.5 * bountyMul(this.w.wave, this.w.cfg.difficulty));
        this.w.credits += bonus;
        this.msg(`Достроковий старт: +${bonus} ₴`, '#ffd23a');
        this.timer = 0;
      }
      if (this.timer <= 0) this.startWave();
    } else {
      this.spawnCd -= dt;
      if (this.spawnQueue.length && this.spawnCd <= 0) {
        const kind = this.spawnQueue.shift();
        const lane = this.spawnIdx++ % 3;
        if (kind === 'fpv') {
          for (let i = 0; i < 4 && this.spawnQueue[0] === 'fpv'; i++) this.spawnQueue.shift();
          for (let i = 0; i < 5; i++) this.spawnWaveEnemy('fpv', lane);
        } else this.spawnWaveEnemy(kind, lane);
        this.spawnCd = Math.max(0.22, 0.65 - this.w.wave * 0.006);
      }
      if (!this.spawnQueue.length && ![...this.enemies.values()].some((e) => !e.wild)) this.waveCleared();
    }

    // табори
    this.campState.forEach((st, ci) => {
      if (st.alive.size === 0 && st.respawn > 0) {
        st.respawn -= dt;
        if (st.respawn <= 0) this.spawnCamp(ci);
      }
    });

    // герої
    for (const h of players) {
      for (const k of ['q', 'e', 'r']) if (h.cds[k] > 0) h.cds[k] = Math.max(0, h.cds[k] - dt);
      if (h.dead) {
        h.respawn -= dt;
        if (h.respawn <= 0) {
          h.dead = false; h.hp = h.stats.maxHp;
          const a = rand() * Math.PI * 2;
          h.x = Math.cos(a) * 11; h.z = Math.sin(a) * 11;
          this.send(h.ws, { t: 'respawn', x: h.x, z: h.z });
        }
        continue;
      }
      h.hp = Math.min(h.stats.maxHp, h.hp + h.stats.regen * dt * (this.time - h.lastHit > 4 ? 2 : 1));
      h.fireCd -= dt;
      if (h.fire && h.fireCd <= 0) {
        h.fireCd = 1 / h.stats.rate;
        const dx = h.ax - h.x, dz = h.az - h.z, d = Math.hypot(dx, dz) || 1;
        const sp = 70;
        const side = (h.profile.kills + Math.floor(this.time * 10)) % 2 ? 0.35 : -0.35;
        const sx = h.x + (-dz / d) * side + (dx / d) * 0.9, sz = h.z + (dx / d) * side + (dz / d) * 0.9;
        this.addProjectile({ k: 'hero', x: sx, y: 1.5, z: sz, vx: (dx / d) * sp, vy: 0, vz: (dz / d) * sp, ttl: 0.42, dmg: h.stats.dmg, owner: h, pierce: h.stats.pierce, c: h.profile.color });
      }
      if (h.stats.drone > 0) {
        h.droneCd -= dt;
        if (h.droneCd <= 0) {
          const t = this.nearestEnemy(h.x, h.z, 18);
          if (t) {
            h.droneCd = 0.5;
            const dx = h.x - 1.6, dz = h.z - 1.2;
            this.events.push({ e: 'drop', a: [r1(dx), r1(dz)], x: r1(t.x), z: r1(t.z) });
            void dx; void dz;
            this.hit(t, h.stats.droneDmg, h, true);
          }
        }
      }
      // підбір луту
      for (const l of this.loot.values()) {
        if (Math.hypot(l.x - h.x, l.z - h.z) < 2.4) {
          if (h.profile.inv.length >= INV_MAX) {
            if (!h.invWarn || this.time - h.invWarn > 5) { h.invWarn = this.time; this.send(h.ws, { t: 'msg', text: 'Інвентар повний! Продайте або розберіть речі (Tab).', color: '#f66' }); }
            continue;
          }
          h.profile.inv.push(l.item);
          this.loot.delete(l.id);
          this.events.push({ e: 'pick', n: h.name, r: l.item.rarity, x: r1(l.x), z: r1(l.z) });
          if (l.item.rarity >= 3) this.msg(`${h.name} знайшов(ла): ${l.item.name} [${['', '', '', 'Епічний', 'Легендарний'][l.item.rarity]}]`, l.item.rarity === 4 ? '#ffa02a' : '#c057ff');
          h.dirty = true;
        }
      }
    }
    for (const l of this.loot.values()) { l.t += dt; if (l.t > 300) this.loot.delete(l.id); }

    this.updateEnemies(dt, players);
    this.updateTowers(dt, players);
    this.updateProjectiles(dt, players);

    if (this.coreHp <= 0 && this.phase === 'wave') this.defeat();

    for (const p of players) if (p.dirty) this.sendMe(p);
    this.broadcastSnap(players);
    if (this.time - this.lastSave > 60) { this.lastSave = this.time; this.save(); }
  }

  nearestEnemy(x, z, r, filter) {
    let best = null, bd = r;
    for (const e of this.enemies.values()) {
      if (filter && !filter(e)) continue;
      const d = Math.hypot(e.x - x, e.z - z);
      if (d < bd) { bd = d; best = e; }
    }
    return best;
  }
  nearestHero(x, z, r, players) {
    let best = null, bd = r;
    for (const h of players) {
      if (h.dead) continue;
      const d = Math.hypot(h.x - x, h.z - z);
      if (d < bd) { bd = d; best = h; }
    }
    return best;
  }

  updateEnemies(dt, players) {
    const w = this.w.wave, diff = this.w.cfg.difficulty;
    for (const e of [...this.enemies.values()]) {
      const def = ENEMIES[e.kind];
      let sp = e.speed * (e.flying ? this.wx().air : (e.wild ? 1 : this.wx().ground));
      if (e.slowT > 0) { e.slowT -= dt; sp *= 1 - e.slowF; }
      if (e.maxShield > 0) {
        e.shieldCd -= dt;
        if (e.shieldCd <= 0) e.shield = Math.min(e.maxShield, e.shield + e.maxShield * 0.12 * dt);
      }
      e.atkCd -= dt; e.abCd -= dt;

      if (e.wild) { this.wildAI(e, def, sp, dt, players); continue; }

      // рух по дорозі: витрачаємо весь крок, переносячи залишок через точки маршруту
      if (this.moveAlongLane(e, sp * dt)) {
        this.coreHp -= e.coreDmg;
        this.enemies.delete(e.id);
        this.events.push({ e: 'core', x: r1(e.x), z: r1(e.z), k: e.ki });
        continue;
      }
      // ближній бій з героями
      if (def.melee && e.atkCd <= 0) {
        const h = this.nearestHero(e.x, e.z, def.reach || 2 + e.size * 0.7, players);
        if (h) {
          e.atkCd = def.reach ? 0.9 : 1.2;
          this.damageHero(h, e.dmg);
          this.events.push({ e: 'eshot', a: [r1(e.x), r1(e.z)], b: [r1(h.x), r1(h.z)], y: e.flying ? 6 : 2 + e.size * 0.6, k: def.reach ? 'mg' : 'rifle' });
        }
      }
      // медик
      if (def.heal && e.abCd <= 0) {
        e.abCd = 1;
        for (const o of this.enemies.values()) {
          if (!o.wild && o !== e && Math.hypot(o.x - e.x, o.z - e.z) < 7) o.hp = Math.min(o.maxHp, o.hp + o.maxHp * def.heal);
        }
        this.events.push({ e: 'heal', x: r1(e.x), z: r1(e.z) });
      }
      // боси
      if (e.boss && e.abCd <= 0) {
        if (e.kind === 'boss_heavy') {
          e.abCd = 6;
          // залп головного калібру по найближчому бійцю
          const h = this.nearestHero(e.x, e.z, 30, players);
          if (h) {
            const tx = h.x, tz = h.z;
            this.events.push({ e: 'eshot', a: [r1(e.x), r1(e.z)], b: [r1(tx), r1(tz)], y: 3, k: 'cannon' });
            this.later(0.25, () => {
              this.events.push({ e: 'boom', x: r1(tx), z: r1(tz), r: 6, k: 'shell' });
              for (const p of this.players()) if (!p.dead && Math.hypot(p.x - tx, p.z - tz) < 6) this.damageHero(p, e.dmg * 1.8);
            });
          }
          if (e.hp < e.maxHp * 0.3) e.speed = ENEMIES.boss_heavy.speed * 1.5;
        } else if (e.kind === 'boss_transport') {
          e.abCd = 6;
          for (let i = 0; i < 4; i++) {
            const s = this.spawnWaveEnemy('infantry', e.lane);
            s.x = e.x + (rand() - 0.5) * 5; s.z = e.z + (rand() - 0.5) * 5; s.wp = e.wp;
          }
          this.events.push({ e: 'unload', x: r1(e.x), z: r1(e.z) });
        } else if (e.kind === 'boss_heli') {
          e.abCd = 1.4;
          const h = this.nearestHero(e.x, e.z, 34, players);
          if (h) {
            const ddx = h.x - e.x, ddz = h.z - e.z, dd = Math.hypot(ddx, ddz) || 1;
            const sp = 34;
            this.addProjectile({ k: 'rocket', x: e.x, y: 7, z: e.z, vx: ddx / dd * sp, vy: -7 / (dd / sp), vz: ddz / dd * sp, ttl: dd / sp, dmg: e.dmg, owner: null, hostile: true, blast: 3.5 });
          }
        }
      }
    }
    void w; void diff;
  }

  // Повертає true, якщо ворог дійшов до штабу. Залишок відстані переноситься через повороти,
  // тож швидкі вороги не втрачають час на кожній точці й не проскакують її.
  laneTarget(e) {
    const lane = LANES[e.lane];
    const wp = lane[e.wp], prev = lane[e.wp - 1];
    const dx = wp[0] - prev[0], dz = wp[1] - prev[1], dl = Math.hypot(dx, dz) || 1;
    return [wp[0] + (-dz / dl) * e.off, wp[1] + (dx / dl) * e.off];
  }
  moveAlongLane(e, dist) {
    if (e.flying) {
      const d = Math.hypot(e.x, e.z), reach = 6 + e.size;
      if (d - dist <= reach) return true;
      e.x -= (e.x / d) * dist; e.z -= (e.z / d) * dist;
      e.prog += dist;
      e.rot = Math.atan2(-e.x, -e.z);
      return false;
    }
    const last = LANES[e.lane].length - 1;
    let remaining = dist;
    for (let guard = 0; remaining > 1e-6 && guard < 16; guard++) {
      const [tx, tz] = this.laneTarget(e);
      const dx = tx - e.x, dz = tz - e.z, d = Math.hypot(dx, dz);
      if (d > remaining) {
        e.x += (dx / d) * remaining; e.z += (dz / d) * remaining;
        e.prog += remaining;
        e.rot = Math.atan2(dx, dz);
        return false;
      }
      e.x = tx; e.z = tz; e.prog += d; remaining -= d;
      if (d > 1e-6) e.rot = Math.atan2(dx, dz);
      if (e.wp >= last) return true;
      e.wp++;
    }
    return false;
  }

  wildAI(e, def, sp, dt, players) {
    const home = Math.hypot(e.x - e.homeX, e.z - e.homeZ);
    if (e.state === 'leash') {
      this.moveNav(e, e.homeX, e.homeZ, sp * 1.4, dt);
      e.hp = Math.min(e.maxHp, e.hp + e.maxHp * 0.25 * dt);
      if (home < 3) e.state = 'idle';
      return;
    }
    if (e.state === 'idle') {
      const h = this.nearestHero(e.x, e.z, 13, players);
      if (h) { e.state = 'chase'; e.target = h; this.alertCamp(e, h); return; }
      e.wanderT -= dt;
      if (e.wanderT <= 0) { e.wanderT = 2 + rand() * 4; const a = rand() * 6.28, r = rand() * 6; e.wx = e.homeX + Math.cos(a) * r; e.wz = e.homeZ + Math.sin(a) * r; }
      this.moveNav(e, e.wx, e.wz, sp * 0.3, dt);
      return;
    }
    // chase
    const h = e.target;
    if (!h || h.dead || !this.clients.has(h.ws) || home > 32) { e.state = 'leash'; e.target = null; return; }
    const d = Math.hypot(h.x - e.x, h.z - e.z);
    const range = def.range || 14;
    const los = !this.blockedLOS(e.x, e.z, h.x, h.z);
    if (d > range * 0.85 || !los) this.moveNav(e, h.x, h.z, sp, dt);
    else e.rot = Math.atan2(h.x - e.x, h.z - e.z);
    if (d < range && los && e.atkCd <= 0) {
      e.atkCd = def.burst ? 0.35 : e.kind === 'w_officer' ? 1.1 : 0.9;
      if (def.burst) { e.burstN = (e.burstN || 0) + 1; if (e.burstN % 5 === 0) e.atkCd = 2.2; }
      const dx = h.x - e.x, dz = h.z - e.z, sp2 = 48;
      const spread = (rand() - 0.5) * 0.12;
      const vx = (dx / d) * Math.cos(spread) - (dz / d) * Math.sin(spread), vz = (dz / d) * Math.cos(spread) + (dx / d) * Math.sin(spread);
      this.addProjectile({ k: 'ebullet', x: e.x, y: 1.4, z: e.z, vx: vx * sp2, vy: 0, vz: vz * sp2, ttl: range / sp2 + 0.1, dmg: e.dmg, owner: null, hostile: true });
    }
  }
  blockedLOS(ax, az, bx, bz) { return !!segmentBlocked(ax, az, bx, bz, this.walls, 0.05); }
  // Куди йти до (tx, tz): напряму, або через кути будинку, що перекриває шлях (граф видимості в
  // мініатюрі). Обраний кут фіксується до досягнення — інакше ворог коливається між двома кутами.
  navPoint(e, tx, tz) {
    const wall = segmentBlocked(e.x, e.z, tx, tz, this.walls, 0.6);
    if (!wall) { e.navWp = null; e.navSeen = null; return [tx, tz]; }
    if (e.navWp) {
      if (Math.hypot(e.navWp[0] - e.x, e.navWp[1] - e.z) > 0.6) return e.navWp;
      (e.navSeen ||= []).push(e.navWp);
      e.navWp = null;
    }
    const pad = 1.6, hw = wall.w / 2 + pad, hd = wall.d / 2 + pad;
    let best = null, bc = Infinity;
    for (const [lx, lz] of [[hw, hd], [-hw, hd], [hw, -hd], [-hw, -hd]]) {
      const [cx, cz] = toWorld(lx, lz, wall);
      if (Math.hypot(cx - e.x, cz - e.z) < 0.6) continue;
      if (e.navSeen && e.navSeen.some(([sx, sz]) => Math.hypot(sx - cx, sz - cz) < 0.5)) continue;
      if (segmentBlocked(e.x, e.z, cx, cz, this.walls, 0.3)) continue;
      const cost = Math.hypot(cx - e.x, cz - e.z) + Math.hypot(tx - cx, tz - cz);
      if (cost < bc) { bc = cost; best = [cx, cz]; }
    }
    if (!best) { e.navSeen = null; return [tx, tz]; }
    e.navWp = best;
    return best;
  }
  moveNav(e, tx, tz, sp, dt) {
    const [nx, nz] = this.navPoint(e, tx, tz);
    this.moveTo(e, nx, nz, sp, dt);
    [e.x, e.z] = pushOutOfRects(e.x, e.z, this.walls, 0.5);
  }

  alertCamp(e, h) {
    if (e.camp < 0) return;
    for (const id of this.campState[e.camp].alive) {
      const o = this.enemies.get(id);
      if (o && o.state === 'idle') { o.state = 'chase'; o.target = h; }
    }
  }
  moveTo(e, tx, tz, sp, dt) {
    const dx = tx - e.x, dz = tz - e.z, d = Math.hypot(dx, dz);
    if (d < 0.05) return;
    const s = Math.min(d, sp * dt);
    e.x += dx / d * s; e.z += dz / d * s;
    e.rot = Math.atan2(dx, dz);
  }

  updateTowers(dt, players) {
    const res = this.w.research;
    const auraHeroes = players.filter((p) => !p.dead && p.stats.towerAura > 0);
    for (const t of this.towers.values()) {
      const def = TOWERS[t.type];
      const st = towerStats(t.type, t.level, res);
      let aura = 0;
      for (const h of auraHeroes) if (Math.hypot(h.x - t.x, h.z - t.z) < 22) aura = Math.max(aura, h.stats.towerAura);
      const dmg = st.dmg * (1 + aura);
      t.aura = aura > 0;
      const range = st.range * this.wx().range;
      // цілі — найдальше просунуті вороги в радіусі
      const cands = [];
      const mode = t.mode || 'first';
      for (const e of this.enemies.values()) {
        if (e.wild) continue;
        if (def.ground && e.flying) continue;
        const d = Math.hypot(e.x - t.x, e.z - t.z);
        if (d > range || (def.minRange && d < def.minRange)) continue;
        let score = mode === 'last' ? -e.prog : mode === 'strongest' ? e.hp + e.shield : mode === 'weakest' ? -(e.hp + e.shield) : mode === 'closest' ? -d : e.prog;
        // спеціалізація позиції: ППО — повітря, ПТРК — техніка (поверх режиму)
        if (def.airMul && e.flying) score += 1e7;
        if (def.heavyMul && ENEMIES[e.kind].vehicle) score += 1e7;
        cands.push([score, e]);
      }
      if (!cands.length) { t.target = -1; continue; }
      cands.sort((a, b) => b[0] - a[0]);
      const target = cands[0][1];
      t.rot = Math.atan2(target.x - t.x, target.z - t.z);
      t.target = target.id;
      t.cd -= dt;
      if (t.cd > 0) continue;
      t.cd = (1 / st.rate) * (0.9 + rand() * 0.2);
      const ty = 1.6;
      const fire = (tg, k, speed, extra = {}) => {
        const ey = tg.flying ? (ENEMIES[tg.kind].low ? 2.5 : 6) : 1;
        const dx = tg.x - t.x, dz = tg.z - t.z, d = Math.hypot(dx, dz);
        const T = Math.hypot(d, ey - ty) / speed;
        this.addProjectile({ k, x: t.x, y: ty, z: t.z, vx: dx / T, vy: (ey - ty) / T, vz: dz / T, ttl: T + 0.05, owner: null, homing: tg.id, ...extra });
      };
      const arc = (tg, k, T, extra) => {
        const g = 30;
        const lead = Math.min(1, T * 0.8);
        const tx = tg.x + (tg.x - (tg.px ?? tg.x)) * 0, tz = tg.z;
        void lead;
        this.addProjectile({ k, x: t.x, y: ty + 0.8, z: t.z, vx: (tx - t.x) / T, vy: (0 - ty - 0.8) / T + 0.5 * g * T, vz: (tz - t.z) / T, ttl: T, owner: null, g, ...extra });
      };
      switch (t.type) {
        case 'mg': fire(target, 'bullet', 90, { dmg }); break;
        case 'ags': arc(target, 'agsg', 0.8, { dmg, splash: st.splash, slow: st.slow, slowT: def.slowT, ground: true }); break;
        case 'aa': {
          for (let i = 0; i < Math.min(st.multi, cands.length); i++) {
            const tg = cands[i][1];
            fire(tg, 'aa', 110, { dmg: dmg * (tg.flying ? def.airMul : 1) });
          }
          break;
        }
        case 'atgm': fire(target, 'atgm', 38, { dmg: dmg * (ENEMIES[target.kind].vehicle ? def.heavyMul : 1), ap: true }); break;
        case 'mortar': arc(target, 'mortar', 1.6, { dmg, splash: st.splash, ground: true }); break;
        case 'howitzer': arc(target, 'shell', 2.4, { dmg, splash: st.splash, ground: true }); break;
      }
      this.events.push({ e: 'tf', id: t.id });
    }
  }

  addProjectile(p) {
    p.id = this.nextId++;
    p.g = p.g || 0;
    this.projectiles.set(p.id, p);
    this.events.push({ e: 'ps', id: p.id, k: p.k, p: [r2(p.x), r2(p.y), r2(p.z)], v: [r2(p.vx), r2(p.vy), r2(p.vz)], g: p.g, ttl: r2(p.ttl), c: p.c });
  }
  updateProjectiles(dt, players) {
    for (const p of [...this.projectiles.values()]) {
      const ox = p.x, oz = p.z;
      if (p.homing) {
        const tg = this.enemies.get(p.homing);
        if (tg) {
          // м'яке наведення не змінюємо візуально — просто зараховуємо влучання
          const d = Math.hypot(tg.x - p.x, tg.z - p.z);
          const spd = Math.hypot(p.vx, p.vz) || 50;
          if (d < 1.2 + tg.size * 0.5 + spd * dt) { this.hit(tg, p.dmg, null, false, true, p.ap); this.endProjectile(p, tg.x, tg.z); continue; }
        }
      }
      p.x += p.vx * dt; p.y += p.vy * dt - 0.5 * p.g * dt * dt; p.z += p.vz * dt;
      p.vy -= p.g * dt;
      p.ttl -= dt;
      // кулі (не снаряди навісного вогню й не ракети з висоти) зупиняються об будинки
      if ((p.k === 'hero' || p.k === 'ebullet') && segmentBlocked(ox, oz, p.x, p.z, this.walls)) {
        this.events.push({ e: 'wallhit', x: r1(p.x), z: r1(p.z) });
        this.endProjectile(p, p.x, p.z);
        continue;
      }
      if (p.hostile) {
        for (const h of players) {
          if (!h.dead && segDist(h.x, h.z, ox, oz, p.x, p.z) < (p.blast ? 1.6 : 0.9)) { this.damageHero(h, p.dmg); p.ttl = -1; break; }
        }
        if (p.ttl <= 0) {
          if (p.blast) this.events.push({ e: 'boom', x: r1(p.x), z: r1(p.z), r: p.blast, k: 'rocket' });
          this.endProjectile(p, p.x, p.z);
        }
        continue;
      }
      if (p.k === 'hero') {
        let ended = false;
        p.hitSet = p.hitSet || new Set();
        for (const e of this.enemies.values()) {
          if (p.hitSet.has(e.id)) continue;
          if (segDist(e.x, e.z, ox, oz, p.x, p.z) < 0.6 + e.size * 0.55) {
            this.hit(e, p.dmg, p.owner && this.clients.has(p.owner.ws) ? p.owner : null, true);
            p.hitSet.add(e.id);
            this.events.push({ e: 'spark', x: r1(e.x), z: r1(e.z), y: e.flying ? 4 : 1.2, c: p.c });
            if (rand() >= p.pierce) { ended = true; break; }
          }
        }
        if (ended) { this.endProjectile(p, p.x, p.z); continue; }
      }
      if (p.ttl <= 0) {
        if (p.splash) {
          this.events.push({ e: 'boom', x: r1(p.x), z: r1(p.z), r: p.splash, k: p.k });
          for (const e of [...this.enemies.values()]) {
            if (e.wild || (p.ground && e.flying && !ENEMIES[e.kind].low)) continue;
            if (Math.hypot(e.x - p.x, e.z - p.z) < p.splash + e.size * 0.4) {
              if (p.slow) { e.slowT = p.slowT; e.slowF = e.boss ? p.slow * 0.5 : p.slow; }
              this.hit(e, p.dmg, null, false, true);
            }
          }
        }
        this.endProjectile(p, p.x, p.z);
      }
    }
  }
  endProjectile(p, x, z) {
    this.projectiles.delete(p.id);
    this.events.push({ e: 'ph', id: p.id, x: r1(x), z: r1(z), k: p.k });
  }

  broadcastSnap(players) {
    const enemies = [];
    for (const e of this.enemies.values()) {
      enemies.push([e.id, e.ki, r1(e.x), r1(e.z), r2(Math.max(0, e.hp / e.maxHp)), e.maxShield ? r2(e.shield / e.maxShield) : -1, e.slowT > 0 ? 1 : 0, r2(e.rot), e.lvl]);
    }
    const snap = {
      t: 'snap',
      time: r2(this.time),
      wave: this.w.wave, phase: this.phase, sea: seasonOf(this.w.wave), wx: this.w.weather.kind, timer: Math.max(0, Math.ceil(this.timer)),
      left: this.phase === 'wave' ? this.spawnQueue.length + [...this.enemies.values()].filter((e) => !e.wild).length : 0,
      credits: Math.floor(this.w.credits), crystals: Math.floor(this.w.crystals),
      core: [Math.round(this.coreHp), Math.round(this.coreMax)],
      players: players.map((p) => ({
        n: p.name, x: r2(p.x), z: r2(p.z), r: r2(p.rot), hp: Math.round(p.hp), mhp: Math.round(p.stats.maxHp),
        l: p.profile.level, d: p.dead ? 1 : 0, rs: Math.ceil(p.respawn), c: p.profile.color, f: p.fire && !p.dead ? 1 : 0,
        rd: p.ready ? 1 : 0, dr: p.stats.drone > 0 ? 1 : 0, cds: p.cds,
      })),
      enemies,
      towers: [...this.towers.values()].map((t) => [t.id, r2(t.rot), t.target, t.aura ? 1 : 0]),
      loot: [...this.loot.values()].map((l) => [l.id, r1(l.x), r1(l.z), l.item.rarity]),
      ev: this.events,
    };
    this.events = [];
    this.broadcast(snap);
  }
}

function clamp(v, a, b) { return Math.max(a, Math.min(b, v)); }
function clampPrep(v) { return clamp(Math.round(+v || 120), 20, 900); }
function segDist(px, pz, ax, az, bx, bz) {
  const dx = bx - ax, dz = bz - az;
  const l2 = dx * dx + dz * dz || 1e-6;
  let t = ((px - ax) * dx + (pz - az) * dz) / l2;
  t = Math.max(0, Math.min(1, t));
  return Math.hypot(px - (ax + dx * t), pz - (az + dz * t));
}

// Windows: дозволяємо всім користувачам ПК перезаписувати спільні збереження (RDP-сесії під різними обліковими записами).
function shareFolder(dir) {
  if (process.platform !== 'win32') return;
  try { execFileSync('icacls', [dir, '/grant', '*S-1-5-32-545:(OI)(CI)F', '/T', '/C', '/Q'], { stdio: 'ignore', windowsHide: true }); } catch { /* немає прав — не страшно */ }
}
