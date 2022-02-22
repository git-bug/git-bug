import { makeStyles } from '@mui/styles';
import * as React from 'react';

const useStyles = makeStyles({
  tag: {
    maxWidth: '100%',
  },
});

const ImageTag: React.FC<React.ImgHTMLAttributes<HTMLImageElement>> = ({
  alt,
  ...props
}) => {
  const classes = useStyles();
  return (
    <>
      <a href={props.src} target="_blank" rel="noopener noreferrer nofollow">
        <img className={classes.tag} alt={alt} {...props} />
      </a>
      <br />
    </>
  );
};

export default ImageTag;
