import React from 'react';
import { RouteComponentProps } from 'react-router-dom';

import CircularProgress from '@material-ui/core/CircularProgress';

import Bug from './Bug';
import { useGetBugQuery } from './BugQuery.generated';

type Props = RouteComponentProps<{
  id: string;
}>;

const BugQuery: React.FC<Props> = ({ match }: Props) => {
  const { loading, error, data } = useGetBugQuery({
    variables: { id: match.params.id },
  });
  if (loading) return <CircularProgress />;
  if (error) return <p>Error: {error}</p>;
  if (!data?.repository?.bug) return <p>404.</p>;
  return <Bug bug={data.repository.bug} />;
};

export default BugQuery;
