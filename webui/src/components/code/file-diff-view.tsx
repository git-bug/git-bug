// Collapsible diff view for a single file in a commit.
// Diff is fetched lazily on first expand via GraphQL.

import { useLazyQuery } from "@apollo/client/react";
import { ChevronRight, FilePlus, FileMinus, FileEdit } from "lucide-react";
import { useState } from "react";

import { graphql } from "@/__generated__/gql";
import { cn } from "@/lib/utils";

const DIFF_QUERY = graphql(`
  query FileDiff($repo: String, $hash: String!, $path: String!) {
    repository(ref: $repo) {
      name
      commit(hash: $hash) {
        diff(path: $path) {
          path
          oldPath
          isBinary
          isNew
          isDelete
          hunks {
            oldStart
            oldLines
            newStart
            newLines
            lines {
              type
              content
              oldLine
              newLine
            }
          }
        }
      }
    }
  }
`);

interface FileDiffViewProps {
  repo: string | null;
  hash: string;
  path: string;
  oldPath?: string;
  status: string;
}

const statusIcon: Record<string, React.ReactNode> = {
  ADDED: <FilePlus className="size-3.5 text-green-600 dark:text-green-400" />,
  DELETED: <FileMinus className="size-3.5 text-red-500 dark:text-red-400" />,
  MODIFIED: <FileEdit className="size-3.5 text-yellow-500 dark:text-yellow-400" />,
  RENAMED: <FileEdit className="size-3.5 text-blue-500 dark:text-blue-400" />,
};
const statusBadge: Record<string, string> = {
  ADDED: "A",
  DELETED: "D",
  MODIFIED: "M",
  RENAMED: "R",
};

export function FileDiffView({ repo, hash, path, oldPath, status }: FileDiffViewProps) {
  const [open, setOpen] = useState(false);
  const [fetchDiff, { data, loading, error }] = useLazyQuery(DIFF_QUERY);

  function toggle() {
    if (!open && !data && !loading) {
      void fetchDiff({ variables: { repo, hash, path } });
    }
    setOpen((v) => !v);
  }

  const diff = data?.repository?.commit?.diff;

  return (
    <div className="divide-border divide-y">
      <button
        onClick={toggle}
        className="hover:bg-muted/50 flex w-full items-center gap-3 px-4 py-2.5 text-left transition-colors"
      >
        <ChevronRight
          className={cn(
            "size-3.5 shrink-0 text-muted-foreground transition-transform duration-150",
            open && "rotate-90",
          )}
        />
        {statusIcon[status] ?? <FileEdit className="text-muted-foreground size-3.5" />}
        <span className="min-w-0 flex-1 font-mono text-sm">
          {status === "RENAMED" ? (
            <>
              <span className="text-muted-foreground line-through">{oldPath}</span>
              {" → "}
              <span>{path}</span>
            </>
          ) : (
            path
          )}
        </span>
        <span className="border-border text-muted-foreground shrink-0 rounded-sm border px-1.5 py-0.5 font-mono text-xs">
          {statusBadge[status] ?? "?"}
        </span>
      </button>

      {open && (
        <div className="overflow-x-auto">
          {loading && <div className="text-muted-foreground px-4 py-3 text-xs">Loading diff…</div>}
          {error && (
            <div className="text-destructive px-4 py-3 text-xs">
              Failed to load diff: {error.message}
            </div>
          )}
          {diff &&
            (diff.isBinary ? (
              <div className="text-muted-foreground px-4 py-3 text-xs">Binary file</div>
            ) : diff.hunks.length === 0 ? (
              <div className="text-muted-foreground px-4 py-3 text-xs">No changes</div>
            ) : (
              diff.hunks.map((hunk: HunkType, i: number) => <Hunk key={i} hunk={hunk} />)
            ))}
        </div>
      )}
    </div>
  );
}

type LineType = { type: string; content: string; oldLine: number; newLine: number };
type HunkType = {
  oldStart: number;
  oldLines: number;
  newStart: number;
  newLines: number;
  lines: LineType[];
};

function Hunk({ hunk }: { hunk: HunkType }) {
  return (
    <div className="font-mono text-xs leading-5">
      <div className="bg-blue-50 px-4 py-0.5 text-blue-600 select-none dark:bg-blue-950/40 dark:text-blue-400">
        @@ -{hunk.oldStart},{hunk.oldLines} +{hunk.newStart},{hunk.newLines} @@
      </div>
      {hunk.lines.map((line, i) => (
        <div
          key={i}
          className={cn(
            "flex",
            line.type === "ADDED" && "bg-green-50  dark:bg-green-950/30",
            line.type === "DELETED" && "bg-red-50    dark:bg-red-950/30",
          )}
        >
          <span className="border-border/50 text-muted-foreground/50 w-10 shrink-0 border-r px-2 text-right select-none">
            {line.oldLine || ""}
          </span>
          <span className="border-border/50 text-muted-foreground/50 w-10 shrink-0 border-r px-2 text-right select-none">
            {line.newLine || ""}
          </span>
          <span
            className={cn(
              "w-5 shrink-0 select-none text-center",
              line.type === "ADDED" && "text-green-600 dark:text-green-400",
              line.type === "DELETED" && "text-red-500   dark:text-red-400",
              line.type === "CONTEXT" && "text-muted-foreground/40",
            )}
          >
            {line.type === "ADDED" ? "+" : line.type === "DELETED" ? "-" : " "}
          </span>
          <pre
            className={cn(
              "flex-1 overflow-visible whitespace-pre px-2",
              line.type === "ADDED" && "text-green-900 dark:text-green-200",
              line.type === "DELETED" && "text-red-900   dark:text-red-200",
            )}
          >
            {line.content}
          </pre>
        </div>
      ))}
    </div>
  );
}
