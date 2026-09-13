import path from "node:path";
import { fileURLToPath } from "node:url";

import { storybookTest } from "@storybook/addon-vitest/vitest-plugin";
import { playwright } from "@vitest/browser-playwright";
import { defineConfig, mergeConfig } from "vitest/config";

import viteConfig from "./vite.config.ts";

const dirname = path.dirname(fileURLToPath(import.meta.url));

export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      coverage: {
        provider: "v8",
        include: ["src/**/*.{ts,tsx}"],
        exclude: [
          "src/__generated__/**",
          "src/routeTree.gen.ts",
          "src/**/*.stories.{ts,tsx}",
          "src/**/*.test.{ts,tsx}",
        ],
      },
      projects: [
        // Storybook smoke & interaction tests (real browser via Playwright)
        {
          extends: true,
          plugins: [
            storybookTest({
              configDir: path.join(dirname, ".storybook"),
            }),
          ],
          // In CI (cold Vite cache) the dep optimizer must finish before the
          // browser connects.  Vite 8 defaults to holdUntilCrawlEnd:true, which
          // holds every browser request until the crawl is done.  If we only use
          // server.warmup, the crawl is triggered mid-warmup and Playwright's
          // goto(url, { timeout: 0 }) waits forever for the page that is held.
          //
          // optimizeDeps.entries makes Vite run the optimizer eagerly at startup
          // (before the server accepts any request), so by the time the browser
          // connects all story deps are already pre-bundled.  The warmup then
          // runs in the background without triggering a new optimizer cycle.
          optimizeDeps: {
            entries: ["./src/**/*.stories.tsx"],
            // Mid-run, a stray request for /index.html hits Vite's SPA
            // fallback and pulls the real app graph (main.tsx -> App ->
            // routeTree.gen -> all routes) into the test server.  That graph
            // imports the bare "@apollo/client" root (src/lib/apollo.ts) and
            // "valibot" (route search schemas), which are unreachable from
            // the stories entries above.  On a cold cache Vite then
            // re-optimizes and reloads the browser mid-run, which vitest
            // does not recover from (tests hang until the CI timeout).
            // Pre-bundling them up front makes the discovery a no-op.
            include: ["@apollo/client", "valibot"],
          },
          server: {
            warmup: {
              clientFiles: ["./src/**/*.stories.tsx"],
            },
          },
          test: {
            name: "storybook",
            browser: {
              enabled: true,
              provider: playwright({}),
              headless: true,
              instances: [{ browser: "chromium" }],
            },
            // Shiki's WASM engine fails in Vitest browser mode
            exclude: ["src/components/code/file-viewer.stories.tsx"],
          },
        },
        // Snapshot tests (happy-dom, fast)
        {
          extends: true,
          test: {
            name: "snapshot",
            include: ["src/**/*.test.{ts,tsx}"],
            environment: "happy-dom",
            setupFiles: ["./.storybook/vitest.setup.ts"],
          },
        },
      ],
    },
  }),
);
