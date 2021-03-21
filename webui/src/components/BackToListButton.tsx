import React from 'react';

import Button from '@material-ui/core/Button';
import { makeStyles } from '@material-ui/core/styles';
import ArrowBackIcon from '@material-ui/icons/ArrowBack';

const useStyles = makeStyles((theme) => ({
  backButton: {
    position: 'sticky',
    top: '80px',
    backgroundColor: theme.palette.primary.dark,
    color: theme.palette.primary.contrastText,
    '&:hover': {
      backgroundColor: theme.palette.primary.main,
      color: theme.palette.primary.contrastText,
    },
  },
}));

function BackToListButton() {
  const classes = useStyles();

  return (
    <Button
      variant="contained"
      className={classes.backButton}
      aria-label="back to issue list"
      href="/"
    >
      <ArrowBackIcon />
      Back to List
    </Button>
  );
}

export default BackToListButton;
