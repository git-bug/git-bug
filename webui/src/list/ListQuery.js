// @flow
import CircularProgress from '@material-ui/core/CircularProgress';
import gql from 'graphql-tag';
import React from 'react';
import { Query } from 'react-apollo';
import BugRow from './BugRow';
import List from './List';

const QUERY = gql`
  query($first: Int = 10, $last: Int, $after: String, $before: String) {
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

const ListQuery = () => (
  <Query query={QUERY}>
    {({ loading, error, data, fetchMore }) => {
      if (loading) return <CircularProgress />;
      if (error) return <p>Error: {error}</p>;
      return <List bugs={data.defaultRepository.bugs} fetchMore={fetchMore} />;
    }}
  </Query>
);

export default ListQuery;
