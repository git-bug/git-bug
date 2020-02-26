import React, { useState } from 'react';

import CircularProgress from '@material-ui/core/CircularProgress';
import { makeStyles } from '@material-ui/styles';

import LabelBullet from './LabelBullet';
import { useValidLabelsQuery } from './ValidLabelsQuery.generated';

const useStyles = makeStyles({
  list: {
    listStyleType: 'none',
    padding: 0,
  },
});

const ValidLabels: React.FC = () => {
  const { loading, error, data } = useValidLabelsQuery();
  const [filter, setFilter] = useState('');
  const classes = useStyles();
  if (loading) return <CircularProgress />;
  if (error) return <p>Error: {error}</p>;
  const labels = data?.repository?.validLabels.nodes.filter(
    label =>
      filter === '' || label.name.toLowerCase().includes(filter.toLowerCase())
  );

  return (
    <>
      <input
        type="text"
        placeholder="Filter labels…"
        onChange={e => setFilter(e.target.value)}
        value={filter}
      />
      <ul className={classes.list}>
        {labels?.map(l => (
          <li key={l.name}>
            <LabelBullet label={l} />
            {l.name}
          </li>
        ))}
      </ul>
    </>
  );
};

export default ValidLabels;
