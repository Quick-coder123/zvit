// Перевірки з навички tower-defense: рух по маршруту без втрат на поворотах і наведення.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import os from 'node:os';
import path from 'node:path';
import fs from 'node:fs';
import { Game } from '../server/game.js';
import { LANES, TICK } from '../shared/data.js';

function makeGame() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'lastline-test-'));
  const g = new Game(dir);
  g.save = () => {};
  const ws = { readyState: 1, send() {} };
  g.onConnect(ws);
  g.onMessage(ws, JSON.stringify({ t: 'join', name: 'Тест' }));
  g.onMessage(ws, JSON.stringify({ t: 'setup', mode: 'new', difficulty: 'normal', prepTime: 600 }));
  for (const e of [...g.enemies.values()]) g.enemies.delete(e.id); // прибираємо ДРГ
  return g;
}
function laneLength(lane) {
  let L = 0;
  for (let i = 1; i < lane.length; i++) L += Math.hypot(lane[i][0] - lane[i - 1][0], lane[i][1] - lane[i - 1][1]);
  return L;
}

test('швидкий ворог (speed ×10) проходить маршрут за path_length / speed з точністю до одного тіку', () => {
  const g = makeGame();
  const lane = LANES[0];
  const e = g.spawnEnemy('moto', lane[0][0], lane[0][1], { hp: 1e9, lane: 0, off: 0 });
  e.speed *= 10;
  const L = laneLength(lane);
  let t = 0, reached = false;
  while (t < 60 && !reached) { reached = g.moveAlongLane(e, e.speed * TICK); t += TICK; }
  assert.ok(reached, 'ворог дійшов до штабу');
  assert.ok(Math.abs(t - L / e.speed) <= TICK + 1e-9, `час ${t.toFixed(3)} проти ${(L / e.speed).toFixed(3)}`);
  assert.ok(Math.abs(e.prog - L) < 1e-6, 'пройдена відстань дорівнює довжині маршруту');
});

function towerWithTwoTargets(mode) {
  const g = makeGame();
  const t = g.addTower('mg', 0, 20, 1, 100);
  t.mode = mode;
  // ворог A: далі по маршруту, але далі від позиції; ворог B: ближче, але менше пройшов
  const a = g.spawnEnemy('infantry', 0, 36, { hp: 500, lane: 0 }); a.prog = 90;
  const b = g.spawnEnemy('infantry', 0, 22, { hp: 50, lane: 0 }); b.prog = 30;
  g.coreHp = 1e9;
  g.updateTowers(TICK, []);
  return { t, a, b };
}

test('режим «перший» обирає ворога з найбільшим прогресом, навіть якщо інший ближче', () => {
  const { t, a } = towerWithTwoTargets('first');
  assert.equal(t.target, a.id);
});
test('режим «найближчий» обирає найближчого', () => {
  const { t, b } = towerWithTwoTargets('closest');
  assert.equal(t.target, b.id);
});
test('режими «найсильніший» / «найслабший» дивляться на здоров’я', () => {

  const s = towerWithTwoTargets('strongest'); assert.equal(s.t.target, s.a.id);
  const w = towerWithTwoTargets('weakest'); assert.equal(w.t.target, w.b.id);
});
