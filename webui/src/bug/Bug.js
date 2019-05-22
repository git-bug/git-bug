import { makeStyles } from '@material-ui/styles';
import Typography from '@material-ui/core/Typography/Typography';
import gql from 'graphql-tag';
import React from 'react';
import Author from '../Author';
import Date from '../Date';
import TimelineQuery from './TimelineQuery';
import Label from '../Label';

const useStyles = makeStyles(theme => ({
  main: {
    maxWidth: 800,
    margin: 'auto',
    marginTop: theme.spacing.unit * 4,
  },
  header: {
    marginLeft: theme.spacing.unit + 40,
  },
  title: {
    ...theme.typography.headline,
  },
  id: {
    ...theme.typography.subheading,
    marginLeft: theme.spacing.unit,
  },
  container: {
    display: 'flex',
    marginBottom: theme.spacing.unit,
  },
  timeline: {
    flex: 1,
    marginTop: theme.spacing.unit * 2,
    marginRight: theme.spacing.unit * 2,
  },
  sidebar: {
    marginTop: theme.spacing.unit * 2,
    flex: '0 0 200px',
  },
  labelList: {
    listStyle: 'none',
    padding: 0,
    margin: 0,
  },
  label: {
    marginTop: theme.spacing.unit,
    marginBottom: theme.spacing.unit,
    '& > *': {
      display: 'block',
    },
  },
}));

function Bug({ bug }) {
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
          <Typography variant={'subheading'}>Labels</Typography>
          <ul className={classes.labelList}>
            {bug.labels.map(l => (
              <li className={classes.label}>
                <Label label={l} key={l.name} />
              </li>
            ))}
          </ul>
        </div>
      </div>
    </main>
  );
}

Bug.fragment = gql`
  fragment Bug on Bug {
    id
    humanId
    status
    title
    labels {
      ...Label
    }
    createdAt
    author {
      email
      name
      displayName
    }
  }
  ${Label.fragment}
`;

export default Bug;
