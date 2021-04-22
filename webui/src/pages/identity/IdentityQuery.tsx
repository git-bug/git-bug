import React from 'react';
import { RouteComponentProps } from 'react-router-dom';

import CircularProgress from '@material-ui/core/CircularProgress';

import { useGetUserByIdQuery } from '../../components/Identity/UserIdentity.generated';

import Identity from './Identity';

type Props = RouteComponentProps<{
  id: string;
}>;

const UserQuery: React.FC<Props> = ({ match }: Props) => {
  const { loading, error, data } = useGetUserByIdQuery({
    variables: { userId: match.params.id },
  });
  if (loading) return <CircularProgress />;
  if (error) return <p>Error: {error}</p>;
  if (!data?.repository?.identity) return <p>404.</p>;
  return <Identity identity={data.repository.identity} />;
};

export default UserQuery;
