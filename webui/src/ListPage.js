import React from "react";
import { Query } from "react-apollo";
import gql from "graphql-tag";
import { withStyles } from "@material-ui/core/styles";

import CircularProgress from "@material-ui/core/CircularProgress";

import BugSummary from "./BugSummary";

const QUERY = gql`
  {
    bugs: allBugs {
      ...BugSummary
    }
  }

  ${BugSummary.fragment}
`;

const styles = theme => ({
  main: {
    maxWidth: 600,
    margin: "auto",
    marginTop: theme.spacing.unit * 4
  }
});

const List = withStyles(styles)(({ bugs, classes }) => (
  <main className={classes.main}>
    {bugs.map(bug => (
      <BugSummary bug={bug} key={bug.id} />
    ))}
  </main>
));

const ListPage = () => (
  <Query query={QUERY}>
    {({ loading, error, data }) => {
      if (loading) return <CircularProgress />;
      if (error) return <p>Error.</p>;
      return <List bugs={data.bugs} />;
    }}
  </Query>
);

export default ListPage;
