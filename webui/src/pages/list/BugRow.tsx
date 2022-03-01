import CheckCircleOutline from '@mui/icons-material/CheckCircleOutline';
import CommentOutlinedIcon from '@mui/icons-material/CommentOutlined';
import ErrorOutline from '@mui/icons-material/ErrorOutline';
import TableCell from '@mui/material/TableCell/TableCell';
import TableRow from '@mui/material/TableRow/TableRow';
import Tooltip from '@mui/material/Tooltip/Tooltip';
import makeStyles from '@mui/styles/makeStyles';
import * as React from 'react';
import { Link } from 'react-router-dom';

import Author from 'src/components/Author';
import Date from 'src/components/Date';
import Label from 'src/components/Label';
import { Status } from 'src/gqlTypes';

import { BugRowFragment } from './BugRow.generated';

type OpenClosedProps = { className: string };
const Open = ({ className }: OpenClosedProps) => (
  <Tooltip title="Open">
    <ErrorOutline htmlColor="#28a745" className={className} />
  </Tooltip>
);

const Closed = ({ className }: OpenClosedProps) => (
  <Tooltip title="Closed">
    <CheckCircleOutline htmlColor="#cb2431" className={className} />
  </Tooltip>
);

type StatusProps = { className: string; status: Status };
const BugStatus: React.FC<StatusProps> = ({
  status,
  className,
}: StatusProps) => {
  switch (status) {
    case 'OPEN':
      return <Open className={className} />;
    case 'CLOSED':
      return <Closed className={className} />;
    default:
      return <p>{'unknown status ' + status}</p>;
  }
};

const useStyles = makeStyles((theme) => ({
  cell: {
    display: 'flex',
    alignItems: 'center',
    padding: theme.spacing(1),
    '& a': {
      textDecoration: 'none',
    },
  },
  status: {
    margin: theme.spacing(1, 2),
  },
  expand: {
    width: '100%',
    lineHeight: '20px',
  },
  bugTitleWrapper: {
    display: 'flex',
    flexDirection: 'row',
    flexWrap: 'wrap',
    //alignItems: 'center',
  },
  title: {
    display: 'inline',
    color: theme.palette.text.primary,
    fontSize: '1.3rem',
    fontWeight: 500,
    marginBottom: theme.spacing(1),
  },
  label: {
    maxWidth: '40ch',
    marginLeft: theme.spacing(0.25),
    marginRight: theme.spacing(0.25),
  },
  details: {
    lineHeight: '1.5rem',
    color: theme.palette.text.secondary,
  },
  commentCount: {
    fontSize: '1rem',
    minWidth: '2rem',
    marginLeft: theme.spacing(0.5),
    marginRight: theme.spacing(1),
  },
  commentCountCell: {
    display: 'inline-flex',
    minWidth: theme.spacing(5),
    marginLeft: theme.spacing(0.5),
  },
}));

type Props = {
  bug: BugRowFragment;
};

function BugRow({ bug }: Props) {
  const classes = useStyles();
  // Subtract 1 from totalCount as 1 comment is the bug description
  const commentCount = bug.comments.totalCount - 1;
  return (
    <TableRow hover>
      <TableCell className={classes.cell}>
        <BugStatus status={bug.status} className={classes.status} />
        <div className={classes.expand}>
          <Link to={'bug/' + bug.id}>
            <div className={classes.bugTitleWrapper}>
              <span className={classes.title}>{bug.title}</span>
              {bug.labels.length > 0 &&
                bug.labels.map((l) => (
                  <Label key={l.name} label={l} className={classes.label} />
                ))}
            </div>
          </Link>
          <div className={classes.details}>
            {bug.humanId} opened&nbsp;
            <Date date={bug.createdAt} />
            &nbsp;by&nbsp;
            <Author className={classes.details} author={bug.author} />
          </div>
        </div>
        <span className={classes.commentCountCell}>
          {commentCount > 0 && (
            <>
              <CommentOutlinedIcon aria-label="Comment count" />
              <span className={classes.commentCount}>{commentCount}</span>
            </>
          )}
        </span>
      </TableCell>
    </TableRow>
  );
}

export default BugRow;
