import AppBar from '@material-ui/core/AppBar';
import CssBaseline from '@material-ui/core/CssBaseline';
import {
  createMuiTheme,
  ThemeProvider,
  makeStyles,
} from '@material-ui/core/styles';
import Toolbar from '@material-ui/core/Toolbar';
import React from 'react';
import { Route, Switch } from 'react-router';
import { Link } from 'react-router-dom';

import BugQuery from './bug/BugQuery';
import ListQuery from './list/ListQuery';
import CurrentIdentity from './CurrentIdentity';

const theme = createMuiTheme({
  palette: {
    primary: {
      main: '#263238',
    },
  },
});

const useStyles = makeStyles(theme => ({
  offset: {
    ...theme.mixins.toolbar,
  },
  filler: {
    flexGrow: 1,
  },
  appTitle: {
    ...theme.typography.h6,
    color: 'white',
    textDecoration: 'none',
    display: 'flex',
    alignItems: 'center',
  },
  logo: {
    height: '42px',
    marginRight: theme.spacing(2),
  },
}));

export default function App() {
  const classes = useStyles();

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <AppBar position="fixed" color="primary">
        <Toolbar>
          <Link to="/" className={classes.appTitle}>
            <img src="logo.svg" className={classes.logo} alt="git-bug" />
            git-bug
          </Link>
          <div className={classes.filler}></div>
          <CurrentIdentity />
        </Toolbar>
      </AppBar>
      <div className={classes.offset} />
      <Switch>
        <Route path="/" exact component={ListQuery} />
        <Route path="/bug/:id" exact component={BugQuery} />
      </Switch>
    </ThemeProvider>
  );
}
