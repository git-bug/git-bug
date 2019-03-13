import { withStyles } from '@material-ui/core/styles';
import gql from 'graphql-tag';
import React from 'react';
import Author from '../Author';
import Date from '../Date';

const styles = theme => ({
  main: {
    ...theme.typography.body2,
  },
  bold: {
    fontWeight: 'bold',
  },
});

const SetTitle = ({ op, classes }) => {
  return (
    <div className={classes.main}>
      <Author author={op.author} bold />
      <span> changed the title from </span>
      <span className={classes.bold}>{op.was}</span>
      <span> to </span>
      <span className={classes.bold}>{op.title}</span>
      <Date date={op.date} />
    </div>
  );
};

SetTitle.fragment = gql`
  fragment SetTitle on TimelineItem {
    ... on SetTitleTimelineItem {
      date
      author {
        name
        email
        displayName
      }
      title
      was
    }
  }
`;

export default withStyles(styles)(SetTitle);
