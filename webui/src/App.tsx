import React from 'react';
import { Route, Switch } from 'react-router';

import Layout from './layout';
import BugPage from './pages/bug';
import ListPage from './pages/list';

export default function App() {
  return (
    <Layout>
      <Switch>
        <Route path="/" exact component={ListPage} />
        <Route path="/bug/:id" exact component={BugPage} />
      </Switch>
    </Layout>
  );
}
