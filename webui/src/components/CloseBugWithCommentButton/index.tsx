import React from 'react';

import Button from '@material-ui/core/Button';
import CircularProgress from '@material-ui/core/CircularProgress';
import { makeStyles, Theme } from '@material-ui/core/styles';
import ErrorOutlineIcon from '@material-ui/icons/ErrorOutline';

import { BugFragment } from 'src/pages/bug/Bug.generated';
import { TimelineDocument } from 'src/pages/bug/TimelineQuery.generated';

import { useAddCommentAndCloseBugMutation } from './CloseBugWithComment.generated';

const useStyles = makeStyles((theme: Theme) => ({
  closeIssueIcon: {
    color: theme.palette.secondary.dark,
    paddingTop: '0.1rem',
  },
}));

interface Props {
  bug: BugFragment;
  comment: string;
  postClick?: () => void;
}

function CloseBugWithCommentButton({ bug, comment, postClick }: Props) {
  const [addCommentAndCloseBug, { loading, error }] =
    useAddCommentAndCloseBugMutation();
  const classes = useStyles();

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
        startIcon={<ErrorOutlineIcon className={classes.closeIssueIcon} />}
      >
        Close bug with comment
      </Button>
    </div>
  );
}

export default CloseBugWithCommentButton;
