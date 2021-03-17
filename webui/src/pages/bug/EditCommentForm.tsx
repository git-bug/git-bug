import React, { useState, useRef } from 'react';

import Button from '@material-ui/core/Button';
import Paper from '@material-ui/core/Paper';
import { makeStyles, Theme } from '@material-ui/core/styles';

import CommentInput from '../../components/CommentInput/CommentInput';

import { BugFragment } from './Bug.generated';
import { useAddCommentMutation } from './CommentForm.generated';
import { AddCommentFragment } from './MessageCommentFragment.generated';
import { CreateFragment } from './MessageCreateFragment.generated';

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
  comment: AddCommentFragment | CreateFragment;
  onCancelClick?: () => void;
  onPostSubmit?: () => void;
};

function EditCommentForm({ bug, comment, onCancelClick, onPostSubmit }: Props) {
  const [addComment, { loading }] = useAddCommentMutation();
  const [issueComment, setIssueComment] = useState<string>(comment.message);
  const [inputProp, setInputProp] = useState<any>('');
  const classes = useStyles({ loading });
  const form = useRef<HTMLFormElement>(null);

  const submit = () => {
    console.log('submit: ' + issueComment);
    resetForm();
    if (onPostSubmit) onPostSubmit();
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

  function getCancelButton() {
    return (
      <Button onClick={onCancelClick} variant="contained">
        Cancel
      </Button>
    );
  }

  return (
    <Paper className={classes.container}>
      <form onSubmit={handleSubmit} ref={form}>
        <CommentInput
          inputProps={inputProp}
          loading={loading}
          onChange={(comment: string) => setIssueComment(comment)}
          inputText={comment.message}
        />
        <div className={classes.actions}>
          {onCancelClick ? getCancelButton() : ''}
          <Button
            className={classes.greenButton}
            variant="contained"
            color="primary"
            type="submit"
            disabled={loading || issueComment.length === 0}
          >
            Update Comment
          </Button>
        </div>
      </form>
    </Paper>
  );
}

export default EditCommentForm;
