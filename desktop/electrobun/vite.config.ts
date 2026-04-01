import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

export default defineConfig({
  plugins: [svelte()],
  build: {
    outDir: "dist/mainview",
    emptyOutDir: true,
    cssCodeSplit: false,
    rollupOptions: {
      input: "src/mainview/index.html",
      output: {
        entryFileNames: "app.js",
        chunkFileNames: "app.js",
        assetFileNames: (assetInfo) => {
          if (assetInfo.name?.endsWith(".css")) return "app.css";
          return "asset.[ext]";
        },
      },
    },
  },
});
