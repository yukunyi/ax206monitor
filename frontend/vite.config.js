import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

const apiTarget = process.env.AX206_MONITOR_API_URL || "http://127.0.0.1:18086";

export default defineConfig({
  plugins: [vue()],
  server: {
    host: "127.0.0.1",
    port: 18087,
    strictPort: true,
    proxy: {
      "/api": {
        target: apiTarget,
        changeOrigin: true,
        ws: true,
      },
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
});
