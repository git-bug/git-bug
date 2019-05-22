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
  bold: {
    fontWeight: 'bold',
  },
}));

function SetTitle({ op }) {
  const classes = useStyles();
  return (
    <div className={classes.main}>
      <Author author={op.author} className={classes.bold} />
      <span> changed the title from </span>
      <span className={classes.bold}>{op.was}</span>
      <span> to </span>
      <span className={classes.bold}>{op.title}</span>
      <Date date={op.date} />
    </div>
  );
}

SetTitle.fragment = gql`
  fragment SetTitle on TimelineItem {
    ... on SetTitleTimelineItem {
      date
      ...authored
      title
      was
    }
  }

  ${Author.fragment}
`;

export default SetTitle;
