import { makeStyles } from '@material-ui/styles';
import { useQuery } from '@apollo/react-hooks';
import gql from 'graphql-tag';
import React from 'react';
import Toolbar from '@material-ui/core/Toolbar';
import ErrorOutline from '@material-ui/icons/ErrorOutline';
import CheckCircleOutline from '@material-ui/icons/CheckCircleOutline';
import Filter, { parse, stringify } from './Filter';

// simple pipe operator
// pipe(o, f, g, h) <=> h(g(f(o)))
// TODO: move this out?
const pipe = (initial, ...funcs) => funcs.reduce((acc, f) => f(acc), initial);

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

const BUG_COUNT_QUERY = gql`
  query($query: String) {
    defaultRepository {
      bugs: allBugs(query: $query) {
        totalCount
      }
    }
  }
`;

// This prepends the filter text with a count
function CountingFilter({ query, children, ...props }) {
  const { data, loading, error } = useQuery(BUG_COUNT_QUERY, {
    variables: { query },
  });

  var prefix;
  if (loading) prefix = '...';
  else if (error) prefix = '???';
  // TODO: better prefixes & error handling
  else prefix = data.defaultRepository.bugs.totalCount;

  return (
    <Filter {...props}>
      {prefix} {children}
    </Filter>
  );
}

function FilterToolbar({ query, queryLocation }) {
  const classes = useStyles();
  const params = parse(query);

  const hasKey = key => params[key] && params[key].length > 0;
  const hasValue = (key, value) => hasKey(key) && params[key].includes(value);
  const loc = params => pipe(params, stringify, queryLocation);
  const replaceParam = (key, value) => params => ({
    ...params,
    [key]: [value],
  });
  const clearParam = key => params => ({
    ...params,
    [key]: [],
  });

  // TODO: author/label filters
  return (
    <Toolbar className={classes.toolbar}>
      <CountingFilter
        active={hasValue('status', 'open')}
        query={pipe(
          params,
          replaceParam('status', 'open'),
          clearParam('sort'),
          stringify
        )}
        to={pipe(params, replaceParam('status', 'open'), loc)}
        icon={ErrorOutline}
      >
        open
      </CountingFilter>
      <CountingFilter
        active={hasValue('status', 'closed')}
        query={pipe(
          params,
          replaceParam('status', 'closed'),
          clearParam('sort'),
          stringify
        )}
        to={pipe(params, replaceParam('status', 'closed'), loc)}
        icon={CheckCircleOutline}
      >
        closed
      </CountingFilter>
      <div className={classes.spacer} />
      {/*
      <Filter active={hasKey('author')}>Author</Filter>
      <Filter active={hasKey('label')}>Label</Filter>
      */}
      <Filter
        dropdown={[
          ['id', 'ID'],
          ['creation', 'Newest'],
          ['creation-asc', 'Oldest'],
          ['edit', 'Recently updated'],
          ['edit-asc', 'Least recently updated'],
        ]}
        active={hasKey('sort')}
        itemActive={key => hasValue('sort', key)}
        to={key => pipe(params, replaceParam('sort', key), loc)}
      >
        Sort
      </Filter>
    </Toolbar>
  );
}

export default FilterToolbar;
