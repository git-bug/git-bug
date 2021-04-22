import React from 'react';

import { Typography } from '@material-ui/core';
import { makeStyles } from '@material-ui/core/styles';

import { Status } from '../../gqlTypes';
import Author from 'src/components/Author';
import Date from 'src/components/Date';

import { SetStatusFragment } from './SetStatusFragment.generated';

const useStyles = makeStyles((theme) => ({
  main: {
    color: theme.palette.text.secondary,
    marginLeft: theme.spacing(1) + 40,
  },
  author: {
    fontWeight: 'bold',
    color: theme.palette.text.secondary,
  },
}));

type Props = {
  op: SetStatusFragment;
};

function SetStatus({ op }: Props) {
  const classes = useStyles();
  const status = { [Status.Open]: 'reopened', [Status.Closed]: 'closed' }[
    op.status
  ];

  return (
    <Typography className={classes.main}>
      <Author author={op.author} className={classes.author} />
      <span> {status} this </span>
      <Date date={op.date} />
    </Typography>
  );
}

export default SetStatus;
