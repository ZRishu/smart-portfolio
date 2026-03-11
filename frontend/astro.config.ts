import { defineConfig } from "astro/config";

export default defineConfig({
  server: {
    port: 5173,
  },
  vite: {
    server: {
      proxy: {
        "/api": {
          target: "http://localhost:8080",
          changeOrigin: true,
        },
      },
    },
  },
});
