// Paginated commit history grouped by calendar date. Each row links to the
// commit detail page. Used in CodePage's "History" view.

import { useQuery } from "@apollo/client/react";
import { Link } from "@tanstack/react-router";
import { formatDistanceToNow } from "date-fns";
import { GitCommit } from "lucide-react";
import { useState } from "react";

import { graphql } from "@/__generated__/gql";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

const COMMITS_QUERY = graphql(`
  query CommitList($repo: String, $ref: String!, $path: String, $after: String, $first: Int) {
    repository(ref: $repo) {
      name
      commits(ref: $ref, path: $path, after: $after, first: $first) {
        nodes {
          hash
          shortHash
          message
          authorName
          date
        }
        pageInfo {
          hasNextPage
          endCursor
        }
      }
    }
  }
`);

const PAGE_SIZE = 30;

interface CommitListProps {
  repo: string | null;
  ref_: string;
  path?: string;
}

type CommitNode = {
  hash: string;
  shortHash: string;
  message: string;
  authorName: string;
  date: string;
};

// Pages appended by "Load more", carrying the pageInfo of the last one fetched.
type MorePages = {
  nodes: CommitNode[];
  cursor: string | null;
  hasNextPage: boolean;
} | null;

export function CommitList({ repo, ref_, path }: CommitListProps) {
  const { data, loading, error, fetchMore } = useQuery(COMMITS_QUERY, {
    variables: { repo, ref: ref_, path: path ?? null, after: null, first: PAGE_SIZE },
    skip: !ref_,
  });

  // The query result is the first page; pages pulled in by "Load more" are
  // accumulated here.  They are dropped whenever the query result itself
  // changes (another repo, ref or path), which adjusting state during render
  // does without the extra render pass an effect would cost.
  const [morePages, setMorePages] = useState<MorePages>(null);
  const [lastData, setLastData] = useState(data);
  if (data !== lastData) {
    setLastData(data);
    setMorePages(null);
  }

  const firstPage = data?.repository?.commits?.nodes ?? [];
  const firstPageInfo = data?.repository?.commits?.pageInfo;
  const allCommits = morePages ? [...firstPage, ...morePages.nodes] : firstPage;
  const cursor = morePages ? morePages.cursor : (firstPageInfo?.endCursor ?? null);
  const hasNextPage = morePages ? morePages.hasNextPage : (firstPageInfo?.hasNextPage ?? false);

  // endCursor is non-null for every non-empty page, the last one included, so
  // only hasNextPage can tell us whether another page exists.
  const hasMore = hasNextPage && !!cursor;
  const [loadingMore, setLoadingMore] = useState(false);

  function loadMore() {
    if (!cursor) return;
    setLoadingMore(true);
    void fetchMore({
      variables: { after: cursor },
    })
      .then((result) => {
        const newNodes = result.data?.repository?.commits?.nodes ?? [];
        const pageInfo = result.data?.repository?.commits?.pageInfo;
        setMorePages((prev) => ({
          nodes: [...(prev?.nodes ?? []), ...newNodes],
          cursor: pageInfo?.endCursor ?? null,
          hasNextPage: pageInfo?.hasNextPage ?? false,
        }));
      })
      .finally(() => setLoadingMore(false));
  }

  if (loading) return <CommitListSkeleton />;

  if (error) {
    return (
      <div className="border-border text-destructive rounded-md border px-4 py-8 text-center text-sm">
        {error.message}
      </div>
    );
  }

  const groups = groupByDate(allCommits);

  return (
    <div className="space-y-6">
      {groups.map(([date, group]) => (
        <div key={date}>
          <h3 className="text-muted-foreground mb-2 text-xs font-semibold tracking-wider uppercase">
            Commits on {date}
          </h3>
          <div className="divide-border border-border divide-y overflow-hidden rounded-md border">
            {group.map((commit) => (
              <CommitRow key={commit.hash} commit={commit} repo={repo} />
            ))}
          </div>
        </div>
      ))}

      {hasMore && (
        <div className="text-center">
          <Button variant="outline" size="sm" onClick={loadMore} disabled={loadingMore}>
            {loadingMore ? "Loading…" : "Load more commits"}
          </Button>
        </div>
      )}
    </div>
  );
}

function CommitRow({ commit, repo }: { commit: CommitNode; repo: string | null }) {
  const commitPath = repo ? `/${repo}/commit/${commit.hash}` : `/commit/${commit.hash}`;
  return (
    <div className="bg-background hover:bg-muted/30 flex items-center gap-3 px-4 py-3">
      <GitCommit className="text-muted-foreground size-4 shrink-0" />
      <div className="min-w-0 flex-1">
        <Link
          to={commitPath}
          className="text-foreground hover:text-primary block truncate font-medium hover:underline"
        >
          {commit.message}
        </Link>
        <p className="text-muted-foreground mt-0.5 text-xs">
          {commit.authorName} &middot;{" "}
          {formatDistanceToNow(new Date(commit.date), { addSuffix: true })}
        </p>
      </div>
      <Link
        to={commitPath}
        className="text-muted-foreground hover:text-foreground shrink-0 font-mono text-xs hover:underline"
        title={commit.hash}
      >
        {commit.shortHash}
      </Link>
    </div>
  );
}

function groupByDate(commits: CommitNode[]): [string, CommitNode[]][] {
  const map = new Map<string, CommitNode[]>();
  for (const c of commits) {
    const date = new Date(c.date).toLocaleDateString("en-US", {
      year: "numeric",
      month: "long",
      day: "numeric",
    });
    const group = map.get(date) ?? [];
    group.push(c);
    map.set(date, group);
  }
  return Array.from(map.entries());
}

function CommitListSkeleton() {
  return (
    <div className="space-y-6">
      {Array.from({ length: 2 }).map((_, g) => (
        <div key={g}>
          <Skeleton className="mb-2 h-3 w-32" />
          <div className="divide-border border-border divide-y overflow-hidden rounded-md border">
            {Array.from({ length: 4 }).map((_c, i) => (
              <div key={i} className="flex items-center gap-3 px-4 py-3">
                <Skeleton className="size-4 rounded-sm" />
                <div className="flex-1 space-y-1.5">
                  <Skeleton className="h-4 w-2/3" />
                  <Skeleton className="h-3 w-1/4" />
                </div>
                <Skeleton className="h-3 w-14" />
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}
