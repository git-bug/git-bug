import { withStyles } from '@material-ui/core/styles';
import gql from 'graphql-tag';
import React from 'react';
import Author from '../Author';
import Date from '../Date';

const styles = theme => ({
  main: {
    ...theme.typography.body2,
  },
});

const SetStatus = ({ op, classes }) => {
  return (
    <div className={classes.main}>
      <Author author={op.author} bold />
      <span> {op.status.toLowerCase()} this</span>
      <Date date={op.date} />
    </div>
  );
};

SetStatus.fragment = gql`
  fragment SetStatus on TimelineItem {
    ... on SetStatusTimelineItem {
      date
      author {
        name
        email
        displayName
      }
      status
    }
  }
`;

export default withStyles(styles)(SetStatus);
