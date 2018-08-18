import Tooltip from '@material-ui/core/Tooltip/Tooltip';
import * as moment from 'moment';
import React from 'react';

const Date = ({ date }) => (
  <Tooltip title={moment(date).format('MMMM D, YYYY, h:mm a')}>
    <span> {moment(date).fromNow()} </span>
  </Tooltip>
);

export default Date;
