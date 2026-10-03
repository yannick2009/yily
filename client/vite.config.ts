import { defineConfig } from "vite";
import solid from "vite-plugin-solid";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [solid(), tailwindcss()],
  server: {
    port: 14615,
    // Fail instead of silently switching to another port if 14615 is busy,
    // so the documented URL is always the right one.
    strictPort: true,
    proxy: {
      // Forward API calls to the Go server. From the browser's point of view,
      // the client and the API share the same origin: no CORS configuration
      // is needed and cookies (session, CSRF) work transparently.
      // 127.0.0.1 is used instead of "localhost" to avoid Node IPv6 (::1) resolution issues.
      "/api": "http://127.0.0.1:16529",
    },
  },
});