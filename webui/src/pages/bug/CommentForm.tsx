import React, { useState, useRef } from 'react';

import Button from '@material-ui/core/Button';
import Paper from '@material-ui/core/Paper';
import { makeStyles, Theme } from '@material-ui/core/styles';

import CommentInput from '../../components/CommentInput/CommentInput';
import CloseBugButton from 'src/components/CloseBugButton/CloseBugButton';
import ReopenBugButton from 'src/components/ReopenBugButton/ReopenBugButton';

import { BugFragment } from './Bug.generated';
import { useAddCommentMutation } from './CommentForm.generated';
import { TimelineDocument } from './TimelineQuery.generated';

type StyleProps = { loading: boolean };
const useStyles = makeStyles<Theme, StyleProps>((theme) => ({
  container: {
    padding: theme.spacing(0, 2, 2, 2),
  },
  textarea: {},
  tabContent: {
    margin: theme.spacing(2, 0),
  },
  preview: {
    borderBottom: `solid 3px ${theme.palette.grey['200']}`,
    minHeight: '5rem',
  },
  actions: {
    display: 'flex',
    justifyContent: 'flex-end',
  },
  greenButton: {
    marginLeft: '8px',
    backgroundColor: '#2ea44fd9',
    color: '#fff',
    '&:hover': {
      backgroundColor: '#2ea44f',
    },
  },
}));

type Props = {
  bug: BugFragment;
};

function CommentForm({ bug }: Props) {
  const [addComment, { loading }] = useAddCommentMutation();
  const [issueComment, setIssueComment] = useState('');
  const [inputProp, setInputProp] = useState<any>('');
  const classes = useStyles({ loading });
  const form = useRef<HTMLFormElement>(null);

  const submit = () => {
    addComment({
      variables: {
        input: {
          prefix: bug.id,
          message: issueComment,
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
    }).then(() => resetForm());
  };

  function resetForm() {
    setInputProp({
      value: '',
    });
  }

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (issueComment.length > 0) submit();
  };

  function getCloseButton() {
    return <CloseBugButton bug={bug} disabled={issueComment.length > 0} />;
  }

  function getReopenButton() {
    return <ReopenBugButton bug={bug} disabled={issueComment.length > 0} />;
  }

  return (
    <Paper className={classes.container}>
      <form onSubmit={handleSubmit} ref={form}>
        <CommentInput
          inputProps={inputProp}
          loading={loading}
          onChange={(comment: string) => setIssueComment(comment)}
        />
        <div className={classes.actions}>
          {bug.status === 'OPEN' ? getCloseButton() : getReopenButton()}
          <Button
            className={classes.greenButton}
            variant="contained"
            color="primary"
            type="submit"
            disabled={loading || issueComment.length === 0}
          >
            Comment
          </Button>
        </div>
      </form>
    </Paper>
  );
}

export default CommentForm;
