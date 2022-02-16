import { Typography } from '@material-ui/core';
import { makeStyles } from '@material-ui/core/styles';

import Author from 'src/components/Author';
import Date from 'src/components/Date';
import Label from 'src/components/Label';

import { LabelChangeFragment } from './LabelChangeFragment.generated';

const useStyles = makeStyles((theme) => ({
  main: {
    color: theme.palette.text.secondary,
    marginLeft: theme.spacing(1) + 40,
  },
  author: {
    fontWeight: 'bold',
    color: theme.palette.text.secondary,
  },
  label: {
    maxWidth: '50ch',
    marginLeft: theme.spacing(0.25),
    marginRight: theme.spacing(0.25),
  },
}));

type Props = {
  op: LabelChangeFragment;
};

function LabelChange({ op }: Props) {
  const { added, removed } = op;
  const classes = useStyles();
  return (
    <Typography className={classes.main}>
      <Author author={op.author} className={classes.author} />
      {added.length > 0 && <span> added the </span>}
      {added.map((label, index) => (
        <Label key={index} label={label} className={classes.label} />
      ))}
      {added.length > 0 && removed.length > 0 && <span> and</span>}
      {removed.length > 0 && <span> removed the </span>}
      {removed.map((label, index) => (
        <Label key={index} label={label} className={classes.label} />
      ))}
      <span>
        {' '}
        label
        {added.length + removed.length > 1 && 's'}{' '}
      </span>
      <Date date={op.date} />
    </Typography>
  );
}

export default LabelChange;
