import { fade, makeStyles } from '@material-ui/core/styles';
import IconButton from '@material-ui/core/IconButton';
import KeyboardArrowLeft from '@material-ui/icons/KeyboardArrowLeft';
import KeyboardArrowRight from '@material-ui/icons/KeyboardArrowRight';
import ErrorOutline from '@material-ui/icons/ErrorOutline';
import Paper from '@material-ui/core/Paper';
import InputBase from '@material-ui/core/InputBase';
import Skeleton from '@material-ui/lab/Skeleton';
import gql from 'graphql-tag';
import React, { useState, useEffect, useRef } from 'react';
import { useQuery } from '@apollo/react-hooks';
import { useLocation, useHistory, Link } from 'react-router-dom';
import BugRow from './BugRow';
import List from './List';
import FilterToolbar from './FilterToolbar';

const useStyles = makeStyles(theme => ({
  main: {
    maxWidth: 800,
    margin: 'auto',
    marginTop: theme.spacing(4),
    marginBottom: theme.spacing(4),
    overflow: 'hidden',
  },
  pagination: {
    ...theme.typography.overline,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  },
  header: {
    display: 'flex',
    padding: theme.spacing(2),
    '& > h1': {
      ...theme.typography.h6,
      margin: theme.spacing(0, 2),
    },
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  search: {
    borderRadius: theme.shape.borderRadius,
    borderColor: fade(theme.palette.primary.main, 0.2),
    borderStyle: 'solid',
    borderWidth: '1px',
    backgroundColor: fade(theme.palette.primary.main, 0.05),
    padding: theme.spacing(0, 1),
    width: ({ searching }) => (searching ? '20rem' : '15rem'),
    transition: theme.transitions.create(),
  },
  searchFocused: {
    borderColor: fade(theme.palette.primary.main, 0.4),
    backgroundColor: theme.palette.background.paper,
    width: '20rem!important',
  },
  placeholderRow: {
    padding: theme.spacing(1),
    borderBottomColor: theme.palette.grey['300'],
    borderBottomWidth: '1px',
    borderBottomStyle: 'solid',
    display: 'flex',
    alignItems: 'center',
  },
  placeholderRowStatus: {
    margin: theme.spacing(1, 2),
  },
  placeholderRowText: {
    flex: 1,
  },
  message: {
    ...theme.typography.h5,
    padding: theme.spacing(8),
    textAlign: 'center',
    borderBottomColor: theme.palette.grey['300'],
    borderBottomWidth: '1px',
    borderBottomStyle: 'solid',
    '& > p': {
      margin: '0',
    },
  },
  errorBox: {
    color: theme.palette.error.main,
    '& > pre': {
      fontSize: '1rem',
      textAlign: 'left',
      backgroundColor: theme.palette.grey['900'],
      color: theme.palette.common.white,
      marginTop: theme.spacing(4),
      padding: theme.spacing(2, 3),
    },
  },
}));

const QUERY = gql`
  query(
    $first: Int
    $last: Int
    $after: String
    $before: String
    $query: String
  ) {
    defaultRepository {
      bugs: allBugs(
        first: $first
        last: $last
        after: $after
        before: $before
        query: $query
      ) {
        totalCount
        edges {
          cursor
          node {
            ...BugRow
          }
        }
        pageInfo {
          hasNextPage
          hasPreviousPage
          startCursor
          endCursor
        }
      }
    }
  }

  ${BugRow.fragment}
`;

function editParams(params, callback) {
  const cloned = new URLSearchParams(params.toString());
  callback(cloned);
  return cloned;
}

// TODO: factor this out
const Placeholder = ({ count }) => {
  const classes = useStyles();
  return (
    <>
      {new Array(count).fill(null).map((_, i) => (
        <div key={i} className={classes.placeholderRow}>
          <Skeleton
            className={classes.placeholderRowStatus}
            variant="circle"
            width={20}
            height={20}
          />
          <div className={classes.placeholderRowText}>
            <Skeleton height={22} />
            <Skeleton height={24} width="60%" />
          </div>
        </div>
      ))}
    </>
  );
};

// TODO: factor this out
const NoBug = () => {
  const classes = useStyles();
  return (
    <div className={classes.message}>
      <ErrorOutline fontSize="large" />
      <p>No results matched your search.</p>
    </div>
  );
};

const Error = ({ error }) => {
  const classes = useStyles();
  return (
    <div className={[classes.errorBox, classes.message].join(' ')}>
      <ErrorOutline fontSize="large" />
      <p>There was an error while fetching bug.</p>
      <p>
        <em>{error.message}</em>
      </p>
      <pre>
        <code>{JSON.stringify(error, null, 2)}</code>
      </pre>
    </div>
  );
};

function ListQuery() {
  const location = useLocation();
  const history = useHistory();
  const params = new URLSearchParams(location.search);
  const query = params.get('q');

  const [input, setInput] = useState(query);

  const classes = useStyles({ searching: !!input });

  // TODO is this the right way to do it?
  const lastQuery = useRef();
  useEffect(() => {
    if (query !== lastQuery.current) {
      setInput(query);
    }
    lastQuery.current = query;
  }, [query, input, lastQuery]);

  const page = {
    first: params.get('first'),
    last: params.get('last'),
    after: params.get('after'),
    before: params.get('before'),
  };

  // If nothing set, show the first 10 items
  if (!page.first && !page.last) {
    page.first = 10;
  }

  const perPage = page.first || page.last;

  const { loading, error, data } = useQuery(QUERY, {
    variables: {
      ...page,
      query,
    },
  });

  let nextPage = null;
  let previousPage = null;
  let hasNextPage = false;
  let hasPreviousPage = false;
  let count = 0;
  if (!loading && !error && data.defaultRepository.bugs) {
    const bugs = data.defaultRepository.bugs;
    hasNextPage = bugs.pageInfo.hasNextPage;
    hasPreviousPage = bugs.pageInfo.hasPreviousPage;
    count = bugs.totalCount;
    // This computes the URL for the next page
    nextPage = {
      ...location,
      search: editParams(params, p => {
        p.delete('last');
        p.delete('before');
        p.set('first', perPage);
        p.set('after', bugs.pageInfo.endCursor);
      }).toString(),
    };
    // and this for the previous page
    previousPage = {
      ...location,
      search: editParams(params, p => {
        p.delete('first');
        p.delete('after');
        p.set('last', perPage);
        p.set('before', bugs.pageInfo.startCursor);
      }).toString(),
    };
  }

  // Prepare params without paging for editing filters
  const paramsWithoutPaging = editParams(params, p => {
    p.delete('first');
    p.delete('last');
    p.delete('before');
    p.delete('after');
  });
  // Returns a new location with the `q` param edited
  const queryLocation = query => ({
    ...location,
    search: editParams(paramsWithoutPaging, p => p.set('q', query)).toString(),
  });

  let content;
  if (loading) {
    content = <Placeholder count={10} />;
  } else if (error) {
    content = <Error error={error} />;
  } else {
    const bugs = data.defaultRepository.bugs;

    if (bugs.totalCount === 0) {
      content = <NoBug />;
    } else {
      content = <List bugs={bugs} />;
    }
  }

  const formSubmit = e => {
    e.preventDefault();
    history.push(queryLocation(input));
  };

  return (
    <Paper className={classes.main}>
      <header className={classes.header}>
        <h1>Issues</h1>
        <form onSubmit={formSubmit}>
          <InputBase
            placeholder="Filter"
            value={input}
            onInput={e => setInput(e.target.value)}
            classes={{
              root: classes.search,
              focused: classes.searchFocused,
            }}
          />
          <button type="submit" hidden>
            Search
          </button>
        </form>
      </header>
      <FilterToolbar query={query} queryLocation={queryLocation} />
      {content}
      <div className={classes.pagination}>
        <IconButton
          component={hasPreviousPage ? Link : 'button'}
          to={previousPage}
          disabled={!hasPreviousPage}
        >
          <KeyboardArrowLeft />
        </IconButton>
        <div>{loading ? 'Loading' : `Total: ${count}`}</div>
        <IconButton
          component={hasNextPage ? Link : 'button'}
          to={nextPage}
          disabled={!hasNextPage}
        >
          <KeyboardArrowRight />
        </IconButton>
      </div>
    </Paper>
  );
}

export default ListQuery;
