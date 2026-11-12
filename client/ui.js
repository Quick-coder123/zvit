// HUD та панелі інтерфейсу.
import {
  TOWERS, TOWER_TYPES, TOWER_MAX_LVL, towerStats, towerUpgradeCost, RESEARCH, researchCost, TALENTS, BRANCHES,
  RARITIES, SLOTS, AFFIXES, itemValue, itemSalvage, itemPower, xpToNext, ABILITIES, fmt, DIFFICULTY, heroStats,
  EXTRACTOR_COST, extractorUpgradeCost, extractorRate, ENEMIES,
} from '/shared/data.js';
import { QUALITY } from './scene.js';

const $ = (id) => document.getElementById(id);
const SLOT_ICON = { weapon: '⌖', armor: '⛨', module: '◎' };
const hex = (n) => '#' + n.toString(16).padStart(6, '0');
const esc = (s) => String(s).replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]));

export class UI {
  constructor(G) {
    this.G = G;
    this.tab = null;
    this.selTower = null;
    this.bannerT = 0;
    this.buildBtns = {};
    this.initBuildBar();
    this.initAbilities();
    $('tabs').addEventListener('click', (e) => {
      const b = e.target.closest('button');
      if (!b) return;
      if (b.id === 'panel-close') return this.closePanel();
      this.openPanel(b.dataset.tab);
    });
    $('ready-btn').addEventListener('click', () => G.send({ t: 'ready' }));
    this.tip = $('tip');
    document.addEventListener('mousemove', (e) => {
      if (!this.tip.classList.contains('hidden')) {
        const x = Math.min(window.innerWidth - 320, e.clientX + 16), y = Math.min(window.innerHeight - this.tip.offsetHeight - 10, e.clientY + 16);
        this.tip.style.left = x + 'px'; this.tip.style.top = y + 'px';
      }
    });
  }

  // ───── Будівництво ─────
  initBuildBar() {
    const bar = $('build-bar');
    for (const type of TOWER_TYPES) {
      const d = TOWERS[type];
      const b = document.createElement('button');
      b.className = 'tb';
      b.innerHTML = `<span class="k">${d.key}</span><div class="sw" style="background:${hex(d.color)};color:${hex(d.color)}"></div><div>${d.name}</div><div class="cost">◈ ${d.cost}</div>`;
      b.addEventListener('click', () => this.G.selectBuild(type));
      b.addEventListener('mouseenter', () => this.showTip(`<div class="tn" style="color:${hex(d.color)}">${d.name}</div><div class="ts">${d.desc}</div>
        <div class="th">Шкода ${d.dmg}${d.rate ? ` · ${d.rate}/с` : '/с (наростає)'} · Дальність ${d.range}</div>${d.unlock && !this.G.world.research[d.unlock] ? `<div class="th" style="color:#f66">Потрібне дослідження: ${RESEARCH[d.unlock].name}</div>` : ''}`, hex(d.color)));
      b.addEventListener('mouseleave', () => this.hideTip());
      bar.appendChild(b);
      this.buildBtns[type] = b;
    }
  }
  updateBuildBar() {
    const G = this.G;
    const thrift = G.me ? G.me.stats.thrift : 0;
    for (const type of TOWER_TYPES) {
      const d = TOWERS[type], b = this.buildBtns[type];
      const cost = Math.round(d.cost * (1 - thrift));
      b.querySelector('.cost').textContent = '◈ ' + cost;
      b.classList.toggle('locked', !!(d.unlock && !G.world.research[d.unlock]));
      b.classList.toggle('poor', G.snap && G.snap.credits < cost);
      b.classList.toggle('sel', G.buildType === type);
    }
  }

  initAbilities() {
    const ic = { q: '➤', e: '✹', r: '☄' };
    $('abilities').innerHTML = ['q', 'e', 'r'].map((k) => `<div class="ab" id="ab-${k}"><span class="k">${k.toUpperCase()}</span><div class="ic">${ic[k]}</div><div>${ABILITIES[k].name}</div><div class="cd"></div><div class="cdt"></div></div>`).join('');
    for (const k of ['q', 'e', 'r']) {
      const el = $('ab-' + k);
      el.addEventListener('mouseenter', () => this.showTip(`<div class="tn">${ABILITIES[k].name} [${k.toUpperCase()}]</div><div class="ts">${ABILITIES[k].desc}</div><div class="th">${this.abilityInfo(k)}</div>`, '#3ae0ff'));
      el.addEventListener('mouseleave', () => this.hideTip());
    }
  }
  abilityInfo(k) {
    const s = this.G.me?.stats;
    if (!s) return '';
    const cd = (ABILITIES[k].cd * (1 - s.cdr)).toFixed(1);
    if (k === 'q') return `Перезарядка ${cd} с${s.dashDmg > 0 ? ` · Шкода ${fmt(s.dashDmg)}` : ''}`;
    if (k === 'e') return `Перезарядка ${cd} с · Шкода ${fmt(s.grenadeDmg)} · Радіус ${s.grenadeR.toFixed(1)} м`;
    return `Перезарядка ${cd} с · Шкода ${fmt(s.orbitalDmg)} · Радіус 9 м`;
  }

  // ───── HUD ─────
  updateHUD() {
    const G = this.G, s = G.snap;
    if (!s) return;
    $('credits').textContent = fmt(s.credits);
    $('crystals').textContent = fmt(s.crystals);
    $('wave-num').textContent = `ХВИЛЯ ${s.wave}${s.wave % 10 === 0 ? ' · БОС' : ''}`;
    const ph = $('wave-phase');
    if (s.phase === 'prep') {
      const m = Math.floor(s.timer / 60), sec = String(s.timer % 60).padStart(2, '0');
      ph.textContent = `Підготовка · напад через ${m}:${sec}`;
      ph.classList.remove('attack');
      $('ready-btn').classList.remove('hidden');
    } else {
      ph.textContent = `АТАКА · ворогів лишилось: ${s.left}`;
      ph.classList.add('attack');
      $('ready-btn').classList.add('hidden');
    }
    const meP = s.players.find((p) => p.n === G.name);
    $('ready-btn').classList.toggle('on', !!(meP && meP.rd));
    $('ready-btn').innerHTML = (meP && meP.rd ? '✓ Готовий — чекаємо інших' : 'Готові до бою') + ' <kbd>G</kbd>';
    const cf = s.core[0] / s.core[1];
    $('core-fill').style.width = Math.max(0, cf * 100) + '%';
    $('core-fill').style.background = cf > 0.5 ? 'linear-gradient(90deg,#1aa6ff,#3ae0ff)' : cf > 0.25 ? 'linear-gradient(90deg,#d08a00,#ffd23a)' : 'linear-gradient(90deg,#a00,#ff4a5a)';
    $('core-text').textContent = `ЯДРО ${fmt(s.core[0])} / ${fmt(s.core[1])}`;
    // команда
    $('team').innerHTML = s.players.map((p) => `<div class="tm ${p.d ? 'dead' : ''}" style="border-left-color:${hex(p.c)}">
      <div class="n"><span>${esc(p.n)}${p.n === G.name ? ' (ви)' : ''}</span><span>рів. ${p.l}${p.rd && s.phase === 'prep' ? ' <span class="rd">✓</span>' : ''}</span></div>
      <div class="b"><div style="width:${(p.hp / p.mhp) * 100}%;background:${p.d ? '#555' : hex(p.c)}"></div></div>${p.d ? `<small>відродження ${p.rs} с</small>` : ''}</div>`).join('');
    // герой
    if (meP && G.me) {
      const pr = G.me.profile;
      $('hb-name').textContent = pr.name;
      $('hb-lvl').textContent = `Рівень ${pr.level}`;
      $('hb-pts').textContent = pr.points > 0 ? `+${pr.points} очок талантів [K]` : '';
      $('hp-fill').style.width = (meP.hp / meP.mhp) * 100 + '%';
      $('hp-text').textContent = `${fmt(meP.hp)} / ${fmt(meP.mhp)}`;
      $('xp-fill').style.width = Math.min(100, (pr.xp / xpToNext(pr.level)) * 100) + '%';
      for (const k of ['q', 'e', 'r']) {
        const cd = G.localCds[k];
        const max = ABILITIES[k].cd * (1 - G.me.stats.cdr);
        const el = $('ab-' + k);
        el.querySelector('.cd').style.height = cd > 0 ? (cd / max) * 100 + '%' : '0';
        el.querySelector('.cdt').textContent = cd > 0 ? Math.ceil(cd) : '';
      }
      $('dead').classList.toggle('hidden', !meP.d);
      $('dead-t').textContent = `Відродження через ${meP.rs} с`;
    }
    this.updateBuildBar();
    if (this.selTower) this.renderTowerPanel();
  }

  feed(text, color = '#cfe') {
    const f = $('feed');
    const d = document.createElement('div');
    d.style.color = color;
    d.textContent = text;
    f.appendChild(d);
    while (f.children.length > 9) f.removeChild(f.firstChild);
    setTimeout(() => (d.style.opacity = '0'), 9000);
    setTimeout(() => d.remove(), 10000);
  }
  banner(text, sub = '', color = '#3ae0ff', dur = 3) {
    const b = $('banner');
    b.innerHTML = `${esc(text)}${sub ? `<small>${esc(sub)}</small>` : ''}`;
    b.style.color = color;
    b.style.opacity = 1;
    clearTimeout(this.bannerTO);
    this.bannerTO = setTimeout(() => (b.style.opacity = 0), dur * 1000);
  }
  prompt(html) {
    const p = $('prompt');
    if (!html) { p.classList.add('hidden'); return; }
    p.innerHTML = html;
    p.classList.remove('hidden');
  }
  showTip(html, color = '#3ae0ff') {
    this.tip.innerHTML = html;
    this.tip.style.borderColor = color;
    this.tip.classList.remove('hidden');
  }
  hideTip() { this.tip.classList.add('hidden'); }

  // ───── Панель вежі ─────
  selectTower(id) {
    this.selTower = id;
    $('tower-panel').classList.toggle('hidden', !id);
    if (id) this.renderTowerPanel();
  }
  renderTowerPanel() {
    const G = this.G;
    const t = G.world.towers.find((x) => x.id === this.selTower);
    if (!t) { this.selectTower(null); return; }
    const d = TOWERS[t.type];
    const st = towerStats(t.type, t.level, G.world.research);
    const max = t.level >= TOWER_MAX_LVL;
    const nx = max ? null : towerStats(t.type, t.level + 1, G.world.research);
    const thrift = G.me ? G.me.stats.thrift : 0;
    const cost = Math.round(towerUpgradeCost(t.type, t.level) * (1 - thrift));
    const row = (n, a, b, f = (v) => fmt(v)) => `<div class="st"><span>${n}</span><span><b>${f(a)}</b>${b != null && f(b) !== f(a) ? ` <i>→ ${f(b)}</i>` : ''}</span></div>`;
    const key = `${t.id}|${t.level}|${cost}|${G.snap?.credits >= cost}`;
    if (this._tpKey === key) return;
    this._tpKey = key;
    $('tower-panel').innerHTML = `<h3 style="color:${hex(d.color)}">${d.name}</h3><div style="color:var(--dim);margin-bottom:8px">Рівень ${t.level} / ${TOWER_MAX_LVL}</div>
      ${row('Шкода', st.dmg, nx?.dmg)}
      ${d.rate ? row('Пострілів/с', st.rate, nx?.rate, (v) => v.toFixed(2)) : row('Наростання', d.ramp, null, (v) => `до ×${(1 + v).toFixed(1)}`)}
      ${row('Дальність', st.range, nx?.range, (v) => v.toFixed(1) + ' м')}
      ${d.splash ? row('Радіус ураження', st.splash, nx?.splash, (v) => v.toFixed(1) + ' м') : ''}
      ${d.slow ? row('Сповільнення', st.slow, nx?.slow, (v) => Math.round(v * 100) + '%') : ''}
      ${d.chain ? row('Перескоків', st.chain, nx?.chain, (v) => v) : ''}
      <div class="st"><span>Вкладено</span><span><b>◈ ${fmt(t.spent)}</b></span></div>
      <div class="row2">
        <button id="tp-up" ${max || G.snap?.credits < cost ? 'disabled' : ''}>${max ? 'Максимум' : `Покращити ◈ ${fmt(cost)}`} <kbd>U</kbd></button>
      </div>
      <div class="row2"><button id="tp-sell">Продати ◈ ${fmt(Math.round(t.spent * 0.7))}</button><button id="tp-close">Закрити</button></div>`;
    $('tp-up').onclick = () => { G.send({ t: 'upgrade', id: t.id }); };
    $('tp-sell').onclick = () => { G.send({ t: 'sell', id: t.id }); this.selectTower(null); G.audio.play('sell'); };
    $('tp-close').onclick = () => this.selectTower(null);
  }

  // ───── Великі панелі ─────
  openPanel(tab) {
    if (this.tab === tab) return this.closePanel();
    this.tab = tab;
    $('panel').classList.remove('hidden');
    document.querySelectorAll('#tabs button[data-tab]').forEach((b) => b.classList.toggle('on', b.dataset.tab === tab));
    this._pKey = null;
    this.renderPanel(true);
  }
  closePanel() { this.tab = null; $('panel').classList.add('hidden'); this.hideTip(); }
  isOpen() { return !!this.tab; }
  renderPanel(force) {
    if (!this.tab) return;
    const G = this.G;
    // перемальовуємо лише коли змінились дані
    const key = this.tab + '|' + JSON.stringify(G.me?.profile) + '|' + JSON.stringify(G.world.research) + '|' + (this.tab === 'research' ? G.snap?.crystals : '') + '|' + (this.tab === 'stats' ? Math.floor(Date.now() / 2000) : '');
    if (!force && key === this._pKey) return;
    this._pKey = key;
    const body = $('panel-body');
    const scroll = body.scrollTop;
    if (this.tab === 'hero') this.renderHero(body);
    else if (this.tab === 'talents') this.renderTalents(body);
    else if (this.tab === 'research') this.renderResearch(body);
    else if (this.tab === 'stats') this.renderStats(body);
    else if (this.tab === 'settings') this.renderSettings(body);
    body.scrollTop = scroll;
  }

  itemTip(it, compare) {
    const r = RARITIES[it.rarity];
    const lines = [];
    if (it.stats.atk) lines.push(`Атака +${it.stats.atk}`);
    if (it.stats.hp) lines.push(`Здоров’я +${it.stats.hp}`);
    if (it.stats.cdr && it.slot === 'module') lines.push(`Перезарядка вмінь −${it.stats.cdr}%`);
    for (const [k, v] of Object.entries(it.stats)) {
      if (k === 'atk' || k === 'hp' || (k === 'cdr' && it.slot === 'module')) continue;
      if (AFFIXES[k]) lines.push(`${AFFIXES[k].name} ${AFFIXES[k].fmt(v)}`);
    }
    let cmp = '';
    if (compare) {
      const d = itemPower(it) - itemPower(compare);
      cmp = `<div class="th">Порівняно з одягненим: <span style="color:${d >= 0 ? '#4dff7a' : '#ff4a5a'}">${d >= 0 ? '+' : ''}${d} сили</span></div>`;
    }
    return `<div class="tn" style="color:${r.color}">${esc(it.name)}</div><div class="tr" style="color:${r.color}">${r.name} · ${SLOTS[it.slot]} · рів. предмета ${it.ilvl}</div>
      <div class="ts">${lines.join('<br>')}</div><div class="th">Сила: ${itemPower(it)}</div>${cmp}`;
  }

  renderHero(body) {
    const G = this.G, p = G.me.profile, s = G.me.stats;
    const eq = (slot) => {
      const it = p.equip[slot];
      if (!it) return `<div class="slot" data-slot="${slot}"><div class="sl">${SLOTS[slot]}</div><div style="color:var(--dim);margin-top:20px;text-align:center">порожньо</div></div>`;
      const r = RARITIES[it.rarity];
      return `<div class="slot" data-slot="${slot}" data-eq="${slot}" style="border:2px solid ${r.color}"><div class="sl">${SLOTS[slot]}</div><div style="color:${r.color};font-weight:600">${SLOT_ICON[slot]} ${esc(it.name)}</div><div style="color:var(--dim);font-size:11px">рів. ${it.ilvl} · сила ${itemPower(it)}</div><div style="font-size:11px;margin-top:4px;color:var(--dim)">ПКМ — зняти</div></div>`;
    };
    const cells = [];
    for (let i = 0; i < 40; i++) {
      const it = p.inv[i];
      if (!it) { cells.push('<div class="it empty"></div>'); continue; }
      const r = RARITIES[it.rarity];
      const cur = p.equip[it.slot];
      const better = !cur || itemPower(it) > itemPower(cur);
      cells.push(`<div class="it ${better ? 'better' : ''}" data-i="${i}" style="border-color:${r.color};color:${r.color};box-shadow:${it.rarity >= 3 ? `0 0 10px ${r.color}` : 'none'}">${SLOT_ICON[it.slot]}<small>${it.ilvl}</small></div>`);
    }
    const stat = (n, v) => `<div><span>${n}</span><b>${v}</b></div>`;
    body.innerHTML = `<div class="cols">
      <div class="col" style="flex:0 0 380px">
        <h2>${esc(p.name)} · рівень ${p.level}</h2>
        <div style="color:var(--dim);font-size:13px;margin-bottom:12px">Досвід ${fmt(p.xp)} / ${fmt(xpToNext(p.level))} · Вбивств ${fmt(p.kills)} · Смертей ${p.deaths} · Босів ${p.bossKills}</div>
        <div class="stats-grid">
          ${stat('Здоров’я', fmt(s.maxHp))}${stat('Атака', fmt(s.dmg))}
          ${stat('Пострілів/с', s.rate.toFixed(2))}${stat('Шкода/с', fmt(s.dmg * s.rate * (1 + s.crit * (s.critMul - 1))))}
          ${stat('Шанс криту', (s.crit * 100).toFixed(1) + '%')}${stat('Крит. шкода', '×' + s.critMul.toFixed(1))}
          ${stat('Броня', s.armor.toFixed(0))}${stat('Регенерація', s.regen.toFixed(1) + '/с')}
          ${stat('Вампіризм', (s.leech * 100).toFixed(1) + '%')}${stat('Зниження шкоди', (s.dr * 100).toFixed(0) + '%')}
          ${stat('Швидкість', s.speed.toFixed(1))}${stat('Перезарядка', '−' + (s.cdr * 100).toFixed(0) + '%')}
          ${stat('Пробиття', (s.pierce * 100).toFixed(0) + '%')}${stat('Аура веж', '+' + (s.towerAura * 100).toFixed(0) + '%')}
          ${stat('Бонус досвіду', '+' + ((s.xpMul - 1) * 100).toFixed(0) + '%')}${stat('Знижка на вежі', (s.thrift * 100).toFixed(0) + '%')}
        </div>
      </div>
      <div class="col">
        <h2>Спорядження</h2>
        <div class="equip">${eq('weapon')}${eq('armor')}${eq('module')}</div>
        <h2>Інвентар <span style="font-size:13px;color:var(--dim)">${p.inv.length}/40 · ЛКМ — одягти · ПКМ — продати · Shift+ПКМ — розібрати на кристали</span></h2>
        <div class="inv" id="inv">${cells.join('')}</div>
        <div class="bulk">Масово:
          <button data-bulk="0" data-m="sell">Продати звичайні</button><button data-bulk="1" data-m="sell">Продати ≤ незвичайні</button>
          <button data-bulk="0" data-m="salvage">Розібрати звичайні</button><button data-bulk="1" data-m="salvage">Розібрати ≤ незвичайні</button><button data-bulk="2" data-m="salvage">Розібрати ≤ рідкісні</button>
        </div>
      </div></div>`;
    body.querySelectorAll('.it[data-i]').forEach((el) => {
      const it = p.inv[+el.dataset.i];
      el.addEventListener('mouseenter', () => this.showTip(this.itemTip(it, p.equip[it.slot]) + `<div class="th">Продаж ◈ ${itemValue(it)} · Розбір ◆ ${itemSalvage(it)}</div>`, RARITIES[it.rarity].color));
      el.addEventListener('mouseleave', () => this.hideTip());
      el.addEventListener('click', () => { G.send({ t: 'equip', id: it.id }); this.hideTip(); G.audio.play('ui'); });
      el.addEventListener('contextmenu', (e) => { e.preventDefault(); G.send({ t: e.shiftKey ? 'salvage' : 'sellItem', id: it.id }); this.hideTip(); G.audio.play(e.shiftKey ? 'ui' : 'sell'); });
    });
    body.querySelectorAll('.slot[data-eq]').forEach((el) => {
      const it = p.equip[el.dataset.eq];
      el.addEventListener('mouseenter', () => this.showTip(this.itemTip(it), RARITIES[it.rarity].color));
      el.addEventListener('mouseleave', () => this.hideTip());
      el.addEventListener('contextmenu', (e) => { e.preventDefault(); G.send({ t: 'unequip', slot: el.dataset.eq }); this.hideTip(); });
    });
    body.querySelectorAll('[data-bulk]').forEach((b) => b.addEventListener('click', () => G.send({ t: 'bulk', max: +b.dataset.bulk, mode: b.dataset.m })));
  }

  renderTalents(body) {
    const G = this.G, p = G.me.profile;
    const spent = Object.values(p.talents).reduce((s, v) => s + v, 0);
    body.innerHTML = `<h2>Таланти · вільних очок: <span style="color:var(--green)">${p.points}</span></h2>
      <div style="color:var(--dim);font-size:13px;margin-bottom:14px">Кожен рівень героя дає 1 очко. Поєднуйте гілки як завгодно — роль у команді визначаєте ви.
      <button id="respec" style="margin-left:12px" ${spent ? '' : 'disabled'}>Скинути таланти (◆ ${20 + spent * 5})</button></div>
      <div class="branches">${Object.entries(BRANCHES).map(([bid, b]) => `<div class="branch" style="border-color:${b.color}55">
        <h3 style="color:${b.color}">${b.name}</h3><div class="bd">${b.desc}</div>
        ${Object.entries(TALENTS).filter(([, t]) => t.branch === bid).map(([id, t]) => {
          const lv = p.talents[id] || 0;
          return `<div class="tal" style="border-color:${lv ? b.color + '88' : ''}"><div class="tx">${t.name}<small>${t.desc}</small></div><div class="lv" style="color:${lv ? b.color : 'var(--dim)'}">${lv}/${t.max}</div>
            <button data-t="${id}" ${p.points <= 0 || lv >= t.max ? 'disabled' : ''}>+</button></div>`;
        }).join('')}</div>`).join('')}</div>`;
    body.querySelectorAll('[data-t]').forEach((b) => b.addEventListener('click', () => { G.send({ t: 'talent', id: b.dataset.t }); G.audio.play('upgrade'); }));
    $('respec').onclick = () => { if (confirm('Скинути всі таланти?')) G.send({ t: 'respec' }); };
  }

  renderResearch(body) {
    const G = this.G, R = G.world.research, cr = G.snap?.crystals || 0;
    const groups = {};
    for (const [id, r] of Object.entries(RESEARCH)) (groups[r.branch] ||= []).push([id, r]);
    body.innerHTML = `<h2>Дослідження · ◆ <span style="color:var(--cry)">${fmt(cr)}</span> кристалів</h2>
      <div style="color:var(--dim);font-size:13px;margin-bottom:14px">Спільні для всієї команди. Кристали дають шахти на родовищах (клавіша F біля родовища), табори монстрів, боси та розбір предметів.</div>
      ${Object.entries(groups).map(([g, list]) => `<div class="rs-branch"><h2 style="font-size:16px;color:var(--cry)">${g}</h2><div class="rs-grid">
        ${list.map(([id, r]) => {
          const lv = R[id] || 0;
          const done = lv >= r.max;
          const lock = r.req && !R[r.req];
          const cost = researchCost(id, lv);
          return `<div class="rs ${done ? 'done' : ''} ${lock ? 'lock' : ''}"><div class="tx">${r.name} ${r.max > 1 ? `<span style="color:var(--dim)">${lv}/${r.max}</span>` : ''}<small>${r.desc}${lock ? ` · потрібно: ${RESEARCH[r.req].name}` : ''}</small></div>
          <button data-r="${id}" ${done || lock || cr < cost ? 'disabled' : ''}>${done ? '✓' : '◆ ' + fmt(cost)}</button></div>`;
        }).join('')}</div></div>`).join('')}`;
    body.querySelectorAll('[data-r]').forEach((b) => b.addEventListener('click', () => { G.send({ t: 'research', id: b.dataset.r }); G.audio.play('upgrade'); }));
  }

  renderStats(body) {
    const G = this.G, st = G.world.stats || {}, s = G.snap;
    const h = Math.floor((st.playTime || 0) / 3600), m = Math.floor(((st.playTime || 0) % 3600) / 60);
    const mines = G.world.extractors.filter((e) => e.level > 0);
    const rate = mines.reduce((a, e) => a + extractorRate(e.level, G.world.research), 0);
    const nextBoss = Math.ceil(s.wave / 10) * 10;
    body.innerHTML = `<div class="cols"><div class="col"><h2>Світ</h2><div class="stats-grid" style="grid-template-columns:1fr">
      <div><span>Складність</span><b>${DIFFICULTY[G.world.cfg.difficulty].name}</b></div>
      <div><span>Поточна хвиля</span><b>${s.wave}</b></div>
      <div><span>Найкраща відбита хвиля</span><b>${st.bestWave || 0}</b></div>
      <div><span>Наступний бос</span><b>хвиля ${nextBoss}</b></div>
      <div><span>Ворогів знищено</span><b>${fmt(st.kills || 0)}</b></div>
      <div><span>Босів знищено</span><b>${st.bosses || 0}</b></div>
      <div><span>Падінь ядра</span><b>${st.defeats || 0}</b></div>
      <div><span>Час у грі</span><b>${h} год ${m} хв</b></div>
      <div><span>Веж збудовано</span><b>${G.world.towers.length}</b></div>
      <div><span>Шахт</span><b>${mines.length} / ${G.world.extractors.length} · ◆ ${rate.toFixed(2)}/с</b></div>
      </div></div>
      <div class="col"><h2>Бестіарій</h2><div class="stats-grid" style="grid-template-columns:1fr">
      ${Object.entries(ENEMIES).filter(([, e]) => !e.wild).map(([, e]) => `<div><span style="color:${hex(e.color)}">${e.name}${e.boss ? ' (бос)' : ''}${e.flying ? ' · літає' : ''}</span><b>${e.armor ? 'броня ' + e.armor : ''}${e.shield ? ' щит' : ''}${e.heal ? ' лікує' : ''}</b></div>`).join('')}
      </div><div style="color:var(--dim);font-size:12px;margin-top:10px">Сила хвиль зростає з кожним номером незалежно від вашого рівня. Що швидше прокачуєтесь, то легше їх відбивати.</div></div></div>`;
  }

  renderSettings(body) {
    const G = this.G;
    const q = G.settings.quality;
    body.innerHTML = `<h2>Налаштування</h2>
      <div class="set-row"><label>Якість графіки</label><div class="seg" id="set-q">${Object.entries(QUALITY).map(([k, v]) => `<button data-v="${k}" class="${k === q ? 'on' : ''}">${v.name}</button>`).join('')}</div></div>
      <div class="set-row"><label>Гучність ефектів</label><input type="range" id="set-vol" min="0" max="1" step="0.05" value="${G.settings.vol}"></div>
      <div class="set-row"><label>Гучність музики</label><input type="range" id="set-mus" min="0" max="1" step="0.05" value="${G.settings.music}"></div>
      <div class="set-row"><label>Показувати цифри шкоди</label><input type="checkbox" id="set-dmg" ${G.settings.dmgNums ? 'checked' : ''}></div>
      <div class="set-row"><label>Тряска камери</label><input type="checkbox" id="set-shake" ${G.settings.shake ? 'checked' : ''}></div>
      <div class="set-row"><label>Показувати FPS</label><input type="checkbox" id="set-fps" ${G.settings.fps ? 'checked' : ''}></div>
      <div class="set-row"><label>Час між нападами ${G.host ? '' : '(змінює лише хост)'}</label><div class="seg" id="set-prep">${[60, 120, 180, 300, 600, 900].map((v) => `<button data-v="${v}" class="${G.world.cfg.prepTime === v ? 'on' : ''}" ${G.host ? '' : 'disabled'}>${v / 60} хв</button>`).join('')}</div></div>
      <div class="set-row"><label>Збереження</label><button id="set-save">Зберегти зараз</button><span style="color:var(--dim);font-size:12px">Автозбереження після кожної хвилі та щохвилини</span></div>
      ${window.bastion ? '<div class="set-row"><label>Вихід</label><button id="set-quit">Зберегти й вийти</button></div>' : ''}
      <div style="color:var(--dim);font-size:12px;margin-top:20px;line-height:1.8">F11 — повноекранний режим · колесо миші — масштаб камери · ПКМ на вежі/предметі — швидкі дії.<br>
      Якщо гра гальмує через RDP, увімкніть на хост-ПК групову політику «Use hardware graphics adapters for all Remote Desktop Services sessions» або знизьте якість.</div>`;
    body.querySelectorAll('#set-q button').forEach((b) => b.addEventListener('click', () => { G.applySettings({ quality: b.dataset.v }); this.renderPanel(true); }));
    $('set-vol').oninput = (e) => G.applySettings({ vol: +e.target.value });
    $('set-mus').oninput = (e) => G.applySettings({ music: +e.target.value });
    $('set-dmg').onchange = (e) => G.applySettings({ dmgNums: e.target.checked });
    $('set-shake').onchange = (e) => G.applySettings({ shake: e.target.checked });
    $('set-fps').onchange = (e) => G.applySettings({ fps: e.target.checked });
    body.querySelectorAll('#set-prep button').forEach((b) => b.addEventListener('click', () => G.send({ t: 'cfg', prepTime: +b.dataset.v })));
    $('set-save').onclick = () => G.send({ t: 'saveNow' });
    if ($('set-quit')) $('set-quit').onclick = () => { G.send({ t: 'saveNow' }); setTimeout(() => window.bastion.quit(), 300); };
  }

  extractorPrompt(ex, dep) {
    const G = this.G;
    const lvl = ex.level;
    const cost = lvl === 0 ? EXTRACTOR_COST : extractorUpgradeCost(lvl);
    const cur = extractorRate(lvl, G.world.research), nx = extractorRate(lvl + 1, G.world.research);
    return lvl === 0
      ? `<b>[F]</b> Збудувати шахту кристалів — ◈ ${fmt(cost)} <span style="color:var(--cry)">(◆ ${nx.toFixed(2)}/с)</span>`
      : `Шахта рів. ${lvl} · ◆ ${cur.toFixed(2)}/с — <b>[F]</b> покращити за ◈ ${fmt(cost)} <span style="color:var(--cry)">(→ ${nx.toFixed(2)}/с)</span>`;
  }
}
export { heroStats };
