import { Button, Paper } from '@mui/material';
import { Theme } from '@mui/material/styles';
import makeStyles from '@mui/styles/makeStyles';
import { FormEvent, useRef, useState } from 'react';
import { useNavigate } from 'react-router';

import BugTitleInput from '../../components/BugTitleForm/BugTitleInput';
import CommentInput from '../../components/CommentInput/CommentInput';

import { useNewBugMutation } from './NewBug.generated';

/**
 * Css in JS styles
 */
const useStyles = makeStyles((theme: Theme) => ({
  main: {
    maxWidth: 800,
    margin: 'auto',
    marginTop: theme.spacing(4),
    marginBottom: theme.spacing(4),
    padding: theme.spacing(2),
    overflow: 'hidden',
  },
  form: {
    display: 'flex',
    flexDirection: 'column',
  },
  actions: {
    display: 'flex',
    justifyContent: 'flex-end',
  },
  greenButton: {
    backgroundColor: theme.palette.success.main,
    color: theme.palette.success.contrastText,
    '&:hover': {
      backgroundColor: theme.palette.success.dark,
      color: theme.palette.primary.contrastText,
    },
  },
}));

/**
 * Form to create a new issue
 */
function NewBugPage() {
  const [newBug, { loading, error }] = useNewBugMutation();
  const [issueTitle, setIssueTitle] = useState('');
  const [issueComment, setIssueComment] = useState('');
  const classes = useStyles();

  const issueTitleInput = useRef<HTMLInputElement>(null);
  const navigate = useNavigate();

  function submitNewIssue(e: FormEvent) {
    e.preventDefault();
    if (!isFormValid()) return;
    newBug({
      variables: {
        input: {
          title: issueTitle,
          message: issueComment,
        },
      },
    }).then(function (data) {
      const id = data.data?.bugCreate.bug.id;
      navigate('/bug/' + id);
    });

    if (issueTitleInput.current) {
      issueTitleInput.current.value = '';
    }
  }

  function isFormValid() {
    return issueTitle.length > 0;
  }

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error</div>;

  return (
    <Paper className={classes.main}>
      <form className={classes.form} onSubmit={submitNewIssue}>
        <BugTitleInput
          inputRef={issueTitleInput}
          label="Title"
          variant="outlined"
          fullWidth
          margin="dense"
          onChange={(event: any) => setIssueTitle(event.target.value)}
        />
        <CommentInput
          loading={false}
          onChange={(comment: string) => setIssueComment(comment)}
        />
        <div className={classes.actions}>
          <Button
            className={classes.greenButton}
            variant="contained"
            type="submit"
            disabled={isFormValid() ? false : true}
          >
            Submit new bug
          </Button>
        </div>
      </form>
    </Paper>
  );
}

export default NewBugPage;
