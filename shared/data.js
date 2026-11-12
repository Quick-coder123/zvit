// Спільні дані та формули для сервера і клієнта.
// Усе, що впливає на баланс, живе тут.

export const PORT = 47621;
export const TICK = 1 / 20;

// ───────────────────────── Світ ─────────────────────────
export const WORLD = 125;            // половина розміру карти
export const BASE_R = 38;            // радіус зони будівництва
export const CORE_R = 8;             // вільна зона довкола ядра
export const GRID = 4;               // крок сітки будівництва
export const LANE_CLEAR = 3.6;       // відступ веж від доріг

const LANE_ANGLES = [-Math.PI / 2, Math.PI / 6, (5 * Math.PI) / 6];
const LANE_PROFILE = [
  [114, 0], [96, 0.13], [80, -0.11], [64, 0.16], [50, -0.14], [36, 0.12], [22, -0.05], [9, 0],
];
export const LANES = LANE_ANGLES.map((a) =>
  LANE_PROFILE.map(([r, off]) => [Math.cos(a + off) * r, Math.sin(a + off) * r]),
);
export const PORTALS = LANES.map((l) => l[0]);

const SECTORS = LANE_ANGLES.map((a) => a + Math.PI / 3); // середини між дорогами

export const DEPOSITS = [];
SECTORS.forEach((a, s) => {
  DEPOSITS.push({ id: DEPOSITS.length, x: Math.cos(a) * 54, z: Math.sin(a) * 54, sector: s });
  DEPOSITS.push({ id: DEPOSITS.length, x: Math.cos(a + 0.28) * 92, z: Math.sin(a + 0.28) * 92, sector: s });
});

// Табори дикої фауни: tier → стартовий рівень
export const CAMP_TIERS = [
  { lvl: 1, name: 'Околиці' },
  { lvl: 5, name: 'Пустища' },
  { lvl: 10, name: 'Кратер' },
  { lvl: 16, name: 'Вогнище вулика' },
];
export const CAMPS = [];
SECTORS.forEach((a, s) => {
  const spots = [
    [62, -0.32, 0], [66, 0.3, 0],
    [80, -0.05, 1], [86, -0.38, 1],
    [102, 0.05, 2], [104, -0.3, 2],
    [112, 0.36, 3],
  ];
  for (const [r, off, tier] of spots) {
    CAMPS.push({ id: CAMPS.length, x: Math.cos(a + off) * r, z: Math.sin(a + off) * r, tier, sector: s });
  }
});

export function distToSeg(px, pz, ax, az, bx, bz) {
  const dx = bx - ax, dz = bz - az;
  const l2 = dx * dx + dz * dz || 1;
  let t = ((px - ax) * dx + (pz - az) * dz) / l2;
  t = Math.max(0, Math.min(1, t));
  const cx = ax + dx * t, cz = az + dz * t;
  return Math.hypot(px - cx, pz - cz);
}
export function distToLanes(x, z) {
  let d = Infinity;
  for (const l of LANES) for (let i = 0; i < l.length - 1; i++) d = Math.min(d, distToSeg(x, z, l[i][0], l[i][1], l[i + 1][0], l[i + 1][1]));
  return d;
}
export function snap(v) { return Math.round(v / GRID) * GRID; }
export function canBuildAt(x, z) {
  const r = Math.hypot(x, z);
  if (r > BASE_R || r < CORE_R + 1) return false;
  return distToLanes(x, z) >= LANE_CLEAR;
}

// ───────────────────────── Складність ─────────────────────────
export const DIFFICULTY = {
  easy: { name: 'Легко', growth: 1.066, hp: 0.8, dmg: 0.75, bounty: 1.15 },
  normal: { name: 'Нормально', growth: 1.087, hp: 1.0, dmg: 1.0, bounty: 1.0 },
  hard: { name: 'Жорстко', growth: 1.106, hp: 1.25, dmg: 1.3, bounty: 0.9 },
};
// Сила хвилі залежить ЛИШЕ від номера хвилі і складності, а не від гравців.
export function waveHpMul(w, diff) {
  const d = DIFFICULTY[diff] || DIFFICULTY.normal;
  return d.hp * (1 + 0.1 * (w - 1)) * Math.pow(d.growth, w - 1);
}
export function waveDmgMul(w, diff) {
  const d = DIFFICULTY[diff] || DIFFICULTY.normal;
  return d.dmg * (1 + 0.05 * (w - 1)) * Math.pow(1.03, w - 1);
}
export function bountyMul(w, diff) {
  const d = DIFFICULTY[diff] || DIFFICULTY.normal;
  return d.bounty * (1 + 0.05 * (w - 1)) * Math.pow(1.035, w - 1);
}

// ───────────────────────── Вороги ─────────────────────────
export const ENEMIES = {
  crawler: { name: 'Повзун', hp: 60, speed: 4.6, armor: 0, core: 15, bounty: 6, xp: 6, size: 1.0, color: 0xff3355 },
  runner: { name: 'Бігун', hp: 34, speed: 8.2, armor: 0, core: 10, bounty: 5, xp: 5, size: 0.8, color: 0xffaa22 },
  swarm: { name: 'Рій', hp: 15, speed: 6.6, armor: 0, core: 5, bounty: 2, xp: 2, size: 0.55, color: 0xaaff33 },
  brute: { name: 'Громило', hp: 280, speed: 2.8, armor: 6, core: 40, bounty: 18, xp: 18, size: 1.7, color: 0xff6622, melee: 22 },
  flyer: { name: 'Шершень', hp: 52, speed: 6.2, armor: 0, core: 14, bounty: 9, xp: 8, size: 0.9, color: 0x33ddff, flying: true },
  shielder: { name: 'Щитоносець', hp: 120, shield: 140, speed: 3.6, armor: 2, core: 25, bounty: 15, xp: 14, size: 1.25, color: 0x6688ff },
  healer: { name: 'Медик', hp: 95, speed: 4.0, armor: 0, core: 15, bounty: 15, xp: 14, size: 1.05, color: 0x44ffaa, heal: 0.06 },
  boss_titan: { name: 'Титан', hp: 5200, speed: 2.1, armor: 14, core: 400, bounty: 400, xp: 400, size: 4.2, color: 0xff2222, boss: true, melee: 70 },
  boss_hive: { name: 'Матка рою', hp: 4300, speed: 2.4, armor: 6, core: 400, bounty: 400, xp: 400, size: 3.8, color: 0x99ff22, boss: true, melee: 40 },
  boss_sentinel: { name: 'Вартовий порожнечі', hp: 4700, shield: 1800, speed: 2.3, armor: 8, core: 400, bounty: 450, xp: 450, size: 3.8, color: 0xaa44ff, boss: true, melee: 50, flying: true },
  w_mutant: { name: 'Мутант', hp: 80, speed: 5.5, armor: 0, bounty: 8, xp: 14, size: 1.1, color: 0xcc7744, wild: true, melee: 9 },
  w_spitter: { name: 'Плювач', hp: 60, speed: 4.5, armor: 0, bounty: 8, xp: 14, size: 0.95, color: 0x88ff44, wild: true, ranged: 7 },
  w_alpha: { name: 'Альфа', hp: 320, speed: 4.8, armor: 4, bounty: 30, xp: 50, size: 2.0, color: 0xffcc33, wild: true, melee: 22 },
};
export const ENEMY_KINDS = Object.keys(ENEMIES);
export const BOSSES = ['boss_titan', 'boss_hive', 'boss_sentinel'];

export function waveComposition(w) {
  const total = Math.min(110, Math.floor(10 + w * 1.7));
  const pool = [['crawler', 10]];
  if (w >= 2) pool.push(['runner', 6]);
  if (w >= 4) pool.push(['swarm', 4]);
  if (w >= 5) pool.push(['brute', 3 + w * 0.05]);
  if (w >= 7) pool.push(['flyer', 4]);
  if (w >= 9) pool.push(['shielder', 3 + w * 0.04]);
  if (w >= 12) pool.push(['healer', 2]);
  const sum = pool.reduce((s, p) => s + p[1], 0);
  const list = [];
  for (const [kind, wt] of pool) {
    let n = Math.round((total * wt) / sum);
    if (kind === 'swarm') n = Math.max(1, Math.round(n / 5)) * 5;
    for (let i = 0; i < n; i++) list.push(kind);
  }
  // перемішуємо детерміновано
  let seed = w * 9973;
  for (let i = list.length - 1; i > 0; i--) {
    seed = (seed * 16807) % 2147483647;
    const j = seed % (i + 1);
    [list[i], list[j]] = [list[j], list[i]];
  }
  if (w % 10 === 0) list.splice(Math.floor(list.length / 2), 0, BOSSES[(w / 10 - 1) % 3]);
  return list;
}

export function wildStats(kind, lvl) {
  const e = ENEMIES[kind];
  return {
    hp: e.hp * Math.pow(1.17, lvl - 1) * (1 + 0.05 * lvl),
    dmg: (e.melee || e.ranged) * Math.pow(1.13, lvl - 1),
    bounty: e.bounty * Math.pow(1.1, lvl - 1),
    xp: e.xp * Math.pow(1.14, lvl - 1),
    armor: e.armor * (1 + lvl * 0.08),
  };
}

// ───────────────────────── Вежі ─────────────────────────
export const TOWERS = {
  blaster: { name: 'Бластер', key: '1', cost: 100, dmg: 13, rate: 4, range: 18, color: 0x33e0ff, desc: 'Швидкі енергетичні заряди по одній цілі.', unlock: null },
  cryo: { name: 'Кріопроектор', key: '2', cost: 150, dmg: 7, rate: 2, range: 15, slow: 0.42, slowT: 2, splash: 3.5, color: 0x99ccff, desc: 'Сповільнює ворогів у невеликому радіусі.', unlock: 'u_cryo' },
  tesla: { name: 'Дуговий вузол', key: '3', cost: 200, dmg: 30, rate: 1.25, range: 14, chain: 4, color: 0xbb77ff, desc: 'Блискавка перескакує між ворогами.', unlock: 'u_tesla' },
  rail: { name: 'Рейкотрон', key: '4', cost: 260, dmg: 135, rate: 0.5, range: 34, pierce: true, color: 0xffee55, desc: 'Пробиває всіх ворогів на лінії. Далекобійний.', unlock: 'u_rail' },
  mortar: { name: 'Плазмова мортира', key: '5', cost: 320, dmg: 65, rate: 0.6, range: 32, minRange: 6, splash: 5.5, ground: true, color: 0xff7733, desc: 'Площинна шкода. Не б’є по літаючих.', unlock: 'u_mortar' },
  laser: { name: 'Фазовий лазер', key: '6', cost: 420, dmg: 42, rate: 0, range: 20, ramp: 2.5, color: 0xff3388, desc: 'Безперервний промінь, що розпікається на одній цілі.', unlock: 'u_laser' },
};
export const TOWER_TYPES = Object.keys(TOWERS);
export const TOWER_MAX_LVL = 25;

export function towerUpgradeCost(type, lvl) {
  return Math.round(TOWERS[type].cost * 0.75 * Math.pow(1.42, lvl - 1));
}
export function towerStats(type, lvl, research = {}) {
  const t = TOWERS[type];
  const r = (id) => research[id] || 0;
  const dmgMul = Math.pow(1.24, lvl - 1) * (1 + 0.1 * r('r_dmg'));
  return {
    dmg: t.dmg * dmgMul,
    rate: t.rate * (1 + 0.025 * (lvl - 1)) * (1 + 0.05 * r('r_rate')),
    range: t.range * (1 + 0.012 * (lvl - 1)) * (1 + 0.04 * r('r_range')),
    splash: t.splash ? t.splash * (1 + 0.02 * (lvl - 1)) : 0,
    slow: t.slow ? Math.min(0.75, t.slow + 0.012 * (lvl - 1)) : 0,
    chain: t.chain ? t.chain + Math.floor((lvl - 1) / 4) : 0,
  };
}

// ───────────────────────── Шахти кристалів ─────────────────────────
export const EXTRACTOR_COST = 300;
export function extractorUpgradeCost(lvl) { return Math.round(300 * Math.pow(1.65, lvl)); }
export function extractorRate(lvl, research = {}) {
  if (lvl <= 0) return 0;
  return 0.35 * Math.pow(1.35, lvl - 1) * (1 + 0.15 * (research.r_mine || 0));
}

// ───────────────────────── Дослідження (кристали) ─────────────────────────
export const RESEARCH = {
  u_cryo: { name: 'Кріотехнології', desc: 'Відкриває вежу «Кріопроектор».', base: 40, max: 1, branch: 'Вежі' },
  u_tesla: { name: 'Дугова фізика', desc: 'Відкриває вежу «Дуговий вузол».', base: 80, max: 1, branch: 'Вежі', req: 'u_cryo' },
  u_rail: { name: 'Магнітні рейки', desc: 'Відкриває вежу «Рейкотрон».', base: 150, max: 1, branch: 'Вежі', req: 'u_tesla' },
  u_mortar: { name: 'Плазмова балістика', desc: 'Відкриває вежу «Плазмова мортира».', base: 220, max: 1, branch: 'Вежі', req: 'u_rail' },
  u_laser: { name: 'Фазові емітери', desc: 'Відкриває вежу «Фазовий лазер».', base: 400, max: 1, branch: 'Вежі', req: 'u_mortar' },
  r_dmg: { name: 'Озброєння', desc: '+10% шкоди всіх веж за рівень.', base: 30, max: 40, branch: 'Вежі' },
  r_rate: { name: 'Сервоприводи', desc: '+5% швидкості стрільби веж.', base: 45, max: 20, branch: 'Вежі' },
  r_range: { name: 'Оптика', desc: '+4% дальності веж.', base: 50, max: 12, branch: 'Вежі' },
  r_core: { name: 'Броня ядра', desc: '+15% міцності ядра.', base: 35, max: 30, branch: 'База' },
  r_regen: { name: 'Нанорепарація', desc: 'Ядро відновлює +0.4% міцності/с.', base: 60, max: 10, branch: 'База' },
  r_mine: { name: 'Глибинне буріння', desc: '+15% видобутку кристалів.', base: 40, max: 30, branch: 'Економіка' },
  r_bounty: { name: 'Трофейна служба', desc: '+8% кредитів за вбивства.', base: 40, max: 30, branch: 'Економіка' },
  r_income: { name: 'Торговий вузол', desc: '+1 кредит/с пасивного доходу.', base: 50, max: 20, branch: 'Економіка' },
  r_team: { name: 'Бойова підготовка', desc: 'Герої: +5% шкоди та здоров’я.', base: 50, max: 30, branch: 'Герої' },
  r_luck: { name: 'Сканери трофеїв', desc: '+10% шанс луту та кращої рідкості.', base: 70, max: 15, branch: 'Герої' },
  r_xp: { name: 'Нейроінтерфейс', desc: '+8% досвіду героям.', base: 60, max: 20, branch: 'Герої' },
};
export function researchCost(id, lvl) {
  return Math.round(RESEARCH[id].base * Math.pow(1.55, lvl));
}

// ───────────────────────── Герої ─────────────────────────
export function xpToNext(lvl) { return Math.round(90 * Math.pow(1.22, lvl - 1) + 40 * lvl); }

export const BRANCHES = {
  vanguard: { name: 'Штурмовик', color: '#ff5a4a', desc: 'Живучість і ближній бій' },
  gunner: { name: 'Стрілець', color: '#ffd23a', desc: 'Шкода, критичні удари' },
  engineer: { name: 'Технік', color: '#3ae0ff', desc: 'Вежі, дрон, ядро' },
};
export const TALENTS = {
  plates: { branch: 'vanguard', name: 'Броньові пластини', desc: '+10% здоров’я, +2 броні', max: 10 },
  regen: { branch: 'vanguard', name: 'Регенерація', desc: '+0.6% здоров’я/с', max: 10 },
  dashstrike: { branch: 'vanguard', name: 'Таранний ривок', desc: 'Ривок (Q) завдає шкоди: +80% атаки за рівень', max: 10 },
  leech: { branch: 'vanguard', name: 'Вампіризм', desc: '+2% вампіризму', max: 10 },
  bulwark: { branch: 'vanguard', name: 'Незламність', desc: '−4% вхідної шкоди', max: 10 },
  caliber: { branch: 'gunner', name: 'Калібр', desc: '+10% шкоди', max: 10 },
  trigger: { branch: 'gunner', name: 'Спусковий механізм', desc: '+6% швидкості стрільби', max: 10 },
  crit: { branch: 'gunner', name: 'Влучність', desc: '+4% шансу криту, +10% крит. шкоди', max: 10 },
  pierce: { branch: 'gunner', name: 'Бронебійні заряди', desc: '+8% шансу пробити ціль наскрізь', max: 10 },
  grenade: { branch: 'gunner', name: 'Плазмова граната+', desc: 'Граната (E): +20% шкоди, +5% радіусу', max: 10 },
  aura: { branch: 'engineer', name: 'Аура підсилення', desc: 'Вежі в радіусі 22 м: +5% шкоди', max: 10 },
  drone: { branch: 'engineer', name: 'Бойовий дрон', desc: 'Дрон-супутник стріляє по ворогах', max: 10 },
  repair: { branch: 'engineer', name: 'Польовий ремонт', desc: 'Біля ядра лікує його на 0.15%/с', max: 10 },
  thrift: { branch: 'engineer', name: 'Ощадливість', desc: '−2% вартості веж і покращень', max: 10 },
  orbital: { branch: 'engineer', name: 'Орбітальний канал', desc: 'Орбітальний удар (R): +25% шкоди', max: 10 },
};

export const RARITIES = [
  { id: 'common', name: 'Звичайний', color: '#c8d0d8', affixes: 1, mul: 1 },
  { id: 'uncommon', name: 'Незвичайний', color: '#4dff7a', affixes: 2, mul: 1.15 },
  { id: 'rare', name: 'Рідкісний', color: '#3aa6ff', affixes: 3, mul: 1.35 },
  { id: 'epic', name: 'Епічний', color: '#c057ff', affixes: 4, mul: 1.6 },
  { id: 'legendary', name: 'Легендарний', color: '#ffa02a', affixes: 5, mul: 2.0 },
];
export const SLOTS = { weapon: 'Зброя', armor: 'Броня', module: 'Модуль' };
const ITEM_NAMES = {
  weapon: ['Імпульсна гвинтівка', 'Плазмовий карабін', 'Іонний пістолет', 'Фотонний бластер', 'Гаус-гвинтівка', 'Кварковий випромінювач'],
  armor: ['Штурмовий екзокостюм', 'Наноброня', 'Кевларовий жилет', 'Силовий панцир', 'Титановий каркас', 'Броня «Обсидіан»'],
  module: ['Тактичний чип', 'Нейроприскорювач', 'Енергоядро', 'Квантовий модуль', 'Імплант «Сокіл»', 'Кристалічна матриця'],
};
export const AFFIXES = {
  dmgPct: { name: 'Шкода', fmt: (v) => `+${v.toFixed(0)}%`, roll: (l) => 4 + l * 0.6 },
  ratePct: { name: 'Швидкість стрільби', fmt: (v) => `+${v.toFixed(0)}%`, roll: (l) => 3 + l * 0.3 },
  hpPct: { name: 'Здоров’я', fmt: (v) => `+${v.toFixed(0)}%`, roll: (l) => 5 + l * 0.6 },
  armor: { name: 'Броня', fmt: (v) => `+${v.toFixed(0)}`, roll: (l) => 1 + l * 0.6 },
  crit: { name: 'Шанс криту', fmt: (v) => `+${v.toFixed(1)}%`, roll: (l) => 1.5 + l * 0.12 },
  speedPct: { name: 'Швидкість руху', fmt: (v) => `+${v.toFixed(0)}%`, roll: (l) => 2 + Math.min(12, l * 0.15) },
  regen: { name: 'Регенерація', fmt: (v) => `+${v.toFixed(1)}/с`, roll: (l) => 0.5 + l * 0.35 },
  leech: { name: 'Вампіризм', fmt: (v) => `+${v.toFixed(1)}%`, roll: (l) => 0.8 + l * 0.06 },
  cdr: { name: 'Перезарядка вмінь', fmt: (v) => `−${v.toFixed(0)}%`, roll: (l) => 2 + Math.min(18, l * 0.2) },
  towerPct: { name: 'Аура веж', fmt: (v) => `+${v.toFixed(0)}%`, roll: (l) => 2 + l * 0.3 },
  xpPct: { name: 'Досвід', fmt: (v) => `+${v.toFixed(0)}%`, roll: (l) => 3 + l * 0.25 },
};
const AFFIX_KEYS = Object.keys(AFFIXES);

export function rollRarity(rand, luck = 0, boss = false) {
  const r = rand() / (1 + luck * 0.1) - (boss ? 0.1 : 0);
  if (r < 0.012) return 4;
  if (r < 0.06) return 3;
  if (r < 0.2) return 2;
  if (r < 0.48) return 1;
  return 0;
}
export function genItem(ilvl, rand, luck = 0, boss = false, forceRarity = -1) {
  const slotKeys = Object.keys(SLOTS);
  const slot = slotKeys[Math.floor(rand() * slotKeys.length)];
  const ri = forceRarity >= 0 ? forceRarity : rollRarity(rand, luck, boss);
  const rar = RARITIES[ri];
  const names = ITEM_NAMES[slot];
  const base = names[Math.min(names.length - 1, Math.floor(ilvl / 8 + rand() * 2))];
  const stats = {};
  // основна характеристика слоту
  if (slot === 'weapon') stats.atk = Math.round((6 + ilvl * 2.4 * Math.pow(1.035, ilvl)) * rar.mul * (0.85 + rand() * 0.3));
  if (slot === 'armor') stats.hp = Math.round((30 + ilvl * 12 * Math.pow(1.035, ilvl)) * rar.mul * (0.85 + rand() * 0.3));
  if (slot === 'module') stats.cdr = Math.round(Math.min(25, (3 + ilvl * 0.25) * rar.mul));
  const pool = AFFIX_KEYS.filter((k) => !(slot === 'module' && k === 'cdr'));
  for (let i = 0; i < rar.affixes && pool.length; i++) {
    const k = pool.splice(Math.floor(rand() * pool.length), 1)[0];
    stats[k] = +(AFFIXES[k].roll(ilvl) * rar.mul * (0.8 + rand() * 0.4)).toFixed(1);
  }
  const prefix = ri === 4 ? '★ ' : '';
  return {
    id: Math.floor(rand() * 1e9).toString(36) + Date.now().toString(36).slice(-4),
    slot, rarity: ri, ilvl, name: prefix + base, stats,
  };
}
export function itemValue(item) {
  return Math.round((20 + item.ilvl * 8) * Math.pow(1.8, item.rarity));
}
export function itemSalvage(item) {
  return Math.round((2 + item.ilvl * 0.6) * Math.pow(2, item.rarity));
}
export function itemPower(item) {
  const s = item.stats;
  let p = (s.atk || 0) * 3 + (s.hp || 0) * 0.5 + (s.cdr || 0) * 6;
  for (const k of AFFIX_KEYS) if (s[k] && k !== 'cdr') p += s[k] * 4;
  return Math.round(p);
}

export function newHeroProfile(name) {
  return {
    name, level: 1, xp: 0, points: 0,
    talents: {}, equip: { weapon: null, armor: null, module: null }, inv: [],
    kills: 0, deaths: 0, bossKills: 0,
    color: null,
  };
}

// Повний розрахунок характеристик героя (і сервер, і клієнт).
export function heroStats(p, research = {}) {
  const t = (k) => p.talents[k] || 0;
  const team = 1 + 0.05 * (research.r_team || 0);
  const lv = p.level;
  const g = { atk: 0, hp: 0, cdr: 0 };
  for (const k of AFFIX_KEYS) g[k] = 0;
  for (const slot of Object.keys(SLOTS)) {
    const it = p.equip[slot];
    if (!it) continue;
    for (const [k, v] of Object.entries(it.stats)) g[k] = (g[k] || 0) + v;
  }
  const maxHp = (220 + lv * 22 + g.hp) * Math.pow(1.035, lv - 1) * (1 + 0.1 * t('plates') + g.hpPct / 100) * team;
  const dmg = (14 + lv * 1.6 + g.atk) * Math.pow(1.03, lv - 1) * (1 + 0.1 * t('caliber') + g.dmgPct / 100) * team;
  return {
    maxHp,
    dmg,
    rate: 5 * (1 + 0.06 * t('trigger') + g.ratePct / 100),
    crit: Math.min(0.75, 0.05 + 0.04 * t('crit') + g.crit / 100),
    critMul: 2 + 0.1 * t('crit'),
    armor: 2 * t('plates') + g.armor + lv * 0.3,
    speed: 10 * (1 + Math.min(0.5, g.speedPct / 100)),
    regen: maxHp * 0.006 * t('regen') + g.regen + maxHp * 0.002,
    leech: (0.02 * t('leech') + g.leech / 100),
    dr: Math.min(0.5, 0.04 * t('bulwark')),
    cdr: Math.min(0.5, (g.cdr || 0) / 100),
    pierce: Math.min(0.9, 0.08 * t('pierce')),
    dashDmg: dmg * 0.8 * t('dashstrike'),
    grenadeDmg: dmg * 6 * (1 + 0.2 * t('grenade')),
    grenadeR: 6 * (1 + 0.05 * t('grenade')),
    orbitalDmg: dmg * 30 * (1 + 0.25 * t('orbital')),
    towerAura: 0.05 * t('aura') + g.towerPct / 100,
    drone: t('drone'),
    droneDmg: dmg * 0.25 * (1 + 0.3 * t('drone')),
    repair: 0.0015 * t('repair'),
    thrift: Math.min(0.3, 0.02 * t('thrift')),
    xpMul: (1 + g.xpPct / 100) * (1 + 0.08 * (research.r_xp || 0)),
  };
}
export const ABILITIES = {
  q: { name: 'Ривок', cd: 5, desc: 'Стрімкий ривок у напрямку руху.' },
  e: { name: 'Плазмова граната', cd: 8, desc: 'Вибух по області в точці прицілу.' },
  r: { name: 'Орбітальний удар', cd: 45, desc: 'Через 1.2 с з орбіти б’є потужний промінь.' },
};
export const HERO_COLORS = [0x33e0ff, 0xff4fd8, 0x9dff4a, 0xffb22e, 0xb98cff, 0xff6a4a];

export function fmt(n) {
  if (n < 1000) return Math.floor(n).toString();
  const u = ['K', 'M', 'B', 'T', 'Qa', 'Qi'];
  let i = -1;
  while (n >= 1000 && i < u.length - 1) { n /= 1000; i++; }
  return n.toFixed(n < 10 ? 2 : n < 100 ? 1 : 0) + u[i];
}
