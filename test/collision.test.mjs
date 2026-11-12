// TDD: будинки міста — справжні укриття (колізії руху й куль), спільні для сервера і клієнта.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { cityBuildings, insideRect, pushOutOfRects, segmentHitsRect } from '../shared/collision.js';
import { distToLanes, CAMPS, DEPOSITS, inCity } from '../shared/data.js';

const box = { x: 10, z: 5, w: 8, d: 4, rot: 0.6 };

test('insideRect враховує поворот прямокутника', () => {
  assert.equal(insideRect(10, 5, box), true);
  // точка вздовж локальної осі X на 3.5 м — всередині (півширина 4)
  const lx = 3.5, px = 10 + lx * Math.cos(0.6), pz = 5 - lx * Math.sin(0.6);
  assert.equal(insideRect(px, pz, box), true);
  // та сама відстань уздовж локальної Z — зовні (півглибина 2)
  const qx = 10 + 3.5 * Math.sin(0.6), qz = 5 + 3.5 * Math.cos(0.6);
  assert.equal(insideRect(qx, qz, box), false);
});

test('pushOutOfRects виштовхує точку на найближчу грань з відступом', () => {
  const [x, z] = pushOutOfRects(10.2, 5.1, [box], 0.5);
  assert.equal(insideRect(x, z, box, 0.49), false);
  assert.ok(Math.hypot(x - 10.2, z - 5.1) < 3, 'зсув невеликий — до найближчої грані');
  const [x2, z2] = pushOutOfRects(50, 50, [box], 0.5);
  assert.deepEqual([x2, z2], [50, 50], 'точку зовні не чіпаємо');
});

test('segmentHitsRect: постріл крізь будинок перетинає його, повз — ні', () => {
  assert.equal(segmentHitsRect(0, 5, 20, 5, box), true);
  assert.equal(segmentHitsRect(0, 30, 20, 30, box), false);
  assert.equal(segmentHitsRect(9, 5, 11, 5, box), true, 'відрізок цілком усередині');
});

test('cityBuildings: детерміновані, у місті, не на дорогах, позиціях ДРГ, складах і не перекриваються', () => {
  const a = cityBuildings(), b = cityBuildings();
  assert.ok(a.length >= 10, `будинків ${a.length}`);
  assert.deepEqual(a, b);
  for (const [i, h] of a.entries()) {
    assert.ok(inCity(h.x, h.z) > 0.5);
    const rad = Math.hypot(h.w, h.d) / 2;
    assert.ok(distToLanes(h.x, h.z) > rad + 2, 'не на дорозі');
    for (const c of CAMPS) assert.ok(Math.hypot(c.x - h.x, c.z - h.z) > rad + 6, 'не на позиції ДРГ');
    for (const d of DEPOSITS) assert.ok(Math.hypot(d.x - h.x, d.z - h.z) > rad + 8, 'не на складі');
    for (const o of a.slice(i + 1)) assert.ok(Math.hypot(o.x - h.x, o.z - h.z) > rad + Math.hypot(o.w, o.d) / 2, 'не перекриваються');
  }
});
