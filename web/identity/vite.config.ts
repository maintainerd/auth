import path from "path"
import tailwindcss from "@tailwindcss/vite"
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    /** Tailwind/Shadcn */
    tailwindcss(),
  ],
  /** Tailwind/shadcn */
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  /** Production build configuration */
  build: {
    // Do not ship source maps to production — they expose original source.
    sourcemap: false,
    rollupOptions: {
      output: {
        // Split large third-party dependencies into a dedicated vendor chunk
        // so the app shell and lazily-loaded route chunks stay small.
        manualChunks: {
          'react-vendor': ['react', 'react-dom', 'react-router-dom'],
        },
      },
    },
  },
  /** Development server configuration */
  server: {
    port: 5174,
    strictPort: false,
    allowedHosts: ['.auth.maintainerd.local'],
    watch: {
      ignored: ['**/coverage/**'],
    },
    proxy: {
      // Proxy API requests to the public API via nginx during development.
      // nginx routes public-api.auth.maintainerd.local → auth:8081 (public surface).
      '/api': {
        target: 'https://public-api.auth.maintainerd.local',
        changeOrigin: true,
        secure: false,
        configure: (proxy) => {
          proxy.on('error', (err) => {
            console.log('Proxy error:', err);
          });
          proxy.on('proxyReq', (_proxyReq, req) => {
            console.log('Sending Request to the Target:', req.method, req.url);
          });
          proxy.on('proxyRes', (proxyRes, req) => {
            console.log('Received Response from the Target:', proxyRes.statusCode, req.url);
          });
        },
      }
    }
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    css: false,
    testTimeout: 15000,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'text-summary', 'lcov'],
      include: [
        'src/components/**',
        'src/pages/**',
        'src/hooks/**',
        'src/services/**',
        'src/utils/**',
        'src/lib/**',
        'src/store/**',
      ],
      exclude: [
        '**/*.test.{ts,tsx}',
        '**/*.d.ts',
        'src/test/**',
        'src/components/ui/**',
      ],
    },
  },
})
