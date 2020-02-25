import React, { useState } from 'react';

import CircularProgress from '@material-ui/core/CircularProgress';

import { useValidLabelsQuery } from './ValidLabelsQuery.generated';

const ValidLabels: React.FC = () => {
  const { loading, error, data } = useValidLabelsQuery();
  const [filter, setFilter] = useState('');
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
      <ul>
        {labels?.map(l => (
          <li>{l.name}</li>
        ))}
      </ul>
    </>
  );
};

export default ValidLabels;
