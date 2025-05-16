import Tooltip from '@mui/material/Tooltip/Tooltip';
import moment from 'moment';

import Moment from './Moment';

const HOUR = 1000 * 3600;
const DAY = 24 * HOUR;
const WEEK = 7 * DAY;

type Props = { date: string };
const Date = ({ date }: Props) => (
  <Tooltip title={moment(date).format('LLLL')}>
    <span>
      <Moment date={date} format="on ll" fromNowDuring={WEEK} />
    </span>
  </Tooltip>
);

export default Date;
