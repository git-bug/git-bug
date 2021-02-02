import React from 'react';

import Button from '@material-ui/core/Button';

import { TimelineDocument } from 'src/pages/bug/TimelineQuery.generated';

import { useCloseBugMutation } from './CloseBug.generated';

interface Props {
  bugId: string;
}

function CloseBugButton({ bugId }: Props) {
  const [closeBug, { loading, error }] = useCloseBugMutation();

  function closeBugAction() {
    closeBug({
      variables: {
        input: {
          prefix: bugId,
        },
      },
      refetchQueries: [
        // TODO: update the cache instead of refetching
        {
          query: TimelineDocument,
          variables: {
            id: bugId,
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
      <Button variant="contained" onClick={() => closeBugAction()}>
        Close issue
      </Button>
    </div>
  );
}

export default CloseBugButton;
