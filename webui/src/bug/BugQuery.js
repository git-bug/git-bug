import CircularProgress from '@material-ui/core/CircularProgress';
import gql from 'graphql-tag';
import React from 'react';
import { Query } from 'react-apollo';

import Bug from './Bug';

const QUERY = gql`
  query GetBug($id: String!) {
    defaultRepository {
      bug(prefix: $id) {
        ...Bug
      }
    }
  }

  ${Bug.fragment}
`;

const BugQuery = ({ match }) => (
  <Query query={QUERY} variables={{ id: match.params.id }}>
    {({ loading, error, data }) => {
      if (loading) return <CircularProgress />;
      if (error) return <p>Error: {error}</p>;
      return <Bug bug={data.defaultRepository.bug} />;
    }}
  </Query>
);

export default BugQuery;
