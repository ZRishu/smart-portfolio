import { defineConfig } from "astro/config";
import { loadEnv } from "vite";
import tailwindcss from "@tailwindcss/vite";

const env = loadEnv("", new URL(".", import.meta.url).pathname, "");
const frontendPort = Number(env.FRONTEND_PORT ?? env.PORT ?? 5173);
const apiProxyTarget =
  env.PUBLIC_DEV_API_PROXY ??
  env.BACKEND_URL ??
  "http://localhost:8080";

export default defineConfig({
  server: {
    host: true,
    port: frontendPort,
  },
  vite: {
    server: {
      proxy: {
        "/api": {
          target: apiProxyTarget,
          changeOrigin: true,
        },
      },
    },
    plugins: [tailwindcss()],
  },
});
