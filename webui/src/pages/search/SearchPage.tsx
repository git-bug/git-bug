import makeStyles from '@mui/styles/makeStyles';
import { useMemo } from 'react';
import { useLocation, Link } from 'react-router';

import { useSearchAllQuery } from './SearchQuery.generated';

const useStyles = makeStyles((theme) => ({
  main: {
    maxWidth: 900,
    margin: 'auto',
    marginTop: theme.spacing(4),
    padding: theme.spacing(2),
  },
  heading: {
    ...theme.typography.h5,
    marginBottom: theme.spacing(1),
  },
  sub: {
    color: theme.palette.text.secondary,
    marginBottom: theme.spacing(2),
  },
  repoBlock: {
    marginTop: theme.spacing(3),
    borderTop: `1px solid ${theme.palette.divider}`,
    paddingTop: theme.spacing(1),
  },
  repoHeader: {
    display: 'flex',
    alignItems: 'baseline',
    gap: theme.spacing(1),
    marginBottom: theme.spacing(1),
  },
  repoName: {
    fontWeight: 600,
    color: theme.palette.text.primary,
    textDecoration: 'none',
  },
  count: {
    color: theme.palette.text.secondary,
    fontSize: '0.9rem',
  },
  row: {
    padding: theme.spacing(0.75, 1),
    display: 'flex',
    alignItems: 'baseline',
    gap: theme.spacing(1),
    '&:hover': { background: theme.palette.action.hover },
    textDecoration: 'none',
    color: theme.palette.text.primary,
  },
  rowMeta: {
    color: theme.palette.text.secondary,
    fontSize: '0.85rem',
    fontFamily:
      'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
  },
  kindBadge: {
    padding: '1px 6px',
    borderRadius: 8,
    fontSize: '0.7rem',
    textTransform: 'uppercase',
    color: 'white',
    background: theme.palette.primary.main,
  },
  empty: {
    color: theme.palette.text.secondary,
    marginTop: theme.spacing(2),
  },
}));

/** parseScope pulls repo: and org: tokens out of the raw query string.
 *  Everything else is passed through to the server unchanged. */
function parseScope(raw: string): {
  repos: string[];
  orgs: string[];
  rest: string;
} {
  const repos: string[] = [];
  const orgs: string[] = [];
  const rest: string[] = [];
  // Minimal tokenizer: respects double-quoted chunks so `title:"foo bar"` is
  // one token. Matches the DSL used by the list page filter.
  const tokens = raw.match(/"[^"]*"|\S+/g) ?? [];
  for (const t of tokens) {
    const m = t.match(/^(\w+):(.+)$/);
    if (m && m[1] === 'repo') {
      repos.push(m[2].replace(/^"|"$/g, ''));
    } else if (m && m[1] === 'org') {
      orgs.push(m[2].replace(/^"|"$/g, ''));
    } else {
      rest.push(t);
    }
  }
  return { repos, orgs, rest: rest.join(' ') };
}

/** matchScope decides whether a given repo name should be included in the
 *  results. An empty scope = accept all. A `repo:` token matches either
 *  the full "org/repo" name or just the basename after the slash, so
 *  `repo:example-counter` and `repo:midnightntwrk/example-counter` both
 *  resolve the expected repo. `org:` matches the first path segment. */
function matchScope(
  name: string,
  repos: string[],
  orgs: string[]
): boolean {
  if (repos.length === 0 && orgs.length === 0) return true;
  const lower = name.toLowerCase();
  const slash = lower.indexOf('/');
  const orgPart = slash >= 0 ? lower.slice(0, slash) : '';
  const basePart = slash >= 0 ? lower.slice(slash + 1) : lower;
  if (
    repos.some((r) => {
      const v = r.toLowerCase();
      return v === lower || v === basePart;
    })
  ) {
    return true;
  }
  if (orgs.some((o) => o.toLowerCase() === orgPart)) return true;
  return false;
}

export default function SearchPage() {
  const classes = useStyles();
  const location = useLocation();
  const raw = new URLSearchParams(location.search).get('q') ?? '';

  const scope = useMemo(() => parseScope(raw), [raw]);

  // Pass only the non-scope portion of the query to the server; the scope
  // is applied client-side after the response is back.
  const { data, loading, error } = useSearchAllQuery({
    variables: { query: scope.rest || 'status:open status:draft' },
    // We want up-to-date results every time since these are cross-repo
    // fan-out; cache-and-network gives immediate stale + fresh update.
    fetchPolicy: 'cache-and-network',
  });

  if (loading && !data) {
    return (
      <main className={classes.main}>
        <div className={classes.heading}>Searching…</div>
        <div className={classes.sub}>{raw}</div>
      </main>
    );
  }
  if (error) {
    return (
      <main className={classes.main}>
        <div className={classes.heading}>Search error</div>
        <div className={classes.sub}>{error.message}</div>
      </main>
    );
  }

  const nodes = data?.repositories.nodes ?? [];
  const matched = nodes
    .filter((r) => r.name && r.name !== '__default')
    .filter((r) => matchScope(r.name!, scope.repos, scope.orgs))
    .filter((r) => r.allBugs.totalCount > 0)
    .sort((a, b) => b.allBugs.totalCount - a.allBugs.totalCount);

  const totalHits = matched.reduce((s, r) => s + r.allBugs.totalCount, 0);
  const totalRepos = matched.length;

  return (
    <main className={classes.main}>
      <div className={classes.heading}>Search</div>
      <div className={classes.sub}>
        <code>{raw || '(no query)'}</code> — {totalHits} result
        {totalHits === 1 ? '' : 's'} across {totalRepos} repo
        {totalRepos === 1 ? '' : 's'}
      </div>

      {matched.length === 0 && (
        <div className={classes.empty}>No matches.</div>
      )}

      {matched.map((repo) => (
        <section key={repo.name} className={classes.repoBlock}>
          <div className={classes.repoHeader}>
            <Link
              to={`/r/${encodeURIComponent(repo.name!)}/?q=${encodeURIComponent(
                scope.rest || 'status:open status:draft'
              )}`}
              className={classes.repoName}
            >
              {repo.name}
            </Link>
            <span className={classes.count}>
              {repo.allBugs.totalCount} hit
              {repo.allBugs.totalCount === 1 ? '' : 's'}
            </span>
            {repo.allBugs.totalCount > repo.allBugs.nodes.length && (
              <Link
                to={`/r/${encodeURIComponent(
                  repo.name!
                )}/?q=${encodeURIComponent(scope.rest || 'status:open status:draft')}`}
                className={classes.count}
              >
                (view all →)
              </Link>
            )}
          </div>
          {repo.allBugs.nodes.map((bug) => (
            <Link
              key={bug.id}
              to={`/r/${encodeURIComponent(repo.name!)}/bug/${bug.id}`}
              className={classes.row}
            >
              <span className={classes.kindBadge}>{bug.kind}</span>
              <span>{bug.title}</span>
              <span className={classes.rowMeta}>
                {bug.humanId} {bug.status.toLowerCase()}
              </span>
            </Link>
          ))}
        </section>
      ))}
    </main>
  );
}
