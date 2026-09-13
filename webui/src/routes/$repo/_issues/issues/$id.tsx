import { useReadQuery } from "@apollo/client/react";
import { createFileRoute, Link } from "@tanstack/react-router";
import { formatDistanceToNow } from "date-fns";

import { graphql } from "@/__generated__/gql";
import { CommentBox } from "@/components/bugs/comment-box";

const BUG_DETAIL_QUERY = graphql(`
  query BugDetail($ref: String, $prefix: String!) {
    repository(ref: $ref) {
      name
      bug(prefix: $prefix) {
        ...BugSummary
        humanId
        title
        status
        createdAt
        labels {
          name
          ...LabelFields
        }
        author {
          humanId
          displayName
        }
        lastEdit
        participants(first: 20) {
          nodes {
            ...IdentitySummary
            id
            humanId
            displayName
            avatarUrl
          }
        }
        timeline(first: 250) {
          ...TimelineItems
        }
      }
    }
  }
`);
import { LabelEditor } from "@/components/bugs/label-editor";
import { Timeline } from "@/components/bugs/timeline";
import { TitleEditor } from "@/components/bugs/title-editor";
import { EmptyState } from "@/components/shared/empty-state";
import { SectionHeading } from "@/components/shared/section-heading";
import { StatusBadge } from "@/components/shared/status-badge";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { BackLink } from "@/components/ui/back-link";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";

export const Route = createFileRoute("/$repo/_issues/issues/$id")({
  component: RouteComponent,
  pendingComponent: BugDetailSkeleton,
  loader: async ({ context: { preloadQuery, ref }, params: { id } }) => {
    const bugDetailRef = preloadQuery(BUG_DETAIL_QUERY, {
      variables: { ref, prefix: id },
    });
    return { bugDetailRef: await preloadQuery.toPromise(bugDetailRef) };
  },
});

// Issue detail page (/:repo/issues/:id). Shows title, status, timeline of
// comments and events, and a sidebar with labels and participants.
function RouteComponent() {
  const { ref } = Route.useRouteContext();
  const { repo } = Route.useParams();
  const { bugDetailRef } = Route.useLoaderData();
  const { labelsRef } = Route.useRouteContext();
  const { data } = useReadQuery(bugDetailRef);
  const { data: labelsData } = useReadQuery(labelsRef);
  const validLabels = labelsData?.repository?.validLabels.nodes ?? [];

  const bug = data?.repository?.bug;
  if (!bug) {
    return <EmptyState className="py-16">Issue not found.</EmptyState>;
  }

  return (
    <div>
      <BackLink to="/$repo/issues" params={{ repo }} search={{ q: "status:open", after: "" }}>
        Back to issues
      </BackLink>

      {/* Title row — hover reveals edit button when logged in */}
      <div className="mb-3">
        <TitleEditor bugPrefix={bug.humanId} title={bug.title} humanId={bug.humanId} ref_={ref} />
      </div>

      <div className="text-muted-foreground mb-6 flex flex-wrap items-center gap-3 text-sm">
        <StatusBadge status={bug.status} />
        <span>
          <Link
            to="/$repo/user/$id"
            params={{ repo: repo, id: bug.author.humanId }}
            search={{ status: "open" as const, after: "" }}
            className="text-foreground font-medium hover:underline"
          >
            {bug.author.displayName}
          </Link>{" "}
          opened this issue {formatDistanceToNow(new Date(bug.createdAt), { addSuffix: true })}
        </span>
      </div>

      <Separator className="mb-6" />

      <div className="flex gap-8">
        {/* Timeline + comment box */}
        <div className="min-w-0 flex-1 space-y-4">
          <Timeline repo={repo} bugPrefix={bug.humanId} timeline={bug.timeline} />
          <CommentBox bugPrefix={bug.humanId} bugStatus={bug.status} ref_={ref} />
        </div>

        {/* Sidebar */}
        <aside className="w-56 shrink-0 space-y-6">
          <LabelEditor
            bugPrefix={bug.humanId}
            currentLabels={bug.labels}
            ref_={ref}
            validLabels={validLabels}
          />

          <Separator />

          <div>
            <SectionHeading>Participants</SectionHeading>
            <div className="flex flex-wrap gap-1.5">
              {bug.participants.nodes.map((p) => {
                return (
                  <Link
                    key={p.id}
                    to="/$repo/user/$id"
                    params={{ repo: repo, id: p.humanId }}
                    search={{ status: "open" as const, after: "" }}
                    title={p.displayName}
                  >
                    <Avatar className="size-6">
                      <AvatarImage src={p.avatarUrl ?? undefined} alt={p.displayName} />
                      <AvatarFallback className="text-[10px]">
                        {p.displayName.slice(0, 2).toUpperCase()}
                      </AvatarFallback>
                    </Avatar>
                  </Link>
                );
              })}
            </div>
          </div>
        </aside>
      </div>
    </div>
  );
}

function BugDetailSkeleton() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-8 w-2/3" />
      <Skeleton className="h-4 w-1/3" />
      <Separator />
      <div className="flex gap-8">
        <div className="flex-1 space-y-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="border-border rounded-md border p-4">
              <Skeleton className="mb-3 h-4 w-1/4" />
              <Skeleton className="h-16 w-full" />
            </div>
          ))}
        </div>
        <div className="w-56 space-y-3">
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-3/4" />
        </div>
      </div>
    </div>
  );
}
