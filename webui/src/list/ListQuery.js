// @flow
import CircularProgress from '@material-ui/core/CircularProgress';
import gql from 'graphql-tag';
import React, { useState } from 'react';
import { Query } from 'react-apollo';
import BugRow from './BugRow';
import List from './List';

const QUERY = gql`
  query($first: Int, $last: Int, $after: String, $before: String) {
    defaultRepository {
      bugs: allBugs(
        first: $first
        last: $last
        after: $after
        before: $before
      ) {
        totalCount
        edges {
          cursor
          node {
            ...BugRow
          }
        }
        pageInfo {
          hasNextPage
          hasPreviousPage
          startCursor
          endCursor
        }
      }
    }
  }

  ${BugRow.fragment}
`;

function ListQuery() {
  const [page, setPage] = useState({ first: 10, after: null });

  const perPage = page.first || page.last;
  const nextPage = pageInfo =>
    setPage({ first: perPage, after: pageInfo.endCursor });
  const prevPage = pageInfo =>
    setPage({ last: perPage, before: pageInfo.startCursor });

  return (
    <Query query={QUERY} variables={page}>
      {({ loading, error, data }) => {
        if (loading) return <CircularProgress />;
        if (error) return <p>Error: {error}</p>;
        const bugs = data.defaultRepository.bugs;
        return (
          <List
            bugs={bugs}
            nextPage={() => nextPage(bugs.pageInfo)}
            prevPage={() => prevPage(bugs.pageInfo)}
          />
        );
      }}
    </Query>
  );
}

export default ListQuery;
