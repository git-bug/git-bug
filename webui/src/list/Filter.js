import React from 'react';
import { makeStyles } from '@material-ui/styles';

const useStyles = makeStyles(theme => ({
  element: {
    ...theme.typography.body2,
    color: ({ active }) => (active ? '#333' : '#444'),
    padding: theme.spacing(0, 1),
    fontWeight: ({ active }) => (active ? 500 : 400),
    textDecoration: 'none',
    display: 'flex',
    alignSelf: ({ end }) => (end ? 'flex-end' : 'auto'),
    background: 'none',
    border: 'none',
  },
  icon: {
    paddingRight: theme.spacing(0.5),
  },
}));

function Filter({ active, children, icon: Icon, end, ...props }) {
  const classes = useStyles({ active, end });

  return (
    <button {...props} className={classes.element}>
      {Icon && <Icon fontSize="small" classes={{ root: classes.icon }} />}
      <div>{children}</div>
    </button>
  );
}

export default Filter;
