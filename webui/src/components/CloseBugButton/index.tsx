import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';

import { BugFragment } from 'src/pages/bug/Bug.generated';
import { TimelineDocument } from 'src/pages/bug/TimelineQuery.generated';

import { useCloseBugMutation } from './CloseBug.generated';

interface Props {
  bug: BugFragment;
  disabled?: boolean;
}

function CloseBugButton({ bug, disabled }: Props) {
  const [closeBug, { loading, error }] = useCloseBugMutation();

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

  if (loading) return <CircularProgress />;
  if (error) return <div>Error</div>;

  return (
    <div>
      <Button
        variant="contained"
        onClick={() => closeBugAction()}
        disabled={bug.status === 'CLOSED' || disabled}
        startIcon={<ErrorOutlineIcon />}
      >
        Close bug
      </Button>
    </div>
  );
}

export default CloseBugButton;
