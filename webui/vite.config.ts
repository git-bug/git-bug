import path from "node:path";
import { fileURLToPath } from "node:url";

import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import compression from "vite-plugin-compression";
import svgr from "vite-plugin-svgr";

const dirname = path.dirname(fileURLToPath(import.meta.url));

// The Go backend URL. Run: git-bug webui --port 3000
const API_URL = process.env.VITE_API_URL || "http://localhost:3000";

export default defineConfig({
  plugins: [
    tanstackRouter({ target: "react", autoCodeSplitting: true }),
    svgr(),
    tailwindcss(),
    react(),
    compression({
      algorithm: "gzip",
      ext: ".gz",
      filter: /\.(js|css|html|json|svg)$/i,
      deleteOriginFile: true,
    }),
  ],
  resolve: {
    alias: {
      "@": path.resolve(dirname, "src"),
    },
  },
  server: {
    proxy: {
      "/graphql": { target: API_URL, changeOrigin: true, ws: true },
      "/gitfile": { target: API_URL, changeOrigin: true },
      "/upload": { target: API_URL, changeOrigin: true },
    },
  },
});
