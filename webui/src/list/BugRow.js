import { withStyles } from '@material-ui/core/styles';
import TableCell from '@material-ui/core/TableCell/TableCell';
import TableRow from '@material-ui/core/TableRow/TableRow';
import Tooltip from '@material-ui/core/Tooltip/Tooltip';
import Typography from '@material-ui/core/Typography';
import ErrorOutline from '@material-ui/icons/ErrorOutline';
import gql from 'graphql-tag';
import React from 'react';
import { Link } from 'react-router-dom';
import Date from '../Date';
import Label from '../Label';

const Open = ({ className }) => (
  <Tooltip title="Open">
    <ErrorOutline nativeColor="#28a745" className={className} />
  </Tooltip>
);

const Closed = ({ className }) => (
  <Tooltip title="Closed">
    <ErrorOutline nativeColor="#cb2431" className={className} />
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

const styles = theme => ({
  cell: {
    display: 'flex',
    alignItems: 'center',
    '& a': {
      textDecoration: 'none',
    },
  },
  status: {
    margin: 10,
  },
  expand: {
    width: '100%',
  },
  title: {
    display: 'inline',
  },
  labels: {
    paddingLeft: theme.spacing.unit,
  },
});

const BugRow = ({ bug, classes }) => (
  <TableRow hover>
    <TableCell className={classes.cell}>
      <Status status={bug.status} className={classes.status} />
      <div className={classes.expand}>
        <Link to={'bug/' + bug.humanId}>
          <div className={classes.expand}>
            <Typography variant={'title'} className={classes.title}>
              {bug.title}
            </Typography>
            {bug.labels.length > 0 && (
              <span className={classes.labels}>
                {bug.labels.map(l => (
                  <Label key={l} label={l} />
                ))}
              </span>
            )}
          </div>
        </Link>
        <Typography color={'textSecondary'}>
          {bug.humanId} opened
          <Date date={bug.createdAt} />
          by {bug.author.name}
        </Typography>
      </div>
    </TableCell>
  </TableRow>
);

BugRow.fragment = gql`
  fragment BugRow on Bug {
    id
    humanId
    title
    status
    createdAt
    labels
    author {
      name
    }
  }
`;

export default withStyles(styles)(BugRow);
