import { withStyles } from '@material-ui/core/styles';
import Table from '@material-ui/core/Table/Table';
import TableBody from '@material-ui/core/TableBody/TableBody';
import TablePagination from '@material-ui/core/TablePagination/TablePagination';
import React from 'react';
import BugRow from './BugRow';

const styles = theme => ({
  main: {
    maxWidth: 600,
    margin: 'auto',
    marginTop: theme.spacing.unit * 4,
  },
});

class List extends React.Component {
  props: {
    bugs: Array,
    fetchMore: any => any,
    classes: any,
  };

  state = {
    page: 0,
    rowsPerPage: 10,
    lastQuery: {},
  };

  handleChangePage = (event, page) => {
    const { bugs, fetchMore } = this.props;
    const { rowsPerPage } = this.state;
    const pageInfo = bugs.pageInfo;

    if (page === this.state.page + 1) {
      if (!pageInfo.hasNextPage) {
        return;
      }

      const variables = {
        after: pageInfo.endCursor,
        first: rowsPerPage,
      };

      fetchMore({
        variables,
        updateQuery: this.updateQuery,
      });

      this.setState({ page, lastQuery: variables });
      return;
    }

    if (page === this.state.page - 1) {
      if (!pageInfo.hasPreviousPage) {
        return;
      }

      const variables = {
        before: pageInfo.startCursor,
        last: rowsPerPage,
      };

      fetchMore({
        variables,
        updateQuery: this.updateQuery,
      });

      this.setState({ page, lastQuery: variables });
      return;
    }

    throw new Error('non neighbour page pagination is not supported');
  };

  handleChangeRowsPerPage = event => {
    const { fetchMore } = this.props;
    const { lastQuery } = this.state;
    const rowsPerPage = event.target.value;

    const variables = lastQuery;

    if (lastQuery.first) {
      variables.first = rowsPerPage;
    } else if (lastQuery.last) {
      variables.last = rowsPerPage;
    } else {
      variables.first = rowsPerPage;
    }

    fetchMore({
      variables,
      updateQuery: this.updateQuery,
    });

    this.setState({ rowsPerPage, lastQuery: variables });
  };

  updateQuery = (previousResult, { fetchMoreResult }) => {
    return fetchMoreResult ? fetchMoreResult : previousResult;
  };

  render() {
    const { classes, bugs } = this.props;
    const { page, rowsPerPage } = this.state;

    return (
      <main className={classes.main}>
        <Table className={classes.table}>
          <TableBody>
            {bugs.edges.map(({ cursor, node }) => (
              <BugRow bug={node} key={cursor} />
            ))}
          </TableBody>
        </Table>
        <TablePagination
          component="div"
          count={bugs.totalCount}
          rowsPerPage={rowsPerPage}
          page={page}
          backIconButtonProps={{
            'aria-label': 'Previous Page',
          }}
          nextIconButtonProps={{
            'aria-label': 'Next Page',
          }}
          onChangePage={this.handleChangePage}
          onChangeRowsPerPage={this.handleChangeRowsPerPage}
        />
      </main>
    );
  }
}

export default withStyles(styles)(List);
