import makeStyles from '@mui/styles/makeStyles';

import { BugFragment } from './Bug.generated';
import LabelChange from './LabelChange';
import Message from './Message';
import SetStatus from './SetStatus';
import SetTitle from './SetTitle';
import { TimelineItemFragment } from './TimelineQuery.generated';

const useStyles = makeStyles((theme) => ({
  main: {
    '& > *:not(:last-child)': {
      marginBottom: theme.spacing(2),
    },
  },
}));

type Props = {
  ops: Array<TimelineItemFragment>;
  bug: BugFragment;
};

function Timeline({ bug, ops }: Props) {
  const classes = useStyles();

  return (
    <div className={classes.main}>
      {ops.map((op, index) => {
        switch (op.__typename) {
          case 'BugCreateTimelineItem':
            return <Message key={index} op={op} bug={bug} />;
          case 'BugAddCommentTimelineItem':
            return <Message key={index} op={op} bug={bug} />;
          case 'BugLabelChangeTimelineItem':
            return <LabelChange key={index} op={op} />;
          case 'BugSetTitleTimelineItem':
            return <SetTitle key={index} op={op} />;
          case 'BugSetStatusTimelineItem':
            return <SetStatus key={index} op={op} />;
        }

        console.warn('unsupported operation type ' + op.__typename);
        return null;
      })}
    </div>
  );
}

export default Timeline;
