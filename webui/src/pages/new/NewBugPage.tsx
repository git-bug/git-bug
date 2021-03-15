import React, { FormEvent, useState } from 'react';
import { useHistory } from 'react-router-dom';

import { Button, Paper } from '@material-ui/core';
import { makeStyles, Theme } from '@material-ui/core/styles';

import BackToListButton from '../../components/BackToListButton/BackToListButton';
import BugTitleInput from '../../components/BugTitleForm/BugTitleInput';
import CommentInput from '../../components/CommentInput/CommentInput';

import { useNewBugMutation } from './NewBug.generated';

/**
 * Css in JS styles
 */
const useStyles = makeStyles((theme: Theme) => ({
  main: {
    maxWidth: 1200,
    margin: 'auto',
    marginTop: theme.spacing(4),
    marginBottom: theme.spacing(4),
    padding: theme.spacing(2),
  },
  container: {
    display: 'flex',
    marginBottom: theme.spacing(1),
    marginRight: theme.spacing(2),
    marginLeft: theme.spacing(2),
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
  },
  leftSidebar: {
    marginTop: theme.spacing(2),
    marginRight: theme.spacing(2),
  },
  rightSidebar: {
    marginTop: theme.spacing(2),
    flex: '0 0 200px',
  },
  timeline: {
    flex: 1,
    marginTop: theme.spacing(2),
    marginRight: theme.spacing(2),
    minWidth: 400,
    padding: theme.spacing(1),
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

  let issueTitleInput: any;
  let history = useHistory();

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
      const id = data.data?.newBug.bug.humanId;
      history.push('/bug/' + id);
    });
    issueTitleInput.value = '';
  }

  function isFormValid() {
    return issueTitle.length > 0 && issueComment.length > 0 ? true : false;
  }

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error</div>;

  return (
    <main className={classes.main}>
      <div className={classes.container}>
        <div className={classes.leftSidebar}>
          <BackToListButton />
        </div>
        <Paper className={classes.timeline}>
          <form className={classes.form} onSubmit={submitNewIssue}>
            <BugTitleInput
              inputRef={(node) => {
                issueTitleInput = node;
              }}
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
                Submit new issue
              </Button>
            </div>
          </form>
        </Paper>
        <div className={classes.rightSidebar}></div>
      </div>
    </main>
  );
}

export default NewBugPage;
