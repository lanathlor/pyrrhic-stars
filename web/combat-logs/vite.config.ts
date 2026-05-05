import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 5180,
    watch: {
      usePolling: true,
    },
    proxy: {
      '/api': {
        target: 'http://gateway:7777',
        changeOrigin: true,
      },
    },
  },
})
