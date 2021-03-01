import React from 'react';
import { Route, Switch } from 'react-router';

import Layout from './components/Header';
import BugPage from './pages/bug';
import ListPage from './pages/list';
import NewBugPage from './pages/new/NewBugPage';
import NotFoundPage from './pages/notfound/NotFoundPage';

export default function App() {
  return (
    <Layout>
      <Switch>
        <Route path="/" exact component={ListPage} />
        <Route path="/new" exact component={NewBugPage} />
        <Route path="/404bug" exact component={NotFoundPage} />
        <Route path="/bug/:id" exact component={BugPage} />
        <Route component={NotFoundPage} />
      </Switch>
    </Layout>
  );
}
