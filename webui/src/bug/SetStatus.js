import { makeStyles } from '@material-ui/styles';
import gql from 'graphql-tag';
import React from 'react';
import Author from '../Author';
import Date from '../Date';

const useStyles = makeStyles(theme => ({
  main: {
    ...theme.typography.body2,
    marginLeft: theme.spacing.unit + 40,
  },
}));

function SetStatus({ op }) {
  const classes = useStyles();
  return (
    <div className={classes.main}>
      <Author author={op.author} bold />
      <span> {op.status.toLowerCase()} this</span>
      <Date date={op.date} />
    </div>
  );
}

SetStatus.fragment = gql`
  fragment SetStatus on TimelineItem {
    ... on SetStatusTimelineItem {
      date
      ...authored
      status
    }
  }

  ${Author.fragment}
`;

export default SetStatus;
