import React, { useState } from 'react';

import {
  Button,
  fade,
  makeStyles,
  TextField,
  Typography,
} from '@material-ui/core';

import { TimelineDocument } from '../../pages/bug/TimelineQuery.generated';
import Author from 'src/components/Author';
import Date from 'src/components/Date';
import { BugFragment } from 'src/pages/bug/Bug.generated';

import { useSetTitleMutation } from './SetTitle.generated';

/**
 * Css in JS styles
 */
const useStyles = makeStyles((theme) => ({
  header: {
    display: 'flex',
    flexDirection: 'column',
  },
  headerTitle: {
    display: 'flex',
    flexDirection: 'row',
    justifyContent: 'space-between',
  },
  readOnlyTitle: {
    ...theme.typography.h5,
  },
  readOnlyId: {
    ...theme.typography.subtitle1,
    marginLeft: theme.spacing(1),
  },
  editButtonContainer: {
    display: 'flex',
    flexDirection: 'row',
    justifyContent: 'flex-start',
    alignItems: 'center',
    minWidth: 200,
    marginLeft: theme.spacing(2),
  },
  titleInput: {
    borderRadius: theme.shape.borderRadius,
    borderColor: fade(theme.palette.primary.main, 0.2),
    borderStyle: 'solid',
    borderWidth: '1px',
    backgroundColor: fade(theme.palette.primary.main, 0.05),
    padding: theme.spacing(0, 0),
    minWidth: 336,
    transition: theme.transitions.create([
      'width',
      'borderColor',
      'backgroundColor',
    ]),
  },
}));

interface Props {
  bug: BugFragment;
}

/**
 * Component for bug title change
 * @param bug Selected bug in list page
 */
function BugTitleForm({ bug }: Props) {
  const [bugTitleEditable, setBugTitleEditable] = useState(false);
  const [setTitle, { loading, error }] = useSetTitleMutation();
  const [issueTitle, setIssueTitle] = useState(bug.title);
  const classes = useStyles();
  let issueTitleInput: any;

  function isFormValid() {
    if (issueTitleInput) {
      return issueTitleInput.value.length > 0 ? true : false;
    } else {
      return false;
    }
  }

  function submitNewTitle() {
    if (!isFormValid()) return;
    setTitle({
      variables: {
        input: {
          prefix: bug.humanId,
          title: issueTitleInput.value,
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
    }).then(() => setBugTitleEditable(false));
  }

  function cancelChange() {
    setIssueTitle(bug.title);
    setBugTitleEditable(false);
  }

  function editableBugTitle() {
    return (
      <form className={classes.headerTitle} onSubmit={submitNewTitle}>
        <TextField
          inputRef={(node) => {
            issueTitleInput = node;
          }}
          className={classes.titleInput}
          variant="outlined"
          fullWidth
          margin="dense"
          value={issueTitle}
          onChange={(event: any) => setIssueTitle(event.target.value)}
        />
        <div className={classes.editButtonContainer}>
          <Button
            size="small"
            variant="contained"
            type="submit"
            disabled={issueTitle.length === 0}
          >
            Save
          </Button>
          <Button size="small" onClick={() => cancelChange()}>
            Cancel
          </Button>
        </div>
      </form>
    );
  }

  function readonlyBugTitle() {
    return (
      <div className={classes.headerTitle}>
        <div>
          <span className={classes.readOnlyTitle}>{bug.title}</span>
          <span className={classes.readOnlyId}>{bug.humanId}</span>
        </div>
        <div className={classes.editButtonContainer}>
          <Button
            size="small"
            variant="contained"
            onClick={() => setBugTitleEditable(!bugTitleEditable)}
          >
            Edit
          </Button>
        </div>
      </div>
    );
  }

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error</div>;

  return (
    <div className={classes.header}>
      {bugTitleEditable ? editableBugTitle() : readonlyBugTitle()}
      <div className="classes.headerSubtitle">
        <Typography color={'textSecondary'}>
          <Author author={bug.author} />
          {' opened this bug '}
          <Date date={bug.createdAt} />
        </Typography>
      </div>
    </div>
  );
}

export default BugTitleForm;
