import { makeStyles } from '@material-ui/styles';
import IconButton from '@material-ui/core/IconButton';
import Table from '@material-ui/core/Table/Table';
import TableBody from '@material-ui/core/TableBody/TableBody';
import KeyboardArrowLeft from '@material-ui/icons/KeyboardArrowLeft';
import KeyboardArrowRight from '@material-ui/icons/KeyboardArrowRight';
import React from 'react';
import BugRow from './BugRow';

const useStyles = makeStyles(theme => ({
  main: {
    maxWidth: 600,
    margin: 'auto',
    marginTop: theme.spacing.unit * 4,
  },
  pagination: {
    ...theme.typography.overline,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'flex-end',
  },
}));

function List({ bugs, nextPage, prevPage }) {
  const classes = useStyles();
  const { hasNextPage, hasPreviousPage } = bugs.pageInfo;
  return (
    <main className={classes.main}>
      <Table className={classes.table}>
        <TableBody>
          {bugs.edges.map(({ cursor, node }) => (
            <BugRow bug={node} key={cursor} />
          ))}
        </TableBody>
      </Table>

      <div className={classes.pagination}>
        <div>Total: {bugs.totalCount}</div>
        <IconButton onClick={prevPage} disabled={!hasPreviousPage}>
          <KeyboardArrowLeft />
        </IconButton>
        <IconButton onClick={nextPage} disabled={!hasNextPage}>
          <KeyboardArrowRight />
        </IconButton>
      </div>
    </main>
  );
}

export default List;
