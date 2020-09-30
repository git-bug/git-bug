import React from 'react';

import { makeStyles } from '@material-ui/core/styles';

import Author from 'src/components/Author';
import Date from 'src/components/Date';

import { SetTitleFragment } from './SetTitleFragment.generated';

const useStyles = makeStyles((theme) => ({
  main: {
    ...theme.typography.body2,
    marginLeft: theme.spacing(1) + 40,
  },
  author: {
    fontWeight: 'bold',
  },
  before: {
    fontWeight: 'bold',
    textDecoration: 'line-through',
  },
  after: {
    fontWeight: 'bold',
  },
}));

type Props = {
  op: SetTitleFragment;
};

function SetTitle({ op }: Props) {
  const classes = useStyles();
  return (
    <div className={classes.main}>
      <Author author={op.author} className={classes.author} />
      <span> changed the title from </span>
      <span className={classes.before}>{op.was}</span>
      <span> to </span>
      <span className={classes.after}>{op.title}</span>&nbsp;
      <Date date={op.date} />
    </div>
  );
}

export default SetTitle;
