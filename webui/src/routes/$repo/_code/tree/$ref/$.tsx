// Tree view: /$repo/tree/$ref/...path

import { useQuery, useReadQuery } from "@apollo/client/react";
import { createFileRoute } from "@tanstack/react-router";

import { GitObjectType } from "@/__generated__/enums";
import { graphql } from "@/__generated__/gql";
import { FileTree } from "@/components/code/file-tree";
import { Markdown } from "@/components/content/markdown";
import { Skeleton } from "@/components/ui/skeleton";

const TREE_QUERY = graphql(`
  query CodePageTree($repo: String, $ref: String!, $path: String) {
    repository(ref: $repo) {
      name
      tree(ref: $ref, path: $path) {
        name
        type
        hash
      }
    }
  }
`);

const LAST_COMMITS_QUERY = graphql(`
  query CodePageLastCommits($repo: String, $ref: String!, $path: String, $names: [String!]!) {
    repository(ref: $repo) {
      name
      lastCommits(ref: $ref, path: $path, names: $names) {
        name
        commit {
          hash
          shortHash
          message
          date
        }
      }
    }
  }
`);

const README_QUERY = graphql(`
  query CodePageReadme($repo: String, $ref: String!, $path: String!) {
    repository(ref: $repo) {
      name
      blob(ref: $ref, path: $path) {
        text
      }
    }
  }
`);

function TreeSkeleton() {
  return (
    <div className="border-border overflow-hidden rounded-md border">
      <div className="divide-border divide-y">
        {Array.from({ length: 8 }).map((_, i) => (
          <div key={i} className="flex items-center gap-3 px-4 py-2">
            <Skeleton className="size-4 rounded-sm" />
            <Skeleton className="h-4 w-32" />
            <Skeleton className="ml-6 hidden h-4 w-64 md:block" />
            <Skeleton className="ml-auto hidden h-4 w-20 md:block" />
          </div>
        ))}
      </div>
    </div>
  );
}

export const Route = createFileRoute("/$repo/_code/tree/$ref/$")({
  component: TreeView,
  pendingComponent: TreeSkeleton,
  beforeLoad: () => ({ viewMode: "tree" as const }),
  loader: async ({ context: { preloadQuery, ref }, params: { ref: gitRef, _splat: path } }) => {
    const treeRef = preloadQuery(TREE_QUERY, {
      variables: { repo: ref, ref: gitRef, ...(path ? { path } : {}) },
    });
    return { treeRef: await preloadQuery.toPromise(treeRef) };
  },
});

function TreeView() {
  const { repo, ref: currentRef, _splat: currentPath = "" } = Route.useParams();
  const { ref: repoRef } = Route.useRouteContext();
  const { treeRef } = Route.useLoaderData();
  const { data: treeData } = useReadQuery(treeRef);
  const entries = treeData?.repository?.tree ?? [];

  // Last commits and readme are cascading queries — they depend on the tree result
  const entryNames = entries.map((e) => e.name);
  const { data: lastCommitsData } = useQuery(LAST_COMMITS_QUERY, {
    variables: {
      repo: repoRef,
      ref: currentRef,
      path: currentPath || null,
      names: entryNames,
    },
    skip: entryNames.length === 0,
  });
  const lastCommitsByName = new Map(
    (lastCommitsData?.repository?.lastCommits ?? []).map((lc) => [lc.name, lc]),
  );
  const entriesWithCommits = entries.map((e) => ({
    ...e,
    lastCommit: lastCommitsByName.get(e.name)?.commit ?? undefined,
  }));

  const readmeEntry = entries.find(
    (e) => e.type === GitObjectType.Blob && /^readme(\.md|\.txt|\.rst)?$/i.test(e.name),
  );
  const readmePath = readmeEntry
    ? currentPath
      ? `${currentPath}/${readmeEntry.name}`
      : readmeEntry.name
    : null;
  const { data: readmeBlobData } = useQuery(README_QUERY, {
    variables: { repo: repoRef, ref: currentRef, path: readmePath || "" },
    skip: !readmePath,
  });
  const readme: string | null = readmeBlobData?.repository?.blob?.text ?? null;

  return (
    <>
      <FileTree
        repo={repo}
        currentRef={currentRef}
        currentPath={currentPath}
        entries={entriesWithCommits}
      />
      {readme && (
        <div className="rounded-md border">
          <div className="text-muted-foreground border-b px-4 py-2 text-xs font-medium">README</div>
          <div className="px-6 py-4">
            <Markdown
              content={readme}
              size="document"
              repoContext={{ repo, ref: currentRef, basePath: currentPath }}
            />
          </div>
        </div>
      )}
    </>
  );
}
