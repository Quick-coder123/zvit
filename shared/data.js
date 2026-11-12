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
// Зруйноване місто займає зовнішню частину першого сектора
export const CITY = { angle: SECTORS[0], spread: 0.62, rMin: 48, rMax: 118 };
export function inCity(x, z) {
  const r = Math.hypot(x, z);
  if (r < CITY.rMin || r > CITY.rMax) return 0;
  let d = Math.atan2(z, x) - CITY.angle;
  while (d > Math.PI) d -= Math.PI * 2;
  while (d < -Math.PI) d += Math.PI * 2;
  const k = 1 - Math.abs(d) / CITY.spread;
  return Math.max(0, Math.min(1, k * 3)) * Math.min(1, (r - CITY.rMin) / 8);
}

export const DEPOSITS = [];
SECTORS.forEach((a, s) => {
  DEPOSITS.push({ id: DEPOSITS.length, x: Math.cos(a) * 54, z: Math.sin(a) * 54, sector: s });
  DEPOSITS.push({ id: DEPOSITS.length, x: Math.cos(a + 0.28) * 92, z: Math.sin(a + 0.28) * 92, sector: s });
});

// Позиції ДРГ противника: tier → стартовий рівень
export const CAMP_TIERS = [
  { lvl: 1, name: 'Передові окопи' },
  { lvl: 5, name: 'Лісосмуга' },
  { lvl: 10, name: 'Промзона' },
  { lvl: 16, name: 'Укріпрайон' },
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
  infantry: { name: 'Піхотинець', hp: 60, speed: 3.8, armor: 0, core: 15, bounty: 6, xp: 6, size: 1.0, color: 0x9a8f6a, human: true },
  moto: { name: 'Мотоцикліст', hp: 38, speed: 8.4, armor: 0, core: 10, bounty: 5, xp: 5, size: 1.0, color: 0x6f6a5a, vehicle: true },
  fpv: { name: 'FPV-дрон', hp: 14, speed: 9.5, armor: 0, core: 6, bounty: 2, xp: 2, size: 0.5, color: 0x3a3a3a, flying: true, low: true },
  btr: { name: 'БТР', hp: 300, speed: 4.2, armor: 6, core: 40, bounty: 18, xp: 18, size: 1.6, color: 0x5d6447, vehicle: true, melee: 18, reach: 10 },
  shahed: { name: 'Ударний дрон', hp: 55, speed: 11, armor: 0, core: 30, bounty: 10, xp: 8, size: 1.2, color: 0xb9b9b4, flying: true },
  tank: { name: 'Танк', hp: 260, shield: 220, speed: 2.9, armor: 12, core: 60, bounty: 28, xp: 22, size: 1.9, color: 0x4f5640, vehicle: true, melee: 30, reach: 12 },
  repair: { name: 'Ремонтна машина', hp: 110, speed: 3.8, armor: 3, core: 15, bounty: 15, xp: 14, size: 1.5, color: 0x666b55, vehicle: true, heal: 0.06 },
  boss_heavy: { name: 'Важкий танк', hp: 5200, shield: 1200, speed: 2.1, armor: 16, core: 400, bounty: 400, xp: 400, size: 2.8, color: 0x3e4433, boss: true, vehicle: true, melee: 70, reach: 14 },
  boss_transport: { name: 'Десантний транспорт', hp: 4300, speed: 2.6, armor: 8, core: 400, bounty: 400, xp: 400, size: 2.6, color: 0x5a5f48, boss: true, vehicle: true, melee: 30, reach: 10 },
  boss_heli: { name: 'Бойовий гелікоптер', hp: 4700, speed: 3, armor: 8, core: 400, bounty: 450, xp: 450, size: 2.6, color: 0x4b5240, boss: true, flying: true, melee: 50 },
  w_rifle: { name: 'Автоматник ДРГ', hp: 80, speed: 5, armor: 0, bounty: 8, xp: 14, size: 1.0, color: 0x6e6a52, wild: true, human: true, ranged: 8, range: 14 },
  w_mg: { name: 'Кулеметник ДРГ', hp: 70, speed: 4.2, armor: 0, bounty: 8, xp: 14, size: 1.0, color: 0x5e5a46, wild: true, human: true, ranged: 6, range: 18, burst: true },
  w_officer: { name: 'Командир ДРГ', hp: 320, speed: 4.6, armor: 4, bounty: 30, xp: 50, size: 1.1, color: 0x4d4a3a, wild: true, human: true, ranged: 20, range: 15 },
};export const ENEMY_KINDS = Object.keys(ENEMIES);
export const BOSSES = ['boss_heavy', 'boss_transport', 'boss_heli'];

export function waveComposition(w) {
  const total = Math.min(110, Math.floor(10 + w * 1.7));
  const pool = [['infantry', 10]];
  if (w >= 2) pool.push(['moto', 6]);
  if (w >= 4) pool.push(['fpv', 4]);
  if (w >= 5) pool.push(['btr', 3 + w * 0.05]);
  if (w >= 7) pool.push(['shahed', 4]);
  if (w >= 9) pool.push(['tank', 3 + w * 0.04]);
  if (w >= 12) pool.push(['repair', 2]);
  const sum = pool.reduce((s, p) => s + p[1], 0);
  const list = [];
  for (const [kind, wt] of pool) {
    let n = Math.round((total * wt) / sum);
    if (kind === 'fpv') n = Math.max(1, Math.round(n / 5)) * 5;
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
  mg: { name: 'Кулеметне гніздо', key: '1', cost: 100, dmg: 11, rate: 6, range: 20, color: 0xffb648, desc: 'Швидкий вогонь по піхоті та легких цілях.', unlock: null },
  ags: { name: 'АГС', key: '2', cost: 150, dmg: 15, rate: 2, range: 19, splash: 3.5, slow: 0.35, slowT: 1.5, ground: true, color: 0xe0c070, desc: 'Автоматичний гранатомет: осколки по площі, придушення сповільнює ворога.', unlock: 'u_ags' },
  aa: { name: 'Зенітна установка', key: '3', cost: 200, dmg: 22, rate: 3, range: 26, multi: 2, airMul: 2.5, color: 0xffd27a, desc: 'Спарені гармати, кожна веде свою ціль. ×2.5 шкоди по повітряних цілях.', unlock: 'u_aa' },
  atgm: { name: 'ПТРК', key: '4', cost: 260, dmg: 150, rate: 0.4, range: 34, ap: true, heavyMul: 1.6, color: 0xff8a3a, desc: 'Протитанкова ракета: ігнорує броню, ×1.6 шкоди по техніці.', unlock: 'u_atgm' },
  mortar: { name: 'Міномет', key: '5', cost: 320, dmg: 70, rate: 0.5, range: 36, minRange: 8, splash: 6, ground: true, color: 0xff7733, desc: 'Навісний вогонь по площі. Не б’є по повітряних цілях.', unlock: 'u_mortar' },
  howitzer: { name: 'Гаубиця', key: '6', cost: 450, dmg: 260, rate: 0.22, range: 60, minRange: 14, splash: 8, ground: true, color: 0xff5a2a, desc: 'Далекобійна артилерія: величезна шкода по площі.', unlock: 'u_howitzer' },
};export const TOWER_TYPES = Object.keys(TOWERS);
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
    multi: t.multi ? t.multi + Math.floor((lvl - 1) / 8) : 0,
  };
}

// ───────────────────────── Майстерні (запчастини) ─────────────────────────
export const EXTRACTOR_COST = 300;
export function extractorUpgradeCost(lvl) { return Math.round(300 * Math.pow(1.65, lvl)); }
export function extractorRate(lvl, research = {}) {
  if (lvl <= 0) return 0;
  return 0.35 * Math.pow(1.35, lvl - 1) * (1 + 0.15 * (research.r_mine || 0));
}

// ───────────────────────── Модернізація (запчастини) ─────────────────────────
export const RESEARCH = {
  u_ags: { name: 'АГС', desc: 'Відкриває «АГС» (автоматичний гранатомет).', base: 40, max: 1, branch: 'Озброєння' },
  u_aa: { name: 'ППО', desc: 'Відкриває «Зенітну установку».', base: 80, max: 1, branch: 'Озброєння', req: 'u_ags' },
  u_atgm: { name: 'Протитанкові засоби', desc: 'Відкриває «ПТРК».', base: 150, max: 1, branch: 'Озброєння', req: 'u_aa' },
  u_mortar: { name: 'Мінометна батарея', desc: 'Відкриває «Міномет».', base: 220, max: 1, branch: 'Озброєння', req: 'u_atgm' },
  u_howitzer: { name: 'Артилерія', desc: 'Відкриває «Гаубицю».', base: 400, max: 1, branch: 'Озброєння', req: 'u_mortar' },
  r_dmg: { name: 'Боєприпаси', desc: '+10% шкоди всіх позицій за рівень.', base: 30, max: 40, branch: 'Озброєння' },
  r_rate: { name: 'Механізми подачі', desc: '+5% швидкості стрільби позицій.', base: 45, max: 20, branch: 'Озброєння' },
  r_range: { name: 'Оптика й розвідка', desc: '+4% дальності позицій.', base: 50, max: 12, branch: 'Озброєння' },
  r_core: { name: 'Фортифікація штабу', desc: '+15% міцності штабу.', base: 35, max: 30, branch: 'База' },
  r_regen: { name: 'Інженерна служба', desc: 'Штаб відновлює +0.4% міцності/с.', base: 60, max: 10, branch: 'База' },
  r_mine: { name: 'Майстерні', desc: '+15% виробництва запчастин.', base: 40, max: 30, branch: 'Логістика' },
  r_bounty: { name: 'Трофейна команда', desc: '+8% коштів за знищення ворога.', base: 40, max: 30, branch: 'Логістика' },
  r_income: { name: 'Постачання', desc: '+1 ₴/с пасивного фінансування.', base: 50, max: 20, branch: 'Логістика' },
  r_team: { name: 'Бойова підготовка', desc: 'Бійці: +5% шкоди та здоров’я.', base: 50, max: 30, branch: 'Особовий склад' },
  r_luck: { name: 'Трофейні пошуки', desc: '+10% шанс трофеїв і кращої якості.', base: 70, max: 15, branch: 'Особовий склад' },
  r_xp: { name: 'Досвід ветеранів', desc: '+8% досвіду бійцям.', base: 60, max: 20, branch: 'Особовий склад' },
};export function researchCost(id, lvl) {
  return Math.round(RESEARCH[id].base * Math.pow(1.55, lvl));
}

// ───────────────────────── Герої ─────────────────────────
export function xpToNext(lvl) { return Math.round(90 * Math.pow(1.22, lvl - 1) + 40 * lvl); }

export const BRANCHES = {
  vanguard: { name: 'Штурмовик', color: '#d9694a', desc: 'Живучість і штурм' },
  gunner: { name: 'Стрілець', color: '#d9b84a', desc: 'Шкода, влучність' },
  engineer: { name: 'Інженер', color: '#7fb3c9', desc: 'Позиції, дрон, штаб' },
};
export const TALENTS = {
  plates: { branch: 'vanguard', name: 'Бронеплити', desc: '+10% здоров’я, +2 броні', max: 10 },
  regen: { branch: 'vanguard', name: 'Тактична медицина', desc: '+0.6% здоров’я/с', max: 10 },
  dashstrike: { branch: 'vanguard', name: 'Штурмовий ривок', desc: 'Ривок (Q) завдає шкоди: +80% атаки за рівень', max: 10 },
  leech: { branch: 'vanguard', name: 'Адреналін', desc: '+2% шкоди повертається здоров’ям', max: 10 },
  bulwark: { branch: 'vanguard', name: 'Незламність', desc: '−4% вхідної шкоди', max: 10 },
  caliber: { branch: 'gunner', name: 'Посилений набій', desc: '+10% шкоди', max: 10 },
  trigger: { branch: 'gunner', name: 'Темп стрільби', desc: '+6% швидкості стрільби', max: 10 },
  crit: { branch: 'gunner', name: 'Влучність', desc: '+4% шансу критичного влучання, +10% крит. шкоди', max: 10 },
  pierce: { branch: 'gunner', name: 'Бронебійні кулі', desc: '+8% шансу прострілити ціль наскрізь', max: 10 },
  grenade: { branch: 'gunner', name: 'Підривна справа', desc: 'Граната (E): +20% шкоди, +5% радіусу', max: 10 },
  aura: { branch: 'engineer', name: 'Коригування вогню', desc: 'Позиції в радіусі 22 м: +5% шкоди', max: 10 },
  drone: { branch: 'engineer', name: 'Дрон-помічник', desc: 'Дрон скидає боєприпаси на ворогів поруч', max: 10 },
  repair: { branch: 'engineer', name: 'Ремонт штабу', desc: 'Біля штабу відновлює його на 0.15%/с', max: 10 },
  thrift: { branch: 'engineer', name: 'Логістика', desc: '−2% вартості позицій і покращень', max: 10 },
  orbital: { branch: 'engineer', name: 'Наведення артилерії', desc: 'Артудар (R): +25% шкоди', max: 10 },
};
export const RARITIES = [
  { id: 'common', name: 'Звичайний', color: '#c8d0d8', affixes: 1, mul: 1 },
  { id: 'uncommon', name: 'Незвичайний', color: '#4dff7a', affixes: 2, mul: 1.15 },
  { id: 'rare', name: 'Рідкісний', color: '#3aa6ff', affixes: 3, mul: 1.35 },
  { id: 'epic', name: 'Епічний', color: '#c057ff', affixes: 4, mul: 1.6 },
  { id: 'legendary', name: 'Легендарний', color: '#ffa02a', affixes: 5, mul: 2.0 },
];
export const SLOTS = { weapon: 'Зброя', armor: 'Захист', module: 'Спорядження' };
const ITEM_NAMES = {
  weapon: ['Пістолет-кулемет', 'Карабін', 'Штурмова гвинтівка', 'Ручний кулемет', 'Марксманська гвинтівка', 'Важка снайперська гвинтівка'],
  armor: ['Бронежилет', 'Плитоносець', 'Бронежилет з балістичним шоломом', 'Штурмовий бронекомплект', 'Плитоносець з керамікою', 'Важкий саперний комплект'],
  module: ['Тактична розвантажка', 'Коліматорний приціл', 'Рація', 'Аптечка IFAK', 'Прилад нічного бачення', 'Тепловізійний приціл'],
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
  cdr: { name: 'Перезарядка навичок', fmt: (v) => `−${v.toFixed(0)}%`, roll: (l) => 2 + Math.min(18, l * 0.2) },
  towerPct: { name: 'Коригування позицій', fmt: (v) => `+${v.toFixed(0)}%`, roll: (l) => 2 + l * 0.3 },
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
  q: { name: 'Ривок', cd: 5, desc: 'Стрімкий кидок у напрямку руху.' },
  e: { name: 'Граната', cd: 8, desc: 'Осколкова граната в точку прицілу.' },
  r: { name: 'Артудар', cd: 45, desc: 'Через 1.2 с по точці б’є залп артилерії.' },
};
export const HERO_COLORS = [0x4aa3ff, 0xffc83a, 0x5ad16a, 0xff7a4a, 0xc58cff, 0x4ad6c8];

export function fmt(n) {
  if (n < 1000) return Math.floor(n).toString();
  const u = ['K', 'M', 'B', 'T', 'Qa', 'Qi'];
  let i = -1;
  while (n >= 1000 && i < u.length - 1) { n /= 1000; i++; }
  return n.toFixed(n < 10 ? 2 : n < 100 ? 1 : 0) + u[i];
}

// ───────────────────────── Пори року й погода ─────────────────────────
export const SEASONS = [
  { id: 'spring', name: 'Весна' },
  { id: 'summer', name: 'Літо' },
  { id: 'autumn', name: 'Осінь' },
  { id: 'winter', name: 'Зима' },
];
export const WAVES_PER_SEASON = 5;
export function seasonOf(wave) { return Math.floor((wave - 1) / WAVES_PER_SEASON) % 4; }
export const WEATHER = {
  clear: { name: 'Ясно', range: 1, ground: 1, air: 1 },
  cloudy: { name: 'Хмарно', range: 1, ground: 1, air: 1 },
  rain: { name: 'Дощ', range: 0.95, ground: 0.95, air: 0.95, desc: '−5% дальності, бездоріжжя' },
  storm: { name: 'Гроза', range: 0.9, ground: 0.95, air: 0.8, desc: '−10% дальності, дрони −20% швидкості' },
  fog: { name: 'Туман', range: 0.85, ground: 1, air: 1, desc: '−15% дальності позицій' },
  snow: { name: 'Сніг', range: 0.95, ground: 0.9, air: 0.95, desc: 'Наземні вороги −10% швидкості' },
  blizzard: { name: 'Хуртовина', range: 0.85, ground: 0.85, air: 0.7, desc: '−15% дальності, вороги повільніші' },
};
export const SEASON_WEATHER = [
  { clear: 3, cloudy: 3, rain: 3, fog: 2, storm: 1 },
  { clear: 6, cloudy: 2, storm: 2, rain: 1 },
  { cloudy: 3, rain: 3, fog: 3, clear: 1, storm: 1 },
  { snow: 4, cloudy: 2, clear: 2, blizzard: 2, fog: 1 },
];
