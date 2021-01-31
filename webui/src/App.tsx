import React from 'react';
import { Route, Switch } from 'react-router';

import Layout from './layout';
import BugPage from './pages/bug';
import ListPage from './pages/list';
import NewBugPage from './pages/new/NewBugPage';

export default function App() {
  return (
    <Layout>
      <Switch>
        <Route path="/" exact component={ListPage} />
        <Route path="/new" exact component={NewBugPage} />
        <Route path="/bug/:id" exact component={BugPage} />
      </Switch>
    </Layout>
  );
}
