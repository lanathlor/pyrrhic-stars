/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [tailwindcss(), react()],
  test: {
    environment: 'happy-dom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.test.{ts,tsx}'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html', 'lcov'],
      include: ['src/**/*.{ts,tsx}'],
      exclude: ['src/test/**', 'src/**/*.test.{ts,tsx}', 'src/routeTree.gen.ts', 'src/main.tsx', 'src/routes.tsx'],
    },
  },
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
