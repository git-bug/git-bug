import React from 'react';
import { makeStyles } from '@material-ui/styles';

const useStyles = makeStyles({
  tag: {
    maxWidth: '100%',
    overflowX: 'auto',
  },
});

const PreTag = props => {
  const classes = useStyles();
  return <pre className={classes.tag} {...props}></pre>;
};

export default PreTag;
