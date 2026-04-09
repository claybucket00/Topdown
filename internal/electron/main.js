const { app, BrowserWindow, dialog, ipcMain } = require('electron/main');
const { join } = require('path');
const fs = require('fs');

const createWindow = () => {
  const win = new BrowserWindow({
    width: 1680,
    height: 1050,
    webPreferences: {
      sandbox: false,
      preload: join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
    },
  });

  if (process.env.NODE_ENV === 'development') {
    win.loadURL('http://localhost:5173')
  } else {
    win.loadFile('./dist/index.html')
  }
};

ipcMain.handle('select-demo-file', async () => {
  return dialog.showOpenDialog({
    title: 'Select .dem file',
    properties: ['openFile'],
    filters: [{ name: 'Demo files', extensions: ['dem'] }],
  });
});

ipcMain.handle('read-file', async (event, filePath) => {
  return fs.readFileSync(filePath);
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