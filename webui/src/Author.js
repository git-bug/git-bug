import Tooltip from '@material-ui/core/Tooltip/Tooltip';
import React from 'react';
import { withStyles } from '@material-ui/core/styles';

const styles = theme => ({
  author: {
    ...theme.typography.body2,
  },
  bold: {
    fontWeight: 'bold',
  },
});

const Author = ({ author, bold, classes }) => {
  const klass = bold ? [classes.author, classes.bold] : [classes.author];

  return (
    <Tooltip title={author.email}>
      <span className={klass.join(' ')}>{author.name}</span>
    </Tooltip>
  );
};

export default withStyles(styles)(Author);
