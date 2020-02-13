import React from 'react';

import { makeStyles } from '@material-ui/core/styles';

import Author from 'src/components/Author';
import Date from 'src/components/Date';

import { SetTitleFragment } from './SetTitleFragment.generated';

const useStyles = makeStyles(theme => ({
  main: {
    ...theme.typography.body1,
    marginLeft: theme.spacing(1) + 40,
  },
  bold: {
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
