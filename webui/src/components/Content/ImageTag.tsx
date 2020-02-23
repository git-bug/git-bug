import React from 'react';

import { makeStyles } from '@material-ui/styles';

const useStyles = makeStyles({
  tag: {
    maxWidth: '100%',
  },
});

const ImageTag = ({
  alt,
  ...props
}: React.ImgHTMLAttributes<HTMLImageElement>) => {
  const classes = useStyles();
  return (
    <a href={props.src} target="_blank" rel="noopener noreferrer nofollow">
      <img className={classes.tag} alt={alt} {...props} />
    </a>
  );
};

export default ImageTag;
