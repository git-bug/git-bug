import { Route, Routes } from 'react-router';

import Layout from './components/Header';
import BugPage from './pages/bug';
import IdentityPage from './pages/identity';
import ListPage from './pages/list';
import NewBugPage from './pages/new/NewBugPage';
import NotFoundPage from './pages/notfound/NotFoundPage';

export default function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<ListPage />} />
        <Route path="/new" element={<NewBugPage />} />
        <Route path="/bug/:id" element={<BugPage />} />
        <Route path="/user/:id" element={<IdentityPage />} />
        <Route element={<NotFoundPage />} />
      </Routes>
    </Layout>
  );
}
