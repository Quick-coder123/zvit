// «Відчуття» гри (game-feel): тряска на травмі та легкі анімації. Без залежностей — тестується в Node.

// Тряска камери: удари додають травму, зсув = trauma² × плавні синуси, спад у реальному часі.
export class Trauma {
  constructor({ decay = 1.4, maxOffset = 0.9, maxRoll = 0.025, strength = 1 } = {}) {
    Object.assign(this, { decay, maxOffset, maxRoll, strength });
    this.trauma = 0; this.t = 0; this.x = 0; this.y = 0; this.roll = 0;
  }
  add(amount) { this.trauma = Math.min(1, this.trauma + amount); }
  update(realDt) {
    if (this.trauma <= 0) { this.x = this.y = this.roll = 0; return; }
    this.trauma = Math.max(0, this.trauma - this.decay * realDt);
    const s = this.trauma * this.trauma * this.strength;
    this.t += realDt * 30;
    this.x = this.maxOffset * s * Math.sin(this.t * 1.7);
    this.y = this.maxOffset * s * Math.sin(this.t * 2.3 + 1.3);
    this.roll = this.maxRoll * s * Math.sin(this.t * 1.1 + 0.4);
    if (this.trauma === 0) this.x = this.y = this.roll = 0;
  }
}

// easeOutBack: швидке «вистрибування» з перельотом і точним приземленням на 1.
export function popScale(k, from = 0.4, overshoot = 2.2) {
  if (k >= 1) return 1;
  const c3 = overshoot + 1, u = k - 1;
  const e = 1 + c3 * u * u * u + overshoot * u * u;
  return from + (1 - from) * e;
}
