import type { ErrorLike } from "@apollo/client";
import { useMutation } from "@apollo/client/react";
import { useState } from "react";

import { Status } from "@/__generated__/enums";
import { graphql } from "@/__generated__/gql";
import { BugDetailDocument } from "@/__generated__/graphql";
import { Markdown } from "@/components/content/markdown";
import * as CommentCard from "@/components/shared/comment-card";
import * as WritePreview from "@/components/shared/write-preview";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { useAuth } from "@/lib/auth";
import { useDraft } from "@/lib/draft";

const BUG_ADD_COMMENT_MUTATION = graphql(`
  mutation BugAddComment($input: BugAddCommentInput!) {
    bugAddComment(input: $input) {
      bug {
        id
      }
    }
  }
`);

const BUG_ADD_COMMENT_AND_CLOSE_MUTATION = graphql(`
  mutation BugAddCommentAndClose($input: BugAddCommentAndCloseInput!) {
    bugAddCommentAndClose(input: $input) {
      bug {
        id
      }
    }
  }
`);

const BUG_ADD_COMMENT_AND_REOPEN_MUTATION = graphql(`
  mutation BugAddCommentAndReopen($input: BugAddCommentAndReopenInput!) {
    bugAddCommentAndReopen(input: $input) {
      bug {
        id
      }
    }
  }
`);

const BUG_STATUS_OPEN_MUTATION = graphql(`
  mutation BugStatusOpen($input: BugStatusOpenInput!) {
    bugStatusOpen(input: $input) {
      bug {
        id
        status
      }
    }
  }
`);

const BUG_STATUS_CLOSE_MUTATION = graphql(`
  mutation BugStatusClose($input: BugStatusCloseInput!) {
    bugStatusClose(input: $input) {
      bug {
        id
        status
      }
    }
  }
`);

interface CommentBoxProps {
  bugPrefix: string;
  bugStatus: Status;
  /** Current repo slug, passed as `repoRef` to the mutations. */
  repo?: string | null;
}

// Write/preview comment form at the bottom of BugDetailPage. Also contains the
// Close / Reopen button. Hidden entirely in read-only mode (no logged-in user).
export function CommentBox({ bugPrefix, bugStatus, repo }: CommentBoxProps) {
  const { user } = useAuth();
  // kept until sent, even across a reload
  const [message, setMessage] = useDraft(`${repo ?? "_"}:${bugPrefix}:comment`);
  const [preview, setPreview] = useState(false);

  // The live updates refetch the bug too, but not without the websocket: the
  // user's own changes must show regardless.  A failure resolves with its
  // error rather than throwing, for submit to show it.
  const options = { refetchQueries: [BugDetailDocument], errorPolicy: "all" as const };

  const [addComment, { loading: addingComment }] = useMutation(BUG_ADD_COMMENT_MUTATION, options);
  const [addAndClose, { loading: addingAndClosing }] = useMutation(
    BUG_ADD_COMMENT_AND_CLOSE_MUTATION,
    options,
  );
  const [addAndReopen, { loading: addingAndReopening }] = useMutation(
    BUG_ADD_COMMENT_AND_REOPEN_MUTATION,
    options,
  );
  const [statusClose, { loading: closing }] = useMutation(BUG_STATUS_CLOSE_MUTATION, options);
  const [statusOpen, { loading: reopening }] = useMutation(BUG_STATUS_OPEN_MUTATION, options);

  const isOpen = bugStatus === Status.Open;
  const busy = addingComment || addingAndClosing || addingAndReopening || closing || reopening;
  const hasMessage = message.trim().length > 0;

  // the error of the last attempt: on failure, the message stays to retry
  const [error, setError] = useState<ErrorLike | null>(null);

  async function submit(mutation: Promise<{ error?: ErrorLike }>) {
    const result = await mutation;
    setError(result.error ?? null);
    if (result.error) return;
    setMessage("");
    setPreview(false);
  }

  async function handleComment() {
    await submit(
      addComment({
        variables: { input: { prefix: bugPrefix, message: message.trim(), repoRef: repo } },
      }),
    );
  }

  async function handleToggleStatus() {
    const withMessage = {
      variables: { input: { prefix: bugPrefix, message: message.trim(), repoRef: repo } },
    };
    const withoutMessage = { variables: { input: { prefix: bugPrefix, repoRef: repo } } };
    if (isOpen) {
      await submit(hasMessage ? addAndClose(withMessage) : statusClose(withoutMessage));
    } else {
      await submit(hasMessage ? addAndReopen(withMessage) : statusOpen(withoutMessage));
    }
  }

  if (!user) return null;

  return (
    <CommentCard.Root>
      <CommentCard.AuthorAvatar author={user} />
      <CommentCard.Card>
        <WritePreview.Root hasContent={hasMessage} preview={preview} onPreviewChange={setPreview}>
          <WritePreview.Tabs className="border-border border-b px-4 py-2" />
          <WritePreview.WriteSlot>
            <Textarea
              placeholder="Leave a comment…"
              className="min-h-[120px] rounded-none border-0 shadow-none focus-visible:ring-0"
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              disabled={busy}
            />
          </WritePreview.WriteSlot>
          <WritePreview.PreviewSlot>
            <div className="min-h-[120px] px-4 py-3">
              <Markdown content={message} />
            </div>
          </WritePreview.PreviewSlot>
        </WritePreview.Root>

        <div className="border-border flex items-center justify-end gap-2 border-t px-3 py-2">
          {error && (
            <p className="text-destructive mr-auto text-sm">Failed to save: {error.message}</p>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              void handleToggleStatus();
            }}
            disabled={busy}
            className="min-w-[7.5rem]"
          >
            {isOpen ? "Close issue" : "Reopen issue"}
          </Button>
          <Button
            size="sm"
            onClick={() => {
              void handleComment();
            }}
            disabled={!hasMessage || busy}
          >
            Comment
          </Button>
        </div>
      </CommentCard.Card>
    </CommentCard.Root>
  );
}
