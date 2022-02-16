import * as React from 'react';

import { makeStyles } from '@material-ui/styles';

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
