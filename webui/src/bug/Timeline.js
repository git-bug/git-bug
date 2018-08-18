import { withStyles } from '@material-ui/core/styles';
import React from 'react';
import LabelChange from './LabelChange';
import Message from './Message';
import SetStatus from './SetStatus';
import SetTitle from './SetTitle';

const styles = theme => ({
  main: {
    '& > *:not(:last-child)': {
      marginBottom: 10,
    },
  },
});

class Timeline extends React.Component {
  props: {
    ops: Array,
    fetchMore: any => any,
    classes: any,
  };

  render() {
    const { ops, classes } = this.props;

    return (
      <div className={classes.main}>
        {ops.map((op, index) => {
          switch (op.__typename) {
            case 'CreateOperation':
              return <Message key={index} op={op} />;
            case 'AddCommentOperation':
              return <Message key={index} op={op} />;
            case 'LabelChangeOperation':
              return <LabelChange key={index} op={op} />;
            case 'SetTitleOperation':
              return <SetTitle key={index} op={op} />;
            case 'SetStatusOperation':
              return <SetStatus key={index} op={op} />;

            default:
              console.log('unsupported operation type ' + op.__typename);
              return null;
          }
        })}
      </div>
    );
  }
}

export default withStyles(styles)(Timeline);
