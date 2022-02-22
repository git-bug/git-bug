import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';

import { BugFragment } from 'src/pages/bug/Bug.generated';
import { TimelineDocument } from 'src/pages/bug/TimelineQuery.generated';

import { useAddCommentAndCloseBugMutation } from './CloseBugWithComment.generated';

interface Props {
  bug: BugFragment;
  comment: string;
  postClick?: () => void;
}

function CloseBugWithCommentButton({ bug, comment, postClick }: Props) {
  const [addCommentAndCloseBug, { loading, error }] =
    useAddCommentAndCloseBugMutation();

  function addCommentAndCloseBugAction() {
    addCommentAndCloseBug({
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
        onClick={() => addCommentAndCloseBugAction()}
        startIcon={<ErrorOutlineIcon />}
      >
        Close bug with comment
      </Button>
    </div>
  );
}

export default CloseBugWithCommentButton;
