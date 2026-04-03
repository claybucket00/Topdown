const { app, BrowserWindow, dialog, ipcMain } = require('electron/main');
const { join } = require('path');

const createWindow = () => {
  const win = new BrowserWindow({
    width: 1680,
    height: 1050,
    webPreferences: {
      preload: join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
    },
  });

  // win.loadFile('./internal/renderer/replay.html')
  win.loadFile('./internal/ui/landing.html');
};

ipcMain.handle('select-demo-file', async () => {
  return dialog.showOpenDialog({
    title: 'Select .dem file',
    properties: ['openFile'],
    filters: [{ name: 'Demo files', extensions: ['dem'] }],
  });
});

app.whenReady().then(() => {
  createWindow()

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow()
    }
  })
})

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit()
  }
})