import { useReadQuery } from "@apollo/client/react";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { formatDistanceToNow } from "date-fns";
import { Search } from "lucide-react";
import { useMemo, useState } from "react";
import * as v from "valibot";

import { graphql } from "@/__generated__/gql";
import { IssueFilters } from "@/components/shared/issue-filters";

const BUG_LIST_QUERY = graphql(`
  query BugList(
    $ref: String
    $openQuery: String!
    $closedQuery: String!
    $listQuery: String!
    $first: Int
    $after: String
  ) {
    repository(ref: $ref) {
      name
      openCount: allBugs(query: $openQuery, first: 1) {
        totalCount
      }
      closedCount: allBugs(query: $closedQuery, first: 1) {
        totalCount
      }
      bugs: allBugs(query: $listQuery, first: $first, after: $after) {
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
            displayName
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
import * as IssueRow from "@/components/shared/issue-row";
import { LabelBadgeLink } from "@/components/shared/label-badge";
import * as Pagination from "@/components/shared/pagination";
import * as QueryInput from "@/components/shared/query-input";
import type { CompletionProvider } from "@/components/shared/query-input";
import * as StatusTabs from "@/components/shared/status-tabs";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import type { SortValue, StatusFilter } from "@/lib/query-utils";
import { buildBaseQuery, buildQueryString, parseQueryString } from "@/lib/query-utils";

const issuesSearchSchema = v.object({
  q: v.fallback(v.string(), "status:open"),
  after: v.fallback(v.string(), ""),
  page: v.optional(v.pipe(v.number(), v.integer(), v.minValue(1))),
  prev: v.optional(v.string()), // comma-separated stack of previous page cursors
});

export const Route = createFileRoute("/$repo/_issues/issues/")({
  component: RouteComponent,
  pendingComponent: BugListSkeleton,
  validateSearch: (search) => v.parse(issuesSearchSchema, search),
  loaderDeps: ({ search: { q, after, page, prev } }) => ({ q, after, page, prev }),
  loader: async ({ context: { preloadQuery, ref }, deps: { q, after } }) => {
    const parsed = parseQueryString(q);
    const baseQuery = buildBaseQuery(parsed.labels, parsed.author, parsed.freeText);
    const bugListRef = preloadQuery(BUG_LIST_QUERY, {
      variables: {
        ref,
        openQuery: `status:open ${baseQuery}`.trim(),
        closedQuery: `status:closed ${baseQuery}`.trim(),
        listQuery: q,
        first: PAGE_SIZE,
        after: after || undefined,
      },
    });
    return { bugListRef: await preloadQuery.toPromise(bugListRef) };
  },
});

const PAGE_SIZE = 25;

function RouteComponent() {
  const { repo } = Route.useParams();
  const navigate = useNavigate({ from: "/$repo/issues/" });
  const { q, after, page = 1, prev = "" } = Route.useSearch();

  // Parse the URL query into structured filter state for the dropdowns
  const parsed = useMemo(() => parseQueryString(q), [q]);
  const {
    status: statusFilter,
    labels: selectedLabels,
    author: selectedAuthorQuery,
    sort,
  } = parsed;
  // Draft is the text input value — starts from URL, only committed on submit
  const [draft, setDraft] = useState(q);

  // Re-seed the draft when the URL query changes from somewhere else (tab
  // clicks, filter dropdowns, back/forward).  Adjusting state during render is
  // React's recommended alternative to syncing it in an effect.
  const [lastQ, setLastQ] = useState(q);
  if (q !== lastQ) {
    setLastQ(q);
    setDraft(q);
  }

  const { bugListRef } = Route.useLoaderData();
  const { labelsRef, identitiesRef } = Route.useRouteContext();
  const { data } = useReadQuery(bugListRef);
  const { data: labelsData } = useReadQuery(labelsRef);
  const { data: identitiesData } = useReadQuery(identitiesRef);

  const openCount = data?.repository?.openCount.totalCount ?? 0;
  const closedCount = data?.repository?.closedCount.totalCount ?? 0;
  const bugs = data?.repository?.bugs;
  const validLabels = labelsData?.repository?.validLabels.nodes;
  const allIdentities = identitiesData?.repository?.allIdentities.nodes;

  // Resolve the author query value (login/name) back to a humanId for the filter UI
  const selectedAuthorId = useMemo(() => {
    if (!selectedAuthorQuery || !allIdentities) return null;
    const match = allIdentities.find(
      (i) =>
        i.login === selectedAuthorQuery ||
        i.name === selectedAuthorQuery ||
        i.humanId === selectedAuthorQuery,
    );
    return match?.humanId ?? null;
  }, [selectedAuthorQuery, allIdentities]);

  const completionProviders: CompletionProvider[] = useMemo(
    () => [
      {
        prefix: "label:",
        highlightClass: "text-yellow-600 dark:text-yellow-500",
        getSuggestions: (query: string) =>
          (validLabels ?? [])
            .filter((l) => query === "" || l.name.toLowerCase().includes(query.toLowerCase()))
            .slice(0, 8)
            .map((l) => ({
              value: l.name.includes(" ") ? `"${l.name}"` : l.name,
              label: l.name,
              icon: (
                <span
                  className="size-2 shrink-0 rounded-full"
                  style={{
                    backgroundColor: `rgb(${l.color.R},${l.color.G},${l.color.B})`,
                  }}
                />
              ),
            })),
      },
      {
        prefix: "author:",
        highlightClass: "text-blue-600 dark:text-blue-400",
        getSuggestions: (query: string) =>
          (allIdentities ?? [])
            .filter(
              (a) =>
                query === "" ||
                a.displayName.toLowerCase().includes(query.toLowerCase()) ||
                (a.login ?? "").toLowerCase().includes(query.toLowerCase()) ||
                (a.name ?? "").toLowerCase().includes(query.toLowerCase()),
            )
            .slice(0, 8)
            .map((a) => {
              const qv = a.login || a.name || a.humanId;
              return {
                value: qv.includes(" ") ? `"${qv}"` : qv,
                label: a.displayName,
                description: a.login && a.login !== a.displayName ? `@${a.login}` : undefined,
              };
            }),
      },
    ],
    [validLabels, allIdentities],
  );

  const totalCount = bugs?.totalCount ?? 0;
  const totalPages = Math.max(1, Math.ceil(totalCount / PAGE_SIZE));
  const hasNext = bugs?.pageInfo.hasNextPage ?? false;
  const hasPrev = page > 1;
  const prevCursors = prev ? prev.split(",") : [];

  // Navigate to new search params (resets pagination)
  function setSearch(newQ: string) {
    setDraft(newQ);
    void navigate({ search: { q: newQ, after: "" } });
  }

  // Apply structured filters → build query string → navigate
  function applyFilters(
    status: StatusFilter | null,
    labels: string[],
    authorQuery: string | null,
    text: string,
    sortVal: SortValue = sort,
  ) {
    setSearch(buildQueryString(status, labels, authorQuery, text, sortVal));
  }

  // Parse the draft text box on submit
  function handleSearch(e?: React.FormEvent) {
    e?.preventDefault();
    setSearch(draft);
  }

  // Build query string with toggled status
  function queryWithStatus(status: StatusFilter): string {
    return buildQueryString(status, selectedLabels, selectedAuthorQuery, parsed.freeText, sort);
  }

  return (
    <div>
      {/* Search bar */}
      <form onSubmit={handleSearch} className="mb-4 flex gap-2">
        <QueryInput.Root
          value={draft}
          onChange={setDraft}
          onSubmit={handleSearch}
          providers={completionProviders}
        >
          <QueryInput.Icon>
            <Search />
          </QueryInput.Icon>
          <QueryInput.Input placeholder="status:open author:… label:…" />
          <QueryInput.Completions />
        </QueryInput.Root>
        <Button type="submit">Search</Button>
      </form>

      {/* List container */}
      <div className="border-border rounded-md border">
        {/* Open / Closed toggle + filter dropdowns */}
        <div className="border-border flex items-center gap-2 overflow-x-auto border-b px-4 py-2">
          <StatusTabs.Root className="shrink-0">
            <StatusTabs.Tab
              to="/$repo/issues"
              params={{ repo }}
              search={{ q: queryWithStatus("open"), after: "" }}
              className={statusFilter === "open" ? "bg-accent text-accent-foreground" : ""}
            >
              <StatusTabs.OpenIndicator active={statusFilter === "open"} />
              Open
              <StatusTabs.Count>{openCount}</StatusTabs.Count>
            </StatusTabs.Tab>
            <StatusTabs.Tab
              to="/$repo/issues"
              params={{ repo }}
              search={{ q: queryWithStatus("closed"), after: "" }}
              className={statusFilter === "closed" ? "bg-accent text-accent-foreground" : ""}
            >
              <StatusTabs.ClosedIndicator active={statusFilter === "closed"} />
              Closed
              <StatusTabs.Count>{closedCount}</StatusTabs.Count>
            </StatusTabs.Tab>
          </StatusTabs.Root>

          <div className="ml-auto">
            <IssueFilters
              labels={validLabels ?? []}
              identities={allIdentities ?? []}
              selectedLabels={selectedLabels}
              onLabelsChange={(labels) =>
                applyFilters(statusFilter, labels, selectedAuthorQuery, parsed.freeText)
              }
              selectedAuthorId={selectedAuthorId}
              onAuthorChange={(_id, qv) =>
                applyFilters(statusFilter, selectedLabels, qv, parsed.freeText)
              }
              recentAuthorIds={bugs?.nodes?.map((b) => b.author.humanId) ?? []}
              sort={sort}
              onSortChange={(s) =>
                applyFilters(statusFilter, selectedLabels, selectedAuthorQuery, parsed.freeText, s)
              }
            />
          </div>
        </div>

        {/* Bug rows */}
        {bugs?.nodes.length === 0 && <EmptyState>No {statusFilter ?? ""} issues found.</EmptyState>}

        {bugs?.nodes.map((bug) => (
          <IssueRow.Root key={bug.id} className="hover:bg-muted transition-colors">
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
                  <LabelBadgeLink
                    key={label.name}
                    label={label}
                    to="/$repo/issues"
                    params={{ repo }}
                    search={{
                      q: buildQueryString(
                        statusFilter,
                        selectedLabels.includes(label.name)
                          ? selectedLabels
                          : [...selectedLabels, label.name],
                        selectedAuthorQuery,
                        parsed.freeText,
                        sort,
                      ),
                      after: "",
                    }}
                    onClick={(e: React.MouseEvent) => e.stopPropagation()}
                  />
                ))}
              </IssueRow.TitleArea>
              <IssueRow.Meta>
                #{bug.humanId} opened{" "}
                {formatDistanceToNow(new Date(bug.createdAt), { addSuffix: true })} by{" "}
                <Link
                  to="/$repo/user/$id"
                  params={{ repo, id: bug.author.humanId }}
                  search={{ status: "open" as const, after: "" }}
                  className="hover:underline"
                >
                  {bug.author.displayName}
                </Link>
              </IssueRow.Meta>
            </div>
            <IssueRow.CommentCount count={bug.comments.totalCount} />
          </IssueRow.Root>
        ))}

        {totalPages > 1 && (
          <Pagination.Root>
            <Pagination.Previous
              to="/$repo/issues"
              params={{ repo }}
              search={{
                q,
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
              to="/$repo/issues"
              params={{ repo }}
              search={{
                q,
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

function BugListSkeleton() {
  return (
    <div className="divide-border divide-y">
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i} className="flex items-start gap-3 px-4 py-3">
          <Skeleton className="mt-0.5 size-4 rounded-full" />
          <div className="flex-1 space-y-2">
            <Skeleton className="h-4 w-2/3" />
            <Skeleton className="h-3 w-1/3" />
          </div>
        </div>
      ))}
    </div>
  );
}
