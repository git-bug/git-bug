import React from 'react';

import { Link } from '@material-ui/core';
import CircularProgress from '@material-ui/core/CircularProgress';

import { useGetBugsByUserQuery } from './GetBugsByUser.generated';

type Props = {
  humanId: string;
};

function BugList({ humanId }: Props) {
  const { loading, error, data } = useGetBugsByUserQuery({
    variables: {
      query: 'author:' + humanId,
    },
  });

  if (loading) return <CircularProgress />;
  if (error) return <p>Error: {error}</p>;
  const bugs = data?.repository?.allBugs.nodes;

  console.log(bugs);
  return (
    <ol>
      <li>{bugs ? bugs[0].title : ''}</li>
      <Link href={'/bug/' + (bugs ? bugs[0].id : '')}>Klick</Link>
    </ol>
  );
}

export default BugList;
