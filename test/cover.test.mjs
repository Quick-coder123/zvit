// TDD: будинки — укриття для куль і перешкоди для руху ДРГ (серверна симуляція).
import { test } from 'node:test';
import assert from 'node:assert/strict';
import os from 'node:os';
import path from 'node:path';
import fs from 'node:fs';
import { Game } from '../server/game.js';
import { cityBuildings, insideRect, toWorld } from '../shared/collision.js';
import { TICK } from '../shared/data.js';

function makeGame() {
  const g = new Game(fs.mkdtempSync(path.join(os.tmpdir(), 'lastline-cover-')));
  g.save = () => {};
  const ws = { readyState: 1, send() {} };
  g.onConnect(ws);
  g.onMessage(ws, JSON.stringify({ t: 'join', name: 'Тест' }));
  g.onMessage(ws, JSON.stringify({ t: 'setup', mode: 'new', difficulty: 'normal', prepTime: 900 }));
  for (const e of [...g.enemies.values()]) g.enemies.delete(e.id);
  const hero = [...g.clients.values()][0];
  return { g, hero };
}
// точки по обидва боки будинку вздовж його локальної осі Z
function across(b, dist) {
  const half = b.d / 2 + dist;
  return [toWorld(0, -half, b), toWorld(0, half, b)];
}

test('куля бійця не пролітає крізь будинок', () => {
  const { g, hero } = makeGame();
  const b = cityBuildings()[0];
  const [[hx, hz], [ex, ez]] = across(b, 3);
  hero.x = hx; hero.z = hz; hero.ax = ex; hero.az = ez; hero.fire = true;
  const e = g.spawnEnemy('w_rifle', ex, ez, { hp: 1000, dmg: 0, camp: -1 });
  e.state = 'leash'; // стоїть на місці
  for (let i = 0; i < 40; i++) g.tick();
  assert.equal(e.hp, 1000, 'ворог за стіною не отримав шкоди');
});

test('куля ДРГ не дістає бійця за будинком', () => {
  const { g, hero } = makeGame();
  const b = cityBuildings()[1];
  const [[hx, hz], [ex, ez]] = across(b, 2);
  hero.x = hx; hero.z = hz; hero.fire = false;
  const hp0 = hero.hp;
  g.addProjectile({ k: 'ebullet', x: ex, y: 1.4, z: ez, vx: (hx - ex) * 10, vy: 0, vz: (hz - ez) * 10, ttl: 0.5, dmg: 50, owner: null, hostile: true });
  for (let i = 0; i < 12; i++) g.updateProjectiles(TICK, [hero]);
  assert.equal(hero.hp, hp0);
});

test('ДРГ, що переслідує бійця, ніколи не заходить у будинок', () => {
  const { g, hero } = makeGame();
  const b = cityBuildings()[2];
  const [[hx, hz], [ex, ez]] = across(b, 9);
  hero.x = hx; hero.z = hz; hero.hp = 1e9; hero.stats.maxHp = 1e9;
  const e = g.spawnEnemy('w_rifle', ex, ez, { hp: 1e9, dmg: 0, camp: -1 });
  e.homeX = ex; e.homeZ = ez; e.state = 'chase'; e.target = hero;
  for (let i = 0; i < 400; i++) {
    g.tick();
    for (const bb of cityBuildings()) assert.equal(insideRect(e.x, e.z, bb), false, `крок ${i}: ворог усередині будинку`);
  }
});

test('ДРГ без прямої видимості не стріляє в стіну, а обходить будинок до видимості', () => {
  const { g, hero } = makeGame();
  const b = cityBuildings()[3];
  const [[hx, hz], [ex, ez]] = across(b, 3);
  hero.x = hx; hero.z = hz; hero.hp = 1e9; hero.stats.maxHp = 1e9;
  const e = g.spawnEnemy('w_rifle', ex, ez, { hp: 1e9, dmg: 1, camp: -1 });
  e.homeX = ex; e.homeZ = ez; e.state = 'chase'; e.target = hero;
  let wallShots = 0, sawHero = false;
  const origAdd = g.addProjectile.bind(g);
  g.addProjectile = (p) => { if (p.k === 'ebullet' && !sawHeroNow()) wallShots++; return origAdd(p); };
  const sawHeroNow = () => !cityBuildings().some((bb) => g.blockedLOS(e.x, e.z, hero.x, hero.z));
  for (let i = 0; i < 600 && !sawHero; i++) { g.tick(); if (sawHeroNow()) sawHero = true; }
  assert.equal(wallShots, 0, 'жодного пострілу в стіну');
  assert.ok(sawHero, 'ворог вийшов на лінію вогню');
});
