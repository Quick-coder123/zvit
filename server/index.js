// HTTP (статичні файли клієнта) + WebSocket сервер гри.
import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { WebSocketServer } from 'ws';
import { Game } from './game.js';
import { TICK } from '../shared/data.js';

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const MIME = { '.html': 'text/html; charset=utf-8', '.js': 'text/javascript; charset=utf-8', '.css': 'text/css; charset=utf-8', '.json': 'application/json', '.png': 'image/png', '.svg': 'image/svg+xml', '.ico': 'image/x-icon', '.ttf': 'font/ttf' };

function resolveThree() {
  const candidates = [path.join(ROOT, 'node_modules', 'three'), path.join(ROOT, '..', 'node_modules', 'three')];
  return candidates.find((c) => fs.existsSync(c)) || candidates[0];
}

export function startServer({ port, saveDir, host = '0.0.0.0', autoContinue = false }) {
  const three = resolveThree();
  const jsmLocal = path.join(ROOT, 'client', 'vendor', 'jsm');
  const jsm = fs.existsSync(jsmLocal) ? jsmLocal : path.join(three, 'examples', 'jsm');
  const routes = [
    ['/vendor/jsm/', jsm + path.sep],
    ['/vendor/three/', three + path.sep],
    ['/shared/', path.join(ROOT, 'shared') + path.sep],
    ['/', path.join(ROOT, 'client') + path.sep],
  ];
  const server = http.createServer((req, res) => {
    let url = decodeURIComponent((req.url || '/').split('?')[0]);
    if (url === '/') url = '/index.html';
    if (url === '/favicon.ico') url = '/favicon.png';
    for (const [prefix, dir] of routes) {
      if (!url.startsWith(prefix)) continue;
      const file = path.normalize(path.join(dir, url.slice(prefix.length)));
      if (!file.startsWith(dir)) break;
      fs.readFile(file, (err, data) => {
        if (err) { res.writeHead(404); res.end('Not found'); return; }
        res.writeHead(200, { 'Content-Type': MIME[path.extname(file)] || 'application/octet-stream', 'Cache-Control': 'no-cache' });
        res.end(data);
      });
      return;
    }
    res.writeHead(404); res.end('Not found');
  });

  const game = new Game(saveDir);
  // підхоплення гри після виходу попереднього хоста — продовжуємо збереження без лобі
  if (autoContinue && game.saveInfo()) game.start({ mode: 'continue' });
  const wss = new WebSocketServer({ server, path: '/ws' });
  wss.on('connection', (ws) => {
    game.onConnect(ws);
    ws.on('message', (data) => game.onMessage(ws, data.toString()));
    ws.on('close', () => game.onClose(ws));
    ws.on('error', () => {});
  });

  let last = performance.now(), acc = 0;
  const loop = setInterval(() => {
    const now = performance.now();
    acc += Math.min(0.25, (now - last) / 1000);
    last = now;
    while (acc >= TICK) {
      acc -= TICK;
      try { game.tick(); } catch (e) { console.error('tick error', e); }
    }
  }, 10);

  return new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(port, host, () => {
      resolve({
        game,
        close() { clearInterval(loop); game.save(); wss.close(); server.close(); },
      });
    });
  });
}
