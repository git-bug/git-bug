import React, { useState, useRef } from 'react';

import Button from '@material-ui/core/Button';
import Paper from '@material-ui/core/Paper';
import { makeStyles, Theme } from '@material-ui/core/styles';

import CommentInput from '../../layout/CommentInput/CommentInput';
import CloseBugButton from 'src/components/CloseBugButton/CloseBugButton';

import { useAddCommentMutation } from './CommentForm.generated';
import { TimelineDocument } from './TimelineQuery.generated';

type StyleProps = { loading: boolean };
const useStyles = makeStyles<Theme, StyleProps>((theme) => ({
  container: {
    margin: theme.spacing(2, 0),
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
  bugId: string;
};

function CommentForm({ bugId }: Props) {
  const [addComment, { loading }] = useAddCommentMutation();
  const [issueComment, setIssueComment] = useState('');
  const [inputProp, setInputProp] = useState<any>('');
  const classes = useStyles({ loading });
  const form = useRef<HTMLFormElement>(null);

  const submit = () => {
    addComment({
      variables: {
        input: {
          prefix: bugId,
          message: issueComment,
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

  return (
    <Paper className={classes.container}>
      <form onSubmit={handleSubmit} ref={form}>
        <CommentInput
          inputProps={inputProp}
          loading={loading}
          onChange={(comment: string) => setIssueComment(comment)}
        />
        <div className={classes.actions}>
          <CloseBugButton bugId={bugId} />
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
