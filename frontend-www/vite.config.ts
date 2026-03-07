import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tsconfigPaths from "vite-tsconfig-paths";

// https://vite.dev/config/
export default defineConfig({
  server: {
    port: 9334,
    host: true,
    proxy: {
      '/api': {
        target: 'http://localhost:9333',
        changeOrigin: true,
      }
    }
  },
  build: {
    sourcemap: 'hidden',
  },
  define: {
    global: 'window',
  },
  plugins: [
    react({
      babel: {
        plugins: [
          'react-dev-locator',
        ],
      },
    }),
    tsconfigPaths()
  ],
})
