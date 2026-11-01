import { defineConfig, lazyPlugins } from "vite-plus";
import solid from "vite-plugin-solid";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  fmt: {},
  lint: {
    jsPlugins: [{ name: "vite-plus", specifier: "vite-plus/oxlint-plugin" }],
    rules: { "vite-plus/prefer-vite-plus-imports": "error" },
    options: { typeAware: true, typeCheck: true },
  },
  plugins: lazyPlugins(() => [solid(), tailwindcss()]),
  server: {
    port: 14615,
    proxy: {
      // Forward API calls to the Go server (no CORS middleware on the server).
      "/api": {
        target: "http://localhost:16529",
        changeOrigin: true,
      },
    },
  },
});
