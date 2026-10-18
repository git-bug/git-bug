import { useMutation } from "@apollo/client/react";
import type { ResultOf } from "@graphql-typed-document-node/core";
import { Link } from "@tanstack/react-router";
import { formatDistanceToNow } from "date-fns";
import { Tag, GitPullRequestClosed, Pencil, CircleDot } from "lucide-react";
import { Fragment, useCallback, useEffect, useState } from "react";

import { Status } from "@/__generated__/enums";
import { useFragment, type FragmentType } from "@/__generated__/fragment-masking";
import { graphql } from "@/__generated__/gql";
import { BugDetailDocument } from "@/__generated__/graphql";
import { Markdown } from "@/components/content/markdown";
import * as CommentCard from "@/components/shared/comment-card";
import { LabelBadge } from "@/components/shared/label-badge";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { toastManager } from "@/components/ui/toast";
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
      ... on Authored {
        author {
          id
          displayName
        }
      }
      ... on BugCreateTimelineItem {
        history {
          author {
            id
            displayName
          }
        }
        ...BugCreateCommentFields
      }
      ... on BugAddCommentTimelineItem {
        history {
          author {
            id
            displayName
          }
        }
        ...BugAddCommentFields
      }
      ... on BugLabelChangeTimelineItem {
        ...LabelChangeFields
      }
      ... on BugSetStatusTimelineItem {
        status
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
  timeline: FragmentType<typeof TIMELINE_ITEMS_FRAGMENT>;
}

// Ordered sequence of events on a bug: comments (create and add-comment) and
// inline events (label changes, status changes, title edits). Comment items
// support inline editing for the logged-in user.  The changes others make
// while it is shown are announced by a toast and highlighted; mount it with
// the bug id as key, so that another bug doesn't count as changes.
export function Timeline({ repo, timeline }: TimelineProps) {
  const data = useFragment(TIMELINE_ITEMS_FRAGMENT, timeline);
  const highlights = useTimelineChanges(data.nodes);

  return (
    <div className="space-y-4">
      {data.nodes.map((item) => (
        <div key={item.id} id={timelineItemId(item.id)} className="relative">
          <TimelineItem item={item} repo={repo} />
          {highlights.has(item.id) && (
            // keyed by the times played, so that replaying restarts the animation
            <div
              key={highlights.get(item.id)}
              className="animate-highlight pointer-events-none absolute -inset-1 rounded-md"
            />
          )}
        </div>
      ))}
    </div>
  );
}

function TimelineItem({ item, repo }: { item: TimelineNode; repo: string | null }) {
  switch (item.__typename) {
    case "BugCreateTimelineItem":
      return <CreateCommentItem item={item} repo={repo} />;
    case "BugAddCommentTimelineItem":
      return <AddCommentItem item={item} repo={repo} />;
    case "BugLabelChangeTimelineItem":
      return <LabelChangeItem item={item} repo={repo} />;
    case "BugSetStatusTimelineItem":
      return <StatusChangeItem item={item} repo={repo} />;
    case "BugSetTitleTimelineItem":
      return <TitleChangeItem item={item} repo={repo} />;
    default:
      return null;
  }
}

// ── Changes ──────────────────────────────────────────────────────────────────

const CHANGES_TOAST_ID = "timeline-changes";

function timelineItemId(id: string) {
  return `timeline-${id}`;
}

// Identifies a state of an item: a comment changes with each version in its
// history, the other items never change.
function changeKey(item: TimelineNode) {
  switch (item.__typename) {
    case "BugCreateTimelineItem":
    case "BugAddCommentTimelineItem":
      return `${item.id}#${item.history.length}`;
    default:
      return item.id;
  }
}

// Who made the latest state of an item: for a comment, the author of its
// latest version, who may not have written the comment.
function changeAuthor(item: TimelineNode) {
  switch (item.__typename) {
    case "BugCreateTimelineItem":
    case "BugAddCommentTimelineItem":
      return item.history.at(-1)?.author ?? item.author;
    default:
      return item.author;
  }
}

function describeChange(item: TimelineNode, edited: boolean) {
  const name = changeAuthor(item).displayName;
  switch (item.__typename) {
    case "BugCreateTimelineItem":
    case "BugAddCommentTimelineItem":
      return edited ? `${name} edited a comment` : `${name} commented`;
    case "BugLabelChangeTimelineItem":
      return `${name} changed the labels`;
    case "BugSetStatusTimelineItem":
      return `${name} ${item.status === Status.Open ? "reopened" : "closed"} this issue`;
    case "BugSetTitleTimelineItem":
      return `${name} changed the title`;
    default:
      return `${name} changed this issue`;
  }
}

// Finds the items others added or edited since the previous render, announces
// them with a toast, and returns the changed items to highlight, as the times
// their highlight played by item id.  The highlight plays on each change, and
// again when the toast's action brings the items into view.  The user's own
// changes go unannounced.
function useTimelineChanges(nodes: TimelineNode[]) {
  const { user } = useAuth();
  const keys = nodes.map(changeKey);
  const signature = keys.join(" ");

  // compared by signature: the nodes may be new objects for the same content
  const [shown, setShown] = useState({
    signature,
    keys: new Set(keys),
    ids: new Set(nodes.map((n) => n.id)),
  });
  const [highlights, setHighlights] = useState<ReadonlyMap<string, number>>(new Map());
  const play = useCallback((ids: string[]) => {
    setHighlights((prev) => {
      const next = new Map(prev);
      for (const id of ids) next.set(id, (prev.get(id) ?? 0) + 1);
      return next;
    });
  }, []);
  const [announced, setAnnounced] = useState<{ ids: string[]; messages: string[] } | null>(null);
  if (signature !== shown.signature) {
    const changed = nodes.filter(
      (n) => !shown.keys.has(changeKey(n)) && changeAuthor(n).id !== user?.id,
    );
    setShown({ signature, keys: new Set(keys), ids: new Set(nodes.map((n) => n.id)) });
    if (changed.length > 0) {
      play(changed.map((n) => n.id));
      setAnnounced({
        ids: changed.map((n) => n.id),
        messages: changed.map((n) => describeChange(n, shown.ids.has(n.id))),
      });
    }
  }

  useEffect(() => {
    if (announced) {
      const { ids, messages } = announced;
      toastManager.add({
        id: CHANGES_TOAST_ID,
        title: messages.length === 1 ? messages[0] : `${messages.length} changes on this issue`,
        description: messages.length > 1 ? messages.join(", ") : undefined,
        actionProps: {
          children: "Show",
          onClick: () => {
            document
              .getElementById(timelineItemId(ids[0]!))
              ?.scrollIntoView({ behavior: "smooth", block: "center" });
            play(ids);
          },
        },
      });
    }
  }, [announced, play]);
  useEffect(
    () => () => {
      toastManager.close(CHANGES_TOAST_ID);
    },
    [],
  );

  return highlights;
}

// ── Comment items ────────────────────────────────────────────────────────────

type CreateNode = Extract<TimelineNode, { __typename: "BugCreateTimelineItem" }>;
type AddCommentNode = Extract<TimelineNode, { __typename: "BugAddCommentTimelineItem" }>;
type CommentData =
  | ResultOf<typeof BUG_CREATE_COMMENT_FRAGMENT>
  | ResultOf<typeof BUG_ADD_COMMENT_FRAGMENT>;

function CreateCommentItem({ item, repo }: { item: CreateNode; repo: string | null }) {
  const data = useFragment(BUG_CREATE_COMMENT_FRAGMENT, item);
  return <CommentBody data={data} repo={repo} />;
}

function AddCommentItem({ item, repo }: { item: AddCommentNode; repo: string | null }) {
  const data = useFragment(BUG_ADD_COMMENT_FRAGMENT, item);
  return <CommentBody data={data} repo={repo} />;
}

function CommentBody({ data: item, repo }: { data: CommentData; repo: string | null }) {
  const { user } = useAuth();
  const [editing, setEditing] = useState(false);
  const [editValue, setEditValue] = useState(item.message ?? "");

  // on failure, the editor stays open with the text and the error
  const [editComment, { loading, error, reset }] = useMutation(BUG_EDIT_COMMENT_MUTATION, {
    refetchQueries: [BugDetailDocument],
    errorPolicy: "all",
  });

  function handleSave() {
    if (editValue.trim() === (item.message ?? "").trim()) {
      setEditing(false);
      return;
    }
    void editComment({
      variables: { input: { targetPrefix: item.id, message: editValue.trim(), repoRef: repo } },
    }).then((result) => {
      if (!result.error) setEditing(false);
    });
  }

  function handleCancel() {
    setEditValue(item.message ?? "");
    setEditing(false);
    reset();
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
              onClick={() => {
                // the message may have changed since the last edit (e.g. after a refetch)
                setEditValue(item.message ?? "");
                reset();
                setEditing(true);
              }}
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
            {error && (
              <p className="text-destructive text-sm">
                Failed to save the comment: {error.message}
              </p>
            )}
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
type LabelChangeData = ResultOf<typeof LABEL_CHANGE_FRAGMENT>;
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
            added <LabelList labels={data.added} />{" "}
          </>
        )}
        {data.added.length > 0 && data.removed.length > 0 && "and "}
        {data.removed.length > 0 && (
          <>
            removed <LabelList labels={data.removed} />{" "}
          </>
        )}
        {formatDistanceToNow(new Date(data.date), { addSuffix: true })}
      </span>
    </EventRow>
  );
}

// The labels as a phrase: "the [a] label", "the [a] and [b] labels", "the [a],
// [b] and [c] labels".
function LabelList({ labels }: { labels: LabelChangeData["added"] }) {
  return (
    <>
      the{" "}
      {labels.map((l, i) => (
        <Fragment key={i}>
          {i > 0 && (i === labels.length - 1 ? " and " : ", ")}
          <LabelBadge label={l} />
        </Fragment>
      ))}{" "}
      {labels.length === 1 ? "label" : "labels"}
    </>
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
