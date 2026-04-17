import makeStyles from '@mui/styles/makeStyles';

import Author from 'src/components/Author';
import Date from 'src/components/Date';

import { BugFragment } from './Bug.generated';

const useStyles = makeStyles((theme) => ({
  section: {
    marginTop: theme.spacing(3),
  },
  heading: {
    fontWeight: 600,
    fontSize: '1rem',
    marginBottom: theme.spacing(1),
  },
  review: {
    border: `1px solid ${theme.palette.divider}`,
    borderRadius: theme.shape.borderRadius,
    padding: theme.spacing(1.5),
    marginBottom: theme.spacing(1.5),
  },
  header: {
    display: 'flex',
    alignItems: 'baseline',
    gap: theme.spacing(1),
    marginBottom: theme.spacing(0.5),
  },
  stateBadge: {
    padding: '2px 8px',
    borderRadius: 12,
    fontSize: '0.75rem',
    color: 'white',
    fontWeight: 600,
  },
  body: {
    marginTop: theme.spacing(0.5),
    whiteSpace: 'pre-wrap',
  },
  comment: {
    marginTop: theme.spacing(1),
    paddingTop: theme.spacing(1),
    borderTop: `1px solid ${theme.palette.divider}`,
    fontSize: '0.9rem',
  },
  anchor: {
    color: theme.palette.text.secondary,
    fontFamily:
      'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
    fontSize: '0.8rem',
  },
}));

type ReviewState = 'APPROVED' | 'CHANGES_REQUESTED' | 'COMMENTED';

function stateColor(state: ReviewState) {
  switch (state) {
    case 'APPROVED':
      return '#2da44e';
    case 'CHANGES_REQUESTED':
      return '#cf222e';
    default:
      return '#57606a';
  }
}

type Props = { bug: BugFragment };

/** Reviews renders the PR's review threads. Hidden for issues. */
export default function Reviews({ bug }: Props) {
  const classes = useStyles();
  if (bug.kind !== 'PR' || !bug.reviews || bug.reviews.length === 0) {
    return null;
  }
  const shortHash = (h?: string | null) => (h ? h.slice(0, 8) : '');
  return (
    <div className={classes.section}>
      <div className={classes.heading}>Reviews ({bug.reviews.length})</div>
      {bug.reviews.map((r) => (
        <div key={r.id} className={classes.review}>
          <div className={classes.header}>
            <Author author={r.author} bold />
            <span
              className={classes.stateBadge}
              style={{ background: stateColor(r.state as ReviewState) }}
            >
              {r.state.replace('_', ' ').toLowerCase()}
            </span>
            <span className={classes.anchor}>
              @{shortHash(r.commitHash)} · <Date date={r.createdAt} />
            </span>
          </div>
          {r.body && <div className={classes.body}>{r.body}</div>}
          {r.comments &&
            r.comments.map((c) => (
              <div key={c.id} className={classes.comment}>
                <div className={classes.anchor}>
                  {c.path}:{c.startLine}
                  {c.endLine && c.endLine !== c.startLine
                    ? '-' + c.endLine
                    : ''}
                  {c.replyTo ? ' (reply)' : ''}
                </div>
                <div>
                  <Author author={c.author} /> <Date date={c.createdAt} />
                </div>
                <div className={classes.body}>{c.body}</div>
              </div>
            ))}
        </div>
      ))}
    </div>
  );
}
