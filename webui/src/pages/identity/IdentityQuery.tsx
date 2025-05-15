import CircularProgress from '@mui/material/CircularProgress';
import * as React from 'react';
import { useParams } from 'react-router';

import { useGetUserByIdQuery } from '../../components/Identity/UserIdentity.generated';

import Identity from './Identity';

const UserQuery: React.FC = () => {
  const params = useParams<'id'>();
  if (params.id === undefined) throw new Error('missing route parameters');

  const { loading, error, data } = useGetUserByIdQuery({
    variables: { userId: params.id },
  });
  if (loading) return <CircularProgress />;
  if (error) return <p>Error: {error.message}</p>;
  if (!data?.repository?.identity) return <p>404.</p>;
  return <Identity identity={data.repository.identity} />;
};

export default UserQuery;
