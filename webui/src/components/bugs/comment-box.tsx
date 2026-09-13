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
  /** Current repo slug, passed as `ref` in refetch query variables. */
  ref_?: string | null;
}

// Write/preview comment form at the bottom of BugDetailPage. Also contains the
// Close / Reopen button. Hidden entirely in read-only mode (no logged-in user).
export function CommentBox({ bugPrefix, bugStatus, ref_ }: CommentBoxProps) {
  const { user } = useAuth();
  const [message, setMessage] = useState("");
  const [preview, setPreview] = useState(false);

  const refetchVars = { variables: { ref: ref_, prefix: bugPrefix } };
  const refetch = { refetchQueries: [{ query: BugDetailDocument, ...refetchVars }] };

  const [addComment, { loading: addingComment }] = useMutation(BUG_ADD_COMMENT_MUTATION, refetch);
  const [addAndClose, { loading: addingAndClosing }] = useMutation(
    BUG_ADD_COMMENT_AND_CLOSE_MUTATION,
    refetch,
  );
  const [addAndReopen, { loading: addingAndReopening }] = useMutation(
    BUG_ADD_COMMENT_AND_REOPEN_MUTATION,
    refetch,
  );
  const [statusClose, { loading: closing }] = useMutation(BUG_STATUS_CLOSE_MUTATION, refetch);
  const [statusOpen, { loading: reopening }] = useMutation(BUG_STATUS_OPEN_MUTATION, refetch);

  const isOpen = bugStatus === Status.Open;
  const busy = addingComment || addingAndClosing || addingAndReopening || closing || reopening;
  const hasMessage = message.trim().length > 0;

  async function handleComment() {
    await addComment({
      variables: { input: { prefix: bugPrefix, message: message.trim(), repoRef: ref_ } },
    });
    setMessage("");
    setPreview(false);
  }

  async function handleToggleStatus() {
    if (isOpen) {
      if (hasMessage) {
        await addAndClose({
          variables: { input: { prefix: bugPrefix, message: message.trim(), repoRef: ref_ } },
        });
      } else {
        await statusClose({ variables: { input: { prefix: bugPrefix, repoRef: ref_ } } });
      }
    } else {
      if (hasMessage) {
        await addAndReopen({
          variables: { input: { prefix: bugPrefix, message: message.trim(), repoRef: ref_ } },
        });
      } else {
        await statusOpen({ variables: { input: { prefix: bugPrefix, repoRef: ref_ } } });
      }
    }
    setMessage("");
    setPreview(false);
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
