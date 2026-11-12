// Запуск сервера без Electron: node server/standalone.js [порт]
import path from 'node:path';
import os from 'node:os';
import { startServer } from './index.js';
import { PORT } from '../shared/data.js';

const port = +process.argv[2] || PORT;
const saveDir = process.env.BASTION_SAVE_DIR || path.join(process.env.PROGRAMDATA || os.homedir(), 'NeonBastion');
const srv = await startServer({ port, saveDir });
console.log(`Неоновий Бастіон: сервер на http://localhost:${port}  (збереження: ${saveDir})`);
for (const sig of ['SIGINT', 'SIGTERM']) process.on(sig, () => { srv.close(); process.exit(0); });
