import { makeStyles } from '@material-ui/styles';
import React from 'react';
import Author from '../Author';
import Date from '../Date';

const useStyles = makeStyles(theme => ({
  main: {
    ...theme.typography.body1,
    marginLeft: theme.spacing(1) + 40,
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

export default SetStatus;
