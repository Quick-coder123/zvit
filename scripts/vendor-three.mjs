// Копіює модулі Three.js (examples/jsm) у client/vendor/jsm:
// electron-builder завжди викидає папки "examples" з node_modules.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const src = path.join(root, 'node_modules', 'three', 'examples', 'jsm');
const dst = path.join(root, 'client', 'vendor', 'jsm');
fs.rmSync(dst, { recursive: true, force: true });
fs.cpSync(src, dst, { recursive: true });
console.log('three/examples/jsm →', dst);
