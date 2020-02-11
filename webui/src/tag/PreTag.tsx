import { makeStyles } from '@material-ui/styles';
import React from 'react';

const useStyles = makeStyles({
  tag: {
    maxWidth: '100%',
    overflowX: 'auto',
  },
});

const PreTag = (props: React.HTMLProps<HTMLPreElement>) => {
  const classes = useStyles();
  return <pre className={classes.tag} {...props}></pre>;
};

export default PreTag;
