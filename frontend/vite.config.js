import { defineConfig } from "vite";

export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: 18087,
    strictPort: true
  },
  build: {
    outDir: "dist",
    emptyOutDir: true
  }
});
