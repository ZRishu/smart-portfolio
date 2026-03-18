import { defineConfig } from "astro/config";

import tailwindcss from "@tailwindcss/vite";

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

    plugins: [tailwindcss()],
  },
});