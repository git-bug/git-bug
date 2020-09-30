import React from 'react';

import { makeStyles } from '@material-ui/core/styles';

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
};

function Timeline({ ops }: Props) {
  const classes = useStyles();

  return (
    <div className={classes.main}>
      {ops.map((op, index) => {
        switch (op.__typename) {
          case 'CreateTimelineItem':
            return <Message key={index} op={op} />;
          case 'AddCommentTimelineItem':
            return <Message key={index} op={op} />;
          case 'LabelChangeTimelineItem':
            return <LabelChange key={index} op={op} />;
          case 'SetTitleTimelineItem':
            return <SetTitle key={index} op={op} />;
          case 'SetStatusTimelineItem':
            return <SetStatus key={index} op={op} />;
        }

        console.warn('unsupported operation type ' + op.__typename);
        return null;
      })}
    </div>
  );
}

export default Timeline;
