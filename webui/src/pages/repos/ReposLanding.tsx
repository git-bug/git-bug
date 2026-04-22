import { gql, useQuery } from '@apollo/client';
import makeStyles from '@mui/styles/makeStyles';
import { Link } from 'react-router';

/** ReposLanding is the root page in multi-repo mode: lists all registered
 *  repositories with per-kind counts and a link into each. A single-repo
 *  server returns one node here; we then render the usual bug list inline
 *  at / instead (handled in App.tsx). */
const REPOS_QUERY = gql`
  query ReposLanding {
    repositories {
      totalCount
      nodes {
        name
        allBugs(query: "kind:pr", first: 0) {
          totalCount
        }
        issues: allBugs(query: "kind:issue", first: 0) {
          totalCount
        }
      }
    }
  }
`;

interface ReposData {
  repositories: {
    totalCount: number;
    nodes: Array<{
      name: string | null;
      allBugs: { totalCount: number };
      issues: { totalCount: number };
    }>;
  };
}

const useStyles = makeStyles((theme) => ({
  main: {
    maxWidth: 900,
    margin: 'auto',
    marginTop: theme.spacing(4),
    padding: theme.spacing(2),
  },
  heading: {
    ...theme.typography.h5,
    marginBottom: theme.spacing(2),
  },
  row: {
    display: 'flex',
    alignItems: 'baseline',
    padding: theme.spacing(1.5),
    borderBottom: `1px solid ${theme.palette.divider}`,
    '&:hover': { background: theme.palette.action.hover },
  },
  name: {
    flex: 1,
    fontSize: '1rem',
    fontWeight: 500,
    color: theme.palette.text.primary,
    textDecoration: 'none',
  },
  count: {
    marginLeft: theme.spacing(2),
    color: theme.palette.text.secondary,
    fontSize: '0.9rem',
  },
}));

export default function ReposLanding() {
  const classes = useStyles();
  const { data, loading, error } = useQuery<ReposData>(REPOS_QUERY);

  if (loading) return <div className={classes.main}>Loading…</div>;
  if (error) return <div className={classes.main}>Error: {error.message}</div>;

  const nodes = data?.repositories.nodes ?? [];
  // Filter out the "default" unnamed repo — multi-repo mode registers all
  // repos by name, but the cwd still appears as the default. Show only named.
  const named = nodes.filter((n) => n.name);

  return (
    <main className={classes.main}>
      <div className={classes.heading}>
        Repositories ({named.length})
      </div>
      {named
        .slice()
        .sort((a, b) => (a.name || '').localeCompare(b.name || ''))
        .map((r) => (
          <Link
            key={r.name}
            to={`/r/${encodeURIComponent(r.name || '')}/?q=kind:issue status:open status:draft`}
            className={classes.row}
            style={{ textDecoration: 'none' }}
          >
            <span className={classes.name}>{r.name}</span>
            <span className={classes.count}>
              {r.issues.totalCount} issues
            </span>
            <span className={classes.count}>{r.allBugs.totalCount} PRs</span>
          </Link>
        ))}
      {named.length === 0 && (
        <div>
          This server was started in single-repo mode. Use{' '}
          <code>git bug webui --root DIR</code> to see multiple repos here.
        </div>
      )}
    </main>
  );
}
