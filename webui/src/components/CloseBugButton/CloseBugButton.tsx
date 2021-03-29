import React from 'react';

import Button from '@material-ui/core/Button';
import { makeStyles, Theme } from '@material-ui/core/styles';
import ErrorOutlineIcon from '@material-ui/icons/ErrorOutline';

import { BugFragment } from 'src/pages/bug/Bug.generated';
import { TimelineDocument } from 'src/pages/bug/TimelineQuery.generated';

import { useCloseBugMutation } from './CloseBug.generated';

const useStyles = makeStyles((theme: Theme) => ({
  closeIssueIcon: {
    color: theme.palette.secondary.dark,
    paddingTop: '0.1rem',
  },
}));

interface Props {
  bug: BugFragment;
  disabled: boolean;
}

function CloseBugButton({ bug, disabled }: Props) {
  const [closeBug, { loading, error }] = useCloseBugMutation();
  const classes = useStyles();

  function closeBugAction() {
    closeBug({
      variables: {
        input: {
          prefix: bug.id,
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
    });
  }

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error</div>;

  return (
    <div>
      <Button
        variant="contained"
        onClick={() => closeBugAction()}
        disabled={bug.status === 'CLOSED' || disabled}
        startIcon={<ErrorOutlineIcon className={classes.closeIssueIcon} />}
      >
        Close bug
      </Button>
    </div>
  );
}

export default CloseBugButton;
