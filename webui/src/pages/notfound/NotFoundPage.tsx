import React from 'react';

import { makeStyles } from '@material-ui/core/styles';

import BackToListButton from '../../components/BackToListButton';

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
    marginTop: theme.spacing(1),
    textAlign: 'center',
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
      <div className={classes.backLink}>
        <BackToListButton />
      </div>
    </main>
  );
}

export default NotFoundPage;
