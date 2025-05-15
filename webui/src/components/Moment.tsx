import moment from 'moment';

type Props = {
  date: moment.MomentInput;
  format: string;
  fromNowDuring?: number;
};

const Moment = ({ date, format, fromNowDuring }: Props) => {
  let dateString: string | undefined;
  const dateMoment = moment(date);

  if (fromNowDuring) {
    const diff = moment().diff(dateMoment, 'ms');
    if (diff < fromNowDuring) {
      dateString = dateMoment.fromNow();
    }
  }

  // we either are out of range or didn't get asked for fromNow
  if (dateString === undefined) {
    dateString = dateMoment.format(format);
  }

  return <span>{dateString}</span>;
};

export default Moment;
