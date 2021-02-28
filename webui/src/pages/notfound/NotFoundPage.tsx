import React from 'react';

import { makeStyles } from '@material-ui/core/styles';
import ArrowBackIcon from '@material-ui/icons/ArrowBack';

const useStyles = makeStyles((theme) => ({
  main: {
    maxWidth: 1000,
    margin: 'auto',
    marginTop: theme.spacing(10),
  },
  logo: {
    height: '350px',
    display: 'block',
    marginLeft: 'auto',
    marginRight: 'auto',
  },
  icon: {
    display: 'block',
    marginLeft: 'auto',
    marginRight: 'auto',
    fontSize: '80px',
  },
  backLink: {
    textDecoration: 'none',
    color: theme.palette.text.primary,
  },
  header: {
    fontSize: '30px',
    textAlign: 'center',
  },
}));

function NotFoundPage() {
  const classes = useStyles();
  return (
    <main className={classes.main}>
      <h1 className={classes.header}>404 – Page not found</h1>
      <img
        src="/logo-alpha-flat-outline.svg"
        className={classes.logo}
        alt="git-bug Logo"
      />
      <a href="/" className={classes.backLink}>
        <h2 className={classes.header}>Go back to start page</h2>
        <ArrowBackIcon className={classes.icon} />
      </a>
    </main>
  );
}

export default NotFoundPage;
