// Pathless layout for the code browser. Reads preloaded refs from the
// $repo context and renders the shared header (breadcrumb + ref selector
// + history toggle). Child routes (tree, blob, commits) render in Outlet.

import { useReadQuery } from "@apollo/client/react";
import {
  createFileRoute,
  Outlet,
  useNavigate,
  useParams,
  useRouterState,
} from "@tanstack/react-router";
import { GitCommit } from "lucide-react";

import { CodeBreadcrumb } from "@/components/code/code-breadcrumb";
import { RefSelector } from "@/components/code/ref-selector";
import { ButtonLink } from "@/components/ui/button-link";
import { Skeleton } from "@/components/ui/skeleton";

export type CodeViewMode = "tree" | "blob" | "commits";

export const Route = createFileRoute("/$repo/_code")({
  component: CodeLayout,
  pendingComponent: CodeLayoutSkeleton,
});

function CodeLayout() {
  const { repo } = Route.useParams();
  const { ref: repoRef, refsRef } = Route.useRouteContext();
  const { data: refsData } = useReadQuery(refsRef);
  const refs = refsData?.repository?.refs;
  const repoName = refsData?.repository?.name || repoRef || "default-repo";

  // Read child route params (ref and splat path)
  const allParams = useParams({ strict: false });
  const currentRef = allParams.ref ?? "";
  const currentPath = allParams._splat ?? "";

  // Read viewMode from the deepest child route's context.
  // Each child (tree, blob, commits) sets viewMode in its beforeLoad.
  const viewMode: CodeViewMode = useRouterState({
    select: (s) => {
      const ctx = s.matches.at(-1)?.context;
      if (ctx && typeof ctx === "object" && "viewMode" in ctx) {
        return ctx.viewMode;
      }
      return "tree";
    },
  });

  const navigate = useNavigate();

  function handleRefSelect(refName: string) {
    if (viewMode === "commits") {
      void navigate({
        to: "/$repo/commits/$ref",
        params: { repo, ref: refName },
        search: { path: currentPath || undefined },
      });
    } else if (viewMode === "blob") {
      void navigate({
        to: "/$repo/blob/$ref/$",
        params: { repo, ref: refName, _splat: currentPath },
      });
    } else {
      void navigate({
        to: "/$repo/tree/$ref/$",
        params: { repo, ref: refName, _splat: currentPath },
      });
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <CodeBreadcrumb
          repoName={repoName}
          currentRef={currentRef}
          path={currentPath}
          repo={repo}
        />
        <div className="flex items-center gap-2">
          {viewMode === "commits" ? (
            <ButtonLink
              to="/$repo/tree/$ref/$"
              params={{ repo, ref: currentRef, _splat: currentPath }}
              variant="secondary"
              size="sm"
            >
              <GitCommit className="size-3.5" />
              History
            </ButtonLink>
          ) : (
            <ButtonLink
              to="/$repo/commits/$ref"
              params={{ repo, ref: currentRef }}
              search={{ path: currentPath || undefined }}
              variant="outline"
              size="sm"
            >
              <GitCommit className="size-3.5" />
              History
            </ButtonLink>
          )}
          {refs && <RefSelector refs={refs} currentRef={currentRef} onSelect={handleRefSelect} />}
        </div>
      </div>

      <Outlet />
    </div>
  );
}

function CodeLayoutSkeleton() {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <Skeleton className="h-5 w-48" />
        <Skeleton className="h-8 w-28" />
      </div>
      <div className="divide-border border-border divide-y rounded-md border">
        {Array.from({ length: 8 }).map((_, i) => (
          <div key={i} className="flex items-center gap-3 px-4 py-2">
            <Skeleton className="size-4 rounded-sm" />
            <Skeleton className="h-4 w-32" />
          </div>
        ))}
      </div>
    </div>
  );
}
