import { makeStyles } from '@material-ui/styles';
import React from 'react';
import LabelChange from './LabelChange';
import Message from './Message';
import SetStatus from './SetStatus';
import SetTitle from './SetTitle';

const useStyles = makeStyles(theme => ({
  main: {
    '& > *:not(:last-child)': {
      marginBottom: theme.spacing.unit * 2,
    },
  },
}));

const componentMap = {
  CreateTimelineItem: Message,
  AddCommentTimelineItem: Message,
  LabelChangeTimelineItem: LabelChange,
  SetTitleTimelineItem: SetTitle,
  SetStatusTimelineItem: SetStatus,
};

function Timeline({ ops }) {
  const classes = useStyles();

  return (
    <div className={classes.main}>
      {ops.map((op, index) => {
        const Component = componentMap[op.__typename];

        if (!Component) {
          console.warn('unsupported operation type ' + op.__typename);
          return null;
        }

        return <Component key={index} op={op} />;
      })}
    </div>
  );
}

export default Timeline;
