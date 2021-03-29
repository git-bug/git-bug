import { ApolloError } from '@apollo/client';
import React, { useState, useEffect, useRef } from 'react';
import { useLocation, useHistory, Link } from 'react-router-dom';

import { Button } from '@material-ui/core';
import IconButton from '@material-ui/core/IconButton';
import InputBase from '@material-ui/core/InputBase';
import Paper from '@material-ui/core/Paper';
import { makeStyles, Theme } from '@material-ui/core/styles';
import ErrorOutline from '@material-ui/icons/ErrorOutline';
import KeyboardArrowLeft from '@material-ui/icons/KeyboardArrowLeft';
import KeyboardArrowRight from '@material-ui/icons/KeyboardArrowRight';
import Skeleton from '@material-ui/lab/Skeleton';

import IfLoggedIn from 'src/components/IfLoggedIn/IfLoggedIn';

import FilterToolbar from './FilterToolbar';
import List from './List';
import { useListBugsQuery } from './ListQuery.generated';

type StylesProps = { searching?: boolean };
const useStyles = makeStyles<Theme, StylesProps>((theme) => ({
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
  filterissueLabel: {
    fontSize: '14px',
    fontWeight: 'bold',
    paddingRight: '12px',
  },
  filterissueContainer: {
    display: 'flex',
    flexDirection: 'row',
    alignItems: 'flex-start',
    justifyContents: 'left',
  },
  search: {
    borderRadius: theme.shape.borderRadius,
    color: theme.palette.text.secondary,
    borderColor: theme.palette.divider,
    borderStyle: 'solid',
    borderWidth: '1px',
    backgroundColor: theme.palette.primary.light,
    padding: theme.spacing(0, 1),
    width: ({ searching }) => (searching ? '20rem' : '15rem'),
    transition: theme.transitions.create([
      'width',
      'borderColor',
      'backgroundColor',
    ]),
  },
  searchFocused: {
    backgroundColor: theme.palette.background.paper,
  },
  placeholderRow: {
    padding: theme.spacing(1),
    borderBottomColor: theme.palette.divider,
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
    color: theme.palette.text.hint,
    borderBottomColor: theme.palette.divider,
    borderBottomWidth: '1px',
    borderBottomStyle: 'solid',
    '& > p': {
      margin: '0',
    },
  },
  errorBox: {
    color: theme.palette.error.dark,
    '& > pre': {
      fontSize: '1rem',
      textAlign: 'left',
      borderColor: theme.palette.divider,
      borderWidth: '1px',
      borderRadius: theme.shape.borderRadius,
      borderStyle: 'solid',
      color: theme.palette.text.primary,
      marginTop: theme.spacing(4),
      padding: theme.spacing(2, 3),
    },
  },
  greenButton: {
    backgroundColor: theme.palette.success.main,
    color: theme.palette.success.contrastText,
    '&:hover': {
      backgroundColor: theme.palette.success.dark,
      color: theme.palette.primary.contrastText,
    },
  },
}));

function editParams(
  params: URLSearchParams,
  callback: (params: URLSearchParams) => void
) {
  const cloned = new URLSearchParams(params.toString());
  callback(cloned);
  return cloned;
}

// TODO: factor this out
type PlaceholderProps = { count: number };
const Placeholder: React.FC<PlaceholderProps> = ({
  count,
}: PlaceholderProps) => {
  const classes = useStyles({});
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
  const classes = useStyles({});
  return (
    <div className={classes.message}>
      <ErrorOutline fontSize="large" />
      <p>No results matched your search.</p>
    </div>
  );
};

type ErrorProps = { error: ApolloError };
const Error: React.FC<ErrorProps> = ({ error }: ErrorProps) => {
  const classes = useStyles({});
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
  const query = params.has('q') ? params.get('q') || '' : 'status:open';

  const [input, setInput] = useState(query);

  const classes = useStyles({ searching: !!input });

  // TODO is this the right way to do it?
  const lastQuery = useRef<string | null>(null);
  useEffect(() => {
    if (query !== lastQuery.current) {
      setInput(query);
    }
    lastQuery.current = query;
  }, [query, input, lastQuery]);

  const num = (param: string | null) => (param ? parseInt(param) : null);
  const page = {
    first: num(params.get('first')),
    last: num(params.get('last')),
    after: params.get('after'),
    before: params.get('before'),
  };

  // If nothing set, show the first 10 items
  if (!page.first && !page.last) {
    page.first = 10;
  }

  const perPage = (page.first || page.last || 10).toString();

  const { loading, error, data } = useListBugsQuery({
    variables: {
      ...page,
      query,
    },
  });

  let nextPage = null;
  let previousPage = null;
  let count = 0;
  if (!loading && !error && data?.repository?.bugs) {
    const bugs = data.repository.bugs;
    count = bugs.totalCount;
    // This computes the URL for the next page
    if (bugs.pageInfo.hasNextPage) {
      nextPage = {
        ...location,
        search: editParams(params, (p) => {
          p.delete('last');
          p.delete('before');
          p.set('first', perPage);
          p.set('after', bugs.pageInfo.endCursor);
        }).toString(),
      };
    }
    // and this for the previous page
    if (bugs.pageInfo.hasPreviousPage) {
      previousPage = {
        ...location,
        search: editParams(params, (p) => {
          p.delete('first');
          p.delete('after');
          p.set('last', perPage);
          p.set('before', bugs.pageInfo.startCursor);
        }).toString(),
      };
    }
  }

  // Prepare params without paging for editing filters
  const paramsWithoutPaging = editParams(params, (p) => {
    p.delete('first');
    p.delete('last');
    p.delete('before');
    p.delete('after');
  });
  // Returns a new location with the `q` param edited
  const queryLocation = (query: string) => ({
    ...location,
    search: editParams(paramsWithoutPaging, (p) =>
      p.set('q', query)
    ).toString(),
  });

  let content;
  if (loading) {
    content = <Placeholder count={10} />;
  } else if (error) {
    content = <Error error={error} />;
  } else if (data?.repository) {
    const bugs = data.repository.bugs;

    if (bugs.totalCount === 0) {
      content = <NoBug />;
    } else {
      content = <List bugs={bugs} />;
    }
  }

  const formSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    history.push(queryLocation(input));
  };

  return (
    <Paper className={classes.main}>
      <header className={classes.header}>
        <div className="filterissueContainer">
          <form onSubmit={formSubmit}>
            <label className={classes.filterissueLabel} htmlFor="issuefilter">
              Filter
            </label>
            <InputBase
              id="issuefilter"
              placeholder="Filter"
              value={input}
              onInput={(e: any) => setInput(e.target.value)}
              classes={{
                root: classes.search,
                focused: classes.searchFocused,
              }}
            />
            <button type="submit" hidden>
              Search
            </button>
          </form>
        </div>
        <IfLoggedIn>
          {() => (
            <Button
              className={classes.greenButton}
              variant="contained"
              href="/new"
            >
              New bug
            </Button>
          )}
        </IfLoggedIn>
      </header>
      <FilterToolbar query={query} queryLocation={queryLocation} />
      {content}
      <div className={classes.pagination}>
        {previousPage ? (
          <IconButton component={Link} to={previousPage}>
            <KeyboardArrowLeft />
          </IconButton>
        ) : (
          <IconButton disabled>
            <KeyboardArrowLeft />
          </IconButton>
        )}
        <div>{loading ? 'Loading' : `Total: ${count}`}</div>
        {nextPage ? (
          <IconButton component={Link} to={nextPage}>
            <KeyboardArrowRight />
          </IconButton>
        ) : (
          <IconButton disabled>
            <KeyboardArrowRight />
          </IconButton>
        )}
      </div>
    </Paper>
  );
}

export default ListQuery;
