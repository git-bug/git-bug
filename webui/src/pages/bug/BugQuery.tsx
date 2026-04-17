import CircularProgress from '@mui/material/CircularProgress';
import * as React from 'react';
import { useParams } from 'react-router';

import NotFoundPage from '../notfound/NotFoundPage';

import Bug from './Bug';
import { useGetBugQuery } from './BugQuery.generated';

const BugQuery: React.FC = () => {
  const params = useParams<'id' | 'repoName'>();
  if (params.id === undefined) throw new Error('missing route parameters');

  // In multi-repo mode, repoName comes from /r/:repoName/bug/:id and is
  // passed as $repoRef so Apollo cache-keys per repo and the server
  // resolves the target repo explicitly.
  const repoRef = params.repoName ? safeDecode(params.repoName) : null;

  const { loading, error, data } = useGetBugQuery({
    variables: { id: params.id, repoRef },
  });
  if (loading) return <CircularProgress />;
  if (!data?.repository?.bug) return <NotFoundPage />;
  if (error) return <p>Error: {error.message}</p>;

  return <Bug bug={data.repository.bug} />;
};

function safeDecode(s: string): string {
  try {
    return decodeURIComponent(s);
  } catch {
    return s;
  }
}

export default BugQuery;
