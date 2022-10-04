import makeStyles from '@mui/styles/makeStyles';
import * as React from 'react';

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

const BlockQuoteTag: React.FC<React.HTMLProps<HTMLQuoteElement>> = (props) => {
  const classes = useStyles();
  return <blockquote className={classes.tag} {...props} />;
};

export default BlockQuoteTag;
