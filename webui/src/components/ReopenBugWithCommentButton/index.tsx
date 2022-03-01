import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';

import { BugFragment } from 'src/pages/bug/Bug.generated';
import { TimelineDocument } from 'src/pages/bug/TimelineQuery.generated';

import { useAddCommentAndReopenBugMutation } from './ReopenBugWithComment.generated';

interface Props {
  bug: BugFragment;
  comment: string;
  postClick?: () => void;
}

function ReopenBugWithCommentButton({ bug, comment, postClick }: Props) {
  const [addCommentAndReopenBug, { loading, error }] =
    useAddCommentAndReopenBugMutation();

  function addCommentAndReopenBugAction() {
    addCommentAndReopenBug({
      variables: {
        input: {
          prefix: bug.id,
          message: comment,
        },
      },
      refetchQueries: [
        // TODO: update the cache instead of refetching
        {
          query: TimelineDocument,
          variables: {
            id: bug.id,
            first: 100,
          },
        },
      ],
      awaitRefetchQueries: true,
    }).then(() => {
      if (postClick) {
        postClick();
      }
    });
  }

  if (loading) return <CircularProgress />;
  if (error) return <div>Error</div>;

  return (
    <div>
      <Button
        variant="contained"
        type="submit"
        onClick={() => addCommentAndReopenBugAction()}
      >
        Reopen bug with comment
      </Button>
    </div>
  );
}

export default ReopenBugWithCommentButton;
