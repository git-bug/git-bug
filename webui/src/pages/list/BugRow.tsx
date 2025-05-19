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
import Content from 'src/components/Content';
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
    '& a': {
      textDecoration: 'none',
    },

    padding: theme.spacing(1),
    [theme.breakpoints.down('md')]: {
      padding: theme.spacing(2, 1),
    },
  },
  status: {
    margin: theme.spacing(1, 2),
  },
  expand: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column',
    gap: theme.spacing(0.5),

    [theme.breakpoints.down('md')]: {
      gap: theme.spacing(1)
    }
  },
  maindataWrapper: {
    flex: '1 0',
    display: 'flex',
    flexWrap: 'wrap',
    alignItems: 'start',
    justifyContent: 'end',
    gap: theme.spacing(0.5),

    [theme.breakpoints.down('md')]: {
      flexDirection: 'column',
    }
  },
  title: {
    display: 'inline-flex',
    whiteSpace: 'nowrap',
    flex: '1 0',
    color: theme.palette.text.primary,
    fontSize: '1.1rem',
    fontWeight: 500,

    [theme.breakpoints.down('md')]: {
      whiteSpace: 'initial',
    }
  },
  labelsWrapper: {
    display: 'inline-flex',
    gap: theme.spacing(0.5),

    [theme.breakpoints.down('md')]: {
      flexWrap: 'wrap',
    }
  },
  sidedataWrapper: {
    display: 'flex',
    alignItems: 'end',
    gap: theme.spacing(1),

    [theme.breakpoints.down('md')]: {
      alignItems: 'start',
      flexDirection: 'column-reverse'
    }
  },
  label: {
    maxWidth: '40ch',
  },
  details: {
    flex: 1,
    color: theme.palette.text.secondary,
  },
  commentCountCell: {
    display: 'flex',
    alignItems: 'end',
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
            <Link className={classes.title} to={'bug/' + bug.id}>
              <Content inline pipeline='title' markdown={bug.title} />
            </Link>
            {bug.labels.length > 0 && 
              <span className={classes.labelsWrapper}>
                {bug.labels.map((l) => 
                  <Label inline key={l.name} label={l} className={classes.label} />
                )}
              </span>}
          </div>

          <div className={classes.sidedataWrapper}>
            <div className={classes.details}>
              {bug.humanId} opened&nbsp;
              <Date date={bug.createdAt} />
              &nbsp;by&nbsp;
              <Author author={bug.author} />
            </div>

            {commentCount > 0 &&
              <div className={classes.commentCountCell}>
                <span className={classes.commentCount}>{commentCount}</span>
                  <CommentOutlinedIcon aria-label="Comment count" fontSize='small' />
                </div>}
              </div>
          </div>
      </TableCell>
    </TableRow>
  );
}

export default BugRow;
