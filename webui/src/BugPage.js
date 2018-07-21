import React from "react";
import { Query } from "react-apollo";
import gql from "graphql-tag";

import CircularProgress from "@material-ui/core/CircularProgress";

import Bug from "./Bug";

const QUERY = gql`
  query GetBug($id: BugID!) {
    bug(id: $id) {
      ...Bug
    }
  }

  ${Bug.fragment}
`;

const BugPage = ({ match }) => (
  <Query query={QUERY} variables={{ id: match.params.id }}>
    {({ loading, error, data }) => {
      if (loading) return <CircularProgress />;
      if (error) return <p>Error.</p>;
      return <Bug bug={data.bug} />;
    }}
  </Query>
);

export default BugPage;
