import React from 'react';

import { makeStyles } from '@material-ui/core/styles';

import Author from 'src/components/Author';
import Date from 'src/components/Date';
import Label from 'src/components/Label';

import { LabelChangeFragment } from './LabelChangeFragment.generated';

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
  op: LabelChangeFragment;
};

function LabelChange({ op }: Props) {
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
