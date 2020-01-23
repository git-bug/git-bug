import AppBar from '@material-ui/core/AppBar';
import CssBaseline from '@material-ui/core/CssBaseline';
import { createMuiTheme, ThemeProvider } from '@material-ui/core/styles';
import { makeStyles } from '@material-ui/styles';
import Toolbar from '@material-ui/core/Toolbar';
import React from 'react';
import { Route, Switch } from 'react-router';
import { Link } from 'react-router-dom';

import BugQuery from './bug/BugQuery';
import ListQuery from './list/ListQuery';

const theme = createMuiTheme({
  palette: {
    primary: {
      main: '#263238',
    },
  },
});

const useStyles = makeStyles(theme => ({
  appTitle: {
    ...theme.typography.h6,
    color: 'white',
    textDecoration: 'none',
  },
}));

export default function App() {
  const classes = useStyles();

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <AppBar position="static" color="primary">
        <Toolbar>
          <Link to="/" className={classes.appTitle}>
            git-bug webui
          </Link>
        </Toolbar>
      </AppBar>
      <Switch>
        <Route path="/" exact component={ListQuery} />
        <Route path="/bug/:id" exact component={BugQuery} />
      </Switch>
    </ThemeProvider>
  );
}
