import makeStyles from '@mui/styles/makeStyles';

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
  row: {
    display: 'flex',
    alignItems: 'baseline',
    marginBottom: theme.spacing(0.5),
  },
  label: {
    color: theme.palette.text.secondary,
    minWidth: '6.5rem',
  },
  value: {
    fontFamily:
      'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
  },
  mergedBadge: {
    display: 'inline-block',
    padding: '2px 8px',
    borderRadius: 12,
    background: '#8250df',
    color: 'white',
    fontSize: '0.75rem',
    marginLeft: theme.spacing(1),
  },
}));

type Props = { bug: BugFragment };

/** PrInfo is a compact panel showing PR-specific metadata: base->head refs,
 *  current head commit, and the merge commit if merged. Rendered only for
 *  bugs with kind=PR. */
export default function PrInfo({ bug }: Props) {
  const classes = useStyles();
  if (bug.kind !== 'PR') {
    return null;
  }
  const shortHash = (h?: string | null) => (h ? h.slice(0, 8) : '');
  const shortRef = (r?: string | null) =>
    r ? r.replace(/^refs\/heads\//, '') : '';

  return (
    <div className={classes.panel}>
      <div className={classes.row}>
        <span className={classes.label}>Branches:</span>
        <span className={classes.value}>
          {shortRef(bug.baseRef)} ← {shortRef(bug.headRef)}
        </span>
      </div>
      {bug.headCommit && (
        <div className={classes.row}>
          <span className={classes.label}>Head commit:</span>
          <span className={classes.value}>{shortHash(bug.headCommit)}</span>
        </div>
      )}
      {bug.mergeCommit && (
        <div className={classes.row}>
          <span className={classes.label}>Merge commit:</span>
          <span className={classes.value}>{shortHash(bug.mergeCommit)}</span>
          <span className={classes.mergedBadge}>merged</span>
        </div>
      )}
    </div>
  );
}
