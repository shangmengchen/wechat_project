import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'node:path'

export default defineConfig({
  plugins: [vue()],
  base: '/admin/',
  build: {
    outDir: resolve(__dirname, '../web/admin'),
    emptyOutDir: true,
    assetsDir: 'assets'
  }
})
