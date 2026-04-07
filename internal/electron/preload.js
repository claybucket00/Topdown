const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('electronAPI', {
  selectFile: async () => {
    const result = await ipcRenderer.invoke('select-demo-file');
    if (result.canceled) return null;
    //return result.filePaths?.[0] || null;
    console.log('File selection result:', result);
    return result.files || null;
  }
});
