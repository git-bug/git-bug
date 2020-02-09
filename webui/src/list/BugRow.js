import { makeStyles } from '@material-ui/styles';
import TableCell from '@material-ui/core/TableCell/TableCell';
import TableRow from '@material-ui/core/TableRow/TableRow';
import Tooltip from '@material-ui/core/Tooltip/Tooltip';
import ErrorOutline from '@material-ui/icons/ErrorOutline';
import CheckCircleOutline from '@material-ui/icons/CheckCircleOutline';
import gql from 'graphql-tag';
import React from 'react';
import { Link } from 'react-router-dom';
import Date from '../Date';
import Label from '../Label';
import Author from '../Author';

const Open = ({ className }) => (
  <Tooltip title="Open">
    <ErrorOutline htmlColor="#28a745" className={className} />
  </Tooltip>
);

const Closed = ({ className }) => (
  <Tooltip title="Closed">
    <CheckCircleOutline htmlColor="#cb2431" className={className} />
  </Tooltip>
);

const Status = ({ status, className }) => {
  switch (status) {
    case 'OPEN':
      return <Open className={className} />;
    case 'CLOSED':
      return <Closed className={className} />;
    default:
      return 'unknown status ' + status;
  }
};

const useStyles = makeStyles(theme => ({
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
  title: {
    display: 'inline',
    color: theme.palette.text.primary,
    fontSize: '1.3rem',
    fontWeight: 500,
  },
  details: {
    ...theme.typography.textSecondary,
    lineHeight: '1.5rem',
    color: theme.palette.text.secondary,
  },
  labels: {
    paddingLeft: theme.spacing(1),
    '& > *': {
      display: 'inline-block',
    },
  },
}));

function BugRow({ bug }) {
  const classes = useStyles();
  return (
    <TableRow hover>
      <TableCell className={classes.cell}>
        <Status status={bug.status} className={classes.status} />
        <div className={classes.expand}>
          <Link to={'bug/' + bug.humanId}>
            <div className={classes.expand}>
              <span className={classes.title}>{bug.title}</span>
              {bug.labels.length > 0 && (
                <span className={classes.labels}>
                  {bug.labels.map(l => (
                    <Label key={l.name} label={l} />
                  ))}
                </span>
              )}
            </div>
          </Link>
          <div className={classes.details}>
            {bug.humanId} opened
            <Date date={bug.createdAt} />
            by {bug.author.displayName}
          </div>
        </div>
      </TableCell>
    </TableRow>
  );
}

BugRow.fragment = gql`
  fragment BugRow on Bug {
    id
    humanId
    title
    status
    createdAt
    labels {
      ...Label
    }
    ...authored
  }

  ${Label.fragment}
  ${Author.fragment}
`;

export default BugRow;
