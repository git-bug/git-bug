import React from "react";
import gql from "graphql-tag";
import { withStyles } from "@material-ui/core/styles";

import Comment from "./Comment";
import BugSummary from "./BugSummary";

const styles = theme => ({
  main: {
    maxWidth: 600,
    margin: "auto",
    marginTop: theme.spacing.unit * 4
  }
});

const Bug = ({ bug, classes }) => (
  <main className={classes.main}>
    <BugSummary bug={bug} />

    {bug.comments.edges.map(({ cursor, node }) => (
      <Comment key={cursor} comment={node} />
    ))}
  </main>
);

Bug.fragment = gql`
  fragment Bug on Bug {
    ...BugSummary
    comments(input: { first: 10 }) {
      edges {
        cursor
        node {
          ...Comment
        }
      }
    }
  }

  ${BugSummary.fragment}
  ${Comment.fragment}
`;

export default withStyles(styles)(Bug);
