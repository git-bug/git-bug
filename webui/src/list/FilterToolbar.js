import { makeStyles } from '@material-ui/styles';
import React from 'react';
import Toolbar from '@material-ui/core/Toolbar';
import ErrorOutline from '@material-ui/icons/ErrorOutline';
import CheckCircleOutline from '@material-ui/icons/CheckCircleOutline';
import Filter, { parse, stringify } from './Filter';

const useStyles = makeStyles(theme => ({
  toolbar: {
    backgroundColor: theme.palette.grey['100'],
    borderColor: theme.palette.grey['300'],
    borderWidth: '1px 0',
    borderStyle: 'solid',
    margin: theme.spacing(0, -1),
  },
  spacer: {
    flex: 1,
  },
}));

function FilterToolbar({ query, queryLocation }) {
  const classes = useStyles();
  const params = parse(query);
  const hasKey = key => params[key] && params[key].length > 0;
  const hasValue = (key, value) => hasKey(key) && params[key].includes(value);
  const replaceParam = (key, value) => {
    const p = {
      ...params,
      [key]: [value],
    };
    return queryLocation(stringify(p));
  };

  // TODO: open/closed count
  // TODO: author/label/sort filters
  return (
    <Toolbar className={classes.toolbar}>
      <Filter
        active={hasValue('status', 'open')}
        to={replaceParam('status', 'open')}
        icon={ErrorOutline}
      >
        open
      </Filter>
      <Filter
        active={hasValue('status', 'closed')}
        to={replaceParam('status', 'closed')}
        icon={CheckCircleOutline}
      >
        closed
      </Filter>
      <div className={classes.spacer} />
      <Filter active={hasKey('author')}>Author</Filter>
      <Filter active={hasKey('label')}>Label</Filter>
      <Filter active={hasKey('sort')}>Sort</Filter>
    </Toolbar>
  );
}

export default FilterToolbar;
