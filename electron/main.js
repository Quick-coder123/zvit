// Electron: перша копія гри на комп'ютері стає сервером, інші підключаються до неї.
import { app, BrowserWindow, ipcMain, Menu } from 'electron';
import path from 'node:path';
import os from 'node:os';
import { fileURLToPath } from 'node:url';
import { startServer } from '../server/index.js';
import { PORT } from '../shared/data.js';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// GPU у RDP-сесіях: не даємо Chromium вимкнути апаратне прискорення
app.commandLine.appendSwitch('ignore-gpu-blocklist');
app.commandLine.appendSwitch('enable-gpu-rasterization');
app.commandLine.appendSwitch('disable-renderer-backgrounding');
app.commandLine.appendSwitch('disable-background-timer-throttling');

const argHost = process.argv.find((a) => a.startsWith('--host='));
const remoteHost = argHost ? argHost.slice(7) : null;
const port = +(process.argv.find((a) => a.startsWith('--port=')) || '').slice(7) || PORT;
const saveDir = path.join(process.env.PROGRAMDATA || app.getPath('userData'), 'NeonBastion');

let srv = null;
let win = null;

async function tryHost(autoContinue = false) {
  if (srv || remoteHost) return !!srv;
  try {
    srv = await startServer({ port, saveDir, autoContinue });
    return true;
  } catch (e) {
    if (e.code !== 'EADDRINUSE') console.error(e);
    return false;
  }
}

function gameUrl() {
  const host = remoteHost || '127.0.0.1';
  let user = 'Гравець';
  try { user = os.userInfo().username; } catch { /* ignore */ }
  return `http://${host}:${port}/?user=${encodeURIComponent(user)}${srv ? '&hosting=1' : ''}`;
}

async function createWindow() {
  await tryHost();
  Menu.setApplicationMenu(null);
  win = new BrowserWindow({
    width: 1600, height: 900, minWidth: 1100, minHeight: 650,
    backgroundColor: '#05070d',
    title: 'Неоновий Бастіон',
    autoHideMenuBar: true,
    webPreferences: { preload: path.join(__dirname, 'preload.cjs'), backgroundThrottling: false },
  });
  win.maximize();
  win.loadURL(gameUrl());
  win.webContents.on('before-input-event', (e, input) => {
    if (input.type === 'keyDown' && input.key === 'F11') { win.setFullScreen(!win.isFullScreen()); e.preventDefault(); }
    if (input.type === 'keyDown' && input.key === 'F12') win.webContents.toggleDevTools();
  });
}

// Якщо хост вийшов — пробуємо підхопити сервер (збереження спільне).
ipcMain.handle('bastion:reconnect', async () => {
  await tryHost(true);
  if (win) win.loadURL(gameUrl());
  return !!srv;
});
ipcMain.handle('bastion:quit', () => app.quit());

app.whenReady().then(createWindow);
app.on('window-all-closed', () => app.quit());
app.on('before-quit', () => { if (srv) { srv.close(); srv = null; } });
