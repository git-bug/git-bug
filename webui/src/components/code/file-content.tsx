// File content view: renders Markdown files with a Preview/Code toggle and
// falls back to the syntax-highlighted source viewer for everything else.
// "Code" mode reuses FileViewer so line linking and the raw source stay
// available.

import { Code2, Eye } from "lucide-react";
import { useState, type ReactNode } from "react";

import { useFragment, type FragmentType } from "@/__generated__/fragment-masking";
import { FileViewer, FILE_VIEWER_BLOB_FRAGMENT } from "@/components/code/file-viewer";
import { Markdown } from "@/components/content/markdown";
import { dirname } from "@/lib/path";
import { cn } from "@/lib/utils";

function isMarkdownPath(path: string): boolean {
  return /\.(?:md|markdown)$/i.test(path);
}

interface FileContentProps {
  blob: FragmentType<typeof FILE_VIEWER_BLOB_FRAGMENT>;
  /** Repo URL slug ("_" for the default repository). */
  repo: string;
  /** Git ref (branch, tag, or hash) the file is viewed at. */
  gitRef: string;
}

export function FileContent({ blob: blobProp, repo, gitRef }: FileContentProps) {
  const blob = useFragment(FILE_VIEWER_BLOB_FRAGMENT, blobProp);
  const [plain, setPlain] = useState(false);

  const text = blob.text;
  // Non-markdown (or binary) files have no rendered view — show source.
  if (blob.isBinary || text == null || !isMarkdownPath(blob.path)) {
    return <FileViewer blob={blobProp} />;
  }

  return (
    <div>
      <div className="mb-3 flex justify-end">
        <div className="bg-muted/40 inline-flex items-center gap-0.5 rounded-md border p-0.5">
          <ToggleButton active={!plain} onClick={() => setPlain(false)}>
            <Eye className="size-3.5" />
            Preview
          </ToggleButton>
          <ToggleButton active={plain} onClick={() => setPlain(true)}>
            <Code2 className="size-3.5" />
            Code
          </ToggleButton>
        </div>
      </div>

      {plain ? (
        <FileViewer blob={blobProp} />
      ) : (
        <div className="rounded-md border">
          <div className="px-6 py-4">
            <Markdown
              content={text}
              size="document"
              repoContext={{ repo, ref: gitRef, basePath: dirname(blob.path) }}
            />
          </div>
        </div>
      )}
    </div>
  );
}

interface ToggleButtonProps {
  active: boolean;
  onClick: () => void;
  children: ReactNode;
}

function ToggleButton({ active, onClick, children }: ToggleButtonProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={cn(
        "flex items-center gap-1.5 rounded-[min(var(--radius-md),8px)] px-2.5 py-1 text-sm font-medium transition-colors",
        active
          ? "bg-background text-foreground shadow-xs"
          : "text-muted-foreground hover:text-foreground",
      )}
    >
      {children}
    </button>
  );
}
