import { useApolloClient } from '@apollo/client';
import SyncIcon from '@mui/icons-material/Sync';
import { Button, CircularProgress, Tooltip } from '@mui/material';
import { alpha } from '@mui/material/styles';
import makeStyles from '@mui/styles/makeStyles';
import { useCallback, useEffect, useRef, useState } from 'react';

interface SyncStatus {
  running: boolean;
  startedAt?: string;
  finishedAt?: string;
  // Up to syncConcurrency repos currently being pulled.
  active: string[];
  done: number;
  total: number;
  importedBugs: number;
  importedIdentities: number;
  errors?: Record<string, string>;
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
  },
}));

/** SyncButton fires POST /sync to start a run and then polls GET /sync
 *  every second. The polling loop runs for the whole life of the component
 *  — simpler than gating on running state, which had timing issues across
 *  re-renders. Apollo cache is reset once per completed run so freshly
 *  imported bugs appear without a hard refresh. */
export default function SyncButton() {
  const classes = useStyles();
  const apollo = useApolloClient();
  const [status, setStatus] = useState<SyncStatus | null>(null);

  // Keep the last-seen running flag in a ref so the interval can detect a
  // running->finished transition without re-registering.
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
    // Initial snapshot, then poll for the lifetime of the component. 1s
    // cadence is cheap (one small JSON) and keeps the spinner responsive.
    fetchStatus();
    const id = setInterval(fetchStatus, 1000);
    return () => clearInterval(id);
  }, [fetchStatus]);

  const kick = async () => {
    const r = await fetch('/sync', { method: 'POST' });
    if (r.status === 409) return; // already running — polling will catch it
    if (!r.ok) return;
    const s = (await r.json()) as SyncStatus;
    setStatus(s);
    wasRunning.current = s.running;
  };

  const running = status?.running ?? false;
  const label = running
    ? `${status?.done ?? 0} / ${status?.total ?? 0}`
    : 'Sync';
  const errorCount = Object.keys(status?.errors ?? {}).length;
  const active = status?.active ?? [];
  const tooltip = running
    ? `Syncing: ${active.join(', ') || '…'} — ${status?.done ?? 0} of ${
        status?.total ?? 0
      } done`
    : status?.finishedAt && new Date(status.finishedAt).getTime() > 0
    ? `Last sync: ${new Date(status.finishedAt).toLocaleTimeString()} · ${
        status.importedBugs
      } bugs, ${status.importedIdentities} identities${
        errorCount > 0 ? `, ${errorCount} errors` : ''
      }`
    : 'Re-sync all registered repositories from their bridges';

  return (
    <Tooltip title={tooltip}>
      <span>
        <Button
          className={classes.button}
          onClick={kick}
          startIcon={
            running ? (
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
        </Button>
      </span>
    </Tooltip>
  );
}
