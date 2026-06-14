import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// The build output (web/dist) is embedded into the Go binary via //go:embed.
// During development, `npm run dev` proxies /api to the Go backend on :8088.
export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    chunkSizeWarningLimit: 900,
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8088',
        changeOrigin: true,
      },
    },
  },
})
