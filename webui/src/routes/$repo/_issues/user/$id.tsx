// User profile page (/user/:id). Fetches an identity by prefix and shows:
//   - avatar, display name, login, email, humanId, protected badge
//   - open/closed issue toggle with BOTH counts always visible
//   - paginated list of that user's bugs

import { useReadQuery } from "@apollo/client/react";
import { createFileRoute, Link } from "@tanstack/react-router";
import { formatDistanceToNow } from "date-fns";
import { CircleDot, CircleCheck, ShieldCheck } from "lucide-react";
import * as v from "valibot";

import { graphql } from "@/__generated__/gql";
import * as IssueRow from "@/components/shared/issue-row";

const USER_PROFILE_QUERY = graphql(`
  query UserProfile(
    $ref: String
    $prefix: String!
    $openQuery: String!
    $closedQuery: String!
    $listQuery: String!
    $after: String
  ) {
    repository(ref: $ref) {
      name
      identity(prefix: $prefix) {
        id
        humanId
        name
        email
        login
        displayName
        avatarUrl
        isProtected
      }
      openCount: allBugs(query: $openQuery, first: 1) {
        totalCount
      }
      closedCount: allBugs(query: $closedQuery, first: 1) {
        totalCount
      }
      bugs: allBugs(query: $listQuery, first: 25, after: $after) {
        totalCount
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          ...BugSummary
          id
          humanId
          status
          title
          createdAt
          labels {
            name
            ...LabelFields
          }
          author {
            humanId
            ...IdentitySummary
          }
          comments {
            totalCount
          }
        }
      }
    }
  }
`);
import { EmptyState } from "@/components/shared/empty-state";
import { LabelBadge } from "@/components/shared/label-badge";
import * as Pagination from "@/components/shared/pagination";
import * as StatusTabs from "@/components/shared/status-tabs";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { BackLink } from "@/components/ui/back-link";
import { Skeleton } from "@/components/ui/skeleton";

const profileSearchSchema = v.object({
  status: v.fallback(v.picklist(["open", "closed"]), "open"),
  after: v.fallback(v.string(), ""),
  page: v.optional(v.pipe(v.number(), v.integer(), v.minValue(1))),
  prev: v.optional(v.string()), // comma-separated stack of previous page cursors
});

const PAGE_SIZE = 25;

export const Route = createFileRoute("/$repo/_issues/user/$id")({
  component: RouteComponent,
  pendingComponent: ProfileSkeleton,
  validateSearch: (search) => v.parse(profileSearchSchema, search),
  loaderDeps: ({ search: { status, after, page, prev } }) => ({ status, after, page, prev }),
  loader: async ({ context: { preloadQuery, ref }, params: { id }, deps: { status, after } }) => {
    const profileRef = preloadQuery(USER_PROFILE_QUERY, {
      variables: {
        ref,
        prefix: id,
        openQuery: `author:${id} status:open`,
        closedQuery: `author:${id} status:closed`,
        listQuery: `author:${id} status:${status}`,
        after: after || undefined,
      },
    });
    return { profileRef: await preloadQuery.toPromise(profileRef) };
  },
});

function RouteComponent() {
  const { id, repo } = Route.useParams();
  const { status: statusFilter, after, page = 1, prev = "" } = Route.useSearch();
  const { profileRef } = Route.useLoaderData();
  const { data } = useReadQuery(profileRef);

  const identity = data?.repository?.identity;
  if (!identity) {
    return <div className="text-muted-foreground py-16 text-center text-sm">User not found.</div>;
  }

  const openCount = data?.repository?.openCount.totalCount ?? 0;
  const closedCount = data?.repository?.closedCount.totalCount ?? 0;

  const bugs = data?.repository?.bugs;
  const totalPages = Math.max(1, Math.ceil((bugs?.totalCount ?? 0) / PAGE_SIZE));
  const hasNext = bugs?.pageInfo.hasNextPage ?? false;
  const hasPrev = page > 1;
  const prevCursors = prev ? prev.split(",") : [];

  return (
    <div>
      <BackLink to="/$repo/issues" params={{ repo }} search={{ q: "status:open", after: "" }}>
        Back to issues
      </BackLink>

      {/* ── Profile header ─────────────────────────────────────────────── */}
      <div className="mb-8 flex items-start gap-5">
        <Avatar className="size-20">
          <AvatarImage src={identity.avatarUrl ?? undefined} alt={identity.displayName} />
          <AvatarFallback className="text-2xl">
            {identity.displayName.slice(0, 2).toUpperCase()}
          </AvatarFallback>
        </Avatar>

        <div className="pt-1">
          <div className="flex items-center gap-2">
            <h1 className="text-xl font-semibold">{identity.displayName}</h1>
            {identity.isProtected && (
              <span title="Protected identity">
                <ShieldCheck className="text-muted-foreground size-4" />
              </span>
            )}
          </div>
          <div className="text-muted-foreground mt-1 space-y-0.5 text-sm">
            {identity.login && <p>@{identity.login}</p>}
            {identity.email && <p>{identity.email}</p>}
            <p className="font-mono text-xs">#{identity.humanId}</p>
          </div>

          <div className="mt-3 flex items-center gap-4 text-sm">
            <span className="text-muted-foreground flex items-center gap-1">
              <CircleDot className="size-3.5 text-green-600 dark:text-green-400" />
              <span className="text-foreground font-medium">{openCount}</span> open
            </span>
            <span className="text-muted-foreground flex items-center gap-1">
              <CircleCheck className="size-3.5 text-purple-600 dark:text-purple-400" />
              <span className="text-foreground font-medium">{closedCount}</span> closed
            </span>
          </div>
        </div>
      </div>

      {/* ── Issue list ─────────────────────────────────────────────────── */}
      <div className="border-border rounded-md border">
        {/* Open / Closed toggle */}
        <StatusTabs.Root className="border-border border-b px-4 py-2">
          <StatusTabs.Tab
            to="/$repo/user/$id"
            params={{ repo, id }}
            search={{ status: "open", after: "" }}
            className={statusFilter === "open" ? "bg-accent text-accent-foreground" : ""}
          >
            <StatusTabs.OpenIndicator active={statusFilter === "open"} />
            Open
            <StatusTabs.Count>{openCount}</StatusTabs.Count>
          </StatusTabs.Tab>
          <StatusTabs.Tab
            to="/$repo/user/$id"
            params={{ repo, id }}
            search={{ status: "closed", after: "" }}
            className={statusFilter === "closed" ? "bg-accent text-accent-foreground" : ""}
          >
            <StatusTabs.ClosedIndicator active={statusFilter === "closed"} />
            Closed
            <StatusTabs.Count>{closedCount}</StatusTabs.Count>
          </StatusTabs.Tab>
        </StatusTabs.Root>

        {bugs?.nodes.length === 0 && <EmptyState>No {statusFilter} issues.</EmptyState>}

        {bugs?.nodes.map((bug) => (
          <IssueRow.Root key={bug.id}>
            <IssueRow.StatusIcon status={bug.status} />
            <div className="min-w-0 flex-1">
              <IssueRow.TitleArea>
                <Link
                  to="/$repo/issues/$id"
                  params={{ repo, id: bug.humanId }}
                  className="text-foreground hover:text-primary font-medium hover:underline"
                >
                  {bug.title}
                </Link>
                {bug.labels.map((label) => (
                  <LabelBadge key={label.name} label={label} />
                ))}
              </IssueRow.TitleArea>
              <IssueRow.Meta>
                #{bug.humanId} opened{" "}
                {formatDistanceToNow(new Date(bug.createdAt), { addSuffix: true })}
              </IssueRow.Meta>
            </div>
            <IssueRow.CommentCount count={bug.comments.totalCount} />
          </IssueRow.Root>
        ))}

        {totalPages > 1 && (
          <Pagination.Root>
            <Pagination.Previous
              to="/$repo/user/$id"
              params={{ repo, id }}
              search={{
                status: statusFilter,
                after: prevCursors.at(-1) ?? "",
                page: page - 1 > 1 ? page - 1 : undefined,
                prev: prevCursors.slice(0, -1).join(",") || undefined,
              }}
              disabled={!hasPrev}
            />
            <Pagination.Info>
              Page {page} of {totalPages}
            </Pagination.Info>
            <Pagination.Next
              to="/$repo/user/$id"
              params={{ repo, id }}
              search={{
                status: statusFilter,
                after: bugs?.pageInfo.endCursor ?? "",
                page: page + 1,
                prev: (prev ? `${prev},${after}` : after) || undefined,
              }}
              disabled={!hasNext}
            />
          </Pagination.Root>
        )}
      </div>
    </div>
  );
}

function ProfileSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-start gap-5">
        <Skeleton className="size-20 rounded-full" />
        <div className="space-y-2 pt-1">
          <Skeleton className="h-6 w-40" />
          <Skeleton className="h-4 w-24" />
          <Skeleton className="h-3 w-16" />
        </div>
      </div>
      <div className="border-border rounded-md border">
        <div className="border-border border-b px-4 py-2">
          <Skeleton className="h-8 w-48" />
        </div>
        <div className="divide-border divide-y">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="flex items-start gap-3 px-4 py-3">
              <Skeleton className="mt-0.5 size-4 rounded-full" />
              <div className="flex-1 space-y-2">
                <Skeleton className="h-4 w-2/3" />
                <Skeleton className="h-3 w-1/3" />
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
