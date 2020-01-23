import { makeStyles } from '@material-ui/styles';
import IconButton from '@material-ui/core/IconButton';
import Toolbar from '@material-ui/core/Toolbar';
import KeyboardArrowLeft from '@material-ui/icons/KeyboardArrowLeft';
import KeyboardArrowRight from '@material-ui/icons/KeyboardArrowRight';
import ErrorOutline from '@material-ui/icons/ErrorOutline';
import CheckCircleOutline from '@material-ui/icons/CheckCircleOutline';
import Paper from '@material-ui/core/Paper';
import Filter from './Filter';
import Skeleton from '@material-ui/lab/Skeleton';
import gql from 'graphql-tag';
import React from 'react';
import { useQuery } from '@apollo/react-hooks';
import { useLocation, Link } from 'react-router-dom';
import BugRow from './BugRow';
import List from './List';

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
  toolbar: {
    backgroundColor: theme.palette.grey['100'],
    borderColor: theme.palette.grey['300'],
    borderWidth: '1px 0',
    borderStyle: 'solid',
    margin: theme.spacing(0, -1),
  },
  header: {
    ...theme.typography.h6,
    padding: theme.spacing(2, 4),
  },
  spacer: {
    flex: 1,
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
  noBug: {
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
    <div className={classes.noBug}>
      <ErrorOutline fontSize="large" />
      <p>No results matched your search.</p>
    </div>
  );
};

function ListQuery() {
  const classes = useStyles();
  const location = useLocation();
  const params = new URLSearchParams(location.search);
  const query = params.get('q');
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

  let content;
  if (loading) {
    content = <Placeholder count={10} />;
  } else if (error) {
    content = <p>Error: {JSON.stringify(error)}</p>;
  } else {
    const bugs = data.defaultRepository.bugs;

    if (bugs.totalCount === 0) {
      content = <NoBug />;
    } else {
      content = <List bugs={bugs} />;
    }
  }

  return (
    <Paper className={classes.main}>
      <header className={classes.header}>Issues</header>
      <Toolbar className={classes.toolbar}>
        {/* TODO */}
        <Filter active icon={ErrorOutline}>
          123 open
        </Filter>
        <Filter icon={CheckCircleOutline}>456 closed</Filter>
        <div className={classes.spacer} />
        <Filter>Author</Filter>
        <Filter>Label</Filter>
        <Filter>Sort</Filter>
      </Toolbar>
      {content}
      <div className={classes.pagination}>
        <IconButton
          component={Link}
          to={previousPage}
          disabled={!hasPreviousPage}
        >
          <KeyboardArrowLeft />
        </IconButton>
        <div>{loading ? 'Loading' : `Total: ${count}`}</div>
        <IconButton component={Link} to={nextPage} disabled={!hasNextPage}>
          <KeyboardArrowRight />
        </IconButton>
      </div>
    </Paper>
  );
}

export default ListQuery;
