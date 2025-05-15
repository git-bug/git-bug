import CircularProgress from '@mui/material/CircularProgress';
import * as React from 'react';
import { useParams } from 'react-router-dom';

import NotFoundPage from '../notfound/NotFoundPage';

import Bug from './Bug';
import { useGetBugQuery } from './BugQuery.generated';

const BugQuery: React.FC = () => {
  const params = useParams<'id'>();
  if (params.id === undefined) throw new Error('missing route parameters');

  const { loading, error, data } = useGetBugQuery({
    variables: { id: params.id },
  });
  if (loading) return <CircularProgress />;
  if (!data?.repository?.bug) return <NotFoundPage />;
  if (error) return <p>Error: {error.message}</p>;

  return <Bug bug={data.repository.bug} />;
};

export default BugQuery;
