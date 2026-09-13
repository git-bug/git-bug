// Commit detail page (/:repo/commit/:hash). Shows commit metadata, full
// message, parent links, and changed files with lazy diffs.

import { useReadQuery } from "@apollo/client/react";
import { createFileRoute, Link } from "@tanstack/react-router";
import { format } from "date-fns";
import { GitCommit } from "lucide-react";

import { graphql } from "@/__generated__/gql";
import { FileDiffView } from "@/components/code/file-diff-view";
import { BackLink } from "@/components/ui/back-link";
import { Skeleton } from "@/components/ui/skeleton";

const COMMIT_QUERY = graphql(`
  query CommitPageDetail($repo: String, $hash: String!) {
    repository(ref: $repo) {
      name
      commit(hash: $hash) {
        hash
        shortHash
        message
        fullMessage
        authorName
        authorEmail
        date
        parents
        files {
          nodes {
            path
            oldPath
            status
          }
        }
      }
    }
  }
`);

export const Route = createFileRoute("/$repo/commit/$hash")({
  component: RouteComponent,
  pendingComponent: CommitPageSkeleton,
  loader: async ({ context: { preloadQuery, ref }, params: { hash } }) => {
    const commitRef = preloadQuery(COMMIT_QUERY, {
      variables: { repo: ref, hash },
    });
    return { commitRef: await preloadQuery.toPromise(commitRef) };
  },
});

function RouteComponent() {
  const { ref } = Route.useRouteContext();
  const { repo } = Route.useParams();
  const { commitRef } = Route.useLoaderData();
  const { data } = useReadQuery(commitRef);

  const commit = data?.repository?.commit;
  if (!commit) return null;

  const date = new Date(commit.date);
  const files = commit.files?.nodes ?? [];

  return (
    <div>
      <BackLink to="/$repo" params={{ repo }}>
        Back
      </BackLink>

      <div className="border-border mb-6 rounded-md border p-5">
        <div className="mb-1 flex items-start gap-3">
          <GitCommit className="text-muted-foreground mt-1 size-5 shrink-0" />
          <h1 className="text-lg leading-snug font-semibold">{commit.message}</h1>
        </div>

        {commit.fullMessage.includes("\n") && (
          <pre className="text-muted-foreground mt-3 mb-4 ml-8 font-sans text-sm whitespace-pre-wrap">
            {commit.fullMessage.split("\n").slice(1).join("\n").trim()}
          </pre>
        )}

        <div className="text-muted-foreground mt-3 ml-8 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm">
          <span>
            <span className="text-foreground font-medium">{commit.authorName}</span>
            {commit.authorEmail && <span> &lt;{commit.authorEmail}&gt;</span>}
          </span>
          <span title={date.toISOString()}>{format(date, "PPP")}</span>
        </div>

        <div className="mt-3 ml-8 flex flex-wrap gap-3 text-xs">
          <span className="text-muted-foreground">
            commit <code className="text-foreground font-mono">{commit.hash}</code>
          </span>
          {commit.parents.map((p: string) => (
            <span key={p} className="text-muted-foreground">
              parent{" "}
              <Link
                to="/$repo/commit/$hash"
                params={{ repo, hash: p }}
                className="text-foreground font-mono hover:underline"
              >
                {p.slice(0, 7)}
              </Link>
            </span>
          ))}
        </div>
      </div>

      <div>
        <h2 className="text-muted-foreground mb-3 text-sm font-semibold">
          {files.length} file{files.length !== 1 ? "s" : ""} changed
        </h2>
        <div className="divide-border border-border divide-y overflow-hidden rounded-md border">
          {files.length === 0 && (
            <p className="text-muted-foreground px-4 py-4 text-sm">No file changes.</p>
          )}
          {files.map((file: { path: string; oldPath?: string | null; status: string }) => (
            <FileDiffView
              key={file.path}
              repo={ref}
              hash={commit.hash}
              path={file.path}
              oldPath={file.oldPath ?? undefined}
              status={file.status}
            />
          ))}
        </div>
      </div>
    </div>
  );
}

function CommitPageSkeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-4 w-24" />
      <div className="border-border space-y-3 rounded-md border p-5">
        <Skeleton className="h-6 w-3/4" />
        <Skeleton className="h-4 w-1/3" />
        <Skeleton className="h-3 w-1/2" />
      </div>
      <div className="divide-border border-border divide-y rounded-md border">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="flex items-center gap-3 px-4 py-2.5">
            <Skeleton className="size-4" />
            <Skeleton className="h-4 flex-1" />
          </div>
        ))}
      </div>
    </div>
  );
}
