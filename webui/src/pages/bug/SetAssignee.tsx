import { Typography } from '@mui/material';
import makeStyles from '@mui/styles/makeStyles';
import { Link } from 'react-router';

import Author from 'src/components/Author';
import Date from 'src/components/Date';

import { SetAssigneeFragment } from './SetAssigneeFragment.generated';

const useStyles = makeStyles((theme) => ({
  main: {
    color: theme.palette.text.secondary,
    marginLeft: theme.spacing(1) + 40,
  },
  author: {
    fontWeight: 'bold',
    color: theme.palette.text.secondary,
  },
  assignee: {
    fontWeight: 'bold',
    color: theme.palette.text.primary,
    textDecoration: 'none',
    '&:hover': {
      textDecoration: 'underline',
    },
  },
}));

type Props = {
  op: SetAssigneeFragment;
};

function SetAssignee({ op }: Props) {
  const classes = useStyles();

  return (
    <Typography className={classes.main}>
      <Author author={op.author} className={classes.author} />
      {op.assignee ? (
        <>
          <span> assigned this to </span>
          <Link to={`/user/${op.assignee.humanId}`} className={classes.assignee}>
            {op.assignee.displayName}
          </Link>
        </>
      ) : (
        <span> removed the assignee</span>
      )}
      &nbsp;
      <Date date={op.date} />
    </Typography>
  );
}

export default SetAssignee;
