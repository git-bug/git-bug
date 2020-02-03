import CircularProgress from '@material-ui/core/CircularProgress';
import React from 'react';
import Timeline from './Timeline';

import { useTimelineQuery } from './TimelineQuery.generated';

const TimelineQuery = ({ id }) => {
  const { loading, error, data, fetchMore } = useTimelineQuery({
    variables: {
      id,
      first: 100,
    },
  });

  if (loading) return <CircularProgress />;
  if (error) return <p>Error: {error}</p>;
  return (
    <Timeline
      ops={data.defaultRepository.bug.timeline.nodes}
      fetchMore={fetchMore}
    />
  );
};

export default TimelineQuery;
