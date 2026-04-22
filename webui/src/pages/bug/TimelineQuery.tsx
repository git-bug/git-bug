import CircularProgress from '@mui/material/CircularProgress';
import { useParams } from 'react-router';

import { BugFragment } from './Bug.generated';
import Timeline from './Timeline';
import { useTimelineQuery } from './TimelineQuery.generated';

type Props = {
  bug: BugFragment;
};

function safeDecode(s: string): string {
  try {
    return decodeURIComponent(s);
  } catch {
    return s;
  }
}

const TimelineQuery = ({ bug }: Props) => {
  const { repoName } = useParams<{ repoName: string }>();
  const repoRef = repoName ? safeDecode(repoName) : null;
  const { loading, error, data } = useTimelineQuery({
    variables: {
      id: bug.id,
      first: 100,
      repoRef,
    },
  });

  if (loading) return <CircularProgress />;
  if (error) return <p>Error: {error.message}</p>;

  const nodes = data?.repository?.bug?.timeline.nodes;
  if (!nodes) {
    return null;
  }

  return <Timeline ops={nodes} bug={bug} />;
};

export default TimelineQuery;
