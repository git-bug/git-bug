import { makeStyles } from '@material-ui/styles';
import React from 'react';
import Author from '../Author';
import Date from '../Date';
import Label from '../Label';

const useStyles = makeStyles(theme => ({
  main: {
    ...theme.typography.body1,
    marginLeft: theme.spacing(1) + 40,
  },
  author: {
    fontWeight: 'bold',
  },
}));

function LabelChange({ op }) {
  const { added, removed } = op;
  const classes = useStyles();
  return (
    <div className={classes.main}>
      <Author author={op.author} className={classes.author} />
      {added.length > 0 && <span> added the </span>}
      {added.map((label, index) => (
        <Label key={index} label={label} />
      ))}
      {added.length > 0 && removed.length > 0 && <span> and</span>}
      {removed.length > 0 && <span> removed the </span>}
      {removed.map((label, index) => (
        <Label key={index} label={label} />
      ))}
      <span>
        {' '}
        label
        {added.length + removed.length > 1 && 's'}{' '}
      </span>
      <Date date={op.date} />
    </div>
  );
}

export default LabelChange;
