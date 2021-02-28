import React from 'react';

import Button from '@material-ui/core/Button';
import { makeStyles } from '@material-ui/core/styles';
import ArrowBackIcon from '@material-ui/icons/ArrowBack';

import BugTitleForm from 'src/components/BugTitleForm/BugTitleForm';
import IfLoggedIn from 'src/components/IfLoggedIn/IfLoggedIn';
import Label from 'src/components/Label';

import { BugFragment } from './Bug.generated';
import CommentForm from './CommentForm';
import TimelineQuery from './TimelineQuery';

/**
 * Css in JS Styles
 */
const useStyles = makeStyles((theme) => ({
  main: {
    maxWidth: 1000,
    margin: 'auto',
    marginTop: theme.spacing(4),
  },
  header: {
    marginLeft: theme.spacing(3) + 40,
    marginRight: theme.spacing(2),
  },
  container: {
    display: 'flex',
    marginBottom: theme.spacing(1),
    marginRight: theme.spacing(2),
    marginLeft: theme.spacing(2),
  },
  timeline: {
    flex: 1,
    marginTop: theme.spacing(2),
    marginRight: theme.spacing(2),
    minWidth: 400,
  },
  sidebar: {
    marginTop: theme.spacing(2),
    flex: '0 0 200px',
  },
  sidebarTitle: {
    fontWeight: 'bold',
  },
  labelList: {
    listStyle: 'none',
    padding: 0,
    margin: 0,
  },
  label: {
    marginTop: theme.spacing(1),
    marginBottom: theme.spacing(1),
    '& > *': {
      display: 'block',
    },
  },
  noLabel: {
    ...theme.typography.body2,
  },
  commentForm: {
    marginLeft: 48,
  },
  backButton: {
    position: 'sticky',
    marginTop: theme.spacing(1),
    top: '80px',
    backgroundColor: '#574142',
    color: '#fff',
    '&:hover': {
      backgroundColor: '#610B0B',
    },
  },
}));

type Props = {
  bug: BugFragment;
};

function Bug({ bug }: Props) {
  const classes = useStyles();

  return (
    <main className={classes.main}>
      <div className={classes.header}>
        <BugTitleForm bug={bug} />
      </div>

      <div className={classes.container}>
        <div className={classes.timeline}>
          <TimelineQuery id={bug.id} />
          <IfLoggedIn>
            {() => (
              <div className={classes.commentForm}>
                <CommentForm bug={bug} />
              </div>
            )}
          </IfLoggedIn>
        </div>
        <div className={classes.sidebar}>
          <span className={classes.sidebarTitle}>Labels</span>
          <ul className={classes.labelList}>
            {bug.labels.length === 0 && (
              <span className={classes.noLabel}>None yet</span>
            )}
            {bug.labels.map((l) => (
              <li className={classes.label} key={l.name}>
                <Label label={l} key={l.name} />
              </li>
            ))}
          </ul>
          <Button
            variant="contained"
            className={classes.backButton}
            aria-label="back"
            href="/"
          >
            <ArrowBackIcon />
            Back to List
          </Button>
        </div>
      </div>
    </main>
  );
}

export default Bug;
