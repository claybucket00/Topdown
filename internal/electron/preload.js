const { contextBridge, ipcRenderer } = require('electron');
const path = require('path');

contextBridge.exposeInMainWorld('electronAPI', {
  selectFile: async () => {
    const result = await ipcRenderer.invoke('select-demo-file');
    if (result.canceled) return null;
    return result.filePaths?.[0] || null;
  },
  readFile: async (filePath) => {
    const buffer = await ipcRenderer.invoke('read-file', filePath);
    return new File([buffer], path.basename(filePath), { type: 'application/octet-stream' });
  }
});
