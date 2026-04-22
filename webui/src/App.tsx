import { gql, useQuery } from '@apollo/client';
import * as React from 'react';
import { Route, Routes } from 'react-router';

import Layout from './components/Header';
import RepoContextBinder from './components/RepoContext';
import BugPage from './pages/bug';
import IdentityPage from './pages/identity';
import ListPage from './pages/list';
import NewBugPage from './pages/new/NewBugPage';
import NotFoundPage from './pages/notfound/NotFoundPage';
import ReposLanding from './pages/repos/ReposLanding';
import SearchPage from './pages/search/SearchPage';

// IS_MULTI_REPO_QUERY decides which component lives at "/". When the server
// hosts more than one repo, "/" is a landing page listing them; in single-
// repo mode "/" is the bug list as before.
const IS_MULTI_REPO_QUERY = gql`
  query IsMultiRepo {
    repositories {
      totalCount
      nodes {
        name
      }
    }
  }
`;

interface IsMultiRepoData {
  repositories: { totalCount: number; nodes: Array<{ name: string | null }> };
}

function RootRoute() {
  const { data, loading } = useQuery<IsMultiRepoData>(IS_MULTI_REPO_QUERY);
  if (loading) return null;
  const named =
    data?.repositories.nodes.filter((n) => n.name && n.name !== '__default') ??
    [];
  return named.length > 0 ? <ReposLanding /> : <ListPage />;
}

export default function App() {
  return (
    <React.StrictMode>
      <Layout>
        <Routes>
          {/* Multi-repo scoped routes. RepoContextBinder reads :repoName and
              sets Apollo's X-Repo-Name header so existing queries (which use
              `repository { ... }` without a ref) land on the right repo. */}
          <Route
            path="/r/:repoName/*"
            element={
              <RepoContextBinder>
                <Routes>
                  <Route path="/" element={<ListPage />} />
                  <Route path="/new" element={<NewBugPage />} />
                  <Route path="/bug/:id" element={<BugPage />} />
                  <Route path="/user/:id" element={<IdentityPage />} />
                  <Route element={<NotFoundPage />} />
                </Routes>
              </RepoContextBinder>
            }
          />

          {/* Global search across all registered repositories. */}
          <Route path="/search" element={<SearchPage />} />

          {/* Single-repo routes (unchanged behaviour) + root that adapts. */}
          <Route path="/" element={<RootRoute />} />
          <Route path="/new" element={<NewBugPage />} />
          <Route path="/bug/:id" element={<BugPage />} />
          <Route path="/user/:id" element={<IdentityPage />} />
          <Route element={<NotFoundPage />} />
        </Routes>
      </Layout>
    </React.StrictMode>
  );
}
