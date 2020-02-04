import Tooltip from '@material-ui/core/Tooltip/Tooltip';
import moment from 'moment';
import React from 'react';

type Props = { date: string };
const Date = ({ date }: Props) => (
  <Tooltip title={moment(date).format('MMMM D, YYYY, h:mm a')}>
    <span> {moment(date).fromNow()} </span>
  </Tooltip>
);

export default Date;
