import { withStyles } from '@material-ui/core/styles';
import Typography from '@material-ui/core/Typography';
import gql from 'graphql-tag';
import React from 'react';
import Author from '../Author';
import Date from '../Date';

const styles = theme => ({
  header: {
    ...theme.typography.body2,
    padding: '3px 3px 3px 6px',
    backgroundColor: '#f1f8ff',
    border: '1px solid #d1d5da',
    borderTopLeftRadius: 3,
    borderTopRightRadius: 3,
  },
  message: {
    borderLeft: '1px solid #d1d5da',
    borderRight: '1px solid #d1d5da',
    borderBottom: '1px solid #d1d5da',
    borderBottomLeftRadius: 3,
    borderBottomRightRadius: 3,
    backgroundColor: '#fff',
    minHeight: 50,
    padding: 5,
    whiteSpace: 'pre-wrap',
  },
});

const Message = ({ op, classes }) => (
  <div>
    <div className={classes.header}>
      <Author className={classes.author} author={op.author} bold />
      <span> commented </span>
      <Date date={op.createdAt} />
    </div>
    <div className={classes.message}>
      <Typography>{op.message}</Typography>
    </div>
  </div>
);

Message.createFragment = gql`
  fragment Create on TimelineItem {
    ... on CreateTimelineItem {
      createdAt
      author {
        name
        email
        displayName
      }
      message
    }
  }
`;

Message.commentFragment = gql`
  fragment AddComment on TimelineItem {
    ... on AddCommentTimelineItem {
      createdAt
      author {
        name
        email
        displayName
      }
      message
    }
  }
`;

export default withStyles(styles)(Message);
