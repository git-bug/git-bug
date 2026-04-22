import { Link as MuiLink, Tooltip } from '@mui/material';
import makeStyles from '@mui/styles/makeStyles';
import { useEffect, useState } from 'react';

const useStyles = makeStyles((theme) => ({
  empty: {
    color: theme.palette.text.secondary,
    fontStyle: 'italic',
    padding: theme.spacing(2),
  },
  error: {
    color: theme.palette.error.main,
    padding: theme.spacing(2),
    fontFamily:
      'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    fontSize: '0.85rem',
  },
  list: {
    listStyle: 'none',
    padding: 0,
    margin: 0,
  },
  row: {
    display: 'flex',
    alignItems: 'baseline',
    gap: theme.spacing(1),
    padding: theme.spacing(0.75, 1),
    borderBottom: `1px solid ${theme.palette.divider}`,
  },
  hash: {
    fontFamily:
      'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    fontSize: '0.8rem',
    color: theme.palette.text.secondary,
    minWidth: '5.5rem',
  },
  message: {
    flex: 1,
    fontSize: '0.9rem',
  },
  author: {
    color: theme.palette.text.secondary,
    fontSize: '0.8rem',
  },
}));

type CommitEntry = {
  hash: string;
  message: string;
  authorName: string;
  authorEmail: string;
  date: string;
};

type Props = {
  repoName: string;
  baseRef: string;
  headRef: string;
  originUrl?: string | null;
};

function prNumberFromOrigin(url?: string | null): number | null {
  if (!url) return null;
  const m = url.match(/\/pull\/(\d+)(?:\D|$)/);
  return m ? Number(m[1]) : null;
}

/** PrCommits renders the list of commits the PR adds on top of its base
 *  branch, oldest first (matching how a PR reviewer reads it). Data comes
 *  from /pr/<repo>/commits which walks the local git history — no GitHub
 *  API call, no rate-limit cost. Each row links to GitHub's commit view
 *  if we have an originUrl to derive the commit URL from. */
export default function PrCommits({ repoName, baseRef, headRef, originUrl }: Props) {
  const classes = useStyles();
  const [commits, setCommits] = useState<CommitEntry[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    const prNum = prNumberFromOrigin(originUrl);
    const url =
      `/pr/${encodeURIComponent(repoName)}/commits` +
      `?base=${encodeURIComponent(baseRef)}&head=${encodeURIComponent(headRef)}` +
      (prNum ? `&pr=${prNum}` : '');
    fetch(url)
      .then(async (r) => {
        if (!r.ok) throw new Error(await r.text());
        return r.json();
      })
      .then((j: { commits: CommitEntry[] }) => {
        if (!cancelled) setCommits(j.commits ?? []);
      })
      .catch((e: Error) => {
        if (!cancelled) setError(e.message || 'request failed');
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [repoName, baseRef, headRef, originUrl]);

  if (loading) return <div className={classes.empty}>Loading commits…</div>;
  if (error) return <div className={classes.error}>⚠ {error}</div>;
  if (!commits || commits.length === 0) {
    return (
      <div className={classes.empty}>
        No commits between base and head — already merged, force-pushed, or
        pointing at the same ref.
      </div>
    );
  }

  // Commits come newest-first from the backend; flip to chronological for
  // the PR timeline feel (bottom = tip).
  const ordered = [...commits].reverse();

  // Derive a GitHub commit URL from the originUrl if we have one. Pattern:
  // .../pull/<n> → .../commit/<sha>.
  const githubBase = originUrl
    ? originUrl.replace(/\/pull\/\d+.*$/, '')
    : null;

  return (
    <ul className={classes.list}>
      {ordered.map((c) => {
        const shortHash = c.hash.slice(0, 8);
        const commitUrl = githubBase ? `${githubBase}/commit/${c.hash}` : null;
        return (
          <li key={c.hash} className={classes.row}>
            <Tooltip title={c.hash}>
              {commitUrl ? (
                <MuiLink
                  href={commitUrl}
                  target="_blank"
                  rel="noreferrer noopener"
                  underline="hover"
                  className={classes.hash}
                >
                  {shortHash}
                </MuiLink>
              ) : (
                <span className={classes.hash}>{shortHash}</span>
              )}
            </Tooltip>
            <span className={classes.message}>{c.message}</span>
            <span className={classes.author}>
              {c.authorName} · {new Date(c.date).toLocaleDateString()}
            </span>
          </li>
        );
      })}
    </ul>
  );
}
