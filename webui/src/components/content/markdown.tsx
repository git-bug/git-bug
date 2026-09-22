import rehypeShikiFromHighlighter from "@shikijs/rehype/core";
import { Link } from "@tanstack/react-router";
import { useMemo, useState, useEffect } from "react";
import ReactMarkdown, { type ExtraProps } from "react-markdown";
import rehypeAutolinkHeadings from "rehype-autolink-headings";
import rehypeExternalLinks from "rehype-external-links";
import rehypeRaw from "rehype-raw";
import rehypeSanitize, { defaultSchema } from "rehype-sanitize";
import rehypeSlug from "rehype-slug";
import remarkEmoji from "remark-emoji";
import remarkGfm from "remark-gfm";
import type { HighlighterCore } from "shiki/core";
import type { PluggableList } from "unified";

import { getHighlighter, SHIKI_THEMES } from "@/lib/shiki";
import { cn } from "@/lib/utils";

// Sanitization schema: start from the safe default and allow a small set of
// presentational/structural HTML tags commonly found in READMEs.
const sanitizeSchema = {
  ...defaultSchema,
  tagNames: [...(defaultSchema.tagNames ?? []), "details", "summary", "picture", "source"],
  attributes: {
    ...defaultSchema.attributes,
    a: [...(defaultSchema.attributes?.a ?? []), "aria-hidden", "class"],
    // Allow Shiki's style attribute (CSS variables for theme colors) and class on code/spans
    pre: [...(defaultSchema.attributes?.pre ?? []), "class", "style", "tabIndex"],
    code: [...(defaultSchema.attributes?.code ?? []), "class", "style"],
    span: [...(defaultSchema.attributes?.span ?? []), "class", "style"],
    "*": [...(defaultSchema.attributes?.["*"] ?? []), "id"],
  },
};

export interface RepoContext {
  repo: string;
  ref: string;
  /** Directory containing the markdown file (e.g. "doc" for doc/README.md). */
  basePath: string;
}

interface MarkdownProps {
  content: string;
  className?: string;
  /** When set, relative links/images are resolved against the repo. */
  repoContext?: RepoContext;
  /**
   * "comment" (the default) sizes the body to the 14px chrome it sits in —
   * timeline comments, the comment box preview, the new-issue preview.
   * "document" is the larger 16px metric, for a README or a .md file rendered
   * as the whole page.
   */
  size?: "comment" | "document";
}

function isRelativeUrl(url: string): boolean {
  return !/^(?:[a-z][a-z0-9+.-]*:|\/\/|#|data:)/i.test(url);
}

function resolveRelativePath(basePath: string, relativePath: string): string {
  const parts = basePath ? basePath.split("/") : [];
  for (const segment of relativePath.split("/")) {
    if (segment === "..") {
      parts.pop();
    } else if (segment !== "." && segment !== "") {
      parts.push(segment);
    }
  }
  return parts.join("/");
}

const IMAGE_EXTENSIONS = new Set([
  "png",
  "jpg",
  "jpeg",
  "gif",
  "svg",
  "webp",
  "avif",
  "ico",
  "bmp",
]);

function isImagePath(path: string): boolean {
  const ext = path.split(".").pop()?.toLowerCase() ?? "";
  return IMAGE_EXTENSIONS.has(ext);
}

// Renders a Markdown string with GitHub-flavoured extensions (tables, task
// lists, strikethrough). Used in Timeline comments and code browser READMEs.
function useShikiHighlighter(): HighlighterCore | null {
  const [highlighter, setHighlighter] = useState<HighlighterCore | null>(null);
  useEffect(() => {
    let cancelled = false;
    void getHighlighter().then((h) => {
      if (!cancelled) setHighlighter(h);
    });
    return () => {
      cancelled = true;
    };
  }, []);
  return highlighter;
}

// `node` is react-markdown's hast node.  It has to be dropped rather than
// spread onto an element, or it renders as node="[object Object]".
type ImgProps = React.ImgHTMLAttributes<HTMLImageElement> & ExtraProps;
type AnchorProps = React.AnchorHTMLAttributes<HTMLAnchorElement> & ExtraProps;
type PreProps = React.ComponentProps<"pre"> & ExtraProps;

// A horizontally scrollable region must be keyboard focusable (axe rule
// scrollable-region-focusable).  Shiki sets tabindex="0" on the blocks it
// highlights, but code blocks without a language keep their plain markup, so
// they get it here.
//
// tabIndex comes after the spread so it wins: sanitizeSchema lets a raw <pre>
// in the document carry its own tabindex, and a tabindex="-1" would otherwise
// leave the block unreachable by keyboard — the very thing this is fixing.
function PreBlock({ node: _node, ...props }: PreProps) {
  return <pre {...props} tabIndex={0} />;
}

export function Markdown({ content, className, repoContext, size = "comment" }: MarkdownProps) {
  const highlighter = useShikiHighlighter();

  // Rewrite image src to /gitfile for raw content serving.
  // Links are handled by the custom `a` component below.
  const urlTransform = useMemo(() => {
    if (!repoContext) return undefined;
    const { repo, ref, basePath } = repoContext;
    return (url: string) => {
      if (!isRelativeUrl(url)) return url;
      const resolved = resolveRelativePath(basePath, url);
      if (isImagePath(resolved)) {
        return `/gitfile/${repo}/${ref}/${resolved}`;
      }
      // Non-image relative URLs are handled by the `a` component override,
      // but urlTransform runs first, so we still need to return something.
      // Return the resolved path prefixed so the `a` component can detect it.
      return `/${repo}/blob/${ref}/${resolved}`;
    };
  }, [repoContext]);

  const components = useMemo(() => {
    if (!repoContext) return undefined;
    const { repo, ref, basePath } = repoContext;
    const gitfilePrefix = `/gitfile/${repo}/${ref}/`;
    return {
      // `node` is dropped here for the same reason as in PreBlock above: spread
      // onto an element it renders as node="[object Object]".
      img: ({ node: _node, src, alt, ...props }: ImgProps) => {
        // Wrap repo-local images in a Link to the blob view
        if (src?.startsWith(gitfilePrefix)) {
          const path = src.slice(gitfilePrefix.length);
          return (
            <Link to="/$repo/blob/$ref/$" params={{ repo, ref, _splat: path }}>
              <img src={src} alt={alt} {...props} />
            </Link>
          );
        }
        return <img src={src} alt={alt} {...props} />;
      },
      a: ({ node: _node, href, children, ...props }: AnchorProps) => {
        if (!href) return <a {...props}>{children}</a>;

        // Anchor links stay as-is
        if (href.startsWith("#"))
          return (
            <a href={href} {...props}>
              {children}
            </a>
          );

        // Check if this is a relative URL that we should route client-side.
        // After urlTransform, repo-local links look like /{repo}/blob/{ref}/{path}
        const prefix = `/${repo}/blob/${ref}/`;
        if (href.startsWith(prefix)) {
          const path = href.slice(prefix.length);
          return (
            <Link to="/$repo/blob/$ref/$" params={{ repo, ref, _splat: path }} {...props}>
              {children}
            </Link>
          );
        }

        // Also handle raw relative URLs that urlTransform didn't process
        // (shouldn't happen but defensive)
        if (isRelativeUrl(href)) {
          const resolved = resolveRelativePath(basePath, href);
          return (
            <Link to="/$repo/blob/$ref/$" params={{ repo, ref, _splat: resolved }} {...props}>
              {children}
            </Link>
          );
        }

        // External links — render as normal anchor
        return (
          <a href={href} {...props}>
            {children}
          </a>
        );
      },
    };
  }, [repoContext]);

  return (
    <div
      className={cn(
        // No `prose-sm` / `dark:prose-invert`: leading and every --tw-prose-*
        // colour are bound to the theme tokens in index.css, which also holds
        // the default body size.
        "prose max-w-none",
        size === "document" && "[--prose-font-size:1rem]",
        // Code blocks: border, rounded corners, fallback bg for non-highlighted blocks.
        // Shiki adds .shiki class which overrides the background via CSS in index.css.
        "prose-pre:rounded-md prose-pre:border prose-pre:border-border prose-pre:bg-muted prose-pre:text-foreground prose-pre:text-sm prose-pre:overflow-x-auto",
        // Inline code: muted background pill
        "prose-code:bg-muted prose-code:px-1 prose-code:py-0.5 prose-code:rounded-sm prose-code:text-sm prose-code:before:content-none prose-code:after:content-none",
        // Reset inline code styles inside highlighted code blocks
        "prose-pre:prose-code:bg-transparent prose-pre:prose-code:p-0",
        "prose-img:inline prose-img:my-0",
        className,
      )}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkEmoji]}
        rehypePlugins={[
          rehypeRaw,
          [rehypeSanitize, sanitizeSchema],
          ...(highlighter
            ? [
                [
                  rehypeShikiFromHighlighter,
                  highlighter,
                  { themes: SHIKI_THEMES, defaultColor: false },
                ] as PluggableList[number],
              ]
            : []),
          rehypeSlug,
          [rehypeAutolinkHeadings, { behavior: "append" }],
          [rehypeExternalLinks, { target: "_blank", rel: ["noopener", "noreferrer"] }],
        ]}
        urlTransform={urlTransform}
        components={{ pre: PreBlock, ...components }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}
