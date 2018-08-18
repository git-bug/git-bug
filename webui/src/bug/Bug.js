import { withStyles } from '@material-ui/core/styles';
import Typography from '@material-ui/core/Typography/Typography';
import gql from 'graphql-tag';
import React from 'react';
import Author from '../Author';
import Date from '../Date';
import TimelineQuery from './TimelineQuery';
import Label from '../Label';

const styles = theme => ({
  main: {
    maxWidth: 600,
    margin: 'auto',
    marginTop: theme.spacing.unit * 4,
  },
  header: {},
  title: {
    ...theme.typography.headline,
  },
  id: {
    ...theme.typography.subheading,
    marginLeft: 15,
  },
  container: {
    display: 'flex',
    marginBottom: 30,
  },
  timeline: {
    width: '70%',
    marginTop: 20,
    marginRight: 20,
  },
  sidebar: {
    width: '30%',
  },
  labelList: {
    listStyle: 'none',
    padding: 0,
    margin: 0,
  },
  label: {
    margin: '4px 0',
    '& > *': {
      display: 'block',
    },
  },
});

const Bug = ({ bug, classes }) => (
  <main className={classes.main}>
    <div className={classes.header}>
      <span className={classes.title}>{bug.title}</span>
      <span className={classes.id}>{bug.humanId}</span>

      <Typography color={'textSecondary'}>
        <Author author={bug.author} />
        <span> opened this bug </span>
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
              <Label label={l} key={l} />
            </li>
          ))}
        </ul>
      </div>
    </div>
  </main>
);

Bug.fragment = gql`
  fragment Bug on Bug {
    id
    humanId
    status
    title
    labels
    createdAt
    author {
      email
      name
    }
  }
`;

export default withStyles(styles)(Bug);
