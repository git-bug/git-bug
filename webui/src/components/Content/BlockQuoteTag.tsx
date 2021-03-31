import React from 'react';

import { makeStyles } from '@material-ui/core/styles';

const useStyles = makeStyles((theme) => ({
  tag: {
    color: theme.palette.text.secondary,
    borderLeftWidth: '0.5ch',
    borderLeftStyle: 'solid',
    borderLeftColor: theme.palette.text.secondary,
    marginLeft: 0,
    paddingLeft: '0.5rem',
  },
}));

const BlockQuoteTag = (props: React.HTMLProps<HTMLPreElement>) => {
  const classes = useStyles();
  return <blockquote className={classes.tag} {...props} />;
};

export default BlockQuoteTag;
