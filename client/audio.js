// Синтезовані звуки (WebAudio) — без зовнішніх файлів, працює офлайн.
export class Audio {
  constructor() {
    this.ctx = null;
    this.vol = 0.5;
    this.musicVol = 0.35;
    this.last = {};
  }
  init() {
    if (this.ctx) return;
    try {
      this.ctx = new (window.AudioContext || window.webkitAudioContext)();
      this.master = this.ctx.createGain();
      this.master.gain.value = this.vol;
      const comp = this.ctx.createDynamicsCompressor();
      this.master.connect(comp).connect(this.ctx.destination);
      this.noiseBuf = this.ctx.createBuffer(1, this.ctx.sampleRate, this.ctx.sampleRate);
      const d = this.noiseBuf.getChannelData(0);
      for (let i = 0; i < d.length; i++) d[i] = Math.random() * 2 - 1;
      this.startMusic();
    } catch { this.ctx = null; }
  }
  setVolume(v) { this.vol = v; if (this.master) this.master.gain.value = v; }
  setMusic(v) { this.musicVol = v; if (this.musicGain) this.musicGain.gain.value = v * 0.18; }
  throttle(k, ms) {
    const now = performance.now();
    if (this.last[k] && now - this.last[k] < ms) return false;
    this.last[k] = now;
    return true;
  }
  tone(type, f0, f1, dur, gain = 0.2, delay = 0) {
    if (!this.ctx) return;
    const t = this.ctx.currentTime + delay;
    const o = this.ctx.createOscillator();
    const g = this.ctx.createGain();
    o.type = type;
    o.frequency.setValueAtTime(f0, t);
    o.frequency.exponentialRampToValueAtTime(Math.max(20, f1), t + dur);
    g.gain.setValueAtTime(gain, t);
    g.gain.exponentialRampToValueAtTime(0.0001, t + dur);
    o.connect(g).connect(this.master);
    o.start(t); o.stop(t + dur + 0.02);
  }
  noise(dur, gain = 0.3, freq = 1200, q = 0.7, delay = 0, type = 'lowpass') {
    if (!this.ctx) return;
    const t = this.ctx.currentTime + delay;
    const s = this.ctx.createBufferSource();
    s.buffer = this.noiseBuf;
    const f = this.ctx.createBiquadFilter();
    f.type = type; f.frequency.value = freq; f.Q.value = q;
    const g = this.ctx.createGain();
    g.gain.setValueAtTime(gain, t);
    g.gain.exponentialRampToValueAtTime(0.0001, t + dur);
    s.connect(f).connect(g).connect(this.master);
    s.start(t, Math.random() * 0.5); s.stop(t + dur + 0.02);
  }
  play(name, vol = 1) {
    if (!this.ctx) return;
    switch (name) {
      case 'shoot': if (this.throttle(name, 60)) this.tone('square', 900, 300, 0.07, 0.04 * vol); break;
      case 'blaster': if (this.throttle(name, 50)) this.tone('sawtooth', 1200, 400, 0.08, 0.025 * vol); break;
      case 'rail': if (this.throttle(name, 80)) { this.tone('sawtooth', 2200, 80, 0.35, 0.06 * vol); this.noise(0.25, 0.06 * vol, 4000, 1, 0, 'highpass'); } break;
      case 'tesla': if (this.throttle(name, 80)) this.noise(0.18, 0.08 * vol, 3000, 4, 0, 'bandpass'); break;
      case 'cryo': if (this.throttle(name, 90)) this.noise(0.2, 0.04 * vol, 6000, 1, 0, 'highpass'); break;
      case 'mortar': if (this.throttle(name, 120)) this.tone('sine', 160, 50, 0.3, 0.12 * vol); break;
      case 'boom': if (this.throttle(name, 60)) { this.noise(0.6, 0.22 * vol, 500); this.tone('sine', 120, 30, 0.5, 0.2 * vol); } break;
      case 'bigboom': this.noise(1.4, 0.4 * vol, 400); this.tone('sine', 90, 20, 1.2, 0.4 * vol); break;
      case 'die': if (this.throttle(name, 40)) this.noise(0.15, 0.05 * vol, 1800, 2, 0, 'bandpass'); break;
      case 'hit': if (this.throttle(name, 50)) this.tone('triangle', 300, 120, 0.06, 0.05 * vol); break;
      case 'hurt': if (this.throttle(name, 200)) this.tone('sawtooth', 200, 80, 0.15, 0.1 * vol); break;
      case 'build': this.tone('square', 300, 600, 0.12, 0.08); this.tone('square', 600, 900, 0.12, 0.06, 0.1); break;
      case 'upgrade': [500, 700, 1000].forEach((f, i) => this.tone('triangle', f, f * 1.02, 0.15, 0.08, i * 0.07)); break;
      case 'sell': this.tone('triangle', 800, 300, 0.2, 0.08); break;
      case 'coin': if (this.throttle(name, 100)) this.tone('sine', 1500, 1900, 0.08, 0.03); break;
      case 'pick': [800, 1200].forEach((f, i) => this.tone('sine', f, f, 0.1, 0.08, i * 0.06)); break;
      case 'legend': [523, 659, 784, 1046].forEach((f, i) => this.tone('triangle', f, f, 0.35, 0.12, i * 0.1)); break;
      case 'lvl': [392, 523, 659, 784, 1046].forEach((f, i) => this.tone('square', f, f, 0.18, 0.06, i * 0.08)); break;
      case 'wave': this.tone('sawtooth', 110, 110, 1.2, 0.12); this.tone('sawtooth', 165, 165, 1.2, 0.08); this.tone('sawtooth', 110, 55, 0.8, 0.1, 1.1); break;
      case 'boss': for (let i = 0; i < 3; i++) this.tone('sawtooth', 70, 50, 0.6, 0.2, i * 0.7); break;
      case 'clear': [523, 659, 784].forEach((f, i) => this.tone('triangle', f, f, 0.4, 0.1, i * 0.12)); break;
      case 'defeat': [400, 300, 200, 100].forEach((f, i) => this.tone('sawtooth', f, f * 0.9, 0.4, 0.12, i * 0.25)); break;
      case 'dash': this.noise(0.25, 0.12, 2500, 1, 0, 'bandpass'); break;
      case 'core': if (this.throttle(name, 150)) this.tone('square', 180, 90, 0.25, 0.1); break;
      case 'ui': this.tone('sine', 1000, 1000, 0.04, 0.04); break;
      case 'error': this.tone('square', 200, 150, 0.15, 0.06); break;
      case 'stomp': this.noise(0.8, 0.35, 200); this.tone('sine', 60, 25, 0.8, 0.35); break;
    }
  }
  // фоновий ембієнт-дрон
  startMusic() {
    const c = this.ctx;
    this.musicGain = c.createGain();
    this.musicGain.gain.value = this.musicVol * 0.18;
    const filt = c.createBiquadFilter();
    filt.type = 'lowpass'; filt.frequency.value = 600; filt.Q.value = 2;
    this.musicGain.connect(c.destination);
    filt.connect(this.musicGain);
    const lfo = c.createOscillator();
    const lfoG = c.createGain();
    lfo.frequency.value = 0.05; lfoG.gain.value = 350;
    lfo.connect(lfoG).connect(filt.frequency);
    lfo.start();
    for (const [f, type, g] of [[55, 'sawtooth', 0.4], [82.4, 'sawtooth', 0.25], [110.5, 'triangle', 0.3], [164.8, 'sine', 0.2]]) {
      const o = c.createOscillator();
      const og = c.createGain();
      o.type = type; o.frequency.value = f; og.gain.value = g;
      o.detune.value = (Math.random() - 0.5) * 12;
      o.connect(og).connect(filt);
      o.start();
    }
    this.musicFilter = filt;
  }
  setIntensity(k) {
    if (this.musicFilter) this.musicFilter.frequency.setTargetAtTime(500 + k * 900, this.ctx.currentTime, 2);
  }
}
