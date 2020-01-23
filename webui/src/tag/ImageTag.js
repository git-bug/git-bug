import React from 'react';
import { makeStyles } from '@material-ui/styles';

const useStyles = makeStyles({
  tag: {
    maxWidth: '100%',
  },
});

const ImageTag = (props) => {
  const classes = useStyles();
  return <img className={classes.tag} {...props} />
};

export default ImageTag;