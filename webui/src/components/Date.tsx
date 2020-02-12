import Tooltip from '@material-ui/core/Tooltip/Tooltip';
import moment from 'moment';
import React from 'react';
import Moment from 'react-moment';

const HOUR = 1000 * 3600;
const DAY = 24 * HOUR;
const WEEK = 7 * DAY;

type Props = { date: string };
const Date = ({ date }: Props) => (
  <Tooltip title={moment(date).format('MMMM D, YYYY, h:mm a')}>
    <Moment date={date} fromNowDuring={WEEK} />
  </Tooltip>
);

export default Date;
