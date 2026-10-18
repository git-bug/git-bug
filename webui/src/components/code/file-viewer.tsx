// Syntax-highlighted file viewer with clickable line numbers.
// Uses Shiki codeToHast → hast-util-to-jsx-runtime for native React rendering.
// Line selection syncs with the URL hash (e.g. #L12 or #L12:25).

import { toJsxRuntime } from "hast-util-to-jsx-runtime";
import { Copy } from "lucide-react";
import { useState, useEffect, useCallback, Fragment, type ReactNode } from "react";
import { jsx, jsxs } from "react/jsx-runtime";
import type { LanguageInput, ShikiTransformer } from "shiki/core";

import { useFragment, type FragmentType } from "@/__generated__/fragment-masking";
import { graphql } from "@/__generated__/gql";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { getHighlighter, SHIKI_THEMES } from "@/lib/shiki";

import styles from "./file-viewer.module.css";

export const FILE_VIEWER_BLOB_FRAGMENT = graphql(`
  fragment FileViewerBlob on GitBlob {
    path
    hash
    text
    size
    isBinary
    isTruncated
  }
`);

interface LangEntry {
  id: string;
  load: LanguageInput;
}

const LANG_MAP: Record<string, LangEntry> = {
  js: { id: "javascript", load: () => import("@shikijs/langs/javascript") },
  mjs: { id: "javascript", load: () => import("@shikijs/langs/javascript") },
  cjs: { id: "javascript", load: () => import("@shikijs/langs/javascript") },
  jsx: { id: "jsx", load: () => import("@shikijs/langs/jsx") },
  ts: { id: "typescript", load: () => import("@shikijs/langs/typescript") },
  mts: { id: "typescript", load: () => import("@shikijs/langs/typescript") },
  cts: { id: "typescript", load: () => import("@shikijs/langs/typescript") },
  tsx: { id: "tsx", load: () => import("@shikijs/langs/tsx") },
  html: { id: "html", load: () => import("@shikijs/langs/html") },
  css: { id: "css", load: () => import("@shikijs/langs/css") },
  scss: { id: "scss", load: () => import("@shikijs/langs/scss") },
  json: { id: "json", load: () => import("@shikijs/langs/json") },
  jsonc: { id: "jsonc", load: () => import("@shikijs/langs/jsonc") },
  yaml: { id: "yaml", load: () => import("@shikijs/langs/yaml") },
  yml: { id: "yaml", load: () => import("@shikijs/langs/yaml") },
  toml: { id: "toml", load: () => import("@shikijs/langs/toml") },
  xml: { id: "xml", load: () => import("@shikijs/langs/xml") },
  svg: { id: "xml", load: () => import("@shikijs/langs/xml") },
  graphql: { id: "graphql", load: () => import("@shikijs/langs/graphql") },
  sql: { id: "sql", load: () => import("@shikijs/langs/sql") },
  md: { id: "markdown", load: () => import("@shikijs/langs/markdown") },
  mdx: { id: "mdx", load: () => import("@shikijs/langs/mdx") },
  sh: { id: "bash", load: () => import("@shikijs/langs/bash") },
  bash: { id: "bash", load: () => import("@shikijs/langs/bash") },
  zsh: { id: "bash", load: () => import("@shikijs/langs/bash") },
  go: { id: "go", load: () => import("@shikijs/langs/go") },
  rs: { id: "rust", load: () => import("@shikijs/langs/rust") },
  c: { id: "c", load: () => import("@shikijs/langs/c") },
  h: { id: "c", load: () => import("@shikijs/langs/c") },
  cpp: { id: "cpp", load: () => import("@shikijs/langs/cpp") },
  hpp: { id: "cpp", load: () => import("@shikijs/langs/cpp") },
  py: { id: "python", load: () => import("@shikijs/langs/python") },
  rb: { id: "ruby", load: () => import("@shikijs/langs/ruby") },
  lua: { id: "lua", load: () => import("@shikijs/langs/lua") },
  java: { id: "java", load: () => import("@shikijs/langs/java") },
  kt: { id: "kotlin", load: () => import("@shikijs/langs/kotlin") },
  swift: { id: "swift", load: () => import("@shikijs/langs/swift") },
  nix: { id: "nix", load: () => import("@shikijs/langs/nix") },
  Dockerfile: { id: "dockerfile", load: () => import("@shikijs/langs/dockerfile") },
  Makefile: { id: "makefile", load: () => import("@shikijs/langs/makefile") },
};

function getLangEntry(path: string): LangEntry | undefined {
  const filename = path.split("/").pop() ?? "";
  const ext = filename.split(".").pop() ?? "";
  return LANG_MAP[ext] ?? LANG_MAP[filename];
}

// ── Shiki transformer: inject line number data attributes ─────────────────────

// Adds data-line-number to each .line span so we can render line numbers
// and handle selection purely from the hast tree — no separate gutter needed.
function lineNumberTransformer(): ShikiTransformer {
  return {
    line(node, line) {
      // Replace Shiki's "line" class with our CSS module class
      node.properties["className"] = [styles["line"]!];
      // Prepend a gutter <span> for the line number — clickable target
      node.children.unshift({
        type: "element",
        tagName: "span",
        properties: {
          className: [styles["line-number"]!],
          dataLineNumber: line,
        },
        children: [{ type: "text", value: String(line) }],
      });
      // Append a \n text node so copy-paste preserves newlines
      // (we strip the inter-element whitespace nodes in code() below).
      node.children.push({ type: "text", value: "\n" });
    },
    // Remove whitespace text nodes between .line spans — they create
    // empty anonymous rows when using display: block.
    code(node) {
      node.children = node.children.filter((c) => !(c.type === "text" && c.value.trim() === ""));
    },
  };
}

// ── Line selection from URL hash ──────────────────────────────────────────────

interface LineRange {
  start: number;
  end: number;
}

function parseHash(hash: string): LineRange | null {
  const match = /^#?L(\d+)(?::(\d+))?$/.exec(hash);
  if (!match) return null;
  const start = parseInt(match[1]!, 10);
  const end = match[2] ? parseInt(match[2], 10) : start;
  return { start: Math.min(start, end), end: Math.max(start, end) };
}

function buildHash(range: LineRange): string {
  return range.start === range.end ? `#L${range.start}` : `#L${range.start}:${range.end}`;
}

// ── Component ─────────────────────────────────────────────────────────────────

interface FileViewerProps {
  blob: FragmentType<typeof FILE_VIEWER_BLOB_FRAGMENT>;
}

export function FileViewer({ blob: blobProp }: FileViewerProps) {
  const blob = useFragment(FILE_VIEWER_BLOB_FRAGMENT, blobProp);

  const [highlighted, setHighlighted] = useState<{ node: ReactNode; lineCount: number } | null>(
    null,
  );
  const [selectedRange, setSelectedRange] = useState<LineRange | null>(() =>
    parseHash(window.location.hash),
  );

  // Sync hash → state on popstate
  useEffect(() => {
    function onHashChange() {
      setSelectedRange(parseHash(window.location.hash));
    }
    window.addEventListener("hashchange", onHashChange);
    return () => window.removeEventListener("hashchange", onHashChange);
  }, []);

  // Scroll to selected line on initial load
  useEffect(() => {
    if (selectedRange && highlighted) {
      const el = document.getElementById(`L${selectedRange.start}`);
      el?.scrollIntoView({ block: "center" });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- only scroll on first render
  }, [highlighted]);

  const handleLineClick = useCallback(
    (lineNumber: number, shiftKey: boolean) => {
      let newRange: LineRange;
      if (shiftKey && selectedRange) {
        const anchor = selectedRange.start;
        newRange = {
          start: Math.min(anchor, lineNumber),
          end: Math.max(anchor, lineNumber),
        };
      } else {
        newRange = { start: lineNumber, end: lineNumber };
      }
      setSelectedRange(newRange);
      window.history.replaceState(null, "", buildHash(newRange));
    },
    [selectedRange],
  );

  useEffect(() => {
    if (blob.isBinary || !blob.text) {
      setHighlighted({ node: null, lineCount: 0 });
      return () => {};
    }
    setHighlighted(null);
    let cancelled = false;

    void (async () => {
      const highlighter = await getHighlighter();
      const entry = getLangEntry(blob.path);

      let lang = "text";
      if (entry) {
        try {
          await highlighter.loadLanguage(entry.load);
          lang = entry.id;
        } catch {
          // Language not available — fall back to plain text
        }
      }

      if (cancelled) return;

      const hast = highlighter.codeToHast(blob.text!, {
        lang,
        themes: SHIKI_THEMES,
        defaultColor: false,
        transformers: [lineNumberTransformer()],
      });

      const node = toJsxRuntime(hast, { Fragment, jsx, jsxs });
      const lineCount = blob.text!.split("\n").length;

      setHighlighted({ node, lineCount });
    })();

    return () => {
      cancelled = true;
    };
  }, [blob]);

  if (highlighted === null) return <FileViewerSkeleton />;
  const { lineCount } = highlighted;

  function copyToClipboard() {
    if (blob?.text) void navigator.clipboard.writeText(blob.text);
  }

  return (
    <div className="border-border overflow-hidden rounded-md border">
      <div className="border-border bg-muted/40 text-muted-foreground flex items-center justify-between border-b px-4 py-2 text-xs">
        <span>
          {lineCount.toLocaleString()} lines · {formatBytes(blob.size)}
          {blob.isTruncated && " · truncated"}
        </span>
        <Button variant="ghost" size="icon-xs" onClick={copyToClipboard} title="Copy">
          <Copy className="size-3.5" />
        </Button>
      </div>

      {blob.isBinary ? (
        <div className="text-muted-foreground px-4 py-8 text-center text-sm">
          Binary file — {formatBytes(blob.size)}
        </div>
      ) : (
        <CodeBlock selectedRange={selectedRange} onLineClick={handleLineClick}>
          {highlighted.node}
        </CodeBlock>
      )}
    </div>
  );
}

// ── Code block with integrated line numbers ───────────────────────────────────

interface CodeBlockProps {
  selectedRange: LineRange | null;
  onLineClick: (line: number, shiftKey: boolean) => void;
  children: ReactNode;
}

function CodeBlock({ selectedRange, onLineClick, children }: CodeBlockProps) {
  // Build a scoped <style> for highlighted lines using nth-child selectors
  // targeting the CSS module's scoped class.
  const highlightStyle = (() => {
    if (!selectedRange) return null;
    const lineClass = styles["line"];
    const selectors: string[] = [];
    for (let i = selectedRange.start; i <= selectedRange.end; i++) {
      selectors.push(`.${lineClass}:nth-child(${i})`);
    }
    const rule = selectors.join(",");
    return (
      <style>{`${rule}{background-color:rgba(255,235,59,0.3)}:root.dark ${rule}{background-color:rgba(255,235,59,0.15)}`}</style>
    );
  })();

  return (
    <div
      className={styles["code-block"]}
      onClick={(e) => {
        if (!(e.target instanceof HTMLElement)) return;
        const lineEl = e.target.closest("[data-line-number]");
        if (lineEl) {
          e.preventDefault();
          const lineNum = parseInt(lineEl.getAttribute("data-line-number")!, 10);
          onLineClick(lineNum, e.shiftKey);
        }
      }}
    >
      {highlightStyle}
      <div className={styles["code-content"]}>{children}</div>
    </div>
  );
}

// ── Utilities ─────────────────────────────────────────────────────────────────

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

function FileViewerSkeleton() {
  return (
    <div className="border-border overflow-hidden rounded-md border">
      <div className="border-border bg-muted/40 border-b px-4 py-2">
        <Skeleton className="h-4 w-32" />
      </div>
      <div className="flex gap-4 p-4">
        <div className="space-y-1.5">
          {Array.from({ length: 20 }).map((_, i) => (
            <Skeleton key={i} className="h-3.5 w-6" />
          ))}
        </div>
        <div className="flex-1 space-y-1.5">
          {Array.from({ length: 20 }).map((_, i) => (
            <Skeleton key={i} className="h-3.5" style={{ width: `${30 + ((i * 47) % 60)}%` }} />
          ))}
        </div>
      </div>
    </div>
  );
}
