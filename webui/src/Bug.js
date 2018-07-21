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

    {bug.comments.map((comment, index) => (
      <Comment key={index} comment={comment} />
    ))}
  </main>
);

Bug.fragment = gql`
  fragment Bug on Bug {
    ...BugSummary
    comments {
      ...Comment
    }
  }

  ${BugSummary.fragment}
  ${Comment.fragment}
`;

export default withStyles(styles)(Bug);
