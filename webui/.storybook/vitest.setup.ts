import "@testing-library/jest-dom/vitest";
import { setProjectAnnotations } from "@storybook/react-vite";
import { cleanup } from "@testing-library/react";
import { afterEach, beforeAll, vi } from "vitest";

import { getHighlighter } from "@/lib/shiki";

import * as previewAnnotations from "./preview";

// RTL does not auto-cleanup without globals:true — register it explicitly.
afterEach(cleanup);

// Drafts and other browser storage must not leak from one test into the next.
afterEach(() => localStorage.clear());

// Apply Storybook decorators/parameters from preview.ts to portable stories
// used by the snapshot test project.
const annotations = setProjectAnnotations([previewAnnotations]);

beforeAll(annotations.beforeAll);

// Shiki stops tokenizing a line after 500ms, per theme. When one theme's pass
// stops early, the themes' tokens no longer line up and colors land on the
// wrong spans. On a busy test machine the first highlight (which compiles the
// regexes) can take that long, which made the snapshots flaky. Lift the limit
// in tests only.
beforeAll(async () => {
  const highlighter = await getHighlighter();
  const codeToHast = highlighter.codeToHast;
  highlighter.codeToHast = (code, options) =>
    codeToHast(code, { tokenizeTimeLimit: 0, ...options });
});

// Make useSuspenseFragment a passthrough in happy-dom snapshot tests.
// The real hook reads from the Apollo cache, which requires data to be
// written via a query first. In stories we pass mock objects directly,
// so we just return the `from` data as-is.
vi.mock("@apollo/client/react", async (importOriginal) => {
  const mod = await importOriginal<typeof import("@apollo/client/react")>();
  return {
    ...mod,
    useSuspenseFragment: ({ from }: { from: unknown }) => ({ data: from }),
  };
});
