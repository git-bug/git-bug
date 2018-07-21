import React from "react";
import { withRouter, Switch, Route } from "react-router";

import AppBar from "@material-ui/core/AppBar";
import CssBaseline from "@material-ui/core/CssBaseline";
import Toolbar from "@material-ui/core/Toolbar";
import Typography from "@material-ui/core/Typography";

import Bug from "./Bug";

const Home = () => <h1>Home</h1>;

const App = ({ location }) => (
  <React.Fragment>
    <CssBaseline />
    <AppBar position="static" color="primary">
      <Toolbar>
        <Typography variant="title" color="inherit">
          git-bug-webui(1)
        </Typography>
      </Toolbar>
    </AppBar>
    <Switch>
      <Route path="/" exact component={Home} />
      <Route path="/bug/:id" exact component={Bug} />
    </Switch>
  </React.Fragment>
);

export default withRouter(App);
