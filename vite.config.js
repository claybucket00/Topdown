import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  base: './', // Required for Electron file:// loading in production
  build: {
    outDir: 'dist',
  },
  server : {
    '/base_api': {
      target: 'http://localhost:8080',
      changeOrigin: true,
    }
  }
})
