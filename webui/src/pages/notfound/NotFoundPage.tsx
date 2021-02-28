import React from 'react';

import { makeStyles } from '@material-ui/core/styles';

const useStyles = makeStyles((theme) => ({
  main: {
    maxWidth: 1000,
    margin: 'auto',
    marginTop: theme.spacing(20),
  },
  logo: {
    height: '350px',
  },
  container: {
    display: 'flex',
    alignItems: 'center',
  },
}));

function NotFoundPage() {
  const classes = useStyles();
  return (
    <main className={classes.main}>
      <div className={classes.container}>
        <h1>404 – Page not found</h1>
        <img
          src="/logo-alpha-flat-outline.svg"
          className={classes.logo}
          alt="git-bug"
        />
        <h1>Go back to start page</h1>
      </div>
    </main>
  );
}

export default NotFoundPage;
