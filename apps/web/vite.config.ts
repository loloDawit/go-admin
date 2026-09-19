import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    // Same-origin to the gateway in dev: the session cookie needs no CORS handling this way.
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
