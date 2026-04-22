import CancelIcon from '@mui/icons-material/Cancel';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorIcon from '@mui/icons-material/Error';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import HourglassTopIcon from '@mui/icons-material/HourglassTop';
import RadioButtonUncheckedIcon from '@mui/icons-material/RadioButtonUnchecked';
import { IconButton, Link as MuiLink, Tooltip } from '@mui/material';
import makeStyles from '@mui/styles/makeStyles';
import { useEffect, useState } from 'react';

import { BugFragment } from './Bug.generated';

const useStyles = makeStyles((theme) => ({
  panel: {
    border: `1px solid ${theme.palette.divider}`,
    borderRadius: theme.shape.borderRadius,
    padding: theme.spacing(1.5),
    marginBottom: theme.spacing(2),
    fontSize: '0.85rem',
    background: theme.palette.background.paper,
  },
  header: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
  },
  title: {
    fontWeight: 600,
  },
  source: {
    fontSize: '0.75rem',
    color: theme.palette.text.secondary,
    marginLeft: 'auto',
  },
  stateIcon: {
    fontSize: '1.2rem',
  },
  success: { color: '#2da44e' },
  failure: { color: '#cf222e' },
  pending: { color: '#bf8700' },
  neutral: { color: theme.palette.text.secondary },
  suiteList: {
    listStyle: 'none',
    padding: 0,
    marginTop: theme.spacing(1),
  },
  suite: {
    marginBottom: theme.spacing(1),
    borderLeft: `3px solid ${theme.palette.divider}`,
    paddingLeft: theme.spacing(1),
  },
  suiteHeader: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(0.5),
    fontSize: '0.85rem',
  },
  suiteApp: {
    color: theme.palette.text.secondary,
  },
  runList: {
    listStyle: 'none',
    padding: 0,
    margin: theme.spacing(0.5, 0, 0, 1),
  },
  run: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(0.5),
    fontSize: '0.8rem',
    padding: theme.spacing(0.25, 0),
  },
  runName: {
    flex: 1,
    color: theme.palette.text.primary,
  },
  error: {
    color: theme.palette.error.main,
    fontSize: '0.85rem',
  },
  empty: {
    color: theme.palette.text.secondary,
    fontSize: '0.85rem',
    fontStyle: 'italic',
  },
}));

type CheckRun = {
  name: string;
  status: string;
  conclusion: string;
  detailsUrl?: string;
  startedAt?: string;
  completedAt?: string;
};

type CheckSuite = {
  appName: string;
  status: string;
  conclusion: string;
  runs: CheckRun[];
};

type CheckStatus = {
  state: string;
  suites: CheckSuite[];
  fetchedAt: string;
};

type ChecksResponse = {
  state?: string;
  suites?: CheckSuite[];
  fetchedAt?: string;
  source: string;
  error?: string;
  cached: boolean;
} | null;

// rollupIcon picks the aggregate badge shown next to the "CI" heading.
// The mapping mirrors GitHub's own status pill. We accept the empty string
// (no rollup yet) and render a neutral circle so the panel never vanishes.
function rollupIcon(state: string, cls: ReturnType<typeof useStyles>) {
  switch (state) {
    case 'SUCCESS':
      return (
        <CheckCircleIcon
          className={`${cls.stateIcon} ${cls.success}`}
          aria-label="all checks passed"
        />
      );
    case 'FAILURE':
    case 'ERROR':
      return (
        <CancelIcon
          className={`${cls.stateIcon} ${cls.failure}`}
          aria-label="checks failed"
        />
      );
    case 'PENDING':
    case 'EXPECTED':
      return (
        <HourglassTopIcon
          className={`${cls.stateIcon} ${cls.pending}`}
          aria-label="checks running"
        />
      );
    default:
      return (
        <RadioButtonUncheckedIcon
          className={`${cls.stateIcon} ${cls.neutral}`}
          aria-label="no checks"
        />
      );
  }
}

function runIcon(status: string, conclusion: string, cls: ReturnType<typeof useStyles>) {
  if (status !== 'COMPLETED') {
    return (
      <HourglassTopIcon
        className={`${cls.stateIcon} ${cls.pending}`}
        fontSize="small"
      />
    );
  }
  switch (conclusion) {
    case 'SUCCESS':
      return (
        <CheckCircleIcon
          className={`${cls.stateIcon} ${cls.success}`}
          fontSize="small"
        />
      );
    case 'FAILURE':
    case 'TIMED_OUT':
    case 'STARTUP_FAILURE':
      return (
        <CancelIcon
          className={`${cls.stateIcon} ${cls.failure}`}
          fontSize="small"
        />
      );
    case 'CANCELLED':
    case 'SKIPPED':
    case 'NEUTRAL':
    case 'STALE':
    case 'ACTION_REQUIRED':
      return (
        <ErrorIcon
          className={`${cls.stateIcon} ${cls.neutral}`}
          fontSize="small"
        />
      );
    default:
      return (
        <RadioButtonUncheckedIcon
          className={`${cls.stateIcon} ${cls.neutral}`}
          fontSize="small"
        />
      );
  }
}

type Props = { bug: BugFragment; repoName: string | null };

/** Checks renders the GitHub check-suite rollup for the PR's head commit.
 *  Hidden for issues and PRs without a head commit (can't query checks
 *  without a SHA). Data is fetched from /checks/<repo>/<sha> — backend
 *  caches for 30s. The "source: github.com" label is deliberate: we're
 *  not pretending this state is self-verifying. */
export default function Checks({ bug, repoName }: Props) {
  const classes = useStyles();
  const [data, setData] = useState<ChecksResponse>(null);
  const [loading, setLoading] = useState(false);
  const [expanded, setExpanded] = useState(false);

  const sha = bug.headCommit;
  const canFetch = bug.kind === 'PR' && !!sha && !!repoName;

  useEffect(() => {
    if (!canFetch) return;
    let cancelled = false;
    setLoading(true);
    fetch(`/checks/${encodeURIComponent(repoName!)}/${sha}`)
      .then((r) => r.json())
      .then((j: ChecksResponse) => {
        if (!cancelled) setData(j);
      })
      .catch(() => {
        if (!cancelled) setData(null);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [canFetch, repoName, sha]);

  if (!canFetch) return null;
  if (loading && !data) {
    return (
      <div className={classes.panel}>
        <div className={classes.header}>
          <span className={classes.title}>CI</span>
          <span className={classes.neutral}>loading…</span>
        </div>
      </div>
    );
  }
  if (!data) return null;

  const state = data.state ?? '';
  const suites = data.suites ?? [];
  const hasAnyRuns = suites.some((s) => s.runs && s.runs.length > 0);

  // Tally per-run conclusions across every suite. Suites without runs
  // contribute their own conclusion so we don't lose signal on apps that
  // only surface a single aggregate status. "In flight" covers any run
  // that isn't COMPLETED — queued, in_progress, pending, waiting, etc.
  let passed = 0;
  let failed = 0;
  let inFlight = 0;
  let neutral = 0;
  let total = 0;
  const bumpFromRun = (status: string, conclusion: string) => {
    total++;
    if (status !== 'COMPLETED') {
      inFlight++;
      return;
    }
    switch (conclusion) {
      case 'SUCCESS':
        passed++;
        break;
      case 'FAILURE':
      case 'TIMED_OUT':
      case 'STARTUP_FAILURE':
        failed++;
        break;
      default:
        neutral++;
    }
  };
  for (const s of suites) {
    if (s.runs && s.runs.length > 0) {
      for (const r of s.runs) bumpFromRun(r.status, r.conclusion);
    } else {
      bumpFromRun(s.status, s.conclusion);
    }
  }
  const summary =
    total === 0
      ? ''
      : `${passed}/${total} passed` +
        (failed > 0 ? `, ${failed} failed` : '') +
        (inFlight > 0 ? `, ${inFlight} in flight` : '') +
        (neutral > 0 ? `, ${neutral} skipped/neutral` : '');

  return (
    <div className={classes.panel}>
      <div className={classes.header}>
        {rollupIcon(state, classes)}
        <span className={classes.title}>CI</span>
        <span className={classes.neutral}>
          {summary || state || 'no checks reported'}
        </span>
        {hasAnyRuns && (
          <IconButton
            size="small"
            onClick={() => setExpanded((x) => !x)}
            aria-label={expanded ? 'collapse' : 'expand'}
          >
            {expanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
          </IconButton>
        )}
        <Tooltip
          title={`Fetched from ${data.source}. Treat as trusted-by-proxy.${
            data.cached ? ' (cached ≤30s)' : ''
          }`}
        >
          <span className={classes.source}>source: {data.source}</span>
        </Tooltip>
      </div>
      {data.error && <div className={classes.error}>⚠ {data.error}</div>}
      {expanded && (
        <ul className={classes.suiteList}>
          {suites.length === 0 && (
            <li className={classes.empty}>No check suites on this commit.</li>
          )}
          {suites.map((s, i) => (
            <li key={i} className={classes.suite}>
              <div className={classes.suiteHeader}>
                {runIcon(s.status, s.conclusion, classes)}
                <span>{s.appName || '(unknown app)'}</span>
                <span className={classes.suiteApp}>
                  {s.conclusion || s.status}
                </span>
              </div>
              {s.runs && s.runs.length > 0 && (
                <ul className={classes.runList}>
                  {s.runs.map((r, j) => (
                    <li key={j} className={classes.run}>
                      {runIcon(r.status, r.conclusion, classes)}
                      {r.detailsUrl ? (
                        <MuiLink
                          href={r.detailsUrl}
                          target="_blank"
                          rel="noreferrer noopener"
                          underline="hover"
                          className={classes.runName}
                        >
                          {r.name}
                        </MuiLink>
                      ) : (
                        <span className={classes.runName}>{r.name}</span>
                      )}
                    </li>
                  ))}
                </ul>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
