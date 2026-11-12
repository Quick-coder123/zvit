// TDD: тряска камери на «травмі» (game-feel): trauma² з плавними синусами, спад у реальному часі.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { Trauma, popScale } from '../client/feel.js';

test('удари додаються, але травма не перевищує 1', () => {
  const t = new Trauma();
  t.add(0.7); t.add(0.7);
  assert.equal(t.trauma, 1);
});
test('зсув пропорційний trauma²: легкий удар майже не трясе, сильний — відчутно', () => {
  const a = new Trauma({ maxOffset: 1 }), b = new Trauma({ maxOffset: 1 });
  a.add(0.2); b.add(1);
  let ma = 0, mb = 0;
  for (let i = 0; i < 30; i++) { a.trauma = 0.2; b.trauma = 1; a.update(1 / 60); b.update(1 / 60); ma = Math.max(ma, Math.hypot(a.x, a.y)); mb = Math.max(mb, Math.hypot(b.x, b.y)); }
  assert.ok(ma < mb * 0.1, `легкий ${ma.toFixed(3)} проти сильного ${mb.toFixed(3)}`);
});
test('травма спадає до нуля за 1/decay секунд, і камера точно повертається на місце', () => {
  const t = new Trauma({ decay: 1.5 });
  t.add(1);
  let time = 0;
  while (t.trauma > 0 && time < 5) { t.update(1 / 60); time += 1 / 60; }
  assert.ok(time <= 1 / 1.5 + 1 / 60 + 1e-9, `час ${time}`);
  assert.equal(t.x, 0); assert.equal(t.y, 0); assert.equal(t.roll, 0);
});
test('сила 0 % (налаштування доступності) вимикає тряску повністю', () => {
  const t = new Trauma({ strength: 0 });
  t.add(1);
  t.update(1 / 60);
  assert.equal(Math.hypot(t.x, t.y, t.roll), 0);
});
test('popScale: «вистрибування» з перельотом за 1 і точним поверненням до 1 в кінці', () => {
  let peak = 0;
  for (let i = 0; i <= 20; i++) peak = Math.max(peak, popScale(i / 20));
  assert.ok(peak > 1.05, `пік ${peak}`);
  assert.equal(popScale(1), 1);
});
