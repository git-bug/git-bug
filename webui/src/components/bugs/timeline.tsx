import { useMutation } from "@apollo/client/react";
import type { ResultOf } from "@graphql-typed-document-node/core";
import { Link } from "@tanstack/react-router";
import { formatDistanceToNow } from "date-fns";
import { Tag, GitPullRequestClosed, Pencil, CircleDot } from "lucide-react";
import { useState } from "react";

import { Status } from "@/__generated__/enums";
import { useFragment, type FragmentType } from "@/__generated__/fragment-masking";
import { graphql } from "@/__generated__/gql";
import { BugDetailDocument } from "@/__generated__/graphql";
import { Markdown } from "@/components/content/markdown";
import * as CommentCard from "@/components/shared/comment-card";
import { LabelBadge } from "@/components/shared/label-badge";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { useAuth } from "@/lib/auth";

// ── Sub-fragments ────────────────────────────────────────────────────────────

const BUG_CREATE_COMMENT_FRAGMENT = graphql(`
  fragment BugCreateCommentFields on BugCreateTimelineItem {
    id
    author {
      id
      humanId
      displayName
      ...IdentitySummary
    }
    message
    createdAt
    lastEdit
    edited
  }
`);

const BUG_ADD_COMMENT_FRAGMENT = graphql(`
  fragment BugAddCommentFields on BugAddCommentTimelineItem {
    id
    author {
      id
      humanId
      displayName
      ...IdentitySummary
    }
    message
    createdAt
    lastEdit
    edited
  }
`);

const LABEL_CHANGE_FRAGMENT = graphql(`
  fragment LabelChangeFields on BugLabelChangeTimelineItem {
    author {
      humanId
      displayName
    }
    date
    added {
      ...LabelFields
    }
    removed {
      ...LabelFields
    }
  }
`);

const STATUS_CHANGE_FRAGMENT = graphql(`
  fragment StatusChangeFields on BugSetStatusTimelineItem {
    author {
      humanId
      displayName
    }
    date
    status
  }
`);

const TITLE_CHANGE_FRAGMENT = graphql(`
  fragment TitleChangeFields on BugSetTitleTimelineItem {
    author {
      humanId
      displayName
    }
    date
    title
    was
  }
`);

// ── Connection-level fragment ────────────────────────────────────────────────

export const TIMELINE_ITEMS_FRAGMENT = graphql(`
  fragment TimelineItems on BugTimelineItemConnection {
    nodes {
      __typename
      id
      ... on BugCreateTimelineItem {
        ...BugCreateCommentFields
      }
      ... on BugAddCommentTimelineItem {
        ...BugAddCommentFields
      }
      ... on BugLabelChangeTimelineItem {
        ...LabelChangeFields
      }
      ... on BugSetStatusTimelineItem {
        ...StatusChangeFields
      }
      ... on BugSetTitleTimelineItem {
        ...TitleChangeFields
      }
    }
  }
`);

// ── Mutation ─────────────────────────────────────────────────────────────────

const BUG_EDIT_COMMENT_MUTATION = graphql(`
  mutation BugEditComment($input: BugEditCommentInput!) {
    bugEditComment(input: $input) {
      bug {
        id
      }
    }
  }
`);

// ── Type helpers ─────────────────────────────────────────────────────────────

type TimelineData = ResultOf<typeof TIMELINE_ITEMS_FRAGMENT>;
type TimelineNode = TimelineData["nodes"][number];

// ── Timeline ─────────────────────────────────────────────────────────────────

interface TimelineProps {
  repo: string | null;
  bugPrefix: string;
  timeline: FragmentType<typeof TIMELINE_ITEMS_FRAGMENT>;
}

// Ordered sequence of events on a bug: comments (create and add-comment) and
// inline events (label changes, status changes, title edits). Comment items
// support inline editing for the logged-in user.
export function Timeline({ repo, bugPrefix, timeline }: TimelineProps) {
  const data = useFragment(TIMELINE_ITEMS_FRAGMENT, timeline);

  return (
    <div className="space-y-4">
      {data.nodes.map((item) => {
        switch (item.__typename) {
          case "BugCreateTimelineItem":
            return (
              <CreateCommentItem key={item.id} item={item} bugPrefix={bugPrefix} repo={repo} />
            );
          case "BugAddCommentTimelineItem":
            return <AddCommentItem key={item.id} item={item} bugPrefix={bugPrefix} repo={repo} />;
          case "BugLabelChangeTimelineItem":
            return <LabelChangeItem key={item.id} item={item} repo={repo} />;
          case "BugSetStatusTimelineItem":
            return <StatusChangeItem key={item.id} item={item} repo={repo} />;
          case "BugSetTitleTimelineItem":
            return <TitleChangeItem key={item.id} item={item} repo={repo} />;
          default:
            return null;
        }
      })}
    </div>
  );
}

// ── Comment items ────────────────────────────────────────────────────────────

type CreateNode = Extract<TimelineNode, { __typename: "BugCreateTimelineItem" }>;
type AddCommentNode = Extract<TimelineNode, { __typename: "BugAddCommentTimelineItem" }>;
type CommentData =
  | ResultOf<typeof BUG_CREATE_COMMENT_FRAGMENT>
  | ResultOf<typeof BUG_ADD_COMMENT_FRAGMENT>;

function CreateCommentItem({
  item,
  bugPrefix,
  repo,
}: {
  item: CreateNode;
  bugPrefix: string;
  repo: string | null;
}) {
  const data = useFragment(BUG_CREATE_COMMENT_FRAGMENT, item);
  return <CommentBody data={data} bugPrefix={bugPrefix} repo={repo} />;
}

function AddCommentItem({
  item,
  bugPrefix,
  repo,
}: {
  item: AddCommentNode;
  bugPrefix: string;
  repo: string | null;
}) {
  const data = useFragment(BUG_ADD_COMMENT_FRAGMENT, item);
  return <CommentBody data={data} bugPrefix={bugPrefix} repo={repo} />;
}

function CommentBody({
  data: item,
  bugPrefix,
  repo,
}: {
  data: CommentData;
  bugPrefix: string;
  repo: string | null;
}) {
  const { user } = useAuth();
  const [editing, setEditing] = useState(false);
  const [editValue, setEditValue] = useState(item.message ?? "");

  const [editComment, { loading }] = useMutation(BUG_EDIT_COMMENT_MUTATION, {
    refetchQueries: [{ query: BugDetailDocument, variables: { ref: repo, prefix: bugPrefix } }],
  });

  function handleSave() {
    if (editValue.trim() === (item.message ?? "").trim()) {
      setEditing(false);
      return;
    }
    void editComment({
      variables: { input: { targetPrefix: item.id, message: editValue.trim(), repoRef: repo } },
    }).then(() => setEditing(false));
  }

  function handleCancel() {
    setEditValue(item.message ?? "");
    setEditing(false);
  }

  const canEdit = user !== null && user.id === item.author.id;

  return (
    <CommentCard.Root>
      <CommentCard.AuthorAvatar author={item.author} />
      <CommentCard.Card>
        <CommentCard.CardHeader>
          <Link
            to="/$repo/user/$id"
            params={{ repo: repo!, id: item.author.humanId }}
            search={{ status: "open" as const, after: "" }}
            className="text-foreground font-medium hover:underline"
          >
            {item.author.displayName}
          </Link>
          <span className="text-muted-foreground">
            {formatDistanceToNow(new Date(item.createdAt), { addSuffix: true })}
          </span>
          {item.edited && !editing && <span className="text-muted-foreground text-xs">edited</span>}
          {canEdit && !editing && (
            <button
              onClick={() => setEditing(true)}
              className="text-muted-foreground hover:bg-muted hover:text-foreground ml-auto rounded-sm px-1.5 py-0.5 text-xs"
            >
              Edit
            </button>
          )}
        </CommentCard.CardHeader>

        {editing ? (
          <div className="space-y-2 p-3">
            <Textarea
              value={editValue}
              onChange={(e) => setEditValue(e.target.value)}
              className="min-h-24 font-mono text-sm"
              autoFocus
              onKeyDown={(e) => {
                if (e.key === "Escape") handleCancel();
                if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
                  e.preventDefault();
                  handleSave();
                }
              }}
            />
            <div className="flex gap-2">
              <Button size="sm" onClick={handleSave} disabled={loading}>
                {loading ? "Saving…" : "Save"}
              </Button>
              <Button size="sm" variant="ghost" onClick={handleCancel} disabled={loading}>
                Cancel
              </Button>
            </div>
          </div>
        ) : (
          <CommentCard.CardBody>
            {item.message ? (
              <Markdown content={item.message} />
            ) : (
              <p className="text-muted-foreground text-sm italic">No description provided.</p>
            )}
          </CommentCard.CardBody>
        )}
      </CommentCard.Card>
    </CommentCard.Root>
  );
}

// ── Inline events ────────────────────────────────────────────────────────────

type LabelChangeNode = Extract<TimelineNode, { __typename: "BugLabelChangeTimelineItem" }>;
type StatusChangeNode = Extract<TimelineNode, { __typename: "BugSetStatusTimelineItem" }>;
type TitleChangeNode = Extract<TimelineNode, { __typename: "BugSetTitleTimelineItem" }>;

function EventRow({ icon, children }: { icon: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="text-muted-foreground flex items-center gap-3 pl-2 text-sm">
      <span className="flex size-8 shrink-0 items-center justify-center">{icon}</span>
      {children}
    </div>
  );
}

function LabelChangeItem({ item, repo }: { item: LabelChangeNode; repo: string | null }) {
  const data = useFragment(LABEL_CHANGE_FRAGMENT, item);
  return (
    <EventRow icon={<Tag className="size-4" />}>
      <span>
        <Link
          to="/$repo/user/$id"
          params={{ repo: repo!, id: data.author.humanId }}
          search={{ status: "open" as const, after: "" }}
          className="text-foreground font-medium hover:underline"
        >
          {data.author.displayName}
        </Link>{" "}
        {data.added.length > 0 && (
          <>
            added{" "}
            {data.added.map((l, i) => (
              <LabelBadge key={i} label={l} />
            ))}{" "}
          </>
        )}
        {data.removed.length > 0 && (
          <>
            removed{" "}
            {data.removed.map((l, i) => (
              <LabelBadge key={i} label={l} />
            ))}{" "}
          </>
        )}
        {formatDistanceToNow(new Date(data.date), { addSuffix: true })}
      </span>
    </EventRow>
  );
}

function StatusChangeItem({ item, repo }: { item: StatusChangeNode; repo: string | null }) {
  const data = useFragment(STATUS_CHANGE_FRAGMENT, item);
  const isOpen = data.status === Status.Open;
  return (
    <EventRow
      icon={
        isOpen ? (
          <CircleDot className="size-4 text-green-600 dark:text-green-400" />
        ) : (
          <GitPullRequestClosed className="size-4 text-purple-600 dark:text-purple-400" />
        )
      }
    >
      <span>
        <Link
          to="/$repo/user/$id"
          params={{ repo: repo!, id: data.author.humanId }}
          search={{ status: "open" as const, after: "" }}
          className="text-foreground font-medium hover:underline"
        >
          {data.author.displayName}
        </Link>{" "}
        {isOpen ? "reopened" : "closed"} this{" "}
        {formatDistanceToNow(new Date(data.date), { addSuffix: true })}
      </span>
    </EventRow>
  );
}

function TitleChangeItem({ item, repo }: { item: TitleChangeNode; repo: string | null }) {
  const data = useFragment(TITLE_CHANGE_FRAGMENT, item);
  return (
    <EventRow icon={<Pencil className="size-4" />}>
      <span>
        <Link
          to="/$repo/user/$id"
          params={{ repo: repo!, id: data.author.humanId }}
          search={{ status: "open" as const, after: "" }}
          className="text-foreground font-medium hover:underline"
        >
          {data.author.displayName}
        </Link>{" "}
        changed the title from <span className="line-through">{data.was}</span> to{" "}
        <span className="text-foreground font-medium">{data.title}</span>{" "}
        {formatDistanceToNow(new Date(data.date), { addSuffix: true })}
      </span>
    </EventRow>
  );
}
