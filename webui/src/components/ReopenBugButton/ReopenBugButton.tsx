import React from 'react';

import Button from '@material-ui/core/Button';

import { BugFragment } from 'src/pages/bug/Bug.generated';
import { TimelineDocument } from 'src/pages/bug/TimelineQuery.generated';

import { useOpenBugMutation } from './OpenBug.generated';

interface Props {
  bug: BugFragment;
  disabled: boolean;
}

function ReopenBugButton({ bug, disabled }: Props) {
  const [openBug, { loading, error }] = useOpenBugMutation();

  function openBugAction() {
    openBug({
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
        onClick={() => openBugAction()}
        disabled={bug.status === 'OPEN' || disabled}
      >
        Reopen bug
      </Button>
    </div>
  );
}

export default ReopenBugButton;
