// Синтезовані звуки бою (WebAudio) + фонові звуки погоди. Без зовнішніх файлів — працює офлайн.
export class Audio {
  constructor() { this.ctx = null; this.vol = 0.5; this.musicVol = 0.35; this.last = {}; this.wx = 'clear'; }
  init() {
    if (this.ctx) return;
    try {
      this.ctx = new (window.AudioContext || window.webkitAudioContext)();
      this.master = this.ctx.createGain();
      this.master.gain.value = this.vol;
      const comp = this.ctx.createDynamicsCompressor();
      comp.threshold.value = -18; comp.ratio.value = 6;
      this.master.connect(comp).connect(this.ctx.destination);
      const len = this.ctx.sampleRate * 2;
      this.noiseBuf = this.ctx.createBuffer(1, len, this.ctx.sampleRate);
      const d = this.noiseBuf.getChannelData(0);
      for (let i = 0; i < len; i++) d[i] = Math.random() * 2 - 1;
      // коричневий шум для вибухів і вітру
      this.brownBuf = this.ctx.createBuffer(1, len, this.ctx.sampleRate);
      const b = this.brownBuf.getChannelData(0);
      let last = 0;
      for (let i = 0; i < len; i++) { last = (last + 0.02 * (Math.random() * 2 - 1)) / 1.02; b[i] = last * 3.5; }
      this.startAmbient();
      this.setWeather(this.wx);
    } catch { this.ctx = null; }
  }
  setVolume(v) { this.vol = v; if (this.master) this.master.gain.value = v; }
  setMusic(v) { this.musicVol = v; this.applyAmbient(); }
  setIntensity() {}
  throttle(k, ms) { const now = performance.now(); if (this.last[k] && now - this.last[k] < ms) return false; this.last[k] = now; return true; }
  tone(type, f0, f1, dur, gain = 0.2, delay = 0) {
    if (!this.ctx) return;
    const t = this.ctx.currentTime + delay;
    const o = this.ctx.createOscillator(), g = this.ctx.createGain();
    o.type = type;
    o.frequency.setValueAtTime(f0, t);
    o.frequency.exponentialRampToValueAtTime(Math.max(20, f1), t + dur);
    g.gain.setValueAtTime(gain, t);
    g.gain.exponentialRampToValueAtTime(0.0001, t + dur);
    o.connect(g).connect(this.master);
    o.start(t); o.stop(t + dur + 0.02);
  }
  noise(dur, gain, freq, q = 0.7, delay = 0, type = 'lowpass', brown = false, attack = 0.002) {
    if (!this.ctx) return;
    const t = this.ctx.currentTime + delay;
    const s = this.ctx.createBufferSource();
    s.buffer = brown ? this.brownBuf : this.noiseBuf;
    const f = this.ctx.createBiquadFilter();
    f.type = type; f.frequency.value = freq; f.Q.value = q;
    const g = this.ctx.createGain();
    g.gain.setValueAtTime(0.0001, t);
    g.gain.linearRampToValueAtTime(gain, t + attack);
    g.gain.exponentialRampToValueAtTime(0.0001, t + dur);
    s.connect(f).connect(g).connect(this.master);
    s.start(t, Math.random()); s.stop(t + dur + 0.05);
  }
  // постріл: різкий тріск + низький «бум» + хвіст
  shot(vol, crack, body, tail) {
    this.noise(0.05, 0.5 * vol, crack, 0.8, 0, 'highpass');
    this.noise(body, 0.6 * vol, 900, 0.7);
    this.tone('sine', 140, 50, body, 0.35 * vol);
    if (tail) this.noise(tail, 0.12 * vol, 500, 0.5, 0.04, 'lowpass', true, 0.02);
  }
  play(name, vol = 1) {
    if (!this.ctx) return;
    switch (name) {
      case 'rifle': if (this.throttle(name, 70)) this.shot(0.35 * vol, 2500, 0.09, 0.25); break;
      case 'erifle': if (this.throttle(name, 90)) this.shot(0.22 * vol, 2000, 0.08, 0.2); break;
      case 'emg': if (this.throttle(name, 110)) { for (let i = 0; i < 3; i++) this.noise(0.07, 0.25 * vol, 1200, 0.7, i * 0.08); } break;
      case 'mg': if (this.throttle(name, 60)) this.shot(0.3 * vol, 2200, 0.08, 0.15); break;
      case 'ags': if (this.throttle(name, 120)) { this.tone('sine', 110, 60, 0.12, 0.3 * vol); this.noise(0.1, 0.25 * vol, 700); } break;
      case 'aa': if (this.throttle(name, 70)) this.shot(0.35 * vol, 1500, 0.12, 0.3); break;
      case 'atgm': if (this.throttle(name, 150)) { this.noise(0.9, 0.3 * vol, 1800, 0.6, 0, 'bandpass', false, 0.05); this.tone('sawtooth', 300, 120, 0.8, 0.04 * vol); } break;
      case 'mortar': if (this.throttle(name, 150)) { this.tone('sine', 90, 40, 0.35, 0.5 * vol); this.noise(0.3, 0.35 * vol, 500, 0.7, 0, 'lowpass', true); } break;
      case 'howitzer': if (this.throttle(name, 200)) { this.tone('sine', 70, 25, 0.9, 0.8 * vol); this.noise(1.6, 0.6 * vol, 400, 0.6, 0, 'lowpass', true); this.noise(0.08, 0.5 * vol, 1500, 0.8, 0, 'highpass'); } break;
      case 'cannon': if (this.throttle(name, 200)) { this.tone('sine', 80, 30, 0.7, 0.7 * vol); this.noise(1.2, 0.5 * vol, 500, 0.6, 0, 'lowpass', true); } break;
      case 'smallboom': if (this.throttle(name, 60)) { this.noise(0.5, 0.35 * vol, 900, 0.7, 0, 'lowpass', true); this.tone('sine', 100, 40, 0.3, 0.3 * vol); } break;
      case 'boom': if (this.throttle(name, 70)) { this.noise(1.2, 0.6 * vol, 600, 0.6, 0, 'lowpass', true); this.tone('sine', 70, 25, 0.6, 0.55 * vol); this.noise(0.08, 0.4 * vol, 2000, 0.7, 0, 'highpass'); } break;
      case 'bigboom': this.noise(2.4, 0.9 * vol, 450, 0.6, 0, 'lowpass', true); this.tone('sine', 55, 18, 1.4, 0.8 * vol); this.noise(0.12, 0.5 * vol, 1800, 0.7, 0, 'highpass'); break;
      case 'incoming': for (let i = 0; i < 3; i++) this.tone('sine', 1400 - i * 80, 500, 1.1, 0.04, i * 0.12); break;
      case 'thunder': this.noise(3.5, 0.8, 220, 0.5, 0, 'lowpass', true, 0.15); this.noise(1.5, 0.4, 600, 0.5, 0.1, 'lowpass', true, 0.05); break;
      case 'siren': { const t = this.ctx.currentTime; const o = this.ctx.createOscillator(), g = this.ctx.createGain(); o.type = 'sawtooth'; o.frequency.setValueAtTime(300, t); o.frequency.linearRampToValueAtTime(700, t + 1.4); o.frequency.linearRampToValueAtTime(300, t + 2.8); const f = this.ctx.createBiquadFilter(); f.type = 'lowpass'; f.frequency.value = 1200; g.gain.setValueAtTime(0.0001, t); g.gain.linearRampToValueAtTime(0.07, t + 0.3); g.gain.linearRampToValueAtTime(0.0001, t + 2.9); o.connect(f).connect(g).connect(this.master); o.start(t); o.stop(t + 3); break; }
      case 'hurt': if (this.throttle(name, 250)) { this.noise(0.12, 0.25, 400, 1, 0, 'bandpass'); this.tone('sine', 120, 60, 0.15, 0.2); } break;
      case 'dash': this.noise(0.35, 0.15, 800, 0.8, 0, 'bandpass', true, 0.05); break;
      case 'build': this.noise(0.15, 0.2, 300, 1, 0, 'bandpass', true); this.noise(0.15, 0.2, 260, 1, 0.18, 'bandpass', true); break;
      case 'upgrade': this.tone('square', 600, 600, 0.05, 0.05); this.noise(0.08, 0.15, 3000, 2, 0.06, 'bandpass'); this.tone('square', 800, 800, 0.05, 0.05, 0.12); break;
      case 'sell': this.noise(0.25, 0.15, 400, 1, 0, 'bandpass', true); break;
      case 'pick': this.noise(0.06, 0.2, 2500, 3, 0, 'bandpass'); this.noise(0.06, 0.2, 1800, 3, 0.08, 'bandpass'); break;
      case 'legend': [523, 659, 784].forEach((f, i) => this.tone('triangle', f, f, 0.3, 0.07, i * 0.1)); break;
      case 'clear': [392, 523, 659].forEach((f, i) => this.tone('triangle', f, f, 0.35, 0.06, i * 0.15)); break;
      case 'defeat': [300, 240, 180].forEach((f, i) => this.tone('sawtooth', f, f * 0.95, 0.5, 0.06, i * 0.3)); break;
      case 'ui': this.noise(0.03, 0.12, 3000, 2, 0, 'bandpass'); break;
      case 'error': this.tone('square', 180, 150, 0.12, 0.05); break;
    }
  }
  // фон: вітер + дощ
  startAmbient() {
    const c = this.ctx;
    const mk = (buf, type, freq, q) => {
      const s = c.createBufferSource(); s.buffer = buf; s.loop = true;
      const f = c.createBiquadFilter(); f.type = type; f.frequency.value = freq; f.Q.value = q;
      const g = c.createGain(); g.gain.value = 0;
      s.connect(f).connect(g).connect(c.destination); s.start();
      return { g, f };
    };
    this.wind = mk(this.brownBuf, 'lowpass', 500, 0.7);
    this.rainS = mk(this.noiseBuf, 'highpass', 2500, 0.4);
    const lfo = c.createOscillator(), lg = c.createGain();
    lfo.frequency.value = 0.08; lg.gain.value = 250;
    lfo.connect(lg).connect(this.wind.f.frequency); lfo.start();
  }
  setWeather(kind) {
    this.wx = kind;
    this.applyAmbient();
  }
  applyAmbient() {
    if (!this.ctx || !this.wind) return;
    const k = this.wx;
    const wind = { clear: 0.12, cloudy: 0.18, rain: 0.2, storm: 0.45, fog: 0.08, snow: 0.2, blizzard: 0.7 }[k] ?? 0.15;
    const rain = { rain: 0.1, storm: 0.16 }[k] ?? 0;
    const t = this.ctx.currentTime, v = this.musicVol * 1.4;
    this.wind.g.gain.setTargetAtTime(wind * v, t, 2);
    this.rainS.g.gain.setTargetAtTime(rain * v, t, 2);
  }
}
