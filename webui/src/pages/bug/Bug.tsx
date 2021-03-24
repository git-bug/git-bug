import React from 'react';

import { makeStyles } from '@material-ui/core/styles';

import BugTitleForm from 'src/components/BugTitleForm/BugTitleForm';
import IfLoggedIn from 'src/components/IfLoggedIn/IfLoggedIn';
import Label from 'src/components/Label';

import { BugFragment } from './Bug.generated';
import CommentForm from './CommentForm';
import TimelineQuery from './TimelineQuery';
import LabelMenu from './labels/LabelMenu';

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
    marginRight: theme.spacing(2),
    marginLeft: theme.spacing(3) + 40,
  },
  title: {
    ...theme.typography.h5,
  },
  id: {
    ...theme.typography.subtitle1,
    marginLeft: theme.spacing(1),
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
  rightSidebar: {
    marginTop: theme.spacing(2),
    flex: '0 0 200px',
  },
  rightSidebarTitle: {
    fontWeight: 'bold',
  },
  labelList: {
    listStyle: 'none',
    padding: 0,
    margin: 0,
    display: 'flex',
    flexDirection: 'row',
    flexWrap: 'wrap',
  },
  label: {
    marginTop: theme.spacing(0.1),
    marginBottom: theme.spacing(0.1),
  },
  noLabel: {
    ...theme.typography.body2,
  },
  commentForm: {
    marginTop: theme.spacing(2),
    marginLeft: 48,
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
          <TimelineQuery bug={bug} />
          <IfLoggedIn>
            {() => (
              <div className={classes.commentForm}>
                <CommentForm bug={bug} />
              </div>
            )}
          </IfLoggedIn>
        </div>
        <div className={classes.rightSidebar}>
          <span className={classes.rightSidebarTitle}>
            <LabelMenu bug={bug} />
          </span>
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
        </div>
      </div>
    </main>
  );
}

export default Bug;
