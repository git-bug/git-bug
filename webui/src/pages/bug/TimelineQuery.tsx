import React from 'react';

import CircularProgress from '@material-ui/core/CircularProgress';

import Timeline from './Timeline';
import { useTimelineQuery } from './TimelineQuery.generated';

type Props = {
  id: string;
};

const TimelineQuery = ({ id }: Props) => {
  const { loading, error, data } = useTimelineQuery({
    variables: {
      id,
      first: 100,
    },
  });

  if (loading) return <CircularProgress />;
  if (error) return <p>Error: {error}</p>;

  const nodes = data?.repository?.bug?.timeline.nodes;
  if (!nodes) {
    return null;
  }

  return <Timeline ops={nodes} />;
};

export default TimelineQuery;
