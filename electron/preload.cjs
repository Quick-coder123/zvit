const { contextBridge, ipcRenderer } = require('electron');
contextBridge.exposeInMainWorld('bastion', {
  reconnect: () => ipcRenderer.invoke('bastion:reconnect'),
  quit: () => ipcRenderer.invoke('bastion:quit'),
});
