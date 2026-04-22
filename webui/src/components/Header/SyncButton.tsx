import { useApolloClient } from '@apollo/client';
import SyncIcon from '@mui/icons-material/Sync';
import { Button, CircularProgress, Tooltip } from '@mui/material';
import { alpha } from '@mui/material/styles';
import makeStyles from '@mui/styles/makeStyles';
import { useCallback, useEffect, useRef, useState } from 'react';
import { useLocation } from 'react-router';

interface SyncStatus {
  // running is true if anything is syncing (bulk or any ad-hoc single-repo).
  running: boolean;
  // bulkRunning is true only for the all-repos run. The UI uses this to
  // gate the "Sync all" button without blocking "Sync this".
  bulkRunning: boolean;
  startedAt?: string;
  finishedAt?: string;
  // Union of repos currently being synced (bulk + ad-hoc).
  active: string[];
  done: number;
  total: number;
  importedBugs: number;
  importedIdentities: number;
  // Breakdown of the last bulk run.
  initialSyncs: number;
  catchupSyncs: number;
  noBridgeRepos: number;
  errors?: Record<string, string>;
  // Most recent GitHub GraphQL rate-limit snapshot, if we've ever observed
  // one during this server's lifetime. Absent on a freshly-started backend.
  rateLimit?: { remaining: number; limit: number; resetAt?: string };
}

const useStyles = makeStyles((theme) => ({
  button: {
    color: theme.palette.primary.contrastText,
    textTransform: 'none',
    marginLeft: theme.spacing(1),
    minWidth: 120,
    '&:hover': {
      backgroundColor: alpha(theme.palette.common.white, 0.1),
    },
    // MUI's default disabled color is too faint against a dark AppBar in
    // light mode ("Sync this" becomes unreadable). Keep enough alpha to
    // stay legible while still signalling the disabled state.
    '&.Mui-disabled': {
      color: alpha(theme.palette.primary.contrastText, 0.55),
    },
  },
  thisButton: {
    minWidth: 110,
    // A faint outline keeps "Sync this" delineated from the adjacent Sync
    // button and gives the button edge contrast with the AppBar in both
    // light and dark modes.
    border: `1px solid ${alpha(theme.palette.primary.contrastText, 0.4)}`,
    '&.Mui-disabled': {
      borderColor: alpha(theme.palette.primary.contrastText, 0.25),
    },
  },
  errorButton: {
    // Tint the button when the last sync failed so the user notices the
    // error indicator without having to hover for the tooltip. Keep the
    // text white for contrast against the warning-colour background.
    backgroundColor: alpha(theme.palette.warning.main, 0.55),
    '&:hover': {
      backgroundColor: alpha(theme.palette.warning.main, 0.75),
    },
  },
}));

// Header sits above the :repoName route — useParams() is empty here, but
// useLocation() works and re-renders on every navigation. Extract the repo
// name from /r/<urlencoded-name>/... ourselves.
function repoNameFromPath(pathname: string): string | null {
  const m = pathname.match(/^\/r\/([^/]+)/);
  if (!m) return null;
  try {
    return decodeURIComponent(m[1]);
  } catch {
    return null;
  }
}

/** SyncButton fires POST /sync to start a run and then polls GET /sync
 *  every second. The polling loop runs for the whole life of the component
 *  — simpler than gating on running state, which had timing issues across
 *  re-renders. Apollo cache is reset once per completed run so freshly
 *  imported bugs appear without a hard refresh.
 *
 *  Two buttons: "Sync" syncs every registered repo; "Sync this" (shown only
 *  on /r/<repo>/* pages) syncs just the repo currently being viewed. Both
 *  share the same polling state. While a run is in flight "Sync this" stays
 *  visible but disabled — the backend 409s a second run anyway, and hiding
 *  the button on long-running syncs made it feel like the feature vanished. */
export default function SyncButton() {
  const classes = useStyles();
  const apollo = useApolloClient();
  const location = useLocation();
  const [status, setStatus] = useState<SyncStatus | null>(null);
  const wasRunning = useRef(false);

  const fetchStatus = useCallback(async () => {
    try {
      const r = await fetch('/sync', { method: 'GET' });
      if (!r.ok) return;
      const s = (await r.json()) as SyncStatus;
      setStatus(s);
      if (wasRunning.current && !s.running) {
        apollo.resetStore().catch(() => {});
      }
      wasRunning.current = s.running;
    } catch {
      // transient network hiccup — next tick will retry.
    }
  }, [apollo]);

  useEffect(() => {
    fetchStatus();
    const id = setInterval(fetchStatus, 1000);
    return () => clearInterval(id);
  }, [fetchStatus]);

  const kick = async (repo?: string) => {
    const url = repo ? `/sync?repo=${encodeURIComponent(repo)}` : '/sync';
    const r = await fetch(url, { method: 'POST' });
    if (r.status === 409) return; // already running — polling will catch it
    if (!r.ok) return;
    const s = (await r.json()) as SyncStatus;
    setStatus(s);
    wasRunning.current = s.running;
  };

  const running = status?.running ?? false;
  const bulkRunning = status?.bulkRunning ?? false;
  const label = bulkRunning
    ? `${status?.done ?? 0} / ${status?.total ?? 0}`
    : 'Sync';
  const errors = status?.errors ?? {};
  const errorCount = Object.keys(errors).length;
  const active = status?.active ?? [];
  // "Sync this" should only be disabled when *this specific repo* is
  // actively syncing — a bulk run of other repos must not block it.
  const repoName = repoNameFromPath(location.pathname);
  const thisRepoActive = !!(repoName && active.includes(repoName));
  // Surface the most recent error for the repo we're looking at, falling
  // back to the first error overall so rate-limit and auth failures don't
  // hide behind a bland "1 error" count.
  const repoError = repoName ? errors[repoName] : undefined;
  const firstError = errorCount > 0 ? Object.values(errors)[0] : undefined;
  const highlightError = repoError ?? firstError;
  const initial = status?.initialSyncs ?? 0;
  const catchup = status?.catchupSyncs ?? 0;
  const noBridge = status?.noBridgeRepos ?? 0;
  const breakdownParts = [
    initial > 0 ? `${initial} initial` : '',
    catchup > 0 ? `${catchup} catchup` : '',
    noBridge > 0 ? `${noBridge} skipped` : '',
  ].filter(Boolean);
  const breakdown =
    breakdownParts.length > 0 ? ` · ${breakdownParts.join(', ')}` : '';
  const runningLine = running
    ? `Syncing: ${active.join(', ') || '…'}${
        bulkRunning ? ` — ${status?.done ?? 0} of ${status?.total ?? 0} done` : ''
      }${breakdown}`
    : status?.finishedAt && new Date(status.finishedAt).getTime() > 0
    ? `Last sync: ${new Date(status.finishedAt).toLocaleTimeString()} · ${
        status.importedBugs
      } bugs, ${status.importedIdentities} identities${breakdown}`
    : 'Re-sync all registered repositories from their bridges';
  const rl = status?.rateLimit;
  const budgetLine = rl
    ? `\nGitHub quota: ${rl.remaining}/${rl.limit}${
        rl.resetAt && rl.remaining < rl.limit * 0.2
          ? ` — resets ${new Date(rl.resetAt).toLocaleTimeString()}`
          : ''
      }`
    : '';
  const tooltip =
    (highlightError ? `${runningLine}\n⚠ ${highlightError}` : runningLine) +
    budgetLine;

  return (
    <>
      {repoName && (
        <Tooltip
          title={
            thisRepoActive
              ? `${repoName} is already being synced — wait for it to finish`
              : repoError
              ? `⚠ ${repoError}\n\nClick to retry`
              : `Re-sync only ${repoName} from its bridge`
          }
        >
          <span>
            <Button
              className={`${classes.button} ${classes.thisButton} ${
                repoError ? classes.errorButton : ''
              }`}
              onClick={() => kick(repoName)}
              disabled={thisRepoActive}
              startIcon={
                thisRepoActive ? (
                  <CircularProgress
                    size={16}
                    thickness={5}
                    sx={{ color: 'inherit' }}
                  />
                ) : (
                  <SyncIcon />
                )
              }
            >
              Sync this
            </Button>
          </span>
        </Tooltip>
      )}
      <Tooltip title={tooltip}>
        <span>
          <Button
            className={`${classes.button} ${
              errorCount > 0 && !bulkRunning ? classes.errorButton : ''
            }`}
            onClick={() => kick()}
            disabled={bulkRunning}
            startIcon={
              bulkRunning ? (
                <CircularProgress
                  size={16}
                  thickness={5}
                  sx={{ color: 'inherit' }}
                />
              ) : (
                <SyncIcon />
              )
            }
          >
            {label}
            {errorCount > 0 && !bulkRunning ? ` · ${errorCount}⚠` : ''}
          </Button>
        </span>
      </Tooltip>
    </>
  );
}
