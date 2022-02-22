import { makeStyles } from '@mui/styles';
import * as React from 'react';

const useStyles = makeStyles({
  tag: {
    maxWidth: '100%',
    overflowX: 'auto',
  },
});

const PreTag: React.FC<React.HTMLProps<HTMLPreElement>> = (props) => {
  const classes = useStyles();
  return <pre className={classes.tag} {...props} />;
};

export default PreTag;
