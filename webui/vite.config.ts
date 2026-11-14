import { readdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { constants, gzipSync } from "node:zlib";

import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import react from "@vitejs/plugin-react";
import { defineConfig, type Plugin } from "vite";

const dirname = path.dirname(fileURLToPath(import.meta.url));

// The Go backend URL. Run: git-bug webui --port 3000
const API_URL = process.env.VITE_API_URL || "http://localhost:3000";

// Replaces each text file of the build output (js, css, html, json, svg) by a
// gzipped copy with a .gz extension. The Go server (//webui:handler.go) sends
// the .gz as is to clients that accept gzip and decompresses it for the
// others. Files under 1025 bytes gain little from compression and stay as is.
function gzipOutput(): Plugin {
  let outDir = "";
  return {
    name: "gzip-output",
    apply: "build",
    enforce: "post",
    configResolved(config) {
      outDir = path.resolve(config.root, config.build.outDir);
    },
    closeBundle() {
      for (const entry of readdirSync(outDir, { recursive: true, withFileTypes: true })) {
        if (!entry.isFile() || !/\.(js|css|html|json|svg)$/i.test(entry.name)) continue;
        const file = path.join(entry.parentPath, entry.name);
        const content = readFileSync(file);
        if (content.length < 1025) continue;
        const gzipped = gzipSync(content, { level: constants.Z_BEST_COMPRESSION });
        writeFileSync(`${file}.gz`, gzipped);
        rmSync(file);
      }
    },
  };
}

export default defineConfig({
  plugins: [
    tanstackRouter({ target: "react", autoCodeSplitting: true }),
    tailwindcss(),
    react(),
    gzipOutput(),
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
