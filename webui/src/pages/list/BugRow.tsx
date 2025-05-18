import CheckCircleOutline from '@mui/icons-material/CheckCircleOutline';
import CommentOutlinedIcon from '@mui/icons-material/CommentOutlined';
import ErrorOutline from '@mui/icons-material/ErrorOutline';
import TableCell from '@mui/material/TableCell/TableCell';
import TableRow from '@mui/material/TableRow/TableRow';
import Tooltip from '@mui/material/Tooltip/Tooltip';
import makeStyles from '@mui/styles/makeStyles';
import * as React from 'react';
import { Link } from 'react-router';

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
    padding: theme.spacing(1),
    '& a': {
      textDecoration: 'none',
    },
  },
  status: {
    margin: theme.spacing(1, 2),
  },
  expand: {
    flex: 1,
    lineHeight: '20px',
    display: 'flex',
    flexDirection: 'row',
    flexWrap: 'wrap',
    alignItems: 'top',
    [theme.breakpoints.down('md')]: {
      flexDirection: 'column'
    }
  },
  title: {
    color: theme.palette.text.primary,
    fontSize: '1.1rem',
    fontWeight: 500,
  },
  maindataWrapper: {
    flex: '1 0',
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'start',

    [theme.breakpoints.down('md')]: {
      marginBottom: theme.spacing(1)
    }
  },
  sidedataWrapper: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'end',
    gap: theme.spacing(1),

    [theme.breakpoints.down('md')]: {
      flexDirection: 'row',
      flex: '1 1'
    }
  },
  labelsWrapper: {
    display: 'flex',
    flexDirection: 'row',
    gap: theme.spacing(0.5),
    [theme.breakpoints.down('md')]: {
      flex: 1,
      flexWrap: 'wrap',
    }
  },
  label: {
    maxWidth: '40ch',
  },
  details: {
    color: theme.palette.text.secondary,
    marginTop: theme.spacing(1),
  },
  commentCountCell: {
    display: 'flex',
    alignItems: 'center',
    marginRight: theme.spacing(1),
  },
  commentCount: {
    margin: theme.spacing(0, 1),
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
              <div className={classes.maindataWrapper}>
                <Link to={'bug/' + bug.id}>
                  <div className={classes.title}>{bug.title}</div>
                </Link>
                <div className={classes.details}>
                  {bug.humanId} opened&nbsp;
                  <Date date={bug.createdAt} />
                  &nbsp;by&nbsp;
                  <Author className={classes.details} author={bug.author} />
                </div>
              </div>
              <div className={classes.sidedataWrapper}>
                <div className={classes.labelsWrapper}>
                  {commentCount > 0 &&
                    <div className={classes.commentCountCell}>
                      <span className={classes.commentCount}>{commentCount}</span>
                      <CommentOutlinedIcon aria-label="Comment count" fontSize='small' />
                    </div>}
                  {bug.labels.length > 0 && bug.labels.map((l) => <Label key={l.name} label={l} className={classes.label} />)}
                </div>
              </div>
            </div>
      </TableCell>
    </TableRow>
  );
}

export default BugRow;
