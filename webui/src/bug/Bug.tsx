import { makeStyles } from '@material-ui/core/styles';
import Typography from '@material-ui/core/Typography/Typography';
import React from 'react';
import Author from '../Author';
import Date from '../Date';
import TimelineQuery from './TimelineQuery';
import Label from '../Label';
import { BugFragment } from './Bug.generated';

const useStyles = makeStyles(theme => ({
  main: {
    maxWidth: 800,
    margin: 'auto',
    marginTop: theme.spacing(4),
  },
  header: {
    marginLeft: theme.spacing(1) + 40,
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
  },
  timeline: {
    flex: 1,
    marginTop: theme.spacing(2),
    marginRight: theme.spacing(2),
    minWidth: 0,
  },
  sidebar: {
    marginTop: theme.spacing(2),
    flex: '0 0 200px',
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
}));

type Props = {
  bug: BugFragment
};

function Bug({ bug }: Props) {
  const classes = useStyles();
  return (
    <main className={classes.main}>
      <div className={classes.header}>
        <span className={classes.title}>{bug.title}</span>
        <span className={classes.id}>{bug.humanId}</span>

        <Typography color={'textSecondary'}>
          <Author author={bug.author} />
          {' opened this bug '}
          <Date date={bug.createdAt} />
        </Typography>
      </div>

      <div className={classes.container}>
        <div className={classes.timeline}>
          <TimelineQuery id={bug.id} />
        </div>
        <div className={classes.sidebar}>
          <Typography variant={'subtitle1'}>Labels</Typography>
          <ul className={classes.labelList}>
            {bug.labels.map(l => (
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
