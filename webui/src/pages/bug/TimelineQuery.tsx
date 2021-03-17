import React from 'react';

import CircularProgress from '@material-ui/core/CircularProgress';

import { BugFragment } from './Bug.generated';
import Timeline from './Timeline';
import { useTimelineQuery } from './TimelineQuery.generated';

type Props = {
  bug: BugFragment;
};

const TimelineQuery = ({ bug }: Props) => {
  const { loading, error, data } = useTimelineQuery({
    variables: {
      id: bug.id,
      first: 100,
    },
  });

  if (loading) return <CircularProgress />;
  if (error) return <p>Error: {error}</p>;

  const nodes = data?.repository?.bug?.timeline.nodes;
  if (!nodes) {
    return null;
  }

  return <Timeline ops={nodes} bug={bug} />;
};

export default TimelineQuery;
