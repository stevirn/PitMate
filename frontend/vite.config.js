import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

// Vite build/dev configuration for the PitMate frontend.
//
// - `npm run build` outputs static files into frontend/dist/, which the Go
//   server serves with its -static flag (one binary serves UI + WebSocket).
// - `npm run dev` runs Vite's dev server with hot reload on port 5173. The /ws
//   proxy below forwards the WebSocket to a backend running on :8080, so the
//   dev UI gets live data without CORS/origin headaches.
export default defineConfig({
  plugins: [svelte()],
  // Relative asset paths so the built app works no matter what path the Go
  // server mounts it at.
  base: './',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      '/ws': { target: 'ws://localhost:8080', ws: true },
    },
  },
});
