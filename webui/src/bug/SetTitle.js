import { makeStyles } from '@material-ui/styles';
import React from 'react';
import Author from '../Author';
import Date from '../Date';

const useStyles = makeStyles(theme => ({
  main: {
    ...theme.typography.body1,
    marginLeft: theme.spacing(1) + 40,
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

export default SetTitle;
