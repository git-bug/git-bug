import { ApolloError } from '@apollo/client';
import { pipe } from '@arrows/composition';
import ArrowDropDownIcon from '@mui/icons-material/ArrowDropDown';
import ErrorOutline from '@mui/icons-material/ErrorOutline';
import KeyboardArrowLeft from '@mui/icons-material/KeyboardArrowLeft';
import KeyboardArrowRight from '@mui/icons-material/KeyboardArrowRight';
import { Button, FormControl, Menu, MenuItem } from '@mui/material';
import IconButton from '@mui/material/IconButton';
import Paper from '@mui/material/Paper';
import Skeleton from '@mui/material/Skeleton';
import { Theme } from '@mui/material/styles';
import makeStyles from '@mui/styles/makeStyles';
import * as React from 'react';
import { useState, useRef } from 'react';
import { useLocation, useParams, Link } from 'react-router';

import { useCurrentIdentityQuery } from '../../components/Identity/CurrentIdentity.generated';
import IfLoggedIn from 'src/components/IfLoggedIn/IfLoggedIn';

import { parse, Query, stringify } from './Filter';
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
    padding: theme.spacing(1),
  },
  filterissueLabel: {
    fontSize: '14px',
    fontWeight: 'bold',
    paddingRight: '12px',
  },
  form: {
    display: 'flex',
    flexGrow: 1,
    marginRight: theme.spacing(1),
  },
  search: {
    borderRadius: theme.shape.borderRadius,
    color: theme.palette.text.secondary,
    borderColor: theme.palette.divider,
    borderStyle: 'solid',
    borderWidth: '1px',
    backgroundColor: theme.palette.primary.light,
    padding: theme.spacing(0, 1),
    width: '100%',
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
    color: theme.palette.text.primary,
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

function safeDecode(s: string): string {
  try {
    return decodeURIComponent(s);
  } catch {
    return s;
  }
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
            variant="circular"
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
  const params = new URLSearchParams(location.search);
  const query = params.has('q')
    ? params.get('q') || ''
    : 'kind:issue status:open status:draft';

  const [filterMenuIsOpen, setFilterMenuIsOpen] = useState(false);
  const filterButtonRef = useRef<HTMLButtonElement>(null);

  const classes = useStyles({});

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

  // repoName comes from /r/:repoName/* — when present, it's passed as
  // $repoRef so Apollo keys its cache per repo (avoiding cross-repo stale
  // reads) and the server resolves the target repo explicitly rather than
  // via the X-Repo-Name header.
  const { repoName } = useParams<{ repoName: string }>();
  const repoRef = repoName ? safeDecode(repoName) : null;

  const { loading, error, data } = useListBugsQuery({
    variables: {
      ...page,
      query,
      repoRef,
    },
    // cache-and-network: show cache immediately but always also issue a
    // network request so status toggles / query edits always end up with
    // fresh data even if the previous variables hit cache.
    fetchPolicy: 'cache-and-network',
    notifyOnNetworkStatusChange: true,
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
  // With fetchPolicy cache-and-network, `loading` is true whenever a
  // network request is in flight even if we already have (stale) cached
  // data. Only show the Placeholder when we have nothing to show at all,
  // so toggling filters shows the old list briefly instead of a skeleton.
  if (error) {
    content = <Error error={error} />;
  } else if (loading && !data?.repository) {
    content = <Placeholder count={10} />;
  } else if (data?.repository) {
    const bugs = data.repository.bugs;

    if (bugs.totalCount === 0) {
      content = <NoBug />;
    } else {
      content = <List bugs={bugs} />;
    }
  }

  const {
    loading: ciqLoading,
    error: ciqError,
    data: ciqData,
  } = useCurrentIdentityQuery();
  // Don't bail on missing user identity: in multi-repo read-only mode there
  // is no logged-in user but the list still needs to render. Only skip while
  // the query is in flight or errored.
  if (ciqError || ciqLoading) {
    return null;
  }
  const user = ciqData?.repository?.userIdentity;

  const loc = pipe(stringify, queryLocation);
  const qparams: Query = parse(query);
  const replaceParam =
    (key: string, value: string) =>
    (params: Query): Query => ({
      ...params,
      [key]: [value],
    });

  return (
    <Paper className={classes.main}>
      <header className={classes.header}>
        <div className={classes.form}>
          <FormControl>
            <Button
              aria-haspopup="true"
              ref={filterButtonRef}
              onClick={(e) => setFilterMenuIsOpen(true)}
            >
              Filter <ArrowDropDownIcon />
            </Button>
            <Menu
              open={filterMenuIsOpen}
              onClose={() => setFilterMenuIsOpen(false)}
              anchorEl={filterButtonRef.current}
              anchorOrigin={{
                vertical: 'bottom',
                horizontal: 'left',
              }}
              transformOrigin={{
                vertical: 'top',
                horizontal: 'left',
              }}
            >
              {user && (
                <MenuItem
                  component={Link}
                  to={pipe(
                    replaceParam('author', user.displayName),
                    replaceParam('sort', 'creation'),
                    loc
                  )(qparams)}
                  onClick={() => setFilterMenuIsOpen(false)}
                >
                  Your newest issues
                </MenuItem>
              )}
            </Menu>
          </FormControl>
        </div>
        <IfLoggedIn>
          {() => (
            <Button
              className={classes.greenButton}
              variant="contained"
              component={Link}
              to="/new"
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
          <IconButton component={Link} to={previousPage} size="large">
            <KeyboardArrowLeft />
          </IconButton>
        ) : (
          <IconButton disabled size="large">
            <KeyboardArrowLeft />
          </IconButton>
        )}
        <div>{loading ? 'Loading' : `Total: ${count}`}</div>
        {nextPage ? (
          <IconButton component={Link} to={nextPage} size="large">
            <KeyboardArrowRight />
          </IconButton>
        ) : (
          <IconButton disabled size="large">
            <KeyboardArrowRight />
          </IconButton>
        )}
      </div>
    </Paper>
  );
}

export default ListQuery;
