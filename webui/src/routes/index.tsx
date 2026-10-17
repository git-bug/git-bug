// Repository picker page (/). Auto-redirects when there is exactly one repo.
// Shows a list when multiple repos are registered.

import { useReadQuery } from "@apollo/client/react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { GitFork, FolderOpen, AlertCircle } from "lucide-react";
import { useEffect } from "react";

import { graphql } from "@/__generated__/gql";

const REPOSITORIES_QUERY = graphql(`
  query Repositories {
    repositories {
      nodes {
        name
      }
      totalCount
    }
  }
`);

export const Route = createFileRoute("/")({
  component: RouteComponent,
  loader: async ({ context: { preloadQuery } }) => {
    const repositoriesRef = preloadQuery(REPOSITORIES_QUERY);
    return { repositoriesRef: await preloadQuery.toPromise(repositoriesRef) };
  },
});

// The default repository has an empty name, and "_" as URL slug.
function repoSlug(name: string): string {
  return name || "_";
}

function repoLabel(name: string): string {
  return name || "default";
}

function RouteComponent() {
  const { repositoriesRef } = Route.useLoaderData();
  const { data, error } = useReadQuery(repositoriesRef);
  const navigate = useNavigate();

  // Auto-redirect when there is exactly one repo — no need to pick.
  useEffect(() => {
    const nodes = data?.repositories.nodes ?? [];
    if (nodes.length === 1 && nodes[0]) {
      void navigate({
        to: "/$repo",
        params: { repo: repoSlug(nodes[0].name) },
        replace: true,
      });
    }
  }, [data, navigate]);

  return (
    <div className="mx-auto max-w-lg py-12">
      <div className="mb-8 flex items-center gap-3">
        <GitFork className="text-muted-foreground size-6" />
        <h1 className="text-xl font-semibold">Repositories</h1>
      </div>

      {error && (
        <div className="border-destructive/30 bg-destructive/10 text-destructive flex items-center gap-2 rounded-md border px-4 py-3 text-sm">
          <AlertCircle className="size-4 shrink-0" />
          Failed to load repositories: {error.message}
        </div>
      )}

      <div className="divide-border border-border divide-y rounded-md border">
        {data?.repositories.nodes.map((repo) => (
          <Link
            key={repoSlug(repo.name)}
            to="/$repo"
            params={{ repo: repoSlug(repo.name) }}
            className="hover:bg-muted/50 flex items-center gap-3 px-4 py-4 transition-colors"
          >
            <FolderOpen className="text-muted-foreground size-5 shrink-0" />
            <p className="text-foreground font-medium">{repoLabel(repo.name)}</p>
          </Link>
        ))}

        {data?.repositories.totalCount === 0 && (
          <p className="text-muted-foreground px-4 py-8 text-center text-sm">
            No repositories found.
          </p>
        )}
      </div>
    </div>
  );
}
