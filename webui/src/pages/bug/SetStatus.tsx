import React from 'react';

import { makeStyles } from '@material-ui/core/styles';

import { Status } from '../../gqlTypes';
import Author from 'src/components/Author';
import Date from 'src/components/Date';

import { SetStatusFragment } from './SetStatusFragment.generated';

const useStyles = makeStyles((theme) => ({
  main: {
    ...theme.typography.body2,
    marginLeft: theme.spacing(1) + 40,
  },
  author: {
    fontWeight: 'bold',
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
    <div className={classes.main}>
      <Author author={op.author} className={classes.author} />
      <span> {status} this </span>
      <Date date={op.date} />
    </div>
  );
}

export default SetStatus;
